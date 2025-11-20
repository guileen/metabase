package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/cockroachdb/pebble"
	_ "github.com/mattn/go-sqlite3"
)

// StorageType represents the type of storage
type StorageType string

const (
	StorageTypeSQLite   StorageType = "sqlite"
	StorageTypePebble   StorageType = "pebble"
	StorageTypeMemory   StorageType = "memory"
)

// Config represents storage engine configuration
type Config struct {
	SQLitePath   string `json:"sqlite_path"`
	PebblePath   string `json:"pebble_path"`
	RedisURL     string `json:"redis_url"`
	CacheEnabled bool   `json:"cache_enabled"`
	CacheSize    int    `json:"cache_size"`
}

// Engine represents the unified storage engine
type Engine struct {
	db     *sql.DB
	kv     *pebble.DB
	config *Config
	ctx    context.Context
	cancel context.CancelFunc
}

// Record represents a database record
type Record struct {
	ID          string                 `json:"id"`
	TenantID    string                 `json:"tenant_id,omitempty"`
	ProjectID   string                 `json:"project_id,omitempty"`
	Table       string                 `json:"table"`
	Data        map[string]interface{} `json:"data"`
	DataJSON    string                 `json:"data_json,omitempty"`
	DataHash    string                 `json:"data_hash,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	Version     int64                  `json:"version"`
	IsDeleted   bool                   `json:"is_deleted"`
	CreatedBy   string                 `json:"created_by,omitempty"`
	UpdatedBy   string                 `json:"updated_by,omitempty"`
}

// Index represents a database index
type Index struct {
	Name   string   `json:"name"`
	Table  string   `json:"table"`
	Fields []string `json:"fields"`
	Unique bool     `json:"unique"`
}

// QueryOptions represents query options
type QueryOptions struct {
	Limit    int                    `json:"limit"`
	Offset   int                    `json:"offset"`
	OrderBy  string                 `json:"order_by"`
	OrderDir string                 `json:"order_dir"`
	Where    map[string]interface{} `json:"where"`
	Select   []string               `json:"select"`
}

// QueryResult represents the result of a query
type QueryResult struct {
	Records []Record            `json:"records"`
	Total   int                 `json:"total"`
	HasNext bool                `json:"has_next"`
	Meta    map[string]interface{} `json:"meta"`
}

// NewConfig creates a new storage configuration with defaults
func NewConfig() *Config {
	return &Config{
		SQLitePath:   "./data/metabase.db",
		PebblePath:   "./data/pebble",
		RedisURL:     "",
		CacheEnabled: false,
		CacheSize:    1000,
	}
}

// NewEngine creates a new storage engine
func NewEngine(config *Config) (*Engine, error) {
	ctx, cancel := context.WithCancel(context.Background())

	engine := &Engine{
		config: config,
		ctx:    ctx,
		cancel: cancel,
	}

	// Ensure data directories exist
	if err := os.MkdirAll(filepath.Dir(config.SQLitePath), 0755); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create SQLite directory: %w", err)
	}

	if err := os.MkdirAll(config.PebblePath, 0755); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create Pebble directory: %w", err)
	}

	// Initialize SQLite
	db, err := sql.Open("sqlite3", config.SQLitePath)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to open SQLite database: %w", err)
	}

	// Configure SQLite connection
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Initialize Pebble
	kv, err := pebble.Open(config.PebblePath, &pebble.Options{
		MaxOpenFiles: 1000,
		LBaseMaxBytes: 64 << 20, // 64MB
	})
	if err != nil {
		db.Close()
		cancel()
		return nil, fmt.Errorf("failed to open Pebble database: %w", err)
	}

	engine.db = db
	engine.kv = kv

	// Initialize schema
	if err := engine.initSchema(); err != nil {
		db.Close()
		kv.Close()
		cancel()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	log.Printf("Storage engine initialized: SQLite=%s, Pebble=%s", config.SQLitePath, config.PebblePath)
	return engine, nil
}

// initSchema initializes the database schema
func (e *Engine) initSchema() error {
	schemaSQL := `
	CREATE TABLE IF NOT EXISTS schema_migrations (
		version INTEGER PRIMARY KEY,
		applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS records (
		id TEXT PRIMARY KEY,
		table_name TEXT NOT NULL,
		data TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		version INTEGER DEFAULT 1
	);

	CREATE TABLE IF NOT EXISTS indexes (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL UNIQUE,
		table_name TEXT NOT NULL,
		fields TEXT NOT NULL,
		unique_flag BOOLEAN DEFAULT FALSE,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_records_table ON records(table_name);
	CREATE INDEX IF NOT EXISTS idx_records_updated ON records(updated_at);
	CREATE INDEX IF NOT EXISTS idx_records_created ON records(created_at);
	`

	_, err := e.db.Exec(schemaSQL)
	return err
}

// Create creates a new record
func (e *Engine) Create(ctx context.Context, table string, data map[string]interface{}) (*Record, error) {
	now := time.Now()
	record := &Record{
		ID:        generateID(),
		Table:     table,
		Data:      data,
		CreatedAt: now,
		UpdatedAt: now,
		Version:   1,
	}

	dataJSON, err := json.Marshal(record.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal record data: %w", err)
	}

	query := `INSERT INTO records (id, table_name, data, created_at, updated_at, version) VALUES (?, ?, ?, ?, ?, ?)`
	_, err = e.db.ExecContext(ctx, query, record.ID, record.Table, string(dataJSON), record.CreatedAt, record.UpdatedAt, record.Version)
	if err != nil {
		return nil, fmt.Errorf("failed to insert record: %w", err)
	}

	// Update indexes in Pebble
	if err := e.updateIndexes(ctx, record, "create"); err != nil {
		log.Printf("Warning: failed to update indexes: %v", err)
	}

	log.Printf("Created record %s in table %s", record.ID, table)
	return record, nil
}

// Get retrieves a record by ID
func (e *Engine) Get(ctx context.Context, table, id string) (*Record, error) {
	var record Record
	var dataJSON string

	query := `SELECT id, table_name, data, created_at, updated_at, version FROM records WHERE table_name = ? AND id = ?`
	err := e.db.QueryRowContext(ctx, query, table, id).Scan(
		&record.ID,
		&record.Table,
		&dataJSON,
		&record.CreatedAt,
		&record.UpdatedAt,
		&record.Version,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("record not found: %s/%s", table, id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query record: %w", err)
	}

	if err := json.Unmarshal([]byte(dataJSON), &record.Data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal record data: %w", err)
	}

	return &record, nil
}

// Update updates an existing record
func (e *Engine) Update(ctx context.Context, table, id string, data map[string]interface{}) (*Record, error) {
	// Get existing record
	existing, err := e.Get(ctx, table, id)
	if err != nil {
		return nil, err
	}

	// Merge data
	for key, value := range data {
		existing.Data[key] = value
	}

	// Update metadata
	existing.UpdatedAt = time.Now()
	existing.Version++

	dataJSON, err := json.Marshal(existing.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal record data: %w", err)
	}

	query := `UPDATE records SET data = ?, updated_at = ?, version = ? WHERE table_name = ? AND id = ?`
	_, err = e.db.ExecContext(ctx, query, string(dataJSON), existing.UpdatedAt, existing.Version, table, id)
	if err != nil {
		return nil, fmt.Errorf("failed to update record: %w", err)
	}

	// Update indexes in Pebble
	if err := e.updateIndexes(ctx, existing, "update"); err != nil {
		log.Printf("Warning: failed to update indexes: %v", err)
	}

	log.Printf("Updated record %s in table %s", id, table)
	return existing, nil
}

// Delete deletes a record by ID
func (e *Engine) Delete(ctx context.Context, table, id string) error {
	// Get existing record for index cleanup
	existing, err := e.Get(ctx, table, id)
	if err != nil {
		return err
	}

	query := `DELETE FROM records WHERE table_name = ? AND id = ?`
	_, err = e.db.ExecContext(ctx, query, table, id)
	if err != nil {
		return fmt.Errorf("failed to delete record: %w", err)
	}

	// Remove from indexes in Pebble
	if err := e.updateIndexes(ctx, existing, "delete"); err != nil {
		log.Printf("Warning: failed to update indexes: %v", err)
	}

	log.Printf("Deleted record %s from table %s", id, table)
	return nil
}

// Query performs a query with options
func (e *Engine) Query(ctx context.Context, table string, options *QueryOptions) (*QueryResult, error) {
	if options == nil {
		options = &QueryOptions{
			Limit:    100,
			Offset:   0,
			OrderBy:  "created_at",
			OrderDir: "DESC",
		}
	}

	// Build query
	query := "SELECT id, table_name, data, created_at, updated_at, version FROM records WHERE table_name = ?"
	args := []interface{}{table}

	// Add WHERE conditions
	if options.Where != nil {
		for field, value := range options.Where {
			// Simple JSON field query (SQLite specific)
			query += fmt.Sprintf(" AND JSON_EXTRACT(data, '$.%s') = ?", field)
			args = append(args, value)
		}
	}

	// Add ORDER BY
	if options.OrderBy != "" {
		dir := "ASC"
		if options.OrderDir == "DESC" {
			dir = "DESC"
		}
		query += fmt.Sprintf(" ORDER BY %s %s", options.OrderBy, dir)
	}

	// Add LIMIT and OFFSET
	query += " LIMIT ? OFFSET ?"
	args = append(args, options.Limit, options.Offset)

	// Execute query
	rows, err := e.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query records: %w", err)
	}
	defer rows.Close()

	var records []Record
	for rows.Next() {
		var record Record
		var dataJSON string

		err := rows.Scan(
			&record.ID,
			&record.Table,
			&dataJSON,
			&record.CreatedAt,
			&record.UpdatedAt,
			&record.Version,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan record: %w", err)
		}

		if err := json.Unmarshal([]byte(dataJSON), &record.Data); err != nil {
			return nil, fmt.Errorf("failed to unmarshal record data: %w", err)
		}

		records = append(records, record)
	}

	// Get total count
	countQuery := "SELECT COUNT(*) FROM records WHERE table_name = ?"
	countArgs := []interface{}{table}
	if options.Where != nil {
		for field, value := range options.Where {
			countQuery += fmt.Sprintf(" AND JSON_EXTRACT(data, '$.%s') = ?", field)
			countArgs = append(countArgs, value)
		}
	}

	var total int
	err = e.db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("failed to get total count: %w", err)
	}

	result := &QueryResult{
		Records: records,
		Total:   total,
		HasNext: options.Offset+len(records) < total,
		Meta: map[string]interface{}{
			"limit":  options.Limit,
			"offset": options.Offset,
			"table":  table,
		},
	}

	return result, nil
}

// Set stores a key-value pair in Pebble
func (e *Engine) Set(ctx context.Context, key string, value []byte) error {
	return e.kv.Set([]byte(key), value, pebble.Sync)
}

// Get retrieves a value from Pebble by key
func (e *Engine) GetKV(ctx context.Context, key string) ([]byte, error) {
	value, closer, err := e.kv.Get([]byte(key))
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	result := make([]byte, len(value))
	copy(result, value)
	return result, nil
}

// Delete removes a key-value pair from Pebble
func (e *Engine) DeleteKV(ctx context.Context, key string) error {
	return e.kv.Delete([]byte(key), pebble.Sync)
}

// CreateIndex creates a new index
func (e *Engine) CreateIndex(ctx context.Context, index *Index) error {
	fieldsJSON, _ := json.Marshal(index.Fields)

	query := `INSERT INTO indexes (name, table_name, fields, unique_flag) VALUES (?, ?, ?, ?)`
	_, err := e.db.ExecContext(ctx, query, index.Name, index.Table, string(fieldsJSON), index.Unique)
	if err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}

	log.Printf("Created index %s on table %s", index.Name, index.Table)
	return nil
}

// ListIndexes returns all indexes for a table
func (e *Engine) ListIndexes(ctx context.Context, table string) ([]Index, error) {
	query := `SELECT name, table_name, fields, unique_flag FROM indexes WHERE table_name = ?`
	rows, err := e.db.QueryContext(ctx, query, table)
	if err != nil {
		return nil, fmt.Errorf("failed to list indexes: %w", err)
	}
	defer rows.Close()

	var indexes []Index
	for rows.Next() {
		var index Index
		var fieldsJSON string

		err := rows.Scan(&index.Name, &index.Table, &fieldsJSON, &index.Unique)
		if err != nil {
			return nil, fmt.Errorf("failed to scan index: %w", err)
		}

		if err := json.Unmarshal([]byte(fieldsJSON), &index.Fields); err != nil {
			return nil, fmt.Errorf("failed to unmarshal index fields: %w", err)
		}

		indexes = append(indexes, index)
	}

	return indexes, nil
}

// updateIndexes updates Pebble indexes based on the operation
func (e *Engine) updateIndexes(ctx context.Context, record *Record, operation string) error {
	// Get indexes for the table
	indexes, err := e.ListIndexes(ctx, record.Table)
	if err != nil {
		return err
	}

	for _, index := range indexes {
		// Build index key
		indexKey := e.buildIndexKey(index, record)

		switch operation {
		case "create", "update":
			// Store record ID in index
			if err := e.Set(ctx, indexKey, []byte(record.ID)); err != nil {
				return err
			}
		case "delete":
			// Remove from index
			if err := e.DeleteKV(ctx, indexKey); err != nil {
				return err
			}
		}
	}

	return nil
}

// buildIndexKey builds an index key for a record
func (e *Engine) buildIndexKey(index Index, record *Record) string {
	var keyParts []string
	keyParts = append(keyParts, "idx", index.Table, index.Name)

	// Add field values to key
	for _, field := range index.Fields {
		if value, exists := record.Data[field]; exists {
			keyParts = append(keyParts, fmt.Sprintf("%v", value))
		} else {
			keyParts = append(keyParts, "")
		}
	}

	return fmt.Sprintf("%s#%s", record.ID, keyParts)
}

// Stats returns storage engine statistics
func (e *Engine) Stats() map[string]interface{} {
	stats := make(map[string]interface{})

	// SQLite stats
	if e.db != nil {
		dbStats := e.db.Stats()
		stats["sqlite"] = map[string]interface{}{
			"open_connections": dbStats.OpenConnections,
			"in_use":          dbStats.InUse,
			"idle":            dbStats.Idle,
		}
	}

	// Pebble stats
	if e.kv != nil {
		stats["pebble"] = map[string]interface{}{
			"flushed": e.kv.Metrics().Flush.Count,
			"compacted": e.kv.Metrics().Compact.Count,
		}
	}

	return stats
}

// Close closes the storage engine
func (e *Engine) Close() error {
	var errors []error

	if e.db != nil {
		if err := e.db.Close(); err != nil {
			errors = append(errors, fmt.Errorf("failed to close SQLite: %w", err))
		}
	}

	if e.kv != nil {
		if err := e.kv.Close(); err != nil {
			errors = append(errors, fmt.Errorf("failed to close Pebble: %w", err))
		}
	}

	if e.cancel != nil {
		e.cancel()
	}

	if len(errors) > 0 {
		return fmt.Errorf("multiple errors occurred: %v", errors)
	}

	return nil
}

// generateID generates a unique ID
func generateID() string {
	return fmt.Sprintf("rec_%d", time.Now().UnixNano())
}

// BeginTx begins a transaction
func (e *Engine) BeginTx(ctx context.Context) (*sql.Tx, error) {
	return e.db.BeginTx(ctx, nil)
}