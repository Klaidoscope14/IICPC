package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Storage  StorageConfig
	Redis    RedisConfig
	Redpanda RedpandaConfig
}

type ServerConfig struct {
	Port string
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

type StorageConfig struct {
	BasePath       string
	MaxUploadBytes int64
	AllowedMIMEs   []string
}

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

type RedpandaConfig struct {
	Brokers []string
}

func Load() (*Config, error) {
	maxUploadMB, _ := strconv.Atoi(getEnv("MAX_UPLOAD_SIZE_MB", "50"))
	redisDB, _ := strconv.Atoi(getEnv("REDIS_DB", "0"))

	cfg := &Config{
		Server: ServerConfig{
			Port: getEnv("SERVER_PORT", "8080"),
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", "postgres"),
			DBName:   getEnv("DB_NAME", "iicpc"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},
		Storage: StorageConfig{
			BasePath:       getEnv("STORAGE_BASE_PATH", "/tmp/iicpc-storage"),
			MaxUploadBytes: int64(maxUploadMB) * 1024 * 1024,
			AllowedMIMEs:   strings.Split(getEnv("ALLOWED_MIME_TYPES", "application/zip,application/x-zip-compressed,application/octet-stream"), ","),
		},
		Redis: RedisConfig{
			Addr:     getEnv("REDIS_ADDR", "localhost:6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       redisDB,
		},
		Redpanda: RedpandaConfig{
			Brokers: strings.Split(getEnv("REDPANDA_BROKERS", "localhost:19092"), ","),
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

func (c *DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.DBName, c.SSLMode,
	)
}
