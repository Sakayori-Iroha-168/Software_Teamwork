package httpx

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/qa/internal/service"
)

type ChatHandler struct {
	chat *service.ChatStreamService
}

func NewChatHandler(chat *service.ChatStreamService) *ChatHandler {
	return &ChatHandler{chat: chat}
}

type chatStreamBody struct {
	Message        string   `json:"message"`
	KnowledgeBases []string `json:"knowledgeBaseIds"`
	Params         *struct {
		TopK               int     `json:"topK"`
		SimilarityThreshold float64 `json:"similarityThreshold"`
		UseRerank          bool    `json:"useRerank"`
		RerankThreshold    float64 `json:"rerankThreshold"`
		RerankTopN         int     `json:"rerankTopN"`
	} `json:"params"`
}

func (h *ChatHandler) Stream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteError(w, http.StatusMethodNotAllowed, 40000, "method not allowed")
		return
	}

	var conversationID string
	if strings.HasPrefix(r.URL.Path, "/api/v1/qa-sessions/") {
		parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/v1/qa-sessions/"), "/")
		if len(parts) > 0 {
			conversationID = parts[0]
		}
	} else {
		var body struct {
			ConversationID string `json:"conversation_id"`
			Message        string `json:"message"`
			KnowledgeBases []string `json:"knowledge_bases"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			WriteError(w, http.StatusBadRequest, 40000, "invalid request body")
			return
		}
		conversationID = body.ConversationID
	}

	var body chatStreamBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteError(w, http.StatusBadRequest, 40000, "invalid request body")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	sse := service.NewSSEWriter(w)
	req := service.ChatStreamRequest{
		ConversationID: strings.TrimSpace(conversationID),
		Message:        strings.TrimSpace(body.Message),
		KnowledgeBases: body.KnowledgeBases,
		UseRetrieval:   len(body.KnowledgeBases) > 0,
	}

	if err := h.chat.Stream(r.Context(), req, sse); err != nil {
		_ = sse.EmitError(50000, "stream failed")
	}
}

func (h *ChatHandler) GetEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		WriteError(w, http.StatusMethodNotAllowed, 40000, "method not allowed")
		return
	}

	responseRunID := r.URL.Query().Get("responseRunId")
	if responseRunID == "" {
		WriteError(w, http.StatusBadRequest, 40000, "responseRunId is required")
		return
	}

	_ = r.URL.Query().Get("afterEventSeq")

	WriteSuccess(w, map[string]any{
		"data": []map[string]any{},
	})
}
