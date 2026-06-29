package httpapi

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/ai-gateway/internal/middleware"
	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/ai-gateway/internal/service"
)

type fakeModelInvoker struct {
	embeddingReq service.ProviderEmbeddingRequest
	rerankingReq service.ProviderRerankingRequest
	embeddingFn  func(service.ProviderEmbeddingRequest) (service.EmbeddingResponse, service.ProviderCallMetadata, error)
	rerankingFn  func(service.ProviderRerankingRequest) (service.RerankingResponse, service.ProviderCallMetadata, error)
}

func (f *fakeModelInvoker) CreateEmbeddings(ctx context.Context, req service.ProviderEmbeddingRequest) (service.EmbeddingResponse, service.ProviderCallMetadata, error) {
	f.embeddingReq = req
	if f.embeddingFn != nil {
		return f.embeddingFn(req)
	}
	return service.EmbeddingResponse{}, service.ProviderCallMetadata{}, nil
}

func (f *fakeModelInvoker) CreateReranking(ctx context.Context, req service.ProviderRerankingRequest) (service.RerankingResponse, service.ProviderCallMetadata, error) {
	f.rerankingReq = req
	if f.rerankingFn != nil {
		return f.rerankingFn(req)
	}
	return service.RerankingResponse{}, service.ProviderCallMetadata{}, nil
}

