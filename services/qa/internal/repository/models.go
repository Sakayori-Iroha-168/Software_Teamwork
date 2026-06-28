package repository

import (
	"context"
	"errors"
	"time"
)

var ErrSessionNotFound = errors.New("qa session not found")

type Store interface {
	CreateSession(ctx context.Context, session Session) (SessionSummary, error)
	GetSessionSummary(ctx context.Context, sessionID string) (SessionSummary, error)
	ListSessionSummaries(ctx context.Context, filter SessionListFilter) (SessionListResult, error)
	UpdateSession(ctx context.Context, sessionID string, update SessionUpdate) (SessionSummary, error)
	SoftDeleteSession(ctx context.Context, sessionID string, deletedAt time.Time) error
}

type Session struct {
	ID             string
	ExternalUserID string
	Title          string
	Status         string
	CreatedAt      time.Time
	UpdatedAt      time.Time
	DeletedAt      *time.Time
}

type Message struct {
	ID             string
	ConversationID string
	Role           string
	Status         string
	SequenceNo     int
	Content        string
	CreatedAt      time.Time
}

type SessionSummary struct {
	Session            Session
	MessageCount       int
	LastMessagePreview string
}

type SessionListFilter struct {
	ExternalUserID string
	Status         string
	Sort           string
	Page           int
	PageSize       int
}

type SessionListResult struct {
	Items []SessionSummary
	Total int
}

type SessionUpdate struct {
	Title     *string
	Status    *string
	UpdatedAt time.Time
}
