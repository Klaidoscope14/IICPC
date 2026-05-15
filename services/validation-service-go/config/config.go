package config

import (
	"log/slog"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Port              string
	DatabaseURL       string
	RedpandaBrokers   []string
	ConsumerGroupID   string
	RedisAddr         string
	RedisPassword     string
	RedisDB           int
	MaxExtractedBytes int64
	MaxFileCount      int
}

func LoadConfig() *Config {
	cfg := &Config{
		Port:              getEnv("PORT", "8081"), // Submission runs on 8080 usually
		DatabaseURL:       getEnv("DATABASE_URL", "postgres://iicpc:iicpc_secret@localhost:5432/iicpc?sslmode=disable"),
		RedpandaBrokers:   strings.Split(getEnv("REDPANDA_BROKERS", "localhost:9092"), ","),
		ConsumerGroupID:   getEnv("CONSUMER_GROUP_ID", "validation-service-group"),
		RedisAddr:         getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword:     getEnv("REDIS_PASSWORD", ""),
		RedisDB:           getEnvAsInt("REDIS_DB", 0),
		MaxExtractedBytes: getEnvAsInt64("MAX_EXTRACTED_BYTES", 500*1024*1024), // 500 MB
		MaxFileCount:      getEnvAsInt("MAX_FILE_COUNT", 1000),
	}

	return cfg
}

func getEnv(key, defaultVal string) string {
	if val, exists := os.LookupEnv(key); exists {
		return val
	}
	return defaultVal
}

func getEnvAsInt(key string, defaultVal int) int {
	valStr := getEnv(key, "")
	if val, err := strconv.Atoi(valStr); err == nil {
		return val
	}
	if valStr != "" {
		slog.Warn("Invalid integer environment variable, using default", "key", key, "value", valStr, "default", defaultVal)
	}
	return defaultVal
}

func getEnvAsInt64(key string, defaultVal int64) int64 {
	valStr := getEnv(key, "")
	if val, err := strconv.ParseInt(valStr, 10, 64); err == nil {
		return val
	}
	if valStr != "" {
		slog.Warn("Invalid int64 environment variable, using default", "key", key, "value", valStr, "default", defaultVal)
	}
	return defaultVal
}
