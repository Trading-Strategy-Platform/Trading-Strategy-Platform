package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"services/api-gateway/internal/client"

	"github.com/gin-gonic/gin"
	sharedErrors "github.com/yourorg/trading-platform/shared/go/errors"
	"github.com/yourorg/trading-platform/shared/go/response"
)

// Validate creates middleware for validating request payloads
func Validate(validator *client.ValidatorClient, model interface{}) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Only validate for methods that can have a body
		if c.Request.Method == http.MethodPost ||
			c.Request.Method == http.MethodPut ||
			c.Request.Method == http.MethodPatch {

			// Read the request body
			body, err := io.ReadAll(c.Request.Body)
			if err != nil {
				c.Error(sharedErrors.NewValidationError("Invalid request body"))
				response.BadRequest(c, "Invalid request body")
				c.Abort()
				return
			}

			// Restore the request body for downstream handlers
			c.Request.Body = io.NopCloser(bytes.NewBuffer(body))

			// Create a new instance of the model to validate
			modelCopy := model

			// Unmarshal the body into the model
			if err := json.Unmarshal(body, &modelCopy); err != nil {
				c.Error(sharedErrors.NewValidationError("Invalid JSON payload"))
				response.BadRequest(c, "Invalid JSON payload")
				c.Abort()
				return
			}

			// Validate the model
			if err := validator.Validate(modelCopy); err != nil {
				// Use the validation error from the validator
				c.Error(err)
				response.BadRequest(c, err.Error())
				c.Abort()
				return
			}
		}

		c.Next()
	}
}
