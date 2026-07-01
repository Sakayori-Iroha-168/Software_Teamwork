package localtools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/qa/internal/service"
)

type fakeAttachmentSearcher struct {
	userID        string
	sessionID     string
	attachmentIDs []string
	query         string
}

func (f *fakeAttachmentSearcher) Search(_ context.Context, userID, sessionID string, attachmentIDs []string, query string, limit int) ([]service.AttachmentChunk, error) {
	f.userID = userID
	f.sessionID = sessionID
	f.attachmentIDs = append([]string(nil), attachmentIDs...)
	f.query = query
	page := 3
	return []service.AttachmentChunk{{
		ID: "chunk-1", AttachmentID: "att-1", Filename: "manual.pdf",
		PageNumber: &page, SectionPath: "1", Preview: "safe preview", CreatedAt: time.Now(),
	}}, nil
}

func TestSearchSessionAttachmentsRequiresMessageScope(t *testing.T) {
	searcher := &fakeAttachmentSearcher{}
	client, err := New(Config{WorkDir: t.TempDir(), MaxFileBytes: 1024, MaxOutputBytes: 1024, CommandTimeout: time.Second, AttachmentSearcher: searcher})
	if err != nil {
		t.Fatal(err)
	}
	tools, err := client.ListTools(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, tool := range tools {
		found = found || tool.Function.Name == ToolSearchSessionAttachments
	}
	if !found {
		t.Fatal("attachment search tool was not listed")
	}
	result, err := client.CallTool(context.Background(), ToolSearchSessionAttachments, json.RawMessage(`{"query":"relay"}`))
	if err != nil || !result.IsError || !strings.Contains(result.Content, "attachment_scope_missing") {
		t.Fatalf("missing scope not rejected: %+v err=%v", result, err)
	}
	ctx := service.WithAttachmentScope(context.Background(), service.AttachmentScope{UserID: "user-1", SessionID: "session-1", AttachmentIDs: []string{"att-1"}})
	result, err = client.CallTool(ctx, ToolSearchSessionAttachments, json.RawMessage(`{"query":"relay","limit":1}`))
	if err != nil || result.IsError {
		t.Fatalf("search failed: %+v err=%v", result, err)
	}
	if searcher.userID != "user-1" || searcher.sessionID != "session-1" || len(searcher.attachmentIDs) != 1 || searcher.query != "relay" {
		t.Fatalf("unexpected search scope: %+v", searcher)
	}
	if !strings.Contains(result.Content, `"attachmentId":"att-1"`) || strings.Contains(result.Content, "Body") {
		t.Fatalf("unexpected safe result payload: %s", result.Content)
	}
}
