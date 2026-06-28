package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/qa/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ProcessStepRepository struct {
	db *pgxpool.Pool
}

func NewProcessStepRepository(db *pgxpool.Pool) *ProcessStepRepository {
	return &ProcessStepRepository{db: db}
}

func (r *ProcessStepRepository) UpsertStep(
	ctx context.Context,
	responseRunID string,
	step domain.ProcessStepRecord,
) (domain.ThinkingStep, error) {
	const query = `
		INSERT INTO response_process_steps (
			response_run_id, step_order, step_type, label, detail, status, started_at, finished_at
		)
		VALUES ($1, $2, $3, $4, NULLIF($5, ''), $6, $7, $8)
		ON CONFLICT (response_run_id, step_order) DO UPDATE SET
			step_type = EXCLUDED.step_type,
			label = EXCLUDED.label,
			detail = COALESCE(NULLIF(EXCLUDED.detail, ''), response_process_steps.detail),
			status = EXCLUDED.status,
			finished_at = EXCLUDED.finished_at
		RETURNING step_type, label, COALESCE(detail, ''), status
	`

	var (
		stepType string
		label    string
		detail   string
		status   string
	)
	err := r.db.QueryRow(
		ctx,
		query,
		responseRunID,
		step.StepOrder,
		string(step.StepType),
		step.Label,
		step.Detail,
		string(step.Status),
		step.StartedAt,
		step.FinishedAt,
	).Scan(&stepType, &label, &detail, &status)
	if err != nil {
		return domain.ThinkingStep{}, fmt.Errorf("upsert process step: %w", err)
	}

	result := domain.ThinkingStep{
		Type:   domain.StepType(stepType),
		Label:  label,
		Status: domain.StepStatus(status),
	}
	if detail != "" {
		result.Detail = detail
	}
	return result, nil
}

func (r *ProcessStepRepository) ListByResponseRunID(
	ctx context.Context,
	responseRunID string,
) ([]domain.ThinkingStep, error) {
	const query = `
		SELECT step_type, label, COALESCE(detail, ''), status
		FROM response_process_steps
		WHERE response_run_id = $1
		ORDER BY step_order ASC
	`
	rows, err := r.db.Query(ctx, query, responseRunID)
	if err != nil {
		return nil, fmt.Errorf("list process steps: %w", err)
	}
	defer rows.Close()

	var steps []domain.ThinkingStep
	for rows.Next() {
		var (
			stepType string
			label    string
			detail   string
			status   string
		)
		if err := rows.Scan(&stepType, &label, &detail, &status); err != nil {
			return nil, fmt.Errorf("scan process step: %w", err)
		}
		step := domain.ThinkingStep{
			Type:   domain.StepType(stepType),
			Label:  label,
			Status: domain.StepStatus(status),
		}
		if detail != "" {
			step.Detail = detail
		}
		steps = append(steps, step)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate process steps: %w", err)
	}
	return steps, nil
}

func (r *ProcessStepRepository) MarkRunningAsFailed(
	ctx context.Context,
	responseRunID string,
	finalStatus domain.StepStatus,
) error {
	now := nowUTC()
	const query = `
		UPDATE response_process_steps
		SET status = $2, finished_at = $3
		WHERE response_run_id = $1 AND status = 'running'
	`
	_, err := r.db.Exec(ctx, query, responseRunID, string(finalStatus), now)
	if err != nil {
		return fmt.Errorf("mark running steps failed: %w", err)
	}
	return nil
}

func (r *ProcessStepRepository) GetStartedAt(
	ctx context.Context,
	responseRunID string,
	stepType domain.StepType,
) (time.Time, bool, error) {
	const query = `
		SELECT started_at
		FROM response_process_steps
		WHERE response_run_id = $1 AND step_type = $2
	`
	var startedAt time.Time
	err := r.db.QueryRow(ctx, query, responseRunID, string(stepType)).Scan(&startedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return time.Time{}, false, nil
	}
	if err != nil {
		return time.Time{}, false, fmt.Errorf("get step started_at: %w", err)
	}
	return startedAt, true, nil
}
