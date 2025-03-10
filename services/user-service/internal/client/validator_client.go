package client

import (
	"github.com/yourorg/trading-platform/shared/go/validator"
)

// ValidatorClient wraps the shared validator to provide validation capabilities
type ValidatorClient struct {
	validator *validator.Validator
}

// NewValidatorClient creates a new validator client
func NewValidatorClient() *ValidatorClient {
	return &ValidatorClient{
		validator: validator.NewValidator(),
	}
}

// Validate validates a struct and returns validation errors if any
func (v *ValidatorClient) Validate(i interface{}) error {
	return v.validator.Validate(i)
}
