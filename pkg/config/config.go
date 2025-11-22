package config

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"github.com/guileen/metabase/pkg/common/config"
	"github.com/guileen/metabase/pkg/common/env"
)

// Global instance
var (
	globalConfig *Config
	once         sync.Once
)

// Config provides a unified configuration interface for MetaBase
type Config struct {
	manager *config.Manager
	logger  *slog.Logger
	mu      sync.RWMutex
}

// AppConfig represents the main application configuration schema
type AppConfig struct {
	// Server configuration
	Server ServerConfig `yaml:"server" json:"server"`

	// Database configuration
	Database DatabaseConfig `yaml:"database" json:"database"`

	// Authentication configuration
	Auth AuthConfig `yaml:"auth" json:"auth"`

	// Logging configuration
	Logging LoggingConfig `yaml:"logging" json:"logging"`

	// Services configuration
	Services ServicesConfig `yaml:"services" json:"services"`

	// Storage configuration
	Storage StorageConfig `yaml:"storage" json:"storage"`

	// Metrics configuration
	Metrics MetricsConfig `yaml:"metrics" json:"metrics"`
}

// ServerConfig contains server-related configuration
type ServerConfig struct {
	Host            string `yaml:"host" json:"host"`
	Port            int    `yaml:"port" json:"port"`
	GatewayPort     int    `yaml:"gateway_port" json:"gateway_port"`
	APIPort         int    `yaml:"api_port" json:"api_port"`
	AdminPort       int    `yaml:"admin_port" json:"admin_port"`
	WebPort         int    `yaml:"web_port" json:"web_port"`
	DevMode         bool   `yaml:"dev_mode" json:"dev_mode"`
	ReadTimeout     string `yaml:"read_timeout" json:"read_timeout"`
	WriteTimeout    string `yaml:"write_timeout" json:"write_timeout"`
	IdleTimeout     string `yaml:"idle_timeout" json:"idle_timeout"`
	MaxHeaderBytes  int    `yaml:"max_header_bytes" json:"max_header_bytes"`
	ShutdownTimeout string `yaml:"shutdown_timeout" json:"shutdown_timeout"`
}

// DatabaseConfig contains database-related configuration
type DatabaseConfig struct {
	Type        string `yaml:"type" json:"type"`
	SQLitePath  string `yaml:"sqlite_path" json:"sqlite_path"`
	PebblePath  string `yaml:"pebble_path" json:"pebble_path"`
	MaxConns    int    `yaml:"max_conns" json:"max_conns"`
	MaxIdle     int    `yaml:"max_idle" json:"max_idle"`
	MaxLifetime string `yaml:"max_lifetime" json:"max_lifetime"`
}

// AuthConfig contains authentication-related configuration
type AuthConfig struct {
	JWTSecret      string `yaml:"jwt_secret" json:"jwt_secret"`
	TokenExpiry    string `yaml:"token_expiry" json:"token_expiry"`
	RefreshExpiry  string `yaml:"refresh_expiry" json:"refresh_expiry"`
	PasswordMinLen int    `yaml:"password_min_len" json:"password_min_len"`
	BCryptCost     int    `yaml:"bcrypt_cost" json:"bcrypt_cost"`
}

// LoggingConfig contains logging-related configuration
type LoggingConfig struct {
	Level      string `yaml:"level" json:"level"`
	Format     string `yaml:"format" json:"format"` // json, text
	Output     string `yaml:"output" json:"output"` // stdout, stderr, file
	File       string `yaml:"file" json:"file"`
	MaxSize    int    `yaml:"max_size" json:"max_size"` // MB
	MaxAge     int    `yaml:"max_age" json:"max_age"`   // days
	MaxBackups int    `yaml:"max_backups" json:"max_backups"`
	Compress   bool   `yaml:"compress" json:"compress"`
	RequestID  bool   `yaml:"request_id" json:"request_id"`
	Caller     bool   `yaml:"caller" json:"caller"`
}

// ServicesConfig contains services configuration
type ServicesConfig struct {
	EnableGateway bool `yaml:"enable_gateway" json:"enable_gateway"`
	EnableAPI     bool `yaml:"enable_api" json:"enable_api"`
	EnableAdmin   bool `yaml:"enable_admin" json:"enable_admin"`
	EnableWeb     bool `yaml:"enable_web" json:"enable_web"`
	EnableRAG     bool `yaml:"enable_rag" json:"enable_rag"`
	EnableCASS    bool `yaml:"enable_cass" json:"enable_cass"`
}

