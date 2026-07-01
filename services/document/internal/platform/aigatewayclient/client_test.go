package aigatewayclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/document/internal/service"
)

// validChatResponse is a minimal AI Gateway response that satisfies all success checks.
func validChatResponse(content string) map[string]any {
	return map[string]any{
		"choices": []map[string]any{{
			"message": map[string]any{"role": "assistant", "content": content},
		}},
	}
}

func TestNewClientRejectsInvalidURL(t *testing.T) {
	tests := []struct {
		name    string
		baseURL string
		wantErr bool
	}{
		{name: "empty string", baseURL: "", wantErr: true},
		{name: "relative path", baseURL: "/api", wantErr: true},
		{name: "ftp scheme", baseURL: "ftp://host", wantErr: true},
		{name: "empty host", baseURL: "http://", wantErr: true},
		{name: "valid http", baseURL: "http://example.com", wantErr: false},
		{name: "valid https", baseURL: "https://example.com", wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := New(tt.baseURL)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("New(%q) error = nil, want error", tt.baseURL)
				}
				if client != nil {
					t.Fatalf("New(%q) client = non-nil, want nil on error", tt.baseURL)
				}
			} else {
				if err != nil {
					t.Fatalf("New(%q) error = %v, want nil", tt.baseURL, err)
				}
				if client == nil {
					t.Fatal("New() client = nil, want non-nil")
				}
			}
		})
	}
}

func TestChatCompletionSendsCorrectHeaders(t *testing.T) {
	var capturedHeaders http.Header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedHeaders = r.Header.Clone()
		_ = json.NewEncoder(w).Encode(validChatResponse("ok"))
	}))
	defer server.Close()

	client, err := New(server.URL)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	msgs := []service.ChatMessage{{Role: "user", Content: "hello"}}

	// With requestID and serviceToken both non-empty.
	capturedHeaders = nil
	if _, err := client.ChatCompletion(context.Background(), "req-123", "svc-token", msgs); err != nil {
		t.Fatalf("ChatCompletion() error = %v", err)
	}
	if got := capturedHeaders.Get("Content-Type"); !strings.HasPrefix(got, "application/json") {
		t.Fatalf("Content-Type = %q, want application/json", got)
	}
	if got := capturedHeaders.Get("X-Caller-Service"); got != callerService {
		t.Fatalf("X-Caller-Service = %q, want %q", got, callerService)
	}
	if got := capturedHeaders.Get("X-Request-Id"); got != "req-123" {
		t.Fatalf("X-Request-Id = %q, want req-123", got)
	}
	if got := capturedHeaders.Get("X-Service-Token"); got != "svc-token" {
		t.Fatalf("X-Service-Token = %q, want svc-token", got)
	}

	// With both empty: X-Request-Id and X-Service-Token must be absent.
	capturedHeaders = nil
	if _, err := client.ChatCompletion(context.Background(), "", "", msgs); err != nil {
		t.Fatalf("ChatCompletion() (empty ids) error = %v", err)
	}
	if got := capturedHeaders.Get("Content-Type"); !strings.HasPrefix(got, "application/json") {
		t.Fatalf("Content-Type = %q, want application/json", got)
	}
	if got := capturedHeaders.Get("X-Caller-Service"); got != callerService {
		t.Fatalf("X-Caller-Service = %q, want %q", got, callerService)
	}
	if got := capturedHeaders.Get("X-Request-Id"); got != "" {
		t.Fatalf("X-Request-Id = %q, want empty when requestID is empty", got)
	}
	if got := capturedHeaders.Get("X-Service-Token"); got != "" {
		t.Fatalf("X-Service-Token = %q, want empty when serviceToken is empty", got)
	}
}

