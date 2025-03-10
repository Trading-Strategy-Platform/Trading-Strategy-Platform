package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/yourorg/trading-platform/shared/go/kafka"
	"go.uber.org/zap"
)

// KafkaService handles Kafka message production for the API Gateway
type KafkaService struct {
	producer  *kafka.Producer
	logger    *zap.Logger
	topics    map[string]string
	brokers   string
	isEnabled bool
}

// KafkaMessage represents a generic Kafka message structure
type KafkaMessage struct {
	Type      string          `json:"type"`
	Timestamp time.Time       `json:"timestamp"`
	Payload   json.RawMessage `json:"payload"`
}

// NewKafkaService creates a new Kafka service
func NewKafkaService(
	brokers string,
	topics map[string]string,
	enabled bool,
	logger *zap.Logger,
) (*KafkaService, error) {
	service := &KafkaService{
		logger:    logger,
		topics:    topics,
		brokers:   brokers,
		isEnabled: enabled,
	}

	// Only initialize the producer if Kafka is enabled
	if enabled {
		// Create Kafka producer
		brokersList := []string{brokers}
		producer := kafka.NewProducer(brokersList, logger)

		service.producer = producer
		logger.Info("Kafka producer initialized", zap.String("brokers", brokers))
	} else {
		logger.Info("Kafka integration is disabled")
	}

	return service, nil
}

// PublishUserEvent publishes a user-related event (login, registration, etc.)
func (s *KafkaService) PublishUserEvent(
	ctx context.Context,
	eventType string,
	userID int,
	metadata map[string]interface{},
) error {
	if !s.isEnabled || s.producer == nil {
		return nil // Silently skip if Kafka is disabled
	}

	// Prepare payload
	payload := map[string]interface{}{
		"event_type": eventType,
		"user_id":    userID,
		"timestamp":  time.Now().Unix(),
	}

	// Add any additional metadata
	for k, v := range metadata {
		payload[k] = v
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal user event payload: %w", err)
	}

	// Create Kafka message
	kafkaMsg := KafkaMessage{
		Type:      "user_event",
		Timestamp: time.Now(),
		Payload:   payloadBytes,
	}

	msgBytes, err := json.Marshal(kafkaMsg)
	if err != nil {
		return fmt.Errorf("failed to marshal Kafka message: %w", err)
	}

	// Publish to the user events topic
	topic := s.topics["userEvents"]
	key := fmt.Sprintf("user-%d", userID)

	err = s.producer.PublishMessage(ctx, topic, key, string(msgBytes))
	if err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	s.logger.Info("Published user event",
		zap.String("topic", topic),
		zap.Int("user_id", userID),
		zap.String("event_type", eventType))

	return nil
}

// PublishAPIMetric publishes API usage metrics
func (s *KafkaService) PublishAPIMetric(
	ctx context.Context,
	endpoint string,
	method string,
	statusCode int,
	latency time.Duration,
	userID *int,
) error {
	if !s.isEnabled || s.producer == nil {
		return nil // Silently skip if Kafka is disabled
	}

	// Prepare payload
	payload := map[string]interface{}{
		"endpoint":    endpoint,
		"method":      method,
		"status_code": statusCode,
		"latency_ms":  latency.Milliseconds(),
		"timestamp":   time.Now().Unix(),
	}

	// Add user ID if available
	if userID != nil {
		payload["user_id"] = *userID
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal API metric payload: %w", err)
	}

	// Create Kafka message
	kafkaMsg := KafkaMessage{
		Type:      "api_metric",
		Timestamp: time.Now(),
		Payload:   payloadBytes,
	}

	msgBytes, err := json.Marshal(kafkaMsg)
	if err != nil {
		return fmt.Errorf("failed to marshal Kafka message: %w", err)
	}

	// Publish to the API metrics topic
	topic := s.topics["apiMetrics"]
	key := fmt.Sprintf("%s-%s", method, endpoint)

	err = s.producer.PublishMessage(ctx, topic, key, string(msgBytes))
	if err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	return nil
}

// PublishSystemEvent publishes system-level events
func (s *KafkaService) PublishSystemEvent(
	ctx context.Context,
	eventType string,
	severity string,
	message string,
	metadata map[string]interface{},
) error {
	if !s.isEnabled || s.producer == nil {
		return nil // Silently skip if Kafka is disabled
	}

	// Prepare payload
	payload := map[string]interface{}{
		"event_type": eventType,
		"severity":   severity,
		"message":    message,
		"timestamp":  time.Now().Unix(),
	}

	// Add any additional metadata
	for k, v := range metadata {
		payload[k] = v
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal system event payload: %w", err)
	}

	// Create Kafka message
	kafkaMsg := KafkaMessage{
		Type:      "system_event",
		Timestamp: time.Now(),
		Payload:   payloadBytes,
	}

	msgBytes, err := json.Marshal(kafkaMsg)
	if err != nil {
		return fmt.Errorf("failed to marshal Kafka message: %w", err)
	}

	// Publish to the system events topic
	topic := s.topics["systemEvents"]
	key := eventType

	err = s.producer.PublishMessage(ctx, topic, key, string(msgBytes))
	if err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	s.logger.Info("Published system event",
		zap.String("topic", topic),
		zap.String("event_type", eventType),
		zap.String("severity", severity))

	return nil
}

// Close shuts down the Kafka producer
func (s *KafkaService) Close() {
	if s.isEnabled && s.producer != nil {
		s.producer.Close()
	}
}
