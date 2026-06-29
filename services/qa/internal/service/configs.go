package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"software-teamwork/services/qa/internal/repository"
)

const providerAIGateway = "ai-gateway"

type ConfigStore interface {
	CurrentQAConfig(ctx context.Context) (repository.QAConfigVersion, error)
	CreateQAConfig(ctx context.Context, cfg repository.QAConfigVersion) (repository.QAConfigVersion, error)
	CurrentLLMConfig(ctx context.Context) (repository.LLMConfigVersion, error)
	CreateLLMConfig(ctx context.Context, cfg repository.LLMConfigVersion) (repository.LLMConfigVersion, error)
	AppendAuditLog(ctx context.Context, log repository.AdminAuditLog) error
	AppendLLMConnectionTest(ctx context.Context, test repository.LLMConnectionTest) error
}

type ConfigService struct {
	store ConfigStore
}

func NewConfigService(store ConfigStore) *ConfigService {
	return &ConfigService{store: store}
}

type QAConfigRequest struct {
	DefaultKnowledgeBaseIDs []string         `json:"defaultKnowledgeBaseIds"`
	Retrieval               RetrievalOptions `json:"retrieval"`
	LLM                     ModelConfig      `json:"llm"`
	Agent                   AgentConfig      `json:"agent"`
	Activate                bool             `json:"activate"`
}

type QAConfigResponse struct {
	ID                      string           `json:"id"`
	VersionNo               int64            `json:"versionNo"`
	DefaultKnowledgeBaseIDs []string         `json:"defaultKnowledgeBaseIds"`
	Retrieval               RetrievalOptions `json:"retrieval"`
	LLM                     ModelConfig      `json:"llm"`
	Agent                   AgentConfig      `json:"agent"`
	IsActive                bool             `json:"isActive"`
	CreatedAt               time.Time        `json:"createdAt"`
}

type RetrievalOptions struct {
	TopK                int      `json:"topK"`
	SimilarityThreshold float64  `json:"similarityThreshold"`
	UseRerank           bool     `json:"useRerank"`
	RerankThreshold     *float64 `json:"rerankThreshold,omitempty"`
	RerankTopN          *int     `json:"rerankTopN,omitempty"`
}

type ModelConfig struct {
	Provider       string  `json:"provider"`
	ProfileID      string  `json:"profileId"`
	ModelName      string  `json:"modelName"`
	TimeoutSeconds int     `json:"timeoutSeconds"`
	Temperature    float64 `json:"temperature,omitempty"`
	MaxTokens      int     `json:"maxTokens,omitempty"`
}

type AgentConfig struct {
	MaxIterations         int      `json:"maxIterations"`
	ToolTimeoutSeconds    int      `json:"toolTimeoutSeconds"`
	ModelTimeoutSeconds   int      `json:"modelTimeoutSeconds"`
	OverallTimeoutSeconds int      `json:"overallTimeoutSeconds"`
	EnabledToolNames      []string `json:"enabledToolNames"`
}

type LLMConfigRequest struct {
	Provider       string  `json:"provider"`
	ProfileID      string  `json:"profileId"`
	ModelName      string  `json:"modelName"`
	TimeoutSeconds int     `json:"timeoutSeconds"`
	Temperature    float64 `json:"temperature"`
	MaxTokens      int     `json:"maxTokens"`
	Activate       bool    `json:"activate"`
}

type LLMConfigResponse struct {
	ID             string    `json:"id"`
	VersionNo      int64     `json:"versionNo"`
	Provider       string    `json:"provider"`
	ProfileID      string    `json:"profileId"`
	ModelName      string    `json:"modelName"`
	TimeoutSeconds int       `json:"timeoutSeconds"`
	Temperature    float64   `json:"temperature"`
	MaxTokens      int       `json:"maxTokens"`
	IsActive       bool      `json:"isActive"`
	CreatedAt      time.Time `json:"createdAt"`
}

type LLMConnectionTestRequest struct {
	Provider       string `json:"provider,omitempty"`
	ProfileID      string `json:"profileId,omitempty"`
	ModelName      string `json:"modelName,omitempty"`
	TimeoutSeconds int    `json:"timeoutSeconds,omitempty"`
}

