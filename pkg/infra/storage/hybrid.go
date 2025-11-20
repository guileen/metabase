package storage

import ("context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/cockroachdb/pebble"
	_ "github.com/mattn/go-sqlite3")

// HybridStorage represents the integrated SQLite + Pebble storage engine
type HybridStorage struct {
	db     *sql.DB
	kv     *pebble.DB
	config *Config
	cache  *StorageCache
	mu     sync.RWMutex
	ctx    context.Context
	cancel context.CancelFunc
}

// StorageCache provides intelligent caching layer
type StorageCache struct {
	localCache map[string]*CacheEntry
	maxSize    int
	mu         sync.RWMutex
	ttl        time.Duration
}

// CacheEntry represents a cache entry
type CacheEntry struct {
	Key       string
	Value     interface{}
	ExpiresAt time.Time
	HitCount  int64
	LastHit   time.Time
}

// QueryBuilder builds optimized queries
type QueryBuilder struct {
	table    string
	where    []WhereClause
	orderBy  []OrderByClause
	limit    int
	offset   int
	joins    []JoinClause
}

// WhereClause represents a WHERE condition
type WhereClause struct {
	Field    string
	Operator string
	Value    interface{}
	Op       string // AND/OR
}

// OrderByClause represents ORDER BY clause
type OrderByClause struct {
	Field string
	Dir   string
}

// JoinClause represents JOIN clause
type JoinClause struct {
	Type       string
	Table      string
	On         string
	Conditions []WhereClause
}

// IndexManager manages database indexes
type IndexManager struct {
	storage *HybridStorage
	indexes map[string]*IndexDefinition
	mu      sync.RWMutex
}

// IndexDefinition defines an index
type IndexDefinition struct {
	Name        string   `json:"name"`
	Table       string   `json:"table"`
	Fields      []string `json:"fields"`
	Type        string   `json:"type"`        // btree, hash, fulltext
	Unique      bool     `json:"unique"`
	Partial     string   `json:"partial"`     // WHERE clause for partial index
	InBackground bool    `json:"in_background"`
}

// NewHybridStorage creates a new hybrid storage engine
func NewHybridStorage(config *Config) (*HybridStorage, error) {
	ctx, cancel := context.WithCancel(context.Background())

	// Ensure data directories exist
	sqliteDir := filepath.Dir(config.SQLitePath)
	pebbleDir := filepath.Dir(config.PebblePath)

	if err := createDirectory(sqliteDir); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create SQLite directory: %w", err)
	}

	if err := createDirectory(pebbleDir); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create Pebble directory: %w", err)
	}

	// Initialize SQLite with optimizations
	db, err := initOptimizedSQLite(config.SQLitePath)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to initialize SQLite: %w", err)
	}

	// Initialize Pebble with optimizations
	kv, err := initOptimizedPebble(config.PebblePath)
	if err != nil {
		db.Close()
		cancel()
		return nil, fmt.Errorf("failed to initialize Pebble: %w", err)
	}

	storage := &HybridStorage{
		db:     db,
		kv:     kv,
		config: config,
		cache:  NewStorageCache(config.CacheSize),
		ctx:    ctx,
		cancel: cancel,
	}

	// Initialize database schema
	if err := storage.initSchema(); err != nil {
		storage.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	// Start background tasks
	go storage.startBackgroundTasks()

	return storage, nil
}

// initOptimizedSQLite initializes SQLite with performance optimizations
func initOptimizedSQLite(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", path+"?"+
		"cache=shared&"+
		"mode=rwc&"+
		"_journal_mode=WAL&"+
		"_synchronous=NORMAL&"+
		"_cache_size=10000&"+
		"_temp_store=MEMORY&"+
		"_mmap_size=268435456&"+
		"foreign_keys=ON&"+
		"journal_size_limit=1048576")
	if err != nil {
		return nil, err
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Set pragmas for optimization
	pragmas := []string{
		"PRAGMA journal_mode = WAL",
		"PRAGMA synchronous = NORMAL",
		"PRAGMA cache_size = -10000",
		"PRAGMA temp_store = MEMORY",
		"PRAGMA mmap_size = 268435456",
		"PRAGMA busy_timeout = 30000",
		"PRAGMA wal_autocheckpoint = 1000",
		"PRAGMA optimize",
	}

	for _, pragma := range pragmas {
		if _, err := db.Exec(pragma); err != nil {
			return nil, fmt.Errorf("failed to set pragma %s: %w", pragma, err)
		}
	}

	return db, nil
}

