package httpapi

import (
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

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/ai-gateway/internal/middleware"
	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/ai-gateway/internal/service"
)

const defaultMaxRequestBytes = int64(2 << 20)

type Config struct {
	Logger          *slog.Logger
	MaxRequestBytes int64
	Authenticator   *middleware.Authenticator
}

type Server struct {
	profiles        *service.Service
	logger          *slog.Logger
	maxRequestBytes int64
	authenticator   *middleware.Authenticator
	mux             *http.ServeMux
}

func NewServer(profiles *service.Service, cfg Config) *Server {
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	if cfg.MaxRequestBytes <= 0 {
		cfg.MaxRequestBytes = defaultMaxRequestBytes
	}
	s := &Server{
		profiles:        profiles,
		logger:          cfg.Logger,
		maxRequestBytes: cfg.MaxRequestBytes,
		authenticator:   cfg.Authenticator,
		mux:             http.NewServeMux(),
	}
	s.routes()
	return s
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /healthz", s.handleHealth)
	s.mux.HandleFunc("GET /readyz", s.handleReady)
	s.mux.HandleFunc("GET /internal/v1/model-profiles", s.handleListProfiles)
	s.mux.HandleFunc("POST /internal/v1/model-profiles", s.handleCreateProfile)
	s.mux.HandleFunc("GET /internal/v1/model-profiles/{profileId}", s.handleGetProfile)
	s.mux.HandleFunc("PATCH /internal/v1/model-profiles/{profileId}", s.handleUpdateProfile)
	s.mux.HandleFunc("DELETE /internal/v1/model-profiles/{profileId}", s.handleDeleteProfile)
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
			s.logger.ErrorContext(ctx, "http panic recovered", "service", "ai-gateway", "request_id", requestID, "operation", "http_request")
			writeError(recorder, http.StatusInternalServerError, service.CodeInternal, "internal server error", requestID, nil)
		}
		status := recorder.status
		if status == 0 {
			status = http.StatusOK
		}
		if status >= http.StatusInternalServerError {
			s.logger.ErrorContext(ctx, "http request failed", "service", "ai-gateway", "request_id", requestID, "method", r.Method, "path", r.URL.Path, "status", status, "duration_ms", time.Since(start).Milliseconds())
		}
	}()
	s.mux.ServeHTTP(recorder, r)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"}, requestIDFromContext(r.Context()))
}

func (s *Server) handleReady(w http.ResponseWriter, r *http.Request) {
	readiness := s.profiles.Readiness(r.Context())
	status := http.StatusOK
	if readiness.Status != "ok" {
		status = http.StatusServiceUnavailable
	}
	writeJSON(w, status, readiness, requestIDFromContext(r.Context()))
}

func (s *Server) handleListProfiles(w http.ResponseWriter, r *http.Request) {
	reqCtx, ok := s.requestContext(w, r)
	if !ok {
		return
	}
	filter, ok := profileFilter(w, r)
	if !ok {
		return
	}
	profiles, err := s.profiles.ListProfiles(r.Context(), reqCtx, filter)
	if err != nil {
		writeAppError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, profiles, requestIDFromContext(r.Context()))
}

func (s *Server) handleCreateProfile(w http.ResponseWriter, r *http.Request) {
	reqCtx, ok := s.requestContext(w, r)
	if !ok {
		return
	}
	var payload createModelProfileRequest
	if !s.decodeJSON(w, r, &payload) {
		return
	}
	created, err := s.profiles.CreateProfile(r.Context(), reqCtx, service.CreateModelProfileInput{
		Name:              payload.Name,
		Purpose:           payload.Purpose,
		Provider:          payload.Provider,
		BaseURL:           payload.BaseURL,
		Model:             payload.Model,
		APIKey:            payload.APIKey,
		Enabled:           payload.Enabled,
		IsDefault:         payload.IsDefault,
		TimeoutMs:         payload.TimeoutMs,
		SupportsStreaming: payload.SupportsStreaming,
		Dimensions:        payload.Dimensions,
		TopN:              payload.TopN,
		DefaultParameters: payload.DefaultParameters,
	})
	if err != nil {
		writeAppError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, created, requestIDFromContext(r.Context()))
}

