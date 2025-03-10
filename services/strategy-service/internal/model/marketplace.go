package model

import (
	"time"
)

// MarketplaceItem represents a strategy listing in the marketplace
type MarketplaceItem struct {
	ID                 int       `json:"id" db:"id"`
	StrategyID         int       `json:"strategy_id" db:"strategy_id"`
	UserID             int       `json:"user_id" db:"user_id"`
	CreatorID          int       `json:"creator_id" db:"creator_id"`
	Name               string    `json:"name" db:"name"`
	Description        string    `json:"description" db:"description"`
	Price              float64   `json:"price" db:"price"`
	IsSubscription     bool      `json:"is_subscription" db:"is_subscription"`
	SubscriptionPeriod string    `json:"subscription_period,omitempty" db:"subscription_period"`
	IsActive           bool      `json:"is_active" db:"is_active"`
	CreatedAt          time.Time `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time `json:"updated_at" db:"updated_at"`
	Strategy           *Strategy `json:"strategy,omitempty" db:"-"`
	CreatorName        string    `json:"creator_name,omitempty" db:"-"`
	AverageRating      float64   `json:"average_rating,omitempty" db:"-"`
	ReviewsCount       int       `json:"reviews_count,omitempty" db:"-"`
}

// MarketplaceListing represents a strategy listing in the marketplace
type MarketplaceListing struct {
	ID                 int       `json:"id" db:"id"`
	StrategyID         int       `json:"strategy_id" db:"strategy_id"`
	CreatorID          int       `json:"creator_id" db:"creator_id"`
	Name               string    `json:"name" db:"name"`
	Description        string    `json:"description" db:"description"`
	Price              float64   `json:"price" db:"price"`
	IsSubscription     bool      `json:"is_subscription" db:"is_subscription"`
	SubscriptionPeriod string    `json:"subscription_period,omitempty" db:"subscription_period"`
	Rating             float64   `json:"rating" db:"rating"`
	ReviewCount        int       `json:"review_count" db:"review_count"`
	CreatedAt          time.Time `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time `json:"updated_at" db:"updated_at"`
	CreatorName        string    `json:"creator_name,omitempty" db:"-"`
	Tags               []string  `json:"tags,omitempty" db:"-"`
	PreviewData        []byte    `json:"preview_data,omitempty" db:"preview_data"`
}
