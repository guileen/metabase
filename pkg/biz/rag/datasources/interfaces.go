package datasources

import (
	"fmt"
	"time"

	"github.com/guileen/metabase/pkg/biz/rag"
)

// DataSourceFactory creates data source instances from configuration
type DataSourceFactory interface {
	// CreateDataSource creates a data source from configuration
	CreateDataSource(config map[string]interface{}) (rag.DataSource, error)

	// GetSupportedTypes returns the list of supported data source types
	GetSupportedTypes() []string

	// ValidateConfig validates data source configuration
	ValidateConfig(config map[string]interface{}) error
}

// BaseDataSource provides common functionality for data sources
type BaseDataSource struct {
	ID       string                 `json:"id"`
	Type     string                 `json:"type"`
	Config   map[string]interface{} `json:"config"`
	Metadata map[string]interface{} `json:"metadata"`
}

// FileSystemConfig represents configuration for file system data sources
type FileSystemConfig struct {
	// Basic configuration
	RootPath  string `json:"root_path"` // Root directory path
	Recursive bool   `json:"recursive"` // Recursively scan subdirectories

	// File filtering
	IncludePatterns []string `json:"include_patterns"` // File patterns to include
	ExcludePatterns []string `json:"exclude_patterns"` // File patterns to exclude
	MaxFileSize     int64    `json:"max_file_size"`    // Maximum file size in bytes
	MinFileSize     int64    `json:"min_file_size"`    // Minimum file size in bytes

	// File type filtering
	IncludeTypes []string `json:"include_types"` // File extensions to include
	ExcludeTypes []string `json:"exclude_types"` // File extensions to exclude

	// Content filtering
	MinLength int `json:"min_length"` // Minimum content length
	MaxLength int `json:"max_length"` // Maximum content length

	// Sync options
	FollowSymlinks bool `json:"follow_symlinks"` // Follow symbolic links
	IgnoreHidden   bool `json:"ignore_hidden"`   // Ignore hidden files and directories

	// Processing options
	ExtractMetadata bool `json:"extract_metadata"` // Extract file metadata
	DetectLanguage  bool `json:"detect_language"`  // Auto-detect file language
	PreserveCode    bool `json:"preserve_code"`    // Preserve code formatting

	// Performance options
	MaxWorkers int `json:"max_workers"` // Maximum parallel workers
	BatchSize  int `json:"batch_size"`  // Files to process in batch

	// Monitoring
	EnableWatch   bool          `json:"enable_watch"`   // Enable file watching
	WatchInterval time.Duration `json:"watch_interval"` // File check interval

	// Cache settings
	EnableCache bool          `json:"enable_cache"` // Enable file content cache
	CacheDir    string        `json:"cache_dir"`    // Cache directory
	CacheTTL    time.Duration `json:"cache_ttl"`    // Cache TTL
}

// DatabaseConfig represents configuration for database data sources
type DatabaseConfig struct {
	// Connection settings
	Driver           string `json:"driver"`            // Database driver (mysql, postgres, sqlite, etc.)
	ConnectionString string `json:"connection_string"` // Connection string or DSN
	Host             string `json:"host"`              // Database host
	Port             int    `json:"port"`              // Database port
	Database         string `json:"database"`          // Database name
	Username         string `json:"username"`          // Database username
	Password         string `json:"password"`          // Database password

	// Connection pool settings
	MaxConnections  int           `json:"max_connections"`   // Maximum connections
	MaxIdle         int           `json:"max_idle"`          // Maximum idle connections
	ConnMaxLifetime time.Duration `json:"conn_max_lifetime"` // Connection max lifetime

	// Table configuration
	Tables          []TableConfig `json:"tables"`           // Tables to index
	PrimaryKey      string        `json:"primary_key"`      // Primary key column
	TextColumns     []string      `json:"text_columns"`     // Text columns to index
	MetadataColumns []string      `json:"metadata_columns"` // Columns to treat as metadata

	// Query configuration
	CustomQuery  string        `json:"custom_query"`  // Custom SQL query
	QueryTimeout time.Duration `json:"query_timeout"` // Query timeout
	BatchSize    int           `json:"batch_size"`    // Records per batch

	// Sync options
	SyncInterval    time.Duration `json:"sync_interval"`    // Sync interval
	TrackChanges    bool          `json:"track_changes"`    // Track database changes
	TimestampColumn string        `json:"timestamp_column"` // Column for change tracking
}

