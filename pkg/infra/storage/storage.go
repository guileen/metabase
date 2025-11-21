package storage

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/guileen/metabase/pkg/common/errors"
)

// StorageType represents different storage types
type StorageType string

const (
	StorageTypeMemory StorageType = "memory"
	StorageTypeFile   StorageType = "file"
	StorageTypeSQLite StorageType = "sqlite"
	StorageTypePebble StorageType = "pebble"
)

// Config represents storage configuration
type Config struct {
	Type    StorageType            `json:"type"`
	Path    string                 `json:"path,omitempty"`
	Options map[string]interface{} `json:"options,omitempty"`
}

// Engine is an alias for Storage interface for backward compatibility
type Engine = Storage

// QueryOptions represents query options for storage operations
type QueryOptions struct {
	Limit  int                    `json:"limit,omitempty"`
	Offset int                    `json:"offset,omitempty"`
	Where  map[string]interface{} `json:"where,omitempty"`
	Order  string                 `json:"order,omitempty"`
}

// Record represents a storage record
type Record struct {
	ID        string                 `json:"id"`
	Data      map[string]interface{} `json:"data"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
}

// QueryResult represents the result of a storage query
type QueryResult struct {
	Records []*Record `json:"records"`
	Total   int64     `json:"total"`
}

// Storage represents a storage interface
type Storage interface {
	// Get retrieves a value by key
	Get(ctx context.Context, key string) ([]byte, error)

	// Set stores a value with key
	Set(ctx context.Context, key string, value []byte) error

	// Delete removes a key-value pair
	Delete(ctx context.Context, key string) error

	// Exists checks if a key exists
	Exists(ctx context.Context, key string) (bool, error)

	// List returns all keys with a given prefix
	List(ctx context.Context, prefix string) ([]string, error)

	// Query performs a complex query on stored data
	Query(ctx context.Context, collection string, options *QueryOptions) (*QueryResult, error)

	// Clear removes all data
	Clear(ctx context.Context) error

	// Stats returns storage statistics
	Stats() map[string]interface{}

	// Close closes the storage
	Close() error
}

// MemoryStorage is an in-memory storage implementation
type MemoryStorage struct {
	data  map[string][]byte
	mutex sync.RWMutex
	stats *StorageStats
}

// StorageStats represents storage statistics
type StorageStats struct {
	KeysCount   int64     `json:"keys_count"`
	BytesSize   int64     `json:"bytes_size"`
	LastAccess  time.Time `json:"last_access"`
	CreatedAt   time.Time `json:"created_at"`
	ReadCount   int64     `json:"read_count"`
	WriteCount  int64     `json:"write_count"`
	DeleteCount int64     `json:"delete_count"`
}

// NewMemoryStorage creates a new in-memory storage
func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		data:  make(map[string][]byte),
		stats: &StorageStats{CreatedAt: time.Now()},
	}
}

// Get retrieves a value by key
func (m *MemoryStorage) Get(ctx context.Context, key string) ([]byte, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if key == "" {
		return nil, errors.InvalidInput("key is required")
	}

	value, exists := m.data[key]
	if !exists {
		return nil, errors.NotFound(fmt.Sprintf("key '%s'", key))
	}

	m.stats.ReadCount++
	m.stats.LastAccess = time.Now()

	// Return a copy to avoid modification
	result := make([]byte, len(value))
	copy(result, value)
	return result, nil
}

// Set stores a value with key
func (m *MemoryStorage) Set(ctx context.Context, key string, value []byte) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if key == "" {
		return errors.InvalidInput("key is required")
	}

	// Update stats for existing key deletion
	if existing, exists := m.data[key]; exists {
		m.stats.BytesSize -= int64(len(existing))
	} else {
		m.stats.KeysCount++
	}

	m.data[key] = make([]byte, len(value))
	copy(m.data[key], value)

	m.stats.BytesSize += int64(len(value))
	m.stats.WriteCount++
	m.stats.LastAccess = time.Now()

	return nil
}

// Delete removes a key-value pair
func (m *MemoryStorage) Delete(ctx context.Context, key string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if key == "" {
		return errors.InvalidInput("key is required")
	}

	value, exists := m.data[key]
	if !exists {
		return errors.NotFound(fmt.Sprintf("key '%s'", key))
	}

	delete(m.data, key)
	m.stats.KeysCount--
	m.stats.BytesSize -= int64(len(value))
	m.stats.DeleteCount++
	m.stats.LastAccess = time.Now()

	return nil
}

// Exists checks if a key exists
func (m *MemoryStorage) Exists(ctx context.Context, key string) (bool, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if key == "" {
		return false, errors.InvalidInput("key is required")
	}

	_, exists := m.data[key]
	m.stats.ReadCount++
	m.stats.LastAccess = time.Now()

	return exists, nil
}

// List returns all keys with a given prefix
func (m *MemoryStorage) List(ctx context.Context, prefix string) ([]string, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	var keys []string
	for key := range m.data {
		if len(prefix) == 0 || len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			keys = append(keys, key)
		}
	}

	m.stats.ReadCount++
	m.stats.LastAccess = time.Now()

	return keys, nil
}

// Clear removes all data
func (m *MemoryStorage) Clear(ctx context.Context) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.data = make(map[string][]byte)
	m.stats.KeysCount = 0
	m.stats.BytesSize = 0
	m.stats.LastAccess = time.Now()

	return nil
}

// Query performs a complex query on stored data
func (m *MemoryStorage) Query(ctx context.Context, collection string, options *QueryOptions) (*QueryResult, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	var records []*Record

	// For this simple implementation, we'll just return all keys as records
	for key, value := range m.data {
		record := &Record{
			ID:   key,
			Data: map[string]interface{}{"value": string(value)},
		}
		records = append(records, record)
	}

	// Apply limit and offset
	if options != nil {
		if options.Offset > 0 && options.Offset < len(records) {
			records = records[options.Offset:]
		}
		if options.Limit > 0 && options.Limit < len(records) {
			records = records[:options.Limit]
		}
	}

	return &QueryResult{
		Records: records,
		Total:   int64(len(records)),
	}, nil
}

// Stats returns storage statistics
func (m *MemoryStorage) Stats() map[string]interface{} {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	return map[string]interface{}{
		"type":         StorageTypeMemory,
		"keys_count":   m.stats.KeysCount,
		"bytes_size":   m.stats.BytesSize,
		"last_access":  m.stats.LastAccess,
		"created_at":   m.stats.CreatedAt,
		"read_count":   m.stats.ReadCount,
		"write_count":  m.stats.WriteCount,
		"delete_count": m.stats.DeleteCount,
	}
}

// Close closes the storage
func (m *MemoryStorage) Close() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.data = make(map[string][]byte)
	return nil
}

// NewStorage creates a new storage instance based on configuration
func NewStorage(config *Config) (Storage, error) {
	if config == nil {
		config = &Config{Type: StorageTypeMemory}
	}

	switch config.Type {
	case StorageTypeMemory:
		return NewMemoryStorage(), nil
	case StorageTypeFile:
		return NewFileStorage(config.Path, config.Options)
	case StorageTypeSQLite:
		return NewSQLiteStorage(config.Path, config.Options)
	case StorageTypePebble:
		return NewPebbleStorage(config.Path, config.Options)
	default:
		return nil, errors.InvalidInput(fmt.Sprintf("unsupported storage type: %s", config.Type))
	}
}

// FileStorage represents file-based storage (placeholder implementation)
type FileStorage struct {
	path  string
	stats *StorageStats
}

// NewFileStorage creates a new file-based storage
func NewFileStorage(path string, options map[string]interface{}) (*FileStorage, error) {
	if path == "" {
		path = "./data/storage"
	}
	return &FileStorage{
		path:  path,
		stats: &StorageStats{CreatedAt: time.Now()},
	}, nil
}

// Implement FileStorage methods (placeholder for now)
func (f *FileStorage) Get(ctx context.Context, key string) ([]byte, error) {
	return nil, errors.Internal("file storage not implemented")
}

func (f *FileStorage) Set(ctx context.Context, key string, value []byte) error {
	return errors.Internal("file storage not implemented")
}

func (f *FileStorage) Delete(ctx context.Context, key string) error {
	return errors.Internal("file storage not implemented")
}

func (f *FileStorage) Exists(ctx context.Context, key string) (bool, error) {
	return false, errors.Internal("file storage not implemented")
}

func (f *FileStorage) List(ctx context.Context, prefix string) ([]string, error) {
	return nil, errors.Internal("file storage not implemented")
}

func (f *FileStorage) Clear(ctx context.Context) error {
	return errors.Internal("file storage not implemented")
}

func (f *FileStorage) Stats() map[string]interface{} {
	return map[string]interface{}{
		"type":       StorageTypeFile,
		"path":       f.path,
		"created_at": f.stats.CreatedAt,
	}
}

func (f *FileStorage) Close() error {
	return nil
}

// Query performs a complex query on file storage data
func (f *FileStorage) Query(ctx context.Context, collection string, options *QueryOptions) (*QueryResult, error) {
	return nil, errors.Internal("file storage Query not implemented")
}

// SQLiteStorage represents SQLite-based storage (placeholder implementation)
type SQLiteStorage struct {
	path  string
	stats *StorageStats
}

// NewSQLiteStorage creates a new SQLite-based storage
func NewSQLiteStorage(path string, options map[string]interface{}) (*SQLiteStorage, error) {
	if path == "" {
		path = "./data/storage.db"
	}
	return &SQLiteStorage{
		path:  path,
		stats: &StorageStats{CreatedAt: time.Now()},
	}, nil
}

// Implement SQLiteStorage methods (placeholder for now)
func (s *SQLiteStorage) Get(ctx context.Context, key string) ([]byte, error) {
	return nil, errors.Internal("SQLite storage not implemented")
}

func (s *SQLiteStorage) Set(ctx context.Context, key string, value []byte) error {
	return errors.Internal("SQLite storage not implemented")
}

func (s *SQLiteStorage) Delete(ctx context.Context, key string) error {
	return errors.Internal("SQLite storage not implemented")
}

func (s *SQLiteStorage) Exists(ctx context.Context, key string) (bool, error) {
	return false, errors.Internal("SQLite storage not implemented")
}

func (s *SQLiteStorage) List(ctx context.Context, prefix string) ([]string, error) {
	return nil, errors.Internal("SQLite storage not implemented")
}

func (s *SQLiteStorage) Clear(ctx context.Context) error {
	return errors.Internal("SQLite storage not implemented")
}

func (s *SQLiteStorage) Stats() map[string]interface{} {
	return map[string]interface{}{
		"type":       StorageTypeSQLite,
		"path":       s.path,
		"created_at": s.stats.CreatedAt,
	}
}

func (s *SQLiteStorage) Close() error {
	return nil
}

// Query performs a complex query on SQLite storage data
func (s *SQLiteStorage) Query(ctx context.Context, collection string, options *QueryOptions) (*QueryResult, error) {
	return nil, errors.Internal("SQLite storage Query not implemented")
}

// PebbleStorage represents Pebble-based storage (placeholder implementation)
type PebbleStorage struct {
	path  string
	stats *StorageStats
}

// NewPebbleStorage creates a new Pebble-based storage
func NewPebbleStorage(path string, options map[string]interface{}) (*PebbleStorage, error) {
	if path == "" {
		path = "./data/pebble"
	}
	return &PebbleStorage{
		path:  path,
		stats: &StorageStats{CreatedAt: time.Now()},
	}, nil
}

// Implement PebbleStorage methods (placeholder for now)
func (p *PebbleStorage) Get(ctx context.Context, key string) ([]byte, error) {
	return nil, errors.Internal("Pebble storage not implemented")
}

func (p *PebbleStorage) Set(ctx context.Context, key string, value []byte) error {
	return errors.Internal("Pebble storage not implemented")
}

func (p *PebbleStorage) Delete(ctx context.Context, key string) error {
	return errors.Internal("Pebble storage not implemented")
}

func (p *PebbleStorage) Exists(ctx context.Context, key string) (bool, error) {
	return false, errors.Internal("Pebble storage not implemented")
}

func (p *PebbleStorage) List(ctx context.Context, prefix string) ([]string, error) {
	return nil, errors.Internal("Pebble storage not implemented")
}

func (p *PebbleStorage) Clear(ctx context.Context) error {
	return errors.Internal("Pebble storage not implemented")
}

func (p *PebbleStorage) Stats() map[string]interface{} {
	return map[string]interface{}{
		"type":       StorageTypePebble,
		"path":       p.path,
		"created_at": p.stats.CreatedAt,
	}
}

func (p *PebbleStorage) Close() error {
	return nil
}

// Query performs a complex query on Pebble storage data
func (p *PebbleStorage) Query(ctx context.Context, collection string, options *QueryOptions) (*QueryResult, error) {
	return nil, errors.Internal("Pebble storage Query not implemented")
}