// initOptimizedPebble initializes Pebble with performance optimizations
func initOptimizedPebble(path string) (*pebble.DB, error) {
	opts := &pebble.Options{
		// Cache settings
		Cache: pebble.NewCache(256 << 20), // 256MB
		MemTableSize: 64 << 20,          // 64MB
		MemTableStopWritesThreshold: 4,

		// Performance settings
		MaxOpenFiles: 1000,
		MaxConcurrentCompactions: func() int { return 2 },

		// WAL settings
		WALDir: path + "-wal",

		// Format settings
		Comparer: pebble.DefaultComparer,
	}

	return pebble.Open(path, opts)
}

// initSchema initializes the database schema with optimizations
func (hs *HybridStorage) initSchema() error {
	schemaSQL := `
	-- Schema version tracking
	CREATE TABLE IF NOT EXISTS schema_migrations (
		version INTEGER PRIMARY KEY,
		applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- Core records table with optimizations
	CREATE TABLE IF NOT EXISTS records (
		id TEXT PRIMARY KEY,
		tenant_id TEXT NOT NULL,
		project_id TEXT,
		table_name TEXT NOT NULL,
		data TEXT NOT NULL,
		data_hash TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		version INTEGER DEFAULT 1,
		is_deleted BOOLEAN DEFAULT FALSE,
		created_by TEXT,
		updated_by TEXT
	);

	-- Users table for authentication
	CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		email TEXT UNIQUE NOT NULL,
		password_hash TEXT NOT NULL,
		name TEXT NOT NULL,
		tenant_id TEXT NOT NULL,
		is_active BOOLEAN DEFAULT TRUE,
		email_verified BOOLEAN DEFAULT FALSE,
		last_login_at DATETIME,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		metadata TEXT
	);

	-- Tenants table for multi-tenancy
	CREATE TABLE IF NOT EXISTS tenants (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		domain TEXT UNIQUE,
		settings TEXT,
		is_active BOOLEAN DEFAULT TRUE,
		plan TEXT DEFAULT 'free',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- Projects table
	CREATE TABLE IF NOT EXISTS projects (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		description TEXT,
		tenant_id TEXT NOT NULL,
		owner_id TEXT NOT NULL,
		settings TEXT,
		is_active BOOLEAN DEFAULT TRUE,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- API Keys table
	CREATE TABLE IF NOT EXISTS api_keys (
		id TEXT PRIMARY KEY,
		key_hash TEXT NOT NULL,
		name TEXT NOT NULL,
		user_id TEXT NOT NULL,
		tenant_id TEXT NOT NULL,
		project_id TEXT,
		permissions TEXT,
		expires_at DATETIME,
		last_used_at DATETIME,
		is_active BOOLEAN DEFAULT TRUE,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- User sessions table
	CREATE TABLE IF NOT EXISTS user_sessions (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		token_hash TEXT NOT NULL,
		refresh_token_hash TEXT,
		expires_at DATETIME NOT NULL,
		ip_address TEXT,
		user_agent TEXT,
		is_active BOOLEAN DEFAULT TRUE,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- Row Level Security policies
	CREATE TABLE IF NOT EXISTS rls_policies (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		table_name TEXT NOT NULL,
		tenant_id TEXT NOT NULL,
		definition TEXT NOT NULL,
		is_enabled BOOLEAN DEFAULT TRUE,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- Optimized indexes
	CREATE INDEX IF NOT EXISTS idx_records_tenant_table ON records(tenant_id, table_name);
	CREATE INDEX IF NOT EXISTS idx_records_project_table ON records(project_id, table_name);
	CREATE INDEX IF NOT EXISTS idx_records_created_at ON records(created_at DESC);
	CREATE INDEX IF NOT EXISTS idx_records_updated_at ON records(updated_at DESC);
	CREATE INDEX IF NOT EXISTS idx_records_hash ON records(data_hash);
	CREATE INDEX IF NOT EXISTS idx_records_tenant_project ON records(tenant_id, project_id);
	CREATE INDEX IF NOT EXISTS idx_records_deleted ON records(is_deleted);

	CREATE INDEX IF NOT EXISTS idx_users_tenant_email ON users(tenant_id, email);
	CREATE INDEX IF NOT EXISTS idx_users_active ON users(tenant_id, is_active);
	CREATE INDEX IF NOT EXISTS idx_users_login ON users(last_login_at DESC);

	CREATE INDEX IF NOT EXISTS idx_projects_tenant ON projects(tenant_id, is_active);
	CREATE INDEX IF NOT EXISTS idx_projects_owner ON projects(owner_id);

	CREATE INDEX IF NOT EXISTS idx_api_keys_user ON api_keys(user_id, is_active);
	CREATE INDEX IF NOT EXISTS idx_api_keys_hash ON api_keys(key_hash);
	CREATE INDEX IF NOT EXISTS idx_api_keys_tenant ON api_keys(tenant_id);

	CREATE INDEX IF NOT EXISTS idx_sessions_user ON user_sessions(user_id, expires_at);
	CREATE INDEX IF NOT EXISTS idx_sessions_token ON user_sessions(token_hash);

	CREATE INDEX IF NOT EXISTS idx_rls_policies_table ON rls_policies(table_name, tenant_id, is_enabled);
	`

	_, err := hs.db.Exec(schemaSQL)
	return err
}

