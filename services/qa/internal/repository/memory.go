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
	qaConfigs     []QAConfigVersion
	llmConfigs    []LLMConfigVersion
	auditLogs     []AdminAuditLog
}

func NewMemoryStore() *MemoryStore {
	rerankThreshold := 0.5
	rerankTopN := 3
	now := time.Now().UTC()
	return &MemoryStore{
		runs: make(map[string]ResponseRun),
		qaConfigs: []QAConfigVersion{
			{
				ID:                  "qa_cfg_seed",
				VersionNo:           1,
				TopK:                5,
				SimilarityThreshold: 0.7,
				UseRerank:           true,
				RerankThreshold:     &rerankThreshold,
				RerankTopN:          &rerankTopN,
				IsActive:            true,
				CreatedAt:           now,
				CreatedByUserID:     "seed",
				KnowledgeBases: []QAConfigKnowledgeBase{
					{ExternalKBID: "kb_power_standard", KBType: "technical_supervision", DisplayNameSnapshot: "电力标准规范库", SortOrder: 1},
				},
			},
		},
		llmConfigs: []LLMConfigVersion{
			{
				ID:             "llm_cfg_seed",
				VersionNo:      1,
				Provider:       "openai-compatible",
				APIURL:         "https://api.example.com/v1",
				ModelName:      "gpt-4o-mini",
				APIKeyLast4:    "",
				TimeoutSeconds: 60,
				Temperature:    0.7,
				MaxTokens:      4096,
				IsActive:       true,
				CreatedAt:      now,
			},
		},
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

func (s *MemoryStore) CurrentQAConfig(_ context.Context) (QAConfigVersion, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := len(s.qaConfigs) - 1; i >= 0; i-- {
		if s.qaConfigs[i].IsActive {
			return s.qaConfigs[i], nil
		}
	}
	return QAConfigVersion{}, fmt.Errorf("active qa config not found")
}

func (s *MemoryStore) CreateQAConfig(_ context.Context, cfg QAConfigVersion) (QAConfigVersion, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var maxVersion int64
	for i := range s.qaConfigs {
		if s.qaConfigs[i].VersionNo > maxVersion {
			maxVersion = s.qaConfigs[i].VersionNo
		}
		s.qaConfigs[i].IsActive = false
	}
	cfg.VersionNo = maxVersion + 1
	cfg.IsActive = true
	cfg.CreatedAt = time.Now().UTC()
	s.qaConfigs = append(s.qaConfigs, cfg)
	return cfg, nil
}

func (s *MemoryStore) CurrentLLMConfig(_ context.Context) (LLMConfigVersion, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := len(s.llmConfigs) - 1; i >= 0; i-- {
		if s.llmConfigs[i].IsActive {
			return s.llmConfigs[i], nil
		}
	}
	return LLMConfigVersion{}, fmt.Errorf("active llm config not found")
}

func (s *MemoryStore) CreateLLMConfig(_ context.Context, cfg LLMConfigVersion) (LLMConfigVersion, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var maxVersion int64
	for i := range s.llmConfigs {
		if s.llmConfigs[i].VersionNo > maxVersion {
			maxVersion = s.llmConfigs[i].VersionNo
		}
		s.llmConfigs[i].IsActive = false
	}
	cfg.VersionNo = maxVersion + 1
	cfg.IsActive = true
	cfg.CreatedAt = time.Now().UTC()
	s.llmConfigs = append(s.llmConfigs, cfg)
	return cfg, nil
}

func (s *MemoryStore) AppendAuditLog(_ context.Context, log AdminAuditLog) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.auditLogs = append(s.auditLogs, log)
	return nil
}
