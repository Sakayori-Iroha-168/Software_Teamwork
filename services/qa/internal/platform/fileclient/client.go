package fileclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/qa/internal/service"
)

type Client struct {
	baseURL string
	token   string
	http    *http.Client
}

func New(baseURL, serviceToken string, timeout time.Duration) (*Client, error) {
	parsed, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return nil, errors.New("file service URL must be absolute http(s)")
	}
	if strings.TrimSpace(serviceToken) == "" {
		return nil, errors.New("service token is required")
	}
	if timeout <= 0 {
		return nil, errors.New("file client timeout must be positive")
	}
	return &Client{baseURL: strings.TrimRight(parsed.String(), "/"), token: serviceToken, http: &http.Client{
		Timeout:       timeout,
		CheckRedirect: rejectRedirect,
	}}, nil
}

func (c *Client) Upload(ctx context.Context, input service.FileUploadInput) (service.FileObject, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", input.Filename)
	if err != nil {
		return service.FileObject{}, fmt.Errorf("create multipart file: %w", err)
	}
	if _, err := io.Copy(part, input.Body); err != nil {
		return service.FileObject{}, fmt.Errorf("copy file content: %w", err)
	}
	if err := writer.Close(); err != nil {
		return service.FileObject{}, fmt.Errorf("close multipart body: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/internal/v1/files", &body)
	if err != nil {
		return service.FileObject{}, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	c.setHeaders(ctx, req, "")
	resp, err := c.http.Do(req)
	if err != nil {
		return service.FileObject{}, service.NewError(service.CodeDependency, "file upload failed", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
		return service.FileObject{}, service.NewError(service.CodeDependency, "file upload failed", fmt.Errorf("file service returned HTTP %d", resp.StatusCode))
	}
	var decoded struct {
		Data service.FileObject `json:"data"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&decoded); err != nil {
		return service.FileObject{}, fmt.Errorf("decode file upload response: %w", err)
	}
	if strings.TrimSpace(decoded.Data.ID) == "" {
		return service.FileObject{}, service.NewError(service.CodeDependency, "file upload failed", errors.New("missing file id"))
	}
	return decoded.Data, nil
}

func (c *Client) Read(ctx context.Context, fileRef string) ([]byte, error) {
	endpoint := c.baseURL + "/internal/v1/files/" + url.PathEscape(fileRef) + "/content"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	c.setHeaders(ctx, req, "")
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, service.NewError(service.CodeDependency, "file content could not be read", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
		return nil, service.NewError(service.CodeDependency, "file content could not be read", fmt.Errorf("file service returned HTTP %d", resp.StatusCode))
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 100<<20))
	if err != nil {
		return nil, fmt.Errorf("read file content: %w", err)
	}
	return data, nil
}

func (c *Client) Delete(ctx context.Context, fileRef string) error {
	endpoint := c.baseURL + "/internal/v1/files/" + url.PathEscape(fileRef)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, endpoint, nil)
	if err != nil {
		return err
	}
	c.setHeaders(ctx, req, "")
	resp, err := c.http.Do(req)
	if err != nil {
		return service.NewError(service.CodeDependency, "file delete request failed", err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
	if resp.StatusCode == http.StatusNotFound {
		return nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return service.NewError(service.CodeDependency, "file delete request failed", fmt.Errorf("file service returned HTTP %d", resp.StatusCode))
	}
	return nil
}

func (c *Client) setHeaders(ctx context.Context, req *http.Request, userID string) {
	req.Header.Set("X-Service-Token", c.token)
	req.Header.Set("X-Caller-Service", "qa")
	if requestID := service.RequestIDFromContext(ctx); requestID != "" {
		req.Header.Set("X-Request-Id", requestID)
	}
	if userID != "" {
		req.Header.Set("X-User-Id", userID)
	}
}

func rejectRedirect(*http.Request, []*http.Request) error {
	return http.ErrUseLastResponse
}
