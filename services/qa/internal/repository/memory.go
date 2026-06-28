package repository

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type MemoryStore struct {
	mu            sync.Mutex
	runs          map[string]ResponseRun
	steps         []ResponseProcessStep
	streamEvents  []ResponseStreamEvent
	contentBlocks []MessageContentBlock
	citations     []Citation
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		runs: make(map[string]ResponseRun),
	}
}

func (s *MemoryStore) CreateRun(_ context.Context, run ResponseRun) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.runs[run.ID]; exists {
		return fmt.Errorf("response run already exists: %s", run.ID)
	}
	s.runs[run.ID] = run
	return nil
}

func (s *MemoryStore) AppendProcessStep(_ context.Context, step ResponseProcessStep) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.steps = append(s.steps, step)
	return nil
}

func (s *MemoryStore) AppendStreamEvent(_ context.Context, event ResponseStreamEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.streamEvents = append(s.streamEvents, event)
	return nil
}

func (s *MemoryStore) AppendContentBlock(_ context.Context, block MessageContentBlock) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.contentBlocks = append(s.contentBlocks, block)
	return nil
}

func (s *MemoryStore) AppendCitation(_ context.Context, citation Citation) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.citations = append(s.citations, citation)
	return nil
}

func (s *MemoryStore) CompleteRun(_ context.Context, runID string, status string, stopReason string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	run, exists := s.runs[runID]
	if !exists {
		return fmt.Errorf("response run not found: %s", runID)
	}
	run.Status = status
	run.StopReason = stopReason
	run.FinishedAt = time.Now().UTC()
	s.runs[runID] = run
	return nil
}
