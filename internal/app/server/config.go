package server

import "time"

// ConsoleConfig represents console server configuration
type ConsoleConfig struct {
	Port string `yaml:"port" json:"port"`
}

// Config represents server configuration
type Config struct {
	Port          string       `yaml:"port" json:"port"`
	Host          string       `yaml:"host" json:"host"`
	DevMode       bool         `yaml:"dev_mode" json:"dev_mode"`
	Console       ConsoleConfig `yaml:"console" json:"console"`
	EnableNRPC    bool         `yaml:"enable_nrpc" json:"enable_nrpc"`
	EnableStorage bool         `yaml:"enable_storage" json:"enable_storage"`
	EnableConsole bool         `yaml:"enable_console" json:"enable_console"`

	// Additional configuration
	Timeout         time.Duration `yaml:"timeout" json:"timeout"`
	ReadTimeout     time.Duration `yaml:"read_timeout" json:"read_timeout"`
	WriteTimeout    time.Duration `yaml:"write_timeout" json:"write_timeout"`
	IdleTimeout     time.Duration `yaml:"idle_timeout" json:"idle_timeout"`
	MaxHeaderBytes  int           `yaml:"max_header_bytes" json:"max_header_bytes"`
}

// NewConfig creates a new server configuration with defaults
func NewConfig() *Config {
	return &Config{
		Port:            "7609",
		Host:            "localhost",
		DevMode:         false,
		Console: ConsoleConfig{
			Port: "7610",
		},
		EnableNRPC:      true,
		EnableStorage:   true,
		EnableConsole:   true,
		Timeout:         30 * time.Second,
		ReadTimeout:     10 * time.Second,
		WriteTimeout:    10 * time.Second,
		IdleTimeout:     60 * time.Second,
		MaxHeaderBytes:  1 << 20, // 1MB
	}
}
