package models

import (
	"encoding/json"
	"time"
)

// NotificationPreferences stores user notification settings
type NotificationPreferences struct {
	ID                            string    `json:"id" db:"id"`
	UserID                        string    `json:"user_id" db:"user_id"`
	EmailEnabled                  bool      `json:"email_enabled" db:"email_enabled"`
	EmailAddress                  *string   `json:"email_address,omitempty" db:"email_address"`
	EmailVerified                 bool      `json:"email_verified" db:"email_verified"`
	PriceAlertsEnabled            bool      `json:"price_alerts_enabled" db:"price_alerts_enabled"`
	VolumeAlertsEnabled           bool      `json:"volume_alerts_enabled" db:"volume_alerts_enabled"`
	NewsAlertsEnabled             bool      `json:"news_alerts_enabled" db:"news_alerts_enabled"`
	EarningsAlertsEnabled         bool      `json:"earnings_alerts_enabled" db:"earnings_alerts_enabled"`
	SECFilingAlertsEnabled        bool      `json:"sec_filing_alerts_enabled" db:"sec_filing_alerts_enabled"`
	DailyDigestEnabled            bool      `json:"daily_digest_enabled" db:"daily_digest_enabled"`
	DailyDigestTime               string    `json:"daily_digest_time" db:"daily_digest_time"`
	WeeklyDigestEnabled           bool      `json:"weekly_digest_enabled" db:"weekly_digest_enabled"`
	WeeklyDigestDay               int       `json:"weekly_digest_day" db:"weekly_digest_day"`
	WeeklyDigestTime              string    `json:"weekly_digest_time" db:"weekly_digest_time"`
	DigestIncludePortfolioSummary bool      `json:"digest_include_portfolio_summary" db:"digest_include_portfolio_summary"`
	DigestIncludeTopMovers        bool      `json:"digest_include_top_movers" db:"digest_include_top_movers"`
	DigestIncludeRecentAlerts     bool      `json:"digest_include_recent_alerts" db:"digest_include_recent_alerts"`
	DigestIncludeNewsHighlights   bool      `json:"digest_include_news_highlights" db:"digest_include_news_highlights"`
	QuietHoursEnabled             bool      `json:"quiet_hours_enabled" db:"quiet_hours_enabled"`
	QuietHoursStart               string    `json:"quiet_hours_start" db:"quiet_hours_start"`
	QuietHoursEnd                 string    `json:"quiet_hours_end" db:"quiet_hours_end"`
	QuietHoursTimezone            string    `json:"quiet_hours_timezone" db:"quiet_hours_timezone"`
	MaxAlertsPerDay               int       `json:"max_alerts_per_day" db:"max_alerts_per_day"`
	MaxEmailsPerDay               int       `json:"max_emails_per_day" db:"max_emails_per_day"`
	CreatedAt                     time.Time `json:"created_at" db:"created_at"`
	UpdatedAt                     time.Time `json:"updated_at" db:"updated_at"`
}

// UpdateNotificationPreferencesRequest is the API request
type UpdateNotificationPreferencesRequest struct {
	EmailEnabled                  *bool   `json:"email_enabled,omitempty"`
	EmailAddress                  *string `json:"email_address,omitempty" binding:"omitempty,email,max=254"`
	PriceAlertsEnabled            *bool   `json:"price_alerts_enabled,omitempty"`
	VolumeAlertsEnabled           *bool   `json:"volume_alerts_enabled,omitempty"`
	NewsAlertsEnabled             *bool   `json:"news_alerts_enabled,omitempty"`
	EarningsAlertsEnabled         *bool   `json:"earnings_alerts_enabled,omitempty"`
	SECFilingAlertsEnabled        *bool   `json:"sec_filing_alerts_enabled,omitempty"`
	DailyDigestEnabled            *bool   `json:"daily_digest_enabled,omitempty"`
	DailyDigestTime               *string `json:"daily_digest_time,omitempty" binding:"omitempty,max=10"`
	WeeklyDigestEnabled           *bool   `json:"weekly_digest_enabled,omitempty"`
	WeeklyDigestDay               *int    `json:"weekly_digest_day,omitempty" binding:"omitempty,min=0,max=6"`
	WeeklyDigestTime              *string `json:"weekly_digest_time,omitempty" binding:"omitempty,max=10"`
	DigestIncludePortfolioSummary *bool   `json:"digest_include_portfolio_summary,omitempty"`
	DigestIncludeTopMovers        *bool   `json:"digest_include_top_movers,omitempty"`
	DigestIncludeRecentAlerts     *bool   `json:"digest_include_recent_alerts,omitempty"`
	DigestIncludeNewsHighlights   *bool   `json:"digest_include_news_highlights,omitempty"`
	QuietHoursEnabled             *bool   `json:"quiet_hours_enabled,omitempty"`
	QuietHoursStart               *string `json:"quiet_hours_start,omitempty" binding:"omitempty,max=10"`
	QuietHoursEnd                 *string `json:"quiet_hours_end,omitempty" binding:"omitempty,max=10"`
	QuietHoursTimezone            *string `json:"quiet_hours_timezone,omitempty" binding:"omitempty,max=100"`
}

