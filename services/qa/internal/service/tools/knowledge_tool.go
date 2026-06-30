package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/qa/internal/service/agent"
)

const (
	ToolSearchKnowledge    = "search_knowledge"
	ToolGetCitationSource  = "get_citation_source"
	maxKnowledgeResultSize = 8192 // bytes
)

type RetrievalTestInput struct {
	Question         string
	KnowledgeBaseIDs []string
	Retrieval        RetrievalSettings
}

type RetrievalSettings struct {
	TopK           int
	ScoreThreshold float64
	EnableRerank   bool
}

type RetrievalTestResult struct {
	RankNo          int
	KnowledgeBaseID string
	DocumentID      string
	DocumentName    string
	ChunkID         string
	SectionPath     string
	ContentPreview  string
	Score           float64
	RerankScore     float64
	Metadata        map[string]any
}

type KnowledgeRetriever interface {
	Retrieve(context.Context, string, RetrievalTestInput) ([]RetrievalTestResult, error)
}

// KnowledgeToolClient adapts the knowledge service HTTP client into MCP tools
// that can be used by the agent loop.
type KnowledgeToolClient struct {
	retrievalClient KnowledgeRetriever
	timeout         time.Duration
}

type KnowledgeToolConfig struct {
	RetrievalClient KnowledgeRetriever
	Timeout         time.Duration
}

func NewKnowledgeToolClient(cfg KnowledgeToolConfig) (*KnowledgeToolClient, error) {
	if cfg.RetrievalClient == nil {
		return nil, errors.New("knowledge retriever is required")
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 10 * time.Second
	}
	return &KnowledgeToolClient{
		retrievalClient: cfg.RetrievalClient,
		timeout:         cfg.Timeout,
	}, nil
}

func (c *KnowledgeToolClient) ListTools(ctx context.Context) ([]agent.ToolDefinition, error) {
	return []agent.ToolDefinition{
		{
			Type: "function",
			Function: agent.FunctionTool{
				Name:        ToolSearchKnowledge,
				Description: "Search user-accessible knowledge bases for relevant information. Returns summarized results with citations.",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"query": map[string]any{
							"type":        "string",
							"description": "The search query text.",
						},
						"knowledge_base_ids": map[string]any{
							"type":        "array",
							"items":       map[string]any{"type": "string"},
							"description": "Optional list of knowledge base IDs to search. If empty, uses default configured bases.",
						},
						"top_k": map[string]any{
							"type":        "integer",
							"minimum":     1,
							"maximum":     20,
							"description": "Maximum number of results to return.",
						},
						"score_threshold": map[string]any{
							"type":        "number",
							"minimum":     0.0,
							"maximum":     1.0,
							"description": "Minimum relevance score threshold.",
						},
						"enable_rerank": map[string]any{
							"type":        "boolean",
							"description": "Whether to enable reranking for better relevance.",
						},
					},
					"required":             []string{"query"},
					"additionalProperties": false,
				},
			},
		},
		{
			Type: "function",
			Function: agent.FunctionTool{
				Name:        ToolGetCitationSource,
				Description: "Retrieve the source document and context for a citation referenced in the answer.",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"citation_id": map[string]any{
							"type":        "string",
							"description": "The citation ID to look up.",
						},
						"chunk_id": map[string]any{
							"type":        "string",
							"description": "Optional chunk ID if citation_id is not available.",
						},
					},
					"required":             []string{},
					"additionalProperties": false,
				},
			},
		},
	}, nil
}

func (c *KnowledgeToolClient) CallTool(ctx context.Context, name string, arguments json.RawMessage) (agent.ToolResult, error) {
	switch name {
	case ToolSearchKnowledge:
		return c.searchKnowledge(ctx, arguments)
	case ToolGetCitationSource:
		return c.getCitationSource(ctx, arguments)
	default:
		return agent.ToolResult{}, fmt.Errorf("knowledge tool %q is not registered", name)
	}
}

