package config

import (
	"os"
)

type Config struct {
	Port          string
	DatabaseURL   string
	AdminPassword string
	JWTSecret     string
}

func LoadConfig() *Config {
	return &Config{
		Port:          getEnv("PORT", "8089"),
		DatabaseURL:   getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/iicpc?sslmode=disable"),
		AdminPassword: getEnv("ADMIN_PASSWORD", "admin123"),
		JWTSecret:     getEnv("JWT_SECRET", "super-secret-key-change-in-prod"),
	}
}

func getEnv(key, defaultVal string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return defaultVal
}
