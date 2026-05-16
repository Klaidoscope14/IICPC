package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config holds all configuration for the benchmark orchestrator service.
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Redpanda RedpandaConfig
	Sandbox  SandboxConfig
}

// ServerConfig holds HTTP server configuration.
type ServerConfig struct {
	Port string
}

// DatabaseConfig holds PostgreSQL connection configuration.
type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

type RedpandaConfig struct {
	Brokers []string
}

// SandboxConfig controls build/deploy execution for submitted engines.
type SandboxConfig struct {
	BuildTimeoutSeconds       int
	DeployTimeoutSeconds      int
	HealthProbeTimeoutSeconds int
	IdleContainerTTLSeconds   int
	RestartAttempts           int
	NetworkMode               string
	BindHost                  string
	ServiceHost               string
}

// Load reads configuration from environment variables with sensible defaults.
func Load() (*Config, error) {
	cfg := &Config{
		Server: ServerConfig{
			Port: getEnv("SERVER_PORT", "8081"),
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", "postgres"),
			DBName:   getEnv("DB_NAME", "iicpc"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},
		Redpanda: RedpandaConfig{
			Brokers: strings.Split(getEnv("REDPANDA_BROKERS", "localhost:19092"), ","),
		},
		Sandbox: SandboxConfig{
			BuildTimeoutSeconds:       getEnvInt("BUILD_TIMEOUT_SECONDS", 300),
			DeployTimeoutSeconds:      getEnvInt("DEPLOY_TIMEOUT_SECONDS", 180),
			HealthProbeTimeoutSeconds: getEnvInt("HEALTH_PROBE_TIMEOUT_SECONDS", 30),
			IdleContainerTTLSeconds:   getEnvInt("IDLE_CONTAINER_TTL_SECONDS", 1800),
			RestartAttempts:           getEnvInt("RESTART_ATTEMPTS", 1),
			NetworkMode:               getEnv("SANDBOX_NETWORK_MODE", "bridge"),
			BindHost:                  getEnv("SANDBOX_BIND_HOST", "127.0.0.1"),
			ServiceHost:               getEnv("SANDBOX_SERVICE_HOST", "localhost"),
		},
	}

	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return defaultValue
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 0 {
		return defaultValue
	}
	return parsed
}

// DSN returns the PostgreSQL connection string.
func (c *DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.DBName, c.SSLMode,
	)
}
