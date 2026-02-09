// Structured API error types and JSON error response writer.
package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
)

type errorCode string

const (
	codeBadRequest    errorCode = "BAD_REQUEST"
	codeNotFound      errorCode = "NOT_FOUND"
	codeConflict      errorCode = "CONFLICT"
	codeInternalError errorCode = "INTERNAL_ERROR"
)

// errorWithStatus is an error that carries an HTTP status code, error code,
// and optional details map.
type errorWithStatus interface {
	Error() string
	StatusCode() int
	Code() errorCode
	Details() map[string]any
}

// apiError is a concrete error type with status code, error code, optional
// details, and optional wrapped error.
type apiError struct {
	statusCode int
	code       errorCode
	message    string
	details    map[string]any
	wrappedErr error
}

func (e *apiError) Error() string {
	if e.wrappedErr != nil {
		return fmt.Sprintf("%s: %v", e.message, e.wrappedErr)
	}
	return e.message
}

func (e *apiError) StatusCode() int {
	return e.statusCode
}

func (e *apiError) Code() errorCode {
	return e.code
}

func (e *apiError) Details() map[string]any {
	return e.details
}

func (e *apiError) Unwrap() error {
	return e.wrappedErr
}

// WithDetail adds a single key/value to the error details.
func (e *apiError) WithDetail(key string, value any) *apiError {
	if e.details == nil {
		e.details = make(map[string]any)
	}
	e.details[key] = value
	return e
}

// Wrap wraps an underlying error.
func (e *apiError) Wrap(err error) *apiError {
	e.wrappedErr = err
	return e
}

// Constructors.

func badRequest(msg string) *apiError {
	return &apiError{statusCode: http.StatusBadRequest, code: codeBadRequest, message: msg}
}

func notFound(resource string) *apiError {
	return &apiError{statusCode: http.StatusNotFound, code: codeNotFound, message: resource + " not found"}
}

func conflict(msg string) *apiError {
	return &apiError{statusCode: http.StatusConflict, code: codeConflict, message: msg}
}

func internalError(msg string) *apiError {
	return &apiError{statusCode: http.StatusInternalServerError, code: codeInternalError, message: msg}
}

// errorResponse is the JSON envelope for error responses.
type errorResponse struct {
	Error   errorBody      `json:"error"`
	Details map[string]any `json:"details,omitempty"`
}

type errorBody struct {
	Code    errorCode `json:"code"`
	Message string    `json:"message"`
}

// writeError writes a structured JSON error response. If err implements
// errorWithStatus, the HTTP status, error code and details are taken from it;
// otherwise 500 is used.
func writeError(w http.ResponseWriter, err error) {
	statusCode := http.StatusInternalServerError
	code := codeInternalError
	var details map[string]any

	var ews errorWithStatus
	if errors.As(err, &ews) {
		statusCode = ews.StatusCode()
		code = ews.Code()
		details = ews.Details()
	}

	slog.Error("handler error", "err", err, "statusCode", statusCode, "code", code)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	resp := errorResponse{
		Error:   errorBody{Code: code, Message: err.Error()},
		Details: details,
	}
	if encErr := json.NewEncoder(w).Encode(resp); encErr != nil {
		slog.Warn("failed to encode error response", "err", encErr)
	}
}

// writeJSONResponse writes a JSON success response or a structured error
// response, unifying both paths into a single call.
func writeJSONResponse[Out any](w http.ResponseWriter, output *Out, err error) {
	if err != nil {
		writeError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if encErr := json.NewEncoder(w).Encode(output); encErr != nil {
		slog.Warn("failed to encode JSON response", "err", encErr)
	}
}
