package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"starterkit/internal/logger"
	"starterkit/internal/model"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-playground/validator/v10"
)

var validate = validator.New()

// respondJSON writes a JSON response with the given status code.
func respondJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		slog.Error("failed to encode response", "error", err)
	}
}

// respondError writes a standardized error response with request ID.
func respondError(w http.ResponseWriter, r *http.Request, status int, code, message string, details ...string) {
	requestID := middleware.GetReqID(r.Context())
	resp := model.ErrorResponse{
		Error: model.ErrorDetail{
			Code:    code,
			Message: message,
			Details: details,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Request-ID", requestID)
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		slog.Error("failed to encode error response", "error", err)
	}
}

// decodeAndValidate decodes the request body and validates it.
func decodeAndValidate(r *http.Request, dst interface{}) error {
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		return fmt.Errorf("decode request: %w", err)
	}
	if err := validate.Struct(dst); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}
	return nil
}

// validationErrorResponse converts validator errors to a user-friendly response.
func validationErrorResponse(err error) (code, message string, details []string) {
	var validationErrors validator.ValidationErrors
	if errors.As(err, &validationErrors) {
		code = "VALIDATION_ERROR"
		message = "request validation failed"
		for _, e := range validationErrors {
			details = append(details, fmt.Sprintf("field '%s' failed validation: %s", strings.ToLower(e.Field()), e.Tag()))
		}
		return
	}
	code = "INVALID_REQUEST"
	message = "invalid request"
	return
}

// LogAndRespondError logs the error and sends an error response.
func LogAndRespondError(w http.ResponseWriter, r *http.Request, status int, code, message string, err error) {
	logger.FromContext(r.Context()).Error(message,
		slog.String("error", err.Error()),
		slog.Int("status", status),
		slog.String("code", code),
	)
	respondError(w, r, status, code, message)
}

// LogAndRespondValidationError logs validation errors and sends a response.
func LogAndRespondValidationError(w http.ResponseWriter, r *http.Request, err error) {
	code, message, details := validationErrorResponse(err)
	logger.FromContext(r.Context()).Warn("validation failed",
		slog.String("code", code),
		slog.String("error", err.Error()),
	)
	respondError(w, r, http.StatusBadRequest, code, message, details...)
}
