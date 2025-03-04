package errors

// ErrorType represents different error categories
type ErrorType string

const (
	// ErrorTypeAuth for authentication/authorization errors
	ErrorTypeAuth ErrorType = "auth"

	// ErrorTypeValidation for input validation errors
	ErrorTypeValidation ErrorType = "validation"

	// ErrorTypePermission for permission errors
	ErrorTypePermission ErrorType = "permission"

	// ErrorTypeNotFound for resource not found errors
	ErrorTypeNotFound ErrorType = "not_found"

	// ErrorTypeDuplicate for duplicate resource errors
	ErrorTypeDuplicate ErrorType = "duplicate"

	// ErrorTypeDatabase for database operation errors
	ErrorTypeDatabase ErrorType = "database"

	// ErrorTypeInternal for internal server errors
	ErrorTypeInternal ErrorType = "internal"

	// ErrorTypeExternal for external service errors
	ErrorTypeExternal ErrorType = "external"
)

// Common error codes
const (
	// Authentication errors
	CodeInvalidCredentials = "invalid_credentials"
	CodeTokenExpired       = "token_expired"
	CodeInvalidToken       = "invalid_token"
	CodeAccountLocked      = "account_locked"
	CodeAccountInactive    = "account_inactive"

	// Validation errors
	CodeMissingField    = "missing_field"
	CodeInvalidFormat   = "invalid_format"
	CodeInvalidValue    = "invalid_value"
	CodePasswordTooWeak = "password_too_weak"

	// Permission errors
	CodeInsufficientPermissions = "insufficient_permissions"
	CodeResourceOwnership       = "resource_ownership"

	// Resource errors
	CodeResourceNotFound  = "resource_not_found"
	CodeResourceConflict  = "resource_conflict"
	CodeResourceDuplicate = "resource_duplicate"

	// Database errors
	CodeDatabaseConnection = "database_connection"
	CodeDatabaseQuery      = "database_query"
	CodeDatabaseConstraint = "database_constraint"

	// External service errors
	CodeExternalServiceUnavailable = "external_service_unavailable"
	CodeExternalServiceTimeout     = "external_service_timeout"

	// Rate limiting
	CodeRateLimitExceeded = "rate_limit_exceeded"
)