func TestChatCompletionMapsNonSuccessStatusToDependencyError(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
	}{
		{name: "bad request", status: 400, body: `{"error":"bad request with sk-secret"}`},
		{name: "unauthorized", status: 401, body: `{"error":"unauthorized provider.internal/key"}`},
		{name: "forbidden", status: 403, body: `{"error":"forbidden"}`},
		{name: "internal server error", status: 500, body: `{"error":"server error secret-data"}`},
		{name: "service unavailable", status: 503, body: `{"error":"unavailable"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer server.Close()

			client, err := New(server.URL)
			if err != nil {
				t.Fatalf("New() error = %v", err)
			}
			_, err = client.ChatCompletion(context.Background(), "req-1", "", []service.ChatMessage{
				{Role: "user", Content: "hello"},
			})
			if err == nil {
				t.Fatalf("ChatCompletion() status=%d: error = nil, want error", tt.status)
			}
			// Body must not appear in the error message.
			if strings.Contains(err.Error(), "sk-secret") ||
				strings.Contains(err.Error(), "provider.internal") ||
				strings.Contains(err.Error(), "secret-data") {
				t.Fatalf("error message leaked downstream body content: %v", err)
			}
			appErr, ok := service.Classify(err)
			if !ok || appErr.Code != service.CodeDependency {
				t.Fatalf("error code = %v, want %q", err, service.CodeDependency)
			}
		})
	}
}

func TestChatCompletionMapsInvalidJSONToDependencyError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("not json"))
	}))
	defer server.Close()

	client, err := New(server.URL)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	_, err = client.ChatCompletion(context.Background(), "", "", []service.ChatMessage{
		{Role: "user", Content: "hello"},
	})
	if err == nil {
		t.Fatal("ChatCompletion() error = nil, want error for invalid JSON")
	}
	appErr, ok := service.Classify(err)
	if !ok || appErr.Code != service.CodeDependency {
		t.Fatalf("error code = %v, want %q", err, service.CodeDependency)
	}
}

func TestChatCompletionMapsEmptyChoicesToDependencyError(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{
			name: "empty choices array",
			body: `{"choices":[]}`,
		},
		{
			name: "whitespace-only content",
			body: `{"choices":[{"message":{"role":"assistant","content":"   "}}]}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(tt.body))
			}))
			defer server.Close()

			client, err := New(server.URL)
			if err != nil {
				t.Fatalf("New() error = %v", err)
			}
			_, err = client.ChatCompletion(context.Background(), "", "", []service.ChatMessage{
				{Role: "user", Content: "hello"},
			})
			if err == nil {
				t.Fatalf("ChatCompletion() %s: error = nil, want error", tt.name)
			}
			appErr, ok := service.Classify(err)
			if !ok || appErr.Code != service.CodeDependency {
				t.Fatalf("error code = %v, want %q", err, service.CodeDependency)
			}
		})
	}
}

func TestChatCompletionReturnsContentOnSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{{
				"message": map[string]any{"role": "assistant", "content": "result"},
			}},
		})
	}))
	defer server.Close()

	client, err := New(server.URL)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	content, err := client.ChatCompletion(context.Background(), "req-1", "token", []service.ChatMessage{
		{Role: "user", Content: "hello"},
	})
	if err != nil {
		t.Fatalf("ChatCompletion() error = %v", err)
	}
	if content != "result" {
		t.Fatalf("ChatCompletion() content = %q, want %q", content, "result")
	}
}

func TestChatCompletionHandlesNetworkError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Handler never reached — context is pre-cancelled.
	}))
	defer server.Close()

	client, err := New(server.URL)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // pre-cancel so httpClient.Do fails immediately

	_, err = client.ChatCompletion(ctx, "", "", []service.ChatMessage{{Role: "user", Content: "hello"}})
	if err == nil {
		t.Fatal("ChatCompletion() error = nil, want error for cancelled context")
	}
	appErr, ok := service.Classify(err)
	if !ok || appErr.Code != service.CodeDependency {
		t.Fatalf("error code = %v, want %q", err, service.CodeDependency)
	}
}

func TestChatCompletionHandlesOversizeResponse(t *testing.T) {
	// Response exceeds maxResponseBytes (2<<20 = 2 MiB).
	oversized := strings.Repeat("x", (2<<20)+1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(oversized))
	}))
	defer server.Close()

	client, err := New(server.URL)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	_, err = client.ChatCompletion(context.Background(), "", "", []service.ChatMessage{{Role: "user", Content: "hello"}})
	if err == nil {
		t.Fatal("ChatCompletion() error = nil, want error for oversized response")
	}
	appErr, ok := service.Classify(err)
	if !ok || appErr.Code != service.CodeDependency {
		t.Fatalf("error code = %v, want %q", err, service.CodeDependency)
	}
}
