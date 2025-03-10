package middleware

import (
	"net/http"

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

			// Use type assertions to handle different error types
			if apiErr, ok := err.(*sharedErrors.APIError); ok {
				// Handle different error types based on the APIError.Type field
				switch apiErr.Type {
				case sharedErrors.ErrorTypeValidation:
					// Handle validation errors
					validationErrors := &sharedModel.ValidationErrors{
						Errors: []sharedModel.ValidationError{
							{
								Field:   "", // Field information might not be available
								Message: apiErr.Message,
							},
						},
					}
					c.JSON(http.StatusBadRequest, validationErrors)

				case sharedErrors.ErrorTypeNotFound:
					// Handle not found errors
					response.NotFound(c, apiErr.Error())

				case sharedErrors.ErrorTypeAuth:
					// Handle authorization errors
					response.Unauthorized(c, apiErr.Error())

				case sharedErrors.ErrorTypePermission:
					// Handle forbidden errors
					response.Forbidden(c, apiErr.Error())

				case sharedErrors.ErrorTypeExternal:
					// Handle external service errors
					response.ServiceUnavailable(c, apiErr.Error())

				case sharedErrors.ErrorTypeDatabase:
					// Log database errors but return generic message to clients
					c.Error(apiErr) // Log the detailed error
					response.InternalError(c, "Database operation failed")

				default:
					// Handle all other errors as internal server errors
					response.InternalError(c, "An unexpected error occurred")
				}
			} else {
				// Handle non-APIError types as generic internal errors
				response.InternalError(c, "An unexpected error occurred")
			}
		}
	}
}
