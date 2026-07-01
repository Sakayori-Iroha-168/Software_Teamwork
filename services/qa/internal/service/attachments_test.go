package service

import (
	"bytes"
	"context"
	"errors"
	"io"
	"sort"
	"testing"
	"time"
)

type fakeAttachmentRepo struct {
	items             map[string]SessionAttachment
	chunks            map[string][]AttachmentChunk
	fileDeleteErrors  map[string]string
	fileDeleteMarkers []string
}

func newFakeAttachmentRepo() *fakeAttachmentRepo {
	return &fakeAttachmentRepo{items: map[string]SessionAttachment{}, chunks: map[string][]AttachmentChunk{}, fileDeleteErrors: map[string]string{}}
}

func (r *fakeAttachmentRepo) CreateAttachment(_ context.Context, item SessionAttachment) (SessionAttachment, error) {
	r.items[item.ID] = item
	return item, nil
}
func (r *fakeAttachmentRepo) CountLiveAttachments(context.Context, string, string) (int, error) {
	count := 0
	for _, item := range r.items {
		if item.DeletedAt == nil && item.Status != AttachmentStatusDeleted {
			count++
		}
	}
	return count, nil
}
func (r *fakeAttachmentRepo) ListAttachments(_ context.Context, userID, sessionID string) ([]SessionAttachment, error) {
	out := []SessionAttachment{}
	for _, item := range r.items {
		if item.OwnerUserID == userID && item.SessionID == sessionID && item.DeletedAt == nil && item.Status != AttachmentStatusDeleted {
			out = append(out, item)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out, nil
}
func (r *fakeAttachmentRepo) GetAttachment(_ context.Context, userID, sessionID, attachmentID string) (SessionAttachment, error) {
	item, ok := r.items[attachmentID]
	if !ok || item.OwnerUserID != userID || item.SessionID != sessionID || item.DeletedAt != nil {
		return SessionAttachment{}, NewError(CodeNotFound, "attachment not found", nil)
	}
	return item, nil
}
func (r *fakeAttachmentRepo) MarkAttachmentParsing(_ context.Context, attachmentID string) (SessionAttachment, error) {
	item, ok := r.items[attachmentID]
	if !ok {
		return SessionAttachment{}, NewError(CodeNotFound, "attachment not found", nil)
	}
	item.Status = AttachmentStatusParsing
	r.items[attachmentID] = item
	return item, nil
}
func (r *fakeAttachmentRepo) MarkAttachmentFailed(_ context.Context, attachmentID, code, summary string) error {
	item := r.items[attachmentID]
	item.Status = AttachmentStatusFailed
	item.ErrorCode = code
	item.ErrorSummary = summary
	r.items[attachmentID] = item
	return nil
}
func (r *fakeAttachmentRepo) ReplaceAttachmentChunks(_ context.Context, attachmentID string, chunks []AttachmentChunk, pageCount int) error {
	item := r.items[attachmentID]
	item.Status = AttachmentStatusReady
	item.PageCount = pageCount
	item.ChunkCount = len(chunks)
	r.items[attachmentID] = item
	r.chunks[attachmentID] = append([]AttachmentChunk(nil), chunks...)
	return nil
}
func (r *fakeAttachmentRepo) SoftDeleteAttachment(_ context.Context, userID, sessionID, attachmentID string) (SessionAttachment, error) {
	item, err := r.GetAttachment(context.Background(), userID, sessionID, attachmentID)
	if err != nil {
		return SessionAttachment{}, err
	}
	now := time.Now().UTC()
	item.Status = AttachmentStatusDeleted
	item.DeletedAt = &now
	r.items[attachmentID] = item
	delete(r.chunks, attachmentID)
	return item, nil
}
func (r *fakeAttachmentRepo) ValidateReadyAttachments(_ context.Context, userID, sessionID string, ids []string) ([]SessionAttachment, error) {
	out := []SessionAttachment{}
	for _, id := range ids {
		item, ok := r.items[id]
		if !ok || item.OwnerUserID != userID || item.SessionID != sessionID || item.Status != AttachmentStatusReady || item.DeletedAt != nil {
			return nil, NewError(CodeNotFound, "attachment not found", nil)
		}
		out = append(out, item)
	}
	return out, nil
}
func (r *fakeAttachmentRepo) SearchSessionAttachmentChunks(_ context.Context, userID, sessionID string, ids []string, query string, limit int) ([]AttachmentChunk, error) {
	out := []AttachmentChunk{}
	for _, id := range ids {
		item := r.items[id]
		if item.OwnerUserID != userID || item.SessionID != sessionID || item.Status != AttachmentStatusReady {
			continue
		}
		for _, chunk := range r.chunks[id] {
			if query == "" || bytes.Contains([]byte(chunk.Body), []byte(query)) || bytes.Contains([]byte(chunk.Preview), []byte(query)) {
				chunk.Filename = item.Filename
				out = append(out, chunk)
			}
		}
	}
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}
func (r *fakeAttachmentRepo) CleanupExpiredAttachments(_ context.Context, now time.Time, limit int) ([]SessionAttachment, error) {
	out := []SessionAttachment{}
	for id, item := range r.items {
		if len(out) >= limit {
			break
		}
		if item.DeletedAt == nil && !item.ExpiresAt.After(now) {
			deletedAt := now
			item.DeletedAt = &deletedAt
			item.Status = AttachmentStatusDeleted
			r.items[id] = item
			delete(r.chunks, id)
			out = append(out, item)
		}
	}
	return out, nil
}
func (r *fakeAttachmentRepo) CleanupSessionAttachments(_ context.Context, userID, sessionID string) ([]SessionAttachment, error) {
	out := []SessionAttachment{}
	now := time.Now().UTC()
	for id, item := range r.items {
		if item.OwnerUserID == userID && item.SessionID == sessionID && item.DeletedAt == nil {
			item.DeletedAt = &now
			item.Status = AttachmentStatusDeleted
			r.items[id] = item
			delete(r.chunks, id)
			out = append(out, item)
		}
	}
	return out, nil
}
func (r *fakeAttachmentRepo) ListAttachmentsPendingFileDeleteRetry(_ context.Context, limit int) ([]SessionAttachment, error) {
	out := []SessionAttachment{}
	for id, summary := range r.fileDeleteErrors {
		if len(out) >= limit {
			break
		}
		if summary == "" {
			continue
		}
		if item, ok := r.items[id]; ok && item.DeletedAt != nil {
			out = append(out, item)
		}
	}
	return out, nil
}
func (r *fakeAttachmentRepo) MarkAttachmentFileDeleteRequested(_ context.Context, attachmentID, summary string) error {
	r.fileDeleteMarkers = append(r.fileDeleteMarkers, attachmentID+":"+summary)
	r.fileDeleteErrors[attachmentID] = summary
	return nil
}

type fakeFileClient struct {
	id             string
	data           []byte
	deleted        []string
	readErr        error
	deleteErr      error
	deleteErrByRef map[string]error
}

func (f *fakeFileClient) Upload(_ context.Context, input FileUploadInput) (FileObject, error) {
	if f.id == "" {
		f.id = "file-1"
	}
	data, _ := io.ReadAll(input.Body)
	f.data = data
	return FileObject{ID: f.id, Filename: input.Filename, ContentType: input.ContentType, SizeBytes: input.SizeBytes}, nil
}
func (f *fakeFileClient) Read(context.Context, string) ([]byte, error) {
	if f.readErr != nil {
		return nil, f.readErr
	}
	return append([]byte(nil), f.data...), nil
}
func (f *fakeFileClient) Delete(_ context.Context, fileRef string) error {
	f.deleted = append(f.deleted, fileRef)
	if f.deleteErrByRef != nil && f.deleteErrByRef[fileRef] != nil {
		return f.deleteErrByRef[fileRef]
	}
	if f.deleteErr != nil {
		return f.deleteErr
	}
	return nil
}

type fakeParserClient struct {
	doc ParsedDocument
	err error
}

func (p fakeParserClient) Parse(context.Context, ParseDocumentInput) (ParsedDocument, error) {
	return p.doc, p.err
}

func TestAttachmentServiceUploadProcessAndSearch(t *testing.T) {
	repo := newFakeAttachmentRepo()
	file := &fakeFileClient{}
	parser := fakeParserClient{doc: ParsedDocument{Pages: []ParsedPage{{PageNumber: 2, Content: "relay protection test content"}}}}
	svc, err := NewAttachmentService(repo, file, parser, AttachmentConfig{TTL: time.Hour, MaxBytes: 1024, MaxPerSession: 2, ProcessTimeout: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	item, err := svc.Upload(context.Background(), "user-1", "session-1", UploadAttachmentInput{Filename: "a.pdf", ContentType: "application/pdf", SizeBytes: 7, Body: bytes.NewReader([]byte("payload"))})
	if err != nil {
		t.Fatal(err)
	}
	if item.FileRef != "file-1" || item.Status != AttachmentStatusUploaded {
		t.Fatalf("unexpected uploaded item: %+v", item)
	}
	if err := svc.Process(context.Background(), item.ID); err != nil {
		t.Fatal(err)
	}
	ready := repo.items[item.ID]
	if ready.Status != AttachmentStatusReady || ready.ChunkCount != 1 || ready.PageCount != 1 {
		t.Fatalf("unexpected ready item: %+v", ready)
	}
	results, err := svc.Search(context.Background(), "user-1", "session-1", []string{item.ID}, "protection", 5)
	if err != nil || len(results) != 1 || results[0].Filename != "a.pdf" {
		t.Fatalf("unexpected search results: %+v err=%v", results, err)
	}
}

func TestAttachmentServiceProcessFailureIsSanitized(t *testing.T) {
	repo := newFakeAttachmentRepo()
	file := &fakeFileClient{readErr: errors.New("http://internal/object-key token secret")}
	parser := fakeParserClient{}
	svc, err := NewAttachmentService(repo, file, parser, AttachmentConfig{TTL: time.Hour, MaxBytes: 1024, MaxPerSession: 2, ProcessTimeout: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	item, err := repo.CreateAttachment(context.Background(), SessionAttachment{ID: "att-1", SessionID: "session-1", OwnerUserID: "user-1", FileRef: "file-1", Filename: "a.pdf", ContentType: "application/pdf", SizeBytes: 1, Status: AttachmentStatusUploaded, ExpiresAt: time.Now().Add(time.Hour)})
	if err != nil {
		t.Fatal(err)
	}
	err = svc.Process(context.Background(), item.ID)
	if err == nil {
		t.Fatal("expected processing error")
	}
	failed := repo.items[item.ID]
	if failed.Status != AttachmentStatusFailed || failed.ErrorSummary != "file content could not be read" {
		t.Fatalf("failure was not sanitized: %+v", failed)
	}
}

func TestAttachmentServiceCleanupExpiredDeletesFileAndChunks(t *testing.T) {
	repo := newFakeAttachmentRepo()
	file := &fakeFileClient{}
	parser := fakeParserClient{}
	svc, err := NewAttachmentService(repo, file, parser, AttachmentConfig{TTL: time.Hour, MaxBytes: 1024, MaxPerSession: 2, ProcessTimeout: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	svc.now = func() time.Time { return time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC) }
	expired := svc.now().Add(-time.Minute)
	repo.items["att-1"] = SessionAttachment{ID: "att-1", SessionID: "session-1", OwnerUserID: "user-1", FileRef: "file-1", Status: AttachmentStatusReady, ExpiresAt: expired}
	repo.chunks["att-1"] = []AttachmentChunk{{ID: "chunk-1", AttachmentID: "att-1", Body: "text"}}
	items, err := svc.CleanupExpired(context.Background(), 10)
	if err != nil || len(items) != 1 {
		t.Fatalf("cleanup failed: items=%+v err=%v", items, err)
	}
	if len(file.deleted) != 1 || file.deleted[0] != "file-1" {
		t.Fatalf("file delete not requested: %+v", file.deleted)
	}
	if len(repo.chunks["att-1"]) != 0 {
		t.Fatalf("chunks not deleted: %+v", repo.chunks)
	}
}

func TestAttachmentServiceDeleteSessionCleansAttachments(t *testing.T) {
	repo := newFakeAttachmentRepo()
	file := &fakeFileClient{}
	svc, err := NewAttachmentService(repo, file, fakeParserClient{}, AttachmentConfig{TTL: time.Hour, MaxBytes: 1024, MaxPerSession: 2, ProcessTimeout: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	repo.items["att-1"] = SessionAttachment{ID: "att-1", SessionID: "session-1", OwnerUserID: "user-1", FileRef: "file-1", Status: AttachmentStatusReady, ExpiresAt: time.Now().Add(time.Hour)}
	repo.chunks["att-1"] = []AttachmentChunk{{ID: "chunk-1", AttachmentID: "att-1", Body: "text"}}
	if err := svc.DeleteSession(context.Background(), "user-1", "session-1"); err != nil {
		t.Fatal(err)
	}
	if repo.items["att-1"].DeletedAt == nil || len(repo.chunks["att-1"]) != 0 || len(file.deleted) != 1 {
		t.Fatalf("session cleanup incomplete: item=%+v chunks=%+v deleted=%+v", repo.items["att-1"], repo.chunks, file.deleted)
	}
}

func TestAttachmentServiceDeleteSessionAttemptsAllFileDeletes(t *testing.T) {
	repo := newFakeAttachmentRepo()
	file := &fakeFileClient{deleteErrByRef: map[string]error{"file-1": errors.New("file unavailable")}}
	svc, err := NewAttachmentService(repo, file, fakeParserClient{}, AttachmentConfig{TTL: time.Hour, MaxBytes: 1024, MaxPerSession: 2, ProcessTimeout: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	expiresAt := time.Now().Add(time.Hour)
	repo.items["att-1"] = SessionAttachment{ID: "att-1", SessionID: "session-1", OwnerUserID: "user-1", FileRef: "file-1", Status: AttachmentStatusReady, ExpiresAt: expiresAt}
	repo.items["att-2"] = SessionAttachment{ID: "att-2", SessionID: "session-1", OwnerUserID: "user-1", FileRef: "file-2", Status: AttachmentStatusReady, ExpiresAt: expiresAt}

	err = svc.DeleteSession(context.Background(), "user-1", "session-1")
	if err == nil {
		t.Fatal("DeleteSession error = nil")
	}
	if len(file.deleted) != 2 {
		t.Fatalf("deleted refs = %+v, want both files attempted", file.deleted)
	}
	if repo.fileDeleteErrors["att-1"] == "" {
		t.Fatalf("failed file delete was not marked for retry: %+v", repo.fileDeleteErrors)
	}
	if repo.fileDeleteErrors["att-2"] != "" {
		t.Fatalf("successful file delete should clear retry state: %+v", repo.fileDeleteErrors)
	}
}

func TestAttachmentServiceCleanupExpiredRetriesFailedFileDelete(t *testing.T) {
	repo := newFakeAttachmentRepo()
	file := &fakeFileClient{}
	svc, err := NewAttachmentService(repo, file, fakeParserClient{}, AttachmentConfig{TTL: time.Hour, MaxBytes: 1024, MaxPerSession: 2, ProcessTimeout: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	deletedAt := time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC)
	repo.items["att-1"] = SessionAttachment{ID: "att-1", SessionID: "session-1", OwnerUserID: "user-1", FileRef: "file-1", Status: AttachmentStatusDeleted, DeletedAt: &deletedAt, ExpiresAt: deletedAt.Add(-time.Hour)}
	repo.fileDeleteErrors["att-1"] = "file delete request failed"

	items, err := svc.CleanupExpired(context.Background(), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Fatalf("deleted retry item should not be reported as newly expired: %+v", items)
	}
	if len(file.deleted) != 1 || file.deleted[0] != "file-1" {
		t.Fatalf("file delete was not retried: %+v", file.deleted)
	}
	if repo.fileDeleteErrors["att-1"] != "" {
		t.Fatalf("file delete error was not cleared: %+v", repo.fileDeleteErrors)
	}
}
