package service

type ChatRequest struct {
	ConversationID string     `json:"conversation_id"`
	Message        string     `json:"message"`
	KnowledgeBases []string   `json:"knowledge_bases,omitempty"`
	Params         ChatParams `json:"params,omitempty"`
	UserID         string     `json:"user_id,omitempty"`
	TraceID        string     `json:"trace_id,omitempty"`
	StreamMode     string     `json:"stream_mode,omitempty"`
}

type ChatParams struct {
	TopK                int      `json:"top_k,omitempty"`
	SimilarityThreshold *float64 `json:"similarity_threshold,omitempty"`
	UseRerank           *bool    `json:"use_rerank,omitempty"`
	RerankThreshold     *float64 `json:"rerank_threshold,omitempty"`
}

type StreamEvent struct {
	Event string
	Data  any
}

type IntentType string

const (
	IntentKnowledgeQA IntentType = "knowledge_qa"
	IntentGeneralChat IntentType = "general_chat"
)

type Route string

const (
	RouteKnowledge Route = "external_rag"
	RouteGeneral   Route = "general_chat"
)
