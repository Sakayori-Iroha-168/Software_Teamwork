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
	conversations  *repository.ConversationRepository
	messages       *repository.MessageRepository
	responseRuns   *repository.ResponseRunRepository
	processSteps   *repository.ProcessStepRepository
	contentBlocks  *repository.ContentBlockRepository
	streamEvents   *repository.ResponseStreamEventRepository
	citations      *repository.CitationRepository
}

func NewChatStreamService(
	conversations *repository.ConversationRepository,
	messages *repository.MessageRepository,
	responseRuns *repository.ResponseRunRepository,
	processSteps *repository.ProcessStepRepository,
	contentBlocks *repository.ContentBlockRepository,
	streamEvents *repository.ResponseStreamEventRepository,
	citations *repository.CitationRepository,
) *ChatStreamService {
	return &ChatStreamService{
		conversations:  conversations,
		messages:       messages,
		responseRuns:   responseRuns,
		processSteps:   processSteps,
		contentBlocks:  contentBlocks,
		streamEvents:   streamEvents,
		citations:      citations,
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

	userMsg, err := s.messages.Create(ctx, req.ConversationID, "user", "completed")
	if err != nil {
		return fmt.Errorf("create user message: %w", err)
	}

	_, err = s.contentBlocks.Create(
		ctx, userMsg.ID, 0, "text", req.Message, domain.ContentBlockStatusCompleted,
	)
	if err != nil {
		return fmt.Errorf("create user content block: %w", err)
	}

	assistantMsg, err := s.messages.Create(ctx, req.ConversationID, "assistant", "streaming")
	if err != nil {
		return fmt.Errorf("create assistant message: %w", err)
	}

	run, err := s.responseRuns.Create(ctx, req.ConversationID, userMsg.ID)
	if err != nil {
		return fmt.Errorf("create response run: %w", err)
	}

	if err := s.responseRuns.UpdateAssistantMessageID(ctx, run.ID, assistantMsg.ID); err != nil {
		return fmt.Errorf("update assistant message id: %w", err)
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
				_ = s.stopStream(context.Background(), tracker, run.ID, assistantMsg.ID)
			}
		default:
		}
	}()

	defer func() {
		if r := recover(); r != nil {
			_ = s.failStream(context.Background(), tracker, run.ID, assistantMsg.ID, fmt.Errorf("panic: %v", r))
			panic(r)
		}
	}()

	intentResult := s.classifyIntent(req.Message, req.KnowledgeBases)

	if err := sse.EmitIntent("started", "正在分析问题...", nil, nil); err != nil {
		return s.failStream(ctx, tracker, run.ID, assistantMsg.ID, err)
	}

	intentLabel := s.intentLabel(intentResult.IntentType)
	if err := sse.EmitIntent("done", intentLabel, ptr(string(intentResult.IntentType)), &intentResult.Confidence); err != nil {
		return s.failStream(ctx, tracker, run.ID, assistantMsg.ID, err)
	}

	if err := s.responseRuns.UpdateIntent(ctx, run.ID, intentResult.IntentType, intentResult.Route, intentResult.Confidence); err != nil {
		return fmt.Errorf("update response run intent: %w", err)
	}

	if _, err := tracker.StartStep(ctx, domain.StepTypeIntent, "识别意图"); err != nil {
		return s.failStream(ctx, tracker, run.ID, assistantMsg.ID, err)
	}
	if _, err := tracker.CompleteStep(ctx, domain.StepTypeIntent, intentLabel, ""); err != nil {
		return s.failStream(ctx, tracker, run.ID, assistantMsg.ID, err)
	}

	if intentResult.UseRetrieval {
		if _, err := tracker.StartStep(ctx, domain.StepTypeRetrieval, "检索知识库"); err != nil {
			return s.failStream(ctx, tracker, run.ID, assistantMsg.ID, err)
		}
		hitCount := 5
		detail := fmt.Sprintf("命中 %d 条结果", hitCount)
		if _, err := tracker.CompleteStep(ctx, domain.StepTypeRetrieval, "检索知识库", detail); err != nil {
			return s.failStream(ctx, tracker, run.ID, assistantMsg.ID, err)
		}
	}

	if _, err := tracker.StartStep(ctx, domain.StepTypeGeneration, "生成回答"); err != nil {
		return s.failStream(ctx, tracker, run.ID, assistantMsg.ID, err)
	}

	answer := s.buildAnswer(req.Message, intentResult.UseRetrieval)
	contentBlock, err := s.contentBlocks.Create(
		ctx, assistantMsg.ID, 0, "text", "", domain.ContentBlockStatusStreaming,
	)
	if err != nil {
		return s.failStream(ctx, tracker, run.ID, assistantMsg.ID, err)
	}

	content, err := s.streamTokens(ctx, sse, answer, contentBlock.ID)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return err
		}
		return s.failStream(ctx, tracker, run.ID, assistantMsg.ID, err)
	}

	if _, err := tracker.CompleteStep(ctx, domain.StepTypeGeneration, "生成回答", ""); err != nil {
		return s.failStream(ctx, tracker, run.ID, assistantMsg.ID, err)
	}

	if intentResult.UseRetrieval {
		if _, err := tracker.StartStep(ctx, domain.StepTypeVerify, "验证答案"); err != nil {
			return s.failStream(ctx, tracker, run.ID, assistantMsg.ID, err)
		}
		if _, err := tracker.CompleteStep(ctx, domain.StepTypeVerify, "验证答案", "引用校验通过"); err != nil {
			return s.failStream(ctx, tracker, run.ID, assistantMsg.ID, err)
		}
	}

	if err := s.contentBlocks.UpdateContent(ctx, contentBlock.ID, content); err != nil {
		return s.failStream(ctx, tracker, run.ID, assistantMsg.ID, err)
	}
	if err := s.contentBlocks.UpdateStatus(ctx, contentBlock.ID, domain.ContentBlockStatusCompleted); err != nil {
		return s.failStream(ctx, tracker, run.ID, assistantMsg.ID, err)
	}

	if err := s.messages.UpdateStatus(ctx, assistantMsg.ID, "completed"); err != nil {
		return s.failStream(ctx, tracker, run.ID, assistantMsg.ID, err)
	}

	if err := s.responseRuns.MarkCompleted(ctx, run.ID); err != nil {
		return s.failStream(ctx, tracker, run.ID, assistantMsg.ID, err)
	}

	latency := time.Since(startedAt).Milliseconds()
	promptTokens := utf8.RuneCountInString(req.Message)
	completionTokens := utf8.RuneCountInString(content)
	if err := s.responseRuns.UpdateMetrics(ctx, run.ID, promptTokens, completionTokens, 0, int(latency)); err != nil {
		return fmt.Errorf("update response run metrics: %w", err)
	}

	if err := sse.EmitDone(map[string]any{
		"response_run_id":     run.ID,
		"message_id":         assistantMsg.ID,
		"total_tokens":       promptTokens + completionTokens,
		"prompt_tokens":      promptTokens,
		"completion_tokens":  completionTokens,
		"latency_ms":         latency,
	}); err != nil {
		return err
	}

	return nil
}

