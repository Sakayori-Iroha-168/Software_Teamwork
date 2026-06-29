package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/gateway/internal/service"
)

const defaultMaxRequestBytes = int64(1 << 20)

type Config struct {
	MaxRequestBytes int64
	Logger          *slog.Logger
	Ready           func(context.Context) error
}

type Server struct {
	auth            service.AuthClient
	sessions        service.SessionStore
	hasher          service.TokenHasher
	maxRequestBytes int64
	logger          *slog.Logger
	ready           func(context.Context) error
	mux             *http.ServeMux
}

func NewServer(auth service.AuthClient, sessions service.SessionStore, hasher service.TokenHasher, cfg Config) (*Server, error) {
	if auth == nil {
		return nil, errors.New("auth client is required")
	}
	if sessions == nil {
		return nil, errors.New("session store is required")
	}
	if cfg.MaxRequestBytes <= 0 {
		cfg.MaxRequestBytes = defaultMaxRequestBytes
	}
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	s := &Server{
		auth:            auth,
		sessions:        sessions,
		hasher:          hasher,
		maxRequestBytes: cfg.MaxRequestBytes,
		logger:          cfg.Logger,
		ready:           cfg.Ready,
		mux:             http.NewServeMux(),
	}
	s.routes()
	return s, nil
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /healthz", s.handleHealth)
	s.mux.HandleFunc("GET /readyz", s.handleReady)
	s.mux.HandleFunc("POST /api/v1/users", s.handleCreateUser)
	s.mux.HandleFunc("POST /api/v1/sessions", s.handleCreateSession)
	s.mux.HandleFunc("GET /api/v1/users/me", s.handleCurrentUser)
	s.mux.HandleFunc("DELETE /api/v1/sessions/current", s.handleDeleteCurrentSession)
	s.mux.HandleFunc("/", s.handleNotFound)
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	requestID := strings.TrimSpace(r.Header.Get("X-Request-Id"))
	if requestID == "" {
		requestID = newRequestID()
	}
	ctx := contextWithRequestID(r.Context(), requestID)
	r = r.WithContext(ctx)
	w.Header().Set("X-Request-Id", requestID)

	recorder := &statusRecorder{ResponseWriter: w}
	start := time.Now()
	defer func() {
		if recovered := recover(); recovered != nil {
			s.logger.ErrorContext(ctx, "http panic recovered", "service", "gateway", "request_id", requestID, "operation", "http_request")
			writeAppError(recorder, r, service.InternalError("internal server error", nil))
		}
		status := recorder.status
		if status == 0 {
			status = http.StatusOK
		}
		if status >= http.StatusInternalServerError {
			s.logger.ErrorContext(ctx, "http request failed", "service", "gateway", "request_id", requestID, "method", r.Method, "path", r.URL.Path, "status", status, "duration_ms", time.Since(start).Milliseconds())
		}
	}()

	s.mux.ServeHTTP(recorder, r)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"}, requestIDFromContext(r.Context()))
}

