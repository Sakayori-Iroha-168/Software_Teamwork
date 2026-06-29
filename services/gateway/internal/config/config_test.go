package config

import (
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("GATEWAY_HTTP_ADDR", "")
	t.Setenv("GATEWAY_SERVICE_VERSION", "")
	t.Setenv("GATEWAY_ENV", "")
	t.Setenv("GATEWAY_MAX_BODY_BYTES", "")
	t.Setenv("GATEWAY_REQUEST_TIMEOUT", "")
	t.Setenv("GATEWAY_SHUTDOWN_TIMEOUT", "")
	t.Setenv("GATEWAY_CORS_ALLOWED_ORIGINS", "")
	t.Setenv("GATEWAY_CORS_ALLOWED_METHODS", "")
	t.Setenv("GATEWAY_CORS_ALLOWED_HEADERS", "")
	t.Setenv("GATEWAY_CORS_ALLOW_CREDENTIALS", "")
	t.Setenv("GATEWAY_AI_GATEWAY_BASE_URL", "")
	t.Setenv("GATEWAY_AI_GATEWAY_SERVICE_TOKEN", "")
	t.Setenv("GATEWAY_ADMIN_TOKEN_HASHES", "")
	t.Setenv("GATEWAY_ADMIN_USER_ID", "")
	t.Setenv("GATEWAY_ADMIN_PERMISSIONS", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.HTTPAddr != DefaultHTTPAddr {
		t.Fatalf("HTTPAddr = %q", cfg.HTTPAddr)
	}
	if cfg.MaxBodyBytes != DefaultMaxBodyBytes {
		t.Fatalf("MaxBodyBytes = %d", cfg.MaxBodyBytes)
	}
	if cfg.RequestTimeout != DefaultRequestTimeout {
		t.Fatalf("RequestTimeout = %s", cfg.RequestTimeout)
	}
	if len(cfg.CORSAllowedOrigins) != 1 || cfg.CORSAllowedOrigins[0] != "*" {
		t.Fatalf("CORSAllowedOrigins = %#v", cfg.CORSAllowedOrigins)
	}
	if cfg.AIGatewayBaseURL != DefaultAIGatewayBaseURL {
		t.Fatalf("AIGatewayBaseURL = %q", cfg.AIGatewayBaseURL)
	}
}

func TestLoadParsesEnvironment(t *testing.T) {
	t.Setenv("GATEWAY_HTTP_ADDR", ":9090")
	t.Setenv("GATEWAY_SERVICE_VERSION", "1.2.3")
	t.Setenv("GATEWAY_ENV", "test")
	t.Setenv("GATEWAY_MAX_BODY_BYTES", "2048")
	t.Setenv("GATEWAY_REQUEST_TIMEOUT", "5s")
	t.Setenv("GATEWAY_SHUTDOWN_TIMEOUT", "2s")
	t.Setenv("GATEWAY_CORS_ALLOWED_ORIGINS", "http://localhost:5173, https://example.com")
	t.Setenv("GATEWAY_CORS_ALLOWED_METHODS", "get,post")
	t.Setenv("GATEWAY_CORS_ALLOWED_HEADERS", "Authorization, X-Request-Id")
	t.Setenv("GATEWAY_CORS_ALLOW_CREDENTIALS", "true")
	t.Setenv("GATEWAY_AI_GATEWAY_BASE_URL", "http://ai-gateway:8086")
	t.Setenv("GATEWAY_AI_GATEWAY_SERVICE_TOKEN", "service-token")
	t.Setenv("GATEWAY_ADMIN_TOKEN_HASHES", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	t.Setenv("GATEWAY_ADMIN_USER_ID", "admin_user")
	t.Setenv("GATEWAY_ADMIN_PERMISSIONS", "admin:model-profiles:read,admin:model-profiles:write")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.HTTPAddr != ":9090" || cfg.ServiceVersion != "1.2.3" || cfg.Environment != "test" {
		t.Fatalf("basic config = %+v", cfg)
	}
	if cfg.MaxBodyBytes != 2048 || cfg.RequestTimeout != 5*time.Second || cfg.ShutdownTimeout != 2*time.Second {
		t.Fatalf("numeric config = %+v", cfg)
	}
	if got, want := cfg.CORSAllowedOrigins[1], "https://example.com"; got != want {
		t.Fatalf("origin = %q, want %q", got, want)
	}
	if !cfg.CORSAllowCredentials {
		t.Fatal("CORSAllowCredentials = false")
	}
	if cfg.AIGatewayBaseURL != "http://ai-gateway:8086" || cfg.AIGatewayServiceToken != "service-token" {
		t.Fatalf("ai gateway config = %+v", cfg)
	}
	if cfg.AdminUserID != "admin_user" || len(cfg.AdminTokenHashes) != 1 || len(cfg.AdminPermissions) != 2 {
		t.Fatalf("admin config = %+v", cfg)
	}
}

func TestLoadRejectsInvalidValues(t *testing.T) {
	tests := []struct {
		name string
		key  string
		val  string
	}{
		{name: "max body", key: "GATEWAY_MAX_BODY_BYTES", val: "0"},
		{name: "request timeout", key: "GATEWAY_REQUEST_TIMEOUT", val: "-1s"},
		{name: "shutdown timeout", key: "GATEWAY_SHUTDOWN_TIMEOUT", val: "bad"},
		{name: "cors credentials", key: "GATEWAY_CORS_ALLOW_CREDENTIALS", val: "maybe"},
		{name: "admin token hash", key: "GATEWAY_ADMIN_TOKEN_HASHES", val: "not-a-sha256"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(tt.key, tt.val)
			if _, err := Load(); err == nil {
				t.Fatal("Load() error = nil")
			}
		})
	}
}
