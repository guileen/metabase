package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestConfigLoading(t *testing.T) {
	// Test default configuration loading
	cfg, err := Load(&LoadOptions{
		DevMode:  true,
		LogLevel: "debug",
	})
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Test default values
	if cfg.GetString("server.host") != "localhost" {
		t.Errorf("Expected host to be 'localhost', got '%s'", cfg.GetString("server.host"))
	}

	if cfg.GetInt("server.port") != 7609 {
		t.Errorf("Expected port to be 7609, got %d", cfg.GetInt("server.port"))
	}

	if !cfg.GetBool("server.dev_mode") {
		t.Error("Expected dev_mode to be true")
	}

	if cfg.GetString("logging.level") != "debug" {
		t.Errorf("Expected log level to be 'debug', got '%s'", cfg.GetString("logging.level"))
	}
}

func TestConfigFromEnv(t *testing.T) {
	// Set environment variables
	os.Setenv("METABASE_SERVER_PORT", "8080")
	os.Setenv("METABASE_SERVER_HOST", "example.com")
	os.Setenv("METABASE_DATABASE_SQLITE_PATH", "/tmp/test.db")
	os.Setenv("METABASE_LOGGING_LEVEL", "warn")
	defer func() {
		os.Unsetenv("METABASE_SERVER_PORT")
		os.Unsetenv("METABASE_SERVER_HOST")
		os.Unsetenv("METABASE_DATABASE_SQLITE_PATH")
		os.Unsetenv("METABASE_LOGGING_LEVEL")
	}()

	cfg, err := Load(&LoadOptions{
		EnvPrefix: "METABASE_",
	})
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Test environment variable overrides
	if cfg.GetInt("server.port") != 8080 {
		t.Errorf("Expected port from env to be 8080, got %d", cfg.GetInt("server.port"))
	}

	if cfg.GetString("server.host") != "example.com" {
		t.Errorf("Expected host from env to be 'example.com', got '%s'", cfg.GetString("server.host"))
	}

	if cfg.GetString("database.sqlite_path") != "/tmp/test.db" {
		t.Errorf("Expected sqlite_path from env to be '/tmp/test.db', got '%s'", cfg.GetString("database.sqlite_path"))
	}

	if cfg.GetString("logging.level") != "warn" {
		t.Errorf("Expected logging level from env to be 'warn', got '%s'", cfg.GetString("logging.level"))
	}
}

func TestConfigFromFile(t *testing.T) {
	// Create temporary config file
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.yaml")

	configContent := `
server:
  host: "filehost"
  port: 9999
  dev_mode: true

database:
  type: "postgres"
  sqlite_path: "/custom/path.db"

logging:
  level: "error"
  format: "text"
  output: "stderr"
`

	err := os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	cfg, err := Load(&LoadOptions{
		ConfigFile: configFile,
	})
	if err != nil {
		t.Fatalf("Failed to load config from file: %v", err)
	}

	appConfig := cfg.GetAppConfig()

	// Test file-based configuration
	if appConfig.Server.Host != "filehost" {
		t.Errorf("Expected host from file to be 'filehost', got '%s'", appConfig.Server.Host)
	}

	if appConfig.Server.Port != 9999 {
		t.Errorf("Expected port from file to be 9999, got %d", appConfig.Server.Port)
	}

	if appConfig.Database.Type != "postgres" {
		t.Errorf("Expected database type from file to be 'postgres', got '%s'", appConfig.Database.Type)
	}

	if appConfig.Logging.Level != "error" {
		t.Errorf("Expected logging level from file to be 'error', got '%s'", appConfig.Logging.Level)
	}
}

func TestGlobalConfig(t *testing.T) {
	// Test global configuration functions
	err := Initialize(&LoadOptions{
		DevMode:  true,
		LogLevel: "info",
	})
	if err != nil {
		t.Fatalf("Failed to initialize global config: %v", err)
	}

	globalCfg := Get()
	if globalCfg == nil {
		t.Fatal("Global config is nil")
	}

	// Test MustGet
	mustCfg := MustGet()
	if mustCfg == nil {
		t.Fatal("MustGet returned nil")
	}

	// Test getting values
	if globalCfg.GetString("server.host") != "localhost" {
		t.Errorf("Expected global host to be 'localhost', got '%s'", globalCfg.GetString("server.host"))
	}

	if !globalCfg.GetBool("server.dev_mode") {
		t.Error("Expected global dev_mode to be true")
	}
}

func TestConfigSetAndGet(t *testing.T) {
	cfg, err := Load(&LoadOptions{})
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Test setting values
	err = cfg.Set("test.key", "test_value")
	if err != nil {
		t.Errorf("Failed to set config value: %v", err)
	}

	// Test getting values
	value, exists := cfg.Get("test.key")
	if !exists {
		t.Error("Config key should exist")
	}

	if value != "test_value" {
		t.Errorf("Expected 'test_value', got '%v'", value)
	}

	// Test typed getters
	err = cfg.Set("test.string", "hello")
	if err != nil {
		t.Errorf("Failed to set string: %v", err)
	}

	if cfg.GetString("test.string") != "hello" {
		t.Errorf("Expected 'hello', got '%s'", cfg.GetString("test.string"))
	}

	err = cfg.Set("test.int", 42)
	if err != nil {
		t.Errorf("Failed to set int: %v", err)
	}

	if cfg.GetInt("test.int") != 42 {
		t.Errorf("Expected 42, got %d", cfg.GetInt("test.int"))
	}

	err = cfg.Set("test.bool", true)
	if err != nil {
		t.Errorf("Failed to set bool: %v", err)
	}

	if !cfg.GetBool("test.bool") {
		t.Error("Expected true")
	}

	// Test GetWithDefault
	def := cfg.GetWithDefault("nonexistent.key", "default_value")
	if def != "default_value" {
		t.Errorf("Expected 'default_value', got '%v'", def)
	}
}

