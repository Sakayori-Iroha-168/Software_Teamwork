package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/qa/internal/repository"
)

const (
	SessionStatusActive   = "active"
	SessionStatusArchived = "archived"
)

type Store interface {
	CreateSession(ctx context.Context, session repository.Session) (repository.SessionSummary, error)
	GetSessionSummary(ctx context.Context, sessionID string) (repository.SessionSummary, error)
	ListSessionSummaries(ctx context.Context, filter repository.SessionListFilter) (repository.SessionListResult, error)
	UpdateSession(ctx context.Context, sessionID string, update repository.SessionUpdate) (repository.SessionSummary, error)
	SoftDeleteSession(ctx context.Context, sessionID string, deletedAt time.Time) error
}

type Service struct {
	store Store
	now   func() time.Time
	newID func() string
}

type Option func(*Service)

func WithClock(clock func() time.Time) Option {
	return func(s *Service) {
		if clock != nil {
			s.now = clock
		}
	}
}

func WithIDGenerator(generator func() string) Option {
	return func(s *Service) {
		if generator != nil {
			s.newID = generator
		}
	}
}

func New(store Store, opts ...Option) *Service {
	s := &Service{
		store: store,
		now:   func() time.Time { return time.Now().UTC() },
		newID: newSessionID,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

type RequestContext struct {
	RequestID      string
	UserID         string
	Roles          []string
	Permissions    []string
	ForwardedFor   string
	ForwardedProto string
}

type QASession struct {
	ID                 string    `json:"id"`
	Title              string    `json:"title,omitempty"`
	Status             string    `json:"status"`
	MessageCount       int       `json:"messageCount"`
	LastMessagePreview string    `json:"lastMessagePreview,omitempty"`
	CreatedAt          time.Time `json:"createdAt"`
	UpdatedAt          time.Time `json:"updatedAt"`
}

type Page struct {
	Page     int `json:"page"`
	PageSize int `json:"pageSize"`
	Total    int `json:"total"`
}

type CreateSessionInput struct {
	Title string
}

type ListSessionsInput struct {
	Page     int
	PageSize int
	Status   string
	Query    string
	Sort     string
}

type ListSessionsResult struct {
	Sessions []QASession
	Page     Page
}

type UpdateSessionInput struct {
	SessionID string
	Title     *string
	Status    *string
}

func (s *Service) CreateSession(ctx context.Context, reqCtx RequestContext, input CreateSessionInput) (QASession, error) {
	if err := validateRequestContext(reqCtx); err != nil {
		return QASession{}, err
	}

	now := s.now()
	summary, err := s.store.CreateSession(ctx, repository.Session{
		ID:             s.newID(),
		ExternalUserID: reqCtx.UserID,
		Title:          strings.TrimSpace(input.Title),
		Status:         SessionStatusActive,
		CreatedAt:      now,
		UpdatedAt:      now,
	})
	if err != nil {
		return QASession{}, mapRepositoryError(err)
	}
	return sessionFromSummary(summary), nil
}

func (s *Service) ListSessions(ctx context.Context, reqCtx RequestContext, input ListSessionsInput) (ListSessionsResult, error) {
	if err := validateRequestContext(reqCtx); err != nil {
		return ListSessionsResult{}, err
	}

	page := input.Page
	if page == 0 {
		page = 1
	}
	pageSize := input.PageSize
	if pageSize == 0 {
		pageSize = 20
	}
	status := input.Status
	if status == "" {
		status = SessionStatusActive
	}
	sortBy := input.Sort
	if sortBy == "" {
		sortBy = "-updatedAt"
	}
	query := strings.TrimSpace(input.Query)

	fields := map[string]string{}
	if page < 1 {
		fields["page"] = "must be greater than or equal to 1"
	}
	if pageSize < 1 || pageSize > 100 {
		fields["pageSize"] = "must be between 1 and 100"
	}
	if !isSessionStatus(status) {
		fields["status"] = "must be active or archived"
	}
	if !isAllowedSort(sortBy) {
		fields["sort"] = "is not supported"
	}
	if len(fields) > 0 {
		return ListSessionsResult{}, ValidationError("request validation failed", fields)
	}

	result, err := s.store.ListSessionSummaries(ctx, repository.SessionListFilter{
		ExternalUserID: reqCtx.UserID,
		Status:         status,
		Query:          query,
		Sort:           sortBy,
		Page:           page,
		PageSize:       pageSize,
	})
	if err != nil {
		return ListSessionsResult{}, mapRepositoryError(err)
	}

	sessions := make([]QASession, len(result.Items))
	for i, item := range result.Items {
		sessions[i] = sessionFromSummary(item)
	}
	return ListSessionsResult{
		Sessions: sessions,
		Page: Page{
			Page:     page,
			PageSize: pageSize,
			Total:    result.Total,
		},
	}, nil
}

func (s *Service) GetSession(ctx context.Context, reqCtx RequestContext, sessionID string) (QASession, error) {
	summary, err := s.getOwnedSession(ctx, reqCtx, sessionID)
	if err != nil {
		return QASession{}, err
	}
	return sessionFromSummary(summary), nil
}

func (s *Service) UpdateSession(ctx context.Context, reqCtx RequestContext, input UpdateSessionInput) (QASession, error) {
	summary, err := s.getOwnedSession(ctx, reqCtx, input.SessionID)
	if err != nil {
		return QASession{}, err
	}

	update := repository.SessionUpdate{UpdatedAt: s.now()}
	fields := map[string]string{}
	if input.Title != nil {
		title := strings.TrimSpace(*input.Title)
		update.Title = &title
	}
	if input.Status != nil {
		status := strings.TrimSpace(*input.Status)
		if !isSessionStatus(status) {
			fields["status"] = "must be active or archived"
		} else {
			update.Status = &status
		}
	}
	if input.Title == nil && input.Status == nil {
		fields["body"] = "title or status is required"
	}
	if len(fields) > 0 {
		return QASession{}, ValidationError("request validation failed", fields)
	}

	updated, err := s.store.UpdateSession(ctx, summary.Session.ID, update)
	if err != nil {
		return QASession{}, mapRepositoryError(err)
	}
	return sessionFromSummary(updated), nil
}

func (s *Service) DeleteSession(ctx context.Context, reqCtx RequestContext, sessionID string) error {
	summary, err := s.getOwnedSession(ctx, reqCtx, sessionID)
	if err != nil {
		return err
	}
	if err := s.store.SoftDeleteSession(ctx, summary.Session.ID, s.now()); err != nil {
		return mapRepositoryError(err)
	}
	return nil
}

func (s *Service) getOwnedSession(ctx context.Context, reqCtx RequestContext, sessionID string) (repository.SessionSummary, error) {
	if err := validateRequestContext(reqCtx); err != nil {
		return repository.SessionSummary{}, err
	}
	if strings.TrimSpace(sessionID) == "" {
		return repository.SessionSummary{}, ValidationError("request validation failed", map[string]string{"sessionId": "is required"})
	}

	summary, err := s.store.GetSessionSummary(ctx, sessionID)
	if err != nil {
		return repository.SessionSummary{}, mapRepositoryError(err)
	}
	if summary.Session.ExternalUserID != reqCtx.UserID {
		return repository.SessionSummary{}, ForbiddenError("qa session access denied")
	}
	return summary, nil
}

func validateRequestContext(reqCtx RequestContext) error {
	if strings.TrimSpace(reqCtx.UserID) == "" {
		return UnauthorizedError()
	}
	return nil
}

func sessionFromSummary(summary repository.SessionSummary) QASession {
	session := summary.Session
	return QASession{
		ID:                 session.ID,
		Title:              session.Title,
		Status:             session.Status,
		MessageCount:       summary.MessageCount,
		LastMessagePreview: summary.LastMessagePreview,
		CreatedAt:          session.CreatedAt,
		UpdatedAt:          session.UpdatedAt,
	}
}

func isSessionStatus(status string) bool {
	return status == SessionStatusActive || status == SessionStatusArchived
}

func isAllowedSort(sortBy string) bool {
	switch sortBy {
	case "-updatedAt", "updatedAt", "-createdAt", "createdAt", "title", "-title":
		return true
	default:
		return false
	}
}

func mapRepositoryError(err error) error {
	if errors.Is(err, repository.ErrSessionNotFound) {
		return NotFoundError("qa session not found")
	}
	return NewError(CodeInternal, "qa session storage failed", err)
}

func newSessionID() string {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		return "qa_sess_" + strconv.FormatInt(time.Now().UnixNano(), 10)
	}
	return "qa_sess_" + hex.EncodeToString(bytes)
}
