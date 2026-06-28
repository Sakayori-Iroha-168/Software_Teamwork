package http

import (
	"encoding/json"
	"errors"
	"net/http"
)

type Code string

const (
	CodeValidation Code = "validation_error"
	CodeInternal   Code = "internal_error"
)

type AppError struct {
	Code    Code
	Message string
	Err     error
}

func NewAppError(code Code, message string, err error) *AppError {
	return &AppError{Code: code, Message: message, Err: err}
}

func (e *AppError) Error() string {
	return e.Message
}

func (e *AppError) Unwrap() error {
	return e.Err
}

func writeError(w http.ResponseWriter, r *http.Request, err error) {
	appErr := &AppError{}
	if !errors.As(err, &appErr) {
		appErr = NewAppError(CodeInternal, "internal server error", err)
	}

	status := http.StatusInternalServerError
	if appErr.Code == CodeValidation {
		status = http.StatusBadRequest
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]any{
			"code":      appErr.Code,
			"message":   appErr.Message,
			"requestId": requestIDFromContext(r.Context()),
		},
	})
}
