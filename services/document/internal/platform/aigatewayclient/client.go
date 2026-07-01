package aigatewayclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/document/internal/service"
)

const (
	defaultTimeout   = 30 * time.Second
	maxResponseBytes = 2 << 20 // 2 MiB
	callerService    = "document"
)

// Client is a lightweight HTTP client for calling the AI Gateway chat completions
// endpoint. Unlike the aigateway package's ChatClient, it accepts any http(s) base
// URL and takes requestID / serviceToken per call rather than at construction time.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// New validates baseURL and returns a ready-to-use Client.
// Returns an error if baseURL is empty, a relative reference, uses a non-http(s)
// scheme, or has an empty host.
func New(baseURL string) (*Client, error) {
	trimmed := strings.TrimSpace(baseURL)
	if trimmed == "" {
		return nil, errors.New("aigatewayclient: baseURL must not be empty")
	}
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return nil, fmt.Errorf("aigatewayclient: invalid baseURL: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, errors.New("aigatewayclient: baseURL must use http or https scheme")
	}
	if parsed.Host == "" {
		return nil, errors.New("aigatewayclient: baseURL must have a non-empty host")
	}
	return &Client{
		baseURL:    strings.TrimRight(trimmed, "/"),
		httpClient: &http.Client{Timeout: defaultTimeout},
	}, nil
}

type chatRequest struct {
	Messages []chatMessage `json:"messages"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
}

// ChatCompletion sends messages to the AI Gateway chat completions endpoint and
// returns the first choice's content string.
//
// requestID is forwarded as X-Request-Id when non-empty.
// serviceToken is forwarded as X-Service-Token when non-empty.
// Non-2xx responses, invalid JSON, empty choices and whitespace-only content all
// map to a service.CodeDependency error; downstream body is never included in the
// returned error message.
func (c *Client) ChatCompletion(ctx context.Context, requestID, serviceToken string, messages []service.ChatMessage) (string, error) {
	reqMsgs := make([]chatMessage, len(messages))
	for i, m := range messages {
		reqMsgs[i] = chatMessage{Role: m.Role, Content: m.Content}
	}
	raw, err := json.Marshal(chatRequest{Messages: reqMsgs})
	if err != nil {
		return "", service.NewError(service.CodeInternal, "encode chat request", err)
	}
	endpoint := c.baseURL + "/internal/v1/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(raw))
	if err != nil {
		return "", service.NewError(service.CodeDependency, "build chat request", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Caller-Service", callerService)
	if id := strings.TrimSpace(requestID); id != "" {
		req.Header.Set("X-Request-Id", id)
	}
	if token := strings.TrimSpace(serviceToken); token != "" {
		req.Header.Set("X-Service-Token", token)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", service.NewError(service.CodeDependency, "chat request failed", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
		return "", service.NewError(service.CodeDependency, "chat request failed", fmt.Errorf("status %d", resp.StatusCode))
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	if err != nil {
		return "", service.NewError(service.CodeDependency, "read chat response", err)
	}
	if len(data) > maxResponseBytes {
		return "", service.NewError(service.CodeDependency, "chat response too large", nil)
	}
	var decoded chatResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		return "", service.NewError(service.CodeDependency, "decode chat response", err)
	}
	if len(decoded.Choices) == 0 {
		return "", service.NewError(service.CodeDependency, "chat response has no choices", nil)
	}
	content := decoded.Choices[0].Message.Content
	if strings.TrimSpace(content) == "" {
		return "", service.NewError(service.CodeDependency, "chat response content is empty", nil)
	}
	return content, nil
}
