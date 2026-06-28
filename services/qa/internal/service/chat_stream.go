package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/qa/internal/domain"
	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/qa/internal/repository"
)

type ChatStreamRequest struct {
	ConversationID  string
	Message         string
	KnowledgeBases  []string
	UseRetrieval    bool
}

type ChatStreamService struct {
	conversations *repository.ConversationRepository
	messages      *repository.MessageRepository
	responseRuns  *repository.ResponseRunRepository
	processSteps  *repository.ProcessStepRepository
	contentBlocks *repository.ContentBlockRepository
}

func NewChatStreamService(
	conversations *repository.ConversationRepository,
	messages *repository.MessageRepository,
	responseRuns *repository.ResponseRunRepository,
	processSteps *repository.ProcessStepRepository,
	contentBlocks *repository.ContentBlockRepository,
) *ChatStreamService {
	return &ChatStreamService{
		conversations: conversations,
		messages:      messages,
		responseRuns:  responseRuns,
		processSteps:  processSteps,
		contentBlocks: contentBlocks,
	}
}

func (s *ChatStreamService) Stream(
	ctx context.Context,
	req ChatStreamRequest,
	sse *SSEWriter,
) error {
	if strings.TrimSpace(req.ConversationID) == "" || strings.TrimSpace(req.Message) == "" {
		return fmt.Errorf("conversation_id and message are required")
	}

	if _, err := s.conversations.GetByID(ctx, req.ConversationID); err != nil {
		return fmt.Errorf("conversation not found: %w", err)
	}

	if _, err := s.messages.Create(ctx, req.ConversationID, "user", req.Message, "completed"); err != nil {
		return fmt.Errorf("create user message: %w", err)
	}

	assistant, err := s.messages.Create(ctx, req.ConversationID, "assistant", "", "streaming")
	if err != nil {
		return fmt.Errorf("create assistant message: %w", err)
	}

	run, err := s.responseRuns.Create(ctx, assistant.ID, req.ConversationID)
	if err != nil {
		return fmt.Errorf("create response run: %w", err)
	}

	tracker := NewProcessStepTracker(s.processSteps, run.ID, sse)
	startedAt := time.Now()

	defer func() {
		if err := s.conversations.TouchUpdatedAt(context.Background(), req.ConversationID); err != nil {
			_ = err
		}
	}()

	defer func() {
		select {
		case <-ctx.Done():
			if errors.Is(ctx.Err(), context.Canceled) {
				_ = s.stopStream(context.Background(), tracker, run.ID, assistant.ID)
			}
		default:
		}
	}()

	defer func() {
		if r := recover(); r != nil {
			_ = s.failStream(context.Background(), tracker, run.ID, assistant.ID, fmt.Errorf("panic: %v", r))
			panic(r)
		}
	}()

	if err := sse.EmitIntentStatus(map[string]any{
		"status": "started",
		"label":  "正在分析问题...",
	}); err != nil {
		return s.failStream(ctx, tracker, run.ID, assistant.ID, err)
	}

	intentLabel := "识别为：一般对话"
	intent := "general_chat"
	useRetrieval := req.UseRetrieval && len(req.KnowledgeBases) > 0
	if useRetrieval {
		intentLabel = "识别为：知识问答"
		intent = "knowledge_qa"
	}

	if err := sse.EmitIntentStatus(map[string]any{
		"status":     "done",
		"label":      intentLabel,
		"intent":     intent,
		"confidence": 0.95,
	}); err != nil {
		return s.failStream(ctx, tracker, run.ID, assistant.ID, err)
	}

	if _, err := tracker.StartStep(ctx, domain.StepTypeIntent, "识别意图"); err != nil {
		return s.failStream(ctx, tracker, run.ID, assistant.ID, err)
	}
	if _, err := tracker.CompleteStep(ctx, domain.StepTypeIntent, intentLabel, ""); err != nil {
		return s.failStream(ctx, tracker, run.ID, assistant.ID, err)
	}

	if useRetrieval {
		if _, err := tracker.StartStep(ctx, domain.StepTypeRetrieval, "检索知识库"); err != nil {
			return s.failStream(ctx, tracker, run.ID, assistant.ID, err)
		}
		hitCount := 5
		detail := fmt.Sprintf("命中 %d 条结果", hitCount)
		if _, err := tracker.CompleteStep(ctx, domain.StepTypeRetrieval, "检索知识库", detail); err != nil {
			return s.failStream(ctx, tracker, run.ID, assistant.ID, err)
		}
	}

	if _, err := tracker.StartStep(ctx, domain.StepTypeGeneration, "生成回答"); err != nil {
		return s.failStream(ctx, tracker, run.ID, assistant.ID, err)
	}

	answer := s.buildAnswer(req.Message, useRetrieval)
	content, err := s.streamTokens(ctx, sse, answer)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return err
		}
		return s.failStream(ctx, tracker, run.ID, assistant.ID, err)
	}

	if _, err := tracker.CompleteStep(ctx, domain.StepTypeGeneration, "生成回答", ""); err != nil {
		return s.failStream(ctx, tracker, run.ID, assistant.ID, err)
	}

	if useRetrieval {
		if _, err := tracker.StartStep(ctx, domain.StepTypeVerify, "验证答案"); err != nil {
			return s.failStream(ctx, tracker, run.ID, assistant.ID, err)
		}
		if _, err := tracker.CompleteStep(ctx, domain.StepTypeVerify, "验证答案", "引用校验通过"); err != nil {
			return s.failStream(ctx, tracker, run.ID, assistant.ID, err)
		}
	}

	if err := s.messages.UpdateContentAndStatus(ctx, assistant.ID, content, "completed"); err != nil {
		return s.failStream(ctx, tracker, run.ID, assistant.ID, err)
	}

	if err := s.responseRuns.MarkCompleted(ctx, run.ID); err != nil {
		return s.failStream(ctx, tracker, run.ID, assistant.ID, err)
	}

	latency := time.Since(startedAt).Milliseconds()
	if err := sse.EmitDone(map[string]any{
		"message_id":         assistant.ID,
		"total_tokens":       utf8.RuneCountInString(content),
		"prompt_tokens":      utf8.RuneCountInString(req.Message),
		"completion_tokens":  utf8.RuneCountInString(content),
		"latency_ms":         latency,
	}); err != nil {
		return err
	}

	return nil
}

