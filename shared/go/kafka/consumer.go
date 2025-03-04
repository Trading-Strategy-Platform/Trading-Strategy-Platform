package kafka

import (
	"context"
	"time"

	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

// MessageHandler is a function that processes a Kafka message
type MessageHandler func(ctx context.Context, message []byte) error

// Consumer is a wrapper around kafka-go Reader
type Consumer struct {
	reader  *kafka.Reader
	handler MessageHandler
	logger  *zap.Logger
	running bool
}

// ConsumerConfig holds the configuration for a Kafka consumer
type ConsumerConfig struct {
	Brokers        []string
	Topic          string
	GroupID        string
	MinBytes       int
	MaxBytes       int
	CommitInterval time.Duration
	StartOffset    int64
	MaxWait        time.Duration
	ReadBackoffMin time.Duration
	ReadBackoffMax time.Duration
	Logger         *zap.Logger
	MaxRetries     int
	RetryBackoff   time.Duration
}

// NewConsumer creates a new Kafka consumer
func NewConsumer(config ConsumerConfig, handler MessageHandler) *Consumer {
	// Set default values if not provided
	if config.MinBytes == 0 {
		config.MinBytes = 10e3 // 10KB
	}
	if config.MaxBytes == 0 {
		config.MaxBytes = 10e6 // 10MB
	}
	if config.StartOffset == 0 {
		config.StartOffset = kafka.FirstOffset
	}
	if config.MaxWait == 0 {
		config.MaxWait = 500 * time.Millisecond
	}
	if config.ReadBackoffMin == 0 {
		config.ReadBackoffMin = 100 * time.Millisecond
	}
	if config.ReadBackoffMax == 0 {
		config.ReadBackoffMax = 1 * time.Second
	}
	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}
	if config.RetryBackoff == 0 {
		config.RetryBackoff = 500 * time.Millisecond
	}

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        config.Brokers,
		Topic:          config.Topic,
		GroupID:        config.GroupID,
		MinBytes:       config.MinBytes,
		MaxBytes:       config.MaxBytes,
		CommitInterval: config.CommitInterval,
		StartOffset:    config.StartOffset,
		MaxWait:        config.MaxWait,
		ReadBackoffMin: config.ReadBackoffMin,
		ReadBackoffMax: config.ReadBackoffMax,
		Logger:         kafka.LoggerFunc(config.Logger.Sugar().Debugf),
		ErrorLogger:    kafka.LoggerFunc(config.Logger.Sugar().Errorf),
	})

	return &Consumer{
		reader:  reader,
		handler: handler,
		logger:  config.Logger,
	}
}

// Start starts consuming messages from Kafka
func (c *Consumer) Start(ctx context.Context) {
	c.running = true
	go func() {
		c.logger.Info("kafka consumer started", zap.String("topic", c.reader.Config().Topic))
		for c.running {
			message, err := c.reader.FetchMessage(ctx)
			if err != nil {
				if ctx.Err() != nil {
					c.logger.Info("context cancelled, stopping consumer")
					break
				}
				c.logger.Error("error fetching message", zap.Error(err))
				time.Sleep(1 * time.Second) // Backoff on error
				continue
			}

			c.logger.Debug("received message from kafka",
				zap.String("topic", message.Topic),
				zap.String("key", string(message.Key)),
				zap.Int("partition", message.Partition),
				zap.Int64("offset", message.Offset),
			)

			// Handle the message
			if err := c.handler(ctx, message.Value); err != nil {
				c.logger.Error("error handling message",
					zap.Error(err),
					zap.String("topic", message.Topic),
					zap.String("key", string(message.Key)),
				)
			}

			// Commit the message
			if err := c.reader.CommitMessages(ctx, message); err != nil {
				c.logger.Error("error committing message", zap.Error(err))
			}
		}
	}()
}

// Stop stops the consumer
func (c *Consumer) Stop() error {
	c.running = false
	return c.reader.Close()
}

// IsRunning returns whether the consumer is running
func (c *Consumer) IsRunning() bool {
	return c.running
}
