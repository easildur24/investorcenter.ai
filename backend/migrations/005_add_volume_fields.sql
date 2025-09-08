-- Migration to add volume and trading data fields to tickers table
-- These fields will store the latest trading volume and related metrics

ALTER TABLE tickers
-- Daily volume data
ADD COLUMN IF NOT EXISTS volume BIGINT,
ADD COLUMN IF NOT EXISTS avg_volume_30d BIGINT,
ADD COLUMN IF NOT EXISTS avg_volume_90d BIGINT,

-- Price data for context with volume
ADD COLUMN IF NOT EXISTS current_price DECIMAL(15,2),
ADD COLUMN IF NOT EXISTS previous_close DECIMAL(15,2),
ADD COLUMN IF NOT EXISTS day_high DECIMAL(15,2),
ADD COLUMN IF NOT EXISTS day_low DECIMAL(15,2),
ADD COLUMN IF NOT EXISTS day_open DECIMAL(15,2),

-- Volume-weighted average price
ADD COLUMN IF NOT EXISTS vwap DECIMAL(15,4),

-- 52-week data
ADD COLUMN IF NOT EXISTS week_52_high DECIMAL(15,2),
ADD COLUMN IF NOT EXISTS week_52_low DECIMAL(15,2),

-- Trading session info
ADD COLUMN IF NOT EXISTS last_trade_timestamp TIMESTAMP WITH TIME ZONE,
ADD COLUMN IF NOT EXISTS market_status VARCHAR(20), -- 'open', 'closed', 'pre-market', 'after-hours'

-- Turnover and liquidity metrics
ADD COLUMN IF NOT EXISTS shares_outstanding BIGINT,
ADD COLUMN IF NOT EXISTS float_shares BIGINT,
ADD COLUMN IF NOT EXISTS turnover_rate DECIMAL(10,4), -- volume/float ratio

-- For crypto: 24h volume in different currencies
ADD COLUMN IF NOT EXISTS volume_24h_usd DECIMAL(20,2),
ADD COLUMN IF NOT EXISTS volume_24h_btc DECIMAL(20,8);

-- Create indexes for volume-related queries
CREATE INDEX IF NOT EXISTS idx_tickers_volume ON tickers(volume) WHERE volume IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_tickers_avg_volume_30d ON tickers(avg_volume_30d) WHERE avg_volume_30d IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_tickers_last_trade ON tickers(last_trade_timestamp) WHERE last_trade_timestamp IS NOT NULL;

-- Add comments for documentation
COMMENT ON COLUMN tickers.volume IS 'Latest daily trading volume';
COMMENT ON COLUMN tickers.avg_volume_30d IS '30-day average daily trading volume';
COMMENT ON COLUMN tickers.avg_volume_90d IS '90-day average daily trading volume';
COMMENT ON COLUMN tickers.vwap IS 'Volume-weighted average price for the day';
COMMENT ON COLUMN tickers.turnover_rate IS 'Daily turnover rate (volume/float)';
COMMENT ON COLUMN tickers.volume_24h_usd IS 'For crypto: 24-hour volume in USD';
COMMENT ON COLUMN tickers.volume_24h_btc IS 'For crypto: 24-hour volume in BTC';