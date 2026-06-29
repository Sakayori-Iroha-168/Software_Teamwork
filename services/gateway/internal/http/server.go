package httpapi

import (
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/gateway/internal/middleware"
	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/gateway/internal/response"
)

type Config struct {
	Logger               *slog.Logger
	ServiceVersion       string
	Environment          string
	RequestTimeout       time.Duration
	MaxBodyBytes         int64
	CORSAllowedOrigins   []string
	CORSAllowedMethods   []string
	CORSAllowedHeaders   []string
	CORSAllowCredentials bool
	QAServiceURL         string
}

type Server struct {
	logger         *slog.Logger
	serviceVersion string
	environment    string
	mux            *http.ServeMux
	handler        http.Handler
	qaProxy        *httputil.ReverseProxy
}

func NewServer(cfg Config) *Server {
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	s := &Server{
		logger:         cfg.Logger,
		serviceVersion: cfg.ServiceVersion,
		environment:    cfg.Environment,
		mux:            http.NewServeMux(),
	}
	if cfg.QAServiceURL != "" {
		s.qaProxy = newQAServiceProxy(cfg.QAServiceURL, cfg.Logger)
	}
	s.routes()
	s.handler = middleware.Chain(
		s.mux,
		middleware.RequestID(),
		middleware.Recover(cfg.Logger),
		middleware.Timeout(cfg.RequestTimeout),
		middleware.CORS(middleware.CORSConfig{
			AllowedOrigins:   cfg.CORSAllowedOrigins,
			AllowedMethods:   cfg.CORSAllowedMethods,
			AllowedHeaders:   cfg.CORSAllowedHeaders,
			AllowCredentials: cfg.CORSAllowCredentials,
		}),
		middleware.BodyLimit(cfg.MaxBodyBytes),
	)
	return s
}

func newQAServiceProxy(targetURL string, logger *slog.Logger) *httputil.ReverseProxy {
	target, _ := url.Parse(targetURL)
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.ModifyResponse = func(r *http.Response) error {
		return nil
	}
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		logger.ErrorContext(r.Context(), "QA service proxy error", "error", err)
		response.WriteError(w, http.StatusBadGateway, response.ErrorDetail{
			Code:      response.CodeDependency,
			Message:   "QA service unavailable",
			RequestID: middleware.RequestIDFromContext(r.Context()),
		})
	}
	return proxy
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /healthz", s.handleHealth)
	s.mux.HandleFunc("GET /readyz", s.handleReady)
	if s.qaProxy != nil {
		s.registerQARoutes()
	}
	s.mux.HandleFunc("/", s.handleNotFound)
}

func (s *Server) registerQARoutes() {
	qaPaths := []string{
		"/api/v1/qa-sessions",
		"/api/v1/response-runs",
		"/api/v1/messages",
		"/api/v1/citations",
		"/api/v1/citation-lookups",
		"/api/v1/qa-config-versions",
		"/api/v1/llm-config-versions",
		"/api/v1/llm-connection-tests",
		"/api/v1/retrieval-test-runs",
		"/api/v1/qa-metrics",
	}
	for _, path := range qaPaths {
		s.mux.HandleFunc("GET "+path, s.handleQAProxy)
		s.mux.HandleFunc("GET "+path+"/{rest...}", s.handleQAProxy)
		s.mux.HandleFunc("POST "+path, s.handleQAProxy)
		s.mux.HandleFunc("POST "+path+"/{rest...}", s.handleQAProxy)
		s.mux.HandleFunc("PATCH "+path+"/{rest...}", s.handleQAProxy)
		s.mux.HandleFunc("DELETE "+path+"/{rest...}", s.handleQAProxy)
	}
}

func (s *Server) handleQAProxy(w http.ResponseWriter, r *http.Request) {
	if s.qaProxy == nil {
		s.handleNotFound(w, r)
		return
	}
	newPath := r.URL.Path
	newPath = strings.Replace(newPath, "/api/v1/", "/internal/v1/", 1)
	r.URL.Path = newPath
	s.qaProxy.ServeHTTP(w, r)
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.handler.ServeHTTP(w, r)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	response.WriteJSON(w, http.StatusOK, healthResponse{
		Status:      "ok",
		Service:     "gateway",
		Version:     s.serviceVersion,
		Environment: s.environment,
	}, middleware.RequestIDFromContext(r.Context()))
}

func (s *Server) handleReady(w http.ResponseWriter, r *http.Request) {
	response.WriteJSON(w, http.StatusOK, healthResponse{
		Status:      "ready",
		Service:     "gateway",
		Version:     s.serviceVersion,
		Environment: s.environment,
	}, middleware.RequestIDFromContext(r.Context()))
}

func (s *Server) handleNotFound(w http.ResponseWriter, r *http.Request) {
	response.WriteError(w, http.StatusNotFound, response.ErrorDetail{
		Code:      response.CodeNotFound,
		Message:   "route not found",
		RequestID: middleware.RequestIDFromContext(r.Context()),
	})
}

type healthResponse struct {
	Status      string `json:"status"`
	Service     string `json:"service"`
	Version     string `json:"version,omitempty"`
	Environment string `json:"environment,omitempty"`
}
