package httpapi_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	aihttp "github.com/Sakayori-Iroha-168/Software_Teamwork/services/ai-gateway/internal/http"
	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/ai-gateway/internal/middleware"
	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/ai-gateway/internal/repository"
	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/ai-gateway/internal/service"
)

func TestModelProfileRequiresServiceToken(t *testing.T) {
	server := newHTTPServer(t)
	req := httptest.NewRequest(http.MethodGet, "/internal/v1/model-profiles", nil)
	req.Header.Set("X-Request-Id", "req_auth")
	req.Header.Set("X-Caller-Service", "gateway")
	res := httptest.NewRecorder()

	server.ServeHTTP(res, req)

	if res.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, body = %s", res.Code, res.Body.String())
	}
	if strings.Contains(res.Body.String(), "secret-token") {
		t.Fatalf("response leaked token: %s", res.Body.String())
	}
}

func TestCreateModelProfileDoesNotReturnAPIKey(t *testing.T) {
	server := newHTTPServer(t)
	body := `{"name":"default-chat","purpose":"chat","provider":"openai_compatible","baseUrl":"https://api.example.com/v1","model":"gpt-test","apiKey":"sk_test_secret","enabled":true,"isDefault":true,"supportsStreaming":true}`
	req := authorizedRequest(http.MethodPost, "/internal/v1/model-profiles", body)
	res := httptest.NewRecorder()

	server.ServeHTTP(res, req)

	if res.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", res.Code, res.Body.String())
	}
	if strings.Contains(res.Body.String(), "sk_test_secret") || strings.Contains(res.Body.String(), "apiKey\"") {
		t.Fatalf("response leaked api key: %s", res.Body.String())
	}
	var envelope profileEnvelope
	decodeJSON(t, res.Body, &envelope)
	if !envelope.Data.APIKeyConfigured {
		t.Fatal("apiKeyConfigured = false")
	}
}

func TestValidationErrorDoesNotReturnRawAPIKey(t *testing.T) {
	server := newHTTPServer(t)
	body := `{"name":"bad","purpose":"chat","provider":"openai_compatible","baseUrl":"https://api.example.com/v1?token=abc","model":"gpt-test","apiKey":"sk_should_not_echo","supportsStreaming":true}`
	req := authorizedRequest(http.MethodPost, "/internal/v1/model-profiles", body)
	res := httptest.NewRecorder()

	server.ServeHTTP(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", res.Code, res.Body.String())
	}
	if strings.Contains(res.Body.String(), "sk_should_not_echo") {
		t.Fatalf("response leaked api key: %s", res.Body.String())
	}
}

func TestCreateModelProfileRejectsUnknownFields(t *testing.T) {
	server := newHTTPServer(t)
	body := `{"name":"default-chat","purpose":"chat","provider":"openai_compatible","baseUrl":"https://api.example.com/v1","model":"gpt-test","apiKey":"sk_test_secret","unexpected":"value"}`
	req := authorizedRequest(http.MethodPost, "/internal/v1/model-profiles", body)
	res := httptest.NewRecorder()

	server.ServeHTTP(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", res.Code, res.Body.String())
	}
	if strings.Contains(res.Body.String(), "sk_test_secret") {
		t.Fatalf("response leaked api key: %s", res.Body.String())
	}
}

func TestReadyReportsMissingProfiles(t *testing.T) {
	server := newHTTPServer(t)
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	res := httptest.NewRecorder()

	server.ServeHTTP(res, req)

	if res.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, body = %s", res.Code, res.Body.String())
	}
	var body readinessEnvelope
	decodeJSON(t, res.Body, &body)
	if body.Data.Status != "degraded" {
		t.Fatalf("readiness status = %q", body.Data.Status)
	}
}

func newHTTPServer(t *testing.T) http.Handler {
	t.Helper()
	auth, err := middleware.NewAuthenticator([]string{hashToken("secret-token")})
	if err != nil {
		t.Fatalf("NewAuthenticator() error = %v", err)
	}
	repo := repository.NewMemoryRepository()
	svc := service.New(repo, service.WithClock(func() time.Time {
		return time.Date(2026, 6, 29, 10, 0, 0, 0, time.UTC)
	}), service.WithEncryptionKeyVersion("test-key-v1"), service.WithCredentialEncryptionKey(testCredentialKey()))
	return aihttp.NewServer(svc, aihttp.Config{
		Logger:          slog.New(slog.NewTextHandler(io.Discard, nil)),
		MaxRequestBytes: 4096,
		Authenticator:   auth,
	})
}

func authorizedRequest(method string, path string, body string) *http.Request {
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Service-Token", "secret-token")
	req.Header.Set("X-Caller-Service", "gateway")
	req.Header.Set("X-Request-Id", "req_test")
	return req
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func testCredentialKey() []byte {
	return []byte("0123456789abcdef0123456789abcdef")
}

func decodeJSON(t *testing.T, reader io.Reader, target any) {
	t.Helper()
	if err := json.NewDecoder(reader).Decode(target); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
}

type profileEnvelope struct {
	Data struct {
		ID               string `json:"id"`
		APIKeyConfigured bool   `json:"apiKeyConfigured"`
	} `json:"data"`
	RequestID string `json:"requestId"`
}

type readinessEnvelope struct {
	Data struct {
		Status string `json:"status"`
	} `json:"data"`
	RequestID string `json:"requestId"`
}
