package model

// RefreshTokenRequest represents a request to refresh an access token
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// ExternalUserInfo represents user information from an external identity provider
type ExternalUserInfo struct {
	ID         string `json:"id"`
	Email      string `json:"email"`
	Name       string `json:"name"`
	GivenName  string `json:"given_name,omitempty"`
	FamilyName string `json:"family_name,omitempty"`
	Picture    string `json:"picture,omitempty"`
	Locale     string `json:"locale,omitempty"`
	Provider   string `json:"provider,omitempty"`
}
