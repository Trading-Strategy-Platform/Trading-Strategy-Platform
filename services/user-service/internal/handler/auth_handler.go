package handler

import (
	"net/http"

	"services/user-service/internal/client"
	"services/user-service/internal/model"
	"services/user-service/internal/service"

	sharedModel "github.com/yourorg/trading-platform/shared/go/model"
	"github.com/yourorg/trading-platform/shared/go/response"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// AuthHandler handles authentication requests
type AuthHandler struct {
	authService     *service.AuthService
	validatorClient *client.ValidatorClient
	logger          *zap.Logger
}

// NewAuthHandler creates a new authentication handler
func NewAuthHandler(authService *service.AuthService, validatorClient *client.ValidatorClient, logger *zap.Logger) *AuthHandler {
	return &AuthHandler{
		authService:     authService,
		validatorClient: validatorClient,
		logger:          logger,
	}
}

// Register handles user registration
func (h *AuthHandler) Register(c *gin.Context) {
	var userCreate model.UserCreate
	if err := c.ShouldBindJSON(&userCreate); err != nil {
		response.BadRequest(c, "Invalid request data")
		return
	}

	// Use shared validator for custom validations
	if err := h.validatorClient.Validate(userCreate); err != nil {
		c.Error(err)
		return
	}

	tokenResponse, err := h.authService.Register(c, &userCreate)
	if err != nil {
		h.logger.Error("Failed to register user", zap.Error(err))
		c.Error(err)
		return
	}

	c.JSON(http.StatusCreated, tokenResponse)
}

// Login handles user login
func (h *AuthHandler) Login(c *gin.Context) {
	var userLogin model.UserLogin
	if err := c.ShouldBindJSON(&userLogin); err != nil {
		response.BadRequest(c, "Invalid request data")
		return
	}

	// Use shared validator
	if err := h.validatorClient.Validate(userLogin); err != nil {
		c.Error(err)
		return
	}

	tokenResponse, err := h.authService.Login(c, &userLogin)
	if err != nil {
		h.logger.Error("Failed to login user", zap.Error(err))
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, tokenResponse)
}

// RefreshToken handles token refresh requests
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var refreshReq model.RefreshTokenRequest
	if err := c.ShouldBindJSON(&refreshReq); err != nil {
		response.BadRequest(c, "Invalid request data")
		return
	}

	// Validate refresh token
	if err := h.validatorClient.Validate(refreshReq); err != nil {
		c.Error(err)
		return
	}

	tokenResponse, err := h.authService.RefreshToken(c, refreshReq.RefreshToken)
	if err != nil {
		h.logger.Error("Failed to refresh token", zap.Error(err))
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, tokenResponse)
}

// Helper function to parse validation errors from binding errors
func parseValidationErrors(err error) *sharedModel.ValidationErrors {
	// Implementation depends on the validation library used
	// For gin's default validator, you would extract field errors
	// This is a simplified example
	return nil
}
