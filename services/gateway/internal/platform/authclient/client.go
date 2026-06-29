package authclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/gateway/internal/service"
)

type Config struct {
	ServiceToken string
	Timeout      time.Duration
	HTTPClient   *http.Client
}

type Client struct {
	baseURL      string
	serviceToken string
	httpClient   *http.Client
}

func New(baseURL string, cfg Config) *Client {
	if cfg.Timeout <= 0 {
		cfg.Timeout = 5 * time.Second
	}
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = &http.Client{Timeout: cfg.Timeout}
	}
	return &Client{
		baseURL:      strings.TrimRight(baseURL, "/"),
		serviceToken: cfg.ServiceToken,
		httpClient:   cfg.HTTPClient,
	}
}

func (c *Client) CreateUser(ctx context.Context, requestID string, username string, password string) (service.SessionIdentity, error) {
	return c.postSessionIdentity(ctx, requestID, "/internal/v1/users", map[string]string{
		"username": username,
		"password": password,
	})
}

func (c *Client) CreateSession(ctx context.Context, requestID string, username string, password string) (service.SessionIdentity, error) {
	return c.postSessionIdentity(ctx, requestID, "/internal/v1/sessions", map[string]string{
		"username": username,
		"password": password,
	})
}

func (c *Client) DeleteSession(ctx context.Context, requestID string, sessionID string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.baseURL+"/internal/v1/sessions/"+sessionID, nil)
	if err != nil {
		return service.InternalError("auth request could not be created", err)
	}
	c.addHeaders(req, requestID)
	res, err := c.httpClient.Do(req)
	if err != nil {
		return service.DependencyError("auth service unavailable", err)
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusNoContent {
		return nil
	}
	return errorFromResponse(res)
}

func (c *Client) postSessionIdentity(ctx context.Context, requestID string, path string, payload any) (service.SessionIdentity, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return service.SessionIdentity{}, service.InternalError("auth request could not be encoded", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return service.SessionIdentity{}, service.InternalError("auth request could not be created", err)
	}
	req.Header.Set("Content-Type", "application/json")
	c.addHeaders(req, requestID)
	res, err := c.httpClient.Do(req)
	if err != nil {
		return service.SessionIdentity{}, service.DependencyError("auth service unavailable", err)
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return service.SessionIdentity{}, errorFromResponse(res)
	}
	var envelope struct {
		Data service.SessionIdentity `json:"data"`
	}
	if err := json.NewDecoder(res.Body).Decode(&envelope); err != nil {
		return service.SessionIdentity{}, service.DependencyError("auth response was invalid", err)
	}
	return envelope.Data, nil
}

func (c *Client) addHeaders(req *http.Request, requestID string) {
	req.Header.Set("X-Request-Id", requestID)
	req.Header.Set("X-Caller-Service", "gateway")
	if c.serviceToken != "" {
		req.Header.Set("X-Service-Token", c.serviceToken)
	}
}

func errorFromResponse(res *http.Response) error {
	raw, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	var envelope struct {
		Error struct {
			Code    service.Code `json:"code"`
			Message string       `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(raw, &envelope); err == nil && envelope.Error.Code != "" {
		switch envelope.Error.Code {
		case service.CodeValidation, service.CodeUnauthorized, service.CodeForbidden, service.CodeConflict, service.CodeRateLimited:
			message := strings.TrimSpace(envelope.Error.Message)
			if message == "" {
				message = "request failed"
			}
			return service.NewError(envelope.Error.Code, message, nil)
		}
	}
	if res.StatusCode == http.StatusUnauthorized {
		return service.UnauthorizedError()
	}
	if res.StatusCode == http.StatusForbidden {
		return service.ForbiddenError("permission denied")
	}
	return service.DependencyError("auth service request failed", fmt.Errorf("status %d", res.StatusCode))
}
