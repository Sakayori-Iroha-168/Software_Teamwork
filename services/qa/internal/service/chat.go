package service

import (
	"context"
	"strings"
	"time"

	"software-teamwork/services/qa/internal/repository"
)

type Store interface {
	CreateRun(ctx context.Context, run repository.ResponseRun) error
	AppendProcessStep(ctx context.Context, step repository.ResponseProcessStep) error
	AppendStreamEvent(ctx context.Context, event repository.ResponseStreamEvent) error
	AppendContentBlock(ctx context.Context, block repository.MessageContentBlock) error
	AppendCitation(ctx context.Context, citation repository.Citation) error
	CompleteRun(ctx context.Context, runID string, status string, stopReason string) error
}

type ChatService struct {
	store Store
}

func NewChatService(store Store) *ChatService {
	return &ChatService{store: store}
}

func (s *ChatService) Stream(ctx context.Context, req ChatRequest) (<-chan StreamEvent, error) {
	intent, confidence := classifyIntent(req.Message)
	route := routeForIntent(intent)
	runID := newID("run")
	userMessageID := newID("msg_user")
	assistantMessageID := newID("msg_assistant")
	startedAt := time.Now().UTC()

	run := repository.ResponseRun{
		ID:                 runID,
		ConversationID:     req.ConversationID,
		UserMessageID:      userMessageID,
		AssistantMessageID: assistantMessageID,
		RequestID:          req.TraceID,
		IntentType:         string(intent),
		Confidence:         confidence,
		Route:              string(route),
		Status:             "running",
		StartedAt:          startedAt,
	}
	if err := s.store.CreateRun(ctx, run); err != nil {
		return nil, err
	}

	out := make(chan StreamEvent)
	go func() {
		defer close(out)

		s.emit(ctx, out, runID, "intent_status", map[string]any{
			"status":     "started",
			"label":      "正在分析问题...",
			"request_id": req.TraceID,
		})
		s.emit(ctx, out, runID, "intent_status", map[string]any{
			"status":     "done",
			"label":      intentLabel(intent),
			"intent":     string(intent),
			"confidence": confidence,
			"route":      string(route),
		})

		stepType := "generation"
		stepLabel := "生成回答"
		if intent == IntentKnowledgeQA {
			stepType = "retrieval"
			stepLabel = "检索知识库"
		}
		_ = s.store.AppendProcessStep(ctx, repository.ResponseProcessStep{
			ID:            newID("step"),
			ResponseRunID: runID,
			StepOrder:     1,
			StepType:      stepType,
			Label:         stepLabel,
			Status:        "completed",
			StartedAt:     time.Now().UTC(),
			FinishedAt:    time.Now().UTC(),
		})
		s.emit(ctx, out, runID, "thinking_step", map[string]any{
			"step": map[string]any{
				"type":   stepType,
				"label":  stepLabel,
				"status": "done",
				"detail": routeDetail(intent),
			},
		})

		answer := answerFor(req.Message, intent)
		tokens := tokenize(answer)
		for index, token := range tokens {
			select {
			case <-ctx.Done():
				_ = s.store.CompleteRun(context.Background(), runID, "stopped", "client_disconnected")
				return
			default:
			}

			_ = s.store.AppendContentBlock(ctx, repository.MessageContentBlock{
				ID:         newID("block"),
				MessageID:  assistantMessageID,
				BlockOrder: index,
				BlockType:  "text",
				Content:    token,
				Status:     "streaming",
				CreatedAt:  time.Now().UTC(),
			})
			s.emit(ctx, out, runID, "token", map[string]any{
				"text":  token,
				"index": index,
			})
			time.Sleep(30 * time.Millisecond)
		}

		if intent == IntentKnowledgeQA {
			citation := repository.Citation{
				ID:              newID("cite"),
				MessageID:       assistantMessageID,
				CitationNo:      1,
				ExternalKBID:    firstOrDefault(req.KnowledgeBases, "default-kb"),
				ExternalDocID:   "DOC-MOCK-001",
				ExternalChunkID: "chunk_mock_001",
				DocName:         "mock-knowledge-source.md",
				QuoteText:       "知识库能力尚未接入时返回 mock 引用，后续由 knowledge 服务替换。",
				Score:           0.8,
				CreatedAt:       time.Now().UTC(),
			}
			_ = s.store.AppendCitation(ctx, citation)
			s.emit(ctx, out, runID, "citation", map[string]any{
				"citation": map[string]any{
					"id":       "1",
					"doc_id":   citation.ExternalDocID,
					"doc_name": citation.DocName,
					"chunk_id": citation.ExternalChunkID,
					"text":     citation.QuoteText,
					"score":    citation.Score,
				},
			})
		}

		_ = s.store.CompleteRun(ctx, runID, "completed", "done")
		s.emit(ctx, out, runID, "done", map[string]any{
			"message_id":        assistantMessageID,
			"total_tokens":      len(tokens),
			"prompt_tokens":     len(tokenize(req.Message)),
			"completion_tokens": len(tokens),
			"latency_ms":        time.Since(startedAt).Milliseconds(),
		})
	}()

	return out, nil
}

func (s *ChatService) emit(ctx context.Context, out chan<- StreamEvent, runID string, event string, data any) {
	_ = s.store.AppendStreamEvent(ctx, repository.ResponseStreamEvent{
		ID:            newID("evt"),
		ResponseRunID: runID,
		EventSeq:      int(time.Now().UTC().UnixNano()),
		EventType:     dbEventType(event),
		Payload:       data,
		CreatedAt:     time.Now().UTC(),
	})
	out <- StreamEvent{Event: event, Data: data}
}

func routeForIntent(intent IntentType) Route {
	if intent == IntentKnowledgeQA {
		return RouteKnowledge
	}
	return RouteGeneral
}

func intentLabel(intent IntentType) string {
	if intent == IntentKnowledgeQA {
		return "识别为：知识问答"
	}
	return "识别为：一般对话"
}

func routeDetail(intent IntentType) string {
	if intent == IntentKnowledgeQA {
		return "当前使用 mock 检索结果，等待 knowledge/RAG 服务接入。"
	}
	return "一般对话不触发知识库检索。"
}

func dbEventType(event string) string {
	switch event {
	case "intent_status":
		return "intent"
	case "thinking_step":
		return "step"
	default:
		return event
	}
}

func answerFor(message string, intent IntentType) string {
	if intent == IntentKnowledgeQA {
		return "已识别为知识库问答。当前先返回稳定的 mock 流式结果，后续接入 knowledge 服务后会基于检索片段生成答案。"
	}
	return "已识别为一般对话。当前 QA 服务已经通过统一 SSE 接口返回回答，并记录运行过程。你刚才的问题是：" + message
}

func tokenize(answer string) []string {
	parts := strings.Fields(answer)
	if len(parts) == 0 {
		return []string{answer}
	}
	return parts
}

func firstOrDefault(values []string, fallback string) string {
	if len(values) == 0 || values[0] == "" {
		return fallback
	}
	return values[0]
}
