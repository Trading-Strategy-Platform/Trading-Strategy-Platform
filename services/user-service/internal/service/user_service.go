package service

import (
	"context"
	"errors"

	"services/user-service/internal/model"
	"services/user-service/internal/repository"

	"go.uber.org/zap"
)

// UserService handles core user operations
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

// GetByEmail retrieves a user by email
func (s *UserService) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	return s.userRepo.GetByEmail(ctx, email)
}

// GetCurrentUser gets the current user by ID from context
func (s *UserService) GetCurrentUser(ctx context.Context, userID int) (*model.UserDetails, error) {
	return s.userRepo.GetUserDetails(ctx, userID)
}

// Update updates a user's details
func (s *UserService) Update(ctx context.Context, id int, update *model.UserUpdate) error {
	success, err := s.userRepo.UpdateUser(
		ctx,
		id,
		update.Username,
		update.Email,
		update.ProfilePhotoURL,
		update.IsActive,
	)

	if err != nil {
		return err
	}

	if !success {
		return errors.New("failed to update user")
	}

	return nil
}

// DeleteUser marks a user as inactive
func (s *UserService) DeleteUser(ctx context.Context, id int) error {
	success, err := s.userRepo.DeleteUser(ctx, id)
	if err != nil {
		return err
	}

	if !success {
		return errors.New("failed to delete user")
	}

	return nil
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

// GetRole gets a user's role
func (s *UserService) GetRole(ctx context.Context, id int) (string, error) {
	return s.userRepo.GetRole(ctx, id)
}

// CheckUserExists checks if a user exists
func (s *UserService) CheckUserExists(ctx context.Context, id int) (bool, error) {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return false, err
	}
	return user != nil, nil
}

// CheckUserActive checks if a user is active
func (s *UserService) CheckUserActive(ctx context.Context, id int) (bool, error) {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return false, err
	}
	if user == nil {
		return false, nil
	}
	return user.IsActive, nil
}
