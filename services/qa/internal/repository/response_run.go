package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ResponseRun struct {
	ID             string
	MessageID      string
	ConversationID string
	Status         string
	StopReason     *string
	CreatedAt      time.Time
	UpdatedAt      time.Time
	FinishedAt     *time.Time
}

type ResponseRunRepository struct {
	db *pgxpool.Pool
}

func NewResponseRunRepository(db *pgxpool.Pool) *ResponseRunRepository {
	return &ResponseRunRepository{db: db}
}

func (r *ResponseRunRepository) Create(ctx context.Context, messageID, conversationID string) (ResponseRun, error) {
	id := "run_" + uuid.NewString()[:8]
	now := nowUTC()
	const query = `
		INSERT INTO response_runs (id, message_id, conversation_id, status, created_at, updated_at)
		VALUES ($1, $2, $3, 'running', $4, $4)
		RETURNING id, message_id, conversation_id, status, stop_reason, created_at, updated_at, finished_at
	`
	var run ResponseRun
	err := r.db.QueryRow(ctx, query, id, messageID, conversationID, now).Scan(
		&run.ID, &run.MessageID, &run.ConversationID, &run.Status,
		&run.StopReason, &run.CreatedAt, &run.UpdatedAt, &run.FinishedAt,
	)
	if err != nil {
		return ResponseRun{}, fmt.Errorf("create response run: %w", err)
	}
	return run, nil
}

func (r *ResponseRunRepository) GetByMessageID(ctx context.Context, messageID string) (ResponseRun, error) {
	const query = `
		SELECT id, message_id, conversation_id, status, stop_reason, created_at, updated_at, finished_at
		FROM response_runs
		WHERE message_id = $1
	`
	var run ResponseRun
	err := r.db.QueryRow(ctx, query, messageID).Scan(
		&run.ID, &run.MessageID, &run.ConversationID, &run.Status,
		&run.StopReason, &run.CreatedAt, &run.UpdatedAt, &run.FinishedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return ResponseRun{}, pgx.ErrNoRows
	}
	if err != nil {
		return ResponseRun{}, fmt.Errorf("get response run by message: %w", err)
	}
	return run, nil
}

func (r *ResponseRunRepository) MarkCompleted(ctx context.Context, runID string) error {
	now := nowUTC()
	const query = `
		UPDATE response_runs
		SET status = 'completed', finished_at = $2, updated_at = $2
		WHERE id = $1
	`
	_, err := r.db.Exec(ctx, query, runID, now)
	if err != nil {
		return fmt.Errorf("mark response run completed: %w", err)
	}
	return nil
}

func (r *ResponseRunRepository) MarkFailed(ctx context.Context, runID string, stopReason string) error {
	now := nowUTC()
	const query = `
		UPDATE response_runs
		SET status = 'failed', stop_reason = $2, finished_at = $3, updated_at = $3
		WHERE id = $1 AND status = 'running'
	`
	_, err := r.db.Exec(ctx, query, runID, stopReason, now)
	if err != nil {
		return fmt.Errorf("mark response run failed: %w", err)
	}
	return nil
}

func (r *ResponseRunRepository) MarkStopped(ctx context.Context, runID string, stopReason string) error {
	now := nowUTC()
	const query = `
		UPDATE response_runs
		SET status = 'stopped', stop_reason = $2, finished_at = $3, updated_at = $3
		WHERE id = $1 AND status = 'running'
	`
	_, err := r.db.Exec(ctx, query, runID, stopReason, now)
	if err != nil {
		return fmt.Errorf("mark response run stopped: %w", err)
	}
	return nil
}
