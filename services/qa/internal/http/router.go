package http

import (
	"net/http"
	"time"
)

func NewRouter(handler *Handler) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", handler.Health)
	mux.HandleFunc("GET /api/v1/qa-config-versions/current", handler.CurrentQAConfig)
	mux.HandleFunc("POST /api/v1/qa-config-versions", handler.CreateQAConfig)
	mux.HandleFunc("GET /api/v1/llm-config-versions/current", handler.CurrentLLMConfig)
	mux.HandleFunc("POST /api/v1/llm-config-versions", handler.CreateLLMConfig)
	mux.HandleFunc("POST /api/v1/llm-connection-tests", handler.TestLLMConnection)
	return withRequestID(withRecovery(withCORS(mux)))
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-Id")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func withRecovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				writeError(w, r, NewAppError(CodeInternal, "internal server error", nil))
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func withRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-Id")
		if requestID == "" {
			requestID = "req_" + time.Now().UTC().Format("20060102150405.000000000")
		}
		w.Header().Set("X-Request-Id", requestID)
		next.ServeHTTP(w, r.WithContext(withRequestIDContext(r.Context(), requestID)))
	})
}
