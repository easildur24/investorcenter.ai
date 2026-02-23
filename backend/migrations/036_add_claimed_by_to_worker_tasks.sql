-- Add claimed_by column to worker_tasks for stateless task queue
-- Replaces assigned_to (pre-assignment) with claimed_by (pull-based claiming)
ALTER TABLE worker_tasks ADD COLUMN IF NOT EXISTS claimed_by UUID REFERENCES users(id) ON DELETE SET NULL;

-- Index for finding tasks claimed by a specific user
CREATE INDEX IF NOT EXISTS idx_worker_tasks_claimed_by ON worker_tasks(claimed_by);

-- Composite index for the claim-next query: pending tasks ordered by priority + created_at
CREATE INDEX IF NOT EXISTS idx_worker_tasks_pending_queue ON worker_tasks(status, priority, created_at ASC)
    WHERE status = 'pending';
