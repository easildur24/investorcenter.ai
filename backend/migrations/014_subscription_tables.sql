-- Subscription Plans table (Phase 7)
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
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- User Subscriptions table
CREATE TABLE IF NOT EXISTS user_subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    plan_id UUID NOT NULL REFERENCES subscription_plans(id),

    -- Subscription Status
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    billing_period VARCHAR(20) NOT NULL DEFAULT 'monthly',

    -- Dates
    started_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    current_period_start TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    current_period_end TIMESTAMP WITH TIME ZONE,
    canceled_at TIMESTAMP WITH TIME ZONE,
    ended_at TIMESTAMP WITH TIME ZONE,

    -- Payment
    stripe_subscription_id VARCHAR(255) UNIQUE,
    stripe_customer_id VARCHAR(255),
    payment_method VARCHAR(50),
    last_payment_date TIMESTAMP WITH TIME ZONE,
    next_payment_date TIMESTAMP WITH TIME ZONE,

    -- Metadata
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT valid_status CHECK (status IN ('active', 'past_due', 'canceled', 'trialing', 'incomplete', 'incomplete_expired', 'unpaid')),
    CONSTRAINT valid_billing_period CHECK (billing_period IN ('monthly', 'yearly'))
);

-- Payment History table
CREATE TABLE IF NOT EXISTS payment_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    subscription_id UUID REFERENCES user_subscriptions(id) ON DELETE SET NULL,

    -- Payment Details
    amount DECIMAL(10, 2) NOT NULL,
    currency VARCHAR(3) DEFAULT 'USD',
    status VARCHAR(20) NOT NULL,
    payment_method VARCHAR(50),

    -- External References
    stripe_payment_intent_id VARCHAR(255) UNIQUE,
    stripe_invoice_id VARCHAR(255),

    -- Metadata
    description TEXT,
    receipt_url TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT valid_payment_status CHECK (status IN ('succeeded', 'pending', 'failed', 'refunded', 'canceled'))
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_user_subscriptions_user_id ON user_subscriptions(user_id);
CREATE INDEX IF NOT EXISTS idx_user_subscriptions_plan_id ON user_subscriptions(plan_id);
CREATE INDEX IF NOT EXISTS idx_user_subscriptions_status ON user_subscriptions(status);
CREATE INDEX IF NOT EXISTS idx_user_subscriptions_next_payment ON user_subscriptions(next_payment_date) WHERE status = 'active';
CREATE INDEX IF NOT EXISTS idx_payment_history_user_id ON payment_history(user_id);
CREATE INDEX IF NOT EXISTS idx_payment_history_subscription_id ON payment_history(subscription_id);
CREATE INDEX IF NOT EXISTS idx_payment_history_created_at ON payment_history(created_at DESC);

-- Insert default subscription plans
INSERT INTO subscription_plans (name, display_name, description, price_monthly, price_yearly, max_watch_lists, max_items_per_watch_list, max_alert_rules, max_heatmap_configs, features)
VALUES
    ('free', 'Free', 'Basic features for casual investors', 0.00, 0.00, 3, 10, 10, 3,
     '{"realtime_data": false, "advanced_alerts": false, "priority_support": false, "api_access": false}'::jsonb),
    ('premium', 'Premium', 'Advanced features for active investors', 19.99, 199.99, 20, 100, 100, 10,
     '{"realtime_data": true, "advanced_alerts": true, "priority_support": true, "api_access": true, "custom_dashboards": true, "export_data": true}'::jsonb),
    ('enterprise', 'Enterprise', 'Full-featured solution for professional traders', 99.99, 999.99, -1, -1, -1, -1,
     '{"realtime_data": true, "advanced_alerts": true, "priority_support": true, "api_access": true, "custom_dashboards": true, "export_data": true, "white_label": true, "dedicated_account_manager": true}'::jsonb)
ON CONFLICT (name) DO NOTHING;

-- Trigger to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_subscriptions_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS update_subscription_plans_updated_at ON subscription_plans;
CREATE TRIGGER update_subscription_plans_updated_at
BEFORE UPDATE ON subscription_plans
FOR EACH ROW EXECUTE FUNCTION update_subscriptions_updated_at();

DROP TRIGGER IF EXISTS update_user_subscriptions_updated_at ON user_subscriptions;
CREATE TRIGGER update_user_subscriptions_updated_at
BEFORE UPDATE ON user_subscriptions
FOR EACH ROW EXECUTE FUNCTION update_subscriptions_updated_at();

-- Function to get user's subscription limits
CREATE OR REPLACE FUNCTION get_user_subscription_limits(p_user_id UUID)
RETURNS TABLE (
    max_watch_lists INTEGER,
    max_items_per_watch_list INTEGER,
    max_alert_rules INTEGER,
    max_heatmap_configs INTEGER,
    features JSONB
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        sp.max_watch_lists,
        sp.max_items_per_watch_list,
        sp.max_alert_rules,
        sp.max_heatmap_configs,
        sp.features
    FROM user_subscriptions us
    JOIN subscription_plans sp ON us.plan_id = sp.id
    WHERE us.user_id = p_user_id
      AND us.status = 'active'
    UNION ALL
    SELECT 3, 10, 10, 3, '{}'::jsonb
    WHERE NOT EXISTS (
        SELECT 1 FROM user_subscriptions
        WHERE user_id = p_user_id AND status = 'active'
    )
    LIMIT 1;
END;
$$ LANGUAGE plpgsql;
