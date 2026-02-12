-- Migration 034: Create worker_task_files table for S3 result file metadata
-- ClawdBots upload files directly to S3, then register metadata here

CREATE TABLE IF NOT EXISTS worker_task_files (
    id           BIGSERIAL PRIMARY KEY,
    task_id      UUID NOT NULL REFERENCES worker_tasks(id) ON DELETE CASCADE,
    filename     VARCHAR(500) NOT NULL,
    s3_key       VARCHAR(1000) NOT NULL,
    content_type VARCHAR(255) NOT NULL DEFAULT 'application/octet-stream',
    size_bytes   BIGINT NOT NULL DEFAULT 0,
    uploaded_by  UUID REFERENCES users(id),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Get all files for a task (admin viewing task output)
CREATE INDEX idx_wtf_task_id ON worker_task_files(task_id, created_at DESC);

-- Deduplicate by S3 key
CREATE UNIQUE INDEX idx_wtf_s3_key ON worker_task_files(s3_key);
