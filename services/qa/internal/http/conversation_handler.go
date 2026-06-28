package httpx

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/qa/internal/service"
)

type ConversationHandler struct {
	conversations *service.ConversationService
}

func NewConversationHandler(conversations *service.ConversationService) *ConversationHandler {
	return &ConversationHandler{conversations: conversations}
}

type createConversationBody struct {
	Title string `json:"title"`
}

func (h *ConversationHandler) Create(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteError(w, http.StatusMethodNotAllowed, 40000, "method not allowed")
		return
	}

	var body createConversationBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteError(w, http.StatusBadRequest, 40000, "invalid request body")
		return
	}

	title := strings.TrimSpace(body.Title)
	if title == "" {
		title = "新对话"
	}

	detail, err := h.conversations.Create(r.Context(), title)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, 50000, "failed to create conversation")
		return
	}

	WriteSuccess(w, map[string]any{
		"id":         detail.ID,
		"title":      detail.Title,
		"messages":   []any{},
		"created_at": detail.CreatedAt,
		"updated_at": detail.UpdatedAt,
	})
}

func (h *ConversationHandler) Get(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		WriteError(w, http.StatusMethodNotAllowed, 40000, "method not allowed")
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/api/conversations/")
	if id == "" || strings.Contains(id, "/") {
		WriteError(w, http.StatusBadRequest, 40000, "invalid conversation id")
		return
	}

	detail, err := h.conversations.GetDetail(r.Context(), id)
	if err != nil {
		WriteError(w, http.StatusNotFound, 40400, "conversation not found")
		return
	}

	WriteSuccess(w, detail)
}
