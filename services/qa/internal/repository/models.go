package repository

import "time"

type ResponseRun struct {
	ID                 string
	ConversationID     string
	UserMessageID      string
	AssistantMessageID string
	RequestID          string
	IntentType         string
	Confidence         float64
	Route              string
	Status             string
	StopReason         string
	StartedAt          time.Time
	FinishedAt         time.Time
}

type ResponseProcessStep struct {
	ID            string
	ResponseRunID string
	StepOrder     int
	StepType      string
	Label         string
	Detail        string
	Status        string
	StartedAt     time.Time
	FinishedAt    time.Time
}

type ResponseStreamEvent struct {
	ID            string
	ResponseRunID string
	EventSeq      int
	EventType     string
	Payload       any
	CreatedAt     time.Time
}

type MessageContentBlock struct {
	ID         string
	MessageID  string
	BlockOrder int
	BlockType  string
	Content    string
	Status     string
	CreatedAt  time.Time
}

type Citation struct {
	ID              string
	MessageID       string
	CitationNo      int
	ExternalKBID    string
	ExternalDocID   string
	ExternalChunkID string
	DocName         string
	QuoteText       string
	Score           float64
	CreatedAt       time.Time
}

type QAConfigVersion struct {
	ID                  string
	VersionNo           int64
	TopK                int
	SimilarityThreshold float64
	UseRerank           bool
	RerankThreshold     *float64
	RerankTopN          *int
	IsActive            bool
	CreatedAt           time.Time
	CreatedByUserID     string
	KnowledgeBases      []QAConfigKnowledgeBase
}

type QAConfigKnowledgeBase struct {
	ExternalKBID        string
	KBType              string
	DisplayNameSnapshot string
	SortOrder           int
}

type LLMConfigVersion struct {
	ID              string
	VersionNo       int64
	Provider        string
	APIURL          string
	ModelName       string
	APIKeySecretRef string
	APIKeyLast4     string
	TimeoutSeconds  int
	Temperature     float64
	MaxTokens       int
	IsActive        bool
	CreatedAt       time.Time
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
