-- Notification Preferences table (Phase 6)
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

-- Notification Queue table
CREATE TABLE IF NOT EXISTS notification_queue (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    alert_log_id UUID REFERENCES alert_logs(id) ON DELETE SET NULL,

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

-- Indexes for notification_queue
CREATE INDEX IF NOT EXISTS idx_notification_queue_user_id ON notification_queue(user_id);
CREATE INDEX IF NOT EXISTS idx_notification_queue_unread ON notification_queue(user_id, is_read) WHERE is_read = false;
CREATE INDEX IF NOT EXISTS idx_notification_queue_created_at ON notification_queue(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_notification_queue_expires_at ON notification_queue(expires_at) WHERE is_dismissed = false;

-- Digest Logs table
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

CREATE INDEX IF NOT EXISTS idx_digest_logs_user_id ON digest_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_digest_logs_sent_at ON digest_logs(sent_at DESC);

-- Trigger to update updated_at timestamp for notification_preferences
CREATE OR REPLACE FUNCTION update_notification_preferences_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS update_notification_preferences_updated_at ON notification_preferences;
CREATE TRIGGER update_notification_preferences_updated_at
BEFORE UPDATE ON notification_preferences
FOR EACH ROW EXECUTE FUNCTION update_notification_preferences_updated_at();
