package repository

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"
)

type MemoryRepository struct {
	mu       sync.RWMutex
	sessions map[string]Session
	messages map[string][]Message
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		sessions: make(map[string]Session),
		messages: make(map[string][]Message),
	}
}

func (r *MemoryRepository) CreateSession(ctx context.Context, session Session) (SessionSummary, error) {
	if err := ctx.Err(); err != nil {
		return SessionSummary{}, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.sessions[session.ID] = session
	return r.summaryLocked(session), nil
}

func (r *MemoryRepository) GetSessionSummary(ctx context.Context, sessionID string) (SessionSummary, error) {
	if err := ctx.Err(); err != nil {
		return SessionSummary{}, err
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	session, ok := r.sessions[sessionID]
	if !ok || session.DeletedAt != nil {
		return SessionSummary{}, ErrSessionNotFound
	}
	return r.summaryLocked(session), nil
}

func (r *MemoryRepository) ListSessionSummaries(ctx context.Context, filter SessionListFilter) (SessionListResult, error) {
	if err := ctx.Err(); err != nil {
		return SessionListResult{}, err
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	page := filter.Page
	if page <= 0 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}

	items := make([]SessionSummary, 0, len(r.sessions))
	for _, session := range r.sessions {
		if session.DeletedAt != nil {
			continue
		}
		if session.ExternalUserID != filter.ExternalUserID {
			continue
		}
		if filter.Status != "" && session.Status != filter.Status {
			continue
		}
		items = append(items, r.summaryLocked(session))
	}
	sortSessionSummaries(items, filter.Sort)

	total := len(items)
	start := (page - 1) * pageSize
	if start >= total {
		return SessionListResult{Items: []SessionSummary{}, Total: total}, nil
	}
	end := start + pageSize
	if end > total {
		end = total
	}

	return SessionListResult{
		Items: append([]SessionSummary(nil), items[start:end]...),
		Total: total,
	}, nil
}

func (r *MemoryRepository) UpdateSession(ctx context.Context, sessionID string, update SessionUpdate) (SessionSummary, error) {
	if err := ctx.Err(); err != nil {
		return SessionSummary{}, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	session, ok := r.sessions[sessionID]
	if !ok || session.DeletedAt != nil {
		return SessionSummary{}, ErrSessionNotFound
	}
	if update.Title != nil {
		session.Title = *update.Title
	}
	if update.Status != nil {
		session.Status = *update.Status
	}
	if update.UpdatedAt.IsZero() {
		session.UpdatedAt = time.Now().UTC()
	} else {
		session.UpdatedAt = update.UpdatedAt
	}
	r.sessions[sessionID] = session
	return r.summaryLocked(session), nil
}

func (r *MemoryRepository) SoftDeleteSession(ctx context.Context, sessionID string, deletedAt time.Time) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	session, ok := r.sessions[sessionID]
	if !ok || session.DeletedAt != nil {
		return ErrSessionNotFound
	}
	if deletedAt.IsZero() {
		deletedAt = time.Now().UTC()
	}
	session.DeletedAt = &deletedAt
	session.UpdatedAt = deletedAt
	r.sessions[sessionID] = session
	return nil
}

func (r *MemoryRepository) SeedSession(session Session) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sessions[session.ID] = session
}

func (r *MemoryRepository) SeedMessages(messages ...Message) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, message := range messages {
		r.messages[message.ConversationID] = append(r.messages[message.ConversationID], message)
	}
}

func (r *MemoryRepository) summaryLocked(session Session) SessionSummary {
	messages := append([]Message(nil), r.messages[session.ID]...)
	sort.SliceStable(messages, func(i, j int) bool {
		if messages[i].SequenceNo == messages[j].SequenceNo {
			return messages[i].CreatedAt.Before(messages[j].CreatedAt)
		}
		return messages[i].SequenceNo < messages[j].SequenceNo
	})
	return SessionSummary{
		Session:            session,
		MessageCount:       len(messages),
		LastMessagePreview: lastMessagePreview(messages),
	}
}

func lastMessagePreview(messages []Message) string {
	for i := len(messages) - 1; i >= 0; i-- {
		if preview := strings.TrimSpace(messages[i].Content); preview != "" {
			runes := []rune(preview)
			if len(runes) > 120 {
				return string(runes[:120])
			}
			return preview
		}
	}
	return ""
}

func sortSessionSummaries(items []SessionSummary, sortBy string) {
	sort.SliceStable(items, func(i, j int) bool {
		left := items[i].Session
		right := items[j].Session
		switch sortBy {
		case "createdAt":
			return left.CreatedAt.Before(right.CreatedAt)
		case "-createdAt":
			return right.CreatedAt.Before(left.CreatedAt)
		case "updatedAt":
			return left.UpdatedAt.Before(right.UpdatedAt)
		case "title":
			return left.Title < right.Title
		case "-title":
			return right.Title < left.Title
		case "-updatedAt":
			fallthrough
		default:
			return right.UpdatedAt.Before(left.UpdatedAt)
		}
	})
}
