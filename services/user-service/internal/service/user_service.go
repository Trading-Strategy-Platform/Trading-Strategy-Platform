package service

import (
	"context"

	"services/user-service/internal/model"
	"services/user-service/internal/repository"

	sharedErrors "github.com/yourorg/trading-platform/shared/go/errors"
	sharedModel "github.com/yourorg/trading-platform/shared/go/model"

	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

// UserService handles user operations
type UserService struct {
	userRepo     *repository.UserRepository
	logger       *zap.Logger
	kafkaService *KafkaService
}

// NewUserService creates a new user service
func NewUserService(userRepo *repository.UserRepository, kafkaService *KafkaService, logger *zap.Logger) *UserService {
	return &UserService{
		userRepo:     userRepo,
		logger:       logger,
		kafkaService: kafkaService,
	}
}

// GetByID retrieves a user by ID
func (s *UserService) GetByID(ctx context.Context, id int) (*model.User, error) {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return nil, sharedErrors.NewDatabaseError("retrieving user by ID", err)
	}
	if user == nil {
		return nil, sharedErrors.NewNotFoundError("User", string(id))
	}
	return user, nil
}

// GetCurrentUser gets the current user by ID from context
func (s *UserService) GetCurrentUser(ctx context.Context, userID int) (*model.User, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, sharedErrors.NewDatabaseError("retrieving current user", err)
	}
	if user == nil {
		return nil, sharedErrors.NewNotFoundError("User", string(userID))
	}
	return user, nil
}

// Update updates a user's details
func (s *UserService) Update(ctx context.Context, id int, update *model.UserUpdate) error {
	// First check if user exists
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return sharedErrors.NewDatabaseError("checking user existence", err)
	}
	if user == nil {
		return sharedErrors.NewNotFoundError("User", string(id))
	}

	// If email is being updated, check it's not already in use
	if update.Email != nil && *update.Email != user.Email {
		existingUser, err := s.userRepo.GetByEmail(ctx, *update.Email)
		if err != nil {
			return sharedErrors.NewDatabaseError("checking email availability", err)
		}
		if existingUser != nil {
			return sharedErrors.NewDuplicateError("User", "email")
		}
	}

	if err := s.userRepo.Update(ctx, id, update); err != nil {
		return sharedErrors.NewDatabaseError("updating user", err)
	}

	// Publish user updated event after successful update
	if s.kafkaService != nil {
		// Get updated user details
		updatedUser, err := s.userRepo.GetByID(ctx, id)
		if err != nil {
			// Log but continue since the update was successful
			s.logger.Warn("Failed to get updated user for event publishing", zap.Error(err), zap.Int("user_id", id))
		} else if updatedUser != nil {
			if err := s.kafkaService.PublishUserUpdated(ctx, updatedUser); err != nil {
				s.logger.Warn("Failed to publish user updated event", zap.Error(err), zap.Int("user_id", id))
			}
		}
	}

	return nil
}

// ChangePassword changes a user's password
func (s *UserService) ChangePassword(ctx context.Context, id int, change *model.UserChangePassword) error {
	// Get user to verify current password
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return sharedErrors.NewDatabaseError("retrieving user", err)
	}
	if user == nil {
		return sharedErrors.NewNotFoundError("User", string(id))
	}

	// Verify current password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(change.CurrentPassword)); err != nil {
		return sharedErrors.NewAuthError("Current password is incorrect")
	}

	// Hash new password
	newHash, err := bcrypt.GenerateFromPassword([]byte(change.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		s.logger.Error("failed to hash password", zap.Error(err))
		return sharedErrors.NewInternalError("Failed to process new password", err)
	}

	// Update password in database
	if err := s.userRepo.UpdatePassword(ctx, id, string(newHash)); err != nil {
		return sharedErrors.NewDatabaseError("updating password", err)
	}
	return nil
}

// ListUsers gets a paginated list of users using shared pagination model
func (s *UserService) ListUsers(ctx context.Context, pagination *sharedModel.Pagination) ([]model.User, *sharedModel.PaginationMeta, error) {
	offset := pagination.GetOffset()
	limit := pagination.GetPerPage()

	users, err := s.userRepo.List(ctx, offset, limit)
	if err != nil {
		return nil, nil, sharedErrors.NewDatabaseError("listing users", err)
	}

	count, err := s.userRepo.Count(ctx)
	if err != nil {
		return nil, nil, sharedErrors.NewDatabaseError("counting users", err)
	}

	meta := sharedModel.NewPaginationMeta(pagination, count)
	return users, &meta, nil
}
