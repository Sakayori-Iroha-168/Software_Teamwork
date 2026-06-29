package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"time"
)

const requestIDKey contextKey = "qa_request_id"

type contextKey string

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

func (r *statusRecorder) Flush() {
	if flusher, ok := r.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

// RequestLog wraps an HTTP handler with request ID propagation and structured access logs.
func RequestLog(logger *slog.Logger, next http.Handler) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-Id")
		if requestID == "" {
			requestID = newRequestID()
		}
		ctx := context.WithValue(r.Context(), requestIDKey, requestID)
		r = r.WithContext(ctx)
		w.Header().Set("X-Request-Id", requestID)

		recorder := &statusRecorder{ResponseWriter: w}
		startedAt := time.Now()
		next.ServeHTTP(recorder, r)

		status := recorder.status
		if status == 0 {
			status = http.StatusOK
		}
		outcome := "success"
		level := slog.LevelInfo
		if status >= http.StatusInternalServerError {
			outcome = "failed"
			level = slog.LevelError
		} else if status >= http.StatusBadRequest {
			outcome = "failed"
			level = slog.LevelWarn
		}
		logger.Log(r.Context(), level, "http request completed",
			"service", "qa",
			"request_id", requestID,
			"operation", "http_request",
			"status", outcome,
			"method", r.Method,
			"path", r.URL.Path,
			"http_status", status,
			"duration_ms", time.Since(startedAt).Milliseconds(),
		)
	})
}

// RequestIDFromContext returns the request ID attached by RequestLog middleware.
func RequestIDFromContext(ctx context.Context) string {
	if value, ok := ctx.Value(requestIDKey).(string); ok {
		return value
	}
	return ""
}

func newRequestID() string {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return hex.EncodeToString([]byte(time.Now().UTC().Format(time.RFC3339Nano)))
	}
	return hex.EncodeToString(bytes[:])
}