// NewStorageCache creates a new storage cache
func NewStorageCache(size int) *StorageCache {
	if size <= 0 {
		size = 1000
	}

	return &StorageCache{
		localCache: make(map[string]*CacheEntry),
		maxSize:    size,
		ttl:        5 * time.Minute,
	}
}

// Get gets value from cache
func (sc *StorageCache) Get(key string) (interface{}, bool) {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	entry, exists := sc.localCache[key]
	if !exists || time.Now().After(entry.ExpiresAt) {
		return nil, false
	}

	entry.HitCount++
	entry.LastHit = time.Now()
	return entry.Value, true
}

// Set sets value in cache
func (sc *StorageCache) Set(key string, value interface{}, ttl time.Duration) {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	// Remove oldest entries if cache is full
	if len(sc.localCache) >= sc.maxSize {
		sc.evictLRU()
	}

	if ttl == 0 {
		ttl = sc.ttl
	}

	sc.localCache[key] = &CacheEntry{
		Key:       key,
		Value:     value,
		ExpiresAt: time.Now().Add(ttl),
		HitCount:  0,
		LastHit:   time.Now(),
	}
}

// Delete removes value from cache
func (sc *StorageCache) Delete(key string) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	delete(sc.localCache, key)
}

// Clear clears all cache entries
func (sc *StorageCache) Clear() {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.localCache = make(map[string]*CacheEntry)
}

// evictLRU removes least recently used entry
func (sc *StorageCache) evictLRU() {
	var oldestKey string
	var oldestTime time.Time = time.Now()

	for key, entry := range sc.localCache {
		if entry.LastHit.Before(oldestTime) {
			oldestTime = entry.LastHit
			oldestKey = key
		}
	}

	if oldestKey != "" {
		delete(sc.localCache, oldestKey)
	}
}

// NewQueryBuilder creates a new query builder
func NewQueryBuilder(table string) *QueryBuilder {
	return &QueryBuilder{
		table: table,
		where: make([]WhereClause, 0),
		orderBy: make([]OrderByClause, 0),
	}
}

// Where adds WHERE clause
func (qb *QueryBuilder) Where(field, operator string, value interface{}) *QueryBuilder {
	op := "AND"
	if len(qb.where) == 0 {
		op = ""
	}

	qb.where = append(qb.where, WhereClause{
		Field:    field,
		Operator: operator,
		Value:    value,
		Op:       op,
	})

	return qb
}

// WhereOr adds OR WHERE clause
func (qb *QueryBuilder) WhereOr(field, operator string, value interface{}) *QueryBuilder {
	qb.where = append(qb.where, WhereClause{
		Field:    field,
		Operator: operator,
		Value:    value,
		Op:       "OR",
	})

	return qb
}

// OrderBy adds ORDER BY clause
func (qb *QueryBuilder) OrderBy(field, dir string) *QueryBuilder {
	qb.orderBy = append(qb.orderBy, OrderByClause{
		Field: field,
		Dir:   dir,
	})

	return qb
}

// Limit sets LIMIT
func (qb *QueryBuilder) Limit(limit int) *QueryBuilder {
	qb.limit = limit
	return qb
}

// Offset sets OFFSET
func (qb *QueryBuilder) Offset(offset int) *QueryBuilder {
	qb.offset = offset
	return qb
}

