-- Migration 012: Alert System (Phases 4-7)
-- Date: 2025-11-12
-- Description: Creates tables for alerts, notifications, and subscriptions

-- ============================================================================
-- Phase 4 & 5: Alert Tables
-- ============================================================================

-- Table: alert_rules
-- Stores user-defined alert rules for watch list items
CREATE TABLE IF NOT EXISTS alert_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    watch_list_id UUID NOT NULL REFERENCES watch_lists(id) ON DELETE CASCADE,
    watch_list_item_id UUID REFERENCES watch_list_items(id) ON DELETE CASCADE,
    symbol VARCHAR(20) NOT NULL,

    -- Alert Type
    alert_type VARCHAR(50) NOT NULL,

    -- Condition Parameters (JSONB for flexibility)
    conditions JSONB NOT NULL,

    -- Alert Settings
    is_active BOOLEAN DEFAULT true,
    frequency VARCHAR(20) DEFAULT 'once',
    notify_email BOOLEAN DEFAULT true,
    notify_in_app BOOLEAN DEFAULT true,

    -- Metadata
    name VARCHAR(255) NOT NULL,
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

-- Indexes for alert_rules
CREATE INDEX IF NOT EXISTS idx_alert_rules_user_id ON alert_rules(user_id);
CREATE INDEX IF NOT EXISTS idx_alert_rules_watch_list_id ON alert_rules(watch_list_id);
CREATE INDEX IF NOT EXISTS idx_alert_rules_symbol ON alert_rules(symbol);
CREATE INDEX IF NOT EXISTS idx_alert_rules_active ON alert_rules(is_active) WHERE is_active = true;
CREATE INDEX IF NOT EXISTS idx_alert_rules_type ON alert_rules(alert_type);
CREATE INDEX IF NOT EXISTS idx_alert_rules_created_at ON alert_rules(created_at DESC);

