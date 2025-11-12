-- Alert Rules table (Phase 4)
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

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_alert_rules_user_id ON alert_rules(user_id);
CREATE INDEX IF NOT EXISTS idx_alert_rules_watch_list_id ON alert_rules(watch_list_id);
CREATE INDEX IF NOT EXISTS idx_alert_rules_symbol ON alert_rules(symbol);
CREATE INDEX IF NOT EXISTS idx_alert_rules_active ON alert_rules(is_active) WHERE is_active = true;
CREATE INDEX IF NOT EXISTS idx_alert_rules_type ON alert_rules(alert_type);
CREATE INDEX IF NOT EXISTS idx_alert_rules_created_at ON alert_rules(created_at DESC);

-- Alert Logs table
CREATE TABLE IF NOT EXISTS alert_logs (
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

-- Indexes for alert_logs
CREATE INDEX IF NOT EXISTS idx_alert_logs_alert_rule_id ON alert_logs(alert_rule_id);
CREATE INDEX IF NOT EXISTS idx_alert_logs_user_id ON alert_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_alert_logs_symbol ON alert_logs(symbol);
CREATE INDEX IF NOT EXISTS idx_alert_logs_triggered_at ON alert_logs(triggered_at DESC);
CREATE INDEX IF NOT EXISTS idx_alert_logs_unread ON alert_logs(user_id, is_read) WHERE is_read = false;

-- Trigger to update updated_at timestamp for alert_rules
CREATE OR REPLACE FUNCTION update_alert_rules_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS update_alert_rules_updated_at ON alert_rules;
CREATE TRIGGER update_alert_rules_updated_at
BEFORE UPDATE ON alert_rules
FOR EACH ROW EXECUTE FUNCTION update_alert_rules_updated_at();
