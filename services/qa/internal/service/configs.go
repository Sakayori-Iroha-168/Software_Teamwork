package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/url"
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
	Provider       string  `json:"provider"`
	APIURL         string  `json:"apiUrl"`
	ModelName      string  `json:"modelName"`
	APIKey         string  `json:"apiKey,omitempty"`
	TimeoutSeconds int     `json:"timeoutSeconds"`
	Temperature    float64 `json:"temperature"`
	MaxTokens      int     `json:"maxTokens"`
}

type LLMConfigResponse struct {
	ID             string    `json:"id"`
	VersionNo      int64     `json:"versionNo"`
	Provider       string    `json:"provider"`
	APIURL         string    `json:"apiUrl"`
	ModelName      string    `json:"modelName"`
	APIKeyLast4    string    `json:"apiKeyLast4,omitempty"`
	TimeoutSeconds int       `json:"timeoutSeconds"`
	Temperature    float64   `json:"temperature"`
	MaxTokens      int       `json:"maxTokens"`
	IsActive       bool      `json:"isActive"`
	CreatedAt      time.Time `json:"createdAt"`
}

type LLMConnectionTestRequest struct {
	Provider       string `json:"provider,omitempty"`
	APIURL         string `json:"apiUrl,omitempty"`
	ModelName      string `json:"modelName,omitempty"`
	APIKey         string `json:"apiKey,omitempty"`
	TimeoutSeconds int    `json:"timeoutSeconds,omitempty"`
}

type LLMConnectionTestResponse struct {
	Success   bool      `json:"success"`
	Provider  string    `json:"provider"`
	ModelName string    `json:"modelName"`
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
	if err := validateLLMConfig(req.Provider, req.APIURL, req.ModelName, req.TimeoutSeconds, req.MaxTokens); err != nil {
		return LLMConfigResponse{}, err
	}
	cfg := repository.LLMConfigVersion{
		ID:              newID("llm_cfg"),
		Provider:        req.Provider,
		APIURL:          req.APIURL,
		ModelName:       req.ModelName,
		APIKeySecretRef: secretRef(req.APIKey),
		APIKeyLast4:     last4(req.APIKey),
		TimeoutSeconds:  req.TimeoutSeconds,
		Temperature:     req.Temperature,
		MaxTokens:       req.MaxTokens,
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
	if req.Provider == "" || req.APIURL == "" || req.ModelName == "" {
		current, err := s.store.CurrentLLMConfig(ctx)
		if err != nil {
			return LLMConnectionTestResponse{}, err
		}
		if req.Provider == "" {
			req.Provider = current.Provider
		}
		if req.APIURL == "" {
			req.APIURL = current.APIURL
		}
		if req.ModelName == "" {
			req.ModelName = current.ModelName
		}
	}
	if err := validateLLMConfig(req.Provider, req.APIURL, req.ModelName, defaultIfZero(req.TimeoutSeconds, 60), 1); err != nil {
		return LLMConnectionTestResponse{}, err
	}
	return LLMConnectionTestResponse{
		Success:   true,
		Provider:  req.Provider,
		ModelName: req.ModelName,
		LatencyMS: time.Since(startedAt).Milliseconds(),
		TestedAt:  time.Now().UTC(),
		Message:   "connection parameters are valid; provider call is mocked in this service draft",
	}, nil
}

func validateQAConfig(req QAConfigRequest) error {
	if req.TopK <= 0 {
		return errors.New("topK must be greater than 0")
	}
	if req.SimilarityThreshold < 0 || req.SimilarityThreshold > 1 {
		return errors.New("similarityThreshold must be between 0 and 1")
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

func validateLLMConfig(provider string, apiURL string, modelName string, timeoutSeconds int, maxTokens int) error {
	if strings.TrimSpace(provider) == "" {
		return errors.New("provider is required")
	}
	if strings.TrimSpace(modelName) == "" {
		return errors.New("modelName is required")
	}
	parsedURL, err := url.ParseRequestURI(apiURL)
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		return errors.New("apiUrl must be a valid absolute URL")
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
		Provider:       cfg.Provider,
		APIURL:         cfg.APIURL,
		ModelName:      cfg.ModelName,
		APIKeyLast4:    cfg.APIKeyLast4,
		TimeoutSeconds: cfg.TimeoutSeconds,
		Temperature:    cfg.Temperature,
		MaxTokens:      cfg.MaxTokens,
		IsActive:       cfg.IsActive,
		CreatedAt:      cfg.CreatedAt,
	}
}

func secretRef(secret string) string {
	if secret == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(secret))
	return "memory://llm-api-key/" + hex.EncodeToString(sum[:8])
}

func last4(secret string) string {
	if len(secret) <= 4 {
		return secret
	}
	return secret[len(secret)-4:]
}

func defaultIfZero(value int, fallback int) int {
	if value == 0 {
		return fallback
	}
	return value
}
