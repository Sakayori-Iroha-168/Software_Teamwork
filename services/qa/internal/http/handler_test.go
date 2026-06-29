package http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"software-teamwork/services/qa/internal/repository"
	"software-teamwork/services/qa/internal/service"
)

func TestStreamChatReturnsSSEEvents(t *testing.T) {
	router := newTestRouter()

	body := strings.NewReader(`{
		"conversation_id": "conv_test",
		"message": "帮我检索知识库里的规程",
		"knowledge_bases": ["kb_power_standard"]
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/chat/stream", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	got := rec.Body.String()
	if !strings.Contains(got, "event: intent_status") {
		t.Fatalf("response missing intent_status event: %s", got)
	}
	if !strings.Contains(got, "event: done") {
		t.Fatalf("response missing done event: %s", got)
	}
}

func TestConfigEndpoints(t *testing.T) {
	router := newTestRouter()

	t.Run("create qa config", func(t *testing.T) {
		body := strings.NewReader(`{
			"topK": 8,
			"similarityThreshold": 0.65,
			"useRerank": true,
			"rerankThreshold": 0.5,
			"rerankTopN": 4,
			"activate": true,
			"knowledgeBases": [
				{"externalKbId":"kb_power_standard","kbType":"technical_supervision","displayNameSnapshot":"电力标准规范库","sortOrder":1}
			],
			"createdByUserId": "admin-001"
		}`)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/qa-config-versions", body)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusCreated, rec.Body.String())
		}
		assertJSONContains(t, rec.Body.String(), "topK", float64(8))
		assertJSONContains(t, rec.Body.String(), "isActive", true)
	})

	t.Run("create llm config uses gateway profile", func(t *testing.T) {
		body := strings.NewReader(`{
			"profileId": "gateway-qwen-prod",
			"modelName": "qwen-test",
			"timeoutSeconds": 30,
			"temperature": 0.2,
			"maxTokens": 2048
		}`)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/llm-config-versions", body)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusCreated, rec.Body.String())
		}
		got := rec.Body.String()
		for _, forbidden := range []string{"apiKey", "apiUrl", "provider"} {
			if strings.Contains(got, forbidden) {
				t.Fatalf("response included forbidden provider field %q: %s", forbidden, got)
			}
		}
		assertJSONContains(t, got, "profileId", "gateway-qwen-prod")
	})

	t.Run("test llm connection", func(t *testing.T) {
		body := strings.NewReader(`{
			"profileId": "gateway-qwen-prod",
			"modelName": "qwen-test"
		}`)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/llm-connection-tests", body)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
		}
		assertJSONContains(t, rec.Body.String(), "success", true)
		assertJSONContains(t, rec.Body.String(), "status", "succeeded")
	})
}

func newTestRouter() http.Handler {
	store := repository.NewMemoryStore()
	chatService := service.NewChatService(store)
	configService := service.NewConfigService(store)
	return NewRouter(NewHandler(chatService, configService))
}

func assertJSONContains(t *testing.T, body string, key string, want any) {
	t.Helper()
	var payload map[string]any
	if err := json.Unmarshal([]byte(body), &payload); err != nil {
		t.Fatalf("invalid JSON response: %v", err)
	}
	data, ok := payload["data"].(map[string]any)
	if !ok {
		t.Fatalf("response missing data object: %s", body)
	}
	if got := data[key]; got != want {
		t.Fatalf("data[%s] = %#v, want %#v", key, got, want)
	}
}
