package config

import (
	"os"
	"strconv"
	"strings"
)

// Config holds all bot-fleet service configuration.
type Config struct {
	// Redpanda
	RedpandaBrokers []string

	// Server
	Port string

	// Bot fleet defaults (can be overridden per-benchmark by the incoming event)
	DefaultBotConcurrency   int
	DefaultDurationSeconds  int
	DefaultOrdersPerSecond  int

	// Per-bot HTTP timeout in milliseconds
	BotHTTPTimeoutMs int

	// Log level
	LogLevel string
}

// Load reads configuration from environment variables with sensible defaults.
func Load() *Config {
	brokers := strings.Split(getEnv("REDPANDA_BROKERS", "localhost:19092"), ",")
	for i := range brokers {
		brokers[i] = strings.TrimSpace(brokers[i])
	}

	return &Config{
		RedpandaBrokers:        brokers,
		Port:                   getEnv("SERVER_PORT", "8085"),
		DefaultBotConcurrency:  getEnvInt("BOT_CONCURRENCY", 100),
		DefaultDurationSeconds: getEnvInt("DEFAULT_DURATION_SECONDS", 30),
		DefaultOrdersPerSecond: getEnvInt("DEFAULT_ORDERS_PER_SECOND", 50),
		BotHTTPTimeoutMs:       getEnvInt("BOT_HTTP_TIMEOUT_MS", 2000),
		LogLevel:               getEnv("LOG_LEVEL", "info"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}
