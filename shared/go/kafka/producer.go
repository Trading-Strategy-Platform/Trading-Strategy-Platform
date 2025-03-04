package kafka

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

// Producer is a wrapper around kafka-go Writer
type Producer struct {
	writer *kafka.Writer
	logger *zap.Logger
}

// NewProducer creates a new Kafka producer
func NewProducer(brokers []string, logger *zap.Logger) *Producer {
	writer := &kafka.Writer{
		Addr:     kafka.TCP(brokers...),
		Balancer: &kafka.LeastBytes{},
		Async:    false, // Use synchronous writes for reliability
	}

	return &Producer{
		writer: writer,
		logger: logger,
	}
}

// PublishMessage publishes a message to a Kafka topic
func (p *Producer) PublishMessage(ctx context.Context, topic string, key string, value interface{}) error {
	// Convert value to JSON bytes
	jsonValue, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal message value: %w", err)
	}

	message := kafka.Message{
		Topic: topic,
		Key:   []byte(key),
		Value: jsonValue,
	}

	err = p.writer.WriteMessages(ctx, message)
	if err != nil {
		p.logger.Error("failed to write message to kafka",
			zap.String("topic", topic),
			zap.String("key", key),
			zap.Error(err),
		)
		return fmt.Errorf("failed to write message to kafka: %w", err)
	}

	p.logger.Debug("message published to kafka",
		zap.String("topic", topic),
		zap.String("key", key),
	)

	return nil
}

// Close closes the Kafka producer
func (p *Producer) Close() error {
	return p.writer.Close()
}

// IsHealthy checks if the Kafka producer is healthy
func (p *Producer) IsHealthy() bool {
	// Simple check if the writer is initialized
	return p.writer != nil
}
