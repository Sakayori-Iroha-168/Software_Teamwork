package config

import (
	"fmt"
	"os"
	"time"
)

type Config struct {
	Addr         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

func Load() (Config, error) {
	port := getenv("QA_PORT", "8080")
	if port == "" {
		return Config{}, fmt.Errorf("QA_PORT is required")
	}

	return Config{
		Addr:         ":" + port,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 0,
		IdleTimeout:  60 * time.Second,
	}, nil
}

func getenv(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
