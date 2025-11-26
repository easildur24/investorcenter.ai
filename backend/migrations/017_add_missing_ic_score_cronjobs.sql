-- Add missing IC Score CronJobs to monitoring
-- Date: 2025-11-23
-- Description: Adds TTM Financials, Ticker Sync, and Coverage Monitor cronjobs

-- 1. TTM Financials Calculator (NEW - Phase 2)
INSERT INTO cronjob_schedules (job_name, job_category, description, schedule_cron, schedule_description, expected_duration_seconds, timeout_seconds)
VALUES
    ('ic-score-ttm-financials', 'ic_score_pipeline', 'Calculates Trailing Twelve Months (TTM) financial metrics from SEC quarterly/annual filings', '0 22 * * 1-5', 'Daily at 10:00 PM UTC (5:00 PM ET) on weekdays', 1800, 7200)
ON CONFLICT (job_name) DO UPDATE SET
    job_category = EXCLUDED.job_category,
    description = EXCLUDED.description,
    schedule_cron = EXCLUDED.schedule_cron,
    schedule_description = EXCLUDED.schedule_description,
    expected_duration_seconds = EXCLUDED.expected_duration_seconds,
    timeout_seconds = EXCLUDED.timeout_seconds,
    updated_at = CURRENT_TIMESTAMP;

-- 2. Ticker Sync (Data Sync - Phase 0)
INSERT INTO cronjob_schedules (job_name, job_category, description, schedule_cron, schedule_description, expected_duration_seconds, timeout_seconds)
VALUES
    ('ic-score-ticker-sync', 'data_sync', 'Syncs new tickers from tickers table to companies table for IC Score coverage', '0 2 * * 0', 'Weekly on Sunday at 2:00 AM UTC', 300, 1800)
ON CONFLICT (job_name) DO UPDATE SET
    job_category = EXCLUDED.job_category,
    description = EXCLUDED.description,
    schedule_cron = EXCLUDED.schedule_cron,
    schedule_description = EXCLUDED.schedule_description,
    expected_duration_seconds = EXCLUDED.expected_duration_seconds,
    timeout_seconds = EXCLUDED.timeout_seconds,
    updated_at = CURRENT_TIMESTAMP;

-- 3. Coverage Monitor (Monitoring)
INSERT INTO cronjob_schedules (job_name, job_category, description, schedule_cron, schedule_description, expected_duration_seconds, timeout_seconds)
VALUES
    ('ic-score-coverage-monitor', 'monitoring', 'Monitors IC Score data coverage and alerts if below 85% threshold', '0 8 * * *', 'Daily at 8:00 AM UTC', 60, 300)
ON CONFLICT (job_name) DO UPDATE SET
    job_category = EXCLUDED.job_category,
    description = EXCLUDED.description,
    schedule_cron = EXCLUDED.schedule_cron,
    schedule_description = EXCLUDED.schedule_description,
    expected_duration_seconds = EXCLUDED.expected_duration_seconds,
    timeout_seconds = EXCLUDED.timeout_seconds,
    updated_at = CURRENT_TIMESTAMP;

-- Add default failure alerts for all three cronjobs
INSERT INTO cronjob_alerts (job_name, alert_type, alert_threshold, notification_channels)
VALUES
    ('ic-score-ttm-financials', 'failure', 3, '["email"]'::JSONB),
    ('ic-score-ticker-sync', 'failure', 2, '["email"]'::JSONB),
    ('ic-score-coverage-monitor', 'failure', 3, '["email"]'::JSONB)
ON CONFLICT DO NOTHING;

-- Add timeout alerts
INSERT INTO cronjob_alerts (job_name, alert_type, alert_threshold, notification_channels)
VALUES
    ('ic-score-ttm-financials', 'timeout', 1, '["email"]'::JSONB),
    ('ic-score-ticker-sync', 'timeout', 1, '["email"]'::JSONB),
    ('ic-score-coverage-monitor', 'timeout', 1, '["email"]'::JSONB)
ON CONFLICT DO NOTHING;