// StorageConfig contains storage configuration
type StorageConfig struct {
	UploadPath    string   `yaml:"upload_path" json:"upload_path"`
	MaxFileSize   int64    `yaml:"max_file_size" json:"max_file_size"` // bytes
	AllowedTypes  []string `yaml:"allowed_types" json:"allowed_types"`
	CleanupPeriod string   `yaml:"cleanup_period" json:"cleanup_period"`
}

// MetricsConfig contains metrics configuration
type MetricsConfig struct {
	Enabled   bool   `yaml:"enabled" json:"enabled"`
	Port      int    `yaml:"port" json:"port"`
	Path      string `yaml:"path" json:"path"`
	Namespace string `yaml:"namespace" json:"namespace"`
	Subsystem string `yaml:"subsystem" json:"subsystem"`
}

// LoadOptions contains options for loading configuration
type LoadOptions struct {
	ConfigFile string
	EnvPrefix  string
	DevMode    bool
	LogLevel   string
	Silent     bool
}

// DefaultConfig returns the default configuration
func DefaultConfig() *AppConfig {
	return &AppConfig{
		Server: ServerConfig{
			Host:            "localhost",
			Port:            7609,
			GatewayPort:     7609,
			APIPort:         7610,
			AdminPort:       7680,
			WebPort:         8080,
			DevMode:         false,
			ReadTimeout:     "30s",
			WriteTimeout:    "30s",
			IdleTimeout:     "120s",
			MaxHeaderBytes:  1 << 20, // 1MB
			ShutdownTimeout: "30s",
		},
		Database: DatabaseConfig{
			Type:        "sqlite",
			SQLitePath:  "./data/metabase.db",
			PebblePath:  "./data/pebble",
			MaxConns:    25,
			MaxIdle:     5,
			MaxLifetime: "1h",
		},
		Auth: AuthConfig{
			JWTSecret:      "",
			TokenExpiry:    "1h",
			RefreshExpiry:  "24h",
			PasswordMinLen: 8,
			BCryptCost:     12,
		},
		Logging: LoggingConfig{
			Level:      "info",
			Format:     "json",
			Output:     "stdout",
			File:       "./logs/metabase.log",
			MaxSize:    100, // MB
			MaxAge:     7,   // days
			MaxBackups: 3,
			Compress:   true,
			RequestID:  true,
			Caller:     false,
		},
		Services: ServicesConfig{
			EnableGateway: true,
			EnableAPI:     true,
			EnableAdmin:   true,
			EnableWeb:     true,
			EnableRAG:     false,
			EnableCASS:    false,
		},
		Storage: StorageConfig{
			UploadPath:    "./uploads",
			MaxFileSize:   10 * 1024 * 1024, // 10MB
			AllowedTypes:  []string{"image/*", "application/pdf", "text/*"},
			CleanupPeriod: "24h",
		},
		Metrics: MetricsConfig{
			Enabled:   true,
			Port:      9090,
			Path:      "/metrics",
			Namespace: "metabase",
			Subsystem: "server",
		},
	}
}

// Load loads configuration from various sources
func Load(opts *LoadOptions) (*Config, error) {
	if opts == nil {
		opts = &LoadOptions{}
	}

	// Load .env file first
	if err := env.LoadEnv(); err != nil && !opts.Silent {
		log.Printf("Warning: Could not load .env file: %v", err)
	}

	// Set defaults
	if opts.EnvPrefix == "" {
		opts.EnvPrefix = "METABASE_"
	}

	cfg := &Config{}

	// Create schema
	schema := createConfigSchema()
	cfg.manager = config.NewManager(schema)

	// Setup logger
	cfg.logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Prepare loaders
	var loaders []config.ConfigLoader

	// Load from config file if provided
	if opts.ConfigFile != "" {
		if _, err := os.Stat(opts.ConfigFile); err == nil {
			ext := filepath.Ext(opts.ConfigFile)
			format := "json"
			if ext == ".yaml" || ext == ".yml" {
				format = "yaml"
			}

			loaders = append(loaders, &config.FileLoader{
				FilePath: opts.ConfigFile,
				Format:   format,
			})
		}
	}

	// Load from environment variables (including .env loaded values)
	loaders = append(loaders, &config.EnvironmentLoader{
		Prefix: opts.EnvPrefix,
	})

	// Load configuration
	ctx := context.Background()
	if err := cfg.manager.Load(ctx, loaders...); err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	// Set development mode if specified
	if opts.DevMode {
		cfg.manager.Set("server.dev_mode", true)
		cfg.manager.Set("logging.level", "debug")
		cfg.manager.Set("logging.format", "text")
	}

	// Override with log level if specified
	if opts.LogLevel != "" {
		cfg.manager.Set("logging.level", opts.LogLevel)
	}

	return cfg, nil
}