// Build builds the SQL query
func (qb *QueryBuilder) Build() (string, []interface{}, error) {
	var args []interface{}
	var query strings.Builder

	// SELECT clause
	query.WriteString("SELECT * FROM ")
	query.WriteString(qb.table)

	// WHERE clause
	if len(qb.where) > 0 {
		query.WriteString(" WHERE ")
		whereParts := make([]string, 0)

		for _, clause := range qb.where {
			whereParts = append(whereParts, fmt.Sprintf("%s %s ?", clause.Field, clause.Operator))
			args = append(args, clause.Value)

			if clause.Op != "" {
				whereParts = append(whereParts, clause.Op)
			}
		}

		query.WriteString(strings.Join(whereParts, " "))
	}

	// ORDER BY clause
	if len(qb.orderBy) > 0 {
		query.WriteString(" ORDER BY ")
		orderParts := make([]string, 0)

		for _, clause := range qb.orderBy {
			orderParts = append(orderParts, fmt.Sprintf("%s %s", clause.Field, clause.Dir))
		}

		query.WriteString(strings.Join(orderParts, ", "))
	}

	// LIMIT clause
	if qb.limit > 0 {
		query.WriteString(fmt.Sprintf(" LIMIT %d", qb.limit))
	}

	// OFFSET clause
	if qb.offset > 0 {
		query.WriteString(fmt.Sprintf(" OFFSET %d", qb.offset))
	}

	return query.String(), args, nil
}

