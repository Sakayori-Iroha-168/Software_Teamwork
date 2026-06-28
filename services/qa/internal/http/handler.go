package http

import (
	"encoding/json"
	"fmt"
	"net/http"

	"software-teamwork/services/qa/internal/service"
)

type Handler struct {
	chat   *service.ChatService
	config *service.ConfigService
}

func NewHandler(chat *service.ChatService, config *service.ConfigService) *Handler {
	return &Handler{chat: chat, config: config}
}

func (h *Handler) Health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"}, "")
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

func (h *Handler) CurrentQAConfig(w http.ResponseWriter, r *http.Request) {
	resp, err := h.config.CurrentQAConfig(r.Context())
	if err != nil {
		writeError(w, r, NewAppError(CodeInternal, "failed to load current QA config", err))
		return
	}
	writeJSON(w, http.StatusOK, resp, requestIDFromContext(r.Context()))
}

func (h *Handler) CreateQAConfig(w http.ResponseWriter, r *http.Request) {
	var req service.QAConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, NewAppError(CodeValidation, "request body must be valid JSON", err))
		return
	}
	resp, err := h.config.CreateQAConfig(r.Context(), req, requestIDFromContext(r.Context()))
	if err != nil {
		writeError(w, r, NewAppError(CodeValidation, err.Error(), err))
		return
	}
	writeJSON(w, http.StatusCreated, resp, requestIDFromContext(r.Context()))
}

func (h *Handler) CurrentLLMConfig(w http.ResponseWriter, r *http.Request) {
	resp, err := h.config.CurrentLLMConfig(r.Context())
	if err != nil {
		writeError(w, r, NewAppError(CodeInternal, "failed to load current LLM config", err))
		return
	}
	writeJSON(w, http.StatusOK, resp, requestIDFromContext(r.Context()))
}

func (h *Handler) CreateLLMConfig(w http.ResponseWriter, r *http.Request) {
	var req service.LLMConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, NewAppError(CodeValidation, "request body must be valid JSON", err))
		return
	}
	resp, err := h.config.CreateLLMConfig(r.Context(), req, requestIDFromContext(r.Context()))
	if err != nil {
		writeError(w, r, NewAppError(CodeValidation, err.Error(), err))
		return
	}
	writeJSON(w, http.StatusCreated, resp, requestIDFromContext(r.Context()))
}

func (h *Handler) TestLLMConnection(w http.ResponseWriter, r *http.Request) {
	var req service.LLMConnectionTestRequest
	if r.Body != nil {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, r, NewAppError(CodeValidation, "request body must be valid JSON", err))
			return
		}
	}
	resp, err := h.config.TestLLMConnection(r.Context(), req)
	if err != nil {
		writeError(w, r, NewAppError(CodeValidation, err.Error(), err))
		return
	}
	writeJSON(w, http.StatusOK, resp, requestIDFromContext(r.Context()))
}

func writeJSON(w http.ResponseWriter, status int, data any, requestID string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"data":      data,
		"requestId": requestID,
	})
}