func TestModelProfileRequiresServiceToken(t *testing.T) {
	server := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/internal/v1/model-profiles", nil)
	req.Header.Set("X-Caller-Service", "gateway")
	rec := httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestModelProfileRequiresCallerService(t *testing.T) {
	server := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/internal/v1/model-profiles", nil)
	req.Header.Set("X-Service-Token", "service-token")
	rec := httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestModelProfileRejectsUnknownCallerService(t *testing.T) {
	server := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/internal/v1/model-profiles", nil)
	req.Header.Set("X-Service-Token", "service-token")
	req.Header.Set("X-Caller-Service", "unknown-service")
	rec := httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"code":"forbidden"`)) {
		t.Fatalf("body = %s, want forbidden error", rec.Body.String())
	}
}

func TestCreateModelProfileDoesNotReturnAPIKey(t *testing.T) {
	server := newTestServer(t)
	body := `{"name":"default-chat","purpose":"chat","provider":"siliconflow","baseUrl":"https://api.siliconflow.cn/v1","model":"Qwen","apiKey":"sk-secret-value","enabled":true,"isDefault":true}`
	req := authedRequest(http.MethodPost, "/internal/v1/model-profiles", strings.NewReader(body))
	rec := httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if bytes.Contains(rec.Body.Bytes(), []byte("sk-secret-value")) || bytes.Contains(rec.Body.Bytes(), []byte("apiKey\"")) {
		t.Fatalf("response leaked api key: %s", rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("apiKeyConfigured")) {
		t.Fatalf("response missing apiKeyConfigured: %s", rec.Body.String())
	}
}

func TestInvalidJSONReturnsSecretSafeError(t *testing.T) {
	server := newTestServer(t)
	req := authedRequest(http.MethodPost, "/internal/v1/model-profiles", strings.NewReader(`{"apiKey":"sk-secret"`))
	rec := httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	if bytes.Contains(rec.Body.Bytes(), []byte("sk-secret")) {
		t.Fatalf("error leaked request body: %s", rec.Body.String())
	}
}

func TestReadyReturnsDegradedWhenProfilesMissing(t *testing.T) {
	server := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("degraded")) {
		t.Fatalf("ready body = %s", rec.Body.String())
	}
}

func TestModelInvocationRoutesReturnNotImplemented(t *testing.T) {
	server := newTestServer(t)
	paths := []string{
		"/internal/v1/chat/completions",
	}
	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			req := authedRequest(http.MethodPost, path, strings.NewReader(`{}`))
			rec := httptest.NewRecorder()

			server.ServeHTTP(rec, req)

			if rec.Code != http.StatusNotImplemented {
				t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
			}
			if !bytes.Contains(rec.Body.Bytes(), []byte(`"type":"not_implemented_error"`)) {
				t.Fatalf("body = %s, want OpenAI-style not implemented error", rec.Body.String())
			}
			if bytes.Contains(rec.Body.Bytes(), []byte(`"data"`)) || bytes.Contains(rec.Body.Bytes(), []byte(`"requestId"`)) {
				t.Fatalf("body = %s, model invocation errors must not use project envelope", rec.Body.String())
			}
		})
	}
}

func TestCreateEmbeddingsReturnsOpenAIShapeAndDoesNotLeakSecrets(t *testing.T) {
	invoker := &fakeModelInvoker{
		embeddingFn: func(service.ProviderEmbeddingRequest) (service.EmbeddingResponse, service.ProviderCallMetadata, error) {
			return service.EmbeddingResponse{
				Object: "list",
				Data: []service.EmbeddingVector{{
					Object:    "embedding",
					Index:     0,
					Embedding: json.RawMessage(`[0.1,0.2]`),
				}},
				Model: "BAAI/bge-m3",
				Usage: &service.TokenUsage{PromptTokens: 3, TotalTokens: 3},
			}, service.ProviderCallMetadata{StatusCode: 200}, nil
		},
	}
	server := newTestServerWithInvoker(t, invoker)
	createBody := `{"name":"default-embedding","purpose":"embedding","provider":"siliconflow","baseUrl":"https://api.siliconflow.cn/v1","model":"BAAI/bge-m3","apiKey":"sk-secret-value","enabled":true,"isDefault":true,"dimensions":1024}`
	createReq := authedRequest(http.MethodPost, "/internal/v1/model-profiles", strings.NewReader(createBody))
	createRec := httptest.NewRecorder()
	server.ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("create profile status = %d, body = %s", createRec.Code, createRec.Body.String())
	}

	req := authedRequest(http.MethodPost, "/internal/v1/embeddings", strings.NewReader(`{"model":"BAAI/bge-m3","input":["sensitive text"],"dimensions":512}`))
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if bytes.Contains(rec.Body.Bytes(), []byte("requestId")) || bytes.Contains(rec.Body.Bytes(), []byte(`"data":{"`)) {
		t.Fatalf("model response used project envelope: %s", rec.Body.String())
	}
	if bytes.Contains(rec.Body.Bytes(), []byte("sk-secret-value")) || bytes.Contains(rec.Body.Bytes(), []byte("sensitive text")) {
		t.Fatalf("response leaked sensitive data: %s", rec.Body.String())
	}
	if invoker.embeddingReq.APIKey != "sk-secret-value" {
		t.Fatalf("provider API key was not decrypted")
	}
	if invoker.embeddingReq.Dimensions == nil || *invoker.embeddingReq.Dimensions != 512 {
		t.Fatalf("provider dimensions = %#v, want 512", invoker.embeddingReq.Dimensions)
	}
}

func TestCreateRerankingReturnsDocumentIDsWithoutDocumentText(t *testing.T) {
	invoker := &fakeModelInvoker{
		rerankingFn: func(service.ProviderRerankingRequest) (service.RerankingResponse, service.ProviderCallMetadata, error) {
			return service.RerankingResponse{
				Object: "list",
				Data: []service.RerankingResult{{
					Index:      0,
					DocumentID: "chunk-1",
					Score:      0.88,
				}},
				Model: "BAAI/bge-reranker-v2-m3",
			}, service.ProviderCallMetadata{StatusCode: 200}, nil
		},
	}
	server := newTestServerWithInvoker(t, invoker)
	createBody := `{"name":"default-rerank","purpose":"rerank","provider":"siliconflow","baseUrl":"https://api.siliconflow.cn/v1","model":"BAAI/bge-reranker-v2-m3","apiKey":"sk-secret-value","enabled":true,"isDefault":true,"topN":1}`
	createReq := authedRequest(http.MethodPost, "/internal/v1/model-profiles", strings.NewReader(createBody))
	createRec := httptest.NewRecorder()
	server.ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("create profile status = %d, body = %s", createRec.Code, createRec.Body.String())
	}

	reqBody := `{"model":"BAAI/bge-reranker-v2-m3","query":"sensitive query","documents":[{"id":"chunk-1","text":"sensitive document text"}]}`
	req := authedRequest(http.MethodPost, "/internal/v1/rerankings", strings.NewReader(reqBody))
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if bytes.Contains(rec.Body.Bytes(), []byte("requestId")) || bytes.Contains(rec.Body.Bytes(), []byte("sensitive document text")) || bytes.Contains(rec.Body.Bytes(), []byte("sensitive query")) {
		t.Fatalf("reranking response leaked envelope or text: %s", rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"document_id":"chunk-1"`)) {
		t.Fatalf("body = %s, want document_id", rec.Body.String())
	}
	if invoker.rerankingReq.TopN == nil || *invoker.rerankingReq.TopN != 1 {
		t.Fatalf("provider topN = %#v, want 1", invoker.rerankingReq.TopN)
	}
}

func TestModelInvocationValidationUsesOpenAIErrorShape(t *testing.T) {
	server := newTestServerWithInvoker(t, &fakeModelInvoker{})
	req := authedRequest(http.MethodPost, "/internal/v1/embeddings", strings.NewReader(`{"model":"BAAI/bge-m3","input":[]}`))
	rec := httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"type":"invalid_request_error"`)) || !bytes.Contains(rec.Body.Bytes(), []byte(`"code":"validation_error"`)) {
		t.Fatalf("body = %s, want OpenAI-style validation error", rec.Body.String())
	}
	if bytes.Contains(rec.Body.Bytes(), []byte("requestId")) || bytes.Contains(rec.Body.Bytes(), []byte(`"fields"`)) {
		t.Fatalf("body = %s, model invocation errors must not use project envelope details", rec.Body.String())
	}
}

func TestModelInvocationRoutesRejectUnknownCallerService(t *testing.T) {
	server := newTestServer(t)
	req := httptest.NewRequest(http.MethodPost, "/internal/v1/chat/completions", strings.NewReader(`{}`))
	req.Header.Set("X-Service-Token", "service-token")
	req.Header.Set("X-Caller-Service", "unknown-service")
	rec := httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"type":"permission_error"`)) {
		t.Fatalf("body = %s, want OpenAI-style permission error", rec.Body.String())
	}
}

func TestModelInvocationRoutesRequireServiceToken(t *testing.T) {
	server := newTestServer(t)
	req := httptest.NewRequest(http.MethodPost, "/internal/v1/chat/completions", strings.NewReader(`{}`))
	req.Header.Set("X-Caller-Service", "qa")
	rec := httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"type":"authentication_error"`)) {
		t.Fatalf("body = %s, want OpenAI-style auth error", rec.Body.String())
	}
}

