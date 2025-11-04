# Watchlist Alert System - Technical Specification
**Phases 4-7: Price Alerts, News Alerts, Notifications & Premium Features**

**Version:** 1.0
**Last Updated:** 2025-11-03
**Status:** Draft

---

## Table of Contents

1. [Overview](#overview)
2. [Architecture](#architecture)
3. [Database Schema](#database-schema)
4. [Backend Implementation](#backend-implementation)
5. [Frontend Implementation](#frontend-implementation)
6. [Alert Processing Engine](#alert-processing-engine)
7. [Notification System](#notification-system)
8. [External Integrations](#external-integrations)
9. [Premium Tier Implementation](#premium-tier-implementation)
10. [Deployment & Operations](#deployment--operations)
11. [Testing Strategy](#testing-strategy)
12. [Performance & Scaling](#performance--scaling)
13. [Security Considerations](#security-considerations)
14. [Implementation Timeline](#implementation-timeline)

---

## Overview

### Scope
This technical specification covers the implementation of:
- **Phase 4:** Price & Volume Alerts
- **Phase 5:** News & Financial Event Alerts
- **Phase 6:** Notification Preferences & Digests
- **Phase 7:** Premium Features & Polish

### Goals
1. Enable users to set automated alerts on watch list items
2. Deliver real-time notifications via email and in-app
3. Provide customizable notification preferences
4. Implement premium tier with advanced alert capabilities
5. Ensure scalable alert processing for 10,000+ concurrent users

### Dependencies
- **Existing:** Phases 1-3 (Auth, Watch Lists, Heatmaps)
- **External APIs:** Polygon.io (market data), SendGrid (email), Finnhub (news)
- **Infrastructure:** Redis (alert queue), PostgreSQL (alert storage), Kubernetes CronJobs

---

## Architecture

### System Components

```
┌─────────────────────────────────────────────────────────────────┐
│                         Frontend (Next.js)                       │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────┐  │
│  │ Alert Rules  │  │ Alert Logs   │  │ Notification Center  │  │
│  │ Management   │  │ History      │  │ (Bell Icon)          │  │
│  └──────────────┘  └──────────────┘  └──────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
                                 │
                                 │ REST API
                                 ▼
┌─────────────────────────────────────────────────────────────────┐
│                      Backend API (Go/Gin)                        │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────┐  │
│  │ Alert        │  │ Notification │  │ Digest               │  │
│  │ Handlers     │  │ Handlers     │  │ Handlers             │  │
│  └──────────────┘  └──────────────┘  └──────────────────────┘  │
│                                                                  │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────┐  │
│  │ Alert        │  │ Notification │  │ Email                │  │
│  │ Service      │  │ Service      │  │ Service              │  │
│  └──────────────┘  └──────────────┘  └──────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
                                 │
                    ┌────────────┴────────────┐
                    ▼                         ▼
         ┌──────────────────┐      ┌──────────────────┐
         │   PostgreSQL     │      │      Redis       │
         │   - alert_rules  │      │   - Alert Queue  │
         │   - alert_logs   │      │   - Rate Limits  │
         │   - notifications│      │   - Dedup Cache  │
         └──────────────────┘      └──────────────────┘
                                           │
                    ┌──────────────────────┘
                    ▼
┌─────────────────────────────────────────────────────────────────┐
│              Background Workers (Kubernetes CronJobs)            │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────┐  │
│  │ Price Alert  │  │ News Alert   │  │ Digest               │  │
│  │ Processor    │  │ Processor    │  │ Generator            │  │
│  │ (Every 1min) │  │ (Every 5min) │  │ (Daily/Weekly)       │  │
│  └──────────────┘  └──────────────┘  └──────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
                    │
                    ▼
         ┌──────────────────┐
         │  External APIs   │
         │  - Polygon.io    │
         │  - Finnhub       │
         │  - SendGrid      │
         └──────────────────┘
```

### Data Flow

#### Alert Creation Flow
```
User → Frontend → POST /api/v1/alerts → Backend Handler
                                            ↓
                                      Validate Rule
                                            ↓
                                      Check Tier Limits
                                            ↓
                                      Save to DB (alert_rules)
                                            ↓
                                      Return Alert ID
```

#### Alert Processing Flow
```
CronJob (1min) → Fetch Active Alerts → For Each Alert:
                                           ↓
                                     Fetch Current Data (Polygon API)
                                           ↓
                                     Evaluate Condition
                                           ↓
                                     Condition Met? → Yes → Create Notification
                                           │                      ↓
                                           No                 Save to alert_logs
                                           ↓                      ↓
                                      Next Alert            Send Email (SendGrid)
                                                                  ↓
                                                            Enqueue In-App Notif
```

---

## Database Schema

### Phase 4 & 5: Alert Tables

#### `alert_rules` Table
Stores user-defined alert rules for watch list items.

```sql
CREATE TABLE alert_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    watch_list_id UUID NOT NULL REFERENCES watch_lists(id) ON DELETE CASCADE,
    watch_list_item_id UUID REFERENCES watch_list_items(id) ON DELETE CASCADE,
    symbol VARCHAR(20) NOT NULL, -- Denormalized for quick lookups

    -- Alert Type
    alert_type VARCHAR(50) NOT NULL, -- 'price_above', 'price_below', 'price_change_pct',
                                     -- 'volume_spike', 'unusual_volume', 'news',
                                     -- 'earnings', 'dividend', 'sec_filing'

    -- Condition Parameters (JSONB for flexibility)
    conditions JSONB NOT NULL,
    -- Examples:
    -- {'threshold': 150.00, 'comparison': 'above'} for price_above
    -- {'percent_change': 5.0, 'period': '1d', 'direction': 'up'} for price_change_pct
    -- {'volume_multiplier': 2.0, 'baseline': 'avg_30d'} for volume_spike
    -- {'keywords': ['acquisition', 'merger'], 'sentiment': 'positive'} for news

    -- Alert Settings
    is_active BOOLEAN DEFAULT true,
    frequency VARCHAR(20) DEFAULT 'once', -- 'once', 'daily', 'always'
    notify_email BOOLEAN DEFAULT true,
    notify_in_app BOOLEAN DEFAULT true,

    -- Metadata
    name VARCHAR(255) NOT NULL, -- User-friendly alert name
    description TEXT,

    -- Tracking
    last_triggered_at TIMESTAMP WITH TIME ZONE,
    trigger_count INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    -- Constraints
    CONSTRAINT valid_alert_type CHECK (alert_type IN (
        'price_above', 'price_below', 'price_change_pct', 'price_change_amount',
        'volume_spike', 'unusual_volume', 'volume_above', 'volume_below',
        'news', 'earnings', 'dividend', 'sec_filing', 'analyst_rating'
    )),
    CONSTRAINT valid_frequency CHECK (frequency IN ('once', 'daily', 'always'))
);

-- Indexes for performance
CREATE INDEX idx_alert_rules_user_id ON alert_rules(user_id);
CREATE INDEX idx_alert_rules_watch_list_id ON alert_rules(watch_list_id);
CREATE INDEX idx_alert_rules_symbol ON alert_rules(symbol);
CREATE INDEX idx_alert_rules_active ON alert_rules(is_active) WHERE is_active = true;
CREATE INDEX idx_alert_rules_type ON alert_rules(alert_type);
CREATE INDEX idx_alert_rules_created_at ON alert_rules(created_at DESC);
```

#### `alert_logs` Table
Stores history of triggered alerts.

```sql
CREATE TABLE alert_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    alert_rule_id UUID NOT NULL REFERENCES alert_rules(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    symbol VARCHAR(20) NOT NULL,

    -- Trigger Details
    triggered_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    alert_type VARCHAR(50) NOT NULL,
    condition_met JSONB NOT NULL, -- The exact condition that was met
    -- Example: {'price': 152.34, 'threshold': 150.00, 'comparison': 'above'}

    -- Market Data Snapshot
    market_data JSONB NOT NULL,
    -- Example: {
    --   'price': 152.34,
    --   'change_pct': 2.5,
    --   'volume': 1234567,
    --   'avg_volume': 800000,
    --   'market_cap': 1000000000,
    --   'timestamp': '2025-01-15T14:30:00Z'
    -- }

    -- Notification Status
    notification_sent BOOLEAN DEFAULT false,
    notification_sent_at TIMESTAMP WITH TIME ZONE,
    notification_error TEXT,

    -- User Interaction
    is_read BOOLEAN DEFAULT false,
    read_at TIMESTAMP WITH TIME ZONE,
    is_dismissed BOOLEAN DEFAULT false,
    dismissed_at TIMESTAMP WITH TIME ZONE
);

-- Indexes
CREATE INDEX idx_alert_logs_alert_rule_id ON alert_logs(alert_rule_id);
CREATE INDEX idx_alert_logs_user_id ON alert_logs(user_id);
CREATE INDEX idx_alert_logs_symbol ON alert_logs(symbol);
CREATE INDEX idx_alert_logs_triggered_at ON alert_logs(triggered_at DESC);
CREATE INDEX idx_alert_logs_unread ON alert_logs(user_id, is_read) WHERE is_read = false;
```

### Phase 6: Notification Tables

#### `notification_preferences` Table
Stores user notification settings.

```sql
CREATE TABLE notification_preferences (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,

    -- Email Preferences
    email_enabled BOOLEAN DEFAULT true,
    email_address VARCHAR(255), -- Can differ from login email
    email_verified BOOLEAN DEFAULT false,

    -- Notification Types
    price_alerts_enabled BOOLEAN DEFAULT true,
    volume_alerts_enabled BOOLEAN DEFAULT true,
    news_alerts_enabled BOOLEAN DEFAULT true,
    earnings_alerts_enabled BOOLEAN DEFAULT true,
    sec_filing_alerts_enabled BOOLEAN DEFAULT true,

    -- Digest Settings
    daily_digest_enabled BOOLEAN DEFAULT false,
    daily_digest_time TIME DEFAULT '09:00:00', -- User's local time
    weekly_digest_enabled BOOLEAN DEFAULT false,
    weekly_digest_day INTEGER DEFAULT 1, -- 1=Monday, 7=Sunday
    weekly_digest_time TIME DEFAULT '09:00:00',

    -- Digest Content
    digest_include_portfolio_summary BOOLEAN DEFAULT true,
    digest_include_top_movers BOOLEAN DEFAULT true,
    digest_include_recent_alerts BOOLEAN DEFAULT true,
    digest_include_news_highlights BOOLEAN DEFAULT true,

    -- Quiet Hours
    quiet_hours_enabled BOOLEAN DEFAULT false,
    quiet_hours_start TIME DEFAULT '22:00:00',
    quiet_hours_end TIME DEFAULT '08:00:00',
    quiet_hours_timezone VARCHAR(50) DEFAULT 'America/Los_Angeles',

    -- Rate Limiting (Premium Feature)
    max_alerts_per_day INTEGER DEFAULT 50, -- Free: 50, Premium: unlimited
    max_emails_per_day INTEGER DEFAULT 20,  -- Free: 20, Premium: 100

    -- Metadata
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Trigger to create default preferences for new users
CREATE OR REPLACE FUNCTION create_default_notification_preferences()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO notification_preferences (user_id, email_address)
    VALUES (NEW.id, NEW.email);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_create_default_notification_preferences
AFTER INSERT ON users
FOR EACH ROW
EXECUTE FUNCTION create_default_notification_preferences();
```

#### `notification_queue` Table
Temporary queue for in-app notifications (can be replaced with Redis).

```sql
CREATE TABLE notification_queue (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    alert_log_id UUID REFERENCES alert_logs(id) ON DELETE SET NULL,

    -- Notification Content
    type VARCHAR(50) NOT NULL, -- 'alert_triggered', 'digest', 'system'
    title VARCHAR(255) NOT NULL,
    message TEXT NOT NULL,
    metadata JSONB, -- Additional data (symbol, price, etc.)

    -- Status
    is_read BOOLEAN DEFAULT false,
    read_at TIMESTAMP WITH TIME ZONE,
    is_dismissed BOOLEAN DEFAULT false,
    dismissed_at TIMESTAMP WITH TIME ZONE,

    -- Scheduling
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP WITH TIME ZONE DEFAULT (CURRENT_TIMESTAMP + INTERVAL '30 days')
);

-- Indexes
CREATE INDEX idx_notification_queue_user_id ON notification_queue(user_id);
CREATE INDEX idx_notification_queue_unread ON notification_queue(user_id, is_read) WHERE is_read = false;
CREATE INDEX idx_notification_queue_created_at ON notification_queue(created_at DESC);

-- Auto-delete expired notifications
CREATE INDEX idx_notification_queue_expires_at ON notification_queue(expires_at) WHERE is_dismissed = false;
```

#### `digest_logs` Table
Tracks sent digests to prevent duplicates.

```sql
CREATE TABLE digest_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    digest_type VARCHAR(20) NOT NULL, -- 'daily', 'weekly'
    period_start TIMESTAMP WITH TIME ZONE NOT NULL,
    period_end TIMESTAMP WITH TIME ZONE NOT NULL,
    sent_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    email_sent BOOLEAN DEFAULT false,
    email_opened BOOLEAN DEFAULT false,
    email_clicked BOOLEAN DEFAULT false,
    content_snapshot JSONB, -- Summary of what was included

    CONSTRAINT unique_digest_per_period UNIQUE (user_id, digest_type, period_start)
);

CREATE INDEX idx_digest_logs_user_id ON digest_logs(user_id);
CREATE INDEX idx_digest_logs_sent_at ON digest_logs(sent_at DESC);
```

### Phase 7: Premium Tier Tables

#### `subscription_plans` Table
Defines available subscription tiers.

```sql
CREATE TABLE subscription_plans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(50) NOT NULL UNIQUE, -- 'free', 'premium', 'enterprise'
    display_name VARCHAR(100) NOT NULL,
    description TEXT,

    -- Pricing
    price_monthly DECIMAL(10, 2) NOT NULL DEFAULT 0.00,
    price_yearly DECIMAL(10, 2) NOT NULL DEFAULT 0.00,

    -- Limits
    max_watch_lists INTEGER NOT NULL DEFAULT 3,
    max_items_per_watch_list INTEGER NOT NULL DEFAULT 10,
    max_alert_rules INTEGER NOT NULL DEFAULT 10,
    max_heatmap_configs INTEGER NOT NULL DEFAULT 3,

    -- Features (JSONB for flexibility)
    features JSONB NOT NULL DEFAULT '{}',
    -- Example: {
    --   "real_time_alerts": true,
    --   "news_alerts": true,
    --   "advanced_charts": true,
    --   "api_access": false,
    --   "priority_support": false,
    --   "custom_branding": false
    -- }

    -- Metadata
    is_active BOOLEAN DEFAULT true,
    sort_order INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Insert default plans
INSERT INTO subscription_plans (name, display_name, price_monthly, price_yearly, max_watch_lists, max_items_per_watch_list, max_alert_rules, max_heatmap_configs, features, sort_order) VALUES
('free', 'Free', 0.00, 0.00, 3, 10, 10, 3,
 '{"real_time_alerts": false, "news_alerts": false, "advanced_charts": false, "api_access": false, "priority_support": false}', 1),
('premium', 'Premium', 19.99, 199.00, 20, 100, 100, 20,
 '{"real_time_alerts": true, "news_alerts": true, "advanced_charts": true, "api_access": true, "priority_support": true, "custom_branding": false}', 2),
('enterprise', 'Enterprise', 99.99, 999.00, -1, -1, -1, -1,
 '{"real_time_alerts": true, "news_alerts": true, "advanced_charts": true, "api_access": true, "priority_support": true, "custom_branding": true}', 3);
```

#### `user_subscriptions` Table
Tracks user subscriptions and billing.

```sql
CREATE TABLE user_subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    plan_id UUID NOT NULL REFERENCES subscription_plans(id),

    -- Subscription Details
    status VARCHAR(20) NOT NULL DEFAULT 'active', -- 'active', 'cancelled', 'expired', 'past_due'
    billing_period VARCHAR(20) NOT NULL, -- 'monthly', 'yearly'

    -- Dates
    started_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    current_period_start TIMESTAMP WITH TIME ZONE NOT NULL,
    current_period_end TIMESTAMP WITH TIME ZONE NOT NULL,
    cancelled_at TIMESTAMP WITH TIME ZONE,

    -- Payment Integration (Stripe/PayPal)
    stripe_customer_id VARCHAR(255),
    stripe_subscription_id VARCHAR(255),
    payment_method_last4 VARCHAR(4),
    payment_method_brand VARCHAR(20),

    -- Metadata
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT valid_status CHECK (status IN ('active', 'cancelled', 'expired', 'past_due', 'trialing')),
    CONSTRAINT valid_billing_period CHECK (billing_period IN ('monthly', 'yearly'))
);

CREATE INDEX idx_user_subscriptions_user_id ON user_subscriptions(user_id);
CREATE INDEX idx_user_subscriptions_status ON user_subscriptions(status);
CREATE INDEX idx_user_subscriptions_period_end ON user_subscriptions(current_period_end);
```

---

## Backend Implementation

### File Structure

```
backend/
├── models/
│   ├── alert.go                    # Alert rule and log models
│   ├── notification.go             # Notification models
│   └── subscription.go             # Subscription models
├── database/
│   ├── alerts.go                   # Alert CRUD operations
│   ├── notifications.go            # Notification CRUD operations
│   └── subscriptions.go            # Subscription CRUD operations
├── services/
│   ├── alert_service.go            # Alert business logic
│   ├── alert_processor.go          # Alert evaluation engine
│   ├── notification_service.go     # Notification delivery
│   ├── email_service.go            # Email sending (extend existing)
│   ├── digest_service.go           # Digest generation
│   └── subscription_service.go     # Subscription management
├── handlers/
│   ├── alert_handlers.go           # Alert API endpoints
│   ├── notification_handlers.go    # Notification API endpoints
│   └── subscription_handlers.go    # Subscription API endpoints
├── workers/
│   ├── price_alert_worker.go       # Price alert processor
│   ├── news_alert_worker.go        # News alert processor
│   └── digest_worker.go            # Digest generator
└── cmd/
    ├── alert-processor/            # Alert worker CLI
    ├── news-processor/             # News worker CLI
    └── digest-generator/           # Digest worker CLI
```

### Models (`backend/models/alert.go`)

```go
package models

import (
    "time"
    "encoding/json"
)

// AlertRule represents a user-defined alert rule
type AlertRule struct {
    ID                 string          `json:"id" db:"id"`
    UserID             string          `json:"user_id" db:"user_id"`
    WatchListID        string          `json:"watch_list_id" db:"watch_list_id"`
    WatchListItemID    *string         `json:"watch_list_item_id,omitempty" db:"watch_list_item_id"`
    Symbol             string          `json:"symbol" db:"symbol"`
    AlertType          string          `json:"alert_type" db:"alert_type"`
    Conditions         json.RawMessage `json:"conditions" db:"conditions"`
    IsActive           bool            `json:"is_active" db:"is_active"`
    Frequency          string          `json:"frequency" db:"frequency"`
    NotifyEmail        bool            `json:"notify_email" db:"notify_email"`
    NotifyInApp        bool            `json:"notify_in_app" db:"notify_in_app"`
    Name               string          `json:"name" db:"name"`
    Description        *string         `json:"description,omitempty" db:"description"`
    LastTriggeredAt    *time.Time      `json:"last_triggered_at,omitempty" db:"last_triggered_at"`
    TriggerCount       int             `json:"trigger_count" db:"trigger_count"`
    CreatedAt          time.Time       `json:"created_at" db:"created_at"`
    UpdatedAt          time.Time       `json:"updated_at" db:"updated_at"`
}

// AlertLog represents a triggered alert instance
type AlertLog struct {
    ID                  string          `json:"id" db:"id"`
    AlertRuleID         string          `json:"alert_rule_id" db:"alert_rule_id"`
    UserID              string          `json:"user_id" db:"user_id"`
    Symbol              string          `json:"symbol" db:"symbol"`
    TriggeredAt         time.Time       `json:"triggered_at" db:"triggered_at"`
    AlertType           string          `json:"alert_type" db:"alert_type"`
    ConditionMet        json.RawMessage `json:"condition_met" db:"condition_met"`
    MarketData          json.RawMessage `json:"market_data" db:"market_data"`
    NotificationSent    bool            `json:"notification_sent" db:"notification_sent"`
    NotificationSentAt  *time.Time      `json:"notification_sent_at,omitempty" db:"notification_sent_at"`
    NotificationError   *string         `json:"notification_error,omitempty" db:"notification_error"`
    IsRead              bool            `json:"is_read" db:"is_read"`
    ReadAt              *time.Time      `json:"read_at,omitempty" db:"read_at"`
    IsDismissed         bool            `json:"is_dismissed" db:"is_dismissed"`
    DismissedAt         *time.Time      `json:"dismissed_at,omitempty" db:"dismissed_at"`
}

// Condition type definitions for type safety
type PriceAboveCondition struct {
    Threshold  float64 `json:"threshold"`
    Comparison string  `json:"comparison"` // "above", "below"
}

type PriceChangeCondition struct {
    PercentChange float64 `json:"percent_change"`
    Period        string  `json:"period"` // "1d", "1w", "1m"
    Direction     string  `json:"direction"` // "up", "down", "either"
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
    WatchListID     string          `json:"watch_list_id" binding:"required"`
    Symbol          string          `json:"symbol" binding:"required"`
    AlertType       string          `json:"alert_type" binding:"required"`
    Conditions      json.RawMessage `json:"conditions" binding:"required"`
    Name            string          `json:"name" binding:"required"`
    Description     *string         `json:"description,omitempty"`
    Frequency       string          `json:"frequency" binding:"required"`
    NotifyEmail     bool            `json:"notify_email"`
    NotifyInApp     bool            `json:"notify_in_app"`
}

// UpdateAlertRuleRequest is the API request for updating alerts
type UpdateAlertRuleRequest struct {
    Name        *string         `json:"name,omitempty"`
    Description *string         `json:"description,omitempty"`
    Conditions  json.RawMessage `json:"conditions,omitempty"`
    IsActive    *bool           `json:"is_active,omitempty"`
    Frequency   *string         `json:"frequency,omitempty"`
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
```

### Models (`backend/models/notification.go`)

```go
package models

import (
    "time"
    "encoding/json"
)

// NotificationPreferences stores user notification settings
type NotificationPreferences struct {
    ID                              string    `json:"id" db:"id"`
    UserID                          string    `json:"user_id" db:"user_id"`
    EmailEnabled                    bool      `json:"email_enabled" db:"email_enabled"`
    EmailAddress                    *string   `json:"email_address,omitempty" db:"email_address"`
    EmailVerified                   bool      `json:"email_verified" db:"email_verified"`
    PriceAlertsEnabled              bool      `json:"price_alerts_enabled" db:"price_alerts_enabled"`
    VolumeAlertsEnabled             bool      `json:"volume_alerts_enabled" db:"volume_alerts_enabled"`
    NewsAlertsEnabled               bool      `json:"news_alerts_enabled" db:"news_alerts_enabled"`
    EarningsAlertsEnabled           bool      `json:"earnings_alerts_enabled" db:"earnings_alerts_enabled"`
    SECFilingAlertsEnabled          bool      `json:"sec_filing_alerts_enabled" db:"sec_filing_alerts_enabled"`
    DailyDigestEnabled              bool      `json:"daily_digest_enabled" db:"daily_digest_enabled"`
    DailyDigestTime                 string    `json:"daily_digest_time" db:"daily_digest_time"`
    WeeklyDigestEnabled             bool      `json:"weekly_digest_enabled" db:"weekly_digest_enabled"`
    WeeklyDigestDay                 int       `json:"weekly_digest_day" db:"weekly_digest_day"`
    WeeklyDigestTime                string    `json:"weekly_digest_time" db:"weekly_digest_time"`
    DigestIncludePortfolioSummary   bool      `json:"digest_include_portfolio_summary" db:"digest_include_portfolio_summary"`
    DigestIncludeTopMovers          bool      `json:"digest_include_top_movers" db:"digest_include_top_movers"`
    DigestIncludeRecentAlerts       bool      `json:"digest_include_recent_alerts" db:"digest_include_recent_alerts"`
    DigestIncludeNewsHighlights     bool      `json:"digest_include_news_highlights" db:"digest_include_news_highlights"`
    QuietHoursEnabled               bool      `json:"quiet_hours_enabled" db:"quiet_hours_enabled"`
    QuietHoursStart                 string    `json:"quiet_hours_start" db:"quiet_hours_start"`
    QuietHoursEnd                   string    `json:"quiet_hours_end" db:"quiet_hours_end"`
    QuietHoursTimezone              string    `json:"quiet_hours_timezone" db:"quiet_hours_timezone"`
    MaxAlertsPerDay                 int       `json:"max_alerts_per_day" db:"max_alerts_per_day"`
    MaxEmailsPerDay                 int       `json:"max_emails_per_day" db:"max_emails_per_day"`
    CreatedAt                       time.Time `json:"created_at" db:"created_at"`
    UpdatedAt                       time.Time `json:"updated_at" db:"updated_at"`
}

// UpdateNotificationPreferencesRequest is the API request
type UpdateNotificationPreferencesRequest struct {
    EmailEnabled                  *bool   `json:"email_enabled,omitempty"`
    EmailAddress                  *string `json:"email_address,omitempty"`
    PriceAlertsEnabled            *bool   `json:"price_alerts_enabled,omitempty"`
    VolumeAlertsEnabled           *bool   `json:"volume_alerts_enabled,omitempty"`
    NewsAlertsEnabled             *bool   `json:"news_alerts_enabled,omitempty"`
    EarningsAlertsEnabled         *bool   `json:"earnings_alerts_enabled,omitempty"`
    SECFilingAlertsEnabled        *bool   `json:"sec_filing_alerts_enabled,omitempty"`
    DailyDigestEnabled            *bool   `json:"daily_digest_enabled,omitempty"`
    DailyDigestTime               *string `json:"daily_digest_time,omitempty"`
    WeeklyDigestEnabled           *bool   `json:"weekly_digest_enabled,omitempty"`
    WeeklyDigestDay               *int    `json:"weekly_digest_day,omitempty"`
    WeeklyDigestTime              *string `json:"weekly_digest_time,omitempty"`
    QuietHoursEnabled             *bool   `json:"quiet_hours_enabled,omitempty"`
    QuietHoursStart               *string `json:"quiet_hours_start,omitempty"`
    QuietHoursEnd                 *string `json:"quiet_hours_end,omitempty"`
    QuietHoursTimezone            *string `json:"quiet_hours_timezone,omitempty"`
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
    UserName           string                  `json:"user_name"`
    PeriodStart        time.Time               `json:"period_start"`
    PeriodEnd          time.Time               `json:"period_end"`
    PortfolioSummary   *PortfolioSummary       `json:"portfolio_summary,omitempty"`
    TopMovers          []TopMover              `json:"top_movers,omitempty"`
    RecentAlerts       []AlertLogWithRule      `json:"recent_alerts,omitempty"`
    NewsHighlights     []NewsHighlight         `json:"news_highlights,omitempty"`
}

type PortfolioSummary struct {
    TotalValue      float64 `json:"total_value"`
    DayChange       float64 `json:"day_change"`
    DayChangePct    float64 `json:"day_change_pct"`
    WeekChange      float64 `json:"week_change"`
    WeekChangePct   float64 `json:"week_change_pct"`
}

type TopMover struct {
    Symbol      string  `json:"symbol"`
    Name        string  `json:"name"`
    Price       float64 `json:"price"`
    ChangePct   float64 `json:"change_pct"`
    Direction   string  `json:"direction"` // "up" or "down"
}

type NewsHighlight struct {
    Symbol      string    `json:"symbol"`
    Title       string    `json:"title"`
    Summary     string    `json:"summary"`
    PublishedAt time.Time `json:"published_at"`
    URL         string    `json:"url"`
}
```

### API Handlers (`backend/handlers/alert_handlers.go`)

```go
package handlers

import (
    "net/http"
    "github.com/gin-gonic/gin"
    "investorcenter/backend/models"
    "investorcenter/backend/services"
)

type AlertHandler struct {
    alertService *services.AlertService
}

func NewAlertHandler(alertService *services.AlertService) *AlertHandler {
    return &AlertHandler{alertService: alertService}
}

// ListAlertRules godoc
// @Summary List all alert rules for a user
// @Tags alerts
// @Produce json
// @Param watch_list_id query string false "Filter by watch list ID"
// @Param is_active query bool false "Filter by active status"
// @Success 200 {array} models.AlertRuleWithDetails
// @Router /api/v1/alerts [get]
func (h *AlertHandler) ListAlertRules(c *gin.Context) {
    userID := c.GetString("user_id")
    watchListID := c.Query("watch_list_id")
    isActive := c.Query("is_active")

    alerts, err := h.alertService.GetUserAlerts(userID, watchListID, isActive)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch alerts"})
        return
    }

    c.JSON(http.StatusOK, alerts)
}

// CreateAlertRule godoc
// @Summary Create a new alert rule
// @Tags alerts
// @Accept json
// @Produce json
// @Param alert body models.CreateAlertRuleRequest true "Alert rule details"
// @Success 201 {object} models.AlertRule
// @Router /api/v1/alerts [post]
func (h *AlertHandler) CreateAlertRule(c *gin.Context) {
    userID := c.GetString("user_id")

    var req models.CreateAlertRuleRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    // Validate watch list ownership
    if err := h.alertService.ValidateWatchListOwnership(userID, req.WatchListID); err != nil {
        c.JSON(http.StatusForbidden, gin.H{"error": "Watch list not found"})
        return
    }

    // Check tier limits
    canCreate, err := h.alertService.CanCreateAlert(userID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check limits"})
        return
    }
    if !canCreate {
        c.JSON(http.StatusForbidden, gin.H{"error": "Alert limit reached. Upgrade to Premium for more alerts."})
        return
    }

    // Create alert
    alert, err := h.alertService.CreateAlert(userID, &req)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create alert"})
        return
    }

    c.JSON(http.StatusCreated, alert)
}

// GetAlertRule godoc
// @Summary Get alert rule by ID
// @Tags alerts
// @Produce json
// @Param id path string true "Alert ID"
// @Success 200 {object} models.AlertRuleWithDetails
// @Router /api/v1/alerts/:id [get]
func (h *AlertHandler) GetAlertRule(c *gin.Context) {
    userID := c.GetString("user_id")
    alertID := c.Param("id")

    alert, err := h.alertService.GetAlertByID(alertID, userID)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "Alert not found"})
        return
    }

    c.JSON(http.StatusOK, alert)
}

// UpdateAlertRule godoc
// @Summary Update alert rule
// @Tags alerts
// @Accept json
// @Produce json
// @Param id path string true "Alert ID"
// @Param alert body models.UpdateAlertRuleRequest true "Update details"
// @Success 200 {object} models.AlertRule
// @Router /api/v1/alerts/:id [put]
func (h *AlertHandler) UpdateAlertRule(c *gin.Context) {
    userID := c.GetString("user_id")
    alertID := c.Param("id")

    var req models.UpdateAlertRuleRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    alert, err := h.alertService.UpdateAlert(alertID, userID, &req)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update alert"})
        return
    }

    c.JSON(http.StatusOK, alert)
}

// DeleteAlertRule godoc
// @Summary Delete alert rule
// @Tags alerts
// @Param id path string true "Alert ID"
// @Success 204
// @Router /api/v1/alerts/:id [delete]
func (h *AlertHandler) DeleteAlertRule(c *gin.Context) {
    userID := c.GetString("user_id")
    alertID := c.Param("id")

    if err := h.alertService.DeleteAlert(alertID, userID); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete alert"})
        return
    }

    c.Status(http.StatusNoContent)
}

// ListAlertLogs godoc
// @Summary Get alert trigger history
// @Tags alerts
// @Produce json
// @Param alert_id query string false "Filter by alert rule ID"
// @Param symbol query string false "Filter by symbol"
// @Param limit query int false "Number of results" default(50)
// @Param offset query int false "Offset for pagination" default(0)
// @Success 200 {array} models.AlertLogWithRule
// @Router /api/v1/alerts/logs [get]
func (h *AlertHandler) ListAlertLogs(c *gin.Context) {
    userID := c.GetString("user_id")
    alertID := c.Query("alert_id")
    symbol := c.Query("symbol")
    limit := c.DefaultQuery("limit", "50")
    offset := c.DefaultQuery("offset", "0")

    logs, err := h.alertService.GetAlertLogs(userID, alertID, symbol, limit, offset)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch alert logs"})
        return
    }

    c.JSON(http.StatusOK, logs)
}

// MarkAlertLogRead godoc
// @Summary Mark alert log as read
// @Tags alerts
// @Param id path string true "Alert log ID"
// @Success 200
// @Router /api/v1/alerts/logs/:id/read [post]
func (h *AlertHandler) MarkAlertLogRead(c *gin.Context) {
    userID := c.GetString("user_id")
    logID := c.Param("id")

    if err := h.alertService.MarkLogAsRead(logID, userID); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to mark as read"})
        return
    }

    c.Status(http.StatusOK)
}

// DismissAlertLog godoc
// @Summary Dismiss alert log
// @Tags alerts
// @Param id path string true "Alert log ID"
// @Success 200
// @Router /api/v1/alerts/logs/:id/dismiss [post]
func (h *AlertHandler) DismissAlertLog(c *gin.Context) {
    userID := c.GetString("user_id")
    logID := c.Param("id")

    if err := h.alertService.DismissLog(logID, userID); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to dismiss"})
        return
    }

    c.Status(http.StatusOK)
}
```

### Alert Processing Service (`backend/services/alert_processor.go`)

```go
package services

import (
    "context"
    "encoding/json"
    "fmt"
    "time"

    "investorcenter/backend/models"
    "investorcenter/backend/database"
)

type AlertProcessor struct {
    db               *database.DB
    polygonService   *PolygonService
    notificationSvc  *NotificationService
}

func NewAlertProcessor(db *database.DB, polygonService *PolygonService, notificationSvc *NotificationService) *AlertProcessor {
    return &AlertProcessor{
        db:              db,
        polygonService:  polygonService,
        notificationSvc: notificationSvc,
    }
}

// ProcessAllAlerts is called by CronJob every minute
func (ap *AlertProcessor) ProcessAllAlerts(ctx context.Context) error {
    // Fetch all active alerts
    alerts, err := ap.db.GetActiveAlerts()
    if err != nil {
        return fmt.Errorf("failed to fetch active alerts: %w", err)
    }

    fmt.Printf("Processing %d active alerts\n", len(alerts))

    for _, alert := range alerts {
        if err := ap.processAlert(ctx, &alert); err != nil {
            fmt.Printf("Error processing alert %s: %v\n", alert.ID, err)
            continue
        }
    }

    return nil
}

// processAlert evaluates a single alert
func (ap *AlertProcessor) processAlert(ctx context.Context, alert *models.AlertRule) error {
    // Check if we should skip based on frequency
    if ap.shouldSkipAlert(alert) {
        return nil
    }

    // Evaluate based on alert type
    triggered, conditionMet, marketData, err := ap.evaluateAlert(ctx, alert)
    if err != nil {
        return fmt.Errorf("failed to evaluate alert: %w", err)
    }

    if !triggered {
        return nil // Condition not met, skip
    }

    // Create alert log
    log := &models.AlertLog{
        AlertRuleID:  alert.ID,
        UserID:       alert.UserID,
        Symbol:       alert.Symbol,
        AlertType:    alert.AlertType,
        ConditionMet: conditionMet,
        MarketData:   marketData,
    }

    if err := ap.db.CreateAlertLog(log); err != nil {
        return fmt.Errorf("failed to create alert log: %w", err)
    }

    // Send notification
    if err := ap.notificationSvc.SendAlertNotification(alert, log); err != nil {
        // Log error but don't fail
        fmt.Printf("Failed to send notification for alert %s: %v\n", alert.ID, err)

        // Update log with error
        ap.db.UpdateAlertLogNotificationError(log.ID, err.Error())
    } else {
        // Mark notification as sent
        ap.db.UpdateAlertLogNotificationSent(log.ID)
    }

    // Update alert rule
    if err := ap.db.UpdateAlertTriggered(alert.ID); err != nil {
        return fmt.Errorf("failed to update alert: %w", err)
    }

    // Disable if frequency is "once"
    if alert.Frequency == "once" {
        if err := ap.db.DisableAlert(alert.ID); err != nil {
            fmt.Printf("Failed to disable one-time alert %s: %v\n", alert.ID, err)
        }
    }

    return nil
}

// evaluateAlert checks if alert condition is met
func (ap *AlertProcessor) evaluateAlert(ctx context.Context, alert *models.AlertRule) (bool, json.RawMessage, json.RawMessage, error) {
    switch alert.AlertType {
    case "price_above", "price_below":
        return ap.evaluatePriceAlert(ctx, alert)
    case "price_change_pct":
        return ap.evaluatePriceChangeAlert(ctx, alert)
    case "volume_spike":
        return ap.evaluateVolumeSpikeAlert(ctx, alert)
    case "news":
        return ap.evaluateNewsAlert(ctx, alert)
    default:
        return false, nil, nil, fmt.Errorf("unsupported alert type: %s", alert.AlertType)
    }
}

// evaluatePriceAlert checks price threshold conditions
func (ap *AlertProcessor) evaluatePriceAlert(ctx context.Context, alert *models.AlertRule) (bool, json.RawMessage, json.RawMessage, error) {
    // Parse condition
    var condition models.PriceAboveCondition
    if err := json.Unmarshal(alert.Conditions, &condition); err != nil {
        return false, nil, nil, err
    }

    // Fetch current price
    quote, err := ap.polygonService.GetRealTimeQuote(alert.Symbol)
    if err != nil {
        return false, nil, nil, err
    }

    currentPrice := quote.Price

    // Evaluate condition
    triggered := false
    if alert.AlertType == "price_above" {
        triggered = currentPrice >= condition.Threshold
    } else {
        triggered = currentPrice <= condition.Threshold
    }

    // Build response JSONs
    conditionMet, _ := json.Marshal(map[string]interface{}{
        "price":      currentPrice,
        "threshold":  condition.Threshold,
        "comparison": condition.Comparison,
    })

    marketData, _ := json.Marshal(map[string]interface{}{
        "price":      currentPrice,
        "change_pct": quote.ChangePct,
        "volume":     quote.Volume,
        "timestamp":  time.Now(),
    })

    return triggered, conditionMet, marketData, nil
}

// evaluatePriceChangeAlert checks percentage change conditions
func (ap *AlertProcessor) evaluatePriceChangeAlert(ctx context.Context, alert *models.AlertRule) (bool, json.RawMessage, json.RawMessage, error) {
    var condition models.PriceChangeCondition
    if err := json.Unmarshal(alert.Conditions, &condition); err != nil {
        return false, nil, nil, err
    }

    // Fetch current and historical price
    quote, err := ap.polygonService.GetRealTimeQuote(alert.Symbol)
    if err != nil {
        return false, nil, nil, err
    }

    // Get historical price based on period
    historicalPrice, err := ap.getHistoricalPrice(alert.Symbol, condition.Period)
    if err != nil {
        return false, nil, nil, err
    }

    // Calculate change
    changePct := ((quote.Price - historicalPrice) / historicalPrice) * 100

    // Evaluate condition
    triggered := false
    switch condition.Direction {
    case "up":
        triggered = changePct >= condition.PercentChange
    case "down":
        triggered = changePct <= -condition.PercentChange
    case "either":
        triggered = abs(changePct) >= condition.PercentChange
    }

    conditionMet, _ := json.Marshal(map[string]interface{}{
        "current_price":    quote.Price,
        "historical_price": historicalPrice,
        "change_pct":       changePct,
        "threshold_pct":    condition.PercentChange,
        "period":           condition.Period,
    })

    marketData, _ := json.Marshal(map[string]interface{}{
        "price":      quote.Price,
        "change_pct": changePct,
        "volume":     quote.Volume,
        "timestamp":  time.Now(),
    })

    return triggered, conditionMet, marketData, nil
}

// evaluateVolumeSpikeAlert checks volume spike conditions
func (ap *AlertProcessor) evaluateVolumeSpikeAlert(ctx context.Context, alert *models.AlertRule) (bool, json.RawMessage, json.RawMessage, error) {
    var condition models.VolumeSpikeCondition
    if err := json.Unmarshal(alert.Conditions, &condition); err != nil {
        return false, nil, nil, err
    }

    // Fetch current volume
    quote, err := ap.polygonService.GetRealTimeQuote(alert.Symbol)
    if err != nil {
        return false, nil, nil, err
    }

    // Get baseline volume
    baselineVolume, err := ap.getBaselineVolume(alert.Symbol, condition.Baseline)
    if err != nil {
        return false, nil, nil, err
    }

    // Calculate multiplier
    volumeMultiplier := float64(quote.Volume) / baselineVolume

    triggered := volumeMultiplier >= condition.VolumeMultiplier

    conditionMet, _ := json.Marshal(map[string]interface{}{
        "current_volume":    quote.Volume,
        "baseline_volume":   baselineVolume,
        "volume_multiplier": volumeMultiplier,
        "threshold":         condition.VolumeMultiplier,
    })

    marketData, _ := json.Marshal(map[string]interface{}{
        "price":      quote.Price,
        "volume":     quote.Volume,
        "avg_volume": baselineVolume,
        "timestamp":  time.Now(),
    })

    return triggered, conditionMet, marketData, nil
}

// Helper functions
func (ap *AlertProcessor) shouldSkipAlert(alert *models.AlertRule) bool {
    if alert.Frequency == "daily" && alert.LastTriggeredAt != nil {
        // Check if already triggered today
        lastTriggered := *alert.LastTriggeredAt
        if time.Since(lastTriggered) < 24*time.Hour {
            return true
        }
    }
    return false
}

func (ap *AlertProcessor) getHistoricalPrice(symbol string, period string) (float64, error) {
    // Implementation depends on period
    // Query database or Polygon API for historical price
    // For now, simplified
    return 0, nil
}

func (ap *AlertProcessor) getBaselineVolume(symbol string, baseline string) (float64, error) {
    // Query database for average volume
    // baseline can be "avg_30d", "avg_90d"
    return 0, nil
}

func abs(x float64) float64 {
    if x < 0 {
        return -x
    }
    return x
}
```

---

## Frontend Implementation

### File Structure

```
app/
├── alerts/
│   ├── page.tsx                  # Alert rules management page
│   ├── logs/
│   │   └── page.tsx              # Alert history page
│   └── create/
│       └── page.tsx              # Create alert page
├── settings/
│   ├── notifications/
│   │   └── page.tsx              # Notification preferences page
│   └── subscription/
│       └── page.tsx              # Subscription management page

components/
├── alerts/
│   ├── AlertRuleList.tsx         # List of alert rules
│   ├── AlertRuleCard.tsx         # Single alert rule card
│   ├── CreateAlertModal.tsx      # Create/edit alert modal
│   ├── AlertLogList.tsx          # Alert history list
│   └── AlertConditionBuilder.tsx # Condition builder UI
├── notifications/
│   ├── NotificationBell.tsx      # Header notification bell
│   ├── NotificationDropdown.tsx  # Notification dropdown
│   ├── NotificationPreferences.tsx # Preferences form
│   └── DigestPreferences.tsx     # Digest settings form
└── subscription/
    ├── PlanCard.tsx              # Subscription plan card
    ├── UpgradeModal.tsx          # Upgrade prompt modal
    └── BillingInfo.tsx           # Billing information

lib/
└── api/
    ├── alerts.ts                 # Alert API client
    ├── notifications.ts          # Notification API client
    └── subscriptions.ts          # Subscription API client
```

### Alert Rules Page (`app/alerts/page.tsx`)

```tsx
'use client';

import { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { AlertRuleList } from '@/components/alerts/AlertRuleList';
import { CreateAlertModal } from '@/components/alerts/CreateAlertModal';
import { Button } from '@/components/ui/Button';
import { alertsApi } from '@/lib/api/alerts';
import { useAuth } from '@/lib/auth/AuthContext';
import type { AlertRuleWithDetails } from '@/types/alerts';

export default function AlertsPage() {
  const router = useRouter();
  const { user } = useAuth();
  const [alerts, setAlerts] = useState<AlertRuleWithDetails[]>([]);
  const [loading, setLoading] = useState(true);
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [selectedWatchList, setSelectedWatchList] = useState<string | null>(null);

  useEffect(() => {
    if (!user) {
      router.push('/auth/login');
      return;
    }
    fetchAlerts();
  }, [user]);

  const fetchAlerts = async () => {
    try {
      setLoading(true);
      const data = await alertsApi.listAlerts();
      setAlerts(data);
    } catch (error) {
      console.error('Failed to fetch alerts:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleCreateAlert = async (alertData: any) => {
    try {
      await alertsApi.createAlert(alertData);
      setShowCreateModal(false);
      fetchAlerts();
    } catch (error) {
      console.error('Failed to create alert:', error);
      throw error;
    }
  };

  const handleDeleteAlert = async (alertId: string) => {
    try {
      await alertsApi.deleteAlert(alertId);
      fetchAlerts();
    } catch (error) {
      console.error('Failed to delete alert:', error);
    }
  };

  const handleToggleAlert = async (alertId: string, isActive: boolean) => {
    try {
      await alertsApi.updateAlert(alertId, { is_active: !isActive });
      fetchAlerts();
    } catch (error) {
      console.error('Failed to toggle alert:', error);
    }
  };

  if (loading) {
    return <div className="flex justify-center items-center h-screen">Loading...</div>;
  }

  return (
    <div className="container mx-auto px-4 py-8">
      <div className="flex justify-between items-center mb-6">
        <div>
          <h1 className="text-3xl font-bold">Alert Rules</h1>
          <p className="text-gray-600 mt-2">
            Manage your price, volume, and news alerts
          </p>
        </div>
        <Button onClick={() => setShowCreateModal(true)}>
          Create Alert
        </Button>
      </div>

      {alerts.length === 0 ? (
        <div className="text-center py-12 bg-gray-50 rounded-lg">
          <h3 className="text-xl font-semibold text-gray-700 mb-2">
            No alerts yet
          </h3>
          <p className="text-gray-600 mb-4">
            Create your first alert to get notified about price movements
          </p>
          <Button onClick={() => setShowCreateModal(true)}>
            Create Your First Alert
          </Button>
        </div>
      ) : (
        <AlertRuleList
          alerts={alerts}
          onDelete={handleDeleteAlert}
          onToggle={handleToggleAlert}
          onEdit={(alert) => {
            // TODO: Implement edit functionality
          }}
        />
      )}

      {showCreateModal && (
        <CreateAlertModal
          onClose={() => setShowCreateModal(false)}
          onSubmit={handleCreateAlert}
        />
      )}
    </div>
  );
}
```

### Notification Bell Component (`components/notifications/NotificationBell.tsx`)

```tsx
'use client';

import { useEffect, useState, useRef } from 'react';
import { BellIcon } from '@heroicons/react/24/outline';
import { NotificationDropdown } from './NotificationDropdown';
import { notificationsApi } from '@/lib/api/notifications';
import type { InAppNotification } from '@/types/notifications';

export function NotificationBell() {
  const [notifications, setNotifications] = useState<InAppNotification[]>([]);
  const [unreadCount, setUnreadCount] = useState(0);
  const [isOpen, setIsOpen] = useState(false);
  const dropdownRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    fetchNotifications();

    // Poll for new notifications every 30 seconds
    const interval = setInterval(fetchNotifications, 30000);
    return () => clearInterval(interval);
  }, []);

  useEffect(() => {
    // Close dropdown when clicking outside
    function handleClickOutside(event: MouseEvent) {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target as Node)) {
        setIsOpen(false);
      }
    }

    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  const fetchNotifications = async () => {
    try {
      const data = await notificationsApi.getNotifications({ limit: 20 });
      setNotifications(data);
      setUnreadCount(data.filter(n => !n.is_read).length);
    } catch (error) {
      console.error('Failed to fetch notifications:', error);
    }
  };

  const handleMarkAsRead = async (notificationId: string) => {
    try {
      await notificationsApi.markAsRead(notificationId);
      fetchNotifications();
    } catch (error) {
      console.error('Failed to mark as read:', error);
    }
  };

  const handleMarkAllAsRead = async () => {
    try {
      await notificationsApi.markAllAsRead();
      fetchNotifications();
    } catch (error) {
      console.error('Failed to mark all as read:', error);
    }
  };

  const handleDismiss = async (notificationId: string) => {
    try {
      await notificationsApi.dismiss(notificationId);
      fetchNotifications();
    } catch (error) {
      console.error('Failed to dismiss:', error);
    }
  };

  return (
    <div className="relative" ref={dropdownRef}>
      <button
        onClick={() => setIsOpen(!isOpen)}
        className="relative p-2 text-gray-600 hover:text-gray-900 focus:outline-none"
      >
        <BellIcon className="h-6 w-6" />
        {unreadCount > 0 && (
          <span className="absolute top-0 right-0 inline-flex items-center justify-center px-2 py-1 text-xs font-bold leading-none text-white transform translate-x-1/2 -translate-y-1/2 bg-red-600 rounded-full">
            {unreadCount > 99 ? '99+' : unreadCount}
          </span>
        )}
      </button>

      {isOpen && (
        <NotificationDropdown
          notifications={notifications}
          onMarkAsRead={handleMarkAsRead}
          onMarkAllAsRead={handleMarkAllAsRead}
          onDismiss={handleDismiss}
          onClose={() => setIsOpen(false)}
        />
      )}
    </div>
  );
}
```

### API Client (`lib/api/alerts.ts`)

```typescript
import { apiClient } from './client';
import type {
  AlertRule,
  AlertRuleWithDetails,
  CreateAlertRuleRequest,
  UpdateAlertRuleRequest,
  AlertLog,
  AlertLogWithRule,
} from '@/types/alerts';

export const alertsApi = {
  // Alert Rules
  listAlerts: async (params?: {
    watch_list_id?: string;
    is_active?: boolean;
  }): Promise<AlertRuleWithDetails[]> => {
    const queryParams = new URLSearchParams();
    if (params?.watch_list_id) queryParams.set('watch_list_id', params.watch_list_id);
    if (params?.is_active !== undefined) queryParams.set('is_active', String(params.is_active));

    const response = await apiClient.get(`/alerts?${queryParams}`);
    return response.data;
  },

  createAlert: async (data: CreateAlertRuleRequest): Promise<AlertRule> => {
    const response = await apiClient.post('/alerts', data);
    return response.data;
  },

  getAlert: async (alertId: string): Promise<AlertRuleWithDetails> => {
    const response = await apiClient.get(`/alerts/${alertId}`);
    return response.data;
  },

  updateAlert: async (
    alertId: string,
    data: UpdateAlertRuleRequest
  ): Promise<AlertRule> => {
    const response = await apiClient.put(`/alerts/${alertId}`, data);
    return response.data;
  },

  deleteAlert: async (alertId: string): Promise<void> => {
    await apiClient.delete(`/alerts/${alertId}`);
  },

  // Alert Logs
  getAlertLogs: async (params?: {
    alert_id?: string;
    symbol?: string;
    limit?: number;
    offset?: number;
  }): Promise<AlertLogWithRule[]> => {
    const queryParams = new URLSearchParams();
    if (params?.alert_id) queryParams.set('alert_id', params.alert_id);
    if (params?.symbol) queryParams.set('symbol', params.symbol);
    if (params?.limit) queryParams.set('limit', String(params.limit));
    if (params?.offset) queryParams.set('offset', String(params.offset));

    const response = await apiClient.get(`/alerts/logs?${queryParams}`);
    return response.data;
  },

  markLogAsRead: async (logId: string): Promise<void> => {
    await apiClient.post(`/alerts/logs/${logId}/read`);
  },

  dismissLog: async (logId: string): Promise<void> => {
    await apiClient.post(`/alerts/logs/${logId}/dismiss`);
  },
};
```

---

## Alert Processing Engine

### Kubernetes CronJob Configuration

#### Price Alert Processor (`k8s/cronjobs/price-alert-processor.yaml`)

```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: price-alert-processor
  namespace: investorcenter
spec:
  schedule: "* * * * *"  # Every minute
  concurrencyPolicy: Forbid  # Don't run concurrent jobs
  successfulJobsHistoryLimit: 3
  failedJobsHistoryLimit: 3
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: price-alert-processor
            image: 360358043271.dkr.ecr.us-east-1.amazonaws.com/investorcenter-backend:latest
            command: ["/app/alert-processor"]
            args: ["--type=price"]
            env:
            - name: DB_HOST
              valueFrom:
                secretKeyRef:
                  name: postgres-secret
                  key: host
            - name: DB_PORT
              value: "5432"
            - name: DB_USER
              valueFrom:
                secretKeyRef:
                  name: postgres-secret
                  key: username
            - name: DB_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: postgres-secret
                  key: password
            - name: DB_NAME
              value: investorcenter_db
            - name: POLYGON_API_KEY
              valueFrom:
                secretKeyRef:
                  name: api-keys
                  key: polygon-api-key
            - name: SENDGRID_API_KEY
              valueFrom:
                secretKeyRef:
                  name: api-keys
                  key: sendgrid-api-key
            - name: REDIS_HOST
              value: redis-service
            - name: REDIS_PORT
              value: "6379"
            resources:
              requests:
                memory: "256Mi"
                cpu: "200m"
              limits:
                memory: "512Mi"
                cpu: "500m"
          restartPolicy: OnFailure
```

#### News Alert Processor (`k8s/cronjobs/news-alert-processor.yaml`)

```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: news-alert-processor
  namespace: investorcenter
spec:
  schedule: "*/5 * * * *"  # Every 5 minutes
  concurrencyPolicy: Forbid
  successfulJobsHistoryLimit: 3
  failedJobsHistoryLimit: 3
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: news-alert-processor
            image: 360358043271.dkr.ecr.us-east-1.amazonaws.com/investorcenter-backend:latest
            command: ["/app/alert-processor"]
            args: ["--type=news"]
            env:
            - name: DB_HOST
              valueFrom:
                secretKeyRef:
                  name: postgres-secret
                  key: host
            - name: FINNHUB_API_KEY
              valueFrom:
                secretKeyRef:
                  name: api-keys
                  key: finnhub-api-key
            - name: SENDGRID_API_KEY
              valueFrom:
                secretKeyRef:
                  name: api-keys
                  key: sendgrid-api-key
            resources:
              requests:
                memory: "256Mi"
                cpu: "100m"
              limits:
                memory: "512Mi"
                cpu: "300m"
          restartPolicy: OnFailure
```

#### Daily Digest Generator (`k8s/cronjobs/daily-digest-generator.yaml`)

```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: daily-digest-generator
  namespace: investorcenter
spec:
  schedule: "0 9 * * *"  # Every day at 9 AM UTC
  concurrencyPolicy: Forbid
  successfulJobsHistoryLimit: 7
  failedJobsHistoryLimit: 3
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: digest-generator
            image: 360358043271.dkr.ecr.us-east-1.amazonaws.com/investorcenter-backend:latest
            command: ["/app/digest-generator"]
            args: ["--type=daily"]
            env:
            - name: DB_HOST
              valueFrom:
                secretKeyRef:
                  name: postgres-secret
                  key: host
            - name: SENDGRID_API_KEY
              valueFrom:
                secretKeyRef:
                  name: api-keys
                  key: sendgrid-api-key
            resources:
              requests:
                memory: "512Mi"
                cpu: "200m"
              limits:
                memory: "1Gi"
                cpu: "500m"
          restartPolicy: OnFailure
```

### Worker CLI (`backend/cmd/alert-processor/main.go`)

```go
package main

import (
    "context"
    "flag"
    "fmt"
    "log"
    "os"
    "time"

    "investorcenter/backend/database"
    "investorcenter/backend/services"
)

func main() {
    alertType := flag.String("type", "price", "Alert type to process (price, volume, news)")
    flag.Parse()

    log.Printf("Starting alert processor for type: %s\n", *alertType)

    // Initialize database
    db, err := database.NewDB()
    if err != nil {
        log.Fatalf("Failed to connect to database: %v", err)
    }
    defer db.Close()

    // Initialize services
    polygonService := services.NewPolygonService(os.Getenv("POLYGON_API_KEY"))
    notificationService := services.NewNotificationService(db)
    alertProcessor := services.NewAlertProcessor(db, polygonService, notificationService)

    // Process alerts
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
    defer cancel()

    startTime := time.Now()
    if err := alertProcessor.ProcessAllAlerts(ctx); err != nil {
        log.Fatalf("Failed to process alerts: %v", err)
    }

    duration := time.Since(startTime)
    log.Printf("Alert processing completed in %v\n", duration)
}
```

---

## Notification System

### Email Templates

Create HTML email templates for different notification types.

#### Alert Triggered Email (`backend/templates/email/alert_triggered.html`)

```html
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Alert Triggered: {{.AlertName}}</title>
</head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333; max-width: 600px; margin: 0 auto; padding: 20px;">
    <div style="background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); padding: 30px; border-radius: 10px 10px 0 0; text-align: center;">
        <h1 style="color: white; margin: 0;">InvestorCenter.ai</h1>
        <p style="color: white; margin: 10px 0 0 0;">Price Alert Triggered</p>
    </div>

    <div style="background: #f9f9f9; padding: 30px; border-radius: 0 0 10px 10px;">
        <h2 style="color: #667eea; margin-top: 0;">{{.AlertName}}</h2>

        <div style="background: white; padding: 20px; border-radius: 8px; margin: 20px 0; border-left: 4px solid #667eea;">
            <h3 style="margin-top: 0;">{{.Symbol}} - {{.CompanyName}}</h3>
            <p style="font-size: 24px; font-weight: bold; color: {{.PriceColor}}; margin: 10px 0;">
                ${{.CurrentPrice}}
            </p>
            <p style="color: {{.ChangeColor}}; font-size: 18px; margin: 5px 0;">
                {{.ChangeAmount}} ({{.ChangePct}}%)
            </p>
        </div>

        <div style="background: white; padding: 20px; border-radius: 8px; margin: 20px 0;">
            <h4 style="margin-top: 0;">Alert Condition Met:</h4>
            <p>{{.ConditionDescription}}</p>

            <ul style="list-style: none; padding: 0;">
                <li><strong>Threshold:</strong> ${{.Threshold}}</li>
                <li><strong>Current Price:</strong> ${{.CurrentPrice}}</li>
                <li><strong>Volume:</strong> {{.Volume}}</li>
                <li><strong>Time:</strong> {{.TriggeredTime}}</li>
            </ul>
        </div>

        <div style="text-align: center; margin: 30px 0;">
            <a href="{{.TickerURL}}" style="background: #667eea; color: white; padding: 12px 30px; text-decoration: none; border-radius: 5px; display: inline-block;">
                View {{.Symbol}} Details
            </a>
        </div>

        <div style="text-align: center; margin: 20px 0;">
            <a href="{{.ManageAlertsURL}}" style="color: #667eea; text-decoration: none; font-size: 14px;">
                Manage Alert Settings
            </a>
        </div>
    </div>

    <div style="text-align: center; padding: 20px; color: #999; font-size: 12px;">
        <p>This alert was sent because you subscribed to price alerts on InvestorCenter.ai</p>
        <p>
            <a href="{{.UnsubscribeURL}}" style="color: #999;">Unsubscribe</a> |
            <a href="{{.PreferencesURL}}" style="color: #999;">Notification Preferences</a>
        </p>
    </div>
</body>
</html>
```

#### Daily Digest Email (`backend/templates/email/daily_digest.html`)

```html
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Your Daily Market Digest</title>
</head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333; max-width: 800px; margin: 0 auto; padding: 20px;">
    <div style="background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); padding: 30px; border-radius: 10px 10px 0 0; text-align: center;">
        <h1 style="color: white; margin: 0;">InvestorCenter.ai</h1>
        <p style="color: white; margin: 10px 0 0 0;">Daily Market Digest - {{.Date}}</p>
    </div>

    <div style="background: #f9f9f9; padding: 30px;">
        <p>Hi {{.UserName}},</p>
        <p>Here's your daily market summary for {{.Date}}.</p>

        <!-- Portfolio Summary -->
        {{if .IncludePortfolioSummary}}
        <div style="background: white; padding: 20px; border-radius: 8px; margin: 20px 0;">
            <h2 style="color: #667eea; margin-top: 0;">Your Watch Lists</h2>
            <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 20px;">
                <div style="text-align: center; padding: 15px; background: #f0f4ff; border-radius: 8px;">
                    <p style="margin: 0; color: #666; font-size: 14px;">Day Change</p>
                    <p style="margin: 5px 0; font-size: 24px; font-weight: bold; color: {{.DayChangeColor}};">
                        {{.DayChangePct}}%
                    </p>
                </div>
                <div style="text-align: center; padding: 15px; background: #f0f4ff; border-radius: 8px;">
                    <p style="margin: 0; color: #666; font-size: 14px;">Week Change</p>
                    <p style="margin: 5px 0; font-size: 24px; font-weight: bold; color: {{.WeekChangeColor}};">
                        {{.WeekChangePct}}%
                    </p>
                </div>
            </div>
        </div>
        {{end}}

        <!-- Top Movers -->
        {{if .IncludeTopMovers}}
        <div style="background: white; padding: 20px; border-radius: 8px; margin: 20px 0;">
            <h2 style="color: #667eea; margin-top: 0;">Top Movers</h2>

            <h3 style="color: #22c55e; font-size: 16px;">Biggest Gainers</h3>
            <table style="width: 100%; border-collapse: collapse;">
                {{range .TopGainers}}
                <tr style="border-bottom: 1px solid #eee;">
                    <td style="padding: 10px;"><strong>{{.Symbol}}</strong></td>
                    <td style="padding: 10px;">{{.Name}}</td>
                    <td style="padding: 10px; text-align: right;">${{.Price}}</td>
                    <td style="padding: 10px; text-align: right; color: #22c55e; font-weight: bold;">
                        +{{.ChangePct}}%
                    </td>
                </tr>
                {{end}}
            </table>

            <h3 style="color: #ef4444; font-size: 16px; margin-top: 20px;">Biggest Losers</h3>
            <table style="width: 100%; border-collapse: collapse;">
                {{range .TopLosers}}
                <tr style="border-bottom: 1px solid #eee;">
                    <td style="padding: 10px;"><strong>{{.Symbol}}</strong></td>
                    <td style="padding: 10px;">{{.Name}}</td>
                    <td style="padding: 10px; text-align: right;">${{.Price}}</td>
                    <td style="padding: 10px; text-align: right; color: #ef4444; font-weight: bold;">
                        {{.ChangePct}}%
                    </td>
                </tr>
                {{end}}
            </table>
        </div>
        {{end}}

        <!-- Recent Alerts -->
        {{if .IncludeRecentAlerts}}
        <div style="background: white; padding: 20px; border-radius: 8px; margin: 20px 0;">
            <h2 style="color: #667eea; margin-top: 0;">Triggered Alerts (Last 24 Hours)</h2>
            {{range .RecentAlerts}}
            <div style="padding: 15px; background: #f9f9f9; border-radius: 8px; margin: 10px 0;">
                <p style="margin: 0; font-weight: bold;">{{.RuleName}}</p>
                <p style="margin: 5px 0; color: #666;">{{.Symbol}} - {{.ConditionDescription}}</p>
                <p style="margin: 5px 0; font-size: 12px; color: #999;">{{.TriggeredAt}}</p>
            </div>
            {{end}}
        </div>
        {{end}}

        <!-- News Highlights -->
        {{if .IncludeNewsHighlights}}
        <div style="background: white; padding: 20px; border-radius: 8px; margin: 20px 0;">
            <h2 style="color: #667eea; margin-top: 0;">News Highlights</h2>
            {{range .NewsHighlights}}
            <div style="padding: 15px; border-bottom: 1px solid #eee;">
                <p style="margin: 0; font-weight: bold; color: #667eea;">{{.Symbol}}</p>
                <h3 style="margin: 5px 0; font-size: 16px;">
                    <a href="{{.URL}}" style="color: #333; text-decoration: none;">{{.Title}}</a>
                </h3>
                <p style="margin: 5px 0; color: #666; font-size: 14px;">{{.Summary}}</p>
                <p style="margin: 5px 0; font-size: 12px; color: #999;">{{.PublishedAt}}</p>
            </div>
            {{end}}
        </div>
        {{end}}

        <div style="text-align: center; margin: 30px 0;">
            <a href="{{.DashboardURL}}" style="background: #667eea; color: white; padding: 12px 30px; text-decoration: none; border-radius: 5px; display: inline-block;">
                View Full Dashboard
            </a>
        </div>
    </div>

    <div style="text-align: center; padding: 20px; color: #999; font-size: 12px;">
        <p>You're receiving this digest because you enabled daily digests on InvestorCenter.ai</p>
        <p>
            <a href="{{.UnsubscribeURL}}" style="color: #999;">Unsubscribe</a> |
            <a href="{{.PreferencesURL}}" style="color: #999;">Notification Preferences</a>
        </p>
    </div>
</body>
</html>
```

---

## External Integrations

### Polygon.io Integration

Extend existing `backend/services/polygon.go`:

```go
// GetHistoricalPrice fetches price from a specific date
func (p *PolygonService) GetHistoricalPrice(symbol string, date time.Time) (float64, error) {
    dateStr := date.Format("2006-01-02")
    url := fmt.Sprintf("%s/v1/open-close/%s/%s", p.baseURL, symbol, dateStr)

    // Add API key
    req, _ := http.NewRequest("GET", url, nil)
    q := req.URL.Query()
    q.Add("apiKey", p.apiKey)
    req.URL.RawQuery = q.Encode()

    resp, err := p.client.Do(req)
    if err != nil {
        return 0, err
    }
    defer resp.Body.Close()

    var result struct {
        Close float64 `json:"close"`
    }

    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return 0, err
    }

    return result.Close, nil
}
```

### Finnhub News API (`backend/services/finnhub_service.go`)

```go
package services

import (
    "encoding/json"
    "fmt"
    "net/http"
    "time"
)

type FinnhubService struct {
    apiKey  string
    baseURL string
    client  *http.Client
}

func NewFinnhubService(apiKey string) *FinnhubService {
    return &FinnhubService{
        apiKey:  apiKey,
        baseURL: "https://finnhub.io/api/v1",
        client:  &http.Client{Timeout: 10 * time.Second},
    }
}

type NewsArticle struct {
    Category    string    `json:"category"`
    Datetime    int64     `json:"datetime"`
    Headline    string    `json:"headline"`
    ID          int64     `json:"id"`
    Image       string    `json:"image"`
    Related     string    `json:"related"`
    Source      string    `json:"source"`
    Summary     string    `json:"summary"`
    URL         string    `json:"url"`
}

// GetCompanyNews fetches recent news for a symbol
func (f *FinnhubService) GetCompanyNews(symbol string, from, to time.Time) ([]NewsArticle, error) {
    url := fmt.Sprintf("%s/company-news?symbol=%s&from=%s&to=%s&token=%s",
        f.baseURL,
        symbol,
        from.Format("2006-01-02"),
        to.Format("2006-01-02"),
        f.apiKey,
    )

    resp, err := f.client.Get(url)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    var articles []NewsArticle
    if err := json.NewDecoder(resp.Body).Decode(&articles); err != nil {
        return nil, err
    }

    return articles, nil
}

// CheckNewsKeywords checks if any recent news contains keywords
func (f *FinnhubService) CheckNewsKeywords(symbol string, keywords []string) (bool, *NewsArticle, error) {
    to := time.Now()
    from := to.Add(-24 * time.Hour)

    articles, err := f.GetCompanyNews(symbol, from, to)
    if err != nil {
        return false, nil, err
    }

    for _, article := range articles {
        for _, keyword := range keywords {
            if contains(article.Headline, keyword) || contains(article.Summary, keyword) {
                return true, &article, nil
            }
        }
    }

    return false, nil, nil
}

func contains(text, substr string) bool {
    return len(text) > 0 && len(substr) > 0 &&
           (len(text) >= len(substr)) &&
           (text == substr || len(text) > len(substr) &&
            (text[:len(substr)] == substr ||
             text[len(text)-len(substr):] == substr ||
             findSubstring(text, substr)))
}

func findSubstring(text, substr string) bool {
    // Simple substring search
    for i := 0; i <= len(text)-len(substr); i++ {
        if text[i:i+len(substr)] == substr {
            return true
        }
    }
    return false
}
```

### SendGrid Email Service

Extend existing `backend/services/email_service.go`:

```go
package services

import (
    "bytes"
    "fmt"
    "html/template"
    "path/filepath"

    "github.com/sendgrid/sendgrid-go"
    "github.com/sendgrid/sendgrid-go/helpers/mail"
)

type EmailService struct {
    apiKey    string
    fromEmail string
    fromName  string
    templates map[string]*template.Template
}

func NewEmailService(apiKey, fromEmail, fromName string) (*EmailService, error) {
    es := &EmailService{
        apiKey:    apiKey,
        fromEmail: fromEmail,
        fromName:  fromName,
        templates: make(map[string]*template.Template),
    }

    // Load email templates
    if err := es.loadTemplates(); err != nil {
        return nil, err
    }

    return es, nil
}

func (es *EmailService) loadTemplates() error {
    templateDir := "backend/templates/email"

    templates := []string{
        "alert_triggered.html",
        "daily_digest.html",
        "weekly_digest.html",
    }

    for _, tmplName := range templates {
        tmplPath := filepath.Join(templateDir, tmplName)
        tmpl, err := template.ParseFiles(tmplPath)
        if err != nil {
            return fmt.Errorf("failed to load template %s: %w", tmplName, err)
        }
        es.templates[tmplName] = tmpl
    }

    return nil
}

type AlertEmailData struct {
    AlertName            string
    Symbol               string
    CompanyName          string
    CurrentPrice         string
    ChangeAmount         string
    ChangePct            string
    PriceColor           string
    ChangeColor          string
    ConditionDescription string
    Threshold            string
    Volume               string
    TriggeredTime        string
    TickerURL            string
    ManageAlertsURL      string
    UnsubscribeURL       string
    PreferencesURL       string
}

func (es *EmailService) SendAlertEmail(toEmail, toName string, data *AlertEmailData) error {
    // Render template
    var body bytes.Buffer
    if err := es.templates["alert_triggered.html"].Execute(&body, data); err != nil {
        return fmt.Errorf("failed to render template: %w", err)
    }

    // Create email
    from := mail.NewEmail(es.fromName, es.fromEmail)
    to := mail.NewEmail(toName, toEmail)
    subject := fmt.Sprintf("Alert Triggered: %s - %s", data.Symbol, data.AlertName)
    message := mail.NewSingleEmail(from, subject, to, "", body.String())

    // Send via SendGrid
    client := sendgrid.NewSendClient(es.apiKey)
    response, err := client.Send(message)
    if err != nil {
        return fmt.Errorf("failed to send email: %w", err)
    }

    if response.StatusCode >= 400 {
        return fmt.Errorf("sendgrid returned error: %d", response.StatusCode)
    }

    return nil
}

func (es *EmailService) SendDigestEmail(toEmail, toName string, digestData interface{}) error {
    var body bytes.Buffer
    if err := es.templates["daily_digest.html"].Execute(&body, digestData); err != nil {
        return fmt.Errorf("failed to render template: %w", err)
    }

    from := mail.NewEmail(es.fromName, es.fromEmail)
    to := mail.NewEmail(toName, toEmail)
    subject := "Your Daily Market Digest"
    message := mail.NewSingleEmail(from, subject, to, "", body.String())

    client := sendgrid.NewSendClient(es.apiKey)
    response, err := client.Send(message)
    if err != nil {
        return fmt.Errorf("failed to send email: %w", err)
    }

    if response.StatusCode >= 400 {
        return fmt.Errorf("sendgrid returned error: %d", response.StatusCode)
    }

    return nil
}
```

---

## Premium Tier Implementation

### Subscription Middleware (`backend/auth/subscription_middleware.go`)

```go
package auth

import (
    "net/http"
    "github.com/gin-gonic/gin"
    "investorcenter/backend/services"
)

func RequirePremium(subscriptionService *services.SubscriptionService) gin.HandlerFunc {
    return func(c *gin.Context) {
        userID := c.GetString("user_id")

        isPremium, err := subscriptionService.IsPremiumUser(userID)
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check subscription"})
            c.Abort()
            return
        }

        if !isPremium {
            c.JSON(http.StatusForbidden, gin.H{
                "error": "This feature requires a Premium subscription",
                "upgrade_url": "/settings/subscription",
            })
            c.Abort()
            return
        }

        c.Next()
    }
}
```

### Tier Enforcement in Handlers

```go
// In alert_handlers.go
func (h *AlertHandler) CreateAlertRule(c *gin.Context) {
    userID := c.GetString("user_id")

    // Check tier-based limits
    canCreate, limitInfo, err := h.alertService.CanCreateAlert(userID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check limits"})
        return
    }

    if !canCreate {
        c.JSON(http.StatusForbidden, gin.H{
            "error": fmt.Sprintf("Alert limit reached. You have %d/%d alerts.",
                     limitInfo.CurrentCount, limitInfo.MaxAllowed),
            "tier": limitInfo.CurrentTier,
            "upgrade_required": true,
            "upgrade_url": "/settings/subscription",
        })
        return
    }

    // Continue with creation...
}
```

---

## Deployment & Operations

### Deployment Checklist

1. **Database Migrations**:
   ```bash
   kubectl apply -f k8s/jobs/alert-tables-migration.yaml
   kubectl logs -f job/alert-tables-migration
   ```

2. **Deploy CronJobs**:
   ```bash
   kubectl apply -f k8s/cronjobs/price-alert-processor.yaml
   kubectl apply -f k8s/cronjobs/news-alert-processor.yaml
   kubectl apply -f k8s/cronjobs/daily-digest-generator.yaml
   kubectl apply -f k8s/cronjobs/weekly-digest-generator.yaml
   ```

3. **Configure Secrets**:
   ```bash
   kubectl create secret generic api-keys \
     --from-literal=polygon-api-key=$POLYGON_API_KEY \
     --from-literal=sendgrid-api-key=$SENDGRID_API_KEY \
     --from-literal=finnhub-api-key=$FINNHUB_API_KEY \
     -n investorcenter
   ```

4. **Deploy Backend Updates**:
   ```bash
   make build
   docker build -t investorcenter-backend:latest .
   docker tag investorcenter-backend:latest 360358043271.dkr.ecr.us-east-1.amazonaws.com/investorcenter-backend:latest
   docker push 360358043271.dkr.ecr.us-east-1.amazonaws.com/investorcenter-backend:latest
   kubectl rollout restart deployment/investorcenter-backend -n investorcenter
   ```

### Monitoring & Logging

1. **Check CronJob Status**:
   ```bash
   kubectl get cronjobs -n investorcenter
   kubectl get jobs -n investorcenter
   ```

2. **View Alert Processor Logs**:
   ```bash
   kubectl logs -l job-name=price-alert-processor -n investorcenter --tail=100
   ```

3. **Monitor Email Delivery**:
   - SendGrid Dashboard: Track delivery, opens, clicks
   - Database query: `SELECT COUNT(*) FROM alert_logs WHERE notification_sent = true`

4. **Set up Prometheus Metrics**:
   ```go
   // In alert_processor.go
   var (
       alertsProcessed = prometheus.NewCounterVec(
           prometheus.CounterOpts{
               Name: "alerts_processed_total",
               Help: "Total number of alerts processed",
           },
           []string{"alert_type", "status"},
       )

       alertProcessingDuration = prometheus.NewHistogram(
           prometheus.HistogramOpts{
               Name: "alert_processing_duration_seconds",
               Help: "Alert processing duration",
           },
       )
   )
   ```

---

## Testing Strategy

### Unit Tests

```go
// backend/services/alert_processor_test.go
package services

import (
    "testing"
    "github.com/stretchr/testify/assert"
)

func TestEvaluatePriceAlert(t *testing.T) {
    processor := NewAlertProcessor(mockDB, mockPolygon, mockNotification)

    alert := &models.AlertRule{
        AlertType: "price_above",
        Symbol: "AAPL",
        Conditions: []byte(`{"threshold": 150.00, "comparison": "above"}`),
    }

    // Mock Polygon API to return price = 152.00
    mockPolygon.On("GetRealTimeQuote", "AAPL").Return(&Quote{Price: 152.00}, nil)

    triggered, conditionMet, marketData, err := processor.evaluatePriceAlert(context.Background(), alert)

    assert.NoError(t, err)
    assert.True(t, triggered)
    assert.NotNil(t, conditionMet)
    assert.NotNil(t, marketData)
}
```

### Integration Tests

```go
// backend/handlers/alert_handlers_test.go
func TestCreateAlertRule_FreeTierLimit(t *testing.T) {
    router := setupTestRouter()

    // Create 10 alerts (free tier limit)
    for i := 0; i < 10; i++ {
        req := createAlertRequest(t, validAlertData)
        resp := performRequest(router, req)
        assert.Equal(t, 201, resp.Code)
    }

    // 11th alert should fail
    req := createAlertRequest(t, validAlertData)
    resp := performRequest(router, req)
    assert.Equal(t, 403, resp.Code)
    assert.Contains(t, resp.Body.String(), "Alert limit reached")
}
```

### E2E Tests

```typescript
// e2e/alerts.spec.ts
import { test, expect } from '@playwright/test';

test('create price alert and verify in list', async ({ page }) => {
  await page.goto('/auth/login');
  await page.fill('input[name="email"]', 'test@example.com');
  await page.fill('input[name="password"]', 'password123');
  await page.click('button[type="submit"]');

  await page.goto('/alerts');
  await page.click('button:has-text("Create Alert")');

  await page.selectOption('select[name="alert_type"]', 'price_above');
  await page.fill('input[name="symbol"]', 'AAPL');
  await page.fill('input[name="threshold"]', '150.00');
  await page.fill('input[name="name"]', 'AAPL above $150');
  await page.click('button:has-text("Create")');

  await expect(page.locator('text=AAPL above $150')).toBeVisible();
});
```

---

## Performance & Scaling

### Database Optimization

1. **Indexes for Alert Processing**:
   ```sql
   -- Already defined in schema, but verify:
   CREATE INDEX idx_alert_rules_active ON alert_rules(is_active) WHERE is_active = true;
   CREATE INDEX idx_alert_rules_symbol ON alert_rules(symbol);
   ```

2. **Partitioning for Alert Logs** (if volume is high):
   ```sql
   -- Partition by month
   CREATE TABLE alert_logs_2025_01 PARTITION OF alert_logs
   FOR VALUES FROM ('2025-01-01') TO ('2025-02-01');
   ```

3. **Query Optimization**:
   ```sql
   -- Use batch fetching for active alerts
   SELECT * FROM alert_rules
   WHERE is_active = true
   AND (frequency = 'always' OR
        (frequency = 'daily' AND (last_triggered_at IS NULL OR last_triggered_at < NOW() - INTERVAL '24 hours')) OR
        (frequency = 'once' AND last_triggered_at IS NULL))
   ORDER BY symbol, user_id;
   ```

### Redis Caching

```go
// Cache alert evaluation results to prevent duplicate processing
type AlertCache struct {
    redis *redis.Client
}

func (ac *AlertCache) CheckRecentlyProcessed(alertID string) (bool, error) {
    key := fmt.Sprintf("alert:processed:%s", alertID)
    val, err := ac.redis.Get(key).Result()
    if err == redis.Nil {
        return false, nil
    }
    if err != nil {
        return false, err
    }
    return val == "1", nil
}

func (ac *AlertCache) MarkProcessed(alertID string, ttl time.Duration) error {
    key := fmt.Sprintf("alert:processed:%s", alertID)
    return ac.redis.Set(key, "1", ttl).Err()
}
```

### Rate Limiting

```go
// Prevent alert spam per user
func (ap *AlertProcessor) checkRateLimit(userID string) (bool, error) {
    key := fmt.Sprintf("alert:rate_limit:%s", userID)
    count, err := ap.redis.Incr(key).Result()
    if err != nil {
        return false, err
    }

    if count == 1 {
        // Set expiry on first increment
        ap.redis.Expire(key, 24*time.Hour)
    }

    // Check user's tier limit
    limit, err := ap.getUserAlertLimit(userID)
    if err != nil {
        return false, err
    }

    return count <= int64(limit), nil
}
```

### Horizontal Scaling

For high alert volumes, use distributed job processing:

```yaml
# k8s/deployments/alert-worker-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: alert-worker
  namespace: investorcenter
spec:
  replicas: 3  # Scale based on load
  selector:
    matchLabels:
      app: alert-worker
  template:
    metadata:
      labels:
        app: alert-worker
    spec:
      containers:
      - name: worker
        image: investorcenter-backend:latest
        command: ["/app/alert-worker"]
        env:
        - name: WORKER_ID
          valueFrom:
            fieldRef:
              fieldPath: metadata.name
        - name: REDIS_HOST
          value: redis-service
```

---

## Security Considerations

### 1. Input Validation

```go
func validateAlertConditions(alertType string, conditions json.RawMessage) error {
    switch alertType {
    case "price_above", "price_below":
        var cond models.PriceAboveCondition
        if err := json.Unmarshal(conditions, &cond); err != nil {
            return fmt.Errorf("invalid condition format")
        }
        if cond.Threshold <= 0 {
            return fmt.Errorf("threshold must be positive")
        }
    case "price_change_pct":
        var cond models.PriceChangeCondition
        if err := json.Unmarshal(conditions, &cond); err != nil {
            return fmt.Errorf("invalid condition format")
        }
        if cond.PercentChange < 0 || cond.PercentChange > 100 {
            return fmt.Errorf("percent change must be between 0-100")
        }
    }
    return nil
}
```

### 2. SQL Injection Prevention

All queries use parameterized statements via sqlx:

```go
func (db *DB) CreateAlertLog(log *models.AlertLog) error {
    query := `
        INSERT INTO alert_logs (
            alert_rule_id, user_id, symbol, alert_type,
            condition_met, market_data
        ) VALUES ($1, $2, $3, $4, $5, $6)
        RETURNING id, triggered_at
    `
    return db.QueryRow(query,
        log.AlertRuleID, log.UserID, log.Symbol, log.AlertType,
        log.ConditionMet, log.MarketData,
    ).Scan(&log.ID, &log.TriggeredAt)
}
```

### 3. Email Security

- **Prevent Email Injection**: Sanitize all user inputs in email content
- **Unsubscribe Links**: Include signed tokens to prevent spoofing
- **Rate Limiting**: Limit emails per user per day (enforced in DB)

```go
func generateUnsubscribeToken(userID, email string, secret string) string {
    h := hmac.New(sha256.New, []byte(secret))
    h.Write([]byte(userID + email))
    return hex.EncodeToString(h.Sum(nil))
}
```

### 4. Notification Privacy

- Only show user's own notifications
- Never expose notification data in logs
- Expire old notifications automatically (see `expires_at` in schema)

---

## Implementation Timeline

### Phase 4: Price & Volume Alerts (3 weeks)

**Week 1: Backend Foundation**
- [ ] Create database migrations (alert_rules, alert_logs)
- [ ] Implement alert models and database layer
- [ ] Build alert service with CRUD operations
- [ ] Create alert handlers and API endpoints
- [ ] Write unit tests for alert evaluation logic

**Week 2: Alert Processing Engine**
- [ ] Build alert processor service
- [ ] Implement price alert evaluation
- [ ] Implement volume alert evaluation
- [ ] Create worker CLI for CronJob
- [ ] Deploy price alert CronJob to Kubernetes
- [ ] Test alert processing end-to-end

**Week 3: Frontend & Integration**
- [ ] Build alert rules management page
- [ ] Create alert rule list and card components
- [ ] Implement create/edit alert modal
- [ ] Build alert condition builder UI
- [ ] Add alert logs history page
- [ ] Integration testing and bug fixes

### Phase 5: News & Financial Event Alerts (2 weeks)

**Week 1: News Integration**
- [ ] Integrate Finnhub API for news
- [ ] Implement news alert evaluation logic
- [ ] Add news keywords and sentiment filtering
- [ ] Create news alert processor CronJob
- [ ] Test news alert triggering

**Week 2: Financial Events**
- [ ] Add earnings alert support
- [ ] Add SEC filing alert support
- [ ] Add dividend alert support
- [ ] Update frontend for new alert types
- [ ] End-to-end testing

### Phase 6: Notification System (3 weeks)

**Week 1: Notification Preferences**
- [ ] Create notification preferences table
- [ ] Build notification preferences API
- [ ] Implement quiet hours logic
- [ ] Create notification preferences page
- [ ] Add email verification for custom email

**Week 2: Email Notifications**
- [ ] Design and build email templates
- [ ] Extend email service for alerts
- [ ] Implement unsubscribe functionality
- [ ] Test email delivery with SendGrid
- [ ] Add email tracking (opens, clicks)

**Week 3: Digests & In-App Notifications**
- [ ] Build digest generation service
- [ ] Create daily digest CronJob
- [ ] Create weekly digest CronJob
- [ ] Implement notification queue
- [ ] Build notification bell component
- [ ] Build notification dropdown
- [ ] Test digest delivery

### Phase 7: Premium Features & Polish (2 weeks)

**Week 1: Premium Tier**
- [ ] Create subscription tables
- [ ] Implement subscription service
- [ ] Add tier enforcement middleware
- [ ] Build subscription management page
- [ ] Create upgrade modal and CTAs
- [ ] Add Stripe integration (payment processing)

**Week 2: Polish & Optimization**
- [ ] Performance optimization (caching, indexing)
- [ ] Add analytics tracking
- [ ] Comprehensive E2E testing
- [ ] Documentation updates
- [ ] Security audit
- [ ] Production deployment
- [ ] User acceptance testing

**Total Timeline: 10 weeks (2.5 months)**

---

## Appendix

### API Endpoint Summary

```
# Alert Rules
GET    /api/v1/alerts                      # List user's alerts
POST   /api/v1/alerts                      # Create alert
GET    /api/v1/alerts/:id                  # Get alert details
PUT    /api/v1/alerts/:id                  # Update alert
DELETE /api/v1/alerts/:id                  # Delete alert

# Alert Logs
GET    /api/v1/alerts/logs                 # Get alert history
POST   /api/v1/alerts/logs/:id/read        # Mark as read
POST   /api/v1/alerts/logs/:id/dismiss     # Dismiss alert

# Notifications
GET    /api/v1/notifications               # Get in-app notifications
POST   /api/v1/notifications/:id/read      # Mark as read
POST   /api/v1/notifications/:id/dismiss   # Dismiss notification
POST   /api/v1/notifications/read-all      # Mark all as read

# Notification Preferences
GET    /api/v1/notifications/preferences   # Get preferences
PUT    /api/v1/notifications/preferences   # Update preferences

# Digests
GET    /api/v1/digests                     # Get digest history
GET    /api/v1/digests/:id                 # Get specific digest

# Subscriptions
GET    /api/v1/subscriptions/plans         # List available plans
GET    /api/v1/subscriptions/current       # Get user's subscription
POST   /api/v1/subscriptions/subscribe     # Subscribe to plan
POST   /api/v1/subscriptions/cancel        # Cancel subscription
POST   /api/v1/subscriptions/upgrade       # Upgrade plan
```

### Environment Variables

```bash
# Existing
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=password
DB_NAME=investorcenter_db
DB_SSLMODE=disable
POLYGON_API_KEY=your_polygon_key

# New for Phases 4-7
SENDGRID_API_KEY=your_sendgrid_key
FINNHUB_API_KEY=your_finnhub_key
REDIS_HOST=localhost
REDIS_PORT=6379
STRIPE_API_KEY=your_stripe_key
STRIPE_WEBHOOK_SECRET=your_webhook_secret
EMAIL_FROM=noreply@investorcenter.ai
EMAIL_FROM_NAME=InvestorCenter.ai
APP_URL=https://investorcenter.ai
```

### Database Size Estimates

Assuming 10,000 active users:

| Table | Rows/User | Total Rows | Est. Size |
|-------|-----------|------------|-----------|
| alert_rules | 5 | 50,000 | 10 MB |
| alert_logs | 100/month | 1M/month | 200 MB/month |
| notification_queue | 20 | 200,000 | 20 MB |
| digest_logs | 30/month | 300K/month | 50 MB/month |

**Total Growth: ~250 MB/month**

### External API Rate Limits

| Service | Free Tier | Cost |
|---------|-----------|------|
| Polygon.io | 5 calls/min | $199/mo unlimited |
| Finnhub | 60 calls/min | $89/mo for 300 calls/min |
| SendGrid | 100 emails/day | $19.95/mo for 50K emails |

---

## Conclusion

This technical specification provides a comprehensive blueprint for implementing Phases 4-7 of the InvestorCenter.ai watchlist alert system. The architecture is designed to be:

- **Scalable**: Handles 10,000+ concurrent users
- **Reliable**: Fault-tolerant with retries and error handling
- **Maintainable**: Clear separation of concerns, well-documented
- **Secure**: Input validation, rate limiting, privacy controls
- **Cost-effective**: Optimized API usage, efficient caching

Key success metrics:
- Alert processing latency < 60 seconds
- Email delivery rate > 99%
- Database query time < 100ms (P95)
- Zero missed alerts (reliability)
- Premium conversion rate > 5%

Next steps: Begin Phase 4 implementation with database migrations and backend foundation.
