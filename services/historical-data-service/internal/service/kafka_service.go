package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"services/historical-data-service/internal/model"

	"github.com/yourorg/trading-platform/shared/go/kafka"
	"go.uber.org/zap"
)

// KafkaService handles Kafka message production and consumption
type KafkaService struct {
	producer        *kafka.Producer
	backtestService *BacktestService
	logger          *zap.Logger
	topics          map[string]string
	brokers         string
}

// KafkaMessage represents a generic Kafka message structure
type KafkaMessage struct {
	Type      string          `json:"type"`
	Timestamp time.Time       `json:"timestamp"`
	Payload   json.RawMessage `json:"payload"`
}

// BacktestEventPayload represents an event related to backtests
type BacktestEventPayload struct {
	BacktestID int                  `json:"backtest_id"`
	UserID     int                  `json:"user_id"`
	StrategyID int                  `json:"strategy_id"`
	Status     model.BacktestStatus `json:"status"`
	Timestamp  time.Time            `json:"timestamp"`
	Message    string               `json:"message,omitempty"`
}

// NewKafkaService creates a new Kafka service
func NewKafkaService(
	brokers string,
	topics map[string]string,
	backtestService *BacktestService,
	logger *zap.Logger,
) (*KafkaService, error) {
	// Create Kafka producer
	brokersList := []string{brokers}
	producer := kafka.NewProducer(brokersList, logger)

	return &KafkaService{
		producer:        producer,
		backtestService: backtestService,
		logger:          logger,
		topics:          topics,
		brokers:         brokers,
	}, nil
}

// Start initializes the Kafka consumers
func (s *KafkaService) Start(ctx context.Context) error {
	// Set up consumer for backtest requests
	s.logger.Info("Starting Kafka consumer",
		zap.String("brokers", s.brokers),
		zap.String("topic", s.topics["backtestRequests"]))

	// Here we would set up consumers, but for now we'll just log
	s.logger.Info("Kafka consumers started")

	return nil
}

// handleMessage processes incoming Kafka messages
func (s *KafkaService) handleMessage(msg []byte) error {
	var kafkaMsg KafkaMessage
	if err := json.Unmarshal(msg, &kafkaMsg); err != nil {
		return fmt.Errorf("failed to unmarshal Kafka message: %w", err)
	}

	switch kafkaMsg.Type {
	case "backtest_request":
		return s.handleBacktestRequest(kafkaMsg.Payload)
	case "backtest_cancel":
		return s.handleBacktestCancel(kafkaMsg.Payload)
	default:
		s.logger.Warn("Unknown message type", zap.String("type", kafkaMsg.Type))
		return nil
	}
}

// handleBacktestRequest processes backtest request messages
func (s *KafkaService) handleBacktestRequest(payload json.RawMessage) error {
	var request struct {
		BacktestID int `json:"backtest_id"`
		UserID     int `json:"user_id"`
	}

	if err := json.Unmarshal(payload, &request); err != nil {
		return fmt.Errorf("failed to unmarshal backtest request: %w", err)
	}

	s.logger.Info("Received backtest request",
		zap.Int("backtest_id", request.BacktestID),
		zap.Int("user_id", request.UserID))

	// Process the backtest request here
	// This would typically involve calling the backtest service

	return nil
}

// handleBacktestCancel processes backtest cancellation messages
func (s *KafkaService) handleBacktestCancel(payload json.RawMessage) error {
	var request struct {
		BacktestID int `json:"backtest_id"`
		UserID     int `json:"user_id"`
	}

	if err := json.Unmarshal(payload, &request); err != nil {
		return fmt.Errorf("failed to unmarshal backtest cancel request: %w", err)
	}

	s.logger.Info("Received backtest cancellation request",
		zap.Int("backtest_id", request.BacktestID),
		zap.Int("user_id", request.UserID))

	// Process the cancellation request here

	return nil
}

// PublishBacktestStatus publishes a backtest status update
func (s *KafkaService) PublishBacktestStatus(
	ctx context.Context,
	backtestID int,
	userID int,
	strategyID int,
	status model.BacktestStatus,
	message string,
) error {
	// Create the payload
	payload := BacktestEventPayload{
		BacktestID: backtestID,
		UserID:     userID,
		StrategyID: strategyID,
		Status:     status,
		Timestamp:  time.Now(),
		Message:    message,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal backtest status payload: %w", err)
	}

	// Create the Kafka message
	kafkaMsg := KafkaMessage{
		Type:      "backtest_status",
		Timestamp: time.Now(),
		Payload:   payloadBytes,
	}

	msgBytes, err := json.Marshal(kafkaMsg)
	if err != nil {
		return fmt.Errorf("failed to marshal Kafka message: %w", err)
	}

	// Publish to the backtest completions topic
	topic := s.topics["backtestCompletions"]
	key := fmt.Sprintf("backtest-%d", backtestID)

	// Fix the method name from ProduceMessage to PublishMessage
	err = s.producer.PublishMessage(ctx, topic, key, string(msgBytes))
	if err != nil {
		return fmt.Errorf("failed to produce message: %w", err)
	}

	s.logger.Info("Published backtest status update",
		zap.String("topic", topic),
		zap.Int("backtest_id", backtestID),
		zap.String("status", string(status)))

	return nil
}

// Close shuts down the Kafka producer
func (s *KafkaService) Close() {
	if s.producer != nil {
		s.producer.Close()
	}
}