// CreateOptimized creates a record with optimizations
func (hs *HybridStorage) CreateOptimized(ctx context.Context, record *Record) error {
	// Calculate data hash for deduplication
	dataHash := calculateDataHash(record.Data)

	// Check cache first
	cacheKey := fmt.Sprintf("record:%s:%s", record.Table, record.ID)
	if _, exists := hs.cache.Get(cacheKey); exists {
		return fmt.Errorf("record already exists")
	}

	// Begin transaction
	tx, err := hs.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Insert into SQLite
	query := `INSERT INTO records
		(id, tenant_id, project_id, table_name, data, data_hash, created_by, updated_by)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

	_, err = tx.ExecContext(ctx, query,
		record.ID, record.TenantID, record.ProjectID,
		record.Table, record.DataJSON, dataHash,
		record.CreatedBy, record.UpdatedBy)
	if err != nil {
		return err
	}

	// Create indexes in Pebble
	if err := hs.createIndexes(ctx, record); err != nil {
		return err
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return err
	}

	// Update cache
	hs.cache.Set(cacheKey, record, 10*time.Minute)

	return nil
}

// GetOptimized gets a record with caching
func (hs *HybridStorage) GetOptimized(ctx context.Context, tenantID, table, id string) (*Record, error) {
	// Check cache first
	cacheKey := fmt.Sprintf("record:%s:%s", table, id)
	if cached, exists := hs.cache.Get(cacheKey); exists {
		if record, ok := cached.(*Record); ok {
			return record, nil
		}
	}

	// Query from SQLite
	query := `SELECT id, tenant_id, project_id, table_name, data, data_hash,
			  created_at, updated_at, version, is_deleted, created_by, updated_by
			  FROM records WHERE id = ? AND table_name = ? AND tenant_id = ? AND is_deleted = FALSE`

	var record Record
	var dataJSON, dataHash string
	var projectID, createdBy, updatedBy sql.NullString
	var deleted bool

	err := hs.db.QueryRowContext(ctx, query, id, table, tenantID).Scan(
		&record.ID, &record.TenantID, &projectID,
		&record.Table, &dataJSON, &dataHash,
		&record.CreatedAt, &record.UpdatedAt, &record.Version,
		&deleted, &createdBy, &updatedBy)

	if err != nil {
		return nil, err
	}

	if projectID.Valid {
		record.ProjectID = projectID.String
	}
	if createdBy.Valid {
		record.CreatedBy = createdBy.String
	}
	if updatedBy.Valid {
		record.UpdatedBy = updatedBy.String
	}

	record.DataJSON = dataJSON
	record.DataHash = dataHash

	// Update cache
	hs.cache.Set(cacheKey, &record, 10*time.Minute)

	return &record, nil
}

// QueryOptimized performs an optimized query with caching
func (hs *HybridStorage) QueryOptimized(ctx context.Context, qb *QueryBuilder) ([]*Record, error) {
	// Build query
	query, args, err := qb.Build()
	if err != nil {
		return nil, err
	}

	// Check cache for complex queries
	if qb.limit <= 100 { // Only cache small result sets
		cacheKey := fmt.Sprintf("query:%s:%v", query, args)
		if cached, exists := hs.cache.Get(cacheKey); exists {
			if records, ok := cached.([]*Record); ok {
				return records, nil
			}
		}
	}

	// Execute query
	rows, err := hs.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []*Record
	for rows.Next() {
		var record Record
		var dataJSON, dataHash string
		var projectID, createdBy, updatedBy sql.NullString
		var deleted bool

		err := rows.Scan(
			&record.ID, &record.TenantID, &projectID,
			&record.Table, &dataJSON, &dataHash,
			&record.CreatedAt, &record.UpdatedAt, &record.Version,
			&deleted, &createdBy, &updatedBy)

		if err != nil {
			return nil, err
		}

		if projectID.Valid {
			record.ProjectID = projectID.String
		}
		if createdBy.Valid {
			record.CreatedBy = createdBy.String
		}
		if updatedBy.Valid {
			record.UpdatedBy = updatedBy.String
		}

		record.DataJSON = dataJSON
		record.DataHash = dataHash

		records = append(records, &record)
	}

	// Update cache for small result sets
	if qb.limit <= 100 && len(records) > 0 {
		cacheKey := fmt.Sprintf("query:%s:%v", query, args)
		hs.cache.Set(cacheKey, records, 2*time.Minute)
	}

	return records, nil
}

// createIndexes creates optimized indexes in Pebble
func (hs *HybridStorage) createIndexes(ctx context.Context, record *Record) error {
	// Create tenant + table index
	key := fmt.Sprintf("idx:tenant_table:%s:%s:%s", record.TenantID, record.Table, record.ID)
	if err := hs.kv.Set([]byte(key), []byte(record.ID), pebble.Sync); err != nil {
		return err
	}

	// Create project + table index
	if record.ProjectID != "" {
		key = fmt.Sprintf("idx:project_table:%s:%s:%s", record.ProjectID, record.Table, record.ID)
		if err := hs.kv.Set([]byte(key), []byte(record.ID), pebble.Sync); err != nil {
			return err
		}
	}

	// Create created_at index (for sorting)
	key = fmt.Sprintf("idx:created_at:%s:%d", record.Table, record.CreatedAt.UnixNano())
	if err := hs.kv.Set([]byte(key), []byte(record.ID), pebble.Sync); err != nil {
		return err
	}

	// Create data hash index (for deduplication)
	key = fmt.Sprintf("idx:data_hash:%s:%s", record.Table, record.DataHash)
	if err := hs.kv.Set([]byte(key), []byte(record.ID), pebble.Sync); err != nil {
		return err
	}

	return nil
}

// startBackgroundTasks starts background maintenance tasks
func (hs *HybridStorage) startBackgroundTasks() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			hs.performMaintenance()
		case <-hs.ctx.Done():
			return
		}
	}
}

// performMaintenance performs background maintenance
func (hs *HybridStorage) performMaintenance() {
	// Clean expired cache entries
	hs.cache.cleanup()

	// Compact Pebble if needed
	hs.kv.Compact([]byte(""), []byte{0xff, 0xff, 0xff, 0xff}, true)

	// Optimize SQLite
	hs.db.Exec("PRAGMA optimize")
}

// cleanup removes expired cache entries
func (sc *StorageCache) cleanup() {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	now := time.Now()
	for key, entry := range sc.localCache {
		if now.After(entry.ExpiresAt) {
			delete(sc.localCache, key)
		}
	}
}

// calculateDataHash calculates hash of record data
func calculateDataHash(data interface{}) string {
	dataBytes, _ := json.Marshal(data)
	return fmt.Sprintf("hash_%x", len(dataBytes)) // Simple hash for now
}

// createDirectory creates directory if it doesn't exist
func createDirectory(path string) error {
	return os.MkdirAll(path, 0755)
}

// Close closes the hybrid storage
func (hs *HybridStorage) Close() error {
	hs.cancel()

	var errors []error

	if hs.db != nil {
		if err := hs.db.Close(); err != nil {
			errors = append(errors, fmt.Errorf("SQLite close error: %w", err))
		}
	}

	if hs.kv != nil {
		if err := hs.kv.Close(); err != nil {
			errors = append(errors, fmt.Errorf("Pebble close error: %w", err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("multiple errors occurred: %v", errors)
	}

	return nil
}