package model

import "time"

// CandleBatch represents a batch of candles for database import
type CandleBatch struct {
	SymbolID int       `json:"symbol_id"`
	Time     time.Time `json:"time"`
	Open     float64   `json:"open"`
	High     float64   `json:"high"`
	Low      float64   `json:"low"`
	Close    float64   `json:"close"`
	Volume   float64   `json:"volume"`
}
