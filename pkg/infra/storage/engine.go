package storage

import ("context"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time")

// StorageType represents file storage backend
type StorageType string

const (
	StorageLocal    StorageType = "local"
	StorageS3       StorageType = "s3"
	StorageMinIO    StorageType = "minio"
	StorageGridFS   StorageType = "gridfs"
	StoragePebbleFS StorageType = "pebblefs" // Embedded distributed filesystem
)

// FileType represents file type category
type FileType string

const (
	FileTypeImage  FileType = "image"
	FileTypeDoc    FileType = "document"
	FileTypeVideo  FileType = "video"
	FileTypeAudio  FileType = "audio"
	FileTypeArchive FileType = "archive"
	FileTypeOther  FileType = "other"
)

// FileMetadata represents file metadata
type FileMetadata struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	OriginalName string                 `json:"original_name"`
	Path         string                 `json:"path"`
	Size         int64                  `json:"size"`
	MimeType     string                 `json:"mime_type"`
	FileType     FileType               `json:"file_type"`
	MD5Hash      string                 `json:"md5_hash"`
	SHA256Hash   string                 `json:"sha256_hash"`
	StorageType  StorageType            `json:"storage_type"`
	TenantID     string                 `json:"tenant_id,omitempty"`
	ProjectID    string                 `json:"project_id,omitempty"`
	CreatedBy    string                 `json:"created_by,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
	AccessCount  int64                  `json:"access_count"`
	LastAccessed *time.Time             `json:"last_accessed,omitempty"`
	Tags         []string               `json:"tags,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	IsPublic     bool                   `json:"is_public"`
	ExpiresAt    *time.Time             `json:"expires_at,omitempty"`
}

