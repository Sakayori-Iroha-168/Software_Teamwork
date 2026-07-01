package parserclient

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/qa/internal/service"
)

type Client struct {
	endpoint string
	token    string
	http     *http.Client
}

func New(baseURL, serviceToken string, timeout time.Duration) (*Client, error) {
	parsed, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return nil, errors.New("parser service URL must be absolute http(s)")
	}
	if strings.TrimSpace(serviceToken) == "" {
		return nil, errors.New("service token is required")
	}
	if timeout <= 0 {
		return nil, errors.New("parser client timeout must be positive")
	}
	return &Client{endpoint: strings.TrimRight(parsed.String(), "/") + "/internal/v1/parsed-documents", token: serviceToken, http: &http.Client{
		Timeout:       timeout,
		CheckRedirect: rejectRedirect,
	}}, nil
}

func (c *Client) Parse(ctx context.Context, input service.ParseDocumentInput) (service.ParsedDocument, error) {
	payload := map[string]any{
		"documentName": input.DocumentName,
		"contentType":  input.ContentType,
		"sizeBytes":    input.SizeBytes,
		"dataBase64":   base64.StdEncoding.EncodeToString(input.Data),
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return service.ParsedDocument{}, fmt.Errorf("encode parser request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		return service.ParsedDocument{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Service-Token", c.token)
	req.Header.Set("X-Caller-Service", "qa")
	if requestID := service.RequestIDFromContext(ctx); requestID != "" {
		req.Header.Set("X-Request-Id", requestID)
	}
	if userID := service.UserIDFromContext(ctx); userID != "" {
		req.Header.Set("X-User-Id", userID)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return service.ParsedDocument{}, service.NewError(service.CodeDependency, "attachment parsing failed", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
		return service.ParsedDocument{}, service.NewError(service.CodeDependency, "attachment parsing failed", fmt.Errorf("parser returned HTTP %d", resp.StatusCode))
	}
	var decoded struct {
		Data service.ParsedDocument `json:"data"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 10<<20)).Decode(&decoded); err != nil {
		return service.ParsedDocument{}, service.NewError(service.CodeDependency, "attachment parsing failed", fmt.Errorf("decode parser response: %w", err))
	}
	if strings.TrimSpace(decoded.Data.Content) == "" && len(decoded.Data.Pages) == 0 {
		return service.ParsedDocument{}, service.NewError(service.CodeValidation, "attachment did not contain readable text", nil)
	}
	return decoded.Data, nil
}

func rejectRedirect(*http.Request, []*http.Request) error {
	return http.ErrUseLastResponse
}
