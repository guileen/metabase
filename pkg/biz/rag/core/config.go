package rag

import ("encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time")

// Config represents the main RAG system configuration
type Config struct {
	// System configuration
	System    SystemConfig    `json:"system"`

	// Data source configuration
	DataSources map[string]interface{} `json:"data_sources"`

	// Processing configuration
	Processing ProcessingConfig `json:"processing"`

	// Retrieval configuration
	Retrieval RetrievalConfig `json:"retrieval"`

	// Generation configuration
	Generation GenerationConfig `json:"generation"`

	// Storage configuration
	Storage   StorageConfig   `json:"storage"`

	// Cache configuration
	Cache     CacheConfig     `json:"cache"`

	// Metrics configuration
	Metrics   MetricsConfig   `json:"metrics"`

	// Security configuration
	Security  SecurityConfig  `json:"security"`
}

// SystemConfig represents system-level configuration
type SystemConfig struct {
	// Basic settings
	Name             string        `json:"name"`
	Version          string        `json:"version"`
	Environment      string        `json:"environment"`        // development, production, etc.
	Debug            bool          `json:"debug"`

	// Performance settings
	MaxWorkers       int           `json:"max_workers"`        // Maximum parallel workers
	MaxConcurrency   int           `json:"max_concurrency"`    // Maximum concurrent operations
	RequestTimeout   time.Duration `json:"request_timeout"`    // Default request timeout
	ShutdownTimeout  time.Duration `json:"shutdown_timeout"`   // Graceful shutdown timeout

	// Resource limits
	MaxMemoryMB      int64         `json:"max_memory_mb"`      // Maximum memory usage in MB
	MaxFileSizeMB    int64         `json:"max_file_size_mb"`   // Maximum file size to process in MB

	// Logging
	LogLevel         string        `json:"log_level"`          // debug, info, warn, error
	LogFormat        string        `json:"log_format"`         // json, text
	LogFile          string        `json:"log_file,omitempty"`
}

// ProcessingConfig represents document processing configuration
type ProcessingConfig struct {
	// Chunking configuration
	Chunking ChunkingConfig `json:"chunking"`

	// Embedding configuration
	Embedding EmbeddingConfig `json:"embedding"`

	// Preprocessing configuration
	Preprocessing PreprocessingConfig `json:"preprocessing"`

	// Indexing configuration
	Indexing IndexingConfig `json:"indexing"`

	// Batch processing
	BatchSize       int           `json:"batch_size"`         // Documents per batch
	BatchTimeout    time.Duration `json:"batch_timeout"`      // Timeout per batch
	MaxRetries      int           `json:"max_retries"`        // Maximum retry attempts
	RetryDelay      time.Duration `json:"retry_delay"`        // Delay between retries
}

// ChunkingConfig represents chunking strategy configuration
type ChunkingConfig struct {
	// Strategy selection
	Strategy        string            `json:"strategy"`          // semantic, fixed, paragraph, etc.

	// Size configuration
	MaxChunkSize    int               `json:"max_chunk_size"`    // Maximum chunk size in characters
	MinChunkSize    int               `json:"min_chunk_size"`    // Minimum chunk size in characters
	OverlapSize     int               `json:"overlap_size"`      // Overlap between chunks in characters

	// Token-based chunking
	MaxTokens       int               `json:"max_tokens"`        // Maximum tokens per chunk
	OverlapTokens   int               `json:"overlap_tokens"`    // Overlap tokens between chunks

	// Semantic chunking
	SimilarityThreshold float64       `json:"similarity_threshold"` // Minimum similarity for semantic chunking
	MinSimilaritySize  int            `json:"min_similarity_size"`  // Minimum size for semantic chunks

	// Language-specific settings
	Languages       map[string]interface{} `json:"languages,omitempty"`

	// Custom parameters
	Custom          map[string]interface{} `json:"custom,omitempty"`
}

