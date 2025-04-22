package client

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"
)

// UserClient handles communication with the User Service
type UserClient struct {
	baseURL    string
	httpClient *http.Client
	logger     *zap.Logger
}

// NewUserClient creates a new User Service client
func NewUserClient(baseURL string, logger *zap.Logger) *UserClient {
	return &UserClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		logger: logger,
	}
}

// CheckUserRole checks if a user has a specific role directly from the JWT token
// Note: This method is now deprecated as role checking should be done directly
// from the JWT token in the middleware
func (c *UserClient) CheckUserRole(ctx context.Context, userID int, role string, token string) (bool, error) {
	// Add warning log that this function is deprecated
	c.logger.Warn("CheckUserRole is deprecated. Roles should now be extracted directly from JWT token",
		zap.Int("userID", userID),
		zap.String("role", role))

	// If a token is provided, extract the role from it
	if token != "" {
		_, userRole, err := extractUserInfoFromToken(token)
		if err != nil {
			c.logger.Error("Failed to extract role from token", zap.Error(err))
			return false, err
		}

		// Return result based on extracted role
		return userRole == role, nil
	}

	// For backward compatibility, make a direct call to the validate endpoint
	url := fmt.Sprintf("%s/api/v1/auth/validate", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		c.logger.Error("Failed to create validation request", zap.Error(err))
		return false, err
	}

	// Add Authorization header if token is provided
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	// Add service key header
	req.Header.Set("X-Service-Key", "strategy-service-key")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error("Failed to validate token with User Service", zap.Error(err))
		return false, err
	}
	defer resp.Body.Close()

	// Handle response
	if resp.StatusCode == http.StatusOK {
		var response struct {
			Valid bool   `json:"valid"`
			Role  string `json:"role"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			c.logger.Error("Failed to decode validation response", zap.Error(err))
			return false, err
		}

		return response.Valid && response.Role == role, nil
	}

	return false, fmt.Errorf("user service returned status code %d", resp.StatusCode)
}

// GetUserByID retrieves a user's username by ID
func (c *UserClient) GetUserByID(ctx context.Context, userID int) (string, error) {
	url := fmt.Sprintf("%s/api/v1/service/users/%d", c.baseURL, userID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}

	// Add service authentication header
	req.Header.Set("X-Service-Key", "strategy-service-key")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error("Failed to get user from User Service", zap.Error(err))
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("user service returned status code %d", resp.StatusCode)
	}

	var user struct {
		ID       int    `json:"id"`
		Username string `json:"username"`
	}

	err = json.NewDecoder(resp.Body).Decode(&user)
	if err != nil {
		c.logger.Error("Failed to decode user response", zap.Error(err))
		return "", err
	}

	return user.Username, nil
}

// ValidateUserAccess validates if a user has access using their token
func (c *UserClient) ValidateUserAccess(ctx context.Context, userID int, token string) (bool, error) {
	url := fmt.Sprintf("%s/api/v1/auth/validate", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false, err
	}

	// Add headers
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Service-Key", "strategy-service-key")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error("Failed to validate token with User Service", zap.Error(err))
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, nil
	}

	var response struct {
		Valid  bool `json:"valid"`
		UserID int  `json:"user_id"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return false, err
	}

	return response.Valid && response.UserID == userID, nil
}

// BatchGetUsersByIDs retrieves multiple users' details by their IDs
func (c *UserClient) BatchGetUsersByIDs(ctx context.Context, userIDs []int) (map[int]UserDetails, error) {
	// Build comma-separated list of user IDs
	var idParams string
	for i, id := range userIDs {
		if i > 0 {
			idParams += ","
		}
		idParams += fmt.Sprintf("%d", id)
	}

	url := fmt.Sprintf("%s/api/v1/service/users/batch?ids=%s", c.baseURL, idParams)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	// Add service authentication header
	req.Header.Set("X-Service-Key", "strategy-service-key")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error("Failed to get users from User Service", zap.Error(err))
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.logger.Warn("User service returned non-200 status",
			zap.Int("status_code", resp.StatusCode),
			zap.String("url", url))
		return nil, fmt.Errorf("user service returned status code %d", resp.StatusCode)
	}

	var response struct {
		Users []UserDetails `json:"users"`
	}

	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		c.logger.Error("Failed to decode users response", zap.Error(err))
		return nil, err
	}

	// Create map of user ID to user details
	result := make(map[int]UserDetails)
	for _, user := range response.Users {
		result[user.ID] = user
	}

	return result, nil
}

// UserDetails represents the user information returned by the user service
type UserDetails struct {
	ID              int    `json:"id"`
	Username        string `json:"username"`
	ProfilePhotoURL string `json:"profile_photo_url,omitempty"`
}

// extractUserInfoFromToken extracts both user ID and role from a JWT token
func extractUserInfoFromToken(token string) (int, string, error) {
	// Split the token into its parts (header.payload.signature)
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return 0, "", fmt.Errorf("invalid token format")
	}

	// Decode the payload (the middle part)
	// Add padding if needed
	payload := parts[1]
	if len(payload)%4 != 0 {
		payload += strings.Repeat("=", 4-len(payload)%4)
	}

	decodedPayload, err := base64.URLEncoding.DecodeString(payload)
	if err != nil {
		// Try with RawURLEncoding if standard URLEncoding fails
		decodedPayload, err = base64.RawURLEncoding.DecodeString(parts[1])
		if err != nil {
			return 0, "", fmt.Errorf("failed to decode payload: %w", err)
		}
	}

	// Parse the JSON payload
	var claims struct {
		Sub  int    `json:"sub"`
		Type string `json:"type"`
		Role string `json:"role"`
	}

	if err := json.Unmarshal(decodedPayload, &claims); err != nil {
		return 0, "", fmt.Errorf("failed to parse token payload: %w", err)
	}

	// Check if this is an access token
	if claims.Type != "access" && claims.Type != "" {
		return 0, "", fmt.Errorf("not an access token")
	}

	// If role is empty, default to "user"
	if claims.Role == "" {
		claims.Role = "user"
	}

	return claims.Sub, claims.Role, nil
}
