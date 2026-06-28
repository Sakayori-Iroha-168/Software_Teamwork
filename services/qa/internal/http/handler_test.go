package http

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"software-teamwork/services/qa/internal/repository"
	"software-teamwork/services/qa/internal/service"
)

func TestStreamChatReturnsSSEEvents(t *testing.T) {
	store := repository.NewMemoryStore()
	chatService := service.NewChatService(store)
	router := NewRouter(NewHandler(chatService))

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
