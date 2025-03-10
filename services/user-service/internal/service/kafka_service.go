package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"services/user-service/internal/config"
	"services/user-service/internal/model"

	"github.com/yourorg/trading-platform/shared/go/kafka"
	"go.uber.org/zap"
)

// KafkaService handles Kafka event publishing for the user service
type KafkaService struct {
	producer *kafka.Producer
	topics   map[string]string
	logger   *zap.Logger
}

// NewKafkaService creates a new Kafka service
func NewKafkaService(cfg *config.Config, logger *zap.Logger) (*KafkaService, error) {
	brokers := []string{cfg.Kafka.Brokers}
	producer := kafka.NewProducer(brokers, logger)

	return &KafkaService{
		producer: producer,
		topics:   cfg.Kafka.Topics,
		logger:   logger,
	}, nil
}

// PublishUserCreated publishes a user created event
func (s *KafkaService) PublishUserCreated(ctx context.Context, user *model.User) error {
	event := map[string]interface{}{
		"event_type": "user_created",
		"user_id":    user.ID,
		"username":   user.Username,
		"email":      user.Email,
		"role":       user.Role,
		"created_at": user.CreatedAt,
	}

	return s.publishEvent(ctx, fmt.Sprintf("user-%d", user.ID), event)
}

// PublishUserUpdated publishes a user updated event
func (s *KafkaService) PublishUserUpdated(ctx context.Context, user *model.User) error {
	event := map[string]interface{}{
		"event_type": "user_updated",
		"user_id":    user.ID,
		"username":   user.Username,
		"email":      user.Email,
		"role":       user.Role,
		"updated_at": user.UpdatedAt,
	}

	return s.publishEvent(ctx, fmt.Sprintf("user-%d", user.ID), event)
}

// PublishLoginEvent publishes a user login event
func (s *KafkaService) PublishLoginEvent(ctx context.Context, userID int, successful bool, ipAddress, userAgent string) error {
	event := map[string]interface{}{
		"event_type": "user_login",
		"user_id":    userID,
		"successful": successful,
		"ip_address": ipAddress,
		"user_agent": userAgent,
		"timestamp":  json.Number(fmt.Sprintf("%d", time.Now().Unix())),
	}

	return s.publishEvent(ctx, fmt.Sprintf("user-%d", userID), event)
}

// publishEvent publishes an event to the events topic
func (s *KafkaService) publishEvent(ctx context.Context, key string, event map[string]interface{}) error {
	topic, exists := s.topics["events"]
	if !exists {
		topic = "user-events" // fallback
	}

	err := s.producer.PublishMessage(ctx, topic, key, event)
	if err != nil {
		s.logger.Error("Failed to publish event to Kafka",
			zap.String("topic", topic),
			zap.String("key", key),
			zap.Error(err))
		return err
	}

	return nil
}

// Close closes the Kafka producer
func (s *KafkaService) Close() error {
	return s.producer.Close()
}
