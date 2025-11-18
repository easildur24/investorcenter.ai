-- Cronjob Monitoring Tables

-- Job Schedule Configuration Table
CREATE TABLE IF NOT EXISTS cronjob_schedules (
    id SERIAL PRIMARY KEY,
    job_name VARCHAR(100) UNIQUE NOT NULL,
    job_category VARCHAR(50), -- 'core_pipeline' or 'ic_score_pipeline'
    description TEXT,
    schedule_cron VARCHAR(50), -- e.g., "0 2 * * *"
    schedule_description VARCHAR(200), -- e.g., "Daily at 2 AM UTC"

    -- Status
    is_active BOOLEAN DEFAULT true,

    -- Performance Expectations
    expected_duration_seconds INTEGER, -- Alert if exceeds this
    timeout_seconds INTEGER,

    -- Last Execution Status
    last_success_at TIMESTAMP WITH TIME ZONE,
    last_failure_at TIMESTAMP WITH TIME ZONE,
    consecutive_failures INTEGER DEFAULT 0,

    -- Metadata
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Job Execution Logs Table
CREATE TABLE IF NOT EXISTS cronjob_execution_logs (
    id SERIAL PRIMARY KEY,
    job_name VARCHAR(100) NOT NULL,
    job_category VARCHAR(50), -- 'core_pipeline' or 'ic_score_pipeline'
    execution_id VARCHAR(100) UNIQUE, -- Kubernetes job ID

    -- Execution Status
    status VARCHAR(20) NOT NULL, -- 'running', 'success', 'failed', 'timeout'
    started_at TIMESTAMP WITH TIME ZONE NOT NULL,
    completed_at TIMESTAMP WITH TIME ZONE,
    duration_seconds INTEGER,

    -- Metrics
    records_processed INTEGER DEFAULT 0,
    records_updated INTEGER DEFAULT 0,
    records_failed INTEGER DEFAULT 0,

    -- Error Tracking
    error_message TEXT,
    error_stack_trace TEXT,

    -- Metadata
    k8s_pod_name VARCHAR(200),
    k8s_namespace VARCHAR(100),
    exit_code INTEGER,

    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT valid_status CHECK (status IN ('running', 'success', 'failed', 'timeout'))
);

-- Alert Configuration Table
CREATE TABLE IF NOT EXISTS cronjob_alerts (
    id SERIAL PRIMARY KEY,
    job_name VARCHAR(100),
    alert_type VARCHAR(50), -- 'failure', 'timeout', 'performance_degradation', 'missed_schedule'
    alert_threshold INTEGER, -- e.g., 3 consecutive failures
    is_active BOOLEAN DEFAULT true,
    notification_channels JSONB, -- ['email', 'slack']
    last_triggered_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT valid_alert_type CHECK (alert_type IN ('failure', 'timeout', 'performance_degradation', 'missed_schedule'))
);

-- Indexes for cronjob_execution_logs
CREATE INDEX IF NOT EXISTS idx_cronjob_logs_job_name_started ON cronjob_execution_logs(job_name, started_at DESC);
CREATE INDEX IF NOT EXISTS idx_cronjob_logs_status_started ON cronjob_execution_logs(status, started_at DESC);
CREATE INDEX IF NOT EXISTS idx_cronjob_logs_job_category ON cronjob_execution_logs(job_category);
CREATE INDEX IF NOT EXISTS idx_cronjob_logs_execution_id ON cronjob_execution_logs(execution_id);
CREATE INDEX IF NOT EXISTS idx_cronjob_logs_created_at ON cronjob_execution_logs(created_at DESC);

-- Indexes for cronjob_schedules
CREATE INDEX IF NOT EXISTS idx_cronjob_schedules_job_name ON cronjob_schedules(job_name);
CREATE INDEX IF NOT EXISTS idx_cronjob_schedules_category ON cronjob_schedules(job_category);
CREATE INDEX IF NOT EXISTS idx_cronjob_schedules_active ON cronjob_schedules(is_active) WHERE is_active = true;

-- Indexes for cronjob_alerts
CREATE INDEX IF NOT EXISTS idx_cronjob_alerts_job_name ON cronjob_alerts(job_name);
CREATE INDEX IF NOT EXISTS idx_cronjob_alerts_active ON cronjob_alerts(is_active) WHERE is_active = true;

-- Trigger to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_cronjob_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS update_cronjob_schedules_updated_at ON cronjob_schedules;
CREATE TRIGGER update_cronjob_schedules_updated_at
BEFORE UPDATE ON cronjob_schedules
FOR EACH ROW EXECUTE FUNCTION update_cronjob_updated_at();

DROP TRIGGER IF EXISTS update_cronjob_alerts_updated_at ON cronjob_alerts;
CREATE TRIGGER update_cronjob_alerts_updated_at
BEFORE UPDATE ON cronjob_alerts
FOR EACH ROW EXECUTE FUNCTION update_cronjob_updated_at();

-- Function to update schedule status after log insertion
CREATE OR REPLACE FUNCTION update_cronjob_schedule_status()
RETURNS TRIGGER AS $$
BEGIN
    -- Update last success/failure times and consecutive failures
    IF NEW.status = 'success' THEN
        UPDATE cronjob_schedules
        SET last_success_at = NEW.completed_at,
            consecutive_failures = 0,
            updated_at = CURRENT_TIMESTAMP
        WHERE job_name = NEW.job_name;
    ELSIF NEW.status IN ('failed', 'timeout') THEN
        UPDATE cronjob_schedules
        SET last_failure_at = NEW.completed_at,
            consecutive_failures = consecutive_failures + 1,
            updated_at = CURRENT_TIMESTAMP
        WHERE job_name = NEW.job_name;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS update_schedule_status_trigger ON cronjob_execution_logs;
CREATE TRIGGER update_schedule_status_trigger
AFTER INSERT OR UPDATE OF status ON cronjob_execution_logs
FOR EACH ROW
WHEN (NEW.status IN ('success', 'failed', 'timeout'))
EXECUTE FUNCTION update_cronjob_schedule_status();

-- Function to calculate duration on completion
CREATE OR REPLACE FUNCTION calculate_execution_duration()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.completed_at IS NOT NULL AND NEW.started_at IS NOT NULL THEN
        NEW.duration_seconds = EXTRACT(EPOCH FROM (NEW.completed_at - NEW.started_at))::INTEGER;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS calculate_duration_trigger ON cronjob_execution_logs;
CREATE TRIGGER calculate_duration_trigger
BEFORE INSERT OR UPDATE OF completed_at ON cronjob_execution_logs
FOR EACH ROW
EXECUTE FUNCTION calculate_execution_duration();

-- Function to get job statistics
CREATE OR REPLACE FUNCTION get_cronjob_statistics(
    p_job_name VARCHAR DEFAULT NULL,
    p_days INTEGER DEFAULT 7
)
RETURNS TABLE (
    job_name VARCHAR,
    total_executions BIGINT,
    successful_executions BIGINT,
    failed_executions BIGINT,
    success_rate DECIMAL,
    avg_duration_seconds DECIMAL,
    p50_duration_seconds DECIMAL,
    p95_duration_seconds DECIMAL,
    last_execution_status VARCHAR,
    last_execution_at TIMESTAMP WITH TIME ZONE
) AS $$
BEGIN
    RETURN QUERY
    WITH execution_stats AS (
        SELECT
            cel.job_name,
            COUNT(*) as total_executions,
            COUNT(*) FILTER (WHERE cel.status = 'success') as successful_executions,
            COUNT(*) FILTER (WHERE cel.status IN ('failed', 'timeout')) as failed_executions,
            AVG(cel.duration_seconds) FILTER (WHERE cel.status = 'success') as avg_duration,
            PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY cel.duration_seconds) FILTER (WHERE cel.status = 'success') as p50_duration,
            PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY cel.duration_seconds) FILTER (WHERE cel.status = 'success') as p95_duration
        FROM cronjob_execution_logs cel
        WHERE (p_job_name IS NULL OR cel.job_name = p_job_name)
          AND cel.started_at >= CURRENT_TIMESTAMP - (p_days || ' days')::INTERVAL
        GROUP BY cel.job_name
    ),
    latest_execution AS (
        SELECT DISTINCT ON (cel.job_name)
            cel.job_name,
            cel.status as last_status,
            COALESCE(cel.completed_at, cel.started_at) as last_execution_time
        FROM cronjob_execution_logs cel
        WHERE (p_job_name IS NULL OR cel.job_name = p_job_name)
        ORDER BY cel.job_name, cel.started_at DESC
    )
    SELECT
        es.job_name::VARCHAR,
        es.total_executions,
        es.successful_executions,
        es.failed_executions,
        ROUND((es.successful_executions::DECIMAL / NULLIF(es.total_executions, 0)) * 100, 2) as success_rate,
        ROUND(es.avg_duration, 2) as avg_duration_seconds,
        ROUND(es.p50_duration, 2) as p50_duration_seconds,
        ROUND(es.p95_duration, 2) as p95_duration_seconds,
        le.last_status::VARCHAR,
        le.last_execution_time
    FROM execution_stats es
    LEFT JOIN latest_execution le ON es.job_name = le.job_name;
