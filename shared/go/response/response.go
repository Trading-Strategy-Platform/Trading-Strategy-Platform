package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/yourorg/trading-platform/shared/go/errors"
)

// Response represents a standardized API response
type Response struct {
	Data  interface{} `json:"data,omitempty"`
	Error *ErrorInfo  `json:"error,omitempty"`
	Meta  interface{} `json:"meta,omitempty"`
}

// ErrorInfo represents error information in a response
type ErrorInfo struct {
	Type    string `json:"type"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Meta represents pagination metadata
type Meta struct {
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	TotalPages int `json:"total_pages"`
	TotalItems int `json:"total_items"`
}

// Success sends a successful response with data
func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Data: data,
	})
}

// SuccessWithMeta sends a successful response with data and metadata
func SuccessWithMeta(c *gin.Context, data interface{}, meta interface{}) {
	c.JSON(http.StatusOK, Response{
		Data: data,
		Meta: meta,
	})
}

// Created sends a 201 Created response with data
func Created(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, Response{
		Data: data,
	})
}

// NoContent sends a 204 No Content response
func NoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

// Error sends an error response
func Error(c *gin.Context, err error) {
	var statusCode int
	var errorInfo *ErrorInfo

	// Check if it's an APIError
	if apiError, ok := err.(*errors.APIError); ok {
		statusCode = apiError.StatusCode()
		errorInfo = &ErrorInfo{
			Type:    string(apiError.Type),
			Code:    apiError.Code,
			Message: apiError.Message,
		}
	} else {
		// Default to internal server error
		statusCode = http.StatusInternalServerError
		errorInfo = &ErrorInfo{
			Type:    string(errors.ErrorTypeInternal),
			Code:    "internal_error",
			Message: "An internal server error occurred",
		}
	}

	c.JSON(statusCode, Response{
		Error: errorInfo,
	})
}

// BadRequest sends a 400 Bad Request response
func BadRequest(c *gin.Context, message string) {
	c.JSON(http.StatusBadRequest, Response{
		Error: &ErrorInfo{
			Type:    string(errors.ErrorTypeValidation),
			Code:    "invalid_request",
			Message: message,
		},
	})
}

// Unauthorized sends a 401 Unauthorized response
func Unauthorized(c *gin.Context, message string) {
	if message == "" {
		message = "Authentication required"
	}
	c.JSON(http.StatusUnauthorized, Response{
		Error: &ErrorInfo{
			Type:    string(errors.ErrorTypeAuth),
			Code:    "unauthorized",
			Message: message,
		},
	})
}

// Forbidden sends a 403 Forbidden response
func Forbidden(c *gin.Context, message string) {
	if message == "" {
		message = "Insufficient permissions"
	}
	c.JSON(http.StatusForbidden, Response{
		Error: &ErrorInfo{
			Type:    string(errors.ErrorTypePermission),
			Code:    "forbidden",
			Message: message,
		},
	})
}

// NotFound sends a 404 Not Found response
func NotFound(c *gin.Context, message string) {
	if message == "" {
		message = "Resource not found"
	}
	c.JSON(http.StatusNotFound, Response{
		Error: &ErrorInfo{
			Type:    string(errors.ErrorTypeNotFound),
			Code:    "not_found",
			Message: message,
		},
	})
}

// InternalError sends a 500 Internal Server Error response
func InternalError(c *gin.Context, message string) {
	if message == "" {
		message = "An internal server error occurred"
	}
	c.JSON(http.StatusInternalServerError, Response{
		Error: &ErrorInfo{
			Type:    string(errors.ErrorTypeInternal),
			Code:    "internal_error",
			Message: message,
		},
	})
}