// EmbeddingConfig represents embedding generation configuration
type EmbeddingConfig struct {
	// Model configuration
	Model           string            `json:"model"`             // Embedding model name
	Provider        string            `json:"provider"`          // openai, local, etc.

	// Performance settings
	BatchSize       int               `json:"batch_size"`        // Batch size for embedding generation
	MaxConcurrency  int               `json:"max_concurrency"`   // Maximum concurrent embedding requests
	Timeout         time.Duration     `json:"timeout"`           // Request timeout

	// Quality settings
	Normalize       bool              `json:"normalize"`         // Normalize embeddings
	Dimension       int               `json:"dimension,omitempty"` // Target dimension

	// Cache settings
	EnableCache     bool              `json:"enable_cache"`      // Enable embedding cache
	CacheSize       int               `json:"cache_size"`        // Maximum cache entries
	CacheTTL        time.Duration     `json:"cache_ttl"`         // Cache TTL

	// Fallback settings
	EnableFallback  bool              `json:"enable_fallback"`   // Enable fallback models
	FallbackModels  []string          `json:"fallback_models"`   // Fallback model list

	// API configuration (for remote providers)
	APIKey          string            `json:"api_key,omitempty"`
	BaseURL         string            `json:"base_url,omitempty"`
	MaxRetries      int               `json:"max_retries"`
	RetryDelay      time.Duration     `json:"retry_delay"`
}

// PreprocessingConfig represents text preprocessing configuration
type PreprocessingConfig struct {
	// Text cleaning
	EnableCleaning  bool              `json:"enable_cleaning"`   // Enable text cleaning
	RemoveWhitespace bool             `json:"remove_whitespace"` // Remove extra whitespace
	NormalizeUnicode bool             `json:"normalize_unicode"` // Normalize unicode characters
	Lowercase       bool              `json:"lowercase"`         // Convert to lowercase

	// Content filtering
	MinLength       int               `json:"min_length"`        // Minimum content length
	MaxLength       int               `json:"max_length"`        // Maximum content length
	MinWordCount    int               `json:"min_word_count"`    // Minimum word count
	MaxWordCount    int               `json:"max_word_count"`    // Maximum word count

	// Language detection
	DetectLanguage  bool              `json:"detect_language"`   // Auto-detect document language
	DefaultLanguage string            `json:"default_language"`  // Default language if detection fails
	SupportedLanguages []string       `json:"supported_languages"` // Supported languages

	// Metadata extraction
	ExtractMetadata bool              `json:"extract_metadata"`  // Extract document metadata
	ExtractTitle    bool              `json:"extract_title"`     // Extract document title
	ExtractAuthor   bool              `json:"extract_author"`    // Extract author information
	ExtractDate     bool              `json:"extract_date"`      // Extract date information

	// Code-specific processing
	ExtractCodeBlocks bool            `json:"extract_code_blocks"` // Extract code blocks
	PreserveCodeFormatting bool       `json:"preserve_code_formatting"` // Preserve code formatting

	// Custom preprocessing
	CustomFilters   []string          `json:"custom_filters"`    // Custom filter names
	CustomRules     map[string]interface{} `json:"custom_rules,omitempty"`
}

// IndexingConfig represents document indexing configuration
type IndexingConfig struct {
	// Index type and strategy
	IndexType       string            `json:"index_type"`        // vector, hybrid, etc.
	IndexStrategy   string            `json:"index_strategy"`    // immediate, batch, etc.

	// Update settings
	AutoUpdate      bool              `json:"auto_update"`       // Enable automatic updates
	UpdateInterval  time.Duration     `json:"update_interval"`   // Update check interval
	Incremental     bool              `json:"incremental"`       // Use incremental indexing

	// Synchronization settings
	SyncOnStart     bool              `json:"sync_on_start"`     // Sync data sources on startup
	SyncInterval    time.Duration     `json:"sync_interval"`     // Regular sync interval

	// Reindexing settings
	ForceReindex    bool              `json:"force_reindex"`     // Force full reindexing
	ReindexInterval time.Duration     `json:"reindex_interval"`  // Regular reindexing interval

	// Index optimization
	OptimizeIndex   bool              `json:"optimize_index"`    // Enable index optimization
	OptimizeInterval time.Duration    `json:"optimize_interval"` // Optimization interval

	// Performance settings
	MaxIndexSizeMB  int64             `json:"max_index_size_mb"` // Maximum index size
	Compression     bool              `json:"compression"`       // Enable index compression

	// Backup settings
	EnableBackup    bool              `json:"enable_backup"`     // Enable index backups
	BackupInterval  time.Duration     `json:"backup_interval"`   // Backup interval
	BackupRetention int               `json:"backup_retention"`  // Number of backups to keep
}

