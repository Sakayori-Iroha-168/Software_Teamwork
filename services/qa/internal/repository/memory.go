package repository

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type MemoryStore struct {
	mu         sync.Mutex
	qaConfigs  []QAConfigVersion
	llmConfigs []LLMConfigVersion
	llmTests   []LLMConnectionTest
	auditLogs  []AdminAuditLog
}

func NewMemoryStore() *MemoryStore {
	rerankThreshold := 0.5
	rerankTopN := 3
	now := time.Now().UTC()
	return &MemoryStore{
		qaConfigs: []QAConfigVersion{
			{
				ID:                      "qa_cfg_seed",
				VersionNo:               1,
				DefaultKnowledgeBaseIDs: []string{"kb_power_standard"},
				Retrieval: RetrievalOptions{
					TopK:                5,
					SimilarityThreshold: 0.65,
					UseRerank:           true,
					RerankThreshold:     &rerankThreshold,
					RerankTopN:          &rerankTopN,
				},
				LLM: ModelConfig{
					Provider:       "ai-gateway",
					ProfileID:      "mp_chat_default",
					ModelName:      "gpt-4o-mini",
					TimeoutSeconds: 60,
					Temperature:    0.7,
					MaxTokens:      4096,
				},
				Agent: AgentConfig{
					MaxIterations:         5,
					ToolTimeoutSeconds:    10,
					ModelTimeoutSeconds:   60,
					OverallTimeoutSeconds: 120,
					EnabledToolNames:      []string{"search_knowledge", "get_citation_source"},
				},
				IsActive:          true,
				ActivateRequested: true,
				CreatedAt:         now,
			},
		},
		llmConfigs: []LLMConfigVersion{
			{
				ID:                "llm_cfg_seed",
				VersionNo:         1,
				Provider:          "ai-gateway",
				ProfileID:         "mp_chat_default",
				ModelName:         "gpt-4o-mini",
				TimeoutSeconds:    60,
				Temperature:       0.7,
				MaxTokens:         4096,
				IsActive:          true,
				ActivateRequested: true,
				CreatedAt:         now,
			},
		},
	}
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
		if cfg.ActivateRequested {
			s.qaConfigs[i].IsActive = false
		}
	}
	cfg.VersionNo = maxVersion + 1
	cfg.IsActive = cfg.ActivateRequested
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
		if cfg.ActivateRequested {
			s.llmConfigs[i].IsActive = false
		}
	}
	cfg.VersionNo = maxVersion + 1
	cfg.IsActive = cfg.ActivateRequested
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

func (s *MemoryStore) AppendLLMConnectionTest(_ context.Context, test LLMConnectionTest) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.llmTests = append(s.llmTests, test)
	return nil
}
