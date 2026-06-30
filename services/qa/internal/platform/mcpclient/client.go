package mcpclient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/qa/internal/platform/httpclient"
	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/qa/internal/service/agent"
)

const (
	TransportStdio          = "stdio"
	TransportStreamableHTTP = "streamable_http"
)

type Config struct {
	Transport   string
	Command     string
	Args        []string
	Endpoint    string
	Token       string
	TokenHeader string
}

type Client struct {
	session *mcp.ClientSession
}

func Connect(ctx context.Context, cfg Config) (*Client, error) {
	transport, err := buildTransport(cfg)
	if err != nil {
		return nil, err
	}
	client := mcp.NewClient(&mcp.Implementation{Name: "qa-agent", Version: "0.1.0"}, nil)
	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		return nil, fmt.Errorf("initialize MCP session: %w", err)
	}
	return &Client{session: session}, nil
}

func buildTransport(cfg Config) (mcp.Transport, error) {
	switch cfg.Transport {
	case TransportStdio:
		if strings.TrimSpace(cfg.Command) == "" {
			return nil, errors.New("MCP stdio command is required")
		}
		command := exec.Command(cfg.Command, cfg.Args...)
		// MCP reserves stdout for JSON-RPC. Child diagnostics belong on stderr.
		command.Stderr = os.Stderr
		return &mcp.CommandTransport{Command: command}, nil
	case TransportStreamableHTTP:
		if strings.TrimSpace(cfg.Endpoint) == "" {
			return nil, errors.New("MCP HTTP endpoint is required")
		}
		base := &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   10 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			ForceAttemptHTTP2:     true,
			ResponseHeaderTimeout: 30 * time.Second,
			IdleConnTimeout:       90 * time.Second,
		}
		client := &http.Client{Transport: httpclient.HeaderTransport{
			Base:   base,
			Header: cfg.TokenHeader,
			Token:  cfg.Token,
		}}
		return &mcp.StreamableClientTransport{
			Endpoint:   cfg.Endpoint,
			HTTPClient: client,
			MaxRetries: 2,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported MCP transport %q", cfg.Transport)
	}
}

func (c *Client) Close() error {
	if c == nil || c.session == nil {
		return nil
	}
	return c.session.Close()
}

func (c *Client) ListTools(ctx context.Context) ([]agent.ToolDefinition, error) {
	if c == nil || c.session == nil {
		return nil, errors.New("MCP client is not connected")
	}
	var tools []agent.ToolDefinition
	cursor := ""
	for {
		result, err := c.session.ListTools(ctx, &mcp.ListToolsParams{Cursor: cursor})
		if err != nil {
			return nil, fmt.Errorf("MCP tools/list: %w", err)
		}
		for _, tool := range result.Tools {
			if tool == nil {
				continue
			}
			parameters := tool.InputSchema
			if parameters == nil {
				parameters = map[string]any{"type": "object", "properties": map[string]any{}}
			}
			tools = append(tools, agent.ToolDefinition{
				Type: "function",
				Function: agent.FunctionTool{
					Name:        tool.Name,
					Description: tool.Description,
					Parameters:  parameters,
				},
			})
		}
		if result.NextCursor == "" {
			break
		}
		cursor = result.NextCursor
	}
	return tools, nil
}

func (c *Client) CallTool(ctx context.Context, name string, arguments json.RawMessage) (agent.ToolResult, error) {
	if c == nil || c.session == nil {
		return agent.ToolResult{}, errors.New("MCP client is not connected")
	}
	var decoded map[string]any
	if len(arguments) == 0 {
		decoded = map[string]any{}
	} else if err := json.Unmarshal(arguments, &decoded); err != nil {
		return agent.ToolResult{}, fmt.Errorf("decode MCP tool arguments: %w", err)
	}
	result, err := c.session.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: decoded})
	if err != nil {
		return agent.ToolResult{}, fmt.Errorf("MCP tools/call %q: %w", name, err)
	}
	content, err := normalizeResult(result)
	if err != nil {
		return agent.ToolResult{}, err
	}
	citations := extractCitations(result)
	return agent.ToolResult{Content: content, IsError: result.IsError, Citations: citations}, nil
}