func (s *Server) handleReady(w http.ResponseWriter, r *http.Request) {
	if s.ready != nil {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := s.ready(ctx); err != nil {
			writeAppError(w, r, service.DependencyError("service is not ready", err))
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"}, requestIDFromContext(r.Context()))
}

func (s *Server) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	var payload createUserRequest
	if err := s.decodeJSON(w, r, &payload); err != nil {
		writeAppError(w, r, err)
		return
	}
	if strings.TrimSpace(payload.Username) == "" || strings.TrimSpace(payload.Password) == "" {
		writeAppError(w, r, service.ValidationError("request validation failed", map[string]string{"username": "is required", "password": "is required"}))
		return
	}
	identity, err := s.auth.CreateUser(r.Context(), requestIDFromContext(r.Context()), payload.Username, payload.Password)
	if err != nil {
		writeAppError(w, r, err)
		return
	}
	if err := s.cacheSession(r, identity); err != nil {
		writeAppError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, sessionIdentityToResponse(identity), requestIDFromContext(r.Context()))
}

func (s *Server) handleCreateSession(w http.ResponseWriter, r *http.Request) {
	var payload createSessionRequest
	if err := s.decodeJSON(w, r, &payload); err != nil {
		writeAppError(w, r, err)
		return
	}
	if strings.TrimSpace(payload.Username) == "" || strings.TrimSpace(payload.Password) == "" {
		writeAppError(w, r, service.ValidationError("request validation failed", map[string]string{"username": "is required", "password": "is required"}))
		return
	}
	identity, err := s.auth.CreateSession(r.Context(), requestIDFromContext(r.Context()), payload.Username, payload.Password)
	if err != nil {
		writeAppError(w, r, err)
		return
	}
	if err := s.cacheSession(r, identity); err != nil {
		writeAppError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, sessionIdentityToResponse(identity), requestIDFromContext(r.Context()))
}

func (s *Server) handleCurrentUser(w http.ResponseWriter, r *http.Request) {
	authCtx, _, ok := s.authContext(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, userSummaryToResponse(authCtx.UserSummary()), requestIDFromContext(r.Context()))
}

func (s *Server) handleDeleteCurrentSession(w http.ResponseWriter, r *http.Request) {
	authCtx, accessTokenHash, ok := s.authContext(w, r)
	if !ok {
		return
	}
	if err := s.auth.DeleteSession(r.Context(), requestIDFromContext(r.Context()), authCtx.SessionID); err != nil {
		writeAppError(w, r, err)
		return
	}
	if err := s.sessions.Delete(r.Context(), accessTokenHash); err != nil {
		writeAppError(w, r, service.DependencyError("session cache unavailable", err))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleNotFound(w http.ResponseWriter, r *http.Request) {
	writeAppError(w, r, service.NewError(service.CodeNotFound, "route not found", nil))
}

func (s *Server) cacheSession(r *http.Request, identity service.SessionIdentity) error {
	accessTokenHash, err := s.hasher.Hash(identity.Session.AccessToken)
	if err != nil {
		return service.InternalError("session token could not be cached", err)
	}
	entry, ttl, err := service.NewCacheEntry(identity, accessTokenHash, requestIDFromContext(r.Context()), time.Now().UTC())
	if err != nil {
		return service.DependencyError("auth returned invalid session", err)
	}
	if err := s.sessions.Save(r.Context(), entry, ttl); err != nil {
		return service.DependencyError("session cache unavailable", err)
	}
	return nil
}

func (s *Server) authContext(w http.ResponseWriter, r *http.Request) (service.AuthContext, string, bool) {
	token, ok := bearerToken(r.Header.Get("Authorization"))
	if !ok {
		writeAppError(w, r, service.UnauthorizedError())
		return service.AuthContext{}, "", false
	}
	accessTokenHash, err := s.hasher.Hash(token)
	if err != nil {
		writeAppError(w, r, service.UnauthorizedError())
		return service.AuthContext{}, "", false
	}
	entry, err := s.sessions.Get(r.Context(), accessTokenHash)
	if err != nil {
		if errors.Is(err, service.ErrSessionNotFound) || errors.Is(err, service.ErrMalformedSession) {
			writeAppError(w, r, service.UnauthorizedError())
			return service.AuthContext{}, "", false
		}
		writeAppError(w, r, service.DependencyError("session cache unavailable", err))
		return service.AuthContext{}, "", false
	}
	if err := entry.Validate(accessTokenHash, time.Now().UTC()); err != nil {
		writeAppError(w, r, service.UnauthorizedError())
		return service.AuthContext{}, "", false
	}
	return entry.AuthContext(requestIDFromContext(r.Context())), accessTokenHash, true
}

func bearerToken(value string) (string, bool) {
	parts := strings.Fields(value)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
		return "", false
	}
	return parts[1], true
}

func (s *Server) decodeJSON(w http.ResponseWriter, r *http.Request, target any) error {
	r.Body = http.MaxBytesReader(w, r.Body, s.maxRequestBytes)
	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return service.ValidationError("request validation failed", map[string]string{"body": "must be a valid JSON object"})
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return service.ValidationError("request validation failed", map[string]string{"body": "must contain only one JSON object"})
	}
	return nil
}

func sessionIdentityToResponse(identity service.SessionIdentity) sessionResponseData {
	return sessionResponseData{
		User: userSummaryToResponse(identity.User),
		Session: sessionSummaryResponse{
			SessionID:   identity.Session.SessionID,
			AccessToken: identity.Session.AccessToken,
			TokenType:   identity.Session.TokenType,
			ExpiresAt:   dateTime(identity.Session.ExpiresAt),
		},
	}
}

func userSummaryToResponse(user service.UserSummary) userSummaryResponse {
	return userSummaryResponse{
		ID:          user.ID,
		Username:    user.Username,
		Roles:       append([]string(nil), user.Roles...),
		Permissions: append([]string(nil), user.Permissions...),
	}
}

func newRequestID() string {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		return "req_" + strconv.FormatInt(time.Now().UnixNano(), 10)
	}
	return "req_" + hex.EncodeToString(bytes)
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	if r.status != 0 {
		return
	}
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *statusRecorder) Write(body []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	return r.ResponseWriter.Write(body)
}
