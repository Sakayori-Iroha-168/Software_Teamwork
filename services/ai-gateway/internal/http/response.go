package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/ai-gateway/internal/service"
)

type successEnvelope struct {
	Data      any    `json:"data"`
	RequestID string `json:"requestId"`
}

type errorEnvelope struct {
	Error errorBody `json:"error"`
}

type errorBody struct {
	Code      service.Code      `json:"code"`
	Message   string            `json:"message"`
	RequestID string            `json:"requestId"`
	Fields    map[string]string `json:"fields,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, data any, requestID string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(successEnvelope{Data: data, RequestID: requestID})
}

func writeAppError(w http.ResponseWriter, r *http.Request, err error) {
	appErr, ok := service.Classify(err)
	if !ok {
		appErr = service.NewError(service.CodeInternal, "internal server error", err)
	}
	writeError(w, statusForCode(appErr.Code), appErr.Code, appErr.Message, requestIDFromContext(r.Context()), appErr.Fields)
}

func writeError(w http.ResponseWriter, status int, code service.Code, message string, requestID string, fields map[string]string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(errorEnvelope{Error: errorBody{
		Code:      code,
		Message:   message,
		RequestID: requestID,
		Fields:    fields,
	}})
}

func statusForCode(code service.Code) int {
	switch code {
	case service.CodeValidation:
		return http.StatusBadRequest
	case service.CodeUnauthorized:
		return http.StatusUnauthorized
	case service.CodeForbidden:
		return http.StatusForbidden
	case service.CodeNotFound:
		return http.StatusNotFound
	case service.CodeConflict:
		return http.StatusConflict
	case service.CodeRateLimited:
		return http.StatusTooManyRequests
	case service.CodeDependency:
		return http.StatusBadGateway
	default:
		return http.StatusInternalServerError
	}
}
