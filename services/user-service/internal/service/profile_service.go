package service

import (
	"context"
	"errors"

	"services/user-service/internal/client"
	"services/user-service/internal/model"
	"services/user-service/internal/repository"

	"go.uber.org/zap"
)

// ProfileService handles profile-related operations
type ProfileService struct {
	profileRepo *repository.ProfileRepository
	userRepo    *repository.UserRepository
	mediaClient *client.MediaClient
	logger      *zap.Logger
}

// NewProfileService creates a new profile service
func NewProfileService(
	profileRepo *repository.ProfileRepository,
	userRepo *repository.UserRepository,
	mediaClient *client.MediaClient,
	logger *zap.Logger,
) *ProfileService {
	return &ProfileService{
		profileRepo: profileRepo,
		userRepo:    userRepo,
		mediaClient: mediaClient,
		logger:      logger,
	}
}

// GetProfilePhotoURL gets a user's profile photo URL
func (s *ProfileService) GetProfilePhotoURL(ctx context.Context, userID int) (string, error) {
	// Check if user exists and is active
	exists, err := s.checkUserActive(ctx, userID)
	if err != nil {
		return "", err
	}
	if !exists {
		return "", errors.New("user not found or inactive")
	}

	return s.profileRepo.GetProfilePhotoURL(ctx, userID)
}

// UpdateProfilePhoto updates a user's profile photo URL
func (s *ProfileService) UpdateProfilePhoto(ctx context.Context, userID int, photoURL string) error {
	// Check if user exists and is active
	exists, err := s.checkUserActive(ctx, userID)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("user not found or inactive")
	}

	success, err := s.profileRepo.UpdateProfilePhoto(ctx, userID, photoURL)
	if err != nil {
		return err
	}

	if !success {
		return errors.New("failed to update profile photo")
	}

	return nil
}

// UploadProfilePhoto uploads a profile photo to the media service and updates the user's profile
func (s *ProfileService) UploadProfilePhoto(
	ctx context.Context,
	userID int,
	fileContent []byte,
	filename string,
	contentType string,
) (*model.ProfilePhotoResponse, error) {
	// Check if user exists and is active
	exists, err := s.checkUserActive(ctx, userID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.New("user not found or inactive")
	}

	// Upload to media service
	mediaFile, err := s.mediaClient.UploadProfilePhoto(ctx, userID, fileContent, filename, contentType)
	if err != nil {
		s.logger.Error("Failed to upload file to media service", zap.Error(err))
		return nil, err
	}

	// Update user profile with new photo URL
	success, err := s.profileRepo.UpdateProfilePhoto(ctx, userID, mediaFile.URL)
	if err != nil {
		s.logger.Error("Failed to update profile photo URL", zap.Error(err))
		return nil, err
	}

	if !success {
		return nil, errors.New("failed to update profile photo URL")
	}

	// Convert client.Thumbnail to model.Thumbnail
	var thumbnails []model.Thumbnail
	for _, t := range mediaFile.Thumbnails {
		thumbnails = append(thumbnails, model.Thumbnail{
			Name:   t.Name,
			URL:    t.URL,
			Width:  t.Width,
			Height: t.Height,
		})
	}

	// Return response with converted thumbnails
	response := &model.ProfilePhotoResponse{
		URL:        mediaFile.URL,
		Thumbnails: thumbnails,
	}

	return response, nil
}

// ClearProfilePhoto removes a user's profile photo
func (s *ProfileService) ClearProfilePhoto(ctx context.Context, userID int) error {
	// Check if user exists and is active
	exists, err := s.checkUserActive(ctx, userID)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("user not found or inactive")
	}

	// Get current photo URL for potential cleanup
	currentURL, err := s.profileRepo.GetProfilePhotoURL(ctx, userID)
	if err != nil {
		return err
	}

	// If there's no photo, nothing to do
	if currentURL == "" {
		return nil
	}

	// Clear the profile photo URL
	success, err := s.profileRepo.ClearProfilePhoto(ctx, userID)
	if err != nil {
		return err
	}

	if !success {
		return errors.New("failed to clear profile photo")
	}

	// TODO: Potentially delete the file from media service if it's owned solely by this user
	// This would require additional logic in the media service and client

	return nil
}

// checkUserActive checks if a user exists and is active
func (s *ProfileService) checkUserActive(ctx context.Context, userID int) (bool, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return false, err
	}
	if user == nil || !user.IsActive {
		return false, nil
	}
	return true, nil
}