func TestConfigExportImport(t *testing.T) {
	cfg, err := Load(&LoadOptions{
		DevMode:  true,
		LogLevel: "debug",
	})
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Test JSON export
	jsonData, err := cfg.Export("json")
	if err != nil {
		t.Errorf("Failed to export JSON: %v", err)
	}

	if len(jsonData) == 0 {
		t.Error("Exported JSON data is empty")
	}

	// Test YAML export
	yamlData, err := cfg.Export("yaml")
	if err != nil {
		t.Errorf("Failed to export YAML: %v", err)
	}

	if len(yamlData) == 0 {
		t.Error("Exported YAML data is empty")
	}

	// Test JSON export
	exportedData, err := cfg.Export("json")
	if err != nil {
		t.Errorf("Failed to export JSON: %v", err)
	}

	// Verify we got some data
	if len(exportedData) == 0 {
		t.Error("Exported JSON data is empty")
	}

	// Verify some basic config was exported
	if !cfg.GetBool("server.dev_mode") {
		t.Error("Expected dev_mode to be true")
	}
}

func TestDefaultConfigSchema(t *testing.T) {
	defaultConfig := DefaultConfig()
	if defaultConfig == nil {
		t.Fatal("Default config is nil")
	}

	// Test default server config
	if defaultConfig.Server.Host != "localhost" {
		t.Errorf("Expected default host to be 'localhost', got '%s'", defaultConfig.Server.Host)
	}

	if defaultConfig.Server.Port != 7609 {
		t.Errorf("Expected default port to be 7609, got %d", defaultConfig.Server.Port)
	}

	if defaultConfig.Server.DevMode != false {
		t.Error("Expected default dev_mode to be false")
	}

	// Test default database config
	if defaultConfig.Database.Type != "sqlite" {
		t.Errorf("Expected default database type to be 'sqlite', got '%s'", defaultConfig.Database.Type)
	}

	// Test default logging config
	if defaultConfig.Logging.Level != "info" {
		t.Errorf("Expected default logging level to be 'info', got '%s'", defaultConfig.Logging.Level)
	}

	if defaultConfig.Logging.Format != "json" {
		t.Errorf("Expected default logging format to be 'json', got '%s'", defaultConfig.Logging.Format)
	}

	// Test default services config
	if !defaultConfig.Services.EnableGateway {
		t.Error("Expected gateway to be enabled by default")
	}

	if !defaultConfig.Services.EnableAPI {
		t.Error("Expected API to be enabled by default")
	}

	if !defaultConfig.Services.EnableAdmin {
		t.Error("Expected admin to be enabled by default")
	}

	if !defaultConfig.Services.EnableWeb {
		t.Error("Expected web to be enabled by default")
	}

	if defaultConfig.Services.EnableRAG {
		t.Error("Expected RAG to be disabled by default")
	}

	// Test default metrics config
	if !defaultConfig.Metrics.Enabled {
		t.Error("Expected metrics to be enabled by default")
	}

	if defaultConfig.Metrics.Port != 9090 {
		t.Errorf("Expected default metrics port to be 9090, got %d", defaultConfig.Metrics.Port)
	}
}

func BenchmarkConfigGet(b *testing.B) {
	cfg, err := Load(&LoadOptions{DevMode: true})
	if err != nil {
		b.Fatalf("Failed to load config: %v", err)
	}

	// Set some test values
	cfg.Set("bench.string", "test")
	cfg.Set("bench.int", 42)
	cfg.Set("bench.bool", true)

	b.ResetTimer()

	b.Run("GetString", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			cfg.GetString("bench.string")
		}
	})

	b.Run("GetInt", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			cfg.GetInt("bench.int")
		}
	})

	b.Run("GetBool", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			cfg.GetBool("bench.bool")
		}
	})
}

func BenchmarkAppConfigGet(b *testing.B) {
	cfg, err := Load(&LoadOptions{DevMode: true})
	if err != nil {
		b.Fatalf("Failed to load config: %v", err)
	}

	b.ResetTimer()

	b.Run("GetAppConfig", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			cfg.GetAppConfig()
		}
	})
}

func TestConfigConcurrentAccess(t *testing.T) {
	cfg, err := Load(&LoadOptions{DevMode: true})
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Test concurrent reads
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- true }()
			for j := 0; j < 100; j++ {
				cfg.GetString("server.host")
				cfg.GetInt("server.port")
				cfg.GetBool("server.dev_mode")
			}
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatal("Concurrent reads took too long")
		}
	}

	// Test concurrent writes
	for i := 0; i < 5; i++ {
		go func(index int) {
			defer func() { done <- true }()
			for j := 0; j < 50; j++ {
				cfg.Set("test.concurrent", index)
			}
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 5; i++ {
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatal("Concurrent writes took too long")
		}
	}
}
