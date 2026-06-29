package httpapi

import (
	"bytes"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
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

type adminAuthenticator struct {
	tokenHashes [][]byte
	userID      string
	permissions []string
}

type adminIdentity struct {
	userID      string
	permissions []string
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

func newAdminAuthenticator(hexHashes []string, userID string, permissions []string) *adminAuthenticator {
	auth := &adminAuthenticator{
		userID:      strings.TrimSpace(userID),
		permissions: normalizePermissions(permissions),
	}
	for _, raw := range hexHashes {
		decoded, err := hex.DecodeString(strings.TrimSpace(raw))
		if err == nil && len(decoded) == sha256.Size {
			auth.tokenHashes = append(auth.tokenHashes, decoded)
		}
	}
	return auth
}

func (s *Server) handleAdminModelProfiles(w http.ResponseWriter, r *http.Request) {
	identity, ok := s.authorizeModelProfileAdmin(w, r)
	if !ok {
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
	s.modelProfiles.proxy(w, r, targetPath, identity)
}

func (s *Server) authorizeModelProfileAdmin(w http.ResponseWriter, r *http.Request) (adminIdentity, bool) {
	requestID := middleware.RequestIDFromContext(r.Context())
	identity, authenticated := s.adminAuth.authenticate(r)
	if !authenticated {
		response.WriteError(w, http.StatusUnauthorized, response.ErrorDetail{
			Code:      response.CodeUnauthorized,
			Message:   "authentication is required",
			RequestID: requestID,
		})
		return adminIdentity{}, false
	}
	required := "admin:model-profiles:read"
	if r.Method != http.MethodGet {
		required = "admin:model-profiles:write"
	}
	if !hasPermission(identity.permissions, required) {
		response.WriteError(w, http.StatusForbidden, response.ErrorDetail{
			Code:      response.CodeForbidden,
			Message:   "admin model profile permission is required",
			RequestID: requestID,
		})
		return adminIdentity{}, false
	}
	return identity, true
}

func (a *adminAuthenticator) authenticate(r *http.Request) (adminIdentity, bool) {
	if a == nil || len(a.tokenHashes) == 0 {
		return adminIdentity{}, false
	}
	token := bearerToken(r.Header.Get("Authorization"))
	if token == "" {
		return adminIdentity{}, false
	}
	sum := sha256.Sum256([]byte(token))
	for _, hash := range a.tokenHashes {
		if subtle.ConstantTimeCompare(sum[:], hash) == 1 {
			return adminIdentity{userID: a.userID, permissions: a.permissions}, true
		}
	}
	return adminIdentity{}, false
}

func bearerToken(header string) string {
	parts := strings.Fields(strings.TrimSpace(header))
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

func hasPermission(permissions []string, required string) bool {
	for _, permission := range permissions {
		if permission == required || permission == "admin:model-profiles:*" || permission == "admin:*" {
			return true
		}
	}
	return false
}

func normalizePermissions(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		permission := strings.TrimSpace(value)
		if permission != "" {
			result = append(result, permission)
		}
	}
	return result
}

func (p *modelProfileProxy) proxy(w http.ResponseWriter, r *http.Request, targetPath string, identity adminIdentity) {
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
	if identity.userID != "" {
		req.Header.Set("X-User-Id", identity.userID)
	}
	if len(identity.permissions) > 0 {
		req.Header.Set("X-User-Permissions", strings.Join(identity.permissions, ","))
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
