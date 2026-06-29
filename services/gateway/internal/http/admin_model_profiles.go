package httpapi

import (
	"bytes"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/gateway/internal/middleware"
	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/gateway/internal/response"
)

type modelProfileProxy struct {
	baseURL      string
	serviceToken string
	client       *http.Client
}

func newModelProfileProxy(baseURL string, serviceToken string, timeoutClient *http.Client) *modelProfileProxy {
	if timeoutClient == nil {
		timeoutClient = http.DefaultClient
	}
	return &modelProfileProxy{
		baseURL:      strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		serviceToken: serviceToken,
		client:       timeoutClient,
	}
}

func (s *Server) handleAdminModelProfiles(w http.ResponseWriter, r *http.Request) {
	if !authorizeModelProfileAdmin(w, r) {
		return
	}
	if s.modelProfiles == nil || s.modelProfiles.baseURL == "" || strings.TrimSpace(s.modelProfiles.serviceToken) == "" {
		response.WriteError(w, http.StatusBadGateway, response.ErrorDetail{
			Code:      response.CodeDependency,
			Message:   "ai-gateway backend is not configured",
			RequestID: middleware.RequestIDFromContext(r.Context()),
		})
		return
	}
	targetPath := "/internal/v1/model-profiles"
	if profileID := strings.TrimSpace(r.PathValue("profileId")); profileID != "" {
		targetPath += "/" + url.PathEscape(profileID)
	}
	s.modelProfiles.proxy(w, r, targetPath)
}

func authorizeModelProfileAdmin(w http.ResponseWriter, r *http.Request) bool {
	requestID := middleware.RequestIDFromContext(r.Context())
	if strings.TrimSpace(r.Header.Get("X-User-Id")) == "" {
		response.WriteError(w, http.StatusUnauthorized, response.ErrorDetail{
			Code:      response.CodeUnauthorized,
			Message:   "authentication is required",
			RequestID: requestID,
		})
		return false
	}
	required := "admin:model-profiles:read"
	if r.Method != http.MethodGet {
		required = "admin:model-profiles:write"
	}
	if !hasPermission(r.Header.Get("X-User-Permissions"), required) {
		response.WriteError(w, http.StatusForbidden, response.ErrorDetail{
			Code:      response.CodeForbidden,
			Message:   "admin model profile permission is required",
			RequestID: requestID,
		})
		return false
	}
	return true
}

func hasPermission(raw string, required string) bool {
	for _, part := range strings.Split(raw, ",") {
		permission := strings.TrimSpace(part)
		if permission == required || permission == "admin:model-profiles:*" || permission == "admin:*" {
			return true
		}
	}
	return false
}

func (p *modelProfileProxy) proxy(w http.ResponseWriter, r *http.Request, targetPath string) {
	var body io.Reader
	if r.Body != nil {
		defer r.Body.Close()
		payload, err := io.ReadAll(r.Body)
		if err != nil {
			response.WriteError(w, http.StatusBadRequest, response.ErrorDetail{
				Code:      response.CodeValidation,
				Message:   "request body is invalid",
				RequestID: middleware.RequestIDFromContext(r.Context()),
			})
			return
		}
		body = bytes.NewReader(payload)
	}
	target := p.baseURL + targetPath
	if r.URL.RawQuery != "" {
		target += "?" + r.URL.RawQuery
	}
	req, err := http.NewRequestWithContext(r.Context(), r.Method, target, body)
	if err != nil {
		response.WriteError(w, http.StatusBadGateway, response.ErrorDetail{
			Code:      response.CodeDependency,
			Message:   "ai-gateway request could not be created",
			RequestID: middleware.RequestIDFromContext(r.Context()),
		})
		return
	}
	if contentType := r.Header.Get("Content-Type"); contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Request-Id", middleware.RequestIDFromContext(r.Context()))
	req.Header.Set("X-Caller-Service", "gateway")
	req.Header.Set("X-Service-Token", p.serviceToken)
	if userID := strings.TrimSpace(r.Header.Get("X-User-Id")); userID != "" {
		req.Header.Set("X-User-Id", userID)
	}

	res, err := p.client.Do(req)
	if err != nil {
		response.WriteError(w, http.StatusBadGateway, response.ErrorDetail{
			Code:      response.CodeDependency,
			Message:   "ai-gateway backend request failed",
			RequestID: middleware.RequestIDFromContext(r.Context()),
		})
		return
	}
	defer res.Body.Close()

	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		writeModelProfileProxyError(w, r, res.StatusCode)
		return
	}
	for _, header := range []string{"Content-Type", "X-Request-Id"} {
		if value := res.Header.Get(header); value != "" {
			w.Header().Set(header, value)
		}
	}
	w.WriteHeader(res.StatusCode)
	_, _ = io.Copy(w, res.Body)
}

func writeModelProfileProxyError(w http.ResponseWriter, r *http.Request, downstreamStatus int) {
	status := downstreamStatus
	code := response.CodeDependency
	message := "ai-gateway backend request failed"

	switch downstreamStatus {
	case http.StatusBadRequest:
		code = response.CodeValidation
		message = "request validation failed"
	case http.StatusNotFound:
		code = response.CodeNotFound
		message = "model profile not found"
	case http.StatusConflict:
		code = response.CodeConflict
		message = "model profile state conflict"
	case http.StatusTooManyRequests:
		code = response.CodeRateLimited
		message = "request rate limit exceeded"
	case http.StatusUnauthorized, http.StatusForbidden:
		status = http.StatusBadGateway
		message = "ai-gateway backend rejected gateway credentials"
	default:
		if downstreamStatus >= http.StatusInternalServerError {
			status = http.StatusBadGateway
		}
	}

	response.WriteError(w, status, response.ErrorDetail{
		Code:      code,
		Message:   message,
		RequestID: middleware.RequestIDFromContext(r.Context()),
	})
}