// RetrievalConfig represents retrieval configuration
type RetrievalConfig struct {
	// Search configuration
	DefaultTopK     int               `json:"default_top_k"`     // Default number of results
	MaxTopK         int               `json:"max_top_k"`         // Maximum number of results
	MinScore        float64           `json:"min_score"`         // Minimum relevance score

	// Search methods
	EnableVectorSearch bool           `json:"enable_vector_search"`  // Enable vector similarity
	EnableKeywordSearch bool          `json:"enable_keyword_search"` // Enable keyword search
	EnableHybridSearch bool           `json:"enable_hybrid_search"`  // Enable hybrid search

	// Hybrid search configuration
	HybridWeight    float64           `json:"hybrid_weight"`     // Weight for vector search (0-1)
	KeywordWeight   float64           `json:"keyword_weight"`    // Weight for keyword search (0-1)
	FusionMethod    string            `json:"fusion_method"`     // Score fusion method

	// Reranking configuration
	EnableRerank    bool              `json:"enable_rerank"`     // Enable result reranking
	RerankModel     string            `json:"rerank_model"`      // Reranking model
	RerankTopK      int               `json:"rerank_top_k"`      // Number of results to rerank
	RerankThreshold float64           `json:"rerank_threshold"`  // Minimum score for reranking

	// Filtering configuration
	EnableFilters   bool              `json:"enable_filters"`    // Enable result filtering
	DefaultFilters  []string          `json:"default_filters"`   // Default filters to apply

	// Performance settings
	MaxQueryTime    time.Duration     `json:"max_query_time"`    // Maximum query time
	EnableCache     bool              `json:"enable_cache"`      // Enable query cache
	CacheSize       int               `json:"cache_size"`        // Maximum cache entries
	CacheTTL        time.Duration     `json:"cache_ttl"`         // Cache TTL

	// Advanced settings
	Diversity       bool              `json:"diversity"`         // Enable result diversity
	DiversityThreshold float64        `json:"diversity_threshold"` // Diversity threshold
	MaxDiversityResults int           `json:"max_diversity_results"` // Max diverse results
}

// GenerationConfig represents generation configuration
type GenerationConfig struct {
	// Model configuration
	Model           string            `json:"model"`             // LLM model name
	Provider        string            `json:"provider"`          // openai, local, etc.

	// Generation parameters
	Temperature     float64           `json:"temperature"`       // Sampling temperature
	MaxTokens       int               `json:"max_tokens"`        // Maximum tokens in response
	TopP            float64           `json:"top_p"`             // Nucleus sampling parameter
	FrequencyPenalty float64          `json:"frequency_penalty"` // Frequency penalty
	PresencePenalty float64           `json:"presence_penalty"`  // Presence penalty

	// Prompt configuration
	SystemPrompt    string            `json:"system_prompt"`     // System prompt template
	UserPromptTemplate string         `json:"user_prompt_template"` // User prompt template
	MaxContextLength int              `json:"max_context_length"` // Maximum context length

	// Response formatting
	Format          string            `json:"format"`            // Response format (markdown, json, etc.)
	EnableCitations bool              `json:"enable_citations"`  // Include source citations
	CitationFormat  string            `json:"citation_format"`   // Citation format style

	// Quality settings
	MinConfidence   float64           `json:"min_confidence"`    // Minimum confidence threshold
	EnableFactCheck bool              `json:"enable_fact_check"` // Enable fact checking
	QualityThreshold float64          `json:"quality_threshold"` // Quality threshold

	// Performance settings
	Streaming       bool              `json:"streaming"`         // Enable streaming responses
	Timeout         time.Duration     `json:"timeout"`           // Generation timeout
	MaxRetries      int               `json:"max_retries"`        // Maximum retry attempts
	RetryDelay      time.Duration     `json:"retry_delay"`        // Delay between retries

	// API configuration
	APIKey          string            `json:"api_key,omitempty"`
	BaseURL         string            `json:"base_url,omitempty"`
}

