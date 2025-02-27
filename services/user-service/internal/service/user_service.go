package service

import (
	"context"
	"errors"

	"services/user-service/internal/model"
	"services/user-service/internal/repository"

	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

// UserService handles user operations
type UserService struct {
	userRepo *repository.UserRepository
	logger   *zap.Logger
}

// NewUserService creates a new user service
func NewUserService(userRepo *repository.UserRepository, logger *zap.Logger) *UserService {
	return &UserService{
		userRepo: userRepo,
		logger:   logger,
	}
}

// GetByID retrieves a user by ID
func (s *UserService) GetByID(ctx context.Context, id int) (*model.User, error) {
	return s.userRepo.GetByID(ctx, id)
}

// GetCurrentUser gets the current user by ID from context
func (s *UserService) GetCurrentUser(ctx context.Context, userID int) (*model.User, error) {
	return s.userRepo.GetByID(ctx, userID)
}

// Update updates a user's details
func (s *UserService) Update(ctx context.Context, id int, update *model.UserUpdate) error {
	return s.userRepo.Update(ctx, id, update)
}

// ChangePassword changes a user's password
func (s *UserService) ChangePassword(ctx context.Context, id int, change *model.UserChangePassword) error {
	// Get user to verify current password
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if user == nil {
		return errors.New("user not found")
	}

	// Verify current password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(change.CurrentPassword)); err != nil {
		return errors.New("current password is incorrect")
	}

	// Hash new password
	newHash, err := bcrypt.GenerateFromPassword([]byte(change.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		s.logger.Error("failed to hash password", zap.Error(err))
		return errors.New("failed to process new password")
	}

	// Update password in database
	return s.userRepo.UpdatePassword(ctx, id, string(newHash))
}

// ListUsers gets a paginated list of users
func (s *UserService) ListUsers(ctx context.Context, page, limit int) ([]model.User, int, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}

	offset := (page - 1) * limit

	users, err := s.userRepo.List(ctx, offset, limit)
	if err != nil {
		return nil, 0, err
	}

	count, err := s.userRepo.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	return users, count, nil
}
