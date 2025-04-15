package model

import (
	"time"
)

// MarketplaceItem represents a strategy listed in the marketplace
type MarketplaceItem struct {
	ID                 int        `json:"id" db:"id"`
	StrategyID         int        `json:"strategy_id" db:"strategy_id"`
	UserID             int        `json:"user_id" db:"user_id"`
	Price              float64    `json:"price" db:"price"`
	IsSubscription     bool       `json:"is_subscription" db:"is_subscription"`
	SubscriptionPeriod string     `json:"subscription_period,omitempty" db:"subscription_period"`
	IsActive           bool       `json:"is_active" db:"is_active"`
	DescriptionPublic  string     `json:"description_public" db:"description_public"`
	CreatedAt          time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt          *time.Time `json:"updated_at,omitempty" db:"updated_at"`

	// Additional fields for responses
	Strategy        *Strategy `json:"strategy,omitempty" db:"-"`
	Name            string    `json:"name,omitempty" db:"name"`                   // Added strategy name
	ThumbnailURL    string    `json:"thumbnail_url,omitempty" db:"thumbnail_url"` // Added thumbnail URL
	CreatorName     string    `json:"creator_name,omitempty" db:"-"`
	CreatorPhotoURL string    `json:"creator_photo_url,omitempty" db:"-"` // Added creator photo URL
	AverageRating   float64   `json:"average_rating,omitempty" db:"-"`
	ReviewsCount    int       `json:"reviews_count,omitempty" db:"-"`
	PurchasesCount  int       `json:"purchases_count,omitempty" db:"-"`
}

// MarketplaceCreate represents data needed to create a marketplace listing
type MarketplaceCreate struct {
	StrategyID         int     `json:"strategy_id" binding:"required"`
	VersionID          int     `json:"version_id" binding:"required"`
	Price              float64 `json:"price" binding:"min=0"`
	IsSubscription     bool    `json:"is_subscription"`
	SubscriptionPeriod string  `json:"subscription_period,omitempty"`
	DescriptionPublic  string  `json:"description_public"`
}

// StrategyPurchase represents a purchase of a strategy from the marketplace
type StrategyPurchase struct {
	ID              int        `json:"id" db:"id"`
	MarketplaceID   int        `json:"marketplace_id" db:"marketplace_id"`
	BuyerID         int        `json:"buyer_id" db:"buyer_id"`
	PurchasePrice   float64    `json:"purchase_price" db:"purchase_price"`
	SubscriptionEnd *time.Time `json:"subscription_end,omitempty" db:"subscription_end"`
	CreatedAt       time.Time  `json:"created_at" db:"created_at"`
}

// StrategyReview represents a review of a purchased strategy
type StrategyReview struct {
	ID            int        `json:"id" db:"id"`
	MarketplaceID int        `json:"marketplace_id" db:"marketplace_id"`
	UserID        int        `json:"user_id" db:"user_id"`
	Rating        int        `json:"rating" db:"rating"`
	Comment       string     `json:"comment" db:"comment"`
	CreatedAt     time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt     *time.Time `json:"updated_at,omitempty" db:"updated_at"`

	// Additional fields for responses
	UserName string `json:"user_name,omitempty" db:"-"`
}

// ReviewCreate represents data needed to create a strategy review
type ReviewCreate struct {
	MarketplaceID int    `json:"marketplace_id" binding:"required"`
	Rating        int    `json:"rating" binding:"required,min=1,max=5"`
	Comment       string `json:"comment"`
}