// StorageConfig represents storage configuration
type StorageConfig struct {
	// Backend selection
	Backend         string            `json:"backend"`           // sqlite, postgres, vector_db, etc.

	// Connection settings
	ConnectionString string           `json:"connection_string,omitempty"`
	Host            string            `json:"host,omitempty"`
	Port            int               `json:"port,omitempty"`
	Database        string            `json:"database,omitempty"`
	Username        string            `json:"username,omitempty"`
	Password        string            `json:"password,omitempty"`

	// File-based storage
	DataDirectory   string            `json:"data_directory"`    // Data directory for file storage
	IndexDirectory  string            `json:"index_directory"`   // Index directory

	// Performance settings
	ConnectionPool  int               `json:"connection_pool"`   // Connection pool size
	MaxConnections  int               `json:"max_connections"`   // Maximum connections
	Timeout         time.Duration     `json:"timeout"`           // Database timeout

	// Index settings
	VectorIndexType string            `json:"vector_index_type"` // hnsw, ivf, etc.
	VectorDimensions int              `json:"vector_dimensions"` // Vector dimensions
	IndexMetric     string            `json:"index_metric"`      // Distance metric (cosine, euclidean, etc.)

	// Backup and recovery
	EnableBackup    bool              `json:"enable_backup"`     // Enable automatic backups
	BackupPath      string            `json:"backup_path"`       // Backup directory
	BackupInterval  time.Duration     `json:"backup_interval"`   // Backup interval
	BackupRetention int               `json:"backup_retention"`  // Number of backups to keep

	// Security
	EnableEncryption bool             `json:"enable_encryption"` // Enable data encryption
	EncryptionKey   string            `json:"encryption_key,omitempty"`

	// Maintenance
	EnableVacuum    bool              `json:"enable_vacuum"`     // Enable vacuum/cleanup
	VacuumInterval  time.Duration     `json:"vacuum_interval"`   // Vacuum interval
}

// CacheConfig represents cache configuration
type CacheConfig struct {
	// Cache type
	Type            string            `json:"type"`              // memory, redis, etc.

	// Memory cache
	MaxSize         int64             `json:"max_size"`          // Maximum cache size in bytes
	MaxEntries      int               `json:"max_entries"`       // Maximum number of entries
	TTL             time.Duration     `json:"ttl"`               // Default TTL

	// Redis cache (if used)
	RedisURL        string            `json:"redis_url,omitempty"`
	RedisPassword   string            `json:"redis_password,omitempty"`
	RedisDB         int               `json:"redis_db"`          // Redis database number

	// Cache policies
	EvictionPolicy  string            `json:"eviction_policy"`   // lru, lfu, fifo, etc.
	EnableCompression bool            `json:"enable_compression"` // Compress cached values

	// Cache strategies
	QueryCache      bool              `json:"query_cache"`       // Enable query result caching
	EmbeddingCache  bool              `json:"embedding_cache"`   // Enable embedding caching
	DocumentCache   bool              `json:"document_cache"`    // Enable document caching

	// Performance settings
	CleanupInterval time.Duration     `json:"cleanup_interval"`  // Cleanup interval
	MaxCleanupTime  time.Duration     `json:"max_cleanup_time"`  // Maximum cleanup time
}