func (c *KnowledgeToolClient) searchKnowledge(ctx context.Context, arguments json.RawMessage) (agent.ToolResult, error) {
	var input struct {
		Query           string   `json:"query"`
		KnowledgeBaseIDs []string `json:"knowledge_base_ids"`
		TopK            int      `json:"top_k"`
		ScoreThreshold  float64  `json:"score_threshold"`
		EnableRerank    bool     `json:"enable_rerank"`
	}

	if err := decodeToolArguments(arguments, &input); err != nil {
		return toolFailure("invalid_arguments", err.Error()), nil
	}

	if strings.TrimSpace(input.Query) == "" {
		return toolFailure("invalid_arguments", "query must not be empty"), nil
	}

	// Get user ID from context
	userID := ctx.Value("userID").(string)
	if strings.TrimSpace(userID) == "" {
		return toolFailure("invalid_arguments", "user ID is required"), nil
	}

	// Apply timeout
	toolCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	// Build retrieval input
	retrievalInput := RetrievalTestInput{
		Question:          input.Query,
		KnowledgeBaseIDs:  input.KnowledgeBaseIDs,
		Retrieval: RetrievalSettings{
			TopK:            input.TopK,
			ScoreThreshold:  input.ScoreThreshold,
			EnableRerank:    input.EnableRerank,
		},
	}

	// Call knowledge service
	results, err := c.retrievalClient.Retrieve(toolCtx, userID, retrievalInput)
	if err != nil {
		return toolFailure("retrieval_failed", "knowledge retrieval service failed"), nil
	}

	// Generate sanitized summary
	summary := generateSearchSummary(results)
	
	// Truncate if too large
	if len(summary) > maxKnowledgeResultSize {
		summary = truncateUTF8(summary, maxKnowledgeResultSize)
	}

	return agent.ToolResult{Content: summary}, nil
}

func (c *KnowledgeToolClient) getCitationSource(ctx context.Context, arguments json.RawMessage) (agent.ToolResult, error) {
	var input struct {
		CitationID string `json:"citation_id"`
		ChunkID    string `json:"chunk_id"`
	}

	if err := decodeToolArguments(arguments, &input); err != nil {
		return toolFailure("invalid_arguments", err.Error()), nil
	}

	if strings.TrimSpace(input.CitationID) == "" && strings.TrimSpace(input.ChunkID) == "" {
		return toolFailure("invalid_arguments", "either citation_id or chunk_id must be provided"), nil
	}

	// TODO: Implement citation source lookup via knowledge service
	// This requires knowledge service to provide a citation/chunk lookup endpoint
	// For now, return a placeholder indicating the feature is pending implementation
	
	summary := map[string]any{
		"status":  "pending",
		"message": "Citation source lookup is pending knowledge service endpoint implementation",
		"hint":    "Use the citation information already embedded in the answer for now",
	}

	payload, _ := json.Marshal(summary)
	return agent.ToolResult{Content: string(payload)}, nil
}

func generateSearchSummary(results []RetrievalTestResult) string {
	if len(results) == 0 {
		return `{"hit_count": 0, "message": "No relevant results found"}`
	}

	summary := map[string]any{
		"hit_count": len(results),
		"results":   make([]map[string]any, 0, len(results)),
	}

	for i, result := range results {
		if i >= 10 {
			break
		}
		
		item := map[string]any{
			"rank":              result.RankNo,
			"score":             result.Score,
			"knowledge_base_id": result.KnowledgeBaseID,
			"document_id":       result.DocumentID,
			"document_name":     truncateString(result.DocumentName, 100),
			"chunk_id":          result.ChunkID,
			"section_path":      truncateString(result.SectionPath, 100),
			"preview":           truncateString(result.ContentPreview, 200),
			"context":           truncateString(result.ContentPreview, 500),
			"rerank_score":      result.RerankScore,
			"metadata":          result.Metadata,
		}
		
		summary["results"] = append(summary["results"].([]map[string]any), item)
	}

	payload, _ := json.Marshal(summary)
	return string(payload)
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func truncateUTF8(value string, maxBytes int) string {
	if len(value) <= maxBytes {
		return value
	}
	
	suffix := "\n...[result truncated]"
	limit := maxBytes - len(suffix)
	if limit <= 0 {
		return suffix[:maxBytes]
	}
	
	// Ensure we don't break UTF-8 characters
	for limit > 0 && (value[limit]&0xC0) == 0x80 {
		limit--
	}
	
	return value[:limit] + suffix
}

func decodeToolArguments(raw json.RawMessage, target any) error {
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return errors.New("arguments do not match the tool schema")
	}
	return nil
}

func toolFailure(code, message string) agent.ToolResult {
	payload, _ := json.Marshal(map[string]any{
		"error": map[string]string{
			"code":    code,
			"message": message,
		},
	})
	return agent.ToolResult{Content: string(payload), IsError: true}
}