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

	gatewayhttp "github.com/Sakayori-Iroha-168/Software_Teamwork/services/gateway/internal/http"
)

func TestHealthReturnsEnvelopeAndRequestID(t *testing.T) {
	server := newHTTPTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	req.Header.Set("X-Request-Id", "req_health")
	res := httptest.NewRecorder()

	server.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", res.Code, res.Body.String())
	}
	if got := res.Header().Get("X-Request-Id"); got != "req_health" {
		t.Fatalf("X-Request-Id = %q", got)
	}
	var body healthBody
	decodeJSON(t, res.Body, &body)
	if body.RequestID != "req_health" {
		t.Fatalf("requestId = %q", body.RequestID)
	}
	if body.Data.Status != "ok" || body.Data.Service != "gateway" {
		t.Fatalf("health data = %+v", body.Data)
	}
}

func TestReadyReturnsEnvelopeAndGeneratedRequestID(t *testing.T) {
	server := newHTTPTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	res := httptest.NewRecorder()

	server.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", res.Code, res.Body.String())
	}
	headerID := res.Header().Get("X-Request-Id")
	if headerID == "" {
		t.Fatal("missing X-Request-Id")
	}
	var body healthBody
	decodeJSON(t, res.Body, &body)
	if body.RequestID != headerID {
		t.Fatalf("requestId = %q, header = %q", body.RequestID, headerID)
	}
	if body.Data.Status != "ready" {
		t.Fatalf("status = %q", body.Data.Status)
	}
}

func TestNotFoundReturnsErrorEnvelope(t *testing.T) {
	server := newHTTPTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/missing", nil)
	req.Header.Set("X-Request-Id", "req_missing")
	res := httptest.NewRecorder()

	server.ServeHTTP(res, req)

	if res.Code != http.StatusNotFound {
		t.Fatalf("status = %d, body = %s", res.Code, res.Body.String())
	}
	var body errorBody
	decodeJSON(t, res.Body, &body)
	if body.Error.Code != "not_found" || body.Error.RequestID != "req_missing" {
		t.Fatalf("error body = %+v", body.Error)
	}
}

func TestCORSPreflight(t *testing.T) {
	server := newHTTPTestServer(t)
	req := httptest.NewRequest(http.MethodOptions, "/api/v1/knowledge-bases", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	req.Header.Set("Access-Control-Request-Method", http.MethodPost)
	res := httptest.NewRecorder()

	server.ServeHTTP(res, req)

	if res.Code != http.StatusNoContent {
		t.Fatalf("status = %d, body = %s", res.Code, res.Body.String())
	}
	if got := res.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("Access-Control-Allow-Origin = %q", got)
	}
	if got := res.Header().Get("Access-Control-Allow-Headers"); got == "" {
		t.Fatal("missing Access-Control-Allow-Headers")
	}
}

func TestBodyLimitRejectsLargeRequest(t *testing.T) {
	server := gatewayhttp.NewServer(gatewayhttp.Config{
		Logger:             slog.New(slog.NewTextHandler(io.Discard, nil)),
		ServiceVersion:     "test",
		Environment:        "test",
		RequestTimeout:     time.Second,
		MaxBodyBytes:       4,
		CORSAllowedOrigins: []string{"*"},
	})
	req := httptest.NewRequest(http.MethodPost, "/missing", bytes.NewBufferString("12345"))
	req.Header.Set("X-Request-Id", "req_large")
	res := httptest.NewRecorder()

	server.ServeHTTP(res, req)

	if res.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, body = %s", res.Code, res.Body.String())
	}
	var body errorBody
	decodeJSON(t, res.Body, &body)
	if body.Error.Code != "validation_error" || body.Error.RequestID != "req_large" {
		t.Fatalf("error body = %+v", body.Error)
	}
}

func TestAdminModelProfilesProxyForwardsToAIGateway(t *testing.T) {
	var gotPath string
	var gotToken string
	var gotUserID string
	var gotPermissions string
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.RequestURI()
		gotToken = r.Header.Get("X-Service-Token")
		gotUserID = r.Header.Get("X-User-Id")
		gotPermissions = r.Header.Get("X-User-Permissions")
		if got := r.Header.Get("X-Caller-Service"); got != "gateway" {
			t.Fatalf("X-Caller-Service = %q", got)
		}
		response := `{"data":[{"id":"mp_1","name":"chat","purpose":"chat","provider":"openai_compatible","baseUrl":"https://api.example.com/v1","model":"gpt","enabled":true,"isDefault":true,"timeoutMs":60000,"apiKeyConfigured":true,"supportsStreaming":true,"createdAt":"2026-06-29T10:00:00Z","updatedAt":"2026-06-29T10:00:00Z"}],"requestId":"req_proxy"}`
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Request-Id", "req_proxy")
		_, _ = w.Write([]byte(response))
	}))
	defer backend.Close()

	server := gatewayhttp.NewServer(gatewayhttp.Config{
		Logger:                slog.New(slog.NewTextHandler(io.Discard, nil)),
		ServiceVersion:        "test",
		Environment:           "test",
		RequestTimeout:        time.Second,
		MaxBodyBytes:          1024,
		CORSAllowedOrigins:    []string{"*"},
		AIGatewayBaseURL:      backend.URL,
		AIGatewayServiceToken: "internal-token",
		AdminTokenHashes:      []string{hashToken("admin-token")},
		AdminUserID:           "admin_user",
		AdminPermissions:      []string{"admin:model-profiles:*"},
	})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/model-profiles?purpose=chat", nil)
	req.Header.Set("X-Request-Id", "req_proxy")
	req.Header.Set("Authorization", "Bearer admin-token")
	req.Header.Set("X-User-Id", "spoofed_user")
	req.Header.Set("X-User-Permissions", "admin:*")
	res := httptest.NewRecorder()

	server.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", res.Code, res.Body.String())
	}
	if gotPath != "/internal/v1/model-profiles?purpose=chat" {
		t.Fatalf("backend path = %q", gotPath)
	}
	if gotToken != "internal-token" {
		t.Fatalf("X-Service-Token = %q", gotToken)
	}
	if gotUserID != "admin_user" {
		t.Fatalf("X-User-Id = %q", gotUserID)
	}
	if gotPermissions != "admin:model-profiles:*" {
		t.Fatalf("X-User-Permissions = %q", gotPermissions)
	}
	if strings.Contains(res.Body.String(), "apiKey\"") {
		t.Fatalf("proxy response leaked apiKey: %s", res.Body.String())
	}
}

