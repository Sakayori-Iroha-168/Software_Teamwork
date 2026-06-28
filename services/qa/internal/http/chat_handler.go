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
	ConversationID string   `json:"conversation_id"`
	Message        string   `json:"message"`
	KnowledgeBases []string `json:"knowledge_bases"`
	Params         *struct {
		UseRerank bool `json:"use_rerank"`
	} `json:"params"`
}

func (h *ChatHandler) Stream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteError(w, http.StatusMethodNotAllowed, 40000, "method not allowed")
		return
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
		ConversationID: strings.TrimSpace(body.ConversationID),
		Message:        strings.TrimSpace(body.Message),
		KnowledgeBases: body.KnowledgeBases,
		UseRetrieval:   len(body.KnowledgeBases) > 0,
	}

	if err := h.chat.Stream(r.Context(), req, sse); err != nil {
		_ = sse.EmitError(50000, "stream failed", true)
	}
}