// UploadOptions represents upload options
type UploadOptions struct {
	TenantID     string                 `json:"tenant_id,omitempty"`
	ProjectID    string                 `json:"project_id,omitempty"`
	CreatedBy    string                 `json:"created_by,omitempty"`
	Tags         []string               `json:"tags,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	IsPublic     bool                   `json:"is_public"`
	ExpiresAt    *time.Time             `json:"expires_at,omitempty"`
	MaxSize      int64                  `json:"max_size"`
	AllowedTypes []string               `json:"allowed_types"`
}

// QueryOptions represents file query options
type QueryOptions struct {
	TenantID    string            `json:"tenant_id,omitempty"`
	ProjectID   string            `json:"project_id,omitempty"`
	FileType    FileType          `json:"file_type,omitempty"`
	Tags        []string          `json:"tags,omitempty"`
	IsPublic    *bool             `json:"is_public,omitempty"`
	CreatedBy   string            `json:"created_by,omitempty"`
	Search      string            `json:"search,omitempty"`
	OrderBy     string            `json:"order_by"`
	OrderDir    string            `json:"order_dir"`
	Limit       int               `json:"limit"`
	Offset      int               `json:"offset"`
	DateRange   map[string]string `json:"date_range,omitempty"`
}

// Config represents file engine configuration
type Config struct {
	StorageType  StorageType          `json:"storage_type"`
	LocalPath    string              `json:"local_path"`
	S3Config     *S3Config           `json:"s3_config,omitempty"`
	MinIOConfig  *MinIOConfig        `json:"minio_config,omitempty"`
	MaxFileSize  int64               `json:"max_file_size"`
	AllowedMimes []string            `json:"allowed_mimes"`
	CleanupAfter time.Duration       `json:"cleanup_after"`
	EnableDedup  bool                `json:"enable_dedup"`
	Compression  *CompressionConfig  `json:"compression,omitempty"`
	CacheConfig  *CacheConfig        `json:"cache,omitempty"`
}

// S3Config represents S3 configuration
type S3Config struct {
	Bucket          string `json:"bucket"`
	Region          string `json:"region"`
	AccessKeyID     string `json:"access_key_id"`
	SecretAccessKey string `json:"secret_access_key"`
	Endpoint        string `json:"endpoint,omitempty"`
	UseSSL          bool   `json:"use_ssl"`
}

// MinIOConfig represents MinIO configuration
type MinIOConfig struct {
	Endpoint        string `json:"endpoint"`
	AccessKeyID     string `json:"access_key_id"`
	SecretAccessKey string `json:"secret_access_key"`
	Bucket          string `json:"bucket"`
	UseSSL          bool   `json:"use_ssl"`
}

// CompressionConfig represents compression configuration
type CompressionConfig struct {
	Enabled       bool   `json:"enabled"`
	Algorithm     string `json:"algorithm"`     // gzip, brotli, zstd
	Level         int    `json:"level"`         // 1-9
	MinSize       int64  `json:"min_size"`      // Minimum size to compress
	ExcludedTypes []string `json:"excluded_types"`
}

// CacheConfig represents cache configuration
type CacheConfig struct {
	Enabled  bool          `json:"enabled"`
	TTL      time.Duration `json:"ttl"`
	MaxSize  int64         `json:"max_size"`
	Backend  string        `json:"backend"` // redis, memory
}

// Engine represents the file storage engine
type Engine struct {
	config  *Config
	storage storage.Engine
	ctx     context.Context
	cancel  context.CancelFunc
}

// NewConfig creates a new file engine configuration
func NewConfig() *Config {
	return &Config{
		StorageType:  StorageLocal,
		LocalPath:    "./uploads",
		MaxFileSize:  100 * 1024 * 1024, // 100MB
		AllowedMimes: []string{
			"image/jpeg", "image/png", "image/gif", "image/webp",
			"application/pdf", "text/plain", "text/csv",
			"application/json", "application/xml",
			"application/zip", "application/x-tar",
		},
		CleanupAfter: 30 * 24 * time.Hour, // 30 days
		EnableDedup:  true,
		Compression: &CompressionConfig{
			Enabled:   true,
			Algorithm: "gzip",
			Level:     6,
			MinSize:   1024, // 1KB
		},
		CacheConfig: &CacheConfig{
			Enabled: true,
			TTL:     24 * time.Hour,
			MaxSize: 100 * 1024 * 1024, // 100MB
			Backend: "memory",
		},
	}
}

// NewEngine creates a new file storage engine
func NewEngine(config *Config, storageEngine storage.Engine) (*Engine, error) {
	ctx, cancel := context.WithCancel(context.Background())

	engine := &Engine{
		config:  config,
		storage: storageEngine,
		ctx:     ctx,
		cancel:  cancel,
	}

	// Ensure directories exist
	if config.StorageType == StorageLocal {
		dirs := []string{
			config.LocalPath,
			filepath.Join(config.LocalPath, "temp"),
			filepath.Join(config.LocalPath, "files"),
			filepath.Join(config.LocalPath, "images"),
			filepath.Join(config.LocalPath, "documents"),
			filepath.Join(config.LocalPath, "cache"),
		}
		for _, dir := range dirs {
			if err := os.MkdirAll(dir, 0755); err != nil {
				cancel()
				return nil, fmt.Errorf("failed to create directory %s: %w", dir, err)
			}
		}
	}

	// Start cleanup goroutine
	go engine.cleanupExpiredFiles()

	return engine, nil
}

// Upload handles file upload
func (e *Engine) Upload(ctx context.Context, file multipart.File, header *multipart.FileHeader, opts *UploadOptions) (*FileMetadata, error) {
	// Validate file
	if err := e.validateFile(header, opts); err != nil {
		return nil, err
	}

	// Read file content
	content, err := io.ReadAll(file)
	file.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Calculate hashes
	md5Hash := md5.Sum(content)
	sha256Hash := sha256.Sum256(content)

	// Check for duplicate if deduplication is enabled
	if e.config.EnableDedup {
		if existing, err := e.findBySHA256(ctx, hex.EncodeToString(sha256Hash[:])); err == nil {
			return existing, nil // Return existing file
		}
	}

	// Determine file type
	fileType := e.determineFileType(header.Header.Get("Content-Type"), header.Filename)

	// Generate file ID and path
	fileID := e.generateFileID()
	filePath := e.generatePath(fileID, header.Filename, fileType)

	// Store file
	storagePath, err := e.storeFile(ctx, content, filePath, fileType)
	if err != nil {
		return nil, fmt.Errorf("failed to store file: %w", err)
	}

	// Create metadata
	metadata := &FileMetadata{
		ID:           fileID,
		Name:         filepath.Base(storagePath),
		OriginalName: header.Filename,
		Path:         storagePath,
		Size:         header.Size,
		MimeType:     header.Header.Get("Content-Type"),
		FileType:     fileType,
		MD5Hash:      hex.EncodeToString(md5Hash[:]),
		SHA256Hash:   hex.EncodeToString(sha256Hash[:]),
		StorageType:  e.config.StorageType,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		AccessCount:  0,
	}

	// Apply options
	if opts != nil {
		metadata.TenantID = opts.TenantID
		metadata.ProjectID = opts.ProjectID
		metadata.CreatedBy = opts.CreatedBy
		metadata.Tags = opts.Tags
		metadata.Metadata = opts.Metadata
		metadata.IsPublic = opts.IsPublic
		metadata.ExpiresAt = opts.ExpiresAt
	}

	// Save metadata to storage
	if err := e.saveMetadata(ctx, metadata); err != nil {
		return nil, fmt.Errorf("failed to save file metadata: %w", err)
	}

	return metadata, nil
}

// Download handles file download
func (e *Engine) Download(ctx context.Context, fileID string) (io.ReadCloser, *FileMetadata, error) {
	// Get metadata
	metadata, err := e.getMetadata(ctx, fileID)
	if err != nil {
		return nil, nil, fmt.Errorf("file not found: %w", err)
	}

	// Check if file is expired
	if metadata.ExpiresAt != nil && time.Now().After(*metadata.ExpiresAt) {
		return nil, nil, fmt.Errorf("file has expired")
	}

	// Get file content
	reader, err := e.getFile(ctx, metadata.Path)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to retrieve file: %w", err)
	}

	// Update access statistics
	go e.updateAccessStats(fileID)

	return reader, metadata, nil
}

// Query searches for files
func (e *Engine) Query(ctx context.Context, opts *QueryOptions) ([]*FileMetadata, int, error) {
	if opts == nil {
		opts = &QueryOptions{
			Limit:    50,
			Offset:   0,
			OrderBy:  "created_at",
			OrderDir: "DESC",
		}
	}

	// Build query conditions
	conditions := make(map[string]interface{})
	if opts.TenantID != "" {
		conditions["tenant_id"] = opts.TenantID
	}
	if opts.ProjectID != "" {
		conditions["project_id"] = opts.ProjectID
	}
	if opts.FileType != "" {
		conditions["file_type"] = opts.FileType
	}
	if opts.IsPublic != nil {
		conditions["is_public"] = *opts.IsPublic
	}
	if opts.CreatedBy != "" {
		conditions["created_by"] = opts.CreatedBy
	}
	if opts.Search != "" {
		conditions["search"] = opts.Search
	}

	// Query storage
	queryOptions := &storage.QueryOptions{
		Limit:    opts.Limit,
		Offset:   opts.Offset,
		OrderBy:  opts.OrderBy,
		OrderDir: opts.OrderDir,
		Where:    conditions,
	}

	result, err := e.storage.Query(ctx, "files", queryOptions)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query files: %w", err)
	}

	// Convert to FileMetadata
	files := make([]*FileMetadata, len(result.Records))
	for i, record := range result.Records {
		file := &FileMetadata{}
		if err := e.mapRecordToFile(&record, file); err != nil {
			return nil, 0, fmt.Errorf("failed to map record: %w", err)
		}
		files[i] = file
	}

	return files, result.Total, nil
}

// Delete removes a file
func (e *Engine) Delete(ctx context.Context, fileID string) error {
	// Get metadata
	metadata, err := e.getMetadata(ctx, fileID)
	if err != nil {
		return fmt.Errorf("file not found: %w", err)
	}

	// Delete physical file
	if err := e.deleteFile(ctx, metadata.Path); err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	// Delete metadata record
	if err := e.storage.Delete(ctx, "files", fileID); err != nil {
		return fmt.Errorf("failed to delete file metadata: %w", err)
	}

	return nil
}

// GetMetadata retrieves file metadata without downloading
func (e *Engine) GetMetadata(ctx context.Context, fileID string) (*FileMetadata, error) {
	return e.getMetadata(ctx, fileID)
}

// Helper methods

func (e *Engine) validateFile(header *multipart.FileHeader, opts *UploadOptions) error {
	// Check file size
	if opts != nil && opts.MaxSize > 0 && header.Size > opts.MaxSize {
		return fmt.Errorf("file size %d exceeds maximum allowed %d", header.Size, opts.MaxSize)
	}
	if header.Size > e.config.MaxFileSize {
		return fmt.Errorf("file size %d exceeds maximum allowed %d", header.Size, e.config.MaxFileSize)
	}

	// Check MIME type
	mimeType := header.Header.Get("Content-Type")
	if !e.isAllowedMimeType(mimeType, opts) {
		return fmt.Errorf("MIME type %s is not allowed", mimeType)
	}

	return nil
}

func (e *Engine) isAllowedMimeType(mimeType string, opts *UploadOptions) bool {
	allowedTypes := e.config.AllowedMimes
	if opts != nil && len(opts.AllowedTypes) > 0 {
		allowedTypes = opts.AllowedTypes
	}

	for _, allowed := range allowedTypes {
		if mimeType == allowed {
			return true
		}
	}
	return false
}

func (e *Engine) determineFileType(mimeType, filename string) FileType {
	ext := strings.ToLower(filepath.Ext(filename))

	switch {
	case strings.HasPrefix(mimeType, "image/"):
		return FileTypeImage
	case strings.HasPrefix(mimeType, "video/"):
		return FileTypeVideo
	case strings.HasPrefix(mimeType, "audio/"):
		return FileTypeAudio
	case mimeType == "application/pdf" || strings.Contains(mimeType, "document") ||
	     ext == ".pdf" || ext == ".doc" || ext == ".docx" || ext == ".txt":
		return FileTypeDoc
	case ext == ".zip" || ext == ".tar" || ext == ".gz" || ext == ".rar":
		return FileTypeArchive
	default:
		return FileTypeOther
	}
}

func (e *Engine) generateFileID() string {
	return fmt.Sprintf("file_%d", time.Now().UnixNano())
}

func (e *Engine) generatePath(fileID, filename string, fileType FileType) string {
	year := time.Now().Format("2006")
	month := time.Now().Format("01")

	var dir string
	switch fileType {
	case FileTypeImage:
		dir = "images"
	case FileTypeDoc:
		dir = "documents"
	default:
		dir = "files"
	}

	ext := filepath.Ext(filename)
	return filepath.Join(dir, year, month, fileID+ext)
}

func (e *Engine) storeFile(ctx context.Context, content []byte, path string, fileType FileType) (string, error) {
	switch e.config.StorageType {
	case StorageLocal:
		fullPath := filepath.Join(e.config.LocalPath, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			return "", err
		}
		if err := os.WriteFile(fullPath, content, 0644); err != nil {
			return "", err
		}
		return fullPath, nil
	// TODO: Implement other storage types (S3, MinIO, etc.)
	default:
		return "", fmt.Errorf("storage type %s not implemented", e.config.StorageType)
	}
}

func (e *Engine) getFile(ctx context.Context, path string) (io.ReadCloser, error) {
	switch e.config.StorageType {
	case StorageLocal:
		return os.Open(path)
	default:
		return nil, fmt.Errorf("storage type %s not implemented", e.config.StorageType)
	}
}

func (e *Engine) deleteFile(ctx context.Context, path string) error {
	switch e.config.StorageType {
	case StorageLocal:
		return os.Remove(path)
	default:
		return fmt.Errorf("storage type %s not implemented", e.config.StorageType)
	}
}

func (e *Engine) saveMetadata(ctx context.Context, metadata *FileMetadata) error {
	data := map[string]interface{}{
		"id":            metadata.ID,
		"name":          metadata.Name,
		"original_name": metadata.OriginalName,
		"path":          metadata.Path,
		"size":          metadata.Size,
		"mime_type":     metadata.MimeType,
		"file_type":     metadata.FileType,
		"md5_hash":      metadata.MD5Hash,
		"sha256_hash":   metadata.SHA256Hash,
		"storage_type":  metadata.StorageType,
		"tenant_id":     metadata.TenantID,
		"project_id":    metadata.ProjectID,
		"created_by":    metadata.CreatedBy,
		"created_at":    metadata.CreatedAt,
		"updated_at":    metadata.UpdatedAt,
		"access_count":  metadata.AccessCount,
		"last_accessed": metadata.LastAccessed,
		"tags":          metadata.Tags,
		"metadata":      metadata.Metadata,
		"is_public":     metadata.IsPublic,
		"expires_at":    metadata.ExpiresAt,
	}

	_, err := e.storage.Create(ctx, "files", data)
	return err
}

func (e *Engine) getMetadata(ctx context.Context, fileID string) (*FileMetadata, error) {
	record, err := e.storage.Get(ctx, "files", fileID)
	if err != nil {
		return nil, err
	}

	metadata := &FileMetadata{}
	if err := e.mapRecordToFile(record, metadata); err != nil {
		return nil, err
	}

	return metadata, nil
}

func (e *Engine) mapRecordToFile(record *storage.Record, file *FileMetadata) error {
	// Map storage record to FileMetadata struct
	// This is a simplified mapping - you'd implement proper JSON marshaling
	file.ID = record.ID
	if name, ok := record.Data["name"].(string); ok {
		file.Name = name
	}
	if originalName, ok := record.Data["original_name"].(string); ok {
		file.OriginalName = originalName
	}
	// ... map other fields

	return nil
}

func (e *Engine) findBySHA256(ctx context.Context, hash string) (*FileMetadata, error) {
	options := &storage.QueryOptions{
		Limit: 1,
		Where: map[string]interface{}{
			"sha256_hash": hash,
		},
	}

	result, err := e.storage.Query(ctx, "files", options)
	if err != nil || len(result.Records) == 0 {
		return nil, fmt.Errorf("file not found")
	}

	file := &FileMetadata{}
	if err := e.mapRecordToFile(&result.Records[0], file); err != nil {
		return nil, err
	}

	return file, nil
}

func (e *Engine) updateAccessStats(fileID string) {
	ctx := context.Background()
	metadata, err := e.getMetadata(ctx, fileID)
	if err != nil {
		return
	}

	metadata.AccessCount++
	now := time.Now()
	metadata.LastAccessed = &now

	// Update in storage
	e.saveMetadata(ctx, metadata)
}

func (e *Engine) cleanupExpiredFiles() {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			e.performCleanup()
		case <-e.ctx.Done():
			return
		}
	}
}

func (e *Engine) performCleanup() {
	ctx := context.Background()
	options := &storage.QueryOptions{
		Limit: 1000,
		Where: map[string]interface{}{
			"expires_at": map[string]interface{}{
				"$lt": time.Now(),
			},
		},
	}

	result, err := e.storage.Query(ctx, "files", options)
	if err != nil {
		return
	}

	for _, record := range result.Records {
		fileID := record.ID
		e.Delete(ctx, fileID)
	}
}

func (e *Engine) Close() error {
	if e.cancel != nil {
		e.cancel()
	}
	return nil
}