package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/qa/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ResponseStreamEventRepository struct {
	db *pgxpool.Pool
}

func NewResponseStreamEventRepository(db *pgxpool.Pool) *ResponseStreamEventRepository {
	return &ResponseStreamEventRepository{db: db}
}

func (r *ResponseStreamEventRepository) Create(
	ctx context.Context,
	responseRunID string,
	eventSeq int,
	eventType domain.StreamEventType,
	payload map[string]any,
	expiresAt *time.Time,
) (domain.ResponseStreamEvent, error) {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return domain.ResponseStreamEvent{}, fmt.Errorf("marshal payload: %w", err)
	}

	const query = `
		INSERT INTO response_stream_events (response_run_id, event_seq, event_type, payload, expires_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, response_run_id, event_seq, event_type, payload, expires_at, created_at
	`

	var event domain.ResponseStreamEvent
	var payloadRaw []byte
	err = r.db.QueryRow(ctx, query, responseRunID, eventSeq, string(eventType), payloadJSON, expiresAt).Scan(
		&event.ID, &event.ResponseRunID, &event.EventSeq, &event.EventType,
		&payloadRaw, &event.ExpiresAt, &event.CreatedAt,
	)
	if err != nil {
		return domain.ResponseStreamEvent{}, fmt.Errorf("create stream event: %w", err)
	}

	if err := json.Unmarshal(payloadRaw, &event.Payload); err != nil {
		return domain.ResponseStreamEvent{}, fmt.Errorf("unmarshal payload: %w", err)
	}
	return event, nil
}

func (r *ResponseStreamEventRepository) ListByResponseRunID(
	ctx context.Context,
	responseRunID string,
	afterEventSeq int,
) ([]domain.ResponseStreamEvent, error) {
	const query = `
		SELECT id, response_run_id, event_seq, event_type, payload, expires_at, created_at
		FROM response_stream_events
		WHERE response_run_id = $1 AND event_seq > $2
		ORDER BY event_seq ASC
	`

	rows, err := r.db.Query(ctx, query, responseRunID, afterEventSeq)
	if err != nil {
		return nil, fmt.Errorf("list stream events: %w", err)
	}
	defer rows.Close()

	var events []domain.ResponseStreamEvent
	for rows.Next() {
		var event domain.ResponseStreamEvent
		var payloadRaw []byte
		if err := rows.Scan(
			&event.ID, &event.ResponseRunID, &event.EventSeq, &event.EventType,
			&payloadRaw, &event.ExpiresAt, &event.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan stream event: %w", err)
		}
		if err := json.Unmarshal(payloadRaw, &event.Payload); err != nil {
			return nil, fmt.Errorf("unmarshal payload: %w", err)
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate stream events: %w", err)
	}
	return events, nil
}

func (r *ResponseStreamEventRepository) DeleteExpired(ctx context.Context, before time.Time) (int64, error) {
	const query = `DELETE FROM response_stream_events WHERE expires_at < $1`
	result, err := r.db.Exec(ctx, query, before)
	if err != nil {
		return 0, fmt.Errorf("delete expired events: %w", err)
	}
	return result.RowsAffected(), nil
}
