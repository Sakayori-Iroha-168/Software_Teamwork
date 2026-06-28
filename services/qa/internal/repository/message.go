package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Message struct {
	ID             string
	ConversationID string
	Role           string
	SequenceNo     int
	Status         string
	ModelName      string
	ErrorCode      string
	ErrorMessage   string
	CreatedAt      time.Time
	CompletedAt    *time.Time
}

type MessageRepository struct {
	db *pgxpool.Pool
}

func NewMessageRepository(db *pgxpool.Pool) *MessageRepository {
	return &MessageRepository{db: db}
}

func (r *MessageRepository) Create(ctx context.Context, conversationID, role, status string) (Message, error) {
	id := uuid.NewString()
	now := nowUTC()

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return Message{}, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var seqNo int
	seqQuery := `
		SELECT COALESCE(MAX(sequence_no), 0) + 1
		FROM messages
		WHERE conversation_id = $1
		FOR UPDATE
	`
	if err := tx.QueryRow(ctx, seqQuery, conversationID).Scan(&seqNo); err != nil {
		return Message{}, fmt.Errorf("get next sequence no: %w", err)
	}

	insertQuery := `
		INSERT INTO messages (id, conversation_id, role, sequence_no, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, conversation_id, role, sequence_no, status, created_at
	`
	var msg Message
	err = tx.QueryRow(ctx, insertQuery, id, conversationID, role, seqNo, status, now).Scan(
		&msg.ID, &msg.ConversationID, &msg.Role, &msg.SequenceNo, &msg.Status, &msg.CreatedAt,
	)
	if err != nil {
		return Message{}, fmt.Errorf("create message: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return Message{}, fmt.Errorf("commit tx: %w", err)
	}
	return msg, nil
}

func (r *MessageRepository) UpdateStatus(ctx context.Context, id, status string) error {
	now := nowUTC()
	var completedAt *time.Time
	if status == "completed" || status == "failed" || status == "stopped" {
		completedAt = &now
	}

	query := `
		UPDATE messages
		SET status = $2, completed_at = $3
		WHERE id = $1
	`
	_, err := r.db.Exec(ctx, query, id, status, completedAt)
	if err != nil {
		return fmt.Errorf("update message status: %w", err)
	}
	return nil
}

func (r *MessageRepository) UpdateModelName(ctx context.Context, id, modelName string) error {
	const query = `
		UPDATE messages SET model_name = $2 WHERE id = $1
	`
	_, err := r.db.Exec(ctx, query, id, modelName)
	if err != nil {
		return fmt.Errorf("update message model name: %w", err)
	}
	return nil
}

func (r *MessageRepository) UpdateError(ctx context.Context, id, errorCode, errorMessage string) error {
	const query = `
		UPDATE messages
		SET error_code = $2, error_message = $3
		WHERE id = $1
	`
	_, err := r.db.Exec(ctx, query, id, errorCode, errorMessage)
	if err != nil {
		return fmt.Errorf("update message error: %w", err)
	}
	return nil
}

func (r *MessageRepository) GetContent(ctx context.Context, id string) (string, error) {
	const query = `
		SELECT COALESCE(STRING_AGG(content, '' ORDER BY block_order), '')
		FROM message_content_blocks
		WHERE message_id = $1 AND status = 'completed' AND block_type = 'text'
	`
	var content string
	err := r.db.QueryRow(ctx, query, id).Scan(&content)
	if err != nil {
		return "", fmt.Errorf("get message content: %w", err)
	}
	return content, nil
}

func (r *MessageRepository) ListByConversationID(ctx context.Context, conversationID string) ([]Message, error) {
	const query = `
		SELECT id, conversation_id, role, sequence_no, status,
			model_name, error_code, error_message, created_at, completed_at
		FROM messages
		WHERE conversation_id = $1
		ORDER BY sequence_no ASC
	`
	rows, err := r.db.Query(ctx, query, conversationID)
	if err != nil {
		return nil, fmt.Errorf("list messages: %w", err)
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var msg Message
		if err := rows.Scan(
			&msg.ID, &msg.ConversationID, &msg.Role, &msg.SequenceNo,
			&msg.Status, &msg.ModelName, &msg.ErrorCode, &msg.ErrorMessage,
			&msg.CreatedAt, &msg.CompletedAt,
		); err != nil {
			return nil, fmt.Errorf("scan message: %w", err)
		}
		messages = append(messages, msg)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate messages: %w", err)
	}
	return messages, nil
}

func (r *MessageRepository) ListByConversationIDWithContent(ctx context.Context, conversationID string) ([]struct {
	Message
	Content string
}, error) {
	messages, err := r.ListByConversationID(ctx, conversationID)
	if err != nil {
		return nil, err
	}

	result := make([]struct {
		Message
		Content string
	}, len(messages))

	for i, msg := range messages {
		result[i].Message = msg
		if strings.ToLower(msg.Role) == "user" || strings.ToLower(msg.Role) == "assistant" {
			content, err := r.GetContent(ctx, msg.ID)
			if err != nil {
				return nil, fmt.Errorf("get content for message %s: %w", msg.ID, err)
			}
			result[i].Content = content
		}
	}
	return result, nil
}

func (r *MessageRepository) GetByID(ctx context.Context, id string) (Message, error) {
	const query = `
		SELECT id, conversation_id, role, sequence_no, status,
			model_name, error_code, error_message, created_at, completed_at
		FROM messages
		WHERE id = $1
	`
	var msg Message
	err := r.db.QueryRow(ctx, query, id).Scan(
		&msg.ID, &msg.ConversationID, &msg.Role, &msg.SequenceNo,
		&msg.Status, &msg.ModelName, &msg.ErrorCode, &msg.ErrorMessage,
		&msg.CreatedAt, &msg.CompletedAt,
	)
	if err != nil {
		return Message{}, fmt.Errorf("get message by id: %w", err)
	}
	return msg, nil
}
