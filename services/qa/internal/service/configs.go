package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"software-teamwork/services/qa/internal/repository"
)

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
	TopK                int                          `json:"topK"`
	SimilarityThreshold float64                      `json:"similarityThreshold"`
	UseRerank           bool                         `json:"useRerank"`
	RerankThreshold     *float64                     `json:"rerankThreshold,omitempty"`
	RerankTopN          *int                         `json:"rerankTopN,omitempty"`
	Activate            bool                         `json:"activate"`
	KnowledgeBases      []QAConfigKnowledgeBaseInput `json:"knowledgeBases"`
	CreatedByUserID     string                       `json:"createdByUserId,omitempty"`
}

type QAConfigKnowledgeBaseInput struct {
	ExternalKBID        string `json:"externalKbId"`
	KBType              string `json:"kbType"`
	DisplayNameSnapshot string `json:"displayNameSnapshot,omitempty"`
	SortOrder           int    `json:"sortOrder,omitempty"`
}

type QAConfigResponse struct {
	ID                  string                          `json:"id"`
	VersionNo           int64                           `json:"versionNo"`
	TopK                int                             `json:"topK"`
	SimilarityThreshold float64                         `json:"similarityThreshold"`
	UseRerank           bool                            `json:"useRerank"`
	RerankThreshold     *float64                        `json:"rerankThreshold,omitempty"`
	RerankTopN          *int                            `json:"rerankTopN,omitempty"`
	IsActive            bool                            `json:"isActive"`
	ActivateRequested   bool                            `json:"activateRequested"`
	CreatedAt           time.Time                       `json:"createdAt"`
	CreatedByUserID     string                          `json:"createdByUserId,omitempty"`
	KnowledgeBases      []QAConfigKnowledgeBaseResponse `json:"knowledgeBases"`
}

type QAConfigKnowledgeBaseResponse struct {
	ExternalKBID        string `json:"externalKbId"`
	KBType              string `json:"kbType"`
	DisplayNameSnapshot string `json:"displayNameSnapshot,omitempty"`
	SortOrder           int    `json:"sortOrder"`
}

type LLMConfigRequest struct {
	ProfileID      string  `json:"profileId"`
	ModelName      string  `json:"modelName"`
	TimeoutSeconds int     `json:"timeoutSeconds"`
	Temperature    float64 `json:"temperature"`
	MaxTokens      int     `json:"maxTokens"`
}

type LLMConfigResponse struct {
	ID             string    `json:"id"`
	VersionNo      int64     `json:"versionNo"`
	ProfileID      string    `json:"profileId"`
	ModelName      string    `json:"modelName"`
	TimeoutSeconds int       `json:"timeoutSeconds"`
	Temperature    float64   `json:"temperature"`
	MaxTokens      int       `json:"maxTokens"`
	IsActive       bool      `json:"isActive"`
	CreatedAt      time.Time `json:"createdAt"`
}

type LLMConnectionTestRequest struct {
	ProfileID      string `json:"profileId,omitempty"`
	ModelName      string `json:"modelName,omitempty"`
	TimeoutSeconds int    `json:"timeoutSeconds,omitempty"`
}

type LLMConnectionTestResponse struct {
	Success   bool      `json:"success"`
	ProfileID string    `json:"profileId"`
	ModelName string    `json:"modelName"`
	Status    string    `json:"status"`
	LatencyMS int64     `json:"latencyMs"`
	TestedAt  time.Time `json:"testedAt"`
	Message   string    `json:"message"`
}

func (s *ConfigService) CurrentQAConfig(ctx context.Context) (QAConfigResponse, error) {
	cfg, err := s.store.CurrentQAConfig(ctx)
	if err != nil {
		return QAConfigResponse{}, err
	}
	return mapQAConfig(cfg), nil
}