// InAppNotification represents in-app notification
type InAppNotification struct {
	ID          string          `json:"id" db:"id"`
	UserID      string          `json:"user_id" db:"user_id"`
	AlertLogID  *string         `json:"alert_log_id,omitempty" db:"alert_log_id"`
	Type        string          `json:"type" db:"type"`
	Title       string          `json:"title" db:"title"`
	Message     string          `json:"message" db:"message"`
	Metadata    json.RawMessage `json:"metadata,omitempty" db:"metadata"`
	IsRead      bool            `json:"is_read" db:"is_read"`
	ReadAt      *time.Time      `json:"read_at,omitempty" db:"read_at"`
	IsDismissed bool            `json:"is_dismissed" db:"is_dismissed"`
	DismissedAt *time.Time      `json:"dismissed_at,omitempty" db:"dismissed_at"`
	CreatedAt   time.Time       `json:"created_at" db:"created_at"`
	ExpiresAt   time.Time       `json:"expires_at" db:"expires_at"`
}

// DigestLog tracks sent digests
type DigestLog struct {
	ID              string          `json:"id" db:"id"`
	UserID          string          `json:"user_id" db:"user_id"`
	DigestType      string          `json:"digest_type" db:"digest_type"`
	PeriodStart     time.Time       `json:"period_start" db:"period_start"`
	PeriodEnd       time.Time       `json:"period_end" db:"period_end"`
	SentAt          time.Time       `json:"sent_at" db:"sent_at"`
	EmailSent       bool            `json:"email_sent" db:"email_sent"`
	EmailOpened     bool            `json:"email_opened" db:"email_opened"`
	EmailClicked    bool            `json:"email_clicked" db:"email_clicked"`
	ContentSnapshot json.RawMessage `json:"content_snapshot,omitempty" db:"content_snapshot"`
}

// DigestContent represents digest email content
type DigestContent struct {
	UserName         string             `json:"user_name"`
	PeriodStart      time.Time          `json:"period_start"`
	PeriodEnd        time.Time          `json:"period_end"`
	PortfolioSummary *PortfolioSummary  `json:"portfolio_summary,omitempty"`
	TopMovers        []TopMover         `json:"top_movers,omitempty"`
	RecentAlerts     []AlertLogWithRule `json:"recent_alerts,omitempty"`
	NewsHighlights   []NewsHighlight    `json:"news_highlights,omitempty"`
}

type PortfolioSummary struct {
	TotalValue    float64 `json:"total_value"`
	DayChange     float64 `json:"day_change"`
	DayChangePct  float64 `json:"day_change_pct"`
	WeekChange    float64 `json:"week_change"`
	WeekChangePct float64 `json:"week_change_pct"`
}

type TopMover struct {
	Symbol    string  `json:"symbol"`
	Name      string  `json:"name"`
	Price     float64 `json:"price"`
	ChangePct float64 `json:"change_pct"`
	Direction string  `json:"direction"` // "up" or "down"
}

type NewsHighlight struct {
	Symbol      string    `json:"symbol"`
	Title       string    `json:"title"`
	Summary     string    `json:"summary"`
	PublishedAt time.Time `json:"published_at"`
	URL         string    `json:"url"`
}