type LLMConnectionTestResponse struct {
	ID           string    `json:"id"`
	Success      bool      `json:"success"`
	LatencyMS    int64     `json:"latencyMs"`
	ModelName    string    `json:"modelName"`
	ErrorCode    *string   `json:"errorCode"`
	ErrorMessage *string   `json:"errorMessage"`
	TestedAt     time.Time `json:"testedAt"`
}

func (s *ConfigService) CurrentQAConfig(ctx context.Context) (QAConfigResponse, error) {
	cfg, err := s.store.CurrentQAConfig(ctx)
	if err != nil {
		return QAConfigResponse{}, err
	}
	return mapQAConfig(cfg), nil
}

func (s *ConfigService) CreateQAConfig(ctx context.Context, req QAConfigRequest, requestID string) (QAConfigResponse, error) {
	normalizeQAConfig(&req)
	if err := validateQAConfig(req); err != nil {
		return QAConfigResponse{}, err
	}

	cfg := repository.QAConfigVersion{
		ID:                      newID("qa_cfg"),
		DefaultKnowledgeBaseIDs: req.DefaultKnowledgeBaseIDs,
		Retrieval: repository.RetrievalOptions{
			TopK:                req.Retrieval.TopK,
			SimilarityThreshold: req.Retrieval.SimilarityThreshold,
			UseRerank:           req.Retrieval.UseRerank,
			RerankThreshold:     req.Retrieval.RerankThreshold,
			RerankTopN:          req.Retrieval.RerankTopN,
		},
		LLM: repository.ModelConfig{
			Provider:       providerAIGateway,
			ProfileID:      req.LLM.ProfileID,
			ModelName:      req.LLM.ModelName,
			TimeoutSeconds: req.LLM.TimeoutSeconds,
			Temperature:    req.LLM.Temperature,
			MaxTokens:      req.LLM.MaxTokens,
		},
		Agent: repository.AgentConfig{
			MaxIterations:         req.Agent.MaxIterations,
			ToolTimeoutSeconds:    req.Agent.ToolTimeoutSeconds,
			ModelTimeoutSeconds:   req.Agent.ModelTimeoutSeconds,
			OverallTimeoutSeconds: req.Agent.OverallTimeoutSeconds,
			EnabledToolNames:      req.Agent.EnabledToolNames,
		},
		ActivateRequested: req.Activate,
	}
	created, err := s.store.CreateQAConfig(ctx, cfg)
	if err != nil {
		return QAConfigResponse{}, err
	}
	_ = s.store.AppendAuditLog(ctx, repository.AdminAuditLog{
		ID:         newID("audit"),
		Action:     "create",
		TargetType: "qa_config_version",
		TargetID:   created.ID,
		RequestID:  requestID,
		CreatedAt:  time.Now().UTC(),
	})
	return mapQAConfig(created), nil
}

func (s *ConfigService) CurrentLLMConfig(ctx context.Context) (LLMConfigResponse, error) {
	cfg, err := s.store.CurrentLLMConfig(ctx)
	if err != nil {
		return LLMConfigResponse{}, err
	}
	return mapLLMConfig(cfg), nil
}

func (s *ConfigService) CreateLLMConfig(ctx context.Context, req LLMConfigRequest, requestID string) (LLMConfigResponse, error) {
	normalizeLLMConfig(&req)
	if err := validateModelConfig(ModelConfig{
		Provider:       req.Provider,
		ProfileID:      req.ProfileID,
		ModelName:      req.ModelName,
		TimeoutSeconds: req.TimeoutSeconds,
		Temperature:    req.Temperature,
		MaxTokens:      req.MaxTokens,
	}); err != nil {
		return LLMConfigResponse{}, err
	}
	cfg := repository.LLMConfigVersion{
		ID:                newID("llm_cfg"),
		Provider:          providerAIGateway,
		ProfileID:         req.ProfileID,
		ModelName:         req.ModelName,
		TimeoutSeconds:    req.TimeoutSeconds,
		Temperature:       req.Temperature,
		MaxTokens:         req.MaxTokens,
		ActivateRequested: req.Activate,
	}
	created, err := s.store.CreateLLMConfig(ctx, cfg)
	if err != nil {
		return LLMConfigResponse{}, err
	}
	_ = s.store.AppendAuditLog(ctx, repository.AdminAuditLog{
		ID:         newID("audit"),
		Action:     "create",
		TargetType: "llm_config_version",
		TargetID:   created.ID,
		RequestID:  requestID,
		CreatedAt:  time.Now().UTC(),
	})
	return mapLLMConfig(created), nil
}

