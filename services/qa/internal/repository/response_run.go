package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/qa/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ResponseRun struct {
	ID                 string
	ConversationID     string
	UserMessageID      string
	AssistantMessageID string
	QAConfigVersionID  string
	LLMConfigVersionID string
	RequestID          string
	IntentType         *domain.IntentType
	Route              string
	Confidence         *float64
	Status             string
	StopReason         *string
	RetryCount         int
	PromptTokens       int
	CompletionTokens   int
	ReasoningTokens    int
	LatencyMs          int
	StartedAt          time.Time
	FinishedAt         *time.Time
}

type ResponseRunRepository struct {
	db *pgxpool.Pool
}

func NewResponseRunRepository(db *pgxpool.Pool) *ResponseRunRepository {
	return &ResponseRunRepository{db: db}
}

func (r *ResponseRunRepository) Create(
	ctx context.Context,
	conversationID string,
	userMessageID string,
) (ResponseRun, error) {
	id := uuid.NewString()
	now := nowUTC()
	const query = `
		INSERT INTO response_runs (
			id, conversation_id, user_message_id, status, started_at
		)
		VALUES ($1, $2, $3, 'running', $4)
		RETURNING id, conversation_id, user_message_id, assistant_message_id,
			qa_config_version_id, llm_config_version_id, request_id,
			intent_type, route, confidence, status, stop_reason, retry_count,
			prompt_tokens, completion_tokens, reasoning_tokens, latency_ms,
			started_at, finished_at
	`
	var run ResponseRun
	var intentTypeStr *string
	var confidenceVal *float64
	err := r.db.QueryRow(ctx, query, id, conversationID, userMessageID, now).Scan(
		&run.ID, &run.ConversationID, &run.UserMessageID, &run.AssistantMessageID,
		&run.QAConfigVersionID, &run.LLMConfigVersionID, &run.RequestID,
		&intentTypeStr, &run.Route, &confidenceVal, &run.Status, &run.StopReason,
		&run.RetryCount, &run.PromptTokens, &run.CompletionTokens, &run.ReasoningTokens,
		&run.LatencyMs, &run.StartedAt, &run.FinishedAt,
	)
	if err != nil {
		return ResponseRun{}, fmt.Errorf("create response run: %w", err)
	}
	if intentTypeStr != nil {
		intentType := domain.IntentType(*intentTypeStr)
		run.IntentType = &intentType
	}
	run.Confidence = confidenceVal
	return run, nil
}

func (r *ResponseRunRepository) UpdateAssistantMessageID(
	ctx context.Context,
	runID string,
	assistantMessageID string,
) error {
	const query = `
		UPDATE response_runs SET assistant_message_id = $2 WHERE id = $1
	`
	_, err := r.db.Exec(ctx, query, runID, assistantMessageID)
	if err != nil {
		return fmt.Errorf("update assistant message id: %w", err)
	}
	return nil
}

func (r *ResponseRunRepository) UpdateIntent(
	ctx context.Context,
	runID string,
	intentType domain.IntentType,
	route string,
	confidence float64,
) error {
	const query = `
		UPDATE response_runs
		SET intent_type = $2, route = $3, confidence = $4
		WHERE id = $1
	`
	_, err := r.db.Exec(ctx, query, runID, string(intentType), route, confidence)
	if err != nil {
		return fmt.Errorf("update response run intent: %w", err)
	}
	return nil
}

func (r *ResponseRunRepository) UpdateMetrics(
	ctx context.Context,
	runID string,
	promptTokens int,
	completionTokens int,
	reasoningTokens int,
	latencyMs int,
) error {
	const query = `
		UPDATE response_runs
		SET prompt_tokens = $2, completion_tokens = $3, reasoning_tokens = $4, latency_ms = $5
		WHERE id = $1
	`
	_, err := r.db.Exec(ctx, query, runID, promptTokens, completionTokens, reasoningTokens, latencyMs)
	if err != nil {
		return fmt.Errorf("update response run metrics: %w", err)
	}
	return nil
}

func (r *ResponseRunRepository) GetByID(ctx context.Context, id string) (ResponseRun, error) {
	const query = `
		SELECT id, conversation_id, user_message_id, assistant_message_id,
			qa_config_version_id, llm_config_version_id, request_id,
			intent_type, route, confidence, status, stop_reason, retry_count,
			prompt_tokens, completion_tokens, reasoning_tokens, latency_ms,
			started_at, finished_at
		FROM response_runs
		WHERE id = $1
	`
	var run ResponseRun
	var intentTypeStr *string
	var confidenceVal *float64
	err := r.db.QueryRow(ctx, query, id).Scan(
		&run.ID, &run.ConversationID, &run.UserMessageID, &run.AssistantMessageID,
		&run.QAConfigVersionID, &run.LLMConfigVersionID, &run.RequestID,
		&intentTypeStr, &run.Route, &confidenceVal, &run.Status, &run.StopReason,
		&run.RetryCount, &run.PromptTokens, &run.CompletionTokens, &run.ReasoningTokens,
		&run.LatencyMs, &run.StartedAt, &run.FinishedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return ResponseRun{}, pgx.ErrNoRows
	}
	if err != nil {
		return ResponseRun{}, fmt.Errorf("get response run by id: %w", err)
	}
	if intentTypeStr != nil {
		intentType := domain.IntentType(*intentTypeStr)
		run.IntentType = &intentType
	}
	run.Confidence = confidenceVal
	return run, nil
}

func (r *ResponseRunRepository) GetByUserMessageID(ctx context.Context, userMessageID string) (ResponseRun, error) {
	const query = `
		SELECT id, conversation_id, user_message_id, assistant_message_id,
			qa_config_version_id, llm_config_version_id, request_id,
			intent_type, route, confidence, status, stop_reason, retry_count,
			prompt_tokens, completion_tokens, reasoning_tokens, latency_ms,
			started_at, finished_at
		FROM response_runs
		WHERE user_message_id = $1
	`
	var run ResponseRun
	var intentTypeStr *string
	var confidenceVal *float64
	err := r.db.QueryRow(ctx, query, userMessageID).Scan(
		&run.ID, &run.ConversationID, &run.UserMessageID, &run.AssistantMessageID,
		&run.QAConfigVersionID, &run.LLMConfigVersionID, &run.RequestID,
		&intentTypeStr, &run.Route, &confidenceVal, &run.Status, &run.StopReason,
		&run.RetryCount, &run.PromptTokens, &run.CompletionTokens, &run.ReasoningTokens,
		&run.LatencyMs, &run.StartedAt, &run.FinishedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return ResponseRun{}, pgx.ErrNoRows
	}
	if err != nil {
		return ResponseRun{}, fmt.Errorf("get response run by user message: %w", err)
	}
	if intentTypeStr != nil {
		intentType := domain.IntentType(*intentTypeStr)
		run.IntentType = &intentType
	}
	run.Confidence = confidenceVal
	return run, nil
}

func (r *ResponseRunRepository) MarkCompleted(ctx context.Context, runID string) error {
	now := nowUTC()
	const query = `
		UPDATE response_runs
		SET status = 'completed', finished_at = $2
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
		SET status = 'failed', stop_reason = $2, finished_at = $3
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
		SET status = 'stopped', stop_reason = $2, finished_at = $3
		WHERE id = $1 AND status = 'running'
	`
	_, err := r.db.Exec(ctx, query, runID, stopReason, now)
	if err != nil {
		return fmt.Errorf("mark response run stopped: %w", err)
	}
	return nil
}