// MetricsConfig represents metrics collection configuration
type MetricsConfig struct {
	// Collection settings
	Enabled         bool              `json:"enabled"`           // Enable metrics collection
	CollectionInterval time.Duration   `json:"collection_interval"` // Collection interval
	RetentionPeriod time.Duration     `json:"retention_period"`  // How long to keep metrics

	// Storage
	StorageType     string            `json:"storage_type"`      // memory, file, database, etc.
	StoragePath     string            `json:"storage_path,omitempty"`

	// Export settings
	ExportFormat    string            `json:"export_format"`     // json, prometheus, etc.
	ExportInterval  time.Duration     `json:"export_interval"`   // Export interval
	ExportPath      string            `json:"export_path,omitempty"`

	// Metrics to collect
	CollectQueryMetrics bool           `json:"collect_query_metrics"`
	CollectPerformanceMetrics bool     `json:"collect_performance_metrics"`
	CollectResourceMetrics bool        `json:"collect_resource_metrics"`
	CollectErrorMetrics bool           `json:"collect_error_metrics"`

	// Sampling
	SampleRate      float64           `json:"sample_rate"`       // Sampling rate (0-1)
	MaxEventsPerSecond int            `json:"max_events_per_second"`

	// Alerts
	EnableAlerts    bool              `json:"enable_alerts"`     // Enable alerting
	AlertThresholds map[string]float64 `json:"alert_thresholds"` // Alert thresholds
}

// SecurityConfig represents security configuration
type SecurityConfig struct {
	// Authentication
	Enabled         bool              `json:"enabled"`           // Enable security features
	AuthType        string            `json:"auth_type"`         // jwt, api_key, etc.

	// API key configuration
	APIKeys         []string          `json:"api_keys,omitempty"`
	APIKeyHeader    string            `json:"api_key_header"`    // API key header name

	// JWT configuration
	JWTSecret       string            `json:"jwt_secret,omitempty"`
	JWTExpiration   time.Duration     `json:"jwt_expiration"`    // JWT token expiration
	JWTRefreshExpiration time.Duration `json:"jwt_refresh_expiration"` // Refresh token expiration

	// Access control
	EnableRBAC      bool              `json:"enable_rbac"`       // Enable role-based access control
	DefaultRole     string            `json:"default_role"`      // Default user role

	// Rate limiting
	EnableRateLimit bool              `json:"enable_rate_limit"` // Enable rate limiting
	MaxRequestsPerMinute int          `json:"max_requests_per_minute"`
	BurstSize       int               `json:"burst_size"`        // Burst size

	// Input validation
	MaxQueryLength  int               `json:"max_query_length"`  // Maximum query length
	AllowedFileTypes []string         `json:"allowed_file_types"` // Allowed file types

	// Data privacy
	EnablePII       bool              `json:"enable_pii"`        // Enable PII detection
	PIIAction       string            `json:"pii_action"`        // Action for PII detection (mask, remove, etc.)

	// Audit logging
	EnableAudit     bool              `json:"enable_audit"`      // Enable audit logging
	AuditLogPath    string            `json:"audit_log_path,omitempty"`
}

