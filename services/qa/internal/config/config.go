package config

import (
	"fmt"
	"os"
	"time"
)

type Config struct {
	HTTPAddr        string
	ShutdownTimeout time.Duration
}

func Load() (Config, error) {
	cfg := Config{
		HTTPAddr:        getEnv("QA_HTTP_ADDR", ":8085"),
		ShutdownTimeout: 10 * time.Second,
	}

	if raw := os.Getenv("QA_SHUTDOWN_TIMEOUT"); raw != "" {
		parsed, err := time.ParseDuration(raw)
		if err != nil {
			return Config{}, fmt.Errorf("QA_SHUTDOWN_TIMEOUT must be a duration: %w", err)
		}
		if parsed <= 0 {
			return Config{}, fmt.Errorf("QA_SHUTDOWN_TIMEOUT must be positive")
		}
		cfg.ShutdownTimeout = parsed
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
