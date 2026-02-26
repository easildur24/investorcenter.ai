package models

import (
	"encoding/json"
	"time"
)

// ---------------------------------------------------------------------------
// SNS Message Types (must match backend/models/price_event.go)
// ---------------------------------------------------------------------------

// PriceUpdateMessage is the SNS payload published by the backend every ~5 seconds.
type PriceUpdateMessage struct {
	Timestamp int64                   `json:"timestamp"`
	Source    string                  `json:"source"`
	Symbols  map[string]SymbolQuote  `json:"symbols"`
}

// SymbolQuote is a lightweight price snapshot for a single symbol.
type SymbolQuote struct {
	Price     float64 `json:"price"`
	Volume    int64   `json:"volume"`
	ChangePct float64 `json:"change_pct"`
}

// ---------------------------------------------------------------------------
// Database Models (subset of backend/models, only what the notification service needs)
// ---------------------------------------------------------------------------

// AlertRule represents a user-created alert rule from the alert_rules table.
type AlertRule struct {
	ID          string          `db:"id"`
	UserID      string          `db:"user_id"`
	WatchListID string          `db:"watch_list_id"`
	Symbol      string          `db:"symbol"`
	AlertType   string          `db:"alert_type"`
	Conditions  json.RawMessage `db:"conditions"`
	IsActive    bool            `db:"is_active"`
	// Frequency controls how often an alert can re-trigger:
	//   "once"   — triggers once, then deactivated (is_active=false)
	//   "daily"  — at most once per 24 hours
	//   "always" — on every evaluation cycle, with a 5-minute cooldown between
	//              notifications to prevent spam. Users selecting "always" will
	//              receive at most 1 notification per 5 minutes per alert rule.
	Frequency       string  `db:"frequency"`
	NotifyEmail     bool    `db:"notify_email"`
	NotifyInApp     bool    `db:"notify_in_app"`
	Name            string  `db:"name"`
	LastTriggeredAt *time.Time `db:"last_triggered_at"`
	TriggerCount    int        `db:"trigger_count"`
	CreatedAt       time.Time  `db:"created_at"`
	UpdatedAt       time.Time  `db:"updated_at"`
}

// AlertLog records a single alert trigger event.
type AlertLog struct {
	ID               string          `db:"id"`
	AlertRuleID      string          `db:"alert_rule_id"`
	UserID           string          `db:"user_id"`
	Symbol           string          `db:"symbol"`
	TriggeredAt      time.Time       `db:"triggered_at"`
	AlertType        string          `db:"alert_type"`
	ConditionMet     json.RawMessage `db:"condition_met"`
	MarketData       json.RawMessage `db:"market_data"`
	NotificationSent bool            `db:"notification_sent"`
	IsRead           bool            `db:"is_read"`
	IsDismissed      bool            `db:"is_dismissed"`
}

// NotificationPreferences holds user notification settings.
type NotificationPreferences struct {
	UserID             string `db:"user_id"`
	EmailEnabled       bool   `db:"email_enabled"`
	EmailAddress       *string `db:"email_address"`
	EmailVerified      bool   `db:"email_verified"`
	QuietHoursEnabled  bool   `db:"quiet_hours_enabled"`
	QuietHoursStart    string `db:"quiet_hours_start"`    // HH:MM:SS
	QuietHoursEnd      string `db:"quiet_hours_end"`      // HH:MM:SS
	QuietHoursTimezone string `db:"quiet_hours_timezone"` // e.g. "America/New_York"
	MaxAlertsPerDay    int    `db:"max_alerts_per_day"`
	MaxEmailsPerDay    int    `db:"max_emails_per_day"`
}

// UserEmail holds the minimal user data needed for email delivery.
type UserEmail struct {
	Email    string `db:"email"`
	FullName string `db:"full_name"`
}

// ---------------------------------------------------------------------------
// Condition Structs (parsed from AlertRule.Conditions JSON)
// ---------------------------------------------------------------------------

// ThresholdCondition covers price_above, price_below, volume_above, volume_below.
type ThresholdCondition struct {
	Threshold float64 `json:"threshold"`
}

// VolumeSpikeCondition covers the volume_spike alert type.
type VolumeSpikeCondition struct {
	VolumeMultiplier float64 `json:"volume_multiplier"`
	Baseline         string  `json:"baseline"` // "avg_30d"
}

// PriceChangeCondition covers the price_change_pct alert type.
type PriceChangeCondition struct {
	PercentChange float64 `json:"percent_change"`
	Direction     string  `json:"direction"` // "up", "down", "either"
}