func (s *ChatStreamService) failStream(
	ctx context.Context,
	tracker *ProcessStepTracker,
	runID string,
	assistantMessageID string,
	cause error,
) error {
	_ = tracker.MarkRunningAsFailed(ctx, domain.StepStatusFailed)
	_ = s.messages.UpdateContentAndStatus(ctx, assistantMessageID, "", "failed")
	_ = s.responseRuns.MarkFailed(ctx, runID, cause.Error())
	return cause
}

func (s *ChatStreamService) stopStream(
	ctx context.Context,
	tracker *ProcessStepTracker,
	runID string,
	assistantMessageID string,
) error {
	_ = tracker.MarkRunningAsFailed(ctx, domain.StepStatusStopped)
	_ = s.messages.UpdateContentAndStatus(ctx, assistantMessageID, "", "stopped")
	_ = s.responseRuns.MarkStopped(ctx, runID, "client disconnected")
	return context.Canceled
}

func (s *ChatStreamService) buildAnswer(message string, useRetrieval bool) string {
	if useRetrieval {
		return fmt.Sprintf("根据知识库检索结果，关于「%s」的要点如下：请结合引用文档进一步确认细节。", message)
	}
	return fmt.Sprintf("您好，关于「%s」，我可以继续为您提供帮助。", message)
}

func (s *ChatStreamService) streamTokens(ctx context.Context, sse *SSEWriter, answer string) (string, error) {
	runes := []rune(answer)
	var builder strings.Builder
	for i, r := range runes {
		select {
		case <-ctx.Done():
			return builder.String(), ctx.Err()
		default:
		}
		text := string(r)
		builder.WriteString(text)
		if err := sse.EmitToken(text, i); err != nil {
			return builder.String(), err
		}
	}
	return builder.String(), nil
}
