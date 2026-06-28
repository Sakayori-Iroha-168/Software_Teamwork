package http

import (
	"encoding/json"
	"fmt"
	"net/http"

	"software-teamwork/services/qa/internal/service"
)

type Handler struct {
	chat *service.ChatService
}

func NewHandler(chat *service.ChatService) *Handler {
	return &Handler{chat: chat}
}

func (h *Handler) Health(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (h *Handler) StreamChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req service.ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, NewAppError(CodeValidation, "request body must be valid JSON", err))
		return
	}
	if req.Message == "" {
		writeError(w, r, NewAppError(CodeValidation, "message is required", nil))
		return
	}
	if req.ConversationID == "" {
		writeError(w, r, NewAppError(CodeValidation, "conversation_id is required", nil))
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, r, NewAppError(CodeInternal, "streaming is not supported", nil))
		return
	}

	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	events, err := h.chat.Stream(r.Context(), req)
	if err != nil {
		writeSSE(w, "error", map[string]string{"message": "failed to start chat stream"})
		flusher.Flush()
		return
	}

	for event := range events {
		writeSSE(w, event.Event, event.Data)
		flusher.Flush()
	}
}

func writeSSE(w http.ResponseWriter, event string, data any) {
	payload, err := json.Marshal(data)
	if err != nil {
		payload = []byte(`{"message":"failed to encode event"}`)
		event = "error"
	}
	_, _ = fmt.Fprintf(w, "event: %s\n", event)
	_, _ = fmt.Fprintf(w, "data: %s\n\n", payload)
}
