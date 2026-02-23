-- Migration 037: Redesign task schema
-- - task_types: replace inline SOP with skill_path pointer
-- - worker_tasks: remove title/description/assigned_to/result, rename to tasks

-- ============================================================
-- 1. task_types: add skill_path, drop sop
-- ============================================================
ALTER TABLE task_types ADD COLUMN IF NOT EXISTS skill_path VARCHAR(200);

-- Seed known skill_path mappings
UPDATE task_types SET skill_path = 'scrape-ycharts-keystats' WHERE name = 'crawl_sa_test';
UPDATE task_types SET skill_path = 'data-ingestion' WHERE name = 'reddit_crawl';

ALTER TABLE task_types DROP COLUMN IF EXISTS sop;

-- ============================================================
-- 2. worker_tasks: make task_type_id NOT NULL
-- ============================================================
-- Backfill orphan tasks with the 'custom' task type
UPDATE worker_tasks
SET task_type_id = (SELECT id FROM task_types WHERE name = 'custom')
WHERE task_type_id IS NULL;

ALTER TABLE worker_tasks ALTER COLUMN task_type_id SET NOT NULL;

-- ============================================================
-- 3. worker_tasks: drop dead columns
-- ============================================================
ALTER TABLE worker_tasks
  DROP COLUMN IF EXISTS title,
  DROP COLUMN IF EXISTS description,
  DROP COLUMN IF EXISTS assigned_to,
  DROP COLUMN IF EXISTS result;

-- Drop the now-orphaned assigned_to index
DROP INDEX IF EXISTS idx_worker_tasks_assigned_to;

-- ============================================================
-- 4. Rename worker_tasks -> tasks
-- ============================================================
ALTER TABLE worker_tasks RENAME TO tasks;

-- Rename indexes
ALTER INDEX IF EXISTS worker_tasks_pkey RENAME TO tasks_pkey;
ALTER INDEX IF EXISTS idx_worker_tasks_status RENAME TO idx_tasks_status;
ALTER INDEX IF EXISTS idx_worker_tasks_created_by RENAME TO idx_tasks_created_by;
ALTER INDEX IF EXISTS idx_worker_tasks_task_type RENAME TO idx_tasks_task_type;
ALTER INDEX IF EXISTS idx_worker_tasks_claimed_by RENAME TO idx_tasks_claimed_by;
ALTER INDEX IF EXISTS idx_worker_tasks_pending_queue RENAME TO idx_tasks_pending_queue;

-- Rename trigger
ALTER TRIGGER trigger_update_worker_tasks_updated_at ON tasks
  RENAME TO trigger_update_tasks_updated_at;