func newTestServer(t *testing.T) *Server {
	t.Helper()
	return newTestServerWithInvoker(t, nil)
}

func newTestServerWithInvoker(t *testing.T, invoker service.ModelInvoker) *Server {
	t.Helper()
	tokenHash := sha256.Sum256([]byte("service-token"))
	auth, err := middleware.NewServiceTokenAuthenticator([]string{"sha256:" + hex.EncodeToString(tokenHash[:])})
	if err != nil {
		t.Fatalf("NewServiceTokenAuthenticator() error = %v", err)
	}
	encryptor, err := service.NewCredentialEncryptor([]byte("12345678901234567890123456789012"), "local-v1")
	if err != nil {
		t.Fatalf("NewCredentialEncryptor() error = %v", err)
	}
	return NewServer(Config{
		Logger:        slog.New(slog.NewTextHandler(io.Discard, nil)),
		Profiles:      service.New(newMemoryRepository(), encryptor, 60000, invoker),
		Authenticator: auth,
	})
}

func authedRequest(method, target string, body io.Reader) *http.Request {
	req := httptest.NewRequest(method, target, body)
	req.Header.Set("X-Service-Token", "service-token")
	req.Header.Set("X-Caller-Service", "gateway")
	req.Header.Set("Content-Type", "application/json")
	return req
}
