package middleware

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRequestLogIncludesRequiredFields(t *testing.T) {
	var buffer bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buffer, nil))
	handler := RequestLog(logger, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))

	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	request.Header.Set("X-Request-Id", "req-test-1")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	output := buffer.String()
	for _, field := range []string{`"service":"qa"`, `"request_id":"req-test-1"`, `"operation":"http_request"`, `"status":"success"`} {
		if !strings.Contains(output, field) {
			t.Fatalf("expected log to contain %s, got %s", field, output)
		}
	}
}
