package models

import (
	"encoding/json"
	"time"
)

// AlertRule represents a user-defined alert rule
type AlertRule struct {
	ID              string          `json:"id" db:"id"`
	UserID          string          `json:"user_id" db:"user_id"`
	WatchListID     string          `json:"watch_list_id" db:"watch_list_id"`
	WatchListItemID *string         `json:"watch_list_item_id,omitempty" db:"watch_list_item_id"`
	Symbol          string          `json:"symbol" db:"symbol"`
	AlertType       string          `json:"alert_type" db:"alert_type"`
	Conditions      json.RawMessage `json:"conditions" db:"conditions"`
	IsActive        bool            `json:"is_active" db:"is_active"`
	Frequency       string          `json:"frequency" db:"frequency"`
	NotifyEmail     bool            `json:"notify_email" db:"notify_email"`
	NotifyInApp     bool            `json:"notify_in_app" db:"notify_in_app"`
	Name            string          `json:"name" db:"name"`
	Description     *string         `json:"description,omitempty" db:"description"`
	LastTriggeredAt *time.Time      `json:"last_triggered_at,omitempty" db:"last_triggered_at"`
	TriggerCount    int             `json:"trigger_count" db:"trigger_count"`
	CreatedAt       time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at" db:"updated_at"`
}

// AlertLog represents a triggered alert instance
type AlertLog struct {
	ID                 string          `json:"id" db:"id"`
	AlertRuleID        string          `json:"alert_rule_id" db:"alert_rule_id"`
	UserID             string          `json:"user_id" db:"user_id"`
	Symbol             string          `json:"symbol" db:"symbol"`
	TriggeredAt        time.Time       `json:"triggered_at" db:"triggered_at"`
	AlertType          string          `json:"alert_type" db:"alert_type"`
	ConditionMet       json.RawMessage `json:"condition_met" db:"condition_met"`
	MarketData         json.RawMessage `json:"market_data" db:"market_data"`
	NotificationSent   bool            `json:"notification_sent" db:"notification_sent"`
	NotificationSentAt *time.Time      `json:"notification_sent_at,omitempty" db:"notification_sent_at"`
	NotificationError  *string         `json:"notification_error,omitempty" db:"notification_error"`
	IsRead             bool            `json:"is_read" db:"is_read"`
	ReadAt             *time.Time      `json:"read_at,omitempty" db:"read_at"`
	IsDismissed        bool            `json:"is_dismissed" db:"is_dismissed"`
	DismissedAt        *time.Time      `json:"dismissed_at,omitempty" db:"dismissed_at"`
}

// Condition type definitions for type safety
type PriceAboveCondition struct {
	Threshold  float64 `json:"threshold"`
	Comparison string  `json:"comparison"` // "above", "below"
}

type PriceChangeCondition struct {
	PercentChange float64 `json:"percent_change"`
	Period        string  `json:"period"`    // "1d", "1w", "1m"
	Direction     string  `json:"direction"` // "up", "down", "either"
}

type VolumeCondition struct {
	Threshold  float64 `json:"threshold"`
	Comparison string  `json:"comparison"` // "above", "below"
}

type VolumeSpikeCondition struct {
	VolumeMultiplier float64 `json:"volume_multiplier"`
	Baseline         string  `json:"baseline"` // "avg_30d", "avg_90d"
}

type NewsCondition struct {
	Keywords  []string `json:"keywords,omitempty"`
	Sentiment string   `json:"sentiment,omitempty"` // "positive", "negative", "neutral", "any"
}

// CreateAlertRuleRequest is the API request for creating alerts
type CreateAlertRuleRequest struct {
	WatchListID string          `json:"watch_list_id" binding:"required,max=100"`
	Symbol      string          `json:"symbol" binding:"required,min=1,max=20"`
	AlertType   string          `json:"alert_type" binding:"required,oneof=price_above price_below price_change volume_above volume_spike news earnings sec_filing"`
	Conditions  json.RawMessage `json:"conditions" binding:"required"`
	Name        string          `json:"name" binding:"required,min=1,max=255"`
	Description *string         `json:"description,omitempty" binding:"omitempty,max=5000"`
	Frequency   string          `json:"frequency" binding:"required,oneof=once always daily weekly"`
	NotifyEmail bool            `json:"notify_email"`
	NotifyInApp bool            `json:"notify_in_app"`
}

// UpdateAlertRuleRequest is the API request for updating alerts
type UpdateAlertRuleRequest struct {
	Name        *string         `json:"name,omitempty" binding:"omitempty,min=1,max=255"`
	Description *string         `json:"description,omitempty" binding:"omitempty,max=5000"`
	Conditions  json.RawMessage `json:"conditions,omitempty"`
	IsActive    *bool           `json:"is_active,omitempty"`
	Frequency   *string         `json:"frequency,omitempty" binding:"omitempty,oneof=once always daily weekly"`
	NotifyEmail *bool           `json:"notify_email,omitempty"`
	NotifyInApp *bool           `json:"notify_in_app,omitempty"`
}

// AlertRuleWithDetails includes related watch list info
type AlertRuleWithDetails struct {
	AlertRule
	WatchListName string `json:"watch_list_name"`
	CompanyName   string `json:"company_name,omitempty"`
}

// AlertLogWithRule includes the rule that triggered it
type AlertLogWithRule struct {
	AlertLog
	RuleName string `json:"rule_name"`
}

// BulkCreateAlertRequest is the API request for creating alerts for all tickers in a watchlist
type BulkCreateAlertRequest struct {
	WatchListID string          `json:"watch_list_id" binding:"required"`
	AlertType   string          `json:"alert_type" binding:"required,oneof=price_above price_below price_change volume_above volume_spike"`
	Conditions  json.RawMessage `json:"conditions" binding:"required"`
	Frequency   string          `json:"frequency" binding:"required,oneof=once always daily"`
	NotifyEmail bool            `json:"notify_email"`
	NotifyInApp bool            `json:"notify_in_app"`
}

// BulkCreateAlertResponse reports how many alerts were created vs skipped
type BulkCreateAlertResponse struct {
	Created int `json:"created"`
	Skipped int `json:"skipped"`
}

// alertTypeLabels maps alert type identifiers to human-readable labels.
var alertTypeLabels = map[string]string{
	"price_above":         "Price Above",
	"price_below":         "Price Below",
	"price_change_pct":    "Price Change %",
	"price_change_amount": "Price Change $",
	"volume_spike":        "Volume Spike",
	"unusual_volume":      "Unusual Volume",
	"volume_above":        "Volume Above",
	"volume_below":        "Volume Below",
	"news":                "News Alert",
	"earnings":            "Earnings Report",
	"dividend":            "Dividend",
	"sec_filing":          "SEC Filing",
	"analyst_rating":      "Analyst Rating",
}

// AlertTypeLabel returns a human-readable label for the given alert type.
// Returns the raw alertType string if no mapping exists.
func AlertTypeLabel(alertType string) string {
	if label, ok := alertTypeLabels[alertType]; ok {
		return label
	}
	return alertType
}
