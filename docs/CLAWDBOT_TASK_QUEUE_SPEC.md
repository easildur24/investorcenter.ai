# ClawdBot Task Queue System: Product & Technical Spec

> **Version**: 4.0
> **Date**: February 2026
> **Status**: Phase 1 Design

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [What Exists Today](#2-what-exists-today)
3. [Phase 1 Design](#3-phase-1-design)
4. [Database Changes](#4-database-changes)
5. [API Specification](#5-api-specification)
6. [ClawdBot Worker Design](#6-clawdbot-worker-design)
7. [SOP System](#7-sop-system)
8. [UI Changes](#8-ui-changes)
9. [Implementation Plan](#9-implementation-plan)
10. [Future Phases](#10-future-phases)

---

## 1. Executive Summary

### What is the ClawdBot Task Queue?

A task execution system where **humans create tasks** via the admin UI and **ClawdBots (AI agents) execute them**. Each task has a **task type** with a **Standard Operating Procedure (SOP)** that tells the ClawdBot how to do the work. The ClawdBot reads the SOP + task-specific params and figures out the rest.

### Phase 1 Scope

- **Extend the existing** `/admin/workers` system — don't rebuild
- **Add `task_types` table** with SOPs (the playbook for each kind of task)
- **Extend `worker_tasks`** with task type, params, and structured results
- **No agent manager** — humans create and assign tasks manually
- **ClawdBots read directly** from the API — SOP + params tells them what to do
- **Same CRUD endpoints** used by both the admin UI and ClawdBots

### Key Insight

ClawdBots are AI agents. They don't need pre-rendered prompts — give them the SOP ("here's how to crawl Reddit") and the params ("ticker: NVDA, subreddits: [wsb]") and they'll figure out the prompt themselves.

---

## 2. What Exists Today

### Database (migrations 028-029)

**`users` table** — extended with:
- `is_worker` (BOOLEAN) — marks a user as a ClawdBot worker
- `last_activity_at` (TIMESTAMP) — updated on heartbeat/activity

**`worker_tasks` table**:
```sql
id          UUID PRIMARY KEY
title       VARCHAR(500) NOT NULL
description TEXT DEFAULT ''
assigned_to UUID REFERENCES users(id)     -- which worker
status      VARCHAR(20) DEFAULT 'pending' -- pending | in_progress | completed | failed
priority    VARCHAR(10) DEFAULT 'medium'  -- low | medium | high | urgent
created_by  UUID REFERENCES users(id)     -- which admin created it
created_at  TIMESTAMP
updated_at  TIMESTAMP                     -- auto-updated via trigger
```

**`worker_task_updates` table** — comment/log thread per task:
```sql
id          UUID PRIMARY KEY
task_id     UUID REFERENCES worker_tasks(id)
content     TEXT NOT NULL
created_by  UUID REFERENCES users(id)
created_at  TIMESTAMP
```

### API Routes

**Admin routes** (`/api/v1/admin/workers/*`, JWT auth):
| Method | Path | Handler | What |
|--------|------|---------|------|
| GET | `/admin/workers` | ListWorkers | List all workers (users with is_worker=true) |
| POST | `/admin/workers` | RegisterWorker | Create worker (creates user with is_worker=true) |
| DELETE | `/admin/workers/:id` | DeleteWorker | Remove worker (sets is_worker=false) |
| GET | `/admin/workers/tasks` | ListTasks | List tasks (filter by status, assigned_to) |
| POST | `/admin/workers/tasks` | CreateTask | Create task (title, desc, assignee, priority) |
| GET | `/admin/workers/tasks/:id` | GetTask | Get single task |
| PUT | `/admin/workers/tasks/:id` | UpdateTask | Update task fields |
| DELETE | `/admin/workers/tasks/:id` | DeleteTask | Delete task |
| GET | `/admin/workers/tasks/:id/updates` | ListTaskUpdates | Get task comment thread |

**Worker routes** (`/api/v1/worker/*`, JWT auth, is_worker=true required):
| Method | Path | Handler | What |
|--------|------|---------|------|
| GET | `/worker/tasks` | WorkerGetMyTasks | Get my assigned tasks (filter by status) |
| GET | `/worker/tasks/:id` | WorkerGetTask | Get single task assigned to me |
| PUT | `/worker/tasks/:id/status` | WorkerUpdateTaskStatus | Update status (in_progress/completed/failed) |
| GET | `/worker/tasks/:id/updates` | WorkerGetTaskUpdates | Get task comment thread |
| POST | `/worker/tasks/:id/updates` | WorkerPostUpdate | Post a comment/update |
| POST | `/worker/heartbeat` | WorkerHeartbeat | Heartbeat (updates last_activity_at) |

### Frontend

- `/admin/workers` page with **Workers** tab and **All Tasks** tab
- Workers tab: list workers, register new worker (email/password/name), show online status
- Tasks tab: list/filter tasks, create task (title/desc/assignee/priority), task detail with update thread
- `/worker/dashboard` page: worker's view of their assigned tasks, post updates

### Auth Model

Workers are **users with `is_worker=true`**. They log in with email/password like any user and get a JWT. No separate API keys. Online status = `last_activity_at` within 5 minutes.

---

## 3. Phase 1 Design

### What We're Adding

```
+---------------------+         +-------------------+         +-----------------+
|   /admin/workers    |         |    Go Backend     |         |    ClawdBot     |
|   (Next.js UI)      |         |    (Gin API)      |         |  (Claude Code)  |
+----------+----------+         +--------+----------+         +--------+--------+
           |                             |                             |
  1. Admin creates                       |                             |
     task type with SOP  +------------>  |                             |
                                         |                             |
  2. Admin creates task  +------------>  |                             |
     (pick type, fill                    |                             |
      params, assign)                    |                             |
                                         |                             |
                                         |  3. ClawdBot fetches        |
                                         |     assigned tasks   <------+
                                         |                             |
                                         |  4. Response includes       |
                                         |     task + SOP (joined) +-->|
                                         |                             |
                                         |                    5. ClawdBot reads SOP
                                         |                       + params, executes
                                         |                             |
                                         |  6. ClawdBot posts   <------+
                                         |     result + status         |
                                         |                             |
  7. Admin sees results  <-------------+ |                             |
     in task detail UI                   |                             |
```

### Core Concepts

**Task Type** = a category of work with a reusable SOP
- Example: "Reddit Crawl", "SEC Filing Crawl", "AI Sentiment Analysis"
- Each task type has an **SOP** (Standard Operating Procedure) — a markdown document that tells the ClawdBot *how* to do this kind of work
- Each task type defines a **param_schema** — what parameters are needed (ticker, date range, etc.)

**Task** = a specific instance of work
- Example: "Reddit Crawl for NVDA, last 30 days, from r/wallstreetbets"
- Links to a task type via `task_type_id`
- Has **params** (JSONB) — the specific values for this task
- Has **result** (JSONB) — structured output from the ClawdBot

**SOP** = the playbook document stored on the task type
- NOT a fill-in-the-blank template — it's documentation
- The ClawdBot is smart enough to combine SOP + params on its own
- Changing the SOP changes behavior for all future tasks of that type — no deploy needed

### End-to-End Flow

```
1. Admin creates task type "Reddit Crawl" with SOP:
   "Use Arctic Shift API. Endpoint: /api/posts/search.
    For each post collect: id, subreddit, title, body, score...
    POST results to /api/v1/reddit/posts in batches of 100..."

2. Admin creates task:
   - Type: Reddit Crawl
   - Params: { "ticker": "NVDA", "subreddits": ["wallstreetbets", "stocks"], "days": 30 }
   - Assigned to: Genesis
   - Priority: high

3. Genesis (ClawdBot) calls GET /api/v1/worker/tasks?status=pending
   Response includes the task with SOP joined in:
   {
     "id": "abc-123",
     "title": "Reddit Crawl for NVDA",
     "task_type": { "name": "reddit_crawl", "sop": "Use Arctic Shift API..." },
     "params": { "ticker": "NVDA", "subreddits": ["wallstreetbets", "stocks"], "days": 30 },
     "status": "pending",
     "priority": "high"
   }

4. Genesis reads the SOP + params and knows exactly what to do:
   - Call Arctic Shift API for NVDA posts in r/wallstreetbets and r/stocks
   - Collect posts from last 30 days
   - POST results to InvestorCenter API

5. Genesis updates status:
   PUT /api/v1/worker/tasks/abc-123/status { "status": "in_progress" }

6. Genesis executes the work, posts progress updates:
   POST /api/v1/worker/tasks/abc-123/updates { "content": "Collected 342 posts from r/wallstreetbets..." }

7. Genesis reports completion:
   PUT /api/v1/worker/tasks/abc-123/status { "status": "completed" }
   POST /api/v1/worker/tasks/abc-123/result {
     "posts_collected": 847,
     "subreddits_crawled": ["wallstreetbets", "stocks"],
     "date_range": "2026-01-10 to 2026-02-09",
     "api_calls_made": 12
   }
```

---

## 4. Database Changes

### Migration 030: Create task_types and extend worker_tasks

```sql
-- Task types table with SOPs
CREATE TABLE task_types (
    id           SERIAL PRIMARY KEY,
    name         VARCHAR(100) NOT NULL UNIQUE,     -- 'reddit_crawl', 'sec_filing', etc.
    label        VARCHAR(200) NOT NULL,            -- 'Reddit Crawl' (display name)
    sop          TEXT NOT NULL,                    -- Standard Operating Procedure (markdown)
    param_schema JSONB,                            -- Describes expected params for the UI
                                                   -- e.g. {"ticker": "string", "subreddits": "string[]"}
    is_active    BOOLEAN NOT NULL DEFAULT TRUE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Extend worker_tasks with task type linkage, params, and results
ALTER TABLE worker_tasks
    ADD COLUMN task_type_id INTEGER REFERENCES task_types(id),
    ADD COLUMN params       JSONB,               -- Task-specific params: {"ticker": "NVDA", ...}
    ADD COLUMN result       JSONB,               -- Structured result from ClawdBot
    ADD COLUMN started_at   TIMESTAMPTZ,
    ADD COLUMN completed_at TIMESTAMPTZ;

CREATE INDEX idx_worker_tasks_task_type ON worker_tasks(task_type_id);
CREATE INDEX idx_worker_tasks_params ON worker_tasks USING GIN (params);

-- Update trigger for task_types
CREATE OR REPLACE FUNCTION update_task_types_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_task_types_updated_at
    BEFORE UPDATE ON task_types
    FOR EACH ROW
    EXECUTE FUNCTION update_task_types_updated_at();

-- Seed initial task types
INSERT INTO task_types (name, label, sop, param_schema) VALUES
('reddit_crawl', 'Reddit Crawl', 'See SOP section in spec', '{"ticker": "string", "subreddits": "string[]", "days": "number"}'),
('ai_sentiment', 'AI Sentiment Analysis', 'See SOP section in spec', '{"ticker": "string", "source": "string", "limit": "number"}'),
('sec_filing', 'SEC Filing Crawl', 'See SOP section in spec', '{"ticker": "string", "filing_type": "string"}'),
('custom', 'Custom Task', 'Follow the task description directly.', NULL);
```

### Schema After Migration

**task_types**:
| Column | Type | Notes |
|--------|------|-------|
| id | SERIAL | PK |
| name | VARCHAR(100) | Unique slug: `reddit_crawl` |
| label | VARCHAR(200) | Display name: `Reddit Crawl` |
| sop | TEXT | Full SOP document (markdown) |
| param_schema | JSONB | What params the UI should render |
| is_active | BOOLEAN | Soft delete |
| created_at | TIMESTAMPTZ | |
| updated_at | TIMESTAMPTZ | Auto-updated |

**worker_tasks** (existing + new columns):
| Column | Type | Notes |
|--------|------|-------|
| id | UUID | PK (existing) |
| title | VARCHAR(500) | (existing) |
| description | TEXT | (existing) |
| assigned_to | UUID FK users | (existing) |
| status | VARCHAR(20) | pending/in_progress/completed/failed (existing) |
| priority | VARCHAR(10) | low/medium/high/urgent (existing) |
| created_by | UUID FK users | (existing) |
| created_at | TIMESTAMP | (existing) |
| updated_at | TIMESTAMP | (existing, auto-trigger) |
| **task_type_id** | **INTEGER FK task_types** | **NEW** — links to task type |
| **params** | **JSONB** | **NEW** — task-specific parameters |
| **result** | **JSONB** | **NEW** — structured result from ClawdBot |
| **started_at** | **TIMESTAMPTZ** | **NEW** — when work began |
| **completed_at** | **TIMESTAMPTZ** | **NEW** — when work finished |

---

## 5. API Specification

### New Endpoints

#### Task Types (Admin)

```
GET /api/v1/admin/workers/task-types
Authorization: Bearer <jwt>

Response 200:
{
  "success": true,
  "data": [
    {
      "id": 1,
      "name": "reddit_crawl",
      "label": "Reddit Crawl",
      "sop": "## Reddit Crawl SOP\n\nUse the Arctic Shift API...",
      "param_schema": {"ticker": "string", "subreddits": "string[]", "days": "number"},
      "is_active": true,
      "created_at": "2026-02-09T00:00:00Z",
      "updated_at": "2026-02-09T00:00:00Z"
    }
  ]
}
```

```
POST /api/v1/admin/workers/task-types
Authorization: Bearer <jwt>

{
  "name": "news_crawl",
  "label": "News Crawl",
  "sop": "## News Crawl SOP\n\nSearch Google News for...",
  "param_schema": {"ticker": "string", "days": "number"}
}

Response 201:
{ "success": true, "data": { "id": 5, ... } }
```

```
PUT /api/v1/admin/workers/task-types/:id
Authorization: Bearer <jwt>

{ "sop": "## Updated SOP\n\n...", "param_schema": { ... } }

Response 200:
{ "success": true, "data": { ... } }
```

```
DELETE /api/v1/admin/workers/task-types/:id
Authorization: Bearer <jwt>

Response 200:
{ "success": true, "message": "Task type deactivated" }
```

#### Worker: Read Task Type SOP

```
GET /api/v1/worker/task-types/:id
Authorization: Bearer <jwt> (worker)

Response 200:
{
  "success": true,
  "data": {
    "id": 1,
    "name": "reddit_crawl",
    "label": "Reddit Crawl",
    "sop": "## Reddit Crawl SOP\n\n..."
  }
}
```

#### Worker: Post Result

```
POST /api/v1/worker/tasks/:id/result
Authorization: Bearer <jwt> (worker)

{
  "result": {
    "posts_collected": 847,
    "posts_stored": 847,
    "subreddits_crawled": ["wallstreetbets", "stocks"],
    "date_range": "2026-01-10 to 2026-02-09"
  }
}

Response 200:
{ "success": true, "data": { ... } }
```

### Modified Endpoints

#### Create Task (Admin) — add task_type_id and params

```
POST /api/v1/admin/workers/tasks
Authorization: Bearer <jwt>

{
  "title": "Reddit Crawl for NVDA",
  "description": "Crawl last 30 days of NVDA mentions",
  "task_type_id": 1,
  "params": {
    "ticker": "NVDA",
    "subreddits": ["wallstreetbets", "stocks", "investing"],
    "days": 30
  },
  "assigned_to": "genesis-user-uuid",
  "priority": "high"
}

Response 201:
{
  "success": true,
  "data": {
    "id": "abc-123-...",
    "title": "Reddit Crawl for NVDA",
    "task_type_id": 1,
    "params": { "ticker": "NVDA", ... },
    "status": "pending",
    "priority": "high",
    ...
  }
}
```

#### Get My Tasks (Worker) — response now includes SOP via JOIN

```
GET /api/v1/worker/tasks?status=pending
Authorization: Bearer <jwt> (worker)

Response 200:
{
  "success": true,
  "data": [
    {
      "id": "abc-123-...",
      "title": "Reddit Crawl for NVDA",
      "description": "Crawl last 30 days of NVDA mentions",
      "status": "pending",
      "priority": "high",
      "params": {
        "ticker": "NVDA",
        "subreddits": ["wallstreetbets", "stocks", "investing"],
        "days": 30
      },
      "task_type": {
        "id": 1,
        "name": "reddit_crawl",
        "label": "Reddit Crawl",
        "sop": "## Reddit Crawl SOP\n\nUse the Arctic Shift API..."
      },
      "result": null,
      "created_at": "2026-02-09T10:00:00Z",
      "updated_at": "2026-02-09T10:00:00Z"
    }
  ]
}
```

This is the key design decision: **SOP is included directly in the task response** (via JOIN on task_types). The ClawdBot gets everything it needs in one API call.

#### Update Task Status (Worker) — also set started_at / completed_at

The existing `PUT /worker/tasks/:id/status` handler is updated to:
- Set `started_at = NOW()` when status changes to `in_progress`
- Set `completed_at = NOW()` when status changes to `completed` or `failed`

---

## 6. ClawdBot Worker Design

### What is a ClawdBot?

A ClawdBot is a **Claude Code session** running on any machine. It logs into InvestorCenter as a worker user, fetches its assigned tasks (with SOPs), executes them, and reports results back via the API.

### ClawdBot CLAUDE.md

Each machine running a ClawdBot has a project directory with this `CLAUDE.md`:

```markdown
# ClawdBot Worker — InvestorCenter.ai

You are a ClawdBot worker. Your job is to execute tasks assigned to you
by the InvestorCenter task queue.

## Authentication

- API Base: https://api.investorcenter.ai/api/v1
- Login: POST /auth/login with email/password from env vars
  - CLAWDBOT_EMAIL
  - CLAWDBOT_PASSWORD
- Store the JWT token for subsequent requests

## Workflow

1. **Fetch your tasks**:
   GET /worker/tasks?status=pending
   (also check status=in_progress for any interrupted tasks)

2. **For each task**:
   a. Read the task's `task_type.sop` — this is your playbook
   b. Read the task's `params` — these are the specifics
   c. Mark as in_progress: PUT /worker/tasks/{id}/status {"status": "in_progress"}
   d. Execute the work described by SOP + params
   e. Post progress updates: POST /worker/tasks/{id}/updates {"content": "..."}
   f. Post structured result: POST /worker/tasks/{id}/result {"result": {...}}
   g. Mark as completed: PUT /worker/tasks/{id}/status {"status": "completed"}
   h. If something fails: PUT /worker/tasks/{id}/status {"status": "failed"}
      and post an update explaining what went wrong

3. **Send heartbeat** every 2 minutes:
   POST /worker/heartbeat

4. **Repeat** — check for more tasks

## Rules

- Follow the SOP instructions precisely
- Always report failures honestly with details
- Post progress updates so humans can see what's happening
- Do not modify the InvestorCenter codebase
- You have access to: bash, curl, python, node on this machine
```

### What Makes This Work

- **Zero deployment** — Just Claude Code + a CLAUDE.md + worker credentials
- **SOP-driven** — Changing how a task type works = editing the SOP in the admin UI
- **Self-sufficient** — ClawdBot reads SOP + params and generates its own plan
- **Observable** — Progress updates posted to the task thread in real-time
- **Any machine** — Your Mac, a friend's machine, a cloud VM

---

## 7. SOP System

### What is an SOP?

An SOP (Standard Operating Procedure) is a markdown document stored on each task type. It tells the ClawdBot **how** to do a category of work. The ClawdBot combines the SOP with task-specific **params** to know both the how and the what.

### Example SOP: reddit_crawl

```markdown
## Reddit Crawl SOP

### Data Source
Use the Arctic Shift API: https://arctic-shift.photon-reddit.com/api
- Endpoint: GET /api/posts/search
- Query params: subreddit, q (search term), after (epoch), before (epoch), limit (max 100)
- Rate limit: Be respectful — add 1s delay between requests

### What to Collect
For each post:
- external_id (Reddit post ID, e.g., "t3_abc123")
- subreddit
- title
- body (selftext, may be empty for link posts)
- score (upvotes - downvotes)
- num_comments
- author
- created_utc (ISO 8601)
- url (permalink)

### How to Store Results
POST batches of up to 100 posts to:
  https://api.investorcenter.ai/api/v1/reddit/posts

Request body:
{
  "posts": [
    {
      "external_id": "t3_abc123",
      "subreddit": "wallstreetbets",
      "title": "NVDA to the moon",
      "body": "...",
      "score": 1234,
      "num_comments": 56,
      "author": "user123",
      "created_utc": "2026-01-15T10:00:00Z",
      "url": "https://reddit.com/r/wallstreetbets/...",
      "tickers": ["NVDA"]
    }
  ]
}

Use your worker JWT in the Authorization header.

### Task Params You'll Receive
- `ticker` (string) — e.g., "NVDA"
- `subreddits` (string[]) — e.g., ["wallstreetbets", "stocks"]
- `days` (number) — how many days back to search

### Completion
Report back with:
- Total posts collected and stored
- Actual date range of posts found
- Any errors or rate limit issues
```

### Example SOP: ai_sentiment

```markdown
## AI Sentiment Analysis SOP

### What You're Doing
Analyze social media posts about a specific ticker and classify each
post's sentiment as bullish, bearish, or neutral.

### Data Source
Fetch posts from InvestorCenter API:
  GET https://api.investorcenter.ai/api/v1/reddit/posts?ticker={ticker}&limit={limit}

### Analysis
For each post, determine:
- sentiment: "bullish" | "bearish" | "neutral"
- confidence: 0.0 to 1.0
- key_phrases: array of phrases that indicate sentiment
- reasoning: one sentence explaining the classification

### Storing Results
POST results to:
  https://api.investorcenter.ai/api/v1/sentiment/batch

Request body:
{
  "analyses": [
    {
      "post_id": "t3_abc123",
      "ticker": "NVDA",
      "sentiment": "bullish",
      "confidence": 0.85,
      "key_phrases": ["to the moon", "strong earnings"],
      "reasoning": "Post expresses strong optimism about upcoming earnings"
    }
  ]
}

### Task Params
- `ticker` (string)
- `source` (string) — "reddit" or "twitter"
- `limit` (number) — max posts to analyze

### Completion
Report: total posts analyzed, sentiment breakdown (X bullish, Y bearish, Z neutral),
average confidence score.
```

### SOP vs Prompt Template

| Aspect | SOP (Phase 1) | Prompt Template (Future) |
|--------|---------------|--------------------------|
| What it is | Documentation / playbook | Fill-in-the-blank template |
| Who generates the prompt? | The ClawdBot itself | A Task Manager agent |
| Variable substitution | None — ClawdBot reads params directly | `{{ticker}}` → "NVDA" |
| Flexibility | High — ClawdBot adapts to edge cases | Lower — template is rigid |
| Requires agent manager? | No | Yes |

SOPs are the right choice for Phase 1 because they're simpler and leverage the ClawdBot's intelligence. In future phases, a Task Manager agent could generate more precise prompts from templates when needed.

---

## 8. UI Changes

### Task Types Management

Add a **Task Types** tab (or section within existing tabs) in `/admin/workers`:

- **List task types** — name, label, active/inactive badge, task count
- **Create task type** — form: name (slug), label, SOP (markdown editor), param_schema (JSON)
- **Edit task type** — edit SOP in a markdown editor, update param_schema
- **Deactivate task type** — soft delete (sets is_active=false)

### Create Task Form (Enhanced)

The existing create task form gets a **Task Type** dropdown. When a type is selected:
- Show param fields dynamically based on `param_schema`
- Auto-populate the title: "[Label] for [ticker]" (e.g., "Reddit Crawl for NVDA")
- Keep the existing description, assignee, and priority fields

For the **Custom** task type, just show title + description (no params).

### Task Detail View (Enhanced)

Add to the existing task detail panel:
- **Task Type** badge
- **Params** displayed as key-value pairs
- **Result** displayed as formatted JSON (when completed)
- **Duration** (completed_at - started_at)
- Keep existing update thread

### Task List (Enhanced)

Add to the existing task list:
- **Task Type** column/badge
- **Filter by task type** dropdown
- **Params preview** (e.g., "NVDA, wsb, 30d")

---

## 9. Implementation Plan

### Phase 1a: Database + Backend (Week 1)

| Task | Effort | Description |
|------|--------|-------------|
| Migration 030 | S | Create `task_types` table, extend `worker_tasks` |
| Go structs | S | TaskType struct, extend WorkerTask struct |
| Task Types CRUD | M | 4 admin endpoints + 1 worker endpoint |
| Extend CreateTask | S | Add task_type_id, params to create handler |
| Extend ListTasks | S | JOIN task_types, include in response |
| Worker GET with SOP | M | JOIN task_types into worker task responses |
| Worker POST result | S | New endpoint for structured results |
| Update status handler | S | Set started_at/completed_at timestamps |
| Seed task types | S | Insert initial SOPs for reddit_crawl, ai_sentiment, custom |

### Phase 1b: Frontend (Week 2)

| Task | Effort | Description |
|------|--------|-------------|
| Task Types tab | M | List, create, edit, deactivate task types |
| SOP markdown editor | M | Text area or simple markdown editor for SOPs |
| Enhanced create task form | M | Task type dropdown, dynamic param fields |
| Enhanced task detail | S | Show task type, params, result, duration |
| Enhanced task list | S | Task type badge, filter, params preview |
| API client updates | S | Add task type functions to `lib/api/workers.ts` |

### Phase 1c: ClawdBot + Testing (Week 2-3)

| Task | Effort | Description |
|------|--------|-------------|
| Write ClawdBot CLAUDE.md | S | System prompt for Claude Code workers |
| Write initial SOPs | M | reddit_crawl, ai_sentiment, sec_filing, custom |
| End-to-end test | M | Create task → ClawdBot fetches → executes → reports |
| Error handling | S | Worker crash mid-task, network failures |

**S** = Small (< 1 day), **M** = Medium (1-3 days)

**Total estimate: ~3 weeks**

---

## 10. Future Phases

### Phase 2: Task Manager Agent

Add an AI agent (Claude) that orchestrates the queue:
- Auto-assigns tasks to available workers (instead of manual assignment)
- Generates optimized prompts from SOPs + params
- Rewrites prompts on retry (analyzes failure, adjusts instructions)
- Sends Telegram notifications for completions and failures
- Receives commands via Telegram (`/status`, `/add reddit NVDA`)

### Phase 3: Autonomous Scheduling + Metrics

The Task Manager gains the ability to create tasks on its own:
- Scans for tickers with stale data and creates crawl tasks
- Prioritizes based on watchlist popularity, trending tickers, data staleness
- Tracks metrics: tasks by status/type, ticker coverage, worker health
- Daily digest via Telegram
- Budget/rate limit awareness

### Phase 4: Advanced

- Task dependencies / DAG execution ("run sentiment after crawl completes")
- Worker auto-scaling (spin up cloud ClawdBots when queue depth is high)
- Cost tracking per task (LLM tokens, API calls)
- Webhook triggers (SEC filing detected → auto-crawl)

---

## Appendix A: Existing System Files

| File | What |
|------|------|
| `backend/handlers/workers.go` | Admin handlers: ListWorkers, RegisterWorker, DeleteWorker, task CRUD |
| `backend/handlers/worker_api.go` | Worker handlers: GetMyTasks, UpdateStatus, PostUpdate, Heartbeat |
| `backend/migrations/028_add_worker_fields_to_users.sql` | Adds is_worker, last_activity_at to users |
| `backend/migrations/029_create_worker_tasks_tables.sql` | Creates worker_tasks, worker_task_updates tables |
| `app/admin/workers/page.tsx` | Admin UI: Workers & Tasks page |
| `app/worker/dashboard/page.tsx` | Worker dashboard page |
| `lib/api/workers.ts` | Frontend API client for workers/tasks |
| `lib/api/worker-api.ts` | Frontend API client for worker dashboard |

## Appendix B: Example Task Lifecycle

```
Time    Event                                                 Status
-----   -----                                                 ------
10:00   Admin creates "Reddit Crawl for NVDA" in UI           pending
        task_type: reddit_crawl
        params: {ticker: "NVDA", subreddits: ["wsb"], days: 30}
        assigned_to: Genesis

10:02   Genesis fetches tasks: GET /worker/tasks?status=pending
10:02   Genesis reads SOP + params from response
10:02   Genesis: PUT /worker/tasks/{id}/status → in_progress   in_progress
        (started_at = now)

10:03   Genesis: POST /worker/tasks/{id}/updates
        "Starting Arctic Shift API crawl for NVDA in r/wsb..."

10:05   Genesis: POST /worker/tasks/{id}/updates
        "Collected 500 posts so far, continuing..."

10:08   Genesis: POST /worker/tasks/{id}/result
        {posts_collected: 847, posts_stored: 847, ...}
10:08   Genesis: PUT /worker/tasks/{id}/status → completed     completed
        (completed_at = now)

10:08   Admin sees results in task detail view
```
