package repository

import "time"

type QAConfigVersion struct {
	ID                      string
	VersionNo               int64
	DefaultKnowledgeBaseIDs []string
	Retrieval               RetrievalOptions
	LLM                     ModelConfig
	Agent                   AgentConfig
	IsActive                bool
	ActivateRequested       bool
	CreatedAt               time.Time
}

type RetrievalOptions struct {
	TopK                int
	SimilarityThreshold float64
	UseRerank           bool
	RerankThreshold     *float64
	RerankTopN          *int
}

type ModelConfig struct {
	Provider       string
	ProfileID      string
	ModelName      string
	TimeoutSeconds int
	Temperature    float64
	MaxTokens      int
}

type AgentConfig struct {
	MaxIterations         int
	ToolTimeoutSeconds    int
	ModelTimeoutSeconds   int
	OverallTimeoutSeconds int
	EnabledToolNames      []string
}

type LLMConfigVersion struct {
	ID                string
	VersionNo         int64
	Provider          string
	ProfileID         string
	ModelName         string
	TimeoutSeconds    int
	Temperature       float64
	MaxTokens         int
	IsActive          bool
	ActivateRequested bool
	CreatedAt         time.Time
}

type AdminAuditLog struct {
	ID             string
	ExternalUserID string
	Action         string
	TargetType     string
	TargetID       string
	RequestID      string
	CreatedAt      time.Time
}

type LLMConnectionTest struct {
	ID           string
	ProfileID    string
	ModelName    string
	Status       string
	LatencyMS    int64
	ErrorCode    string
	ErrorMessage string
	RequestID    string
	CreatedAt    time.Time
}