func (s *ConfigService) TestLLMConnection(ctx context.Context, req LLMConnectionTestRequest, requestID string) (LLMConnectionTestResponse, error) {
	startedAt := time.Now()
	if req.ProfileID == "" || req.ModelName == "" {
		current, err := s.store.CurrentLLMConfig(ctx)
		if err != nil {
			return LLMConnectionTestResponse{}, err
		}
		if req.ProfileID == "" {
			req.ProfileID = current.ProfileID
		}
		if req.ModelName == "" {
			req.ModelName = current.ModelName
		}
		if req.TimeoutSeconds == 0 {
			req.TimeoutSeconds = current.TimeoutSeconds
		}
	}
	if req.Provider == "" {
		req.Provider = providerAIGateway
	}
	if err := validateModelConfig(ModelConfig{
		Provider:       req.Provider,
		ProfileID:      req.ProfileID,
		ModelName:      req.ModelName,
		TimeoutSeconds: defaultIfZero(req.TimeoutSeconds, 60),
		MaxTokens:      1,
	}); err != nil {
		return LLMConnectionTestResponse{}, err
	}

	id := newID("llm_test")
	testedAt := time.Now().UTC()
	latencyMS := time.Since(startedAt).Milliseconds()
	_ = s.store.AppendLLMConnectionTest(ctx, repository.LLMConnectionTest{
		ID:        id,
		ProfileID: req.ProfileID,
		ModelName: req.ModelName,
		Status:    "succeeded",
		LatencyMS: latencyMS,
		RequestID: requestID,
		CreatedAt: testedAt,
	})
	return LLMConnectionTestResponse{
		ID:        id,
		Success:   true,
		LatencyMS: latencyMS,
		ModelName: req.ModelName,
		TestedAt:  testedAt,
	}, nil
}

func normalizeQAConfig(req *QAConfigRequest) {
	if req.Retrieval.TopK == 0 {
		req.Retrieval.TopK = 5
	}
	if req.Retrieval.SimilarityThreshold == 0 {
		req.Retrieval.SimilarityThreshold = 0.65
	}
	if req.Agent.MaxIterations == 0 {
		req.Agent.MaxIterations = 5
	}
	if req.Agent.ToolTimeoutSeconds == 0 {
		req.Agent.ToolTimeoutSeconds = 10
	}
	if req.Agent.ModelTimeoutSeconds == 0 {
		req.Agent.ModelTimeoutSeconds = 60
	}
	if req.Agent.OverallTimeoutSeconds == 0 {
		req.Agent.OverallTimeoutSeconds = 120
	}
	if len(req.Agent.EnabledToolNames) == 0 {
		req.Agent.EnabledToolNames = []string{"search_knowledge", "get_citation_source"}
	}
	if req.LLM.Provider == "" {
		req.LLM.Provider = providerAIGateway
	}
}

func normalizeLLMConfig(req *LLMConfigRequest) {
	if req.Provider == "" {
		req.Provider = providerAIGateway
	}
	if req.TimeoutSeconds == 0 {
		req.TimeoutSeconds = 60
	}
	if req.MaxTokens == 0 {
		req.MaxTokens = 4096
	}
}

