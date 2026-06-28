package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Message struct {
	ID             string
	ConversationID string
	Role           string
	Content        string
	Status         string
	CreatedAt      time.Time
}

type MessageRepository struct {
	db *pgxpool.Pool
}

func NewMessageRepository(db *pgxpool.Pool) *MessageRepository {
	return &MessageRepository{db: db}
}

func (r *MessageRepository) Create(ctx context.Context, conversationID, role, content, status string) (Message, error) {
	id := "msg_" + uuid.NewString()[:8]
	now := nowUTC()
	const query = `
		INSERT INTO messages (id, conversation_id, role, content, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, conversation_id, role, content, status, created_at
	`
	var msg Message
	err := r.db.QueryRow(ctx, query, id, conversationID, role, content, status, now).Scan(
		&msg.ID, &msg.ConversationID, &msg.Role, &msg.Content, &msg.Status, &msg.CreatedAt,
	)
	if err != nil {
		return Message{}, fmt.Errorf("create message: %w", err)
	}
	return msg, nil
}

func (r *MessageRepository) UpdateContentAndStatus(ctx context.Context, id, content, status string) error {
	const query = `
		UPDATE messages
		SET content = $2, status = $3
		WHERE id = $1
	`
	_, err := r.db.Exec(ctx, query, id, content, status)
	if err != nil {
		return fmt.Errorf("update message content: %w", err)
	}
	return nil
}

func (r *MessageRepository) ListByConversation(ctx context.Context, conversationID string) ([]Message, error) {
	const query = `
		SELECT id, conversation_id, role, content, status, created_at
		FROM messages
		WHERE conversation_id = $1
		ORDER BY created_at ASC
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
			&msg.ID, &msg.ConversationID, &msg.Role, &msg.Content, &msg.Status, &msg.CreatedAt,
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
