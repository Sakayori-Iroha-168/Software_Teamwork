package service

import (
	"encoding/json"
	"time"
)

type ModelPurpose string

const (
	PurposeChat      ModelPurpose = "chat"
	PurposeEmbedding ModelPurpose = "embedding"
	PurposeRerank    ModelPurpose = "rerank"
)

type ModelProvider string

const (
	ProviderOpenAICompatible ModelProvider = "openai_compatible"
	ProviderSiliconFlow      ModelProvider = "siliconflow"
	ProviderLocalCompatible  ModelProvider = "local_compatible"
)

type ModelProfile struct {
	ID                string          `json:"id"`
	Name              string          `json:"name"`
	Purpose           ModelPurpose    `json:"purpose"`
	Provider          ModelProvider   `json:"provider"`
	BaseURL           string          `json:"baseUrl"`
	Model             string          `json:"model"`
	Enabled           bool            `json:"enabled"`
	IsDefault         bool            `json:"isDefault"`
	TimeoutMs         int             `json:"timeoutMs"`
	APIKeyConfigured  bool            `json:"apiKeyConfigured"`
	SupportsStreaming bool            `json:"supportsStreaming"`
	Dimensions        *int            `json:"dimensions,omitempty"`
	TopN              *int            `json:"topN,omitempty"`
	DefaultParameters json.RawMessage `json:"defaultParameters,omitempty"`
	CredentialID      *string         `json:"-"`
	CreatedByUserID   string          `json:"-"`
	UpdatedByUserID   string          `json:"-"`
	CreatedAt         time.Time       `json:"createdAt"`
	UpdatedAt         time.Time       `json:"updatedAt"`
	DeletedAt         *time.Time      `json:"-"`
}

type ProviderCredential struct {
	ID                   string
	ProfileID            string
	StorageMode          string
	Ciphertext           []byte
	EncryptionKeyVersion string
	FingerprintSHA256    string
	KeyLast4             string
	Status               string
	CreatedByUserID      string
	CreatedAt            time.Time
	RotatedAt            *time.Time
}

type ModelProfileRevision struct {
	ID                 string
	ProfileID          string
	RevisionNo         int
	ChangeType         string
	ChangedFieldsJSON  json.RawMessage
	BeforeSnapshotJSON json.RawMessage
	AfterSnapshotJSON  json.RawMessage
	ChangedByUserID    string
	CallerService      string
	RequestID          string
	CreatedAt          time.Time
}

type RequestContext struct {
	RequestID     string
	CallerService string
	UserID        string
	ServiceToken  string
}

type CreateModelProfileInput struct {
	Name              string
	Purpose           ModelPurpose
	Provider          ModelProvider
	BaseURL           string
	Model             string
	APIKey            string
	Enabled           *bool
	IsDefault         *bool
	TimeoutMs         *int
	SupportsStreaming *bool
	Dimensions        *int
	TopN              *int
	DefaultParameters json.RawMessage
}

type UpdateModelProfileInput struct {
	ID                string
	Name              *string
	Provider          *ModelProvider
	BaseURL           *string
	Model             *string
	APIKey            *string
	Enabled           *bool
	IsDefault         *bool
	TimeoutMs         *int
	SupportsStreaming *bool
	Dimensions        *int
	TopN              *int
	DefaultParameters *json.RawMessage
}

type ListFilter struct {
	Purpose *ModelPurpose
	Enabled *bool
}

type Readiness struct {
	Status string           `json:"status"`
	Checks []ReadinessCheck `json:"checks"`
}

type ReadinessCheck struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}
