package kafka

import (
	"context"
	"encoding/json"
	"time"

	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

// Producer handles producing messages to Kafka topics
type Producer struct {
	writers  map[string]*kafka.Writer
	brokers  []string
	clientID string
	logger   *zap.Logger
}

// Message represents a Kafka message to be sent
type Message struct {
	Key     string
	Value   interface{}
	Headers []kafka.Header
}

// NewProducer creates a new Kafka producer
func NewProducer(brokers []string, clientID string, logger *zap.Logger) *Producer {
	return &Producer{
		writers:  make(map[string]*kafka.Writer),
		brokers:  brokers,
		clientID: clientID,
		logger:   logger,
	}
}

// getWriter returns a Kafka writer for the specified topic
func (p *Producer) getWriter(topic string) *kafka.Writer {
	if writer, exists := p.writers[topic]; exists {
		return writer
	}

	writer := &kafka.Writer{
		Addr:         kafka.TCP(p.brokers...),
		Topic:        topic,
		Balancer:     &kafka.LeastBytes{},
		BatchSize:    100,
		BatchTimeout: 10 * time.Millisecond,
		RequiredAcks: kafka.RequireOne,
		Async:        false,
		Transport: &kafka.Transport{
			ClientID: p.clientID,
		},
	}

	p.writers[topic] = writer
	return writer
}

// Publish sends a message to a Kafka topic
func (p *Producer) Publish(ctx context.Context, topic string, msg Message) error {
	// Get or create Kafka writer for the topic
	writer := p.getWriter(topic)

	// Marshal the message value to JSON
	jsonValue, err := json.Marshal(msg.Value)
	if err != nil {
		p.logger.Error("Failed to marshal message",
			zap.String("topic", topic),
			zap.Error(err))
		return err
	}

	// Create Kafka message
	kafkaMsg := kafka.Message{
		Key:     []byte(msg.Key),
		Value:   jsonValue,
		Headers: msg.Headers,
		Time:    time.Now(),
	}

	// Write the message
	err = writer.WriteMessages(ctx, kafkaMsg)
	if err != nil {
		p.logger.Error("Failed to publish message",
			zap.String("topic", topic),
			zap.String("key", msg.Key),
			zap.Error(err))
		return err
	}

	p.logger.Debug("Message published",
		zap.String("topic", topic),
		zap.String("key", msg.Key))

	return nil
}

// Close closes all Kafka writers
func (p *Producer) Close() error {
	for topic, writer := range p.writers {
		if err := writer.Close(); err != nil {
			p.logger.Error("Failed to close Kafka writer",
				zap.String("topic", topic),
				zap.Error(err))
		}
	}
	return nil
}
