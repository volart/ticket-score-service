package config

import (
	"os"
)

type Config struct {
	Port         string
	DatabasePath string
}

func New() *Config {
	return &Config{
		Port:         getEnv("PORT", "50051"),
		DatabasePath: getEnv("DATABASE_PATH", "./database.db"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
