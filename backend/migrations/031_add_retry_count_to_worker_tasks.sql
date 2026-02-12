-- Migration 031: Add retry_count to worker_tasks for automatic retry support

ALTER TABLE worker_tasks ADD COLUMN IF NOT EXISTS retry_count INTEGER NOT NULL DEFAULT 0;
