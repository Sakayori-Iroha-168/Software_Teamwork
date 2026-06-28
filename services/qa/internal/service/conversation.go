package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/qa/internal/domain"
	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/qa/internal/repository"
	"github.com/jackc/pgx/v5"
)

type ConversationDetail struct {
	ID        string                `json:"id"`
	Title     string                `json:"title"`
	Messages  []ConversationMessage `json:"messages"`
	CreatedAt string                `json:"created_at"`
	UpdatedAt string                `json:"updated_at"`
}

type ConversationMessage struct {
	ID        string               `json:"id"`
	Role      string               `json:"role"`
	Content   string               `json:"content"`
	Status    string               `json:"status"`
	Timestamp string               `json:"timestamp"`
	Thinking  []domain.ThinkingStep `json:"thinking,omitempty"`
}

type CreateConversationResult struct {
	ID        string
	Title     string
	CreatedAt string
	UpdatedAt string
}

type ConversationService struct {
	conversations *repository.ConversationRepository
	messages      *repository.MessageRepository
	responseRuns  *repository.ResponseRunRepository
	processSteps  *repository.ProcessStepRepository
	contentBlocks *repository.ContentBlockRepository
}

func NewConversationService(
	conversations *repository.ConversationRepository,
	messages *repository.MessageRepository,
	responseRuns *repository.ResponseRunRepository,
	processSteps *repository.ProcessStepRepository,
	contentBlocks *repository.ContentBlockRepository,
) *ConversationService {
	return &ConversationService{
		conversations: conversations,
		messages:      messages,
		responseRuns:  responseRuns,
		processSteps:  processSteps,
		contentBlocks: contentBlocks,
	}
}

func (s *ConversationService) Create(ctx context.Context, title string) (CreateConversationResult, error) {
	conv, err := s.conversations.Create(ctx, title)
	if err != nil {
		return CreateConversationResult{}, fmt.Errorf("create conversation: %w", err)
	}
	return CreateConversationResult{
		ID:        conv.ID,
		Title:     conv.Title,
		CreatedAt: conv.CreatedAt.UTC().Format(timeRFC3339),
		UpdatedAt: conv.UpdatedAt.UTC().Format(timeRFC3339),
	}, nil
}

func (s *ConversationService) GetDetail(ctx context.Context, conversationID string) (ConversationDetail, error) {
	conv, err := s.conversations.GetByID(ctx, conversationID)
	if err != nil {
		return ConversationDetail{}, fmt.Errorf("get conversation: %w", err)
	}

	msgs, err := s.messages.ListByConversation(ctx, conversationID)
	if err != nil {
		return ConversationDetail{}, fmt.Errorf("list messages: %w", err)
	}

	detail := ConversationDetail{
		ID:        conv.ID,
		Title:     conv.Title,
		CreatedAt: conv.CreatedAt.UTC().Format(timeRFC3339),
		UpdatedAt: conv.UpdatedAt.UTC().Format(timeRFC3339),
	}

	for _, msg := range msgs {
		item := ConversationMessage{
			ID:        msg.ID,
			Role:      msg.Role,
			Content:   msg.Content,
			Status:    msg.Status,
			Timestamp: msg.CreatedAt.UTC().Format(timeRFC3339),
		}
		if msg.Role == "assistant" {
			thinking, err := s.loadThinking(ctx, msg.ID)
			if err != nil {
				return ConversationDetail{}, err
			}
			if len(thinking) > 0 {
				item.Thinking = thinking
			}
		}
		detail.Messages = append(detail.Messages, item)
	}

	return detail, nil
}

func (s *ConversationService) loadThinking(ctx context.Context, messageID string) ([]domain.ThinkingStep, error) {
	run, err := s.responseRuns.GetByMessageID(ctx, messageID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("load response run: %w", err)
	}
	return s.processSteps.ListByResponseRunID(ctx, run.ID)
}

const timeRFC3339 = "2006-01-02T15:04:05Z"
