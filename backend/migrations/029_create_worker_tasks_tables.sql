-- Migration 029: Create worker tasks and updates tables

CREATE TABLE IF NOT EXISTS worker_tasks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title VARCHAR(500) NOT NULL,
    description TEXT DEFAULT '',
    assigned_to UUID REFERENCES users(id) ON DELETE SET NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'in_progress', 'completed', 'failed')),
    priority VARCHAR(10) NOT NULL DEFAULT 'medium'
        CHECK (priority IN ('low', 'medium', 'high', 'urgent')),
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS worker_task_updates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id UUID NOT NULL REFERENCES worker_tasks(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_worker_tasks_assigned_to ON worker_tasks(assigned_to);
CREATE INDEX IF NOT EXISTS idx_worker_tasks_status ON worker_tasks(status);
CREATE INDEX IF NOT EXISTS idx_worker_tasks_created_by ON worker_tasks(created_by);
CREATE INDEX IF NOT EXISTS idx_worker_task_updates_task_id ON worker_task_updates(task_id);

-- Update trigger for worker_tasks
CREATE OR REPLACE FUNCTION update_worker_tasks_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_worker_tasks_updated_at
    BEFORE UPDATE ON worker_tasks
    FOR EACH ROW
    EXECUTE FUNCTION update_worker_tasks_updated_at();
