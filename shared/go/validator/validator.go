package validator

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/yourorg/trading-platform/shared/go/errors"
)

// Validator is a custom validator for the application
type Validator struct {
	validator *validator.Validate
}

// NewValidator creates a new validator
func NewValidator() *Validator {
	v := validator.New()

	// Register custom validations
	v.RegisterValidation("alphanum_space", validateAlphanumSpace)
	v.RegisterValidation("alphanum_dash", validateAlphanumDash)
	v.RegisterValidation("strong_password", validateStrongPassword)
	v.RegisterValidation("timeframe", validateTimeframe)
	v.RegisterValidation("symbol", validateSymbol)

	// Use JSON tag names for error messages
	v.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		return name
	})

	return &Validator{validator: v}
}

// Validate validates a struct and returns a validation error if any
func (v *Validator) Validate(i interface{}) error {
	err := v.validator.Struct(i)
	if err == nil {
		return nil
	}

	// Convert validation errors to APIError
	validationErrors, ok := err.(validator.ValidationErrors)
	if !ok {
		return errors.NewInternalError("Validation failed", err)
	}

	// Create error messages
	var messages []string
	for _, e := range validationErrors {
		field := e.Field()
		tag := e.Tag()
		param := e.Param()

		var message string
		switch tag {
		case "required":
			message = fmt.Sprintf("%s is required", field)
		case "email":
			message = fmt.Sprintf("%s must be a valid email address", field)
		case "min":
			message = fmt.Sprintf("%s must be at least %s characters long", field, param)
		case "max":
			message = fmt.Sprintf("%s must be at most %s characters long", field, param)
		case "alphanum_space":
			message = fmt.Sprintf("%s can only contain letters, numbers, and spaces", field)
		case "alphanum_dash":
			message = fmt.Sprintf("%s can only contain letters, numbers, dashes, and underscores", field)
		case "strong_password":
			message = fmt.Sprintf("%s must be a strong password", field)
		case "timeframe":
			message = fmt.Sprintf("%s must be a valid timeframe (e.g., 1m, 5m, 15m, 1h, 4h, 1d)", field)
		case "symbol":
			message = fmt.Sprintf("%s must be a valid trading symbol", field)
		default:
			message = fmt.Sprintf("%s failed validation on '%s'", field, tag)
		}
		messages = append(messages, message)
	}

	return errors.NewValidationError(strings.Join(messages, "; "))
}

// Custom validators

func validateAlphanumSpace(fl validator.FieldLevel) bool {
	re := regexp.MustCompile(`^[a-zA-Z0-9 ]+$`)
	return re.MatchString(fl.Field().String())
}

func validateAlphanumDash(fl validator.FieldLevel) bool {
	re := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	return re.MatchString(fl.Field().String())
}

func validateStrongPassword(fl validator.FieldLevel) bool {
	password := fl.Field().String()
	hasUpper := regexp.MustCompile(`[A-Z]`).MatchString(password)
	hasLower := regexp.MustCompile(`[a-z]`).MatchString(password)
	hasNumber := regexp.MustCompile(`[0-9]`).MatchString(password)
	hasSpecial := regexp.MustCompile(`[^a-zA-Z0-9]`).MatchString(password)

	return len(password) >= 8 && hasUpper && hasLower && hasNumber && hasSpecial
}

func validateTimeframe(fl validator.FieldLevel) bool {
	timeframe := fl.Field().String()
	validTimeframes := map[string]bool{
		"1m": true, "3m": true, "5m": true, "15m": true, "30m": true,
		"1h": true, "2h": true, "4h": true, "6h": true, "8h": true, "12h": true,
		"1d": true, "3d": true, "1w": true, "1M": true,
	}

	return validTimeframes[timeframe]
}

func validateSymbol(fl validator.FieldLevel) bool {
	symbol := fl.Field().String()
	re := regexp.MustCompile(`^[A-Z0-9]{1,20}$`)
	return re.MatchString(symbol)
}
