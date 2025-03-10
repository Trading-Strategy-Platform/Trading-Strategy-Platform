package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	sharedErrors "github.com/yourorg/trading-platform/shared/go/errors"
	sharedModel "github.com/yourorg/trading-platform/shared/go/model"
	"github.com/yourorg/trading-platform/shared/go/response"
)

// ErrorHandler middleware captures and formats errors consistently
func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Process the request
		c.Next()

		// If there were errors
		if len(c.Errors) > 0 {
			err := c.Errors.Last().Err

			// Check if it's an API error from shared errors package
			if apiErr, ok := err.(*sharedErrors.APIError); ok {
				// Handle different error types based on the APIError.Type field
				switch apiErr.Type {
				case sharedErrors.ErrorTypeValidation:
					// Parse validation error message
					errorMsg := apiErr.Error()
					errorFields := strings.Split(errorMsg, "; ")

					validationErrors := &sharedModel.ValidationErrors{
						Errors: make([]sharedModel.ValidationError, 0, len(errorFields)),
					}

					for _, fieldError := range errorFields {
						parts := strings.SplitN(fieldError, " ", 2)
						if len(parts) == 2 {
							validationErrors.Errors = append(validationErrors.Errors, sharedModel.ValidationError{
								Field:   parts[0],
								Message: fieldError,
							})
						} else {
							// Fallback for malformed error messages
							validationErrors.Errors = append(validationErrors.Errors, sharedModel.ValidationError{
								Field:   "unknown",
								Message: fieldError,
							})
						}
					}

					c.JSON(http.StatusBadRequest, validationErrors)

				case sharedErrors.ErrorTypeNotFound:
					response.NotFound(c, apiErr.Error())

				case sharedErrors.ErrorTypeAuth:
					response.Unauthorized(c, apiErr.Error())

				case sharedErrors.ErrorTypePermission:
					response.Forbidden(c, apiErr.Error())

				case sharedErrors.ErrorTypeDuplicate:
					response.Conflict(c, apiErr.Error())

				case sharedErrors.ErrorTypeDatabase:
					// Log database errors but return generic message to clients
					c.Error(apiErr)
					response.InternalError(c, "Database operation failed")

				default:
					response.InternalError(c, "An unexpected error occurred")
				}
			} else {
				// For non-APIError types, return a generic error
				response.InternalError(c, "An unexpected error occurred")
			}
		}
	}
}
