package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	DefaultHTTPAddr         = ":8086"
	DefaultDefaultTimeout   = 60 * time.Second
	DefaultMaxRequestBytes  = int64(2 << 20)
	DefaultProfileStorePath = ".ai-gateway-store/model_profiles.json"
	DefaultShutdownTimeout  = 10 * time.Second
	DefaultSecretMode       = "encrypted_column"
)

type Config struct {
	HTTPAddr                   string
	DatabaseURL                string
	ServiceTokenHashes         []string
	SecretMode                 string
	CredentialEncryptionKeyRef string
	DefaultTimeout             time.Duration
	MaxRequestBytes            int64
	ProfileStorePath           string
	ShutdownTimeout            time.Duration
}

func Load() (Config, error) {
	cfg := Config{
		HTTPAddr:                   stringValue("AI_GATEWAY_HTTP_ADDR", DefaultHTTPAddr),
		DatabaseURL:                strings.TrimSpace(os.Getenv("AI_GATEWAY_DATABASE_URL")),
		ServiceTokenHashes:         csvValue("AI_GATEWAY_SERVICE_TOKEN_HASHES", nil),
		SecretMode:                 stringValue("AI_GATEWAY_SECRET_MODE", DefaultSecretMode),
		CredentialEncryptionKeyRef: strings.TrimSpace(os.Getenv("AI_GATEWAY_CREDENTIAL_ENCRYPTION_KEY_REF")),
		DefaultTimeout:             DefaultDefaultTimeout,
		MaxRequestBytes:            DefaultMaxRequestBytes,
		ProfileStorePath:           stringValue("AI_GATEWAY_PROFILE_STORE_PATH", DefaultProfileStorePath),
		ShutdownTimeout:            DefaultShutdownTimeout,
	}
	if raw := strings.TrimSpace(os.Getenv("AI_GATEWAY_DEFAULT_TIMEOUT_MS")); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 1000 {
			return Config{}, fmt.Errorf("AI_GATEWAY_DEFAULT_TIMEOUT_MS must be an integer >= 1000")
		}
		cfg.DefaultTimeout = time.Duration(value) * time.Millisecond
	}
	if raw := strings.TrimSpace(os.Getenv("AI_GATEWAY_MAX_REQUEST_BYTES")); raw != "" {
		value, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || value <= 0 {
			return Config{}, fmt.Errorf("AI_GATEWAY_MAX_REQUEST_BYTES must be a positive integer")
		}
		cfg.MaxRequestBytes = value
	}
	if raw := strings.TrimSpace(os.Getenv("AI_GATEWAY_SHUTDOWN_TIMEOUT")); raw != "" {
		value, err := time.ParseDuration(raw)
		if err != nil || value <= 0 {
			return Config{}, fmt.Errorf("AI_GATEWAY_SHUTDOWN_TIMEOUT must be a positive duration")
		}
		cfg.ShutdownTimeout = value
	}
	if strings.TrimSpace(cfg.HTTPAddr) == "" {
		return Config{}, fmt.Errorf("AI_GATEWAY_HTTP_ADDR must not be empty")
	}
	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("AI_GATEWAY_DATABASE_URL is required")
	}
	if len(cfg.ServiceTokenHashes) == 0 {
		return Config{}, fmt.Errorf("AI_GATEWAY_SERVICE_TOKEN_HASHES is required")
	}
	if strings.TrimSpace(cfg.ProfileStorePath) == "" {
		return Config{}, fmt.Errorf("AI_GATEWAY_PROFILE_STORE_PATH must not be empty")
	}
	for _, hash := range cfg.ServiceTokenHashes {
		if len(hash) != 64 {
			return Config{}, fmt.Errorf("AI_GATEWAY_SERVICE_TOKEN_HASHES must contain SHA-256 hex hashes")
		}
	}
	switch cfg.SecretMode {
	case "encrypted_column":
		if cfg.CredentialEncryptionKeyRef == "" {
			return Config{}, fmt.Errorf("AI_GATEWAY_CREDENTIAL_ENCRYPTION_KEY_REF is required when AI_GATEWAY_SECRET_MODE=encrypted_column")
		}
	case "secret_ref":
		return Config{}, fmt.Errorf("AI_GATEWAY_SECRET_MODE=secret_ref is not supported by this baseline implementation")
	default:
		return Config{}, fmt.Errorf("AI_GATEWAY_SECRET_MODE must be secret_ref or encrypted_column")
	}
	return cfg, nil
}

func stringValue(key string, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func csvValue(key string, fallback []string) []string {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return append([]string(nil), fallback...)
	}
	parts := strings.Split(raw, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.ToLower(strings.TrimSpace(part))
		if trimmed != "" {
			values = append(values, trimmed)
		}
	}
	return values
}