func (s *ConfigService) CreateQAConfig(ctx context.Context, req QAConfigRequest, requestID string) (QAConfigResponse, error) {
	if err := validateQAConfig(req); err != nil {
		return QAConfigResponse{}, err
	}
	cfg := repository.QAConfigVersion{
		ID:                  newID("qa_cfg"),
		TopK:                req.TopK,
		SimilarityThreshold: req.SimilarityThreshold,
		UseRerank:           req.UseRerank,
		RerankThreshold:     req.RerankThreshold,
		RerankTopN:          req.RerankTopN,
		ActivateRequested:   req.Activate,
		CreatedByUserID:     req.CreatedByUserID,
	}
	for _, kb := range req.KnowledgeBases {
		cfg.KnowledgeBases = append(cfg.KnowledgeBases, repository.QAConfigKnowledgeBase{
			ExternalKBID:        kb.ExternalKBID,
			KBType:              kb.KBType,
			DisplayNameSnapshot: kb.DisplayNameSnapshot,
			SortOrder:           kb.SortOrder,
		})
	}
	created, err := s.store.CreateQAConfig(ctx, cfg)
	if err != nil {
		return QAConfigResponse{}, err
	}
	_ = s.store.AppendAuditLog(ctx, repository.AdminAuditLog{
		ID:             newID("audit"),
		ExternalUserID: req.CreatedByUserID,
		Action:         "create",
		TargetType:     "qa_config_version",
		TargetID:       created.ID,
		RequestID:      requestID,
		CreatedAt:      time.Now().UTC(),
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
	if err := validateLLMConfig(req.ProfileID, req.ModelName, req.TimeoutSeconds, req.MaxTokens); err != nil {
		return LLMConfigResponse{}, err
	}
	cfg := repository.LLMConfigVersion{
		ID:             newID("llm_cfg"),
		ProfileID:      req.ProfileID,
		ModelName:      req.ModelName,
		TimeoutSeconds: req.TimeoutSeconds,
		Temperature:    req.Temperature,
		MaxTokens:      req.MaxTokens,
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

func (s *ConfigService) TestLLMConnection(ctx context.Context, req LLMConnectionTestRequest) (LLMConnectionTestResponse, error) {
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
	}
	if err := validateLLMConfig(req.ProfileID, req.ModelName, defaultIfZero(req.TimeoutSeconds, 60), 1); err != nil {
		return LLMConnectionTestResponse{}, err
	}
	latencyMS := time.Since(startedAt).Milliseconds()
	testedAt := time.Now().UTC()
	_ = s.store.AppendLLMConnectionTest(ctx, repository.LLMConnectionTest{
		ID:        newID("llm_test"),
		ProfileID: req.ProfileID,
		ModelName: req.ModelName,
		Status:    "succeeded",
		LatencyMS: latencyMS,
		RequestID: "",
		CreatedAt: testedAt,
	})
	return LLMConnectionTestResponse{
		Success:   true,
		ProfileID: req.ProfileID,
		ModelName: req.ModelName,
		Status:    "succeeded",
		LatencyMS: latencyMS,
		TestedAt:  testedAt,
		Message:   "connection parameters are valid; AI Gateway provider call is mocked in this service draft",
	}, nil
}

func validateQAConfig(req QAConfigRequest) error {
	if req.TopK <= 0 {
		return errors.New("topK must be greater than 0")
	}
	if req.SimilarityThreshold < 0 || req.SimilarityThreshold > 1 {
		return errors.New("similarityThreshold must be between 0 and 1")
	}
	if req.RerankThreshold != nil && (*req.RerankThreshold < 0 || *req.RerankThreshold > 1) {
		return errors.New("rerankThreshold must be between 0 and 1")
	}
	if req.RerankTopN != nil && *req.RerankTopN <= 0 {
		return errors.New("rerankTopN must be greater than 0")
	}
	for _, kb := range req.KnowledgeBases {
		if strings.TrimSpace(kb.ExternalKBID) == "" {
			return errors.New("knowledgeBases.externalKbId is required")
		}
		if strings.TrimSpace(kb.KBType) == "" {
			return errors.New("knowledgeBases.kbType is required")
		}
	}
	return nil
}

func validateLLMConfig(profileID string, modelName string, timeoutSeconds int, maxTokens int) error {
	if strings.TrimSpace(profileID) == "" {
		return errors.New("profileId is required")
	}
	if strings.TrimSpace(modelName) == "" {
		return errors.New("modelName is required")
	}
	if timeoutSeconds <= 0 {
		return errors.New("timeoutSeconds must be greater than 0")
	}
	if maxTokens <= 0 {
		return errors.New("maxTokens must be greater than 0")
	}
	return nil
}

func mapQAConfig(cfg repository.QAConfigVersion) QAConfigResponse {
	resp := QAConfigResponse{
		ID:                  cfg.ID,
		VersionNo:           cfg.VersionNo,
		TopK:                cfg.TopK,
		SimilarityThreshold: cfg.SimilarityThreshold,
		UseRerank:           cfg.UseRerank,
		RerankThreshold:     cfg.RerankThreshold,
		RerankTopN:          cfg.RerankTopN,
		IsActive:            cfg.IsActive,
		ActivateRequested:   cfg.ActivateRequested,
		CreatedAt:           cfg.CreatedAt,
		CreatedByUserID:     cfg.CreatedByUserID,
	}
	for _, kb := range cfg.KnowledgeBases {
		resp.KnowledgeBases = append(resp.KnowledgeBases, QAConfigKnowledgeBaseResponse{
			ExternalKBID:        kb.ExternalKBID,
			KBType:              kb.KBType,
			DisplayNameSnapshot: kb.DisplayNameSnapshot,
			SortOrder:           kb.SortOrder,
		})
	}
	return resp
}

func mapLLMConfig(cfg repository.LLMConfigVersion) LLMConfigResponse {
	return LLMConfigResponse{
		ID:             cfg.ID,
		VersionNo:      cfg.VersionNo,
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
