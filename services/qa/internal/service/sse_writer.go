package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/qa/internal/domain"
)

type SSEWriter struct {
	w        http.ResponseWriter
	eventSeq int
}

func NewSSEWriter(w http.ResponseWriter) *SSEWriter {
	return &SSEWriter{w: w, eventSeq: 0}
}

func (s *SSEWriter) nextSeq() int {
	s.eventSeq++
	return s.eventSeq
}

func (s *SSEWriter) writeEvent(eventType string, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal sse payload: %w", err)
	}
	seq := s.nextSeq()
	if _, err := fmt.Fprintf(s.w, "event: %s\n", eventType); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(s.w, "id: %d\n", seq); err != nil {
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

func (s *SSEWriter) EmitIntent(status string, label string, intent *string, confidence *float64) error {
	payload := map[string]any{
		"status": status,
		"label":  label,
	}
	if intent != nil {
		payload["intent"] = *intent
	}
	if confidence != nil {
		payload["confidence"] = *confidence
	}
	return s.writeEvent("intent", payload)
}

func (s *SSEWriter) EmitStep(_ context.Context, step domain.ThinkingStep) error {
	return s.writeEvent("step", map[string]any{"step": step})
}

func (s *SSEWriter) EmitToken(text string, index int) error {
	return s.writeEvent("token", map[string]any{
		"text":  text,
		"index": index,
	})
}

func (s *SSEWriter) EmitCitation(citation map[string]any) error {
	return s.writeEvent("citation", map[string]any{"citation": citation})
}

func (s *SSEWriter) EmitDone(payload map[string]any) error {
	return s.writeEvent("done", payload)
}

func (s *SSEWriter) EmitError(code int, message string) error {
	return s.writeEvent("error", map[string]any{
		"code":    code,
		"message": message,
	})
}

func (s *SSEWriter) EmitHeartbeat() error {
	return s.writeEvent("heartbeat", map[string]any{})
}

func (s *SSEWriter) CurrentSeq() int {
	return s.eventSeq
}

var _ StepEmitter = (*SSEWriter)(nil)
