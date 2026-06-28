package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/qa/internal/domain"
)

type SSEWriter struct {
	w http.ResponseWriter
}

func NewSSEWriter(w http.ResponseWriter) *SSEWriter {
	return &SSEWriter{w: w}
}

func (s *SSEWriter) writeEvent(event string, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal sse payload: %w", err)
	}
	if _, err := fmt.Fprintf(s.w, "event: %s\n", event); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(s.w, "data: %s\n\n", data); err != nil {
		return err
	}
	if flusher, ok := s.w.(http.Flusher); ok {
		flusher.Flush()
	}
	return nil
}

func (s *SSEWriter) EmitThinkingStep(_ context.Context, step domain.ThinkingStep) error {
	return s.writeEvent("thinking_step", map[string]any{"step": step})
}

func (s *SSEWriter) EmitIntentStatus(payload map[string]any) error {
	return s.writeEvent("intent_status", payload)
}

func (s *SSEWriter) EmitToken(text string, index int) error {
	return s.writeEvent("token", map[string]any{
		"text":  text,
		"index": index,
	})
}

func (s *SSEWriter) EmitDone(payload map[string]any) error {
	return s.writeEvent("done", payload)
}

func (s *SSEWriter) EmitError(code int, message string, fatal bool) error {
	return s.writeEvent("error", map[string]any{
		"code":    code,
		"message": message,
		"fatal":   fatal,
	})
}

// Ensure SSEWriter satisfies StepEmitter.
var _ StepEmitter = (*SSEWriter)(nil)
