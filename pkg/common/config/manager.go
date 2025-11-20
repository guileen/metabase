package config

import ("context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v2")

// Manager manages application configuration
type Manager struct {
	config     map[string]interface{}
	schema     *ConfigSchema
	watchers   []ConfigWatcher
	validators map[string]ConfigValidator
	mu         sync.RWMutex
	loader     ConfigLoader
	saver      ConfigSaver
}

// ConfigSchema defines the structure and validation rules for configuration
type ConfigSchema struct {
	Version     string                    `yaml:"version"`
	Description string                    `yaml:"description"`
	Definitions map[string]*FieldDefinition `yaml:"definitions"`
	Required    []string                  `yaml:"required"`
	Environment string                    `yaml:"environment"`
}

// FieldDefinition defines a configuration field
type FieldDefinition struct {
	Type        interface{}      `yaml:"type"`        // string, number, boolean, array, object
	Default     interface{}      `yaml:"default"`     // default value
	Description string           `yaml:"description"` // field description
	Required    bool             `yaml:"required"`    // is required
	MinLength   *int             `yaml:"min_length"`  // min length for strings
	MaxLength   *int             `yaml:"max_length"`  // max length for strings
	Minimum     *float64         `yaml:"minimum"`     // min value for numbers
	Maximum     *float64         `yaml:"maximum"`     // max value for numbers
	Pattern     string           `yaml:"pattern"`     // regex pattern for strings
	Enum        []interface{}    `yaml:"enum"`        // allowed values
	EnvVar      string           `yaml:"env_var"`     // environment variable name
	Sensitive   bool             `yaml:"sensitive"`   // is sensitive (password, key)
	Reloadable  bool             `yaml:"reloadable"`  // can be reloaded without restart
}

// ConfigWatcher watches for configuration changes
type ConfigWatcher interface {
	OnConfigChange(key string, oldValue interface{}, newValue interface{}) error
	GetWatchedKeys() []string
}

// ConfigValidator validates configuration values
type ConfigValidator interface {
	Validate(key string, value interface{}) error
	GetKey() string
}

// ConfigLoader loads configuration from various sources
type ConfigLoader interface {
	Load(ctx context.Context) (map[string]interface{}, error)
	GetSource() string
}

// ConfigSaver saves configuration to storage
type ConfigSaver interface {
	Save(ctx context.Context, config map[string]interface{}) error
	GetDestination() string
}

// FileLoader loads configuration from files
type FileLoader struct {
	FilePath string
	Format   string // json, yaml, toml
}

// Load loads configuration from file
func (fl *FileLoader) Load(ctx context.Context) (map[string]interface{}, error) {
	data, err := os.ReadFile(fl.FilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", fl.FilePath, err)
	}

	var config map[string]interface{}

	switch strings.ToLower(fl.Format) {
	case "json":
		err = json.Unmarshal(data, &config)
	case "yaml", "yml":
		err = yaml.Unmarshal(data, &config)
	default:
		return nil, fmt.Errorf("unsupported config format: %s", fl.Format)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to parse config file %s: %w", fl.FilePath, err)
	}

	return config, nil
}

// GetSource returns the source identifier
func (fl *FileLoader) GetSource() string {
	return fmt.Sprintf("file:%s", fl.FilePath)
}

// FileSaver saves configuration to files
type FileSaver struct {
	FilePath string
	Format   string
	Backup   bool
}

