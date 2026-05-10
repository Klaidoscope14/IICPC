package config

import "os"

// GetEnv reads an environment variable, returning the default value if unset or empty.
func GetEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// ServerConfig holds common HTTP server configuration.
type ServerConfig struct {
	Port string
}