END;
$$ LANGUAGE plpgsql;

-- Insert initial cronjob schedule configurations
INSERT INTO cronjob_schedules (job_name, job_category, description, schedule_cron, schedule_description, expected_duration_seconds, timeout_seconds)
VALUES
    -- Core Pipeline
    ('sec-filing-daily', 'core_pipeline', 'Fetches SEC filing metadata (10-K, 10-Q)', '0 2 * * *', 'Daily at 2 AM UTC (9 PM EST)', 1200, 43200),
    ('polygon-ticker-update', 'core_pipeline', 'Incremental ticker updates from Polygon API', '30 6 * * *', 'Daily at 6:30 AM UTC', 300, 3600),
    ('polygon-volume-update', 'core_pipeline', 'Updates volume and price data for top 500 tickers', '0 14,18,22 * * *', 'Every 4 hours during market hours', 600, 3600),
    ('reddit-collector', 'core_pipeline', 'Collects Reddit stock rankings from ApeWisdom API', '0 2 * * *', 'Daily at 2 AM UTC', 150, 1800),

    -- IC Score Pipeline
    ('ic-score-sec-financials', 'ic_score_pipeline', 'Fetches 5 years of quarterly financial statements from SEC EDGAR', '0 2 * * 0', 'Weekly on Sunday at 2:00 AM UTC', 14400, 43200),
    ('ic-score-technical-indicators', 'ic_score_pipeline', 'Calculates technical indicators (RSI, MACD, moving averages)', '0 23 * * 1-5', 'Daily at 11:00 PM UTC on weekdays', 5400, 7200),
    ('ic-score-13f-holdings', 'ic_score_pipeline', 'Processes institutional 13F holdings filings', '0 3 15 1,4,7,10 *', 'Quarterly on 15th at 3 AM UTC', 10800, 21600),
    ('ic-score-insider-trades', 'ic_score_pipeline', 'Fetches Form 4 insider trading data from SEC', '0 14-21 * * 1-5', 'Hourly during market hours', 1800, 3600),
    ('ic-score-analyst-ratings', 'ic_score_pipeline', 'Fetches Wall Street analyst ratings from Benzinga API', '0 4 * * *', 'Daily at 4:00 AM UTC', 3600, 7200),
    ('ic-score-news-sentiment', 'ic_score_pipeline', 'Fetches and analyzes news sentiment from Polygon API', '0 */4 * * *', 'Every 4 hours', 1800, 3600),
    ('ic-score-calculator', 'ic_score_pipeline', 'Calculates final IC Scores using 10-factor weighted model', '0 0 * * *', 'Daily at 12:00 AM UTC (7 PM ET)', 7200, 14400)
ON CONFLICT (job_name) DO NOTHING;

-- Insert default alert configurations (3 consecutive failures for all jobs)
INSERT INTO cronjob_alerts (job_name, alert_type, alert_threshold, notification_channels)
SELECT
    job_name,
    'failure',
    3,
    '["email"]'::JSONB
FROM cronjob_schedules
WHERE NOT EXISTS (
    SELECT 1 FROM cronjob_alerts ca
    WHERE ca.job_name = cronjob_schedules.job_name
    AND ca.alert_type = 'failure'
);

-- Insert timeout alerts for critical jobs
INSERT INTO cronjob_alerts (job_name, alert_type, alert_threshold, notification_channels)
VALUES
    ('ic-score-sec-financials', 'timeout', 1, '["email"]'::JSONB),
    ('ic-score-technical-indicators', 'timeout', 1, '["email"]'::JSONB),
    ('ic-score-calculator', 'timeout', 1, '["email"]'::JSONB)
ON CONFLICT DO NOTHING;
