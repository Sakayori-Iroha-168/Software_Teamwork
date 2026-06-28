package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Conversation struct {
	ID        string
	Title     string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type ConversationRepository struct {
	db *pgxpool.Pool
}

func NewConversationRepository(db *pgxpool.Pool) *ConversationRepository {
	return &ConversationRepository{db: db}
}

func (r *ConversationRepository) Create(ctx context.Context, title string) (Conversation, error) {
	id := "conv_" + uuid.NewString()[:8]
	now := nowUTC()
	const query = `
		INSERT INTO conversations (id, title, created_at, updated_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id, title, created_at, updated_at
	`
	var conv Conversation
	err := r.db.QueryRow(ctx, query, id, title, now, now).Scan(
		&conv.ID, &conv.Title, &conv.CreatedAt, &conv.UpdatedAt,
	)
	if err != nil {
		return Conversation{}, fmt.Errorf("create conversation: %w", err)
	}
	return conv, nil
}

func (r *ConversationRepository) GetByID(ctx context.Context, id string) (Conversation, error) {
	const query = `
		SELECT id, title, created_at, updated_at
		FROM conversations
		WHERE id = $1
	`
	var conv Conversation
	err := r.db.QueryRow(ctx, query, id).Scan(
		&conv.ID, &conv.Title, &conv.CreatedAt, &conv.UpdatedAt,
	)
	if err != nil {
		return Conversation{}, fmt.Errorf("get conversation: %w", err)
	}
	return conv, nil
}

func (r *ConversationRepository) TouchUpdatedAt(ctx context.Context, id string) error {
	const query = `UPDATE conversations SET updated_at = $2 WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id, nowUTC())
	if err != nil {
		return fmt.Errorf("touch conversation updated_at: %w", err)
	}
	return nil
}