func (s *Server) handleGetProfile(w http.ResponseWriter, r *http.Request) {
	reqCtx, ok := s.requestContext(w, r)
	if !ok {
		return
	}
	profile, err := s.profiles.GetProfile(r.Context(), reqCtx, r.PathValue("profileId"))
	if err != nil {
		writeAppError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, profile, requestIDFromContext(r.Context()))
}

func (s *Server) handleUpdateProfile(w http.ResponseWriter, r *http.Request) {
	reqCtx, ok := s.requestContext(w, r)
	if !ok {
		return
	}
	var payload updateModelProfileRequest
	if !s.decodeJSON(w, r, &payload) {
		return
	}
	updated, err := s.profiles.UpdateProfile(r.Context(), reqCtx, service.UpdateModelProfileInput{
		ID:                r.PathValue("profileId"),
		Name:              payload.Name,
		Provider:          payload.Provider,
		BaseURL:           payload.BaseURL,
		Model:             payload.Model,
		APIKey:            payload.APIKey,
		Enabled:           payload.Enabled,
		IsDefault:         payload.IsDefault,
		TimeoutMs:         payload.TimeoutMs,
		SupportsStreaming: payload.SupportsStreaming,
		Dimensions:        payload.Dimensions,
		TopN:              payload.TopN,
		DefaultParameters: payload.DefaultParameters,
	})
	if err != nil {
		writeAppError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, updated, requestIDFromContext(r.Context()))
}

func (s *Server) handleDeleteProfile(w http.ResponseWriter, r *http.Request) {
	reqCtx, ok := s.requestContext(w, r)
	if !ok {
		return
	}
	if err := s.profiles.DeleteProfile(r.Context(), reqCtx, r.PathValue("profileId")); err != nil {
		writeAppError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleNotFound(w http.ResponseWriter, r *http.Request) {
	writeAppError(w, r, service.NotFoundError("route not found"))
}

func (s *Server) requestContext(w http.ResponseWriter, r *http.Request) (service.RequestContext, bool) {
	token := strings.TrimSpace(r.Header.Get("X-Service-Token"))
	if !s.authenticator.Verify(token) {
		writeAppError(w, r, service.UnauthorizedError())
		return service.RequestContext{}, false
	}
	caller := strings.TrimSpace(r.Header.Get("X-Caller-Service"))
	if caller == "" {
		writeAppError(w, r, service.UnauthorizedError())
		return service.RequestContext{}, false
	}
	return service.RequestContext{
		RequestID:     requestIDFromContext(r.Context()),
		CallerService: caller,
		UserID:        strings.TrimSpace(r.Header.Get("X-User-Id")),
		ServiceToken:  token,
	}, true
}

func (s *Server) decodeJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, s.maxRequestBytes)
	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		writeAppError(w, r, service.ValidationError("request validation failed", map[string]string{"body": "must be a valid JSON object"}))
		return false
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeAppError(w, r, service.ValidationError("request validation failed", map[string]string{"body": "must contain only one JSON object"}))
		return false
	}
	return true
}

func profileFilter(w http.ResponseWriter, r *http.Request) (service.ListFilter, bool) {
	filter := service.ListFilter{}
	if raw := strings.TrimSpace(r.URL.Query().Get("purpose")); raw != "" {
		purpose := service.ModelPurpose(raw)
		switch purpose {
		case service.PurposeChat, service.PurposeEmbedding, service.PurposeRerank:
			filter.Purpose = &purpose
		default:
			writeAppError(w, r, service.ValidationError("request validation failed", map[string]string{"purpose": "must be chat, embedding, or rerank"}))
			return service.ListFilter{}, false
		}
	}
	if raw := strings.TrimSpace(r.URL.Query().Get("enabled")); raw != "" {
		value, err := strconv.ParseBool(raw)
		if err != nil {
			writeAppError(w, r, service.ValidationError("request validation failed", map[string]string{"enabled": "must be a boolean"}))
			return service.ListFilter{}, false
		}
		filter.Enabled = &value
	}
	return filter, true
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