// DefaultConfig returns a default RAG configuration
func DefaultConfig() *Config {
	homeDir, _ := os.UserHomeDir()
	dataDir := os.Getenv("RAG_DATA_DIR")
	if dataDir == "" {
		dataDir = filepath.Join(homeDir, ".metabase", "rag")
	}

	return &Config{
		System: SystemConfig{
			Name:            "metabase-rag",
			Version:         "1.0.0",
			Environment:     "development",
			Debug:           true,
			MaxWorkers:      4,
			MaxConcurrency:  10,
			RequestTimeout:  30 * time.Second,
			ShutdownTimeout: 10 * time.Second,
			MaxMemoryMB:     1024,
			MaxFileSizeMB:   100,
			LogLevel:        "info",
			LogFormat:       "json",
		},
		DataSources: make(map[string]interface{}),
		Processing: ProcessingConfig{
			Chunking: ChunkingConfig{
				Strategy:           "semantic",
				MaxChunkSize:       1000,
				MinChunkSize:       100,
				OverlapSize:        200,
				MaxTokens:          300,
				OverlapTokens:      50,
				SimilarityThreshold: 0.7,
				MinSimilaritySize:  200,
			},
			Embedding: EmbeddingConfig{
				Model:          "text-embedding-3-small",
				Provider:       "openai",
				BatchSize:      32,
				MaxConcurrency: 4,
				Timeout:        60 * time.Second,
				Normalize:      true,
				EnableCache:    true,
				CacheSize:      10000,
				CacheTTL:       24 * time.Hour,
				EnableFallback: true,
				MaxRetries:     3,
				RetryDelay:     time.Second,
			},
			Preprocessing: PreprocessingConfig{
				EnableCleaning:       true,
				RemoveWhitespace:     true,
				NormalizeUnicode:     true,
				MinLength:           10,
				MaxLength:           100000,
				MinWordCount:        2,
				MaxWordCount:        20000,
				DetectLanguage:      true,
				DefaultLanguage:     "en",
				SupportedLanguages:  []string{"en", "zh", "es", "fr", "de", "ja"},
				ExtractMetadata:     true,
				ExtractTitle:        true,
				ExtractAuthor:       true,
				ExtractDate:         true,
				ExtractCodeBlocks:   true,
				PreserveCodeFormatting: true,
			},
			Indexing: IndexingConfig{
				IndexType:        "hybrid",
				IndexStrategy:    "batch",
				AutoUpdate:       true,
				UpdateInterval:   5 * time.Minute,
				Incremental:      true,
				SyncOnStart:      true,
				SyncInterval:     time.Hour,
				ForceReindex:     false,
				ReindexInterval:  24 * time.Hour,
				OptimizeIndex:    true,
				OptimizeInterval: 6 * time.Hour,
				MaxIndexSizeMB:   1024,
				Compression:      true,
				EnableBackup:     true,
				BackupInterval:   12 * time.Hour,
				BackupRetention:  7,
			},
			BatchSize:    10,
			BatchTimeout: 5 * time.Minute,
			MaxRetries:   3,
			RetryDelay:   time.Second,
		},
		Retrieval: RetrievalConfig{
			DefaultTopK:          10,
			MaxTopK:              100,
			MinScore:             0.5,
			EnableVectorSearch:   true,
			EnableKeywordSearch:  true,
			EnableHybridSearch:   true,
			HybridWeight:         0.7,
			KeywordWeight:        0.3,
			FusionMethod:         "weighted",
			EnableRerank:         true,
			RerankModel:          "BAAI/bge-reranker-v2-m3",
			RerankTopK:           20,
			RerankThreshold:      0.6,
			EnableFilters:        true,
			MaxQueryTime:         30 * time.Second,
			EnableCache:          true,
			CacheSize:            1000,
			CacheTTL:             time.Hour,
			Diversity:            true,
			DiversityThreshold:   0.8,
			MaxDiversityResults:  20,
		},
		Generation: GenerationConfig{
			Model:              "gpt-3.5-turbo",
			Provider:           "openai",
			Temperature:        0.7,
			MaxTokens:          1000,
			TopP:               0.9,
			FrequencyPenalty:   0.0,
			PresencePenalty:    0.0,
			SystemPrompt:       "You are a helpful AI assistant that answers questions based on the provided context.",
			UserPromptTemplate: "Context: {{.Context}}\n\nQuestion: {{.Query}}\n\nAnswer:",
			MaxContextLength:   8000,
			Format:             "markdown",
			EnableCitations:    true,
			CitationFormat:     "numeric",
			MinConfidence:      0.5,
			EnableFactCheck:    false,
			QualityThreshold:   0.6,
			Streaming:          false,
			Timeout:            60 * time.Second,
			MaxRetries:         3,
			RetryDelay:         time.Second,
		},
		Storage: StorageConfig{
			Backend:           "sqlite",
			DataDirectory:     dataDir,
			IndexDirectory:    filepath.Join(dataDir, "index"),
			ConnectionPool:    10,
			MaxConnections:    50,
			Timeout:           30 * time.Second,
			VectorIndexType:   "hnsw",
			VectorDimensions:  1536,
			IndexMetric:       "cosine",
			EnableBackup:      true,
			BackupPath:        filepath.Join(dataDir, "backups"),
			BackupInterval:    12 * time.Hour,
			BackupRetention:   7,
			EnableEncryption:  false,
			EnableVacuum:      true,
			VacuumInterval:    24 * time.Hour,
		},
		Cache: CacheConfig{
			Type:               "memory",
			MaxSize:            100 * 1024 * 1024, // 100MB
			MaxEntries:         10000,
			TTL:                time.Hour,
			EvictionPolicy:     "lru",
			EnableCompression:  true,
			QueryCache:         true,
			EmbeddingCache:     true,
			DocumentCache:      false,
			CleanupInterval:    time.Hour,
			MaxCleanupTime:     5 * time.Minute,
		},
		Metrics: MetricsConfig{
			Enabled:               true,
			CollectionInterval:    time.Minute,
			RetentionPeriod:       7 * 24 * time.Hour, // 7 days
			StorageType:           "file",
			StoragePath:           filepath.Join(dataDir, "metrics"),
			ExportFormat:          "json",
			ExportInterval:        time.Hour,
			CollectQueryMetrics:   true,
			CollectPerformanceMetrics: true,
			CollectResourceMetrics: true,
			CollectErrorMetrics:   true,
			SampleRate:            1.0,
			EnableAlerts:          false,
		},
		Security: SecurityConfig{
			Enabled:               false,
			AuthType:              "none",
			EnableRateLimit:       false,
			MaxRequestsPerMinute:  60,
			BurstSize:             10,
			MaxQueryLength:        1000,
			AllowedFileTypes:      []string{".txt", ".md", ".pdf", ".doc", ".docx"},
			EnablePII:             false,
			PIIAction:             "mask",
			EnableAudit:           false,
		},
	}
}

