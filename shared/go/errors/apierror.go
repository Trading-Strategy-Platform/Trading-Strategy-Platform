package errors

import (
	"fmt"
	"net/http"
)

// APIError represents a standardized API error
type APIError struct {
	Type     ErrorType `json:"type"`
	Code     string    `json:"code"`
	Message  string    `json:"message"`
	Details  string    `json:"details,omitempty"`
	Internal error     `json:"-"` // Internal error not exposed
}

// Error implements the error interface
func (e *APIError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("%s: %s - %s", e.Type, e.Message, e.Details)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

// StatusCode returns the appropriate HTTP status code for the error
func (e *APIError) StatusCode() int {
	switch e.Type {
	case ErrorTypeAuth:
		return http.StatusUnauthorized
	case ErrorTypeValidation:
		return http.StatusBadRequest
	case ErrorTypePermission:
		return http.StatusForbidden
	case ErrorTypeNotFound:
		return http.StatusNotFound
	case ErrorTypeDuplicate:
		return http.StatusConflict
	case ErrorTypeExternal:
		return http.StatusBadGateway
	case ErrorTypeDatabase:
		return http.StatusInternalServerError
	case ErrorTypeInternal:
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}

// WithDetails adds additional details to the error
func (e *APIError) WithDetails(details string) *APIError {
	e.Details = details
	return e
}

// WithInternal adds an internal error
func (e *APIError) WithInternal(err error) *APIError {
	e.Internal = err
	return e
}

// NewAuthError creates a new authentication error
func NewAuthError(message string) *APIError {
	return &APIError{
		Type:    ErrorTypeAuth,
		Code:    "unauthorized",
		Message: message,
	}
}

// NewValidationError creates a new validation error
func NewValidationError(message string) *APIError {
	return &APIError{
		Type:    ErrorTypeValidation,
		Code:    "invalid_input",
		Message: message,
	}
}

// NewPermissionError creates a new permission error
func NewPermissionError(message string) *APIError {
	return &APIError{
		Type:    ErrorTypePermission,
		Code:    "forbidden",
		Message: message,
	}
}

// NewNotFoundError creates a new not found error
func NewNotFoundError(resourceType, resourceID string) *APIError {
	return &APIError{
		Type:    ErrorTypeNotFound,
		Code:    "not_found",
		Message: fmt.Sprintf("%s with ID %s not found", resourceType, resourceID),
	}
}

// NewDuplicateError creates a new duplicate resource error
func NewDuplicateError(resourceType, field string) *APIError {
	return &APIError{
		Type:    ErrorTypeDuplicate,
		Code:    "duplicate_resource",
		Message: fmt.Sprintf("%s with this %s already exists", resourceType, field),
	}
}

// NewDatabaseError creates a new database error
func NewDatabaseError(operation string, err error) *APIError {
	return &APIError{
		Type:     ErrorTypeDatabase,
		Code:     "database_error",
		Message:  fmt.Sprintf("Database error during %s", operation),
		Internal: err,
	}
}

// NewInternalError creates a new internal server error
func NewInternalError(message string, err error) *APIError {
	return &APIError{
		Type:     ErrorTypeInternal,
		Code:     "internal_error",
		Message:  message,
		Internal: err,
	}
}

// NewExternalServiceError creates a new external service error
func NewExternalServiceError(service, message string, err error) *APIError {
	return &APIError{
		Type:     ErrorTypeExternal,
		Code:     "external_service_error",
		Message:  fmt.Sprintf("Error from %s service: %s", service, message),
		Internal: err,
	}
}
