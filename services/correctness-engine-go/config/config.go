package config

import (
	"os"
	"strings"
)

type Config struct {
	Port            string
	RedpandaBrokers []string
	TracesDir       string
}

func Load() Config {
	port := getEnv("PORT", "8086")
	brokers := getEnv("REDPANDA_BROKERS", "localhost:19092")
	tracesDir := getEnv("TRACES_DIR", "../../traces") // Default to shared local traces folder

	return Config{
		Port:            port,
		RedpandaBrokers: strings.Split(brokers, ","),
		TracesDir:       tracesDir,
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
