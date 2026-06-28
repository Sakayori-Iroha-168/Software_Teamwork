package service

import (
	"context"
	"fmt"
	"time"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/qa/internal/repository"
)

type CreateConversationRequest struct {
	ExternalUserID string
}

type ListConversationsRequest struct {
	ExternalUserID string
	Page           int
	PageSize       int
}

type ConversationResult struct {
	ID             string
	ExternalUserID string
	Title          string
	Status         string
	CreatedAt      string
	UpdatedAt      string
	Messages       []map[string]any
}

type ListConversationsResult struct {
	Data     []ConversationResult
	Page     int
	PageSize int
	Total    int64
}

type ConversationService struct {
	conversations *repository.ConversationRepository
	messages      *repository.MessageRepository
	responseRuns  *repository.ResponseRunRepository
	processSteps  *repository.ProcessStepRepository
}

func NewConversationService(
	conversations *repository.ConversationRepository,
	messages *repository.MessageRepository,
	responseRuns *repository.ResponseRunRepository,
	processSteps *repository.ProcessStepRepository,
) *ConversationService {
	return &ConversationService{
		conversations: conversations,
		messages:      messages,
		responseRuns:  responseRuns,
		processSteps:  processSteps,
	}
}

func (s *ConversationService) Create(ctx context.Context, req CreateConversationRequest) (ConversationResult, error) {
	conv, err := s.conversations.Create(ctx, req.ExternalUserID)
	if err != nil {
		return ConversationResult{}, fmt.Errorf("create conversation: %w", err)
	}

	return ConversationResult{
		ID:             conv.ID,
		ExternalUserID: conv.ExternalUserID,
		Title:          conv.Title,
		Status:         conv.Status,
		CreatedAt:      conv.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:      conv.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}, nil
}

func (s *ConversationService) GetByID(ctx context.Context, id string) (ConversationResult, error) {
	conv, err := s.conversations.GetByID(ctx, id)
	if err != nil {
		return ConversationResult{}, fmt.Errorf("get conversation: %w", err)
	}

	msgsWithContent, err := s.messages.ListByConversationIDWithContent(ctx, id)
	if err != nil {
		return ConversationResult{}, fmt.Errorf("get messages: %w", err)
	}

	assistantMsgToThinking := make(map[string][]map[string]any)
	for _, msg := range msgsWithContent {
		if msg.Role == "assistant" {
			run, err := s.responseRuns.GetByAssistantMessageID(ctx, msg.ID)
			if err == nil {
				steps, err := s.processSteps.ListByResponseRunID(ctx, run.ID)
				if err == nil {
					thinking := make([]map[string]any, 0, len(steps))
					for _, step := range steps {
						thinking = append(thinking, map[string]any{
							"type":   step.Type,
							"label":  step.Label,
							"status": step.Status,
							"detail": step.Detail,
						})
					}
					assistantMsgToThinking[msg.ID] = thinking
				}
			}
		}
	}

	messages := make([]map[string]any, 0, len(msgsWithContent))
	for _, msg := range msgsWithContent {
		messageData := map[string]any{
			"id":            msg.ID,
			"role":          msg.Role,
			"sequence_no":   msg.SequenceNo,
			"content":       msg.Content,
			"status":        msg.Status,
			"model_name":    msg.ModelName,
			"error_code":    msg.ErrorCode,
			"error_message": msg.ErrorMessage,
			"created_at":    msg.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			"completed_at":  formatTimePtr(msg.CompletedAt),
		}
		if thinking, ok := assistantMsgToThinking[msg.ID]; ok {
			messageData["thinking"] = thinking
		}
		messages = append(messages, messageData)
	}

	return ConversationResult{
		ID:             conv.ID,
		ExternalUserID: conv.ExternalUserID,
		Title:          conv.Title,
		Status:         conv.Status,
		CreatedAt:      conv.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:      conv.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		Messages:       messages,
	}, nil
}

func (s *ConversationService) List(ctx context.Context, req ListConversationsRequest) (ListConversationsResult, error) {
	convs, total, err := s.conversations.List(ctx, req.ExternalUserID, req.Page, req.PageSize)
	if err != nil {
		return ListConversationsResult{}, fmt.Errorf("list conversations: %w", err)
	}

	data := make([]ConversationResult, 0, len(convs))
	for _, conv := range convs {
		data = append(data, ConversationResult{
			ID:             conv.ID,
			ExternalUserID: conv.ExternalUserID,
			Title:          conv.Title,
			Status:         conv.Status,
			CreatedAt:      conv.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt:      conv.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}

	return ListConversationsResult{
		Data:     data,
		Page:     req.Page,
		PageSize: req.PageSize,
		Total:    total,
	}, nil
}

func (s *ConversationService) Delete(ctx context.Context, id string) error {
	return s.conversations.SoftDelete(ctx, id)
}

func (s *ConversationService) UpdateStatus(ctx context.Context, id string, status string) error {
	return s.conversations.UpdateStatus(ctx, id, status)
}

func formatTimePtr(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format("2006-01-02T15:04:05Z07:00")
}