// MustLoad loads configuration and panics on error
func MustLoad(opts *LoadOptions) *Config {
	cfg, err := Load(opts)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}
	return cfg
}

// Initialize initializes the global configuration
func Initialize(opts *LoadOptions) error {
	var err error
	once.Do(func() {
		globalConfig, err = Load(opts)
	})
	return err
}

// Get returns the global configuration
func Get() *Config {
	once.Do(func() {
		if globalConfig == nil {
			globalConfig, _ = Load(&LoadOptions{})
		}
	})
	return globalConfig
}

// MustGet returns the global configuration and panics if not initialized
func MustGet() *Config {
	cfg := Get()
	if cfg == nil {
		log.Panic("Configuration not initialized")
	}
	return cfg
}

// SetLogger sets the logger for the configuration
func (c *Config) SetLogger(logger *slog.Logger) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.logger = logger
}

// GetLogger returns the configuration logger
func (c *Config) GetLogger() *slog.Logger {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.logger
}

// GetAppConfig returns the application configuration
func (c *Config) GetAppConfig() *AppConfig {
	return &AppConfig{
		Server: ServerConfig{
			Host:            c.GetString("server.host"),
			Port:            c.GetInt("server.port"),
			GatewayPort:     c.GetInt("server.gateway_port"),
			APIPort:         c.GetInt("server.api_port"),
			AdminPort:       c.GetInt("server.admin_port"),
			WebPort:         c.GetInt("server.web_port"),
			DevMode:         c.GetBool("server.dev_mode"),
			ReadTimeout:     c.GetString("server.read_timeout"),
			WriteTimeout:    c.GetString("server.write_timeout"),
			IdleTimeout:     c.GetString("server.idle_timeout"),
			MaxHeaderBytes:  c.GetInt("server.max_header_bytes"),
			ShutdownTimeout: c.GetString("server.shutdown_timeout"),
		},
		Database: DatabaseConfig{
			Type:        c.GetString("database.type"),
			SQLitePath:  c.GetString("database.sqlite_path"),
			PebblePath:  c.GetString("database.pebble_path"),
			MaxConns:    c.GetInt("database.max_conns"),
			MaxIdle:     c.GetInt("database.max_idle"),
			MaxLifetime: c.GetString("database.max_lifetime"),
		},
		Auth: AuthConfig{
			JWTSecret:      c.GetString("auth.jwt_secret"),
			TokenExpiry:    c.GetString("auth.token_expiry"),
			RefreshExpiry:  c.GetString("auth.refresh_expiry"),
			PasswordMinLen: c.GetInt("auth.password_min_len"),
			BCryptCost:     c.GetInt("auth.bcrypt_cost"),
		},
		Logging: LoggingConfig{
			Level:      c.GetString("logging.level"),
			Format:     c.GetString("logging.format"),
			Output:     c.GetString("logging.output"),
			File:       c.GetString("logging.file"),
			MaxSize:    c.GetInt("logging.max_size"),
			MaxAge:     c.GetInt("logging.max_age"),
			MaxBackups: c.GetInt("logging.max_backups"),
			Compress:   c.GetBool("logging.compress"),
			RequestID:  c.GetBool("logging.request_id"),
			Caller:     c.GetBool("logging.caller"),
		},
		Services: ServicesConfig{
			EnableGateway: c.GetBool("services.enable_gateway"),
			EnableAPI:     c.GetBool("services.enable_api"),
			EnableAdmin:   c.GetBool("services.enable_admin"),
			EnableWeb:     c.GetBool("services.enable_web"),
			EnableRAG:     c.GetBool("services.enable_rag"),
			EnableCASS:    c.GetBool("services.enable_cass"),
		},
		Storage: StorageConfig{
			UploadPath:    c.GetString("storage.upload_path"),
			MaxFileSize:   int64(c.GetInt("storage.max_file_size")),
			AllowedTypes:  c.GetStringSlice("storage.allowed_types"),
			CleanupPeriod: c.GetString("storage.cleanup_period"),
		},
		Metrics: MetricsConfig{
			Enabled:   c.GetBool("metrics.enabled"),
			Port:      c.GetInt("metrics.port"),
			Path:      c.GetString("metrics.path"),
			Namespace: c.GetString("metrics.namespace"),
			Subsystem: c.GetString("metrics.subsystem"),
		},
	}
}

// Delegate methods to the underlying config manager
func (c *Config) Get(key string) (interface{}, bool) {
	return c.manager.Get(key)
}