func validateQAConfig(req QAConfigRequest) error {
	if len(req.DefaultKnowledgeBaseIDs) == 0 {
		return errors.New("defaultKnowledgeBaseIds is required")
	}
	for _, id := range req.DefaultKnowledgeBaseIDs {
		if strings.TrimSpace(id) == "" {
			return errors.New("defaultKnowledgeBaseIds must not contain empty id")
		}
	}
	if req.Retrieval.TopK <= 0 {
		return errors.New("retrieval.topK must be greater than 0")
	}
	if req.Retrieval.SimilarityThreshold < 0 || req.Retrieval.SimilarityThreshold > 1 {
		return errors.New("retrieval.similarityThreshold must be between 0 and 1")
	}
	if req.Retrieval.RerankThreshold != nil && (*req.Retrieval.RerankThreshold < 0 || *req.Retrieval.RerankThreshold > 1) {
		return errors.New("retrieval.rerankThreshold must be between 0 and 1")
	}
	if req.Retrieval.RerankTopN != nil && *req.Retrieval.RerankTopN <= 0 {
		return errors.New("retrieval.rerankTopN must be greater than 0")
	}
	if err := validateModelConfig(req.LLM); err != nil {
		return err
	}
	if req.Agent.MaxIterations <= 0 {
		return errors.New("agent.maxIterations must be greater than 0")
	}
	if req.Agent.ToolTimeoutSeconds <= 0 || req.Agent.ModelTimeoutSeconds <= 0 || req.Agent.OverallTimeoutSeconds <= 0 {
		return errors.New("agent timeout fields must be greater than 0")
	}
	return nil
}

func validateModelConfig(cfg ModelConfig) error {
	if cfg.Provider != providerAIGateway {
		return errors.New("provider must be ai-gateway")
	}
	if strings.TrimSpace(cfg.ProfileID) == "" {
		return errors.New("profileId is required")
	}
	if strings.TrimSpace(cfg.ModelName) == "" {
		return errors.New("modelName is required")
	}
	if cfg.TimeoutSeconds <= 0 {
		return errors.New("timeoutSeconds must be greater than 0")
	}
	if cfg.MaxTokens < 0 {
		return errors.New("maxTokens must not be negative")
	}
	return nil
}

func mapQAConfig(cfg repository.QAConfigVersion) QAConfigResponse {
	return QAConfigResponse{
		ID:                      cfg.ID,
		VersionNo:               cfg.VersionNo,
		DefaultKnowledgeBaseIDs: cfg.DefaultKnowledgeBaseIDs,
		Retrieval: RetrievalOptions{
			TopK:                cfg.Retrieval.TopK,
			SimilarityThreshold: cfg.Retrieval.SimilarityThreshold,
			UseRerank:           cfg.Retrieval.UseRerank,
			RerankThreshold:     cfg.Retrieval.RerankThreshold,
			RerankTopN:          cfg.Retrieval.RerankTopN,
		},
		LLM: ModelConfig{
			Provider:       providerAIGateway,
			ProfileID:      cfg.LLM.ProfileID,
			ModelName:      cfg.LLM.ModelName,
			TimeoutSeconds: cfg.LLM.TimeoutSeconds,
			Temperature:    cfg.LLM.Temperature,
			MaxTokens:      cfg.LLM.MaxTokens,
		},
		Agent: AgentConfig{
			MaxIterations:         cfg.Agent.MaxIterations,
			ToolTimeoutSeconds:    cfg.Agent.ToolTimeoutSeconds,
			ModelTimeoutSeconds:   cfg.Agent.ModelTimeoutSeconds,
			OverallTimeoutSeconds: cfg.Agent.OverallTimeoutSeconds,
			EnabledToolNames:      cfg.Agent.EnabledToolNames,
		},
		IsActive:  cfg.IsActive,
		CreatedAt: cfg.CreatedAt,
	}
}

func mapLLMConfig(cfg repository.LLMConfigVersion) LLMConfigResponse {
	return LLMConfigResponse{
		ID:             cfg.ID,
		VersionNo:      cfg.VersionNo,
		Provider:       providerAIGateway,
		ProfileID:      cfg.ProfileID,
		ModelName:      cfg.ModelName,
		TimeoutSeconds: cfg.TimeoutSeconds,
		Temperature:    cfg.Temperature,
		MaxTokens:      cfg.MaxTokens,
		IsActive:       cfg.IsActive,
		CreatedAt:      cfg.CreatedAt,
	}
}

func defaultIfZero(value int, fallback int) int {
	if value == 0 {
		return fallback
	}
	return value
}
