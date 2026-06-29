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

func TestConfigEndpointsFollowQASkeleton(t *testing.T) {
	router := newTestRouter()

	t.Run("create qa config version and activate it", func(t *testing.T) {
		body := strings.NewReader(`{
			"defaultKnowledgeBaseIds": ["kb_power_standard"],
			"retrieval": {
				"topK": 8,
				"similarityThreshold": 0.65,
				"useRerank": true,
				"rerankThreshold": 0.5,
				"rerankTopN": 4
			},
			"llm": {
				"provider": "ai-gateway",
				"profileId": "mp_chat_default",
				"modelName": "gpt-4o-mini",
				"timeoutSeconds": 60,
				"temperature": 0.2,
				"maxTokens": 2048
			},
			"agent": {
				"maxIterations": 5,
				"toolTimeoutSeconds": 10,
				"modelTimeoutSeconds": 60,
				"overallTimeoutSeconds": 120,
				"enabledToolNames": ["search_knowledge", "get_citation_source"]
			},
			"activate": true
		}`)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/qa-config-versions", body)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusCreated, rec.Body.String())
		}
		data := responseData(t, rec.Body.String())
		assertEqual(t, data["isActive"], true)
		retrieval := data["retrieval"].(map[string]any)
		assertEqual(t, retrieval["topK"], float64(8))
		llm := data["llm"].(map[string]any)
		assertEqual(t, llm["provider"], "ai-gateway")
		assertNoProviderSecret(t, rec.Body.String())
	})

	t.Run("llm config version can be created without activation", func(t *testing.T) {
		body := strings.NewReader(`{
			"provider": "ai-gateway",
			"profileId": "gateway-qwen-draft",
			"modelName": "Qwen/Qwen2.5-7B-Instruct",
			"timeoutSeconds": 30,
			"temperature": 0.1,
			"maxTokens": 2048,
			"activate": false
		}`)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/llm-config-versions", body)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusCreated, rec.Body.String())
		}
		data := responseData(t, rec.Body.String())
		assertEqual(t, data["profileId"], "gateway-qwen-draft")
		assertEqual(t, data["isActive"], false)
		assertNoProviderSecret(t, rec.Body.String())
	})

	t.Run("llm connection test returns redacted created result", func(t *testing.T) {
		body := strings.NewReader(`{
			"provider": "ai-gateway",
			"profileId": "gateway-qwen-prod",
			"modelName": "Qwen/Qwen2.5-7B-Instruct",
			"timeoutSeconds": 30
		}`)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/llm-connection-tests", body)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusCreated, rec.Body.String())
		}
		data := responseData(t, rec.Body.String())
		assertEqual(t, data["success"], true)
		assertEqual(t, data["modelName"], "Qwen/Qwen2.5-7B-Instruct")
		if _, ok := data["id"].(string); !ok {
			t.Fatalf("connection test response missing id: %s", rec.Body.String())
		}
		assertNoProviderSecret(t, rec.Body.String())
	})
}

func newTestRouter() http.Handler {
	store := repository.NewMemoryStore()
	configService := service.NewConfigService(store)
	return NewRouter(NewHandler(configService))
}

func responseData(t *testing.T, body string) map[string]any {
	t.Helper()
	var payload map[string]any
	if err := json.Unmarshal([]byte(body), &payload); err != nil {
		t.Fatalf("invalid JSON response: %v", err)
	}
	data, ok := payload["data"].(map[string]any)
	if !ok {
		t.Fatalf("response missing data object: %s", body)
	}
	return data
}

func assertEqual(t *testing.T, got any, want any) {
	t.Helper()
	if got != want {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

func assertNoProviderSecret(t *testing.T, body string) {
	t.Helper()
	for _, forbidden := range []string{"apiKey", "apiUrl", "baseUrl"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("response included forbidden provider field %q: %s", forbidden, body)
		}
	}
}
