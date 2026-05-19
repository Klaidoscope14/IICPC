package config

import (
	"os"
	"strconv"
)

// Config holds all configuration for the API gateway.
type Config struct {
	Port                 string
	SubmissionServiceURL string
	ValidationServiceURL string
	OrchestratorURL      string
	BotFleetURL          string
	RateLimitPerMinute   int
	MaxBodyBytes         int64
	AuthToken            string
}

// Load reads gateway configuration from environment variables.
func Load() *Config {
	return &Config{
		Port:                 getEnv("SERVER_PORT", "8082"),
		SubmissionServiceURL: getEnv("SUBMISSION_SERVICE_URL", "http://localhost:8080"),
		ValidationServiceURL: getEnv("VALIDATION_SERVICE_URL", "http://localhost:8084"),
		OrchestratorURL:      getEnv("ORCHESTRATOR_URL", "http://localhost:8081"),
		BotFleetURL:          getEnv("BOT_FLEET_URL", "http://localhost:8085"),
		RateLimitPerMinute:   getEnvAsInt("RATE_LIMIT_PER_MINUTE", 600),
		MaxBodyBytes:         int64(getEnvAsInt("MAX_BODY_SIZE_MB", 64)) * 1024 * 1024,
		AuthToken:            getEnv("API_AUTH_TOKEN", ""),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}
	return parsed
}
