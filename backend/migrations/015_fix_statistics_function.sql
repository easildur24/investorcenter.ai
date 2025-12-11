-- Fix for get_cronjob_statistics function
-- The ROUND function needs explicit casting to NUMERIC

DROP FUNCTION IF EXISTS get_cronjob_statistics(VARCHAR, INTEGER);

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
        ROUND(es.avg_duration::NUMERIC, 2) as avg_duration_seconds,
        ROUND(es.p50_duration::NUMERIC, 2) as p50_duration_seconds,
        ROUND(es.p95_duration::NUMERIC, 2) as p95_duration_seconds,
        le.last_status::VARCHAR,
        le.last_execution_time
    FROM execution_stats es
    LEFT JOIN latest_execution le ON es.job_name = le.job_name;
END;
$$ LANGUAGE plpgsql;