// LoadConfig loads configuration from file
func LoadConfig(configPath string) (*Config, error) {
	if configPath == "" {
		return DefaultConfig(), nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultConfig(), nil
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

// SaveConfig saves configuration to file
func SaveConfig(config *Config, configPath string) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Create directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// Validate validates the configuration
func (config *Config) Validate() error {
	// Validate system config
	if config.System.Name == "" {
		return fmt.Errorf("system name is required")
	}
	if config.System.MaxWorkers <= 0 {
		return fmt.Errorf("max_workers must be positive")
	}
	if config.System.MaxConcurrency <= 0 {
		return fmt.Errorf("max_concurrency must be positive")
	}

	// Validate processing config
	if config.Processing.Chunking.MaxChunkSize <= 0 {
		return fmt.Errorf("max_chunk_size must be positive")
	}
	if config.Processing.Chunking.MinChunkSize <= 0 {
		return fmt.Errorf("min_chunk_size must be positive")
	}
	if config.Processing.Chunking.MinChunkSize > config.Processing.Chunking.MaxChunkSize {
		return fmt.Errorf("min_chunk_size cannot be greater than max_chunk_size")
	}

	// Validate retrieval config
	if config.Retrieval.DefaultTopK <= 0 {
		return fmt.Errorf("default_top_k must be positive")
	}
	if config.Retrieval.MaxTopK <= 0 {
		return fmt.Errorf("max_top_k must be positive")
	}
	if config.Retrieval.DefaultTopK > config.Retrieval.MaxTopK {
		return fmt.Errorf("default_top_k cannot be greater than max_top_k")
	}

	// Validate generation config
	if config.Generation.Model == "" {
		return fmt.Errorf("generation model is required")
	}
	if config.Generation.MaxTokens <= 0 {
		return fmt.Errorf("max_tokens must be positive")
	}

	// Validate storage config
	if config.Storage.Backend == "" {
		return fmt.Errorf("storage backend is required")
	}

	return nil
}

// Merge merges another configuration into this one
func (config *Config) Merge(other *Config) {
	if other == nil {
		return
	}

	// Merge system config
	if other.System.Name != "" {
		config.System.Name = other.System.Name
	}
	if other.System.Version != "" {
		config.System.Version = other.System.Version
	}
	if other.System.Environment != "" {
		config.System.Environment = other.System.Environment
	}
	if other.System.Debug {
		config.System.Debug = true
	}
	if other.System.MaxWorkers > 0 {
		config.System.MaxWorkers = other.System.MaxWorkers
	}
	if other.System.MaxConcurrency > 0 {
		config.System.MaxConcurrency = other.System.MaxConcurrency
	}

	// Merge processing config
	mergeProcessingConfig(&config.Processing, &other.Processing)

	// Merge retrieval config
	mergeRetrievalConfig(&config.Retrieval, &other.Retrieval)

	// Merge generation config
	mergeGenerationConfig(&config.Generation, &other.Generation)

	// Merge storage config
	mergeStorageConfig(&config.Storage, &other.Storage)

	// Merge cache config
	mergeCacheConfig(&config.Cache, &other.Cache)

	// Merge metrics config
	mergeMetricsConfig(&config.Metrics, &other.Metrics)

	// Merge security config
	mergeSecurityConfig(&config.Security, &other.Security)

	// Merge data sources
	for key, value := range other.DataSources {
		config.DataSources[key] = value
	}
}

// Helper functions for merging configuration sections
func mergeProcessingConfig(dest *ProcessingConfig, src *ProcessingConfig) {
	if src.Chunking.Strategy != "" {
		dest.Chunking.Strategy = src.Chunking.Strategy
	}
	if src.Chunking.MaxChunkSize > 0 {
		dest.Chunking.MaxChunkSize = src.Chunking.MaxChunkSize
	}
	if src.Chunking.MinChunkSize > 0 {
		dest.Chunking.MinChunkSize = src.Chunking.MinChunkSize
	}
	if src.Embedding.Model != "" {
		dest.Embedding.Model = src.Embedding.Model
	}
	if src.Embedding.Provider != "" {
		dest.Embedding.Provider = src.Embedding.Provider
	}
}

func mergeRetrievalConfig(dest *RetrievalConfig, src *RetrievalConfig) {
	if src.DefaultTopK > 0 {
		dest.DefaultTopK = src.DefaultTopK
	}
	if src.MaxTopK > 0 {
		dest.MaxTopK = src.MaxTopK
	}
	if src.MinScore > 0 {
		dest.MinScore = src.MinScore
	}
}

func mergeGenerationConfig(dest *GenerationConfig, src *GenerationConfig) {
	if src.Model != "" {
		dest.Model = src.Model
	}
	if src.Provider != "" {
		dest.Provider = src.Provider
	}
	if src.Temperature >= 0 {
		dest.Temperature = src.Temperature
	}
	if src.MaxTokens > 0 {
		dest.MaxTokens = src.MaxTokens
	}
}

func mergeStorageConfig(dest *StorageConfig, src *StorageConfig) {
	if src.Backend != "" {
		dest.Backend = src.Backend
	}
	if src.ConnectionString != "" {
		dest.ConnectionString = src.ConnectionString
	}
	if src.DataDirectory != "" {
		dest.DataDirectory = src.DataDirectory
	}
}

func mergeCacheConfig(dest *CacheConfig, src *CacheConfig) {
	if src.Type != "" {
		dest.Type = src.Type
	}
	if src.MaxSize > 0 {
		dest.MaxSize = src.MaxSize
	}
	if src.MaxEntries > 0 {
		dest.MaxEntries = src.MaxEntries
	}
}

func mergeMetricsConfig(dest *MetricsConfig, src *MetricsConfig) {
	if src.Enabled {
		dest.Enabled = true
	}
	if src.StorageType != "" {
		dest.StorageType = src.StorageType
	}
	if src.StoragePath != "" {
		dest.StoragePath = src.StoragePath
	}
}

func mergeSecurityConfig(dest *SecurityConfig, src *SecurityConfig) {
	if src.Enabled {
		dest.Enabled = true
	}
	if src.AuthType != "" {
		dest.AuthType = src.AuthType
	}
}