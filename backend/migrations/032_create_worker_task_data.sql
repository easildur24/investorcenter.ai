-- Migration 032: Create worker_task_data table for generic worker-collected data
-- Purpose: Store individual data items collected by workers during task execution

CREATE TABLE IF NOT EXISTS worker_task_data (
    id           BIGSERIAL PRIMARY KEY,
    task_id      UUID NOT NULL REFERENCES worker_tasks(id) ON DELETE CASCADE,
    data_type    VARCHAR(100) NOT NULL,
    ticker       VARCHAR(20),
    external_id  VARCHAR(255),
    data         JSONB NOT NULL,
    collected_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Get all data for a task (admin viewing task output)
CREATE INDEX idx_wtd_task_id ON worker_task_data(task_id, created_at DESC);

-- Get all data of a type for a ticker (future analytics)
CREATE INDEX idx_wtd_type_ticker ON worker_task_data(data_type, ticker, collected_at DESC)
    WHERE ticker IS NOT NULL;

-- Deduplicate by external ID within a data type
CREATE UNIQUE INDEX idx_wtd_dedup ON worker_task_data(data_type, external_id)
    WHERE external_id IS NOT NULL;

-- Time-range queries for cleanup/retention
CREATE INDEX idx_wtd_created_at ON worker_task_data(created_at);
