package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/qa/internal/repository"
	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/qa/internal/service"
)

func TestCreateGetUpdateDeleteSession(t *testing.T) {
	now := time.Date(2026, 6, 28, 9, 0, 0, 0, time.UTC)
	repo := repository.NewMemoryRepository()
	sessions := service.New(repo,
		service.WithIDGenerator(func() string { return "qa_sess_test" }),
		service.WithClock(func() time.Time { return now }),
	)
	reqCtx := service.RequestContext{UserID: "user_1", RequestID: "req_test"}

	created, err := sessions.CreateSession(context.Background(), reqCtx, service.CreateSessionInput{Title: "daily check"})
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	if created.ID != "qa_sess_test" || created.Status != service.SessionStatusActive || created.Title != "daily check" {
		t.Fatalf("created session = %+v", created)
	}

	got, err := sessions.GetSession(context.Background(), reqCtx, created.ID)
	if err != nil {
		t.Fatalf("GetSession() error = %v", err)
	}
	if got.ID != created.ID {
		t.Fatalf("GetSession id = %q", got.ID)
	}

	archived := service.SessionStatusArchived
	updatedTitle := "renamed"
	updated, err := sessions.UpdateSession(context.Background(), reqCtx, service.UpdateSessionInput{
		SessionID: created.ID,
		Title:     &updatedTitle,
		Status:    &archived,
	})
	if err != nil {
		t.Fatalf("UpdateSession() error = %v", err)
	}
	if updated.Title != updatedTitle || updated.Status != archived {
		t.Fatalf("updated session = %+v", updated)
	}

	if err := sessions.DeleteSession(context.Background(), reqCtx, created.ID); err != nil {
		t.Fatalf("DeleteSession() error = %v", err)
	}
	if _, err := sessions.GetSession(context.Background(), reqCtx, created.ID); !hasCode(err, service.CodeNotFound) {
		t.Fatalf("GetSession after delete error = %v, want not_found", err)
	}
}

func TestListSessionsFiltersOwnerAndAggregatesMessages(t *testing.T) {
	now := time.Date(2026, 6, 28, 9, 0, 0, 0, time.UTC)
	repo := repository.NewMemoryRepository()
	repo.SeedSession(repository.Session{ID: "s1", ExternalUserID: "user_1", Title: "one", Status: service.SessionStatusActive, CreatedAt: now, UpdatedAt: now.Add(time.Hour)})
	repo.SeedSession(repository.Session{ID: "s2", ExternalUserID: "user_2", Title: "other", Status: service.SessionStatusActive, CreatedAt: now, UpdatedAt: now.Add(2 * time.Hour)})
	repo.SeedSession(repository.Session{ID: "s3", ExternalUserID: "user_1", Title: "archived", Status: service.SessionStatusArchived, CreatedAt: now, UpdatedAt: now.Add(3 * time.Hour)})
	repo.SeedSession(repository.Session{ID: "s4", ExternalUserID: "user_1", Title: "different", Status: service.SessionStatusActive, CreatedAt: now, UpdatedAt: now.Add(4 * time.Hour)})
	repo.SeedMessages(
		repository.Message{ID: "m1", ConversationID: "s1", SequenceNo: 1, Content: "first", CreatedAt: now},
		repository.Message{ID: "m2", ConversationID: "s1", SequenceNo: 2, Content: "latest preview", CreatedAt: now.Add(time.Minute)},
	)

	sessions := service.New(repo)
	result, err := sessions.ListSessions(context.Background(), service.RequestContext{UserID: "user_1"}, service.ListSessionsInput{})
	if err != nil {
		t.Fatalf("ListSessions() error = %v", err)
	}
	if result.Page.Total != 2 || len(result.Sessions) != 2 {
		t.Fatalf("result count = total %d len %d", result.Page.Total, len(result.Sessions))
	}
	session := result.Sessions[1]
	if session.ID != "s1" || session.MessageCount != 2 || session.LastMessagePreview != "latest preview" {
		t.Fatalf("session summary = %+v", session)
	}

	archived, err := sessions.ListSessions(context.Background(), service.RequestContext{UserID: "user_1"}, service.ListSessionsInput{Status: service.SessionStatusArchived})
	if err != nil {
		t.Fatalf("ListSessions archived error = %v", err)
	}
	if archived.Page.Total != 1 || archived.Sessions[0].ID != "s3" {
		t.Fatalf("archived result = %+v", archived)
	}

	filtered, err := sessions.ListSessions(context.Background(), service.RequestContext{UserID: "user_1"}, service.ListSessionsInput{Query: "preview"})
	if err != nil {
		t.Fatalf("ListSessions query error = %v", err)
	}
	if filtered.Page.Total != 1 || filtered.Sessions[0].ID != "s1" {
		t.Fatalf("query filtered result = %+v", filtered)
	}
}

func TestSessionAccessControlAndValidation(t *testing.T) {
	repo := repository.NewMemoryRepository()
	repo.SeedSession(repository.Session{ID: "s1", ExternalUserID: "user_1", Status: service.SessionStatusActive, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()})
	sessions := service.New(repo)

	if _, err := sessions.GetSession(context.Background(), service.RequestContext{UserID: "user_2"}, "s1"); !hasCode(err, service.CodeForbidden) {
		t.Fatalf("cross-user error = %v, want forbidden", err)
	}
	if _, err := sessions.ListSessions(context.Background(), service.RequestContext{}, service.ListSessionsInput{}); !hasCode(err, service.CodeUnauthorized) {
		t.Fatalf("missing user error = %v, want unauthorized", err)
	}
	if _, err := sessions.ListSessions(context.Background(), service.RequestContext{UserID: "user_1"}, service.ListSessionsInput{Status: "deleted"}); !hasCode(err, service.CodeValidation) {
		t.Fatalf("invalid status error = %v, want validation_error", err)
	}
}

func hasCode(err error, code service.Code) bool {
	appErr, ok := service.Classify(err)
	return ok && appErr.Code == code
}