// Save saves configuration to file
func (fs *FileSaver) Save(ctx context.Context, config map[string]interface{}) error {
	// Create backup if enabled
	if fs.Backup {
		if err := fs.createBackup(); err != nil {
			return fmt.Errorf("failed to create backup: %w", err)
		}
	}

	// Ensure directory exists
	dir := filepath.Dir(fs.FilePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	var data []byte
	var err error

	switch strings.ToLower(fs.Format) {
	case "json":
		data, err = json.MarshalIndent(config, "", "  ")
	case "yaml", "yml":
		data, err = yaml.Marshal(config)
	default:
		return fmt.Errorf("unsupported config format: %s", fs.Format)
	}

	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	err = os.WriteFile(fs.FilePath, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// createBackup creates a backup of the current config file
func (fs *FileSaver) createBackup() error {
	if _, err := os.Stat(fs.FilePath); os.IsNotExist(err) {
		return nil // No file to backup
	}

	backupPath := fmt.Sprintf("%s.backup.%d", fs.FilePath, time.Now().Unix())
	data, err := os.ReadFile(fs.FilePath)
	if err != nil {
		return err
	}

	return os.WriteFile(backupPath, data, 0644)
}

// GetDestination returns the destination identifier
func (fs *FileSaver) GetDestination() string {
	return fmt.Sprintf("file:%s", fs.FilePath)
}

// EnvironmentLoader loads configuration from environment variables
type EnvironmentLoader struct {
	Prefix string
}

// Load loads configuration from environment variables
func (el *EnvironmentLoader) Load(ctx context.Context) (map[string]interface{}, error) {
	config := make(map[string]interface{})

	for _, env := range os.Environ() {
		if el.Prefix != "" && !strings.HasPrefix(env, el.Prefix) {
			continue
		}

		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := parts[0]
		value := parts[1]

		// Remove prefix
		if el.Prefix != "" {
			key = strings.TrimPrefix(key, el.Prefix)
		}

		// Convert environment variable format to nested keys
		config[key] = value
	}

	return config, nil
}

// GetSource returns the source identifier
func (el *EnvironmentLoader) GetSource() string {
	return "environment"
}

// NewManager creates a new configuration manager
func NewManager(schema *ConfigSchema) *Manager {
	return &Manager{
		config:     make(map[string]interface{}),
		schema:     schema,
		watchers:   make([]ConfigWatcher, 0),
		validators: make(map[string]ConfigValidator),
	}
}

// Load loads configuration from specified sources
func (m *Manager) Load(ctx context.Context, loaders ...ConfigLoader) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Load from each source
	for _, loader := range loaders {
		sourceConfig, err := loader.Load(ctx)
		if err != nil {
			return fmt.Errorf("failed to load from %s: %w", loader.GetSource(), err)
		}

		// Merge configuration (later sources override earlier ones)
		m.mergeConfig(sourceConfig)
	}

	// Apply schema defaults
	if m.schema != nil {
		m.applyDefaults()
	}

	// Validate configuration
	if err := m.validateConfig(); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	return nil
}

// Save saves configuration to specified destinations
func (m *Manager) Save(ctx context.Context, savers ...ConfigSaver) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Validate before saving
	if err := m.validateConfig(); err != nil {
		return fmt.Errorf("cannot save invalid config: %w", err)
	}

	for _, saver := range savers {
		if err := saver.Save(ctx, m.config); err != nil {
			return fmt.Errorf("failed to save to %s: %w", saver.GetDestination(), err)
		}
	}

	return nil
}

// Get retrieves a configuration value
func (m *Manager) Get(key string) (interface{}, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	value, exists := m.getNestedValue(key, m.config)
	return value, exists
}

// Set updates a configuration value
func (m *Manager) Set(key string, value interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Get old value
	oldValue, _ := m.getNestedValue(key, m.config)

	// Validate new value
	if err := m.validateValue(key, value); err != nil {
		return fmt.Errorf("validation failed for %s: %w", key, err)
	}

	// Set new value
	m.setNestedValue(key, value)

	// Notify watchers
	m.notifyWatchers(key, oldValue, value)

	return nil
}

// GetWithDefault retrieves a configuration value with default
func (m *Manager) GetWithDefault(key string, defaultValue interface{}) interface{} {
	value, exists := m.Get(key)
	if !exists {
		return defaultValue
	}
	return value
}

// GetString retrieves a string configuration value
func (m *Manager) GetString(key string) string {
	value, exists := m.Get(key)
	if !exists {
		return ""
	}
	return fmt.Sprintf("%v", value)
}

// GetInt retrieves an integer configuration value
func (m *Manager) GetInt(key string) int {
	value, exists := m.Get(key)
	if !exists {
		return 0
	}

	switch v := value.(type) {
	case int:
		return v
	case float64:
		return int(v)
	case string:
		// Try to parse string as int
		var result int
		fmt.Sscanf(v, "%d", &result)
		return result
	default:
		return 0
	}
}

// GetBool retrieves a boolean configuration value
func (m *Manager) GetBool(key string) bool {
	value, exists := m.Get(key)
	if !exists {
		return false
	}

	switch v := value.(type) {
	case bool:
		return v
	case string:
		return strings.ToLower(v) == "true"
	default:
		return false
	}
}

// GetDuration retrieves a time.Duration configuration value
func (m *Manager) GetDuration(key string) time.Duration {
	value, exists := m.Get(key)
	if !exists {
		return 0
	}

	if duration, ok := value.(time.Duration); ok {
		return duration
	}

	// Try to parse string as duration
	if str, ok := value.(string); ok {
		if duration, err := time.ParseDuration(str); err == nil {
			return duration
		}
	}

	return 0
}

// GetStringSlice retrieves a string slice configuration value
func (m *Manager) GetStringSlice(key string) []string {
	value, exists := m.Get(key)
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
	case string:
		// Parse comma-separated string
		if v == "" {
			return []string{}
		}
		return strings.Split(v, ",")
	default:
		return []string{}
	}
}

