package config

import "os"

// Config holds all configuration for the API gateway.
type Config struct {
	Port                string
	SubmissionServiceURL string
	OrchestratorURL      string
	RateLimitPerMinute   int
}

// Load reads gateway configuration from environment variables.
func Load() *Config {
	return &Config{
		Port:                getEnv("SERVER_PORT", "8082"),
		SubmissionServiceURL: getEnv("SUBMISSION_SERVICE_URL", "http://localhost:8080"),
		OrchestratorURL:      getEnv("ORCHESTRATOR_URL", "http://localhost:8081"),
		RateLimitPerMinute:   60,
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