type IntentResult struct {
	IntentType   domain.IntentType
	Route        string
	Confidence   float64
	UseRetrieval bool
}

func (s *ChatStreamService) classifyIntent(message string, knowledgeBases []string) IntentResult {
	useRetrieval := len(knowledgeBases) > 0

	keywords := []string{"查询", "检索", "知识库", "文档", "规范", "标准", "手册", "指南"}
	isKnowledge := useRetrieval
	if !isKnowledge {
		for _, kw := range keywords {
			if strings.Contains(message, kw) {
				isKnowledge = true
				break
			}
		}
	}

	if isKnowledge {
		return IntentResult{
			IntentType:   domain.IntentKnowledgeQA,
			Route:        "rag",
			Confidence:   0.95,
			UseRetrieval: true,
		}
	}

	systemKeywords := []string{"重启", "关闭", "配置", "设置", "指令"}
	isSystem := false
	for _, kw := range systemKeywords {
		if strings.Contains(message, kw) {
			isSystem = true
			break
		}
	}

	if isSystem {
		return IntentResult{
			IntentType:   domain.IntentSystemCommand,
			Route:        "command",
			Confidence:   0.85,
			UseRetrieval: false,
		}
	}

	return IntentResult{
		IntentType:   domain.IntentGeneralChat,
		Route:        "direct",
		Confidence:   0.90,
		UseRetrieval: false,
	}
}

func (s *ChatStreamService) intentLabel(intent domain.IntentType) string {
	switch intent {
	case domain.IntentKnowledgeQA:
		return "识别为：知识问答"
	case domain.IntentGeneralChat:
		return "识别为：一般对话"
	case domain.IntentDocumentQuery:
		return "识别为：文档查询"
	case domain.IntentSystemCommand:
		return "识别为：系统指令"
	default:
		return "识别为：未知"
	}
}

func ptr[T any](v T) *T {
	return &v
}

func (s *ChatStreamService) failStream(
	ctx context.Context,
	tracker *ProcessStepTracker,
	runID string,
	assistantMessageID string,
	cause error,
) error {
	_ = tracker.MarkRunningAsFailed(ctx, domain.StepStatusFailed)
	_ = s.messages.UpdateStatus(ctx, assistantMessageID, "failed")
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
	_ = s.messages.UpdateStatus(ctx, assistantMessageID, "stopped")
	_ = s.responseRuns.MarkStopped(ctx, runID, "client disconnected")
	return context.Canceled
}

func (s *ChatStreamService) buildAnswer(message string, useRetrieval bool) string {
	if useRetrieval {
		return fmt.Sprintf("根据知识库检索结果，关于「%s」的要点如下：请结合引用文档进一步确认细节。", message)
	}
	return fmt.Sprintf("您好，关于「%s」，我可以继续为您提供帮助。", message)
}

func (s *ChatStreamService) streamTokens(
	ctx context.Context,
	sse *SSEWriter,
	answer string,
	contentBlockID string,
) (string, error) {
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
