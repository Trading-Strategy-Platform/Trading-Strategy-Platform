package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

// TokenType defines the type of token
type TokenType string

const (
	// AccessToken is used for API access
	AccessToken TokenType = "access"
	// RefreshToken is used to obtain new access tokens
	RefreshToken TokenType = "refresh"
)

// Claims represents the custom JWT claims
type Claims struct {
	UserID int       `json:"sub"`
	Type   TokenType `json:"type"`
	Role   string    `json:"role,omitempty"`
	jwt.RegisteredClaims
}

// GenerateToken creates a new signed JWT token
func GenerateToken(userID int, secret string, tokenType TokenType, duration time.Duration, role string) (string, time.Time, error) {
	expiryTime := time.Now().Add(duration)

	claims := Claims{
		UserID: userID,
		Type:   tokenType,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiryTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString([]byte(secret))

	return signedToken, expiryTime, err
}

// ValidateToken validates a JWT token and returns the parsed claims
func ValidateToken(tokenString, secret string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secret), nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, errors.New("invalid claims")
	}

	return claims, nil
}

// GenerateTokenPair creates both access and refresh tokens
func GenerateTokenPair(userID int, secret string, accessDuration, refreshDuration time.Duration, role string) (accessToken, refreshToken string, accessExpiry time.Time, err error) {
	accessToken, accessExpiry, err = GenerateToken(userID, secret, AccessToken, accessDuration, role)
	if err != nil {
		return "", "", time.Time{}, err
	}

	refreshToken, _, err = GenerateToken(userID, secret, RefreshToken, refreshDuration, role)
	if err != nil {
		return "", "", time.Time{}, err
	}

	return accessToken, refreshToken, accessExpiry, nil
}