// extractCitations attempts to parse citation-worthy data from the structured
// content of a knowledge-search MCP tool result. It is intentionally tolerant
// — if the result shape does not match expectations the returned slice is
// simply empty. It never exposes internal file IDs, object keys, raw vectors,
// or provider secrets.
func extractCitations(result *mcp.CallToolResult) []agent.CitationData {
	if result == nil || result.StructuredContent == nil {
		return nil
	}
	raw, ok := result.StructuredContent.(map[string]any)
	if !ok {
		return nil
	}
	// The knowledge service wraps results under "data"."results".
	data, _ := raw["data"].(map[string]any)
	if data == nil {
		data = raw
	}
	results, _ := data["results"].([]any)
	if len(results) == 0 {
		return nil
	}
	citations := make([]agent.CitationData, 0, len(results))
	for i, item := range results {
		obj, ok := item.(map[string]any)
		if !ok {
			continue
		}
		citation := agent.CitationData{
			CitationNo: i + 1,
			Metadata:   map[string]any{},
		}
		if v := stringField(obj, "knowledgeBaseId"); v != "" {
			citation.ExternalKbID = v
		}
		if v := stringField(obj, "documentId"); v != "" {
			citation.ExternalDocID = v
		}
		if v := stringField(obj, "chunkId"); v != "" {
			citation.ExternalChunkID = v
		}
		if v := stringField(obj, "documentName"); v != "" {
			citation.DocName = v
		}
		if v := stringField(obj, "sectionPath"); v != "" {
			citation.SectionPath = v
		}
		if v := stringField(obj, "contentPreview"); v != "" {
			citation.QuoteText = v
		}
		if v := stringField(obj, "context"); v != "" {
			citation.Context = v
		}
		if v := stringField(obj, "chunkType"); v != "" {
			citation.ChunkType = v
		}
		if v, ok := obj["score"].(float64); ok {
			sc := v
			citation.Score = &sc
		}
		if v, ok := obj["rerankScore"].(float64); ok {
			rs := v
			citation.RerankScore = &rs
		}
		if v, ok := obj["pageNumber"].(float64); ok {
			pn := int(v)
			citation.PageNumber = &pn
		}
		for k, v := range obj {
			if isStandardCitationField(k) {
				continue
			}
			citation.Metadata[k] = v
		}
		citations = append(citations, citation)
	}
	return citations
}

func stringField(obj map[string]any, key string) string {
	v, ok := obj[key].(string)
	if !ok {
		return ""
	}
	return v
}

func isStandardCitationField(key string) bool {
	switch key {
	case "knowledgeBaseId", "documentId", "chunkId", "documentName",
		"sectionPath", "contentPreview", "context", "chunkType",
		"score", "rerankScore", "pageNumber":
		return true
	}
	return false
}

func normalizeResult(result *mcp.CallToolResult) (string, error) {
	if result == nil {
		return "", errors.New("MCP server returned an empty tool result")
	}
	if result.StructuredContent != nil {
		payload, err := json.Marshal(result.StructuredContent)
		if err != nil {
			return "", fmt.Errorf("encode structured MCP result: %w", err)
		}
		return string(payload), nil
	}
	parts := make([]string, 0, len(result.Content))
	for _, item := range result.Content {
		switch value := item.(type) {
		case *mcp.TextContent:
			parts = append(parts, value.Text)
		default:
			payload, err := json.Marshal(value)
			if err != nil {
				return "", fmt.Errorf("encode MCP content: %w", err)
			}
			parts = append(parts, string(payload))
		}
	}
	if len(parts) == 0 {
		return "{}", nil
	}
	return strings.Join(parts, "\n"), nil
}