// TableConfig represents configuration for a database table
type TableConfig struct {
	Name        string `json:"name"`         // Table name
	Alias       string `json:"alias"`        // Table alias
	WhereClause string `json:"where_clause"` // WHERE clause for filtering
	OrderBy     string `json:"order_by"`     // ORDER BY clause
	Limit       int    `json:"limit"`        // LIMIT clause

	// Column mapping
	IDColumn      string `json:"id_column"`      // Document ID column
	TitleColumn   string `json:"title_column"`   // Document title column
	ContentColumn string `json:"content_column"` // Document content column
	URIColumn     string `json:"uri_column"`     // Document URI column

	// Additional columns
	TextColumns     []string `json:"text_columns"`     // Additional text columns
	MetadataColumns []string `json:"metadata_columns"` // Metadata columns

	// Processing options
	JoinColumns map[string]string `json:"join_columns"` // JOIN clauses
	ConvertJSON bool              `json:"convert_json"` // Convert JSON columns to metadata
}

// WebConfig represents configuration for web crawler data sources
type WebConfig struct {
	// Seed URLs
	SeedURLs       []string `json:"seed_urls"`       // Starting URLs
	Domains        []string `json:"domains"`         // Allowed domains (whitelist)
	BlockedDomains []string `json:"blocked_domains"` // Blocked domains (blacklist)

	// Crawling behavior
	MaxDepth     int  `json:"max_depth"`      // Maximum crawl depth
	MaxPages     int  `json:"max_pages"`      // Maximum pages to crawl
	FollowLinks  bool `json:"follow_links"`   // Follow links on pages
	StayInDomain bool `json:"stay_in_domain"` // Stay within starting domains

	// Request settings
	UserAgent      string        `json:"user_agent"`      // User agent string
	RequestTimeout time.Duration `json:"request_timeout"` // Request timeout
	MaxRetries     int           `json:"max_retries"`     // Maximum retry attempts
	RetryDelay     time.Duration `json:"retry_delay"`     // Delay between retries

	// Rate limiting
	RequestsPerSecond    float64       `json:"requests_per_second"`    // Rate limit
	DelayBetweenRequests time.Duration `json:"delay_between_requests"` // Delay between requests

	// Content filtering
	ContentTypes    []string `json:"content_types"`    // Allowed content types
	ExcludePatterns []string `json:"exclude_patterns"` // URL patterns to exclude
	MinLength       int      `json:"min_length"`       // Minimum content length
	MaxLength       int      `json:"max_length"`       // Maximum content length

	// Processing options
	ExtractTitle      bool `json:"extract_title"`      // Extract page title
	ExtractMetadata   bool `json:"extract_metadata"`   // Extract page metadata
	ExtractLinks      bool `json:"extract_links"`      // Extract links
	RemoveHTML        bool `json:"remove_html"`        // Remove HTML tags
	PreserveStructure bool `json:"preserve_structure"` // Preserve HTML structure

	// Authentication
	AuthType    string `json:"auth_type"`    // Authentication type (basic, oauth, etc.)
	Username    string `json:"username"`     // Username for basic auth
	Password    string `json:"password"`     // Password for basic auth
	APIKey      string `json:"api_key"`      // API key for authentication
	BearerToken string `json:"bearer_token"` // Bearer token

	// Headers
	CustomHeaders map[string]string `json:"custom_headers"` // Custom HTTP headers

	// Performance options
	MaxWorkers     int `json:"max_workers"`     // Maximum parallel crawlers
	ConnectionPool int `json:"connection_pool"` // Connection pool size

	// Caching
	EnableCache bool          `json:"enable_cache"` // Enable page caching
	CacheDir    string        `json:"cache_dir"`    // Cache directory
	CacheTTL    time.Duration `json:"cache_ttl"`    // Cache TTL

	// Monitoring
	EnableLogging bool   `json:"enable_logging"` // Enable crawl logging
	LogFile       string `json:"log_file"`       // Log file path
	LogLevel      string `json:"log_level"`      // Log level
}

// APIConfig represents configuration for API data sources
type APIConfig struct {
	// API endpoint
	BaseURL  string `json:"base_url"` // Base URL for API
	Endpoint string `json:"endpoint"` // API endpoint path
	Method   string `json:"method"`   // HTTP method (GET, POST, etc.)

	// Authentication
	AuthType    string `json:"auth_type"`    // Authentication type
	APIKey      string `json:"api_key"`      // API key
	BearerToken string `json:"bearer_token"` // Bearer token
	Username    string `json:"username"`     // Username for basic auth
	Password    string `json:"password"`     // Password for basic auth

	// Headers
	Headers map[string]string `json:"headers"` // Custom headers

	// Request parameters
	Params map[string]interface{} `json:"params"` // Query parameters
	Body   interface{}            `json:"body"`   // Request body for POST requests

	// Pagination
	PaginationType string `json:"pagination_type"` // Pagination type (offset, cursor, page)
	PageParam      string `json:"page_param"`      // Page parameter name
	LimitParam     string `json:"limit_param"`     // Limit parameter name
	MaxPages       int    `json:"max_pages"`       // Maximum pages to fetch
	PageSize       int    `json:"page_size"`       // Page size

	// Response parsing
	DataPath      string            `json:"data_path"`      // JSON path to data array
	IDPath        string            `json:"id_path"`        // JSON path to ID field
	TitlePath     string            `json:"title_path"`     // JSON path to title field
	ContentPath   string            `json:"content_path"`   // JSON path to content field
	MetadataPaths map[string]string `json:"metadata_paths"` // JSON paths to metadata fields

	// Rate limiting
	RequestsPerSecond    float64       `json:"requests_per_second"`    // Rate limit
	DelayBetweenRequests time.Duration `json:"delay_between_requests"` // Delay between requests

	// Retry settings
	MaxRetries       int           `json:"max_retries"`        // Maximum retry attempts
	RetryDelay       time.Duration `json:"retry_delay"`        // Delay between retries
	RetryStatusCodes []int         `json:"retry_status_codes"` // HTTP status codes to retry

	// Timeout settings
	RequestTimeout time.Duration `json:"request_timeout"` // Request timeout
	TotalTimeout   time.Duration `json:"total_timeout"`   // Total operation timeout

	// Processing options
	MinLength   int  `json:"min_length"`   // Minimum content length
	MaxLength   int  `json:"max_length"`   // Maximum content length
	FilterEmpty bool `json:"filter_empty"` // Filter empty responses

	// Performance
	MaxWorkers     int `json:"max_workers"`     // Maximum parallel requests
	ConnectionPool int `json:"connection_pool"` // Connection pool size

	// Validation
	ValidateSchema bool   `json:"validate_schema"` // Validate response schema
	SchemaFile     string `json:"schema_file"`     // JSON schema file path
}

// S3Config represents configuration for AWS S3 data sources
type S3Config struct {
	// AWS credentials and configuration
	Region       string `json:"region"`        // AWS region
	Bucket       string `json:"bucket"`        // S3 bucket name
	AccessKey    string `json:"access_key"`    // AWS access key
	SecretKey    string `json:"secret_key"`    // AWS secret key
	SessionToken string `json:"session_token"` // AWS session token (temporary credentials)

	// Alternative authentication
	Profile    string `json:"profile"`     // AWS profile name
	RoleARN    string `json:"role_arn"`    // AWS IAM role ARN
	ExternalID string `json:"external_id"` // External ID for role assumption

	// Object filtering
	Prefix          string   `json:"prefix"`           // Object prefix filter
	Delimiter       string   `json:"delimiter"`        // Delimiter for hierarchical listing
	IncludePatterns []string `json:"include_patterns"` // Include patterns for object keys
	ExcludePatterns []string `json:"exclude_patterns"` // Exclude patterns for object keys

	// File filtering
	IncludeTypes []string `json:"include_types"` // File extensions to include
	ExcludeTypes []string `json:"exclude_types"` // File extensions to exclude
	MaxFileSize  int64    `json:"max_file_size"` // Maximum file size
	MinFileSize  int64    `json:"min_file_size"` // Minimum file size

	// Content filtering
	MinLength int `json:"min_length"` // Minimum content length
	MaxLength int `json:"max_length"` // Maximum content length

	// Sync options
	Recursive     bool `json:"recursive"`      // Recursively list objects
	TrackVersions bool `json:"track_versions"` // Track object versions

	// Performance options
	MaxWorkers     int           `json:"max_workers"`     // Maximum parallel downloads
	PartSize       int64         `json:"part_size"`       // Multipart upload part size
	RequestTimeout time.Duration `json:"request_timeout"` // Request timeout

	// Caching
	EnableCache bool          `json:"enable_cache"` // Enable object cache
	CacheDir    string        `json:"cache_dir"`    // Cache directory
	CacheTTL    time.Duration `json:"cache_ttl"`    // Cache TTL

	// Security
	EncryptData          bool   `json:"encrypt_data"`           // Enable server-side encryption
	ServerSideEncryption string `json:"server_side_encryption"` // Encryption algorithm

	// Metadata
	UseMetadata    bool `json:"use_metadata"`     // Use S3 object metadata
	TagsAsMetadata bool `json:"tags_as_metadata"` // Use S3 object tags as metadata
}

// GitConfig represents configuration for Git repository data sources
type GitConfig struct {
	// Repository information
	RepositoryURL string `json:"repository_url"` // Git repository URL
	Branch        string `json:"branch"`         // Branch to index (default: main/master)
	Commit        string `json:"commit"`         // Specific commit to index
	Tag           string `json:"tag"`            // Specific tag to index

	// Local repository
	LocalPath    string `json:"local_path"`    // Local path for cloned repository
	CloneDepth   int    `json:"clone_depth"`   // Clone depth (0 for full history)
	ShallowClone bool   `json:"shallow_clone"` // Perform shallow clone

	// Authentication
	Username       string `json:"username"`         // Git username
	Password       string `json:"password"`         // Git password or personal access token
	SSHKey         string `json:"ssh_key"`          // SSH private key path
	SSHKeyPassword string `json:"ssh_key_password"` // SSH private key password

	// File filtering
	IncludePatterns []string `json:"include_patterns"` // File patterns to include
	ExcludePatterns []string `json:"exclude_patterns"` // File patterns to exclude
	IncludeTypes    []string `json:"include_types"`    // File extensions to include
	ExcludeTypes    []string `json:"exclude_types"`    // File extensions to exclude
	MaxFileSize     int64    `json:"max_file_size"`    // Maximum file size

	// Content filtering
	MinLength int `json:"min_length"` // Minimum content length
	MaxLength int `json:"max_length"` // Maximum content length

	// Processing options
	FollowSymlinks     bool `json:"follow_symlinks"`      // Follow symbolic links
	IgnoreGitignore    bool `json:"ignore_gitignore"`     // Ignore .gitignore file
	ExtractGitMetadata bool `json:"extract_git_metadata"` // Extract Git metadata

	// Sync options
	SyncInterval time.Duration `json:"sync_interval"` // Sync interval
	TrackChanges bool          `json:"track_changes"` // Track repository changes
	AutoPull     bool          `json:"auto_pull"`     // Automatically pull changes

	// Performance
	MaxWorkers int `json:"max_workers"` // Maximum parallel workers
	BatchSize  int `json:"batch_size"`  // Files to process in batch

	// Caching
	EnableCache bool          `json:"enable_cache"` // Enable content cache
	CacheDir    string        `json:"cache_dir"`    // Cache directory
	CacheTTL    time.Duration `json:"cache_ttl"`    // Cache TTL
}

// DataSourceRegistry manages data source factories
type DataSourceRegistry struct {
	factories map[string]DataSourceFactory
}

// NewDataSourceRegistry creates a new data source registry
func NewDataSourceRegistry() *DataSourceRegistry {
	return &DataSourceRegistry{
		factories: make(map[string]DataSourceFactory),
	}
}

// RegisterFactory registers a data source factory
func (r *DataSourceRegistry) RegisterFactory(sourceType string, factory DataSourceFactory) {
	r.factories[sourceType] = factory
}

// GetFactory returns a factory for the specified data source type
func (r *DataSourceRegistry) GetFactory(sourceType string) (DataSourceFactory, error) {
	factory, exists := r.factories[sourceType]
	if !exists {
		return nil, fmt.Errorf("unsupported data source type: %s", sourceType)
	}
	return factory, nil
}

// CreateDataSource creates a data source of the specified type
func (r *DataSourceRegistry) CreateDataSource(sourceType string, config map[string]interface{}) (rag.DataSource, error) {
	factory, err := r.GetFactory(sourceType)
	if err != nil {
		return nil, err
	}

	return factory.CreateDataSource(config)
}

// GetSupportedTypes returns all supported data source types
func (r *DataSourceRegistry) GetSupportedTypes() []string {
	types := make([]string, 0, len(r.factories))
	for sourceType := range r.factories {
		types = append(types, sourceType)
	}
	return types
}

// ValidateConfig validates configuration for a data source type
func (r *DataSourceRegistry) ValidateConfig(sourceType string, config map[string]interface{}) error {
	factory, err := r.GetFactory(sourceType)
	if err != nil {
		return err
	}

	return factory.ValidateConfig(config)
}

// Default registry instance
var defaultRegistry = NewDataSourceRegistry()

// RegisterDataSourceFactory registers a factory with the default registry
func RegisterDataSourceFactory(sourceType string, factory DataSourceFactory) {
	defaultRegistry.RegisterFactory(sourceType, factory)
}

// GetSupportedDataSourceTypes returns supported types from the default registry
func GetSupportedDataSourceTypes() []string {
	return defaultRegistry.GetSupportedTypes()
}

// CreateDataSourceFromConfig creates a data source from configuration using the default registry
func CreateDataSourceFromConfig(config map[string]interface{}) (rag.DataSource, error) {
	sourceType, ok := config["type"].(string)
	if !ok {
		return nil, fmt.Errorf("data source type not specified in configuration")
	}

	return defaultRegistry.CreateDataSource(sourceType, config)
}
