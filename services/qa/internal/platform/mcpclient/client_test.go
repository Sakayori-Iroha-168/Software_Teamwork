package mcpclient

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/qa/internal/platform/mcpclient/testserver"
)

func TestMCPHelperProcess(t *testing.T) {
	if os.Getenv("QA_MCP_HELPER_PROCESS") != "1" {
		return
	}
	if err := testserver.EchoServer().Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		os.Exit(2)
	}
	os.Exit(0)
}

func TestStdioClientLifecycleAndToolCall(t *testing.T) {
	t.Setenv("QA_MCP_HELPER_PROCESS", "1")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client, err := Connect(ctx, Config{
		Transport: TransportStdio,
		Command:   os.Args[0],
		Args:      []string{"-test.run=TestMCPHelperProcess"},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	assertEchoClient(t, ctx, client)
}

func TestStreamableHTTPClientAddsTokenAndCallsTool(t *testing.T) {
	httpServer := testserver.StreamableHTTP(t, "mcp-token", "Authorization")
	defer httpServer.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client, err := Connect(ctx, Config{
		Transport:   TransportStreamableHTTP,
		Endpoint:    httpServer.URL,
		Token:       "mcp-token",
		TokenHeader: "Authorization",
	})
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	assertEchoClient(t, ctx, client)
}

func assertEchoClient(t *testing.T, ctx context.Context, client *Client) {
	t.Helper()
	tools, err := client.ListTools(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(tools) != 1 || tools[0].Function.Name != "echo" {
		t.Fatalf("unexpected tools: %+v", tools)
	}
	result, err := client.CallTool(ctx, "echo", json.RawMessage(`{"text":"hello"}`))
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError || !strings.Contains(result.Content, "hello") {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestNormalizeTextResult(t *testing.T) {
	got, err := normalizeResult(&mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "hello"}}})
	if err != nil {
		t.Fatal(err)
	}
	if got != "hello" {
		t.Fatalf("normalizeResult = %q", got)
	}
}

func TestExtractCitationsFromStructuredResult(t *testing.T) {
	results := []any{
		map[string]any{
			"knowledgeBaseId": "kb-1",
			"documentId":      "doc-1",
			"chunkId":         "chunk-1",
			"documentName":    "Test.pdf",
			"sectionPath":     "/Ch1/S2",
			"contentPreview":  "Some text content",
			"context":         "Broader context",
			"chunkType":       "text",
			"score":           0.95,
			"rerankScore":     0.91,
			"pageNumber":      3.0,
			"customField":     "extra",
		},
	}
	result := &mcp.CallToolResult{
		StructuredContent: map[string]any{
			"data": map[string]any{
				"results": results,
			},
		},
	}
	citations := extractCitations(result)
	if len(citations) != 1 {
		t.Fatalf("expected 1 citation, got %d", len(citations))
	}
	c := citations[0]
	if c.CitationNo != 1 || c.ExternalDocID != "doc-1" || c.DocName != "Test.pdf" ||
		c.ExternalKbID != "kb-1" || c.ExternalChunkID != "chunk-1" ||
		c.SectionPath != "/Ch1/S2" || c.QuoteText != "Some text content" ||
		c.Context != "Broader context" || c.ChunkType != "text" {
		t.Fatalf("unexpected citation fields: %+v", c)
	}
	if c.Score == nil || *c.Score != 0.95 {
		t.Fatal("expected score 0.95")
	}
	if c.RerankScore == nil || *c.RerankScore != 0.91 {
		t.Fatal("expected rerankScore 0.91")
	}
	if c.PageNumber == nil || *c.PageNumber != 3 {
		t.Fatal("expected pageNumber 3")
	}
	if v, ok := c.Metadata["customField"]; !ok || v != "extra" {
		t.Fatalf("expected customField in metadata, got %v", c.Metadata)
	}
}

func TestExtractCitationsHandlesMissingFields(t *testing.T) {
	result := &mcp.CallToolResult{
		StructuredContent: map[string]any{
			"data": map[string]any{
				"results": []any{
					map[string]any{
						"documentId": "doc-minimal",
						"score":      0.5,
					},
				},
			},
		},
	}
	citations := extractCitations(result)
	if len(citations) != 1 {
		t.Fatalf("expected 1 citation, got %d", len(citations))
	}
	c := citations[0]
	if c.ExternalDocID != "doc-minimal" || c.DocName != "" {
		t.Fatalf("unexpected fields: %+v", c)
	}
}

func TestExtractCitationsEmptyInputs(t *testing.T) {
	if citations := extractCitations(nil); len(citations) != 0 {
		t.Fatalf("nil result should return empty, got %d", len(citations))
	}
	if citations := extractCitations(&mcp.CallToolResult{}); len(citations) != 0 {
		t.Fatalf("empty result should return empty, got %d", len(citations))
	}
	if citations := extractCitations(&mcp.CallToolResult{StructuredContent: "not an object"}); len(citations) != 0 {
		t.Fatalf("non-map structured content should return empty, got %d", len(citations))
	}
	if citations := extractCitations(&mcp.CallToolResult{StructuredContent: map[string]any{"data": map[string]any{"results": []any{}}}}); len(citations) != 0 {
		t.Fatalf("empty results should return empty, got %d", len(citations))
	}
}

func TestExtractCitationsPreservesExtraMetadata(t *testing.T) {
	result := &mcp.CallToolResult{
		StructuredContent: map[string]any{
			"results": []any{
				map[string]any{
					"documentId":  "doc-extra",
					"score":       0.88,
					"customTag":   "important",
					"authorField": "John Doe",
				},
			},
		},
	}
	citations := extractCitations(result)
	if len(citations) != 1 {
		t.Fatalf("expected 1 citation, got %d", len(citations))
	}
	c := citations[0]
	if len(c.Metadata) < 2 {
		t.Fatalf("expected metadata with custom fields, got %v", c.Metadata)
	}
	if c.Metadata["customTag"] != "important" || c.Metadata["authorField"] != "John Doe" {
		t.Fatalf("unexpected metadata: %v", c.Metadata)
	}
}
