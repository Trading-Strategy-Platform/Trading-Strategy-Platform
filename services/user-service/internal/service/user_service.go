package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"services/user-service/internal/model"
	"services/user-service/internal/repository"

	"github.com/go-redis/redis/v8"
	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

// UserService handles core user operations
type UserService struct {
	userRepo    *repository.UserRepository
	logger      *zap.Logger
	redisClient *redis.Client // Added Redis client
	kafkaWriter *kafka.Writer // Added Kafka writer
}

// NewUserService creates a new user service
func NewUserService(
	userRepo *repository.UserRepository,
	logger *zap.Logger,
	redisClient *redis.Client, // New parameter
	kafkaWriter *kafka.Writer, // New parameter
) *UserService {
	return &UserService{
		userRepo:    userRepo,
		logger:      logger,
		redisClient: redisClient,
		kafkaWriter: kafkaWriter,
	}
}

// GetByID retrieves a user by ID with Redis caching
func (s *UserService) GetByID(ctx context.Context, id int) (*model.User, error) {
	// Try to get from cache first
	cacheKey := fmt.Sprintf("user:%d", id)

	// Only try cache if Redis is available
	if s.redisClient != nil {
		userData, err := s.redisClient.Get(ctx, cacheKey).Bytes()
		if err == nil {
			// Found in cache, unmarshal and return
			var user model.User
			if err := json.Unmarshal(userData, &user); err == nil {
				s.logger.Debug("User found in cache", zap.Int("user_id", id))
				return &user, nil
			}
		}
	}

	// Get from database
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Store in cache for future requests if Redis is available
	if user != nil && s.redisClient != nil {
		if userData, err := json.Marshal(user); err == nil {
			s.redisClient.Set(ctx, cacheKey, userData, 15*time.Minute)
		}
	}

	return user, nil
}

// GetByEmail retrieves a user by email
func (s *UserService) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	// Try cache if Redis is available
	if s.redisClient != nil {
		cacheKey := fmt.Sprintf("user:email:%s", email)
		userData, err := s.redisClient.Get(ctx, cacheKey).Bytes()
		if err == nil {
			var user model.User
			if err := json.Unmarshal(userData, &user); err == nil {
				return &user, nil
			}
		}
	}

	// Get from database
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil, err
	}

	// Cache the result if Redis is available
	if user != nil && s.redisClient != nil {
		userData, err := json.Marshal(user)
		if err == nil {
			emailKey := fmt.Sprintf("user:email:%s", email)
			s.redisClient.Set(ctx, emailKey, userData, 15*time.Minute)
		}
	}

	return user, nil
}

