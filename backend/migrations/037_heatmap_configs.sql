-- Heatmap Configurations table
-- Stores user's custom heatmap settings for reuse
CREATE TABLE IF NOT EXISTS heatmap_configs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    watch_list_id UUID REFERENCES watch_lists(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,

    -- Metric settings
    size_metric VARCHAR(50) DEFAULT 'market_cap',
        -- Options: 'market_cap', 'volume', 'avg_volume', 'reddit_mentions', 'reddit_popularity'
    color_metric VARCHAR(50) DEFAULT 'price_change_pct',
        -- Options: 'price_change_pct', 'volume_change_pct', 'reddit_rank', 'reddit_trend'
    time_period VARCHAR(10) DEFAULT '1D', -- '1D', '1W', '1M', '3M', '6M', 'YTD', '1Y', '5Y'

    -- Visual settings
    color_scheme VARCHAR(50) DEFAULT 'red_green', -- 'red_green', 'heatmap', 'blue_red', 'custom'
    label_display VARCHAR(50) DEFAULT 'symbol_change', -- 'symbol', 'symbol_change', 'full'
    layout_type VARCHAR(50) DEFAULT 'treemap', -- 'treemap', 'grid'

    -- Filter settings (JSON for flexibility)
    filters_json JSONB DEFAULT '{}'::jsonb,
    -- Example filters:
    -- {
    --   "asset_types": ["stock", "crypto"],
    --   "sectors": ["Technology", "Finance"],
    --   "min_price": 10.0,
    --   "max_price": 1000.0,
    --   "min_market_cap": 1000000000,
    --   "max_market_cap": null
    -- }

    -- Custom color gradient (for custom color scheme)
    color_gradient_json JSONB,
    -- Example: {"negative": "#FF0000", "neutral": "#FFFFFF", "positive": "#00FF00"}

    is_default BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_heatmap_configs_user_id ON heatmap_configs(user_id);
CREATE INDEX IF NOT EXISTS idx_heatmap_configs_watch_list_id ON heatmap_configs(watch_list_id);
CREATE INDEX IF NOT EXISTS idx_heatmap_configs_user_watch_list ON heatmap_configs(user_id, watch_list_id);

-- Trigger to update updated_at
CREATE OR REPLACE FUNCTION update_heatmap_configs_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS update_heatmap_configs_updated_at ON heatmap_configs;
CREATE TRIGGER update_heatmap_configs_updated_at
BEFORE UPDATE ON heatmap_configs
FOR EACH ROW EXECUTE FUNCTION update_heatmap_configs_updated_at();

-- Function to create default heatmap config for new watch lists
CREATE OR REPLACE FUNCTION create_default_heatmap_config()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO heatmap_configs (
        user_id,
        watch_list_id,
        name,
        is_default
    )
    VALUES (
        NEW.user_id,
        NEW.id,
        'Default Heatmap',
        TRUE
    );
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger to auto-create default heatmap config when watch list is created
DROP TRIGGER IF EXISTS auto_create_default_heatmap_config ON watch_lists;
CREATE TRIGGER auto_create_default_heatmap_config
AFTER INSERT ON watch_lists
FOR EACH ROW EXECUTE FUNCTION create_default_heatmap_config();
