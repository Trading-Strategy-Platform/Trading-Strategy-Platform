package client

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
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

// ValidateToken validates a user's token with the User Service
func (c *UserClient) ValidateToken(ctx context.Context, token string) (int, error) {
	url := fmt.Sprintf("%s/api/v1/auth/validate", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, err
	}

	// Add the token to be validated
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Service-Key", "historical-service-key")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error("Failed to validate token with User Service", zap.Error(err))
		return 0, err
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode == http.StatusUnauthorized {
		return 0, fmt.Errorf("invalid token")
	}

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		c.logger.Error("User service returned unexpected status",
			zap.Int("status_code", resp.StatusCode),
			zap.String("body", string(bodyBytes)))
		return 0, fmt.Errorf("user service returned status code %d", resp.StatusCode)
	}

	var response struct {
		Valid  bool `json:"valid"`
		UserID int  `json:"user_id"`
	}

	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		c.logger.Error("Failed to decode validation response", zap.Error(err))
		return 0, err
	}

	if !response.Valid {
		return 0, fmt.Errorf("invalid token")
	}

	return response.UserID, nil
}

// CheckUserRole checks if a user has a specific role
func (c *UserClient) CheckUserRole(ctx context.Context, userID int, role string, token string) (bool, error) {
	url := fmt.Sprintf("%s/api/v1/admin/users/%d/roles", c.baseURL, userID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false, err
	}

	// Add service authentication header
	req.Header.Set("X-Service-Key", "historical-service-key")

	// Add Authorization header if token is provided
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error("Failed to check user role with User Service", zap.Error(err))

		// Fallback for development - user ID 1 is always admin
		if userID == 1 && (role == "admin" || role == "user") {
			c.logger.Warn("Using fallback role check", zap.Int("userID", userID))
			return true, nil
		}

		// All other users have the 'user' role by default
		if role == "user" {
			return true, nil
		}

		return false, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusUnauthorized {
		c.logger.Warn("User roles endpoint error",
			zap.Int("statusCode", resp.StatusCode),
			zap.Int("userID", userID),
			zap.String("role", role))

		// Fallback for development
		if userID == 1 && (role == "admin" || role == "user") {
			return true, nil
		}

		if role == "user" {
			return true, nil
		}

		return false, nil
	}

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		c.logger.Error("User service returned error status",
			zap.Int("statusCode", resp.StatusCode),
			zap.String("response", string(bodyBytes)))

		// Fallback for development
		if userID == 1 && (role == "admin" || role == "user") {
			return true, nil
		}

		if role == "user" {
			return true, nil
		}

		return false, nil
	}

	var response struct {
		Roles []string `json:"roles"`
	}

	bodyBytes, _ := io.ReadAll(resp.Body)
	err = json.Unmarshal(bodyBytes, &response)
	if err != nil {
		c.logger.Error("Failed to decode roles response",
			zap.Error(err),
			zap.String("responseBody", string(bodyBytes)))

		// Fallback for development
		if userID == 1 && (role == "admin" || role == "user") {
			return true, nil
		}

		if role == "user" {
			return true, nil
		}

		return false, nil
	}

	// Check if the required role is in the user's roles
	for _, userRole := range response.Roles {
		if userRole == role {
			return true, nil
		}
	}

	return false, nil
}

// HasRole checks if a user has a specific role - legacy method for compatibility
func (c *UserClient) HasRole(ctx context.Context, userID int, role string, token string) (bool, error) {
	return c.CheckUserRole(ctx, userID, role, token)
}

// GetUsername gets a username by user ID
func (c *UserClient) GetUsername(ctx context.Context, userID int) (string, error) {
	url := fmt.Sprintf("%s/api/v1/admin/users/%d", c.baseURL, userID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}

	// Add service authentication header
	req.Header.Set("X-Service-Key", "historical-service-key")

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

// ExtractUserIDFromToken extracts the user ID from a JWT token
func ExtractUserIDFromToken(token string) (int, error) {
	// Split the token into its parts (header.payload.signature)
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return 0, fmt.Errorf("invalid token format")
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
			return 0, fmt.Errorf("failed to decode payload: %w", err)
		}
	}

	// Parse the JSON payload
	var claims struct {
		Sub  int    `json:"sub"`
		Type string `json:"type"`
	}

	if err := json.Unmarshal(decodedPayload, &claims); err != nil {
		return 0, fmt.Errorf("failed to parse token payload: %w", err)
	}

	// Check if this is an access token
	if claims.Type != "access" && claims.Type != "" {
		return 0, fmt.Errorf("not an access token")
	}

	return claims.Sub, nil
}

// GetUserDetails gets user details by ID
func (c *UserClient) GetUserDetails(ctx context.Context, userID int) (*struct {
	ID              int    `json:"id"`
	Username        string `json:"username"`
	ProfilePhotoURL string `json:"profile_photo_url,omitempty"`
}, error) {
	url := fmt.Sprintf("%s/api/v1/admin/users/%d", c.baseURL, userID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	// Add service authentication header
	req.Header.Set("X-Service-Key", "historical-service-key")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error("Failed to get user from User Service", zap.Error(err))
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("user service returned status code %d", resp.StatusCode)
	}

	var user struct {
		ID              int    `json:"id"`
		Username        string `json:"username"`
		ProfilePhotoURL string `json:"profile_photo_url,omitempty"`
	}

	err = json.NewDecoder(resp.Body).Decode(&user)
	if err != nil {
		c.logger.Error("Failed to decode user response", zap.Error(err))
		return nil, err
	}

	return &user, nil
}

// BatchGetUsersByIDs gets details for multiple users by their IDs
func (c *UserClient) BatchGetUsersByIDs(ctx context.Context, userIDs []int) (map[int]struct {
	ID              int    `json:"id"`
	Username        string `json:"username"`
	ProfilePhotoURL string `json:"profile_photo_url,omitempty"`
}, error) {
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
	req.Header.Set("X-Service-Key", "historical-service-key")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error("Failed to get users from User Service", zap.Error(err))
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("user service returned status code %d", resp.StatusCode)
	}

	var response struct {
		Users []struct {
			ID              int    `json:"id"`
			Username        string `json:"username"`
			ProfilePhotoURL string `json:"profile_photo_url,omitempty"`
		} `json:"users"`
	}

	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		c.logger.Error("Failed to decode users response", zap.Error(err))
		return nil, err
	}

	// Create map of user ID to user details
	result := make(map[int]struct {
		ID              int    `json:"id"`
		Username        string `json:"username"`
		ProfilePhotoURL string `json:"profile_photo_url,omitempty"`
	})
	for _, user := range response.Users {
		result[user.ID] = user
	}

	return result, nil
}
