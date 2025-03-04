package auth

import (
	"errors"
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/crypto/bcrypt"
)

// PasswordConfig contains the configuration for password validation
type PasswordConfig struct {
	MinLength      int
	RequireUpper   bool
	RequireLower   bool
	RequireNumber  bool
	RequireSpecial bool
}

// DefaultPasswordConfig provides sensible defaults for password validation
var DefaultPasswordConfig = PasswordConfig{
	MinLength:      8,
	RequireUpper:   true,
	RequireLower:   true,
	RequireNumber:  true,
	RequireSpecial: true,
}

// HashPassword hashes a password using bcrypt
func HashPassword(password string) (string, error) {
	if password == "" {
		return "", errors.New("password cannot be empty")
	}

	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	return string(hashedBytes), nil
}

// VerifyPassword checks if a password matches a hash
func VerifyPassword(hashedPassword, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}

// ValidatePassword checks if a password meets the requirements specified in config
func ValidatePassword(password string, config PasswordConfig) error {
	if len(password) < config.MinLength {
		return errors.New("password is too short")
	}

	var hasUpper, hasLower, hasNumber, hasSpecial bool

	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsNumber(char):
			hasNumber = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			hasSpecial = true
		}
	}

	if config.RequireUpper && !hasUpper {
		return errors.New("password must contain at least one uppercase letter")
	}

	if config.RequireLower && !hasLower {
		return errors.New("password must contain at least one lowercase letter")
	}

	if config.RequireNumber && !hasNumber {
		return errors.New("password must contain at least one number")
	}

	if config.RequireSpecial && !hasSpecial {
		return errors.New("password must contain at least one special character")
	}

	return nil
}

// IsCommonPassword checks if a password is in a list of common passwords
func IsCommonPassword(password string) bool {
	commonPasswords := map[string]bool{
		"password":   true,
		"123456":     true,
		"123456789":  true,
		"qwerty":     true,
		"12345678":   true,
		"111111":     true,
		"1234567890": true,
		"admin":      true,
		"welcome":    true,
		"password1":  true,
		// Add more common passwords as needed
	}

	return commonPasswords[password]
}

// SanitizeEmail sanitizes an email address
func SanitizeEmail(email string) string {
	email = strings.TrimSpace(email)
	email = strings.ToLower(email)

	return email
}

// ValidateEmail validates an email address
func ValidateEmail(email string) bool {
	pattern := `^[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}$`
	match, _ := regexp.MatchString(pattern, email)
	return match
}

// SanitizeUsername sanitizes a username
func SanitizeUsername(username string) string {
	username = strings.TrimSpace(username)
	// Remove potentially harmful characters
	re := regexp.MustCompile(`[<>&'"]`)
	username = re.ReplaceAllString(username, "")

	return username
}
