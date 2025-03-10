package service

import (
	"context"
	"fmt"
	"time"

	"services/strategy-service/internal/config"
	"services/strategy-service/internal/model"

	"github.com/yourorg/trading-platform/shared/go/kafka"
	"go.uber.org/zap"
)

// KafkaService handles Kafka event publishing for the strategy service
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

// PublishStrategyCreated publishes a strategy created event
func (s *KafkaService) PublishStrategyCreated(ctx context.Context, strategy *model.Strategy) error {
	event := map[string]interface{}{
		"event_type":  "strategy_created",
		"strategy_id": strategy.ID,
		"user_id":     strategy.UserID,
		"name":        strategy.Name,
		"is_public":   strategy.IsPublic,
		"version":     strategy.Version,
		"created_at":  strategy.CreatedAt.Unix(),
	}

	return s.publishStrategyEvent(ctx, fmt.Sprintf("strategy-%d", strategy.ID), event)
}

// PublishStrategyUpdated publishes a strategy updated event
func (s *KafkaService) PublishStrategyUpdated(ctx context.Context, strategy *model.Strategy) error {
	event := map[string]interface{}{
		"event_type":  "strategy_updated",
		"strategy_id": strategy.ID,
		"user_id":     strategy.UserID,
		"name":        strategy.Name,
		"is_public":   strategy.IsPublic,
		"version":     strategy.Version,
		"updated_at":  strategy.UpdatedAt.Unix(),
	}

	return s.publishStrategyEvent(ctx, fmt.Sprintf("strategy-%d", strategy.ID), event)
}

// PublishBacktestCreated publishes a backtest created event
func (s *KafkaService) PublishBacktestCreated(ctx context.Context, backtest *model.BacktestRequest, backtestID int, userID int) error {
	event := map[string]interface{}{
		"event_type":      "backtest_created",
		"backtest_id":     backtestID,
		"strategy_id":     backtest.StrategyID,
		"user_id":         userID,
		"symbol_id":       backtest.SymbolID,
		"timeframe_id":    backtest.TimeframeID,
		"start_date":      backtest.StartDate.Unix(),
		"end_date":        backtest.EndDate.Unix(),
		"initial_capital": backtest.InitialCapital,
		"timestamp":       time.Now().Unix(),
	}

	return s.publishStrategyEvent(ctx, fmt.Sprintf("backtest-%d", backtestID), event)
}

// PublishMarketplaceListing publishes a marketplace listing event
func (s *KafkaService) PublishMarketplaceListing(ctx context.Context, listing *model.MarketplaceListing) error {
	event := map[string]interface{}{
		"event_type":      "marketplace_listing_created",
		"marketplace_id":  listing.ID,
		"strategy_id":     listing.StrategyID,
		"user_id":         listing.CreatorID,
		"price":           listing.Price,
		"is_subscription": listing.IsSubscription,
		"created_at":      listing.CreatedAt.Unix(),
	}

	return s.publishMarketplaceEvent(ctx, fmt.Sprintf("marketplace-%d", listing.ID), event)
}

// PublishPurchaseCreated publishes a strategy purchase event
func (s *KafkaService) PublishPurchaseCreated(ctx context.Context, purchase *model.StrategyPurchase) error {
	event := map[string]interface{}{
		"event_type":       "strategy_purchased",
		"purchase_id":      purchase.ID,
		"marketplace_id":   purchase.MarketplaceID,
		"buyer_id":         purchase.BuyerID,
		"purchase_price":   purchase.PurchasePrice,
		"subscription_end": nil,
		"created_at":       purchase.CreatedAt.Unix(),
	}

	if purchase.SubscriptionEnd != nil {
		event["subscription_end"] = purchase.SubscriptionEnd.Unix()
	}

	return s.publishMarketplaceEvent(ctx, fmt.Sprintf("purchase-%d", purchase.ID), event)
}

// PublishReviewCreated publishes a strategy review event
func (s *KafkaService) PublishReviewCreated(ctx context.Context, review *model.StrategyReview) error {
	event := map[string]interface{}{
		"event_type":     "review_created",
		"review_id":      review.ID,
		"marketplace_id": review.MarketplaceID,
		"user_id":        review.UserID,
		"rating":         review.Rating,
		"created_at":     review.CreatedAt.Unix(),
	}

	return s.publishMarketplaceEvent(ctx, fmt.Sprintf("review-%d", review.ID), event)
}

// publishStrategyEvent publishes an event to the strategy events topic
func (s *KafkaService) publishStrategyEvent(ctx context.Context, key string, event map[string]interface{}) error {
	topic, exists := s.topics["strategyEvents"]
	if !exists {
		topic = "strategy-events" // fallback
	}

	err := s.producer.PublishMessage(ctx, topic, key, event)
	if err != nil {
		s.logger.Error("Failed to publish strategy event to Kafka",
			zap.String("topic", topic),
			zap.String("key", key),
			zap.Error(err))
		return err
	}

	return nil
}

// publishMarketplaceEvent publishes an event to the marketplace events topic
func (s *KafkaService) publishMarketplaceEvent(ctx context.Context, key string, event map[string]interface{}) error {
	topic, exists := s.topics["marketplaceEvents"]
	if !exists {
		topic = "marketplace-events" // fallback
	}

	err := s.producer.PublishMessage(ctx, topic, key, event)
	if err != nil {
		s.logger.Error("Failed to publish marketplace event to Kafka",
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
