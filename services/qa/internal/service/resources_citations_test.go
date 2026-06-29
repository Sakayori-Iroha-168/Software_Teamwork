package service

import (
	"context"
	"testing"
)

// stubResourceRepo 嵌入 ResourceRepository 接口，只覆盖 SaveCitations，
// 其余方法不会在本测试中被调用。
type stubResourceRepo struct {
	ResourceRepository
	savedMessageID string
	saved          []Citation
}

func (s *stubResourceRepo) SaveCitations(_ context.Context, messageID string, citations []Citation) ([]Citation, error) {
	s.savedMessageID = messageID
	s.saved = citations
	return citations, nil
}

func TestSaveCitationsAssignsStableCitationNo(t *testing.T) {
	repo := &stubResourceRepo{}
	svc := &ResourceService{repository: repo}

	if _, err := svc.SaveCitations(context.Background(), "msg-1", []Citation{
		{DocumentName: "a"},
		{DocumentName: "b", CitationNo: 5},
		{DocumentName: "c"},
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if repo.savedMessageID != "msg-1" {
		t.Fatalf("messageID not propagated: %q", repo.savedMessageID)
	}
	want := []int{1, 5, 3}
	for i, c := range repo.saved {
		if c.CitationNo != want[i] {
			t.Errorf("citation %d: got citationNo %d, want %d", i, c.CitationNo, want[i])
		}
		if c.MessageID != "msg-1" {
			t.Errorf("citation %d: messageID not set", i)
		}
		if c.Metadata == nil {
			t.Errorf("citation %d: metadata should default to non-nil", i)
		}
	}
}

func TestSaveCitationsRequiresMessageID(t *testing.T) {
	svc := &ResourceService{repository: &stubResourceRepo{}}
	if _, err := svc.SaveCitations(context.Background(), "", nil); err == nil {
		t.Fatal("expected validation error for empty messageId")
	}
}