func (c *Config) GetString(key string) string {
	return c.manager.GetString(key)
}

func (c *Config) GetInt(key string) int {
	return c.manager.GetInt(key)
}

func (c *Config) GetBool(key string) bool {
	return c.manager.GetBool(key)
}

func (c *Config) GetWithDefault(key string, defaultValue interface{}) interface{} {
	return c.manager.GetWithDefault(key, defaultValue)
}

func (c *Config) Set(key string, value interface{}) error {
	return c.manager.Set(key, value)
}

func (c *Config) GetStringSlice(key string) []string {
	value, exists := c.Get(key)
	if !exists {
		return []string{}
	}

	switch v := value.(type) {
	case []string:
		return v
	case []interface{}:
		result := make([]string, 0)
		for _, item := range v {
			if str, ok := item.(string); ok {
				result = append(result, str)
			}
		}
		return result
	default:
		return []string{}
	}
}

func (c *Config) AddWatcher(watcher config.ConfigWatcher) {
	c.manager.AddWatcher(watcher)
}

func (c *Config) Export(format string) ([]byte, error) {
	return c.manager.Export(format)
}

// createConfigSchema creates the configuration schema
func pointerToFloat64(v float64) *float64 {
	return &v
}

func createConfigSchema() *config.ConfigSchema {
	return &config.ConfigSchema{
		Version:     "1.0",
		Description: "MetaBase application configuration schema",
		Required:    []string{"server.port"},
		Definitions: map[string]*config.FieldDefinition{
			"server.host": {
				Type:    "string",
				Default: "localhost",
				Pattern: `^[a-zA-Z0-9.-]+$`,
			},
			"server.port": {
				Type:     "number",
				Default:  7609,
				Required: true,
				Minimum:  pointerToFloat64(1024),
				Maximum:  pointerToFloat64(65535),
			},
			"server.gateway_port": {
				Type:    "number",
				Default: 7609,
				Minimum: pointerToFloat64(1024),
				Maximum: pointerToFloat64(65535),
			},
			"server.api_port": {
				Type:    "number",
				Default: 7610,
				Minimum: pointerToFloat64(1024),
				Maximum: pointerToFloat64(65535),
			},
			"server.admin_port": {
				Type:    "number",
				Default: 7680,
				Minimum: pointerToFloat64(1024),
				Maximum: pointerToFloat64(65535),
			},
			"server.web_port": {
				Type:    "number",
				Default: 8080,
				Minimum: pointerToFloat64(1024),
				Maximum: pointerToFloat64(65535),
			},
			"server.dev_mode": {
				Type:    "boolean",
				Default: false,
			},
			"database.type": {
				Type:    "string",
				Default: "sqlite",
				Enum:    []interface{}{"sqlite", "postgres", "mysql"},
			},
			"database.sqlite_path": {
				Type:     "string",
				Default:  "./data/metabase.db",
				Required: true,
			},
			"database.pebble_path": {
				Type:    "string",
				Default: "./data/pebble",
			},
			"database.max_conns": {
				Type:    "number",
				Default: 25,
				Minimum: pointerToFloat64(1),
				Maximum: pointerToFloat64(1000),
			},
			"auth.jwt_secret": {
				Type:      "string",
				Required:  true,
				Sensitive: true,
			},
			"auth.token_expiry": {
				Type:    "string",
				Default: "1h",
			},
			"logging.level": {
				Type:    "string",
				Default: "info",
				Enum:    []interface{}{"debug", "info", "warn", "error"},
			},
			"logging.format": {
				Type:    "string",
				Default: "json",
				Enum:    []interface{}{"json", "text"},
			},
			"logging.output": {
				Type:    "string",
				Default: "stdout",
				Enum:    []interface{}{"stdout", "stderr", "file"},
			},
			"logging.file": {
				Type:    "string",
				Default: "./logs/metabase.log",
			},
			"services.enable_gateway": {
				Type:    "boolean",
				Default: true,
			},
			"services.enable_api": {
				Type:    "boolean",
				Default: true,
			},
			"services.enable_admin": {
				Type:    "boolean",
				Default: true,
			},
			"services.enable_web": {
				Type:    "boolean",
				Default: true,
			},
			"services.enable_rag": {
				Type:    "boolean",
				Default: false,
			},
			"services.enable_cass": {
				Type:    "boolean",
				Default: false,
			},
			"metrics.enabled": {
				Type:    "boolean",
				Default: true,
			},
			"metrics.port": {
				Type:    "number",
				Default: 9090,
				Minimum: pointerToFloat64(1024),
				Maximum: pointerToFloat64(65535),
			},
		},
	}
}
