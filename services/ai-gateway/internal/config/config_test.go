package config

import "testing"

func TestLoadParsesRequiredConfig(t *testing.T) {
	t.Setenv("AI_GATEWAY_HTTP_ADDR", ":9090")
	t.Setenv("AI_GATEWAY_DATABASE_URL", "postgres://user:pass@localhost:5432/ai_gateway")
	t.Setenv("AI_GATEWAY_SERVICE_TOKEN_HASHES", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	t.Setenv("AI_GATEWAY_SECRET_MODE", "encrypted_column")
	t.Setenv("AI_GATEWAY_CREDENTIAL_ENCRYPTION_KEY_REF", "local-key-v1")
	t.Setenv("AI_GATEWAY_DEFAULT_TIMEOUT_MS", "45000")
	t.Setenv("AI_GATEWAY_PROFILE_STORE_PATH", "tmp/profiles.json")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.HTTPAddr != ":9090" || cfg.DatabaseURL == "" {
		t.Fatalf("basic config = %+v", cfg)
	}
	if cfg.DefaultTimeout.Milliseconds() != 45000 {
		t.Fatalf("DefaultTimeout = %s", cfg.DefaultTimeout)
	}
	if cfg.ProfileStorePath != "tmp/profiles.json" {
		t.Fatalf("ProfileStorePath = %q", cfg.ProfileStorePath)
	}
}

func TestLoadRejectsMissingRequiredConfig(t *testing.T) {
	t.Setenv("AI_GATEWAY_DATABASE_URL", "")
	t.Setenv("AI_GATEWAY_SERVICE_TOKEN_HASHES", "")
	if _, err := Load(); err == nil {
		t.Fatal("Load() error = nil")
	}
}

func TestLoadRejectsUnsupportedSecretRefMode(t *testing.T) {
	t.Setenv("AI_GATEWAY_DATABASE_URL", "postgres://user:pass@localhost:5432/ai_gateway")
	t.Setenv("AI_GATEWAY_SERVICE_TOKEN_HASHES", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	t.Setenv("AI_GATEWAY_SECRET_MODE", "secret_ref")

	if _, err := Load(); err == nil {
		t.Fatal("Load() error = nil")
	}
}
