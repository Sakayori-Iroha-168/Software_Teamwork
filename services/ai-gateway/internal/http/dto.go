package httpapi

import (
	"encoding/json"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/ai-gateway/internal/service"
)

type createModelProfileRequest struct {
	Name              string                `json:"name"`
	Purpose           service.ModelPurpose  `json:"purpose"`
	Provider          service.ModelProvider `json:"provider"`
	BaseURL           string                `json:"baseUrl"`
	Model             string                `json:"model"`
	APIKey            string                `json:"apiKey"`
	Enabled           *bool                 `json:"enabled"`
	IsDefault         *bool                 `json:"isDefault"`
	TimeoutMs         *int                  `json:"timeoutMs"`
	SupportsStreaming *bool                 `json:"supportsStreaming"`
	Dimensions        *int                  `json:"dimensions"`
	TopN              *int                  `json:"topN"`
	DefaultParameters json.RawMessage       `json:"defaultParameters"`
}

type updateModelProfileRequest struct {
	Name              *string                `json:"name"`
	Provider          *service.ModelProvider `json:"provider"`
	BaseURL           *string                `json:"baseUrl"`
	Model             *string                `json:"model"`
	APIKey            *string                `json:"apiKey"`
	Enabled           *bool                  `json:"enabled"`
	IsDefault         *bool                  `json:"isDefault"`
	TimeoutMs         *int                   `json:"timeoutMs"`
	SupportsStreaming *bool                  `json:"supportsStreaming"`
	Dimensions        *int                   `json:"dimensions"`
	TopN              *int                   `json:"topN"`
	DefaultParameters *json.RawMessage       `json:"defaultParameters"`
}