// GetCurrentUser gets the current user by ID from context
func (s *UserService) GetCurrentUser(ctx context.Context, userID int) (*model.UserDetails, error) {
	// Try cache if Redis is available
	if s.redisClient != nil {
		cacheKey := fmt.Sprintf("user:details:%d", userID)
		userData, err := s.redisClient.Get(ctx, cacheKey).Bytes()
		if err == nil {
			var userDetails model.UserDetails
			if err := json.Unmarshal(userData, &userDetails); err == nil {
				return &userDetails, nil
			}
		}
	}

	// Get from database
	userDetails, err := s.userRepo.GetUserDetails(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Cache the result if Redis is available
	if userDetails != nil && s.redisClient != nil {
		if userData, err := json.Marshal(userDetails); err == nil {
			cacheKey := fmt.Sprintf("user:details:%d", userID)
			s.redisClient.Set(ctx, cacheKey, userData, 15*time.Minute)
		}
	}

	return userDetails, nil
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

	// Invalidate cache entries if Redis is available
	if s.redisClient != nil {
		// Invalidate user ID cache
		s.redisClient.Del(ctx, fmt.Sprintf("user:%d", id))
		// Invalidate email cache if changed
		if update.Email != nil && *update.Email != "" {
			s.redisClient.Del(ctx, fmt.Sprintf("user:email:%s", *update.Email))
		}
		// Invalidate details cache
		s.redisClient.Del(ctx, fmt.Sprintf("user:details:%d", id))
	}

	// Publish update event to Kafka if available
	if s.kafkaWriter != nil {
		event := map[string]interface{}{
			"event_type": "user_updated",
			"user_id":    id,
			"username":   update.Username,
			"email":      update.Email,
			"is_active":  update.IsActive,
			"timestamp":  time.Now().Format(time.RFC3339),
		}

		if eventJSON, err := json.Marshal(event); err == nil {
			// Don't block on Kafka errors
			go func() {
				message := kafka.Message{
					Key:   []byte(fmt.Sprintf("%d", id)),
					Value: eventJSON,
					Time:  time.Now(),
				}

				if err := s.kafkaWriter.WriteMessages(ctx, message); err != nil {
					s.logger.Error("Failed to publish user update event",
						zap.Error(err),
						zap.Int("user_id", id))
				}
			}()
		}
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

	// Invalidate cache if Redis is available
	if s.redisClient != nil {
		s.redisClient.Del(ctx, fmt.Sprintf("user:%d", id))
		s.redisClient.Del(ctx, fmt.Sprintf("user:details:%d", id))
		// Note: We can't easily invalidate the email cache since we don't have the email here
	}

	// Publish delete event to Kafka if available
	if s.kafkaWriter != nil {
		event := map[string]interface{}{
			"event_type": "user_deleted",
			"user_id":    id,
			"timestamp":  time.Now().Format(time.RFC3339),
		}

		if eventJSON, err := json.Marshal(event); err == nil {
			// Don't block on Kafka errors
			go func() {
				message := kafka.Message{
					Key:   []byte(fmt.Sprintf("%d", id)),
					Value: eventJSON,
					Time:  time.Now(),
				}

				if err := s.kafkaWriter.WriteMessages(ctx, message); err != nil {
					s.logger.Error("Failed to publish user delete event",
						zap.Error(err),
						zap.Int("user_id", id))
				}
			}()
		}
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

	// For list operations, we typically don't cache as they change frequently
	// However, we could cache short-lived special lists if needed

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
	// Try cache if Redis is available
	if s.redisClient != nil {
		cacheKey := fmt.Sprintf("user:role:%d", id)
		role, err := s.redisClient.Get(ctx, cacheKey).Result()
		if err == nil {
			return role, nil
		}
	}

	// Get from database
	role, err := s.userRepo.GetRole(ctx, id)
	if err != nil {
		return "", err
	}

	// Cache the result if Redis is available
	if s.redisClient != nil && role != "" {
		cacheKey := fmt.Sprintf("user:role:%d", id)
		s.redisClient.Set(ctx, cacheKey, role, 30*time.Minute)
	}

	return role, nil
}

// CheckUserExists checks if a user exists
func (s *UserService) CheckUserExists(ctx context.Context, id int) (bool, error) {
	// Try cache if Redis is available
	if s.redisClient != nil {
		cacheKey := fmt.Sprintf("user:%d", id)
		exists, err := s.redisClient.Exists(ctx, cacheKey).Result()
		if err == nil && exists > 0 {
			return true, nil
		}
	}

	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return false, err
	}
	return user != nil, nil
}

// CheckUserActive checks if a user is active
func (s *UserService) CheckUserActive(ctx context.Context, id int) (bool, error) {
	// We'll use GetByID which already implements caching
	user, err := s.GetByID(ctx, id)
	if err != nil {
		return false, err
	}
	if user == nil {
		return false, nil
	}
	return user.IsActive, nil
}

func (s *UserService) GetUsersByIDs(ctx context.Context, ids []int) ([]model.User, error) {
	if len(ids) == 0 {
		return []model.User{}, nil
	}

	// Create a placeholder for the results
	users := make([]model.User, 0, len(ids))
	missingIDs := make([]int, 0)

	// Check cache first if Redis is available
	if s.redisClient != nil {
		for _, id := range ids {
			cacheKey := fmt.Sprintf("user:%d", id)
			userData, err := s.redisClient.Get(ctx, cacheKey).Bytes()

			if err == nil {
				// Found in cache
				var user model.User
				if err := json.Unmarshal(userData, &user); err == nil {
					users = append(users, user)
					continue
				}
			}

			// Not in cache, add to list to fetch from DB
			missingIDs = append(missingIDs, id)
		}

		// If all users were in cache, return early
		if len(missingIDs) == 0 {
			return users, nil
		}

		// Update ids to only include missing ones
		ids = missingIDs
	}

	// For each user ID in the batch
	for _, id := range ids {
		user, err := s.userRepo.GetByID(ctx, id)
		if err != nil {
			s.logger.Warn("Error fetching user in batch",
				zap.Error(err),
				zap.Int("user_id", id))
			continue // Skip this user but continue with others
		}

		if user != nil {
			users = append(users, *user)

			// Cache the user if Redis is available
			if s.redisClient != nil {
				if userData, err := json.Marshal(user); err == nil {
					cacheKey := fmt.Sprintf("user:%d", id)
					s.redisClient.Set(ctx, cacheKey, userData, 15*time.Minute)
				}
			}
		}
	}

	return users, nil
}

func (s *UserService) ValidateServiceKey(ctx context.Context, serviceName, keyHash string) (bool, error) {
	// Try cache if Redis is available
	if s.redisClient != nil {
		cacheKey := fmt.Sprintf("service-key:%s:%s", serviceName, keyHash)
		valid, err := s.redisClient.Get(ctx, cacheKey).Bool()
		if err == nil {
			return valid, nil
		}
	}

	// Check if service name is valid
	if serviceName != "strategy-service" &&
		serviceName != "historical-service" &&
		serviceName != "media-service" {
		return false, nil
	}

	// Check if key hash matches expected value for the service
	// In a real implementation, you would fetch this from the database
	expectedKeyHash := "strategy-service-key"
	isValid := keyHash == expectedKeyHash

	// Cache the result if Redis is available
	if s.redisClient != nil {
		cacheKey := fmt.Sprintf("service-key:%s:%s", serviceName, keyHash)
		s.redisClient.Set(ctx, cacheKey, isValid, 24*time.Hour)
	}

	return isValid, nil
}

// AddAuthSession creates a new authentication session in Redis
func (s *UserService) AddAuthSession(ctx context.Context, userID int, token string, duration time.Duration) error {
	// Only proceed if Redis is available
	if s.redisClient == nil {
		return errors.New("redis is not available")
	}

	// Get user details to store in session
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	if user == nil {
		return errors.New("user not found")
	}

	// Create session data
	sessionData := map[string]interface{}{
		"user_id":    userID,
		"username":   user.Username,
		"is_active":  user.IsActive,
		"created_at": time.Now().Unix(),
	}

	// Store in Redis
	sessionJSON, err := json.Marshal(sessionData)
	if err != nil {
		return err
	}

	sessionKey := fmt.Sprintf("session:%s", token)
	if err := s.redisClient.Set(ctx, sessionKey, sessionJSON, duration).Err(); err != nil {
		return err
	}

	// Also publish login event to Kafka if available
	if s.kafkaWriter != nil {
		event := map[string]interface{}{
			"event_type": "user_login",
			"user_id":    userID,
			"timestamp":  time.Now().Format(time.RFC3339),
		}

		if eventJSON, err := json.Marshal(event); err == nil {
			// Use a goroutine to avoid blocking on Kafka
			go func() {
				message := kafka.Message{
					Key:   []byte(fmt.Sprintf("%d", userID)),
					Value: eventJSON,
					Time:  time.Now(),
				}

				if err := s.kafkaWriter.WriteMessages(context.Background(), message); err != nil {
					s.logger.Error("Failed to publish user login event",
						zap.Error(err),
						zap.Int("user_id", userID))
				}
			}()
		}
	}

	return nil
}

// ValidateAuthSession validates a session token
func (s *UserService) ValidateAuthSession(ctx context.Context, token string) (int, error) {
	// Only proceed if Redis is available
	if s.redisClient == nil {
		return 0, errors.New("redis is not available")
	}

	sessionKey := fmt.Sprintf("session:%s", token)
	sessionJSON, err := s.redisClient.Get(ctx, sessionKey).Bytes()
	if err != nil {
		if err == redis.Nil {
			return 0, errors.New("session not found or expired")
		}
		return 0, err
	}

	var sessionData map[string]interface{}
	if err := json.Unmarshal(sessionJSON, &sessionData); err != nil {
		return 0, err
	}

	// Extract user ID from session
	userIDFloat, ok := sessionData["user_id"].(float64)
	if !ok {
		return 0, errors.New("invalid session data: user_id not found")
	}

	userID := int(userIDFloat)

	// Check if user is still active
	isActive, err := s.CheckUserActive(ctx, userID)
	if err != nil {
		return 0, err
	}

	if !isActive {
		// Remove invalid session
		s.redisClient.Del(ctx, sessionKey)
		return 0, errors.New("user account is inactive")
	}

	return userID, nil
}

// InvalidateAuthSession removes a session
func (s *UserService) InvalidateAuthSession(ctx context.Context, token string) error {
	// Only proceed if Redis is available
	if s.redisClient == nil {
		return errors.New("redis is not available")
	}

	// Get session first to extract user ID for logout event
	sessionKey := fmt.Sprintf("session:%s", token)
	sessionJSON, err := s.redisClient.Get(ctx, sessionKey).Bytes()

	// Remove the session regardless of whether we got it
	s.redisClient.Del(ctx, sessionKey)

	// If we retrieved the session, log logout event
	if err == nil && s.kafkaWriter != nil {
		var sessionData map[string]interface{}
		if err := json.Unmarshal(sessionJSON, &sessionData); err == nil {
			if userIDFloat, ok := sessionData["user_id"].(float64); ok {
				userID := int(userIDFloat)

				// Log logout event to Kafka
				event := map[string]interface{}{
					"event_type": "user_logout",
					"user_id":    userID,
					"timestamp":  time.Now().Format(time.RFC3339),
				}

				if eventJSON, err := json.Marshal(event); err == nil {
					// Use a goroutine to avoid blocking
					go func() {
						message := kafka.Message{
							Key:   []byte(fmt.Sprintf("%d", userID)),
							Value: eventJSON,
							Time:  time.Now(),
						}

						if err := s.kafkaWriter.WriteMessages(context.Background(), message); err != nil {
							s.logger.Error("Failed to publish user logout event",
								zap.Error(err),
								zap.Int("user_id", userID))
						}
					}()
				}
			}
		}
	}

	return nil
}
