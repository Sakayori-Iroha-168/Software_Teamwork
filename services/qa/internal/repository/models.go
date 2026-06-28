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
