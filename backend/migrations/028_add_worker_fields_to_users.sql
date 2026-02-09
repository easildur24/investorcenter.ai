-- Migration 028: Add worker fields to users table
ALTER TABLE users ADD COLUMN IF NOT EXISTS is_worker BOOLEAN DEFAULT FALSE;
ALTER TABLE users ADD COLUMN IF NOT EXISTS last_activity_at TIMESTAMP;

CREATE INDEX IF NOT EXISTS idx_users_is_worker ON users(is_worker) WHERE is_worker = TRUE;
