package env

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
)

// LoadEnv loads environment variables from .env file
func LoadEnv() error {
	// Try to load .env file from current directory and parent directories
	for i := 0; i < 3; i++ {
		path := filepath.Join(strings.Repeat("../", i), ".env")
		if _, err := os.Stat(path); err == nil {
			if err := godotenv.Load(path); err != nil {
				return err
			}
			break
		}
	}

	return nil
}

// LoadEnvFrom loads environment variables from a specific .env file path
func LoadEnvFrom(path string) error {
	if _, err := os.Stat(path); err == nil {
		return godotenv.Load(path)
	}
	return nil
}

// GetEnvOrDefault returns environment variable value or default if not set
func GetEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// GetEnvWithPrefix returns environment variable with optional prefix
func GetEnvWithPrefix(prefix, key, defaultValue string) string {
	if prefix != "" {
		prefixedKey := prefix + "_" + strings.ToUpper(key)
		if value := os.Getenv(prefixedKey); value != "" {
			return value
		}
	}

	// Try without prefix
	if value := os.Getenv(key); value != "" {
		return value
	}

	return defaultValue
}