// Watch adds a configuration watcher
func (m *Manager) AddWatcher(watcher ConfigWatcher) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.watchers = append(m.watchers, watcher)
}

// AddValidator adds a configuration validator
func (m *Manager) AddValidator(validator ConfigValidator) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.validators[validator.GetKey()] = validator
}

// GetConfigSchema returns the configuration schema
func (m *Manager) GetConfigSchema() *ConfigSchema {
	return m.schema
}

// GetAll returns all configuration as a map
func (m *Manager) GetAll() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Create deep copy
	result := make(map[string]interface{})
	m.deepCopy(result, m.config)
	return result
}

// Export exports configuration to specified format
func (m *Manager) Export(format string) ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	switch strings.ToLower(format) {
	case "json":
		return json.MarshalIndent(m.config, "", "  ")
	case "yaml":
		return yaml.Marshal(m.config)
	default:
		return nil, fmt.Errorf("unsupported export format: %s", format)
	}
}

// Import imports configuration from specified format
func (m *Manager) Import(data []byte, format string) error {
	var config map[string]interface{}

	switch strings.ToLower(format) {
	case "json":
		err := json.Unmarshal(data, &config)
		if err != nil {
			return fmt.Errorf("failed to parse JSON: %w", err)
		}
	case "yaml":
		err := yaml.Unmarshal(data, &config)
		if err != nil {
			return fmt.Errorf("failed to parse YAML: %w", err)
		}
	default:
		return fmt.Errorf("unsupported import format: %s", format)
	}

	// Validate imported configuration
	m.mu.Lock()
	defer m.mu.Unlock()

	// Merge with existing config
	m.mergeConfig(config)

	// Validate merged config
	if err := m.validateConfig(); err != nil {
		return fmt.Errorf("invalid imported configuration: %w", err)
	}

	return nil
}

// Helper methods

func (m *Manager) getNestedValue(key string, config map[string]interface{}) (interface{}, bool) {
	parts := strings.Split(key, ".")
	current := config

	for i, part := range parts {
		if i == len(parts)-1 {
			// Last part, return the value
			value, exists := current[part]
			return value, exists
		}

		// Navigate to nested object
		if next, exists := current[part]; exists {
			if nextMap, ok := next.(map[string]interface{}); ok {
				current = nextMap
			} else {
				return nil, false
			}
		} else {
			return nil, false
		}
	}

	return nil, false
}

func (m *Manager) setNestedValue(key string, value interface{}) {
	parts := strings.Split(key, ".")
	current := m.config

	for i, part := range parts {
		if i == len(parts)-1 {
			// Last part, set the value
			current[part] = value
			return
		}

		// Navigate to or create nested object
		if next, exists := current[part]; exists {
			if nextMap, ok := next.(map[string]interface{}); ok {
				current = nextMap
			} else {
				// Replace non-object with object
				current[part] = make(map[string]interface{})
				current = current[part].(map[string]interface{})
			}
		} else {
			// Create nested object
			current[part] = make(map[string]interface{})
			current = current[part].(map[string]interface{})
		}
	}
}

func (m *Manager) mergeConfig(source map[string]interface{}) {
	m.deepMerge(m.config, source)
}

func (m *Manager) deepMerge(target, source map[string]interface{}) {
	for key, value := range source {
		if existingValue, exists := target[key]; exists {
			// Both exist, check if they are maps
			if existingMap, ok := existingValue.(map[string]interface{}); ok {
				if sourceMap, ok := value.(map[string]interface{}); ok {
					// Both are maps, merge recursively
					m.deepMerge(existingMap, sourceMap)
					continue
				}
			}
		}
		// Set or override
		target[key] = value
	}
}

func (m *Manager) deepCopy(target, source map[string]interface{}) {
	for key, value := range source {
		if sourceMap, ok := value.(map[string]interface{}); ok {
			target[key] = make(map[string]interface{})
			m.deepCopy(target[key].(map[string]interface{}), sourceMap)
		} else {
			target[key] = value
		}
	}
}

func (m *Manager) applyDefaults() {
	if m.schema == nil {
		return
	}

	for key, definition := range m.schema.Definitions {
		if _, exists := m.getNestedValue(key, m.config); !exists && definition.Default != nil {
			m.setNestedValue(key, definition.Default)
		}
	}
}

func (m *Manager) validateConfig() error {
	// Validate required fields
	if m.schema != nil {
		for _, required := range m.schema.Required {
			if _, exists := m.getNestedValue(required, m.config); !exists {
				return fmt.Errorf("required configuration field missing: %s", required)
			}
		}
	}

	// Run field validators
	for key, validator := range m.validators {
		if value, exists := m.getNestedValue(key, m.config); exists {
			if err := validator.Validate(key, value); err != nil {
				return fmt.Errorf("validation failed for %s: %w", key, err)
			}
		}
	}

	return nil
}

