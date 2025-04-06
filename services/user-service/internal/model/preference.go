package model

import (
	"encoding/json"
)

// UserPreferences represents user preferences
type UserPreferences struct {
	Theme                string          `json:"theme,omitempty" db:"theme"`
	DefaultTimeframe     string          `json:"default_timeframe,omitempty" db:"default_timeframe"`
	ChartPreferences     json.RawMessage `json:"chart_preferences,omitempty" db:"chart_preferences"`
	NotificationSettings json.RawMessage `json:"notification_settings,omitempty" db:"notification_settings"`
}

// PreferencesUpdate represents data for updating user preferences
type PreferencesUpdate struct {
	Theme                *string         `json:"theme,omitempty"`
	DefaultTimeframe     *string         `json:"default_timeframe,omitempty"`
	ChartPreferences     json.RawMessage `json:"chart_preferences,omitempty"`
	NotificationSettings json.RawMessage `json:"notification_settings,omitempty"`
}

// ChartPreference represents chart-specific preferences
type ChartPreference struct {
	ShowVolume    bool     `json:"show_volume"`
	ShowGrid      bool     `json:"show_grid"`
	ShowLegend    bool     `json:"show_legend"`
	DefaultColors []string `json:"default_colors,omitempty"`
	Indicators    []string `json:"indicators,omitempty"`
}

// NotificationSetting represents notification settings
type NotificationSetting struct {
	EmailNotifications bool `json:"email_notifications"`
	PriceAlerts        bool `json:"price_alerts"`
	NewMessages        bool `json:"new_messages"`
	SystemUpdates      bool `json:"system_updates"`
	MarketingEmails    bool `json:"marketing_emails"`
}
