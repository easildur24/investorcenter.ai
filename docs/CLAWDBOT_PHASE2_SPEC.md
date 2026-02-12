# ClawdBot Phase 2: Pull-Based Task Queue

> **Version**: 3.0
> **Date**: February 2026
> **Status**: Design
> **Depends on**: Phase 1 (task-service, task_types, SOPs, worker API, admin UI)

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [What Phase 1 Built](#2-what-phase-1-built)
3. [Phase 2 Design](#3-phase-2-design)
4. [Database Changes](#4-database-changes)
5. [API Changes](#5-api-changes)
6. [Atomic Task Claiming](#6-atomic-task-claiming)
7. [Background Goroutines](#7-background-goroutines)
8. [Retry System](#8-retry-system)
9. [Telegram Notifications](#9-telegram-notifications)
10. [UI Changes](#10-ui-changes)
11. [Implementation Plan](#11-implementation-plan)
12. [Future (Phase 3)](#12-future-phase-3)

---

## 1. Executive Summary

### What is Phase 2?

Phase 1 built a task queue where **humans create and assign tasks** to specific ClawdBot workers via the admin UI. Phase 2 removes the manual assignment step — ClawdBots **pull tasks from a shared queue** and PostgreSQL handles concurrency. No Task Manager, no orchestrator, no extra process.

### Phase 2 Scope

- **Pull-based queue** — ClawdBots claim unassigned tasks atomically via `POST /worker/tasks/claim` (PostgreSQL `FOR UPDATE SKIP LOCKED`)
- **Retry on failure** — background goroutine in task-service creates retry tasks, with optional Claude API call for intelligent retry descriptions
- **Stuck task detection** — background goroutine marks tasks failed when the assigned worker goes offline
- **Telegram notifications** — background goroutine sends notifications for completions, failures, retries
- **Humans still create tasks** — task creation stays manual (Phase 3 adds autonomous scheduling)

### Key Insight: PostgreSQL Is the Queue

No external message broker, no orchestrator process, no Task Manager. PostgreSQL's `SELECT ... FOR UPDATE SKIP LOCKED` provides atomic claiming — if two ClawdBots try to claim simultaneously, each gets a different task. Workers naturally self-balance: a busy bot won't claim more work, an idle one will.

### What Phase 2 Does NOT Do (Yet)

- Does NOT create tasks autonomously (Phase 3)
- Does NOT accept Telegram commands (Phase 3)
- Does NOT auto-scale workers

---

## 2. What Phase 1 Built

Quick recap of the foundation Phase 2 builds on:

| Component | What |
|-----------|------|
| **task-service** | Standalone Go microservice (port 8001), extracted from main backend. Main backend proxies `/api/v1/admin/workers/*` and `/api/v1/worker/*` to it. |
| `task_types` table | Categories of work with SOPs and param_schema |
| `worker_tasks` table | Tasks with task_type_id, params, result, retry_count, started_at, completed_at |
| `worker_task_updates` table | Comment/log thread per task |
| Admin UI | Create task types, create tasks (pick type + params + assignee), view results |
| Worker API | ClawdBots fetch tasks (with SOP joined), update status, post results |
| Worker Data (S3) | Workers upload collected data to S3 (`claw-treasure` bucket) at `worker-data/{task_id}/{data_type}/` |
| Heartbeat | Workers POST /worker/heartbeat every 2 min; online = active within 5 min |
| Auth | Workers are users with `is_worker=true`, JWT auth, same as regular users |

### Phase 1 Flow (push-based)

```
Human creates task → Human assigns to specific worker → Worker polls for assigned tasks → Executes → Reports
```

### Phase 2 Flow (pull-based)

```
Human creates task (no assignee) → Task sits in queue
                                         ↑
ClawdBot claims next task ───────────────┘ (atomic via PostgreSQL)
ClawdBot executes task
ClawdBot reports result ──────────────────→ task-service goroutine sends Telegram notification

If task fails ────────────────────────────→ task-service goroutine creates retry → back in queue
If worker goes offline ───────────────────→ task-service goroutine marks failed → back in queue
```

---

## 3. Phase 2 Design

### Architecture

```
                           task-service (Go/Gin, :8001)
                          ┌──────────────────────────────┐
                          │                               │
Human ─── create task ──► │  worker_tasks table (PG)      │
  (admin UI)              │  ┌─────────────────────────┐  │
                          │  │ pending, unassigned      │  │  ◄── tasks wait here
                          │  │ pending, unassigned      │  │
                          │  │ pending, unassigned      │  │
                          │  └─────────────────────────┘  │
                          │                               │
                          │  POST /worker/tasks/claim      │  ◄── atomic, race-free
                          │  (FOR UPDATE SKIP LOCKED)      │
                          │                               │
                          │  Background goroutines:       │
                          │   • stuck detector (60s)      │
                          │   • retry creator             │
                          │   • telegram notifier         │
                          └──────┬──────────┬─────────────┘
                                 │          │
                   ┌─────────────┘          └──────────────┐
                   ▼                                       ▼
             ClawdBot A                              ClawdBot B
             ┌───────────┐                           ┌───────────┐
             │ 1. claim   │                           │ 1. claim   │
             │ 2. execute │                           │ 2. execute │
             │ 3. report  │                           │ 3. report  │
             │ 4. repeat  │                           │ 4. repeat  │
             └───────────┘                           └───────────┘
```

> **Note**: The main Go backend (port 8080) reverse-proxies all `/api/v1/admin/workers/*` and `/api/v1/worker/*` routes to the task-service via `backend/services/task_service_proxy.go`. This proxy is transparent — all clients use `/api/v1/...` URLs.

### Why No Task Manager?

The original spec proposed a Claude Code session as an orchestrator. This was eliminated because:

| Concern | Pull-based queue | Task Manager |
|---------|-----------------|--------------|
| Assignment | Workers self-assign (claim). Natural load balancing. | Central process must track capacity, pick workers. |
| Concurrency | PostgreSQL `FOR UPDATE SKIP LOCKED` — battle-tested. | LLM deciding who gets what — slow and expensive. |
| Reliability | Goroutines in task-service (already deployed on K8s). | Extra process on a persistent VM to babysit. |
| Cost | Zero LLM cost for orchestration. Claude API only for retry intelligence. | LLM tokens every 30 seconds, 24/7. |
| Complexity | One new endpoint + 3 goroutines. | Entire CLAUDE.md, env vars, separate machine. |

The only value the Task Manager provided was **retry intelligence** (reading failure reasons and adapting). Phase 2 preserves this with a targeted Claude API call in the retry goroutine — LLM intelligence where it matters, deterministic code everywhere else.

### ClawdBot Work Loop

Each ClawdBot runs this loop (defined in its SOP or bootstrap CLAUDE.md):

```
1. Authenticate (POST /auth/login)
2. Send heartbeat (POST /worker/heartbeat)
3. Claim next task (POST /worker/tasks/claim)
4. If no task available → wait 30s → go to 2
5. Read task's SOP + params
6. Execute the task
7. Post updates along the way (POST /worker/tasks/:id/updates)
8. Post result (POST /worker/tasks/:id/result)
9. Upload collected data to S3 (POST /worker/tasks/:id/data)
10. Mark task completed (PUT /worker/tasks/:id/status)
11. Go to 2
```

If the ClawdBot crashes mid-task, the stuck detector goroutine will eventually mark the task as failed and return it to the queue.

---

## 4. Database Changes

The task-service owns its schema. These changes are applied as ALTER statements against the shared `investorcenter_db` database.

> **Note**: `retry_count` already exists on `worker_tasks` from Phase 1.

### Schema Changes

```sql
-- Add max_retries to task_types
ALTER TABLE task_types
    ADD COLUMN IF NOT EXISTS max_retries INTEGER NOT NULL DEFAULT 3;

-- Add retry lineage to worker_tasks (retry_count already exists)
ALTER TABLE worker_tasks
    ADD COLUMN IF NOT EXISTS parent_task_id UUID REFERENCES worker_tasks(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_worker_tasks_parent ON worker_tasks(parent_task_id);

-- Index for claim query performance (unassigned pending tasks)
CREATE INDEX IF NOT EXISTS idx_worker_tasks_claim
    ON worker_tasks(status, assigned_to)
    WHERE status = 'pending' AND assigned_to IS NULL;
```

### Schema After Changes

**task_types** (extended):
| Column | Type | Notes |
|--------|------|-------|
| ... | ... | (all Phase 1 columns) |
| **max_retries** | **INTEGER** | **NEW** — max retry attempts (default 3) |

**worker_tasks** (extended):
| Column | Type | Notes |
|--------|------|-------|
| ... | ... | (all Phase 1 columns, including retry_count) |
| retry_count | INTEGER | Already exists from Phase 1 (0 = original) |
| **parent_task_id** | **UUID FK** | **NEW** — if this is a retry, points to the failed task |

### What's NOT Needed

- **No `task_assignments` table** — workers claim directly, assignment is implicit in `assigned_to`
- **No worker capacity endpoint** — workers self-balance by pulling when ready
- **No assignment log** — the `worker_task_updates` table already provides an audit trail

---

## 5. API Changes

All endpoints live in the **task-service** (port 8001). The main Go backend proxies them under `/api/v1/...`. Implementation goes in `task-service/handlers/`.

### New Endpoints

#### Claim Next Task (Worker)

```
POST /api/v1/worker/tasks/claim
(task-service route: POST /worker/tasks/claim)
Authorization: Bearer <jwt> (worker)

Response 200 (task claimed):
{
  "success": true,
  "data": {
    "id": "abc-123",
    "title": "Reddit Crawl for NVDA",
    "status": "in_progress",
    "assigned_to": "worker-uuid",
    "started_at": "2026-02-10T14:30:00Z",
    "task_type": { "id": 1, "name": "reddit_crawl", "label": "Reddit Crawl", "sop": "...", "max_retries": 3 },
    "params": { "ticker": "NVDA", "subreddits": ["wsb"], "days": 30 },
    "retry_count": 0,
    "parent_task_id": null,
    ...
  }
}

Response 200 (no tasks available):
{
  "success": true,
  "data": null
}
```

This is the core new endpoint. See [Section 6](#6-atomic-task-claiming) for implementation details.

### Modified Endpoints

#### Create Task (Admin) — assigned_to now optional

```
POST /api/v1/admin/workers/tasks

{
  "title": "Reddit Crawl for NVDA",
  "task_type_id": 1,
  "params": { "ticker": "NVDA", "subreddits": ["wsb"], "days": 30 },
  "priority": "high"
  // assigned_to is OPTIONAL — omit for queue-based claiming
  // If provided, task is pre-assigned to a specific worker (Phase 1 behavior)
}
```

#### Task Response — new fields

```json
{
  "id": "abc-123",
  "title": "Reddit Crawl for NVDA",
  "retry_count": 1,
  "parent_task_id": "def-456",
  "task_type": { "id": 1, "name": "reddit_crawl", "label": "Reddit Crawl", "max_retries": 3 },
  ...
}
```

#### List Tasks (Admin) — new filter

```
GET /api/v1/admin/workers/tasks?unassigned=true
```

Returns tasks where `assigned_to IS NULL` and `status = 'pending'` (queue depth).

---

## 6. Atomic Task Claiming

### The Claim Query

```sql
UPDATE worker_tasks
SET assigned_to = $1,           -- worker_id
    status = 'in_progress',
    started_at = NOW()
WHERE id = (
    SELECT id FROM worker_tasks
    WHERE assigned_to IS NULL
      AND status = 'pending'
    ORDER BY
        CASE priority
            WHEN 'urgent' THEN 0
            WHEN 'high'   THEN 1
            WHEN 'medium' THEN 2
            WHEN 'low'    THEN 3
        END,
        created_at ASC
    LIMIT 1
    FOR UPDATE SKIP LOCKED
)
RETURNING *;
```

### How It Works

1. `FOR UPDATE` — locks the selected row so no other transaction can claim it
2. `SKIP LOCKED` — if a row is already locked by another concurrent claim, skip it and take the next one
3. The `UPDATE ... WHERE id = (SELECT ...)` is atomic — claim and status change happen in one statement
4. Priority ordering ensures urgent/high tasks are claimed first
5. `created_at ASC` ensures FIFO within the same priority level

### Why This Is Sufficient

| Concern | Answer |
|---------|--------|
| Two bots claim at once? | Each gets a different task (`SKIP LOCKED`) |
| 10 bots, 1 task? | First one gets it, other 9 get `null` (no rows returned) |
| Task priority? | ORDER BY priority, then FIFO within priority |
| Load balancing? | Natural — idle bots claim, busy bots don't |
| Bot crashes mid-task? | Stuck detector goroutine reclaims it (see Section 7) |

### Handler Implementation

```go
// In task-service/handlers/worker_api.go
func WorkerClaimTask(c *gin.Context) {
    workerID := auth.GetUserIDFromContext(c)

    var task WorkerTask
    err := db.QueryRowx(`
        UPDATE worker_tasks
        SET assigned_to = $1, status = 'in_progress', started_at = NOW()
        WHERE id = (
            SELECT id FROM worker_tasks
            WHERE assigned_to IS NULL AND status = 'pending'
            ORDER BY
                CASE priority WHEN 'urgent' THEN 0 WHEN 'high' THEN 1 WHEN 'medium' THEN 2 WHEN 'low' THEN 3 END,
                created_at ASC
            LIMIT 1
            FOR UPDATE SKIP LOCKED
        )
        RETURNING *;
    `, workerID).StructScan(&task)

    if err == sql.ErrNoRows {
        c.JSON(200, gin.H{"success": true, "data": nil})
        return
    }
    if err != nil {
        c.JSON(500, gin.H{"success": false, "error": err.Error()})
        return
    }

    // Join task_type details for the response
    // ... (same pattern as existing WorkerGetTask)

    c.JSON(200, gin.H{"success": true, "data": task})
}
```

---

## 7. Background Goroutines

Three goroutines run inside the task-service process. They start in `main.go` alongside the HTTP server.

### 7a. Stuck Task Detector

**Runs every**: 60 seconds

**Purpose**: Detect tasks stuck `in_progress` with an offline worker and return them to the queue.

```
A task is STUCK if ALL of these are true:
  1. status = 'in_progress'
  2. assigned worker is OFFLINE (last_activity_at > 10 minutes ago)
  3. task has been in_progress for > 15 minutes (started_at > 15 min ago)
```

**Resolution**:

```sql
-- Find stuck tasks (worker offline + task stale)
UPDATE worker_tasks wt
SET status = 'failed',
    completed_at = NOW()
FROM users u
WHERE wt.assigned_to = u.id
  AND wt.status = 'in_progress'
  AND wt.started_at < NOW() - INTERVAL '15 minutes'
  AND u.last_activity_at < NOW() - INTERVAL '10 minutes';
```

The stuck detector also inserts a `worker_task_updates` entry: "Worker {name} went offline. Task marked as failed for retry."

Failed tasks are picked up by the retry creator (below).

### 7b. Retry Creator

**Runs every**: 30 seconds

**Purpose**: Create retry tasks for failed tasks that haven't exhausted their retry limit.

**Logic**:

```sql
-- Find failed tasks eligible for retry
SELECT wt.*, tt.max_retries
FROM worker_tasks wt
JOIN task_types tt ON wt.task_type_id = tt.id
WHERE wt.status = 'failed'
  AND wt.retry_count < tt.max_retries
  AND NOT EXISTS (
      SELECT 1 FROM worker_tasks child
      WHERE child.parent_task_id = wt.id
  );
```

For each eligible task:
1. **(Optional) Call Claude API** to analyze the failure and generate adjusted instructions
2. Create a new task:
   - Same title with " (retry N)" appended
   - Same task_type_id and params
   - `parent_task_id` = the failed task's ID
   - `retry_count` = failed task's retry_count + 1
   - `assigned_to = NULL` — goes back into the claim queue
   - Description = original + failure context from Claude API (or templated fallback)
3. Insert a `worker_task_updates` entry on the original: "Retry #{N} created"
4. Queue a Telegram notification

### Retry Intelligence (Optional Claude API Call)

For simple failures (worker offline, timeout), a templated retry description is sufficient:

```
PREVIOUS ATTEMPT FAILED: Worker went offline during execution.
This is retry 1 of 3. Please complete the task.
```

For complex failures (rate limits, data issues), the retry creator can make a **single Claude API call**:

```go
prompt := fmt.Sprintf(
    "A task failed with this error:\n%s\n\nGenerate a 1-2 sentence instruction "+
    "for the retry, explaining what to do differently. If the error looks permanent "+
    "(invalid data, missing API), respond with SKIP.",
    failureReason,
)
// Call Claude API, get response
// If "SKIP" → don't create retry, send Telegram alert instead
// Otherwise → use response as retry description
```

This is the only place LLM is used — a single API call per failure, not a continuous orchestration loop.

### 7c. Telegram Notifier

**Runs**: event-driven (triggered by stuck detector and retry creator), with 5-second batching

**Purpose**: Send notifications to a Telegram chat for key events.

**Notification Types**:

| Event | Message Format |
|-------|---------------|
| Task completed | `*{title}* completed by {worker}. {result_summary}` |
| Task failed | `*{title}* failed. Reason: {reason}. Retry: {yes/no}` |
| All retries exhausted | `*{title}* failed after {N} attempts. Needs human attention.` |
| Task stuck | `*{title}* stuck — {worker} offline. Returning to queue.` |

**Implementation**: Uses Telegram Bot API via HTTP POST from Go (no external dependency).

```go
func sendTelegram(botToken, chatID, message string) error {
    url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", botToken)
    body := map[string]string{
        "chat_id":    chatID,
        "text":       message,
        "parse_mode": "Markdown",
    }
    // ... standard HTTP POST
}
```

**Config** (env vars on task-service):

```bash
TELEGRAM_BOT_TOKEN=<bot-token>
TELEGRAM_CHAT_ID=<chat-id>
TELEGRAM_ENABLED=true    # easy kill switch
```

### Goroutine Lifecycle

```go
// In task-service/main.go
func main() {
    db := database.Connect()
    router := setupRoutes(db)

    // Start background goroutines
    go orchestrator.RunStuckDetector(db, 60*time.Second)
    go orchestrator.RunRetryCreator(db, 30*time.Second)
    go orchestrator.RunTelegramNotifier(db)

    router.Run(":8001")
}
```

All goroutines are gracefully shut down on SIGTERM (K8s pod termination).

---

## 8. Retry System

### How Retries Work

```
Original Task (retry_count=0)
    |
    | fails (worker crash, error, stuck detection)
    v
Retry creator goroutine picks it up
    |
    v
Creates new task:
  - title: "Reddit Crawl for NVDA (retry 1)"
  - parent_task_id: original task ID
  - retry_count: 1
  - description: original + failure context
  - assigned_to: NULL → goes back in queue
    |
    | ClawdBot claims and executes
    | if retry 1 also fails...
    v
Creates retry 2 (retry_count=2, parent=retry 1)
    |
    | if retry 2 also fails...
    v
Creates retry 3 (retry_count=3, parent=retry 2)
    |
    | if retry 3 fails
    v
max_retries (3) reached — Telegram alert, no more retries
"Reddit Crawl for NVDA failed after 3 attempts. Needs human attention."
```

### Retry Fields

| Field | Where | Description |
|-------|-------|-------------|
| `max_retries` | `task_types` | Max retries for this type of task (default 3) |
| `retry_count` | `worker_tasks` | Which retry attempt this is (0 = original) |
| `parent_task_id` | `worker_tasks` | Link to the task this retries (NULL for originals) |

### Retry Intelligence Examples

**Rate limit failure**:
- Failure: "Arctic Shift API returned 429 after 50 calls"
- Claude API response: "Use longer delays (3-5s) between requests and smaller batch sizes."
- Retry description: "PREVIOUS ATTEMPT FAILED: Rate limited. Please use longer delays between requests (3-5 seconds) and smaller batch sizes."

**Permanent failure (no retry)**:
- Failure: "No posts found for ticker XYZZ in any subreddit"
- Claude API response: "SKIP"
- Action: No retry created. Telegram: "No data found for XYZZ — is this a valid ticker?"

**Worker crash (templated, no Claude API needed)**:
- Failure: "Worker went offline during execution"
- Retry description: "PREVIOUS ATTEMPT FAILED: Worker went offline. This is retry 1 of 3."

---

## 9. Telegram Notifications

### Setup

1. Create a Telegram bot via @BotFather
2. Get the bot token
3. Create a Telegram group/channel for notifications
4. Add the bot to the group
5. Get the chat ID
6. Set env vars on task-service deployment

### Configuration

```bash
# In task-service K8s deployment or .env
TELEGRAM_BOT_TOKEN=<telegram-bot-token>
TELEGRAM_CHAT_ID=<telegram-chat-id>
TELEGRAM_ENABLED=true
```

### Rate Limiting

The notifier goroutine batches messages — if 5 tasks complete within one check cycle, it sends a single summary message instead of 5 individual messages.

---

## 10. UI Changes

### Task Detail — Retry Chain

When viewing a task that's part of a retry chain, show the lineage:

```
Reddit Crawl for NVDA (retry 2)
├─ Original: Reddit Crawl for NVDA (failed)
├─ Retry 1: Reddit Crawl for NVDA (retry 1) (failed)
└─ Retry 2: Reddit Crawl for NVDA (retry 2) (in_progress) ← current
```

Display:
- `retry_count` badge on task cards (e.g., "Retry 2/3")
- Link to parent task
- Link to child retries (if any)

### Task List — New Filters

Add filters:
- `unassigned` — show only tasks in the queue (pending + no assignee)
- `retries` — show only retry tasks (retry_count > 0)

### Task List — Queue Depth Indicator

Show the number of unassigned pending tasks: "Queue: 3 tasks waiting"

### Workers Tab — Activity View

Show per-worker info:
- Online/offline status
- Current task (if any)
- Completed today count

---

## 11. Implementation Plan

### Phase 2a: Task Service Changes (Week 1)

All work happens in `task-service/`. The main Go backend requires no changes (proxy already forwards all routes).

| Task | Effort | Description |
|------|--------|-------------|
| Schema changes | S | Add `max_retries` to task_types, `parent_task_id` to worker_tasks, add claim index |
| `POST /worker/tasks/claim` | S | Atomic claim endpoint with `FOR UPDATE SKIP LOCKED` in `task-service/handlers/worker_api.go` |
| Unassigned filter | S | Add `?unassigned=true` filter to `ListTasks` |
| Extend task response | S | Include `parent_task_id`, `max_retries` in task JSON responses |
| Extend CreateTask | S | Accept `parent_task_id` in create request body, make `assigned_to` optional |
| Register routes | S | Add new route in `task-service/main.go` |
| Tests for claim | M | Unit tests for claim handler, concurrent claim tests |

### Phase 2b: Background Goroutines (Week 1-2)

| Task | Effort | Description |
|------|--------|-------------|
| Stuck detector | M | Goroutine in `task-service/orchestrator/stuck.go` — 60s loop, marks stuck tasks failed |
| Retry creator | M | Goroutine in `task-service/orchestrator/retry.go` — creates retry tasks, optional Claude API call |
| Telegram notifier | S | Goroutine in `task-service/orchestrator/telegram.go` — sends notifications |
| Goroutine lifecycle | S | Graceful startup/shutdown in `main.go` |
| Tests | M | Unit tests for stuck detection logic, retry creation, notification formatting |

### Phase 2c: Frontend (Week 2)

| Task | Effort | Description |
|------|--------|-------------|
| Retry chain in task detail | M | Show parent/child links, retry badge |
| Queue depth indicator | S | Show unassigned pending count |
| New task list filters | S | Unassigned, retries filters |
| Worker activity view | S | Current task, completed today |
| Frontend API client updates | S | Add new types/endpoints to `lib/api.ts` |

**S** = Small (< 1 day), **M** = Medium (1-3 days)

**Total estimate: ~2 weeks** (down from ~3 weeks — no Task Manager to build/test)

---

## 12. Future (Phase 3)

Phase 3 adds **autonomous task creation**:

- Background goroutine or K8s CronJob scans for tickers with stale data → creates crawl tasks automatically
- Priority scoring based on watchlist popularity, data staleness, trending tickers
- Telegram commands (`/status`, `/add reddit NVDA`, `/pause`) via a webhook endpoint in task-service
- Daily digest via Telegram
- Budget/rate limit awareness (track LLM token usage per task)

---

## Appendix A: Service Architecture Summary

### Services Involved

| Service | Port | Role in Phase 2 |
|---------|------|-----------------|
| **Frontend** (Next.js) | 3000 | Admin UI for task management |
| **Backend** (Go/Gin) | 8080 | Reverse-proxies `/api/v1/admin/workers/*` and `/api/v1/worker/*` to task-service |
| **Task Service** (Go/Gin) | 8001 | Queue, claim endpoint, CRUD, background goroutines (stuck/retry/notify) |
| **ClawdBot Workers** (Claude Code) | N/A | Pull tasks from queue, execute, report results |

### How ClawdBots Interact with the Queue

| Step | Endpoint | Description |
|------|----------|-------------|
| Auth | `POST /auth/login` | Get JWT token |
| Heartbeat | `POST /worker/heartbeat` | Stay online (every 2 min) |
| Claim | `POST /worker/tasks/claim` | Atomic claim of next available task |
| Updates | `POST /worker/tasks/:id/updates` | Post progress logs |
| Result | `POST /worker/tasks/:id/result` | Post structured result |
| Data | `POST /worker/tasks/:id/data` | Upload collected data to S3 |
| Complete | `PUT /worker/tasks/:id/status` | Mark task completed/failed |

## Appendix B: Example Scenario

```
Time    Event
-----   -----
10:00   Human creates "Reddit Crawl for NVDA" in admin UI (no assignee)
10:00   Task status: pending, assigned_to: NULL (in queue)

10:01   ClawdBot "Genesis" calls POST /worker/tasks/claim
10:01   PostgreSQL atomically assigns task to Genesis, sets status=in_progress
10:01   Genesis receives task with SOP + params in response

10:05   Genesis posts update: "Collected 300 posts from r/wsb..."

10:08   Genesis's machine loses network. Heartbeat stops.

10:18   Stuck detector goroutine runs: Genesis offline 10 min, task in_progress 17 min
10:18   Marks task as failed, inserts update: "Worker Genesis went offline"
10:18   → Telegram: "*Reddit Crawl for NVDA* stuck — Genesis offline. Returning to queue."

10:18   Retry creator goroutine runs: task failed, retry_count=0, max_retries=3
10:18   Creates retry: "Reddit Crawl for NVDA (retry 1)", retry_count=1, assigned_to=NULL
10:18   → Telegram: "Retry 1/3: *Reddit Crawl for NVDA* back in queue."

10:19   ClawdBot "Nexus" calls POST /worker/tasks/claim
10:19   Gets the retry task (it's next in queue)
10:19   Nexus reads description: "Previous attempt: worker went offline mid-execution"

10:25   Nexus completes successfully, posts result
10:25   → Telegram: "*Reddit Crawl for NVDA (retry 1)* completed by Nexus. 847 posts."
```

## Appendix C: Why Not a Task Manager?

The original Phase 2 spec proposed a Task Manager (Claude Code session) as a central orchestrator. This was replaced with the pull-based model because:

1. **95% of orchestration is deterministic** — assignment, stuck detection, notifications don't need an LLM
2. **PostgreSQL provides atomic claiming for free** — no coordinator needed for work distribution
3. **Workers self-balance naturally** — idle bots claim, busy bots don't
4. **LLM intelligence is preserved** — targeted Claude API call for retry decisions only
5. **Operational simplicity** — no extra process to run, monitor, and restart
6. **Cost** — LLM calls only on failures (rare) vs. every 30 seconds (continuous)
7. **Reliability** — Go goroutines don't crash from context overflow or hallucinate API paths
