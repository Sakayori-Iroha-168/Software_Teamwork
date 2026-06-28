package domain

import "time"

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

type ContentBlockVisibility string

const (
	ContentBlockVisibilityPublic   ContentBlockVisibility = "public"
	ContentBlockVisibilityInternal ContentBlockVisibility = "internal"
)

type MessageContentBlock struct {
	ID         int64
	MessageID  string
	BlockType  string
	Content    string
	Visibility ContentBlockVisibility
	SortOrder  int
	CreatedAt  time.Time
}