-- Table: alert_trigger_logs
-- Stores history of triggered alerts
CREATE TABLE IF NOT EXISTS alert_trigger_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    alert_rule_id UUID NOT NULL REFERENCES alert_rules(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    symbol VARCHAR(20) NOT NULL,

    -- Trigger Details
    triggered_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    alert_type VARCHAR(50) NOT NULL,
    condition_met JSONB NOT NULL,

    -- Market Data Snapshot
    market_data JSONB NOT NULL,

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

-- Indexes for alert_trigger_logs
CREATE INDEX IF NOT EXISTS idx_alert_logs_alert_rule_id ON alert_trigger_logs(alert_rule_id);
CREATE INDEX IF NOT EXISTS idx_alert_logs_user_id ON alert_trigger_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_alert_logs_symbol ON alert_trigger_logs(symbol);
CREATE INDEX IF NOT EXISTS idx_alert_logs_triggered_at ON alert_trigger_logs(triggered_at DESC);
CREATE INDEX IF NOT EXISTS idx_alert_logs_unread ON alert_trigger_logs(user_id, is_read) WHERE is_read = false;

-- ============================================================================
-- Phase 6: Notification Tables
-- ============================================================================

-- Table: notification_preferences
-- Stores user notification settings
CREATE TABLE IF NOT EXISTS notification_preferences (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,

    -- Email Preferences
    email_enabled BOOLEAN DEFAULT true,
    email_address VARCHAR(255),
    email_verified BOOLEAN DEFAULT false,

    -- Notification Types
    price_alerts_enabled BOOLEAN DEFAULT true,
    volume_alerts_enabled BOOLEAN DEFAULT true,
    news_alerts_enabled BOOLEAN DEFAULT true,
    earnings_alerts_enabled BOOLEAN DEFAULT true,
    sec_filing_alerts_enabled BOOLEAN DEFAULT true,

    -- Digest Settings
    daily_digest_enabled BOOLEAN DEFAULT false,
    daily_digest_time TIME DEFAULT '09:00:00',
    weekly_digest_enabled BOOLEAN DEFAULT false,
    weekly_digest_day INTEGER DEFAULT 1,
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

    -- Rate Limiting
    max_alerts_per_day INTEGER DEFAULT 50,
    max_emails_per_day INTEGER DEFAULT 20,

    -- Metadata
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Trigger to create default preferences for new users
CREATE OR REPLACE FUNCTION create_default_notification_preferences()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO notification_preferences (user_id, email_address)
    VALUES (NEW.id, NEW.email)
    ON CONFLICT (user_id) DO NOTHING;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trigger_create_default_notification_preferences ON users;
CREATE TRIGGER trigger_create_default_notification_preferences
AFTER INSERT ON users
FOR EACH ROW
EXECUTE FUNCTION create_default_notification_preferences();

-- Table: notifications
-- In-app notification queue
CREATE TABLE IF NOT EXISTS notifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    alert_log_id UUID REFERENCES alert_trigger_logs(id) ON DELETE SET NULL,

    -- Notification Content
    type VARCHAR(50) NOT NULL,
    title VARCHAR(255) NOT NULL,
    message TEXT NOT NULL,
    metadata JSONB,

    -- Status
    is_read BOOLEAN DEFAULT false,
    read_at TIMESTAMP WITH TIME ZONE,
    is_dismissed BOOLEAN DEFAULT false,
    dismissed_at TIMESTAMP WITH TIME ZONE,

    -- Scheduling
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP WITH TIME ZONE DEFAULT (CURRENT_TIMESTAMP + INTERVAL '30 days')
);

-- Indexes for notifications
CREATE INDEX IF NOT EXISTS idx_notifications_user_id ON notifications(user_id);
CREATE INDEX IF NOT EXISTS idx_notifications_unread ON notifications(user_id, is_read) WHERE is_read = false;
CREATE INDEX IF NOT EXISTS idx_notifications_created_at ON notifications(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_notifications_expires_at ON notifications(expires_at) WHERE is_dismissed = false;

-- Table: digest_logs
-- Tracks sent digests to prevent duplicates
CREATE TABLE IF NOT EXISTS digest_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    digest_type VARCHAR(20) NOT NULL,
    period_start TIMESTAMP WITH TIME ZONE NOT NULL,
    period_end TIMESTAMP WITH TIME ZONE NOT NULL,
    sent_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    email_sent BOOLEAN DEFAULT false,
    email_opened BOOLEAN DEFAULT false,
    email_clicked BOOLEAN DEFAULT false,
    content_snapshot JSONB,

    CONSTRAINT unique_digest_per_period UNIQUE (user_id, digest_type, period_start)
);

-- Indexes for digest_logs
CREATE INDEX IF NOT EXISTS idx_digest_logs_user_id ON digest_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_digest_logs_sent_at ON digest_logs(sent_at DESC);

-- ============================================================================
-- Phase 7: Premium Tier Tables
-- ============================================================================

-- Table: subscription_plans
-- Defines available subscription tiers
CREATE TABLE IF NOT EXISTS subscription_plans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(50) NOT NULL UNIQUE,
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

    -- Metadata
    is_active BOOLEAN DEFAULT true,
    sort_order INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Insert default subscription plans
INSERT INTO subscription_plans (name, display_name, description, price_monthly, price_yearly, max_watch_lists, max_items_per_watch_list, max_alert_rules, max_heatmap_configs, features, sort_order)
VALUES
    ('free', 'Free', 'Basic features for casual investors', 0.00, 0.00, 3, 10, 10, 3,
     '{"real_time_alerts": false, "news_alerts": false, "advanced_charts": false, "api_access": false, "priority_support": false}'::jsonb, 1),
    ('premium', 'Premium', 'Advanced features for active traders', 19.99, 199.00, 20, 100, 100, 20,
     '{"real_time_alerts": true, "news_alerts": true, "advanced_charts": true, "api_access": true, "priority_support": true, "custom_branding": false}'::jsonb, 2),
    ('enterprise', 'Enterprise', 'Unlimited features for professional investors', 99.99, 999.00, -1, -1, -1, -1,
     '{"real_time_alerts": true, "news_alerts": true, "advanced_charts": true, "api_access": true, "priority_support": true, "custom_branding": true}'::jsonb, 3)
ON CONFLICT (name) DO NOTHING;

-- Table: user_subscriptions
-- Tracks user subscriptions and billing
CREATE TABLE IF NOT EXISTS user_subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    plan_id UUID NOT NULL REFERENCES subscription_plans(id),

    -- Subscription Details
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    billing_period VARCHAR(20) NOT NULL,

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

-- Indexes for user_subscriptions
CREATE INDEX IF NOT EXISTS idx_user_subscriptions_user_id ON user_subscriptions(user_id);
CREATE INDEX IF NOT EXISTS idx_user_subscriptions_status ON user_subscriptions(status);
CREATE INDEX IF NOT EXISTS idx_user_subscriptions_period_end ON user_subscriptions(current_period_end);

-- ============================================================================
-- Default User Subscriptions
-- ============================================================================

-- Assign free plan to all existing users
INSERT INTO user_subscriptions (user_id, plan_id, billing_period, current_period_start, current_period_end)
SELECT
    u.id as user_id,
    sp.id as plan_id,
    'monthly' as billing_period,
    CURRENT_TIMESTAMP as current_period_start,
    CURRENT_TIMESTAMP + INTERVAL '100 years' as current_period_end
FROM users u
CROSS JOIN subscription_plans sp
WHERE sp.name = 'free'
ON CONFLICT DO NOTHING;

-- ============================================================================
-- Migration Complete
-- ============================================================================

-- Verify tables created
DO $$
DECLARE
    table_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO table_count
    FROM information_schema.tables
    WHERE table_schema = 'public'
    AND table_name IN (
        'alert_rules', 'alert_trigger_logs', 'notification_preferences',
        'notifications', 'digest_logs', 'subscription_plans', 'user_subscriptions'
    );

    IF table_count = 7 THEN
        RAISE NOTICE 'Migration 012: Alert System - Successfully created % tables', table_count;
    ELSE
        RAISE WARNING 'Migration 012: Alert System - Expected 7 tables, created %', table_count;
    END IF;
END $$;
