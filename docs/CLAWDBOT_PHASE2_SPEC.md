# ClawdBot Phase 2: Task Manager Agent

> **Version**: 1.0
> **Date**: February 2026
> **Status**: Design
> **Depends on**: Phase 1 (task_types, SOPs, worker API, admin UI)

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [What Phase 1 Built](#2-what-phase-1-built)
3. [Phase 2 Design](#3-phase-2-design)
4. [Database Changes](#4-database-changes)
5. [API Changes](#5-api-changes)
6. [Task Manager Design](#6-task-manager-design)
7. [Assignment Algorithm](#7-assignment-algorithm)
8. [Retry System](#8-retry-system)
9. [Telegram Notifications](#9-telegram-notifications)
10. [Stuck Task Detection](#10-stuck-task-detection)
11. [UI Changes](#11-ui-changes)
12. [Implementation Plan](#12-implementation-plan)
13. [Future (Phase 3)](#13-future-phase-3)

---

## 1. Executive Summary

### What is Phase 2?

Phase 1 built a task queue where **humans create and assign tasks** to ClawdBot workers via the admin UI. Phase 2 adds a **Task Manager** — itself a ClawdBot — that automates the middle step: assigning tasks to available workers, retrying failures, detecting stuck tasks, and sending Telegram notifications.

### Phase 2 Scope

- **Task Manager = a ClawdBot** — a Claude Code session with its own CLAUDE.md, not backend code
- **Auto-assignment** — unassigned tasks get routed to available, online workers
- **Retry on failure** — failed tasks are retried up to N times (configurable per task type, default 3)
- **Stuck task detection** — tasks stuck `in_progress` with an offline worker get reassigned
- **Telegram notifications** — one-way notifications to a Telegram chat for completions, failures, retries
- **Humans still create tasks** — task creation stays manual (Phase 3 adds autonomous scheduling)

### What the Task Manager Does NOT Do (Yet)

- Does NOT create tasks (Phase 3)
- Does NOT accept Telegram commands (Phase 3)
- Does NOT optimize or rewrite prompts — SOPs are sufficient for now
- Does NOT auto-scale workers

---

## 2. What Phase 1 Built

Quick recap of the foundation Phase 2 builds on:

| Component | What |
|-----------|------|
| `task_types` table | Categories of work with SOPs and param_schema |
| `worker_tasks` table | Tasks with task_type_id, params, result, started_at, completed_at |
| Admin UI | Create task types, create tasks (pick type + params + assignee), view results |
| Worker API | ClawdBots fetch tasks (with SOP joined), update status, post results |
| Heartbeat | Workers POST /worker/heartbeat every 2 min; online = active within 5 min |
| Auth | Workers are users with `is_worker=true`, JWT auth, same as regular users |

### Current Flow (Phase 1)

```
Human creates task in UI → Human assigns to a worker → Worker polls → Worker executes → Worker reports
```

### Phase 2 Flow

```
Human creates task in UI → Task Manager assigns → Worker polls → Worker executes → Worker reports
                                    ↓                                       ↓
                           (if stuck/failed)                      Task Manager notifies
                           Task Manager retries                   via Telegram
```

---

## 3. Phase 2 Design

### The Task Manager Is a ClawdBot

The Task Manager is a Claude Code session running on a persistent machine (a cloud VM or always-on Mac). It is registered as a worker user with **both `is_worker=true` and `is_admin=true`**, giving it access to:

- **Admin endpoints** — to list all workers, list all tasks, assign tasks, create retry tasks
- **Worker endpoints** — for heartbeat (so it shows as online in the admin UI)

It does NOT execute data tasks (reddit crawls, sentiment analysis, etc.). Its only job is orchestration.

### Why a ClawdBot, Not Backend Code?

| Aspect | ClawdBot Task Manager | Backend Service |
|--------|----------------------|-----------------|
| Deploy | Drop a CLAUDE.md, run Claude Code | Write Go code, deploy to K8s |
| Change behavior | Edit the CLAUDE.md or SOP | Write code, PR, deploy |
| Retry intelligence | Claude analyzes failure reason, adjusts instructions | Dumb retry (same params) |
| Flexibility | Handles edge cases naturally (Claude is smart) | Must code every case |
| Observability | Posts updates to task threads like any worker | Need separate logging |
| Cost | LLM API calls for orchestration loops | Compute only |
| Failure mode | If it crashes, tasks pile up unassigned; recovers on restart | Same, but more ops overhead |

The key advantage: **retry intelligence**. When a task fails, the Task Manager reads the failure reason and creates a retry with adjusted instructions. A backend service would just blindly retry with the same params. Claude can say "last attempt failed because Arctic Shift API returned 429 — waiting 5 minutes and using smaller batch sizes."

### Architecture

```
+------------------+         +------------------+         +------------------+
|   Admin UI       |         |   Go Backend     |         | Task Manager     |
|  (Next.js)       |         |   (Gin API)      |         | (Claude Code)    |
+--------+---------+         +--------+---------+         +--------+---------+
         |                            |                            |
  1. Create task    +------------>    |                            |
     (no assignee)                    |                            |
                                      |    2. Poll unassigned      |
                                      |       tasks         <------+
                                      |                            |
                                      |    3. Poll online          |
                                      |       workers       <------+
                                      |                            |
                                      |                   4. Pick best worker,
                                      |                      assign task
                                      |    5. PUT assign   <------+
                                      |                            |
                                      |                            |
+------------------+                  |                            |
|   ClawdBot       |                  |                            |
|  Worker          |                  |                            |
+--------+---------+                  |                            |
         |                            |                            |
  6. Poll pending   +------------>    |                            |
  7. Execute task                     |                            |
  8. Post result    +------------>    |                            |
                                      |   9. TM sees completion    |
                                      |      or failure     <------+
                                      |                            |
                                      |                  10. Send Telegram
                                      |                      notification
                                      |                            +------> Telegram
```

---

## 4. Database Changes

### Migration 031: Retry support and Task Manager config

```sql
-- Add retry fields to task_types
ALTER TABLE task_types
    ADD COLUMN IF NOT EXISTS max_retries INTEGER NOT NULL DEFAULT 3;

-- Add retry and lineage fields to worker_tasks
ALTER TABLE worker_tasks
    ADD COLUMN IF NOT EXISTS retry_count    INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS parent_task_id UUID REFERENCES worker_tasks(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_worker_tasks_parent ON worker_tasks(parent_task_id);

-- Assignment log: who assigned a task, when, and why
CREATE TABLE IF NOT EXISTS task_assignments (
    id          SERIAL PRIMARY KEY,
    task_id     UUID NOT NULL REFERENCES worker_tasks(id) ON DELETE CASCADE,
    assigned_to UUID NOT NULL REFERENCES users(id),
    assigned_by UUID REFERENCES users(id),    -- NULL = system/auto, UUID = human or Task Manager
    reason      TEXT,                          -- "auto: least loaded online worker" or "retry: previous attempt failed"
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_task_assignments_task ON task_assignments(task_id);
```

### Schema After Migration

**task_types** (extended):
| Column | Type | Notes |
|--------|------|-------|
| ... | ... | (all Phase 1 columns) |
| **max_retries** | **INTEGER** | **NEW** — max retry attempts (default 3) |

**worker_tasks** (extended):
| Column | Type | Notes |
|--------|------|-------|
| ... | ... | (all Phase 1 columns) |
| **retry_count** | **INTEGER** | **NEW** — how many times this task has been retried (0 = original) |
| **parent_task_id** | **UUID FK** | **NEW** — if this is a retry, points to the failed task |

**task_assignments** (new):
| Column | Type | Notes |
|--------|------|-------|
| id | SERIAL | PK |
| task_id | UUID FK | Which task |
| assigned_to | UUID FK | Which worker it was assigned to |
| assigned_by | UUID FK | Who assigned it (NULL = auto) |
| reason | TEXT | Why this assignment was made |
| created_at | TIMESTAMPTZ | When |

### Why task_assignments?

This table provides an audit trail. When a task gets reassigned (worker went offline, manual override), you can see the full history. It also lets the Task Manager explain its reasoning ("assigned to Genesis because it was the least loaded online worker").

---

## 5. API Changes

### New Fields on Existing Endpoints

#### Create Task (Admin) — new optional fields

```
POST /api/v1/admin/workers/tasks

{
  "title": "Reddit Crawl for NVDA",
  "task_type_id": 1,
  "params": { "ticker": "NVDA", "subreddits": ["wsb"], "days": 30 },
  "priority": "high"
  // NOTE: assigned_to is now OPTIONAL — leave empty for auto-assignment
}
```

When `assigned_to` is omitted, the task stays `pending` with `assigned_to = NULL`. The Task Manager picks it up and assigns it.

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

### New Endpoints

#### List Unassigned Tasks

The existing `GET /admin/workers/tasks` already supports `?assigned_to=<id>`. Add a special value:

```
GET /api/v1/admin/workers/tasks?unassigned=true
```

Returns tasks where `assigned_to IS NULL` and `status = 'pending'`.

#### Get Worker Capacity

```
GET /api/v1/admin/workers/capacity

Response 200:
{
  "success": true,
  "data": [
    {
      "id": "worker-uuid-1",
      "full_name": "Genesis",
      "is_online": true,
      "in_progress_count": 2,
      "pending_count": 1,
      "completed_today": 5,
      "last_activity_at": "2026-02-10T14:30:00Z"
    },
    {
      "id": "worker-uuid-2",
      "full_name": "Nexus",
      "is_online": true,
      "in_progress_count": 0,
      "pending_count": 0,
      "completed_today": 3,
      "last_activity_at": "2026-02-10T14:28:00Z"
    }
  ]
}
```

This gives the Task Manager everything it needs to make assignment decisions in one call.

#### Get Task Assignment History

```
GET /api/v1/admin/workers/tasks/:id/assignments

Response 200:
{
  "success": true,
  "data": [
    {
      "id": 1,
      "task_id": "abc-123",
      "assigned_to": "worker-uuid-1",
      "assigned_to_name": "Genesis",
      "assigned_by": "task-manager-uuid",
      "assigned_by_name": "Task Manager",
      "reason": "auto: least loaded online worker (0 in_progress tasks)",
      "created_at": "2026-02-10T14:30:00Z"
    }
  ]
}
```

#### Log Assignment (used by Task Manager)

```
POST /api/v1/admin/workers/tasks/:id/assign
Authorization: Bearer <jwt> (admin)

{
  "assigned_to": "worker-uuid-1",
  "reason": "auto: least loaded online worker (0 in_progress tasks)"
}

Response 200:
{ "success": true, "data": { "task": {...}, "assignment": {...} } }
```

This endpoint sets `assigned_to` on the task AND creates a `task_assignments` record. It replaces using the generic `PUT /tasks/:id` for assignments, providing atomicity and audit logging.

---

## 6. Task Manager Design

### How It Runs

The Task Manager is a Claude Code session with a project directory containing a `CLAUDE.md`. It runs on a persistent machine (cloud VM, always-on Mac, etc.):

```bash
# On the Task Manager's machine
cd /path/to/task-manager-project
claude  # starts Claude Code, reads CLAUDE.md, begins orchestration loop
```

### Task Manager CLAUDE.md

```markdown
# Task Manager — InvestorCenter.ai

You are the Task Manager for InvestorCenter's ClawdBot task queue. Your job
is to assign tasks to workers, retry failures, detect stuck tasks, and send
Telegram notifications. You do NOT execute data tasks yourself.

## Authentication

- API Base: https://api.investorcenter.ai/api/v1
- Login: POST /auth/login with email/password from env vars
  - TASK_MANAGER_EMAIL
  - TASK_MANAGER_PASSWORD
- Store the JWT token for subsequent requests
- You have both admin and worker access

## Main Loop

Run this loop continuously with a 30-second pause between iterations:

### 1. Heartbeat
POST /worker/heartbeat
(keeps you showing as online in the admin UI)

### 2. Check for Unassigned Tasks
GET /admin/workers/tasks?unassigned=true

If there are unassigned tasks, run the assignment algorithm (see below).

### 3. Check for Failed Tasks (retry candidates)
GET /admin/workers/tasks?status=failed

For each failed task:
- Skip if retry_count >= task_type.max_retries
- Skip if a retry already exists (check parent_task_id references)
- Read the task's updates to understand WHY it failed
- Create a retry task (see Retry section below)

### 4. Check for Stuck Tasks
GET /admin/workers/tasks?status=in_progress

For each in_progress task:
- Check if the assigned worker is online (GET /admin/workers/capacity)
- If the worker has been offline for >10 minutes AND the task has been
  in_progress for >15 minutes, mark it as failed with reason
  "Worker went offline during execution"
- This will trigger a retry on the next loop iteration

### 5. Send Notifications
After processing, send Telegram notifications for any events:
- Task completed: "{task_title} completed by {worker_name}. Result: {summary}"
- Task failed: "{task_title} failed. Reason: {failure_reason}. Retrying: {yes/no}"
- Task stuck: "{task_title} stuck — {worker_name} went offline. Reassigning."
- All retries exhausted: "{task_title} failed after {N} attempts. Needs human attention."

## Assignment Algorithm

When assigning an unassigned task:

1. GET /admin/workers/capacity to see all workers and their load
2. Filter to ONLINE workers only (exclude yourself — you're not a data worker)
3. Filter out workers that already have >= 3 in_progress tasks
4. Sort by in_progress_count ASC (least loaded first)
5. Assign to the first worker in the sorted list
6. POST /admin/workers/tasks/{id}/assign with reason explaining the choice

If no workers are available:
- Post an update on the task: "No available workers. Will retry assignment on next cycle."
- Send Telegram notification: "Task {title} waiting for available worker"
- Skip to next task

## Retry Logic

When creating a retry for a failed task:

1. Read the failed task's updates to understand the failure
2. POST /admin/workers/tasks to create a new task:
   - Same title with " (retry N)" appended
   - Same task_type_id and params
   - Description = original description + "\n\n---\n\nPREVIOUS ATTEMPT FAILED:\n{failure reason}\n\nPlease adjust your approach to avoid this issue."
   - parent_task_id = the failed task's ID
   - retry_count = failed task's retry_count + 1
   - priority = same as original (or bump to "high" if it was "medium")
   - Leave assigned_to empty — you'll assign it on the next loop iteration
3. Post an update on the original failed task: "Retry #{N} created: {new_task_id}"
4. Send Telegram notification about the retry

## Telegram Notifications

Use the Telegram Bot API directly:
- Bot token: from env var TELEGRAM_BOT_TOKEN
- Chat ID: from env var TELEGRAM_CHAT_ID
- Send messages with: curl POST https://api.telegram.org/bot{token}/sendMessage

Format messages in Markdown:
- *Bold* for task titles
- Status emoji: completed, failed, stuck, retrying
- Include task ID for reference
- Keep messages concise (under 200 chars when possible)

## Rules

- Never assign tasks to yourself
- Never execute data tasks — you are an orchestrator only
- Always explain your reasoning in assignment/retry reasons
- If something unexpected happens, post a Telegram notification and continue
- Send heartbeat every iteration to stay online
- Be conservative with retries — if the failure looks permanent (e.g., "API
  endpoint does not exist"), don't retry
- Log all decisions as task updates so humans can audit
```

### Environment Variables

The Task Manager machine needs:

```bash
TASK_MANAGER_EMAIL=taskmanager@investorcenter.ai
TASK_MANAGER_PASSWORD=<secure-password>
TELEGRAM_BOT_TOKEN=<telegram-bot-token>
TELEGRAM_CHAT_ID=<telegram-chat-id>
API_BASE=https://api.investorcenter.ai
```

---

## 7. Assignment Algorithm

### Decision Flow

```
Unassigned task arrives
        |
        v
GET /admin/workers/capacity
        |
        v
Filter: online workers only
        |
        v
Filter: exclude Task Manager (self)
        |
        v
Filter: in_progress_count < 3
        |
        v
Sort: by in_progress_count ASC (least loaded)
        |
        v
Tie-break: by completed_today ASC (spread work evenly)
        |
        v
Assign to top candidate
        |
        v
POST /admin/workers/tasks/{id}/assign
  reason: "auto: least loaded online worker ({name}, {N} in_progress)"
```

### Edge Cases

| Scenario | Behavior |
|----------|----------|
| No workers online | Skip, retry next cycle. Post update on task + Telegram notification. |
| All workers at capacity (>=3 in_progress) | Skip, retry next cycle. Telegram: "All workers at capacity." |
| Only 1 worker online | Assign to them (unless at capacity). |
| Task has been waiting >30 min | Bump Telegram notification to include "waiting {N} min for assignment." |
| Human manually assigns before Task Manager | Task Manager sees it's already assigned, skips it. |

### Capacity Limits

The default concurrency limit is **3 in_progress tasks per worker**. This is a constant in the Task Manager's CLAUDE.md, not a database field. If we need per-worker limits later, we can add a `max_concurrent_tasks` column to the users table.

---

## 8. Retry System

### How Retries Work

```
Original Task (retry_count=0)
    |
    | fails
    v
Task Manager reads failure reason from task updates
    |
    v
Creates new task:
  - title: "Reddit Crawl for NVDA (retry 1)"
  - parent_task_id: original task ID
  - retry_count: 1
  - description: original + failure context
  - assigned_to: NULL (assigned on next loop)
    |
    | if retry 1 also fails
    v
Creates retry 2 (retry_count=2, parent=retry 1)
    |
    | if retry 2 also fails
    v
Creates retry 3 (retry_count=3, parent=retry 2)
    |
    | if retry 3 fails
    v
max_retries (3) reached — sends Telegram alert, no more retries
"Reddit Crawl for NVDA failed after 3 attempts. Needs human attention."
```

### Retry Intelligence

The key advantage of having Claude as the Task Manager: it reads the failure reason and adapts.

**Example 1: Rate limit failure**
- Failure: "Arctic Shift API returned 429 Too Many Requests after 50 calls"
- Retry description appended: "PREVIOUS ATTEMPT FAILED: Rate limited after 50 API calls. Please use longer delays between requests (3-5 seconds) and smaller batch sizes."

**Example 2: Auth failure**
- Failure: "Got 401 Unauthorized when posting results to /api/v1/reddit/posts"
- Task Manager recognizes this is likely a permanent issue (JWT expired?), sends Telegram notification: "Auth failure on task X — may need fresh credentials" but still creates the retry (worker will re-auth).

**Example 3: Data issue**
- Failure: "No posts found for ticker XYZZ in any subreddit"
- Task Manager recognizes this might be a bad ticker, sends Telegram: "No data found for XYZZ — is this a valid ticker?" and flags it as `failed` without retry (permanent failure detected).

### Retry Fields

| Field | Where | Description |
|-------|-------|-------------|
| `max_retries` | `task_types` | Max retries for this type of task (default 3) |
| `retry_count` | `worker_tasks` | Which retry attempt this is (0 = original) |
| `parent_task_id` | `worker_tasks` | Link to the task this retries (NULL for originals) |

### The Task Manager Decides Whether to Retry

The Task Manager uses judgment (it's Claude) to decide:
- **Retry**: transient errors (rate limits, timeouts, worker crash)
- **Don't retry**: permanent errors (invalid ticker, API doesn't exist, auth misconfigured)
- **Retry with changes**: fixable errors (add delay, reduce batch size, try different subreddit)

This is expressed in the retry task's description, not in code.

---

## 9. Telegram Notifications

### Setup

1. Create a Telegram bot via @BotFather
2. Get the bot token
3. Create a Telegram group/channel for notifications
4. Add the bot to the group
5. Get the chat ID
6. Set env vars on the Task Manager machine

### Notification Types

| Event | Message Format | Priority |
|-------|---------------|----------|
| Task completed | `*{title}* completed by {worker}. {result_summary}` | Normal |
| Task failed | `*{title}* failed. Reason: {reason}. Retry: {yes/no}` | High |
| All retries exhausted | `*{title}* failed after {N} attempts. Needs human attention.` | Urgent |
| Task stuck | `*{title}* stuck — {worker} offline for {N} min. Reassigning.` | High |
| No workers available | `*{title}* waiting for available worker ({N} min)` | Normal |
| Worker went offline | `Worker {name} went offline with {N} in_progress tasks` | Normal |

### Implementation

The Task Manager sends notifications directly using `curl`:

```bash
curl -s -X POST "https://api.telegram.org/bot${TELEGRAM_BOT_TOKEN}/sendMessage" \
  -H "Content-Type: application/json" \
  -d '{
    "chat_id": "'${TELEGRAM_CHAT_ID}'",
    "text": "*Reddit Crawl for NVDA* completed by Genesis.\n847 posts collected.",
    "parse_mode": "Markdown"
  }'
```

No backend notification service needed. The Task Manager is a Claude Code session — it can run curl/python directly.

### Rate Limiting

The Task Manager should batch notifications when possible. If 5 tasks complete within one loop iteration, send a single summary message instead of 5 individual messages.

---

## 10. Stuck Task Detection

### What is a "Stuck" Task?

A task that's been `in_progress` for too long with an offline worker. This usually means the ClawdBot session crashed or lost network.

### Detection Rules

```
A task is STUCK if ALL of these are true:
  1. status = "in_progress"
  2. assigned worker is OFFLINE (last_activity_at > 10 minutes ago)
  3. task has been in_progress for > 15 minutes (started_at > 15 min ago)
```

### Resolution

1. Task Manager posts an update: "Worker {name} appears offline. Marking task as failed for reassignment."
2. Task Manager marks task as `failed` via admin API
3. This triggers the retry logic on the next loop iteration
4. Telegram notification sent

### Why Not Reassign Directly?

We mark as `failed` and let the retry system handle it, rather than directly reassigning. This ensures:
- The failure is logged
- retry_count is incremented
- The retry gets failure context in its description
- Assignment history is clean (the original assignment "failed", the retry is a new assignment)

---

## 11. UI Changes

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

### Task Detail — Assignment History

Show the assignment log below the task metadata:

```
Assignment History:
  Feb 10, 14:30 — Assigned to Genesis by Task Manager
    Reason: auto: least loaded online worker (0 in_progress tasks)
  Feb 10, 14:45 — Assigned to Nexus by Task Manager
    Reason: retry: Genesis went offline during execution
```

### Task List — New Filters

Add filters:
- `unassigned` — show only unassigned tasks
- `retries` — show only retry tasks (retry_count > 0)
- `stuck` — show tasks in_progress with offline worker (client-side filter)

### Workers Tab — Capacity View

Show per-worker capacity info:
- In-progress count: "2/3"
- Pending count
- Completed today count

### Admin Dashboard — Task Manager Status

Show whether the Task Manager is online (via heartbeat), and a summary:
- "Task Manager: Online (last heartbeat: 30s ago)"
- "Unassigned tasks: 0"
- "Failed awaiting retry: 2"
- "Stuck tasks: 0"

---

## 12. Implementation Plan

### Phase 2a: Database + Backend (Week 1)

| Task | Effort | Description |
|------|--------|-------------|
| Migration 031 | S | Add retry fields, create task_assignments table |
| Worker capacity endpoint | M | GET /admin/workers/capacity with task counts |
| Assignment endpoint | S | POST /admin/workers/tasks/:id/assign (atomic assign + log) |
| Assignment history endpoint | S | GET /admin/workers/tasks/:id/assignments |
| Unassigned filter | S | Add `?unassigned=true` to ListTasks |
| Extend task response | S | Include retry_count, parent_task_id, max_retries in responses |
| Extend CreateTask | S | Accept parent_task_id, retry_count in create |

### Phase 2b: Task Manager (Week 2)

| Task | Effort | Description |
|------|--------|-------------|
| Register Task Manager user | S | Create user with is_admin + is_worker |
| Write CLAUDE.md | M | Task Manager system prompt (the SOP above) |
| Set up Telegram bot | S | Create bot, get token, set up notification channel |
| Test assignment loop | M | Verify: create unassigned task → TM assigns → worker picks up |
| Test retry loop | M | Verify: task fails → TM creates retry → retry succeeds |
| Test stuck detection | M | Verify: worker goes offline → TM detects → marks failed → retries |
| Test Telegram notifications | S | Verify all notification types send correctly |

### Phase 2c: Frontend (Week 2-3)

| Task | Effort | Description |
|------|--------|-------------|
| Retry chain in task detail | M | Show parent/child links, retry badge |
| Assignment history in task detail | S | Show assignment log |
| Worker capacity view | S | Show in_progress/pending/completed counts |
| Task Manager status indicator | S | Online badge, unassigned/failed/stuck counts |
| New task list filters | S | Unassigned, retries filters |
| Frontend API client updates | S | Add new types, endpoints |

**S** = Small (< 1 day), **M** = Medium (1-3 days)

**Total estimate: ~3 weeks**

---

## 13. Future (Phase 3)

Phase 3 adds **autonomous task creation** and **Telegram commands**:

- Task Manager scans for tickers with stale data → creates crawl tasks automatically
- Task Manager receives Telegram commands: `/status`, `/add reddit NVDA`, `/pause`
- Priority scoring based on watchlist popularity, data staleness, trending tickers
- Daily digest via Telegram
- Budget/rate limit awareness (track LLM token usage per task)

---

## Appendix A: Task Manager vs Workers Comparison

| Aspect | Task Manager | Data Worker (ClawdBot) |
|--------|-------------|----------------------|
| Role | Orchestrator | Executor |
| Auth | is_admin + is_worker | is_worker only |
| Executes data tasks? | No | Yes |
| Reads SOPs? | No (uses its own CLAUDE.md) | Yes (SOP tells it what to do) |
| Assigns tasks? | Yes | No |
| Creates tasks? | Only retries | No |
| Sends notifications? | Yes (Telegram) | No (just task updates) |
| Loop interval | 30 seconds | Task-dependent |
| Runs on | Persistent VM | Any machine |

## Appendix B: Example Scenario

```
Time    Event
-----   -----
10:00   Human creates "Reddit Crawl for NVDA" in admin UI (no assignee)
10:00   Task status: pending, assigned_to: NULL

10:00   Task Manager loop runs
10:00   TM sees unassigned task
10:00   TM fetches worker capacity: Genesis (0 in_progress), Nexus (2 in_progress)
10:00   TM assigns to Genesis: "auto: least loaded (0 in_progress)"
10:00   → Telegram: "Assigned *Reddit Crawl for NVDA* to Genesis"

10:02   Genesis polls, picks up task, marks in_progress (started_at = now)

10:05   Genesis posts update: "Collected 300 posts from r/wsb..."

10:08   Genesis's machine loses network. Heartbeat stops.

10:18   Task Manager loop detects: Genesis offline for 10 min, task in_progress for 16 min
10:18   TM marks task as failed: "Worker Genesis went offline during execution"
10:18   → Telegram: "*Reddit Crawl for NVDA* stuck — Genesis offline. Retrying."

10:18   Task Manager sees failed task, retry_count=0, max_retries=3
10:18   TM reads failure: "Worker went offline"
10:18   TM creates retry task: "Reddit Crawl for NVDA (retry 1)"
         retry_count=1, parent_task_id=original
         description includes: "Previous attempt: worker went offline mid-execution"

10:19   TM assigns retry to Nexus: "auto: only available online worker"
10:19   → Telegram: "Retry 1/3: *Reddit Crawl for NVDA* assigned to Nexus"

10:20   Nexus picks up, executes, completes successfully
10:25   → Telegram: "*Reddit Crawl for NVDA (retry 1)* completed by Nexus. 847 posts."
```
