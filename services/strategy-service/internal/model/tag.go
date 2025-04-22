package model

// Tag represents a strategy tag
type Tag struct {
	ID   int    `json:"id" db:"id"`
	Name string `json:"name" db:"name"`
}

// TagWithCount represents a tag with usage count
type TagWithCount struct {
	ID            int    `json:"id" db:"id"`
	Name          string `json:"name" db:"name"`
	StrategyCount int64  `json:"strategy_count" db:"strategy_count"`
}
