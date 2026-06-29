package http

import (
	"encoding/json"
	"net/http"

	"software-teamwork/services/qa/internal/service"
)

type Handler struct {
	config *service.ConfigService
}

func NewHandler(config *service.ConfigService) *Handler {
	return &Handler{config: config}
}

func (h *Handler) Health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"}, "")
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
	resp, err := h.config.TestLLMConnection(r.Context(), req, requestIDFromContext(r.Context()))
	if err != nil {
		writeError(w, r, NewAppError(CodeValidation, err.Error(), err))
		return
	}
	writeJSON(w, http.StatusCreated, resp, requestIDFromContext(r.Context()))
}

func writeJSON(w http.ResponseWriter, status int, data any, requestID string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"data":      data,
		"requestId": requestID,
	})
}
