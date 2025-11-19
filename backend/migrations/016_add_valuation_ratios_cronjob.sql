-- Add IC Score Valuation Ratios Cronjob to monitoring

INSERT INTO cronjob_schedules (job_name, job_category, description, schedule_cron, schedule_description, expected_duration_seconds, timeout_seconds)
VALUES
    ('ic-score-valuation-ratios', 'ic_score_pipeline', 'Calculates valuation ratios (P/E, P/B, P/S) using TTM financials and current prices', '30 23 * * 1-5', 'Daily at 11:30 PM UTC (6:30 PM ET) on weekdays', 600, 3600)
ON CONFLICT (job_name) DO UPDATE SET
    job_category = EXCLUDED.job_category,
    description = EXCLUDED.description,
    schedule_cron = EXCLUDED.schedule_cron,
    schedule_description = EXCLUDED.schedule_description,
    expected_duration_seconds = EXCLUDED.expected_duration_seconds,
    timeout_seconds = EXCLUDED.timeout_seconds,
    updated_at = CURRENT_TIMESTAMP;

-- Add default failure alert for valuation ratios cronjob
INSERT INTO cronjob_alerts (job_name, alert_type, alert_threshold, notification_channels)
VALUES
    ('ic-score-valuation-ratios', 'failure', 3, '["email"]'::JSONB)
ON CONFLICT DO NOTHING;

-- Add timeout alert for valuation ratios cronjob
INSERT INTO cronjob_alerts (job_name, alert_type, alert_threshold, notification_channels)
VALUES
    ('ic-score-valuation-ratios', 'timeout', 1, '["email"]'::JSONB)
ON CONFLICT DO NOTHING;