func TestAdminModelProfilesProxyRejectsSpoofedUserHeaders(t *testing.T) {
	backendCalled := false
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		backendCalled = true
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	server := gatewayhttp.NewServer(gatewayhttp.Config{
		Logger:                slog.New(slog.NewTextHandler(io.Discard, nil)),
		ServiceVersion:        "test",
		Environment:           "test",
		RequestTimeout:        time.Second,
		MaxBodyBytes:          1024,
		CORSAllowedOrigins:    []string{"*"},
		AIGatewayBaseURL:      backend.URL,
		AIGatewayServiceToken: "internal-token",
		AdminTokenHashes:      []string{hashToken("admin-token")},
		AdminUserID:           "admin_user",
		AdminPermissions:      []string{"admin:model-profiles:*"},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/model-profiles", nil)
	req.Header.Set("X-Request-Id", "req_spoof")
	req.Header.Set("X-User-Id", "user_admin")
	req.Header.Set("X-User-Permissions", "admin:*")
	res := httptest.NewRecorder()

	server.ServeHTTP(res, req)

	if res.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, body = %s", res.Code, res.Body.String())
	}
	if backendCalled {
		t.Fatal("backend was called for spoofed public headers")
	}
}

func TestAdminModelProfilesProxyRequiresConfiguredAdminPermission(t *testing.T) {
	server := gatewayhttp.NewServer(gatewayhttp.Config{
		Logger:                slog.New(slog.NewTextHandler(io.Discard, nil)),
		ServiceVersion:        "test",
		Environment:           "test",
		RequestTimeout:        time.Second,
		MaxBodyBytes:          1024,
		CORSAllowedOrigins:    []string{"*"},
		AIGatewayBaseURL:      "http://ai-gateway.local",
		AIGatewayServiceToken: "internal-token",
		AdminTokenHashes:      []string{hashToken("admin-token")},
		AdminUserID:           "admin_user",
		AdminPermissions:      []string{"admin:model-profiles:read"},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/model-profiles", nil)
	req.Header.Set("X-Request-Id", "req_forbidden")
	req.Header.Set("Authorization", "Bearer admin-token")
	res := httptest.NewRecorder()

	server.ServeHTTP(res, req)

	if res.Code != http.StatusForbidden {
		t.Fatalf("status = %d, body = %s", res.Code, res.Body.String())
	}
}

func TestAdminModelProfilesProxyNormalizesDownstreamError(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"code":"unauthorized","message":"bad internal token","requestId":"req_downstream","fields":{"token":"internal-token"}}}`))
	}))
	defer backend.Close()

	server := gatewayhttp.NewServer(gatewayhttp.Config{
		Logger:                slog.New(slog.NewTextHandler(io.Discard, nil)),
		ServiceVersion:        "test",
		Environment:           "test",
		RequestTimeout:        time.Second,
		MaxBodyBytes:          1024,
		CORSAllowedOrigins:    []string{"*"},
		AIGatewayBaseURL:      backend.URL,
		AIGatewayServiceToken: "internal-token",
		AdminTokenHashes:      []string{hashToken("admin-token")},
		AdminUserID:           "admin_user",
		AdminPermissions:      []string{"admin:model-profiles:*"},
	})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/model-profiles", nil)
	req.Header.Set("X-Request-Id", "req_public")
	req.Header.Set("Authorization", "Bearer admin-token")
	res := httptest.NewRecorder()

	server.ServeHTTP(res, req)

	if res.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, body = %s", res.Code, res.Body.String())
	}
	if strings.Contains(res.Body.String(), "bad internal token") || strings.Contains(res.Body.String(), "internal-token") || strings.Contains(res.Body.String(), "req_downstream") {
		t.Fatalf("proxy leaked downstream error: %s", res.Body.String())
	}
	var body errorBody
	decodeJSON(t, res.Body, &body)
	if body.Error.Code != "dependency_error" || body.Error.RequestID != "req_public" {
		t.Fatalf("error body = %+v", body.Error)
	}
}

func newHTTPTestServer(t *testing.T) http.Handler {
	t.Helper()
	return gatewayhttp.NewServer(gatewayhttp.Config{
		Logger:             slog.New(slog.NewTextHandler(io.Discard, nil)),
		ServiceVersion:     "test",
		Environment:        "test",
		RequestTimeout:     time.Second,
		MaxBodyBytes:       1024,
		CORSAllowedOrigins: []string{"*"},
	})
}

func decodeJSON(t *testing.T, reader io.Reader, target any) {
	t.Helper()
	if err := json.NewDecoder(reader).Decode(target); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

type healthBody struct {
	Data struct {
		Status      string `json:"status"`
		Service     string `json:"service"`
		Version     string `json:"version"`
		Environment string `json:"environment"`
	} `json:"data"`
	RequestID string `json:"requestId"`
}

type errorBody struct {
	Error struct {
		Code      string `json:"code"`
		Message   string `json:"message"`
		RequestID string `json:"requestId"`
	} `json:"error"`
}
