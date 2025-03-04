package kafka

import (
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

// Config holds Kafka configuration
type Config struct {
	Brokers         []string      `json:"brokers"`
	ClientID        string        `json:"client_id"`
	ProducerTimeout time.Duration `json:"producer_timeout"`
	ConsumerGroupID string        `json:"consumer_group_id"`
	Topics          Topics        `json:"topics"`
}

// Topics defines the Kafka topics used in the application
type Topics struct {
	UserNotifications string `json:"user_notifications"`
	StrategyUpdates   string `json:"strategy_updates"`
	BacktestResults   string `json:"backtest_results"`
	MarketData        string `json:"market_data"`
}

// CreateTopic creates a Kafka topic if it doesn't exist
func CreateTopic(brokers []string, topic string, partitions, replicationFactor int, logger *zap.Logger) error {
	conn, err := kafka.Dial("tcp", brokers[0])
	if err != nil {
		return fmt.Errorf("failed to connect to kafka: %w", err)
	}
	defer conn.Close()

	controller, err := conn.Controller()
	if err != nil {
		return fmt.Errorf("failed to get controller: %w", err)
	}

	controllerConn, err := kafka.Dial("tcp", fmt.Sprintf("%s:%d", controller.Host, controller.Port))
	if err != nil {
		return fmt.Errorf("failed to connect to controller: %w", err)
	}
	defer controllerConn.Close()

	topicConfigs := []kafka.TopicConfig{
		{
			Topic:             topic,
			NumPartitions:     partitions,
			ReplicationFactor: replicationFactor,
		},
	}

	err = controllerConn.CreateTopics(topicConfigs...)
	if err != nil {
		// Check if the error is because the topic already exists
		if err.Error() == "kafka server: Topic already exists" {
			logger.Info("topic already exists", zap.String("topic", topic))
			return nil
		}
		return fmt.Errorf("failed to create topic: %w", err)
	}

	logger.Info("topic created successfully", zap.String("topic", topic))
	return nil
}
