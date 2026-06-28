package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Conversation struct {
	ID             string
	ExternalUserID string
	Title          string
	Status         string
	CreatedAt      time.Time
	UpdatedAt      time.Time
	DeletedAt      *time.Time
}

type ConversationRepository struct {
	db *pgxpool.Pool
}

func NewConversationRepository(db *pgxpool.Pool) *ConversationRepository {
	return &ConversationRepository{db: db}
}

func (r *ConversationRepository) Create(ctx context.Context, externalUserID string) (Conversation, error) {
	id := uuid.NewString()
	now := nowUTC()
	const query = `
		INSERT INTO conversations (id, external_user_id, status, created_at, updated_at)
		VALUES ($1, $2, 'active', $3, $3)
		RETURNING id, external_user_id, title, status, created_at, updated_at
	`
	var conv Conversation
	var title *string
	err := r.db.QueryRow(ctx, query, id, externalUserID, now).Scan(
		&conv.ID, &conv.ExternalUserID, &title, &conv.Status, &conv.CreatedAt, &conv.UpdatedAt,
	)
	if err != nil {
		return Conversation{}, fmt.Errorf("create conversation: %w", err)
	}
	if title != nil {
		conv.Title = *title
	}
	return conv, nil
}

func (r *ConversationRepository) GetByID(ctx context.Context, id string) (Conversation, error) {
	const query = `
		SELECT id, external_user_id, title, status, created_at, updated_at
		FROM conversations
		WHERE id = $1 AND deleted_at IS NULL
	`
	var conv Conversation
	var title *string
	err := r.db.QueryRow(ctx, query, id).Scan(
		&conv.ID, &conv.ExternalUserID, &title, &conv.Status, &conv.CreatedAt, &conv.UpdatedAt,
	)
	if err != nil {
		return Conversation{}, fmt.Errorf("get conversation: %w", err)
	}
	if title != nil {
		conv.Title = *title
	}
	return conv, nil
}

func (r *ConversationRepository) List(
	ctx context.Context,
	externalUserID string,
	page int,
	pageSize int,
) ([]Conversation, int64, error) {
	offset := (page - 1) * pageSize

	countQuery := `
		SELECT COUNT(*) FROM conversations
		WHERE external_user_id = $1 AND deleted_at IS NULL
	`
	var total int64
	err := r.db.QueryRow(ctx, countQuery, externalUserID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count conversations: %w", err)
	}

	query := `
		SELECT id, external_user_id, title, status, created_at, updated_at
		FROM conversations
		WHERE external_user_id = $1 AND deleted_at IS NULL
		ORDER BY updated_at DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := r.db.Query(ctx, query, externalUserID, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list conversations: %w", err)
	}
	defer rows.Close()

	var convs []Conversation
	for rows.Next() {
		var conv Conversation
		var title *string
		if err := rows.Scan(
			&conv.ID, &conv.ExternalUserID, &title, &conv.Status, &conv.CreatedAt, &conv.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan conversation: %w", err)
		}
		if title != nil {
			conv.Title = *title
		}
		convs = append(convs, conv)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate conversations: %w", err)
	}
	return convs, total, nil
}

func (r *ConversationRepository) UpdateTitle(ctx context.Context, id string, title string) error {
	now := nowUTC()
	const query = `
		UPDATE conversations SET title = $2, updated_at = $3 WHERE id = $1
	`
	_, err := r.db.Exec(ctx, query, id, title, now)
	if err != nil {
		return fmt.Errorf("update conversation title: %w", err)
	}
	return nil
}

func (r *ConversationRepository) UpdateStatus(ctx context.Context, id string, status string) error {
	now := nowUTC()
	const query = `
		UPDATE conversations SET status = $2, updated_at = $3 WHERE id = $1
	`
	_, err := r.db.Exec(ctx, query, id, status, now)
	if err != nil {
		return fmt.Errorf("update conversation status: %w", err)
	}
	return nil
}

func (r *ConversationRepository) SoftDelete(ctx context.Context, id string) error {
	now := nowUTC()
	const query = `
		UPDATE conversations SET deleted_at = $2, updated_at = $2 WHERE id = $1
	`
	_, err := r.db.Exec(ctx, query, id, now)
	if err != nil {
		return fmt.Errorf("soft delete conversation: %w", err)
	}
	return nil
}

func (r *ConversationRepository) TouchUpdatedAt(ctx context.Context, id string) error {
	const query = `UPDATE conversations SET updated_at = $2 WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id, nowUTC())
	if err != nil {
		return fmt.Errorf("touch conversation updated_at: %w", err)
	}
	return nil
}
