package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	DefaultHTTPAddr            = ":8080"
	DefaultAuthBaseURL         = "http://localhost:8081"
	DefaultRedisAddr           = "localhost:6379"
	DefaultTokenHashKeyVersion = "v1"
	DefaultAuthTimeout         = 5 * time.Second
	DefaultShutdownTimeout     = 10 * time.Second
	DefaultMaxRequestBytes     = int64(1 << 20)
)

type Config struct {
	HTTPAddr            string
	AuthBaseURL         string
	AuthServiceToken    string
	AuthTimeout         time.Duration
	RedisAddr           string
	RedisPassword       string
	RedisDB             int
	TokenHashSecret     string
	TokenHashKeyVersion string
	MaxRequestBytes     int64
	ShutdownTimeout     time.Duration
}

func Load() (Config, error) {
	cfg := Config{
		HTTPAddr:            stringValue("GATEWAY_HTTP_ADDR", DefaultHTTPAddr),
		AuthBaseURL:         strings.TrimRight(stringValue("GATEWAY_AUTH_BASE_URL", DefaultAuthBaseURL), "/"),
		AuthServiceToken:    os.Getenv("GATEWAY_AUTH_SERVICE_TOKEN"),
		RedisAddr:           stringValue("GATEWAY_REDIS_ADDR", DefaultRedisAddr),
		RedisPassword:       os.Getenv("GATEWAY_REDIS_PASSWORD"),
		TokenHashSecret:     os.Getenv("GATEWAY_TOKEN_HASH_SECRET"),
		TokenHashKeyVersion: stringValue("GATEWAY_TOKEN_HASH_KEY_VERSION", DefaultTokenHashKeyVersion),
		AuthTimeout:         DefaultAuthTimeout,
		MaxRequestBytes:     DefaultMaxRequestBytes,
		ShutdownTimeout:     DefaultShutdownTimeout,
	}

	if cfg.AuthBaseURL == "" {
		return Config{}, fmt.Errorf("GATEWAY_AUTH_BASE_URL is required")
	}
	if strings.TrimSpace(cfg.TokenHashSecret) == "" {
		return Config{}, fmt.Errorf("GATEWAY_TOKEN_HASH_SECRET is required")
	}

	var err error
	if cfg.AuthTimeout, err = durationValue("GATEWAY_AUTH_TIMEOUT", DefaultAuthTimeout); err != nil {
		return Config{}, err
	}
	if cfg.ShutdownTimeout, err = durationValue("GATEWAY_SHUTDOWN_TIMEOUT", DefaultShutdownTimeout); err != nil {
		return Config{}, err
	}
	if cfg.MaxRequestBytes, err = int64Value("GATEWAY_MAX_REQUEST_BYTES", DefaultMaxRequestBytes); err != nil {
		return Config{}, err
	}
	if cfg.RedisDB, err = intValue("GATEWAY_REDIS_DB", 0); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func stringValue(key string, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func durationValue(key string, fallback time.Duration) (time.Duration, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback, nil
	}
	value, err := time.ParseDuration(raw)
	if err != nil || value <= 0 {
		return 0, fmt.Errorf("%s must be a positive duration", key)
	}
	return value, nil
}

func int64Value(key string, fallback int64) (int64, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback, nil
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value <= 0 {
		return 0, fmt.Errorf("%s must be a positive integer", key)
	}
	return value, nil
}

func intValue(key string, fallback int) (int, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 0 {
		return 0, fmt.Errorf("%s must be a non-negative integer", key)
	}
	return value, nil
}
