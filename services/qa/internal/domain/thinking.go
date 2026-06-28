package domain

import "time"

type IntentType string

const (
	IntentKnowledgeQA   IntentType = "knowledge_qa"
	IntentGeneralChat   IntentType = "general_chat"
	IntentDocumentQuery IntentType = "document_query"
	IntentSystemCommand IntentType = "system_command"
)

type StepType string

const (
	StepTypeIntent     StepType = "intent"
	StepTypeRetrieval  StepType = "retrieval"
	StepTypeGeneration StepType = "generation"
	StepTypeVerify     StepType = "verify"
)

type StepStatus string

const (
	StepStatusPending StepStatus = "pending"
	StepStatusRunning StepStatus = "running"
	StepStatusDone    StepStatus = "done"
	StepStatusFailed  StepStatus = "failed"
	StepStatusStopped StepStatus = "stopped"
)

var stepOrder = map[StepType]int{
	StepTypeIntent:     1,
	StepTypeRetrieval:  2,
	StepTypeGeneration: 3,
	StepTypeVerify:     4,
}

func StepOrderFor(stepType StepType) int {
	return stepOrder[stepType]
}

type ThinkingStep struct {
	Type   StepType   `json:"type"`
	Label  string     `json:"label"`
	Status StepStatus `json:"status"`
	Detail string     `json:"detail,omitempty"`
}

type ProcessStepRecord struct {
	ResponseRunID string
	StepOrder     int
	StepType      StepType
	Label         string
	Detail        string
	Status        StepStatus
	StartedAt     time.Time
	FinishedAt    *time.Time
}

type MessageContentBlockStatus string

const (
	ContentBlockStatusStreaming MessageContentBlockStatus = "streaming"
	ContentBlockStatusCompleted MessageContentBlockStatus = "completed"
	ContentBlockStatusStopped   MessageContentBlockStatus = "stopped"
)

type MessageContentBlock struct {
	ID                string
	MessageID         string
	BlockOrder        int
	BlockType         string
	Content           string
	Status            MessageContentBlockStatus
	ProviderBlockID   string
	ProviderMetadata  map[string]any
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type StreamEventType string

const (
	StreamEventTypeIntent    StreamEventType = "intent"
	StreamEventTypeStep      StreamEventType = "step"
	StreamEventTypeToken     StreamEventType = "token"
	StreamEventTypeCitation  StreamEventType = "citation"
	StreamEventTypeDone      StreamEventType = "done"
	StreamEventTypeError     StreamEventType = "error"
)

type ResponseStreamEvent struct {
	ID            int64
	ResponseRunID string
	EventSeq      int
	EventType     StreamEventType
	Payload       map[string]any
	ExpiresAt     *time.Time
	CreatedAt     time.Time
}

type Citation struct {
	ID              string
	MessageID       string
	CitationNo      int
	CharStart       *int
	CharEnd         *int
	ExternalKBID    string
	ExternalDocID   string
	ExternalChunkID string
	DocName         string
	QuoteText       string
	Context         string
	PageNumber      *int
	Score           *float64
	Metadata        map[string]any
}