func (m *Manager) validateValue(key string, value interface{}) error {
	// Run schema validation
	if m.schema != nil {
		if definition, exists := m.schema.Definitions[key]; exists {
			if err := m.validateFieldDefinition(key, value, definition); err != nil {
				return err
			}
		}
	}

	// Run custom validators
	if validator, exists := m.validators[key]; exists {
		return validator.Validate(key, value)
	}

	return nil
}

func (m *Manager) validateFieldDefinition(key string, value interface{}, definition *FieldDefinition) error {
	// Type validation
	if definition.Type != nil {
		if err := m.validateType(value, definition.Type); err != nil {
			return fmt.Errorf("type validation failed for %s: %w", key, err)
		}
	}

	// String validation
	if str, ok := value.(string); ok {
		if definition.MinLength != nil && len(str) < *definition.MinLength {
			return fmt.Errorf("value too short for %s: minimum %d characters", key, *definition.MinLength)
		}
		if definition.MaxLength != nil && len(str) > *definition.MaxLength {
			return fmt.Errorf("value too long for %s: maximum %d characters", key, *definition.MaxLength)
		}
		if definition.Pattern != "" {
			// Simple regex validation - in a real implementation, use regexp package
			// For now, skip regex validation
		}
	}

	// Number validation
	if num, ok := value.(float64); ok {
		if definition.Minimum != nil && num < *definition.Minimum {
			return fmt.Errorf("value too small for %s: minimum %f", key, *definition.Minimum)
		}
		if definition.Maximum != nil && num > *definition.Maximum {
			return fmt.Errorf("value too large for %s: maximum %f", key, *definition.Maximum)
		}
	}

	// Enum validation
	if len(definition.Enum) > 0 {
		found := false
		for _, enumValue := range definition.Enum {
			if reflect.DeepEqual(value, enumValue) {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("invalid value for %s: must be one of %v", key, definition.Enum)
		}
	}

	return nil
}

func (m *Manager) validateType(value interface{}, expectedType interface{}) error {
	// Simple type validation - in a real implementation, this would be more sophisticated
	switch expectedType {
	case "string":
		if _, ok := value.(string); !ok {
			return fmt.Errorf("expected string, got %T", value)
		}
	case "number":
		if _, ok := value.(float64); !ok {
			return fmt.Errorf("expected number, got %T", value)
		}
	case "boolean":
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("expected boolean, got %T", value)
		}
	case "array":
		if _, ok := value.([]interface{}); !ok {
			return fmt.Errorf("expected array, got %T", value)
		}
	case "object":
		if _, ok := value.(map[string]interface{}); !ok {
			return fmt.Errorf("expected object, got %T", value)
		}
	}

	return nil
}

func (m *Manager) notifyWatchers(key string, oldValue, newValue interface{}) {
	for _, watcher := range m.watchers {
		watchedKeys := watcher.GetWatchedKeys()
		shouldNotify := false

		for _, watchedKey := range watchedKeys {
			if key == watchedKey || strings.HasPrefix(key, watchedKey+".") {
				shouldNotify = true
				break
			}
		}

		if shouldNotify {
			if err := watcher.OnConfigChange(key, oldValue, newValue); err != nil {
				// Log error but don't fail
				fmt.Printf("Config watcher error: %v\n", err)
			}
		}
	}
}

// Helper functions for creating common configurations

// NewDefaultSchema creates a default configuration schema
func NewDefaultSchema() *ConfigSchema {
	return &ConfigSchema{
		Version:     "1.0",
		Description: "MetaBase default configuration schema",
		Required:    []string{"server.port"},
		Definitions: map[string]*FieldDefinition{
			"server.port": {
				Type:     "number",
				Default:  7609,
				Required: true,
				Minimum:  pointerToFloat64(1024),
				Maximum:  pointerToFloat64(65535),
			},
			"server.host": {
				Type:    "string",
				Default: "localhost",
				Pattern: `^[a-zA-Z0-9.-]+$`,
			},
			"database.sqlite_path": {
				Type:     "string",
				Default:  "./data/metabase.db",
				Required: true,
			},
			"auth.jwt_secret": {
				Type:     "string",
				Required: true,
				Sensitive: true,
			},
			"auth.token_expiry": {
				Type:    "string",
				Default: "1h",
			},
		},
	}
}

func pointerToFloat64(v float64) *float64 {
	return &v
}

// GenerateSecretKey generates a random secret key
func GenerateSecretKey(length int) string {
	bytes := make([]byte, length)
	rand.Read(bytes)
	return base64.URLEncoding.EncodeToString(bytes)
}