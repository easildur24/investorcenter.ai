# ClawdBot Task Queue System: Product & Technical Spec

> **Version**: 3.0
> **Date**: February 2026
> **Status**: Draft for Review

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [Problem Statement](#2-problem-statement)
3. [Product Vision](#3-product-vision)
4. [Phase 0: Basic Task Queue & Admin UI](#4-phase-0-basic-task-queue--admin-ui)
5. [Phase 1: Task Manager Agent & ClawdBots](#5-phase-1-task-manager-agent--clawdbots)
6. [Phase 2: Autonomous Agent & Metrics](#6-phase-2-autonomous-agent--metrics)
7. [Technical Architecture](#7-technical-architecture)
8. [Database Schema](#8-database-schema)
9. [API Specification](#9-api-specification)
10. [Task Manager Design](#10-task-manager-design)
11. [ClawdBot Worker Design](#11-clawdbot-worker-design)
12. [Prompt System](#12-prompt-system)
13. [ClawdBot Context Window Management](#13-clawdbot-context-window-management)
14. [Communication Layer](#14-communication-layer)
15. [Security & Auth](#15-security--auth)
16. [Implementation Roadmap](#16-implementation-roadmap)
17. [Open Questions & Decisions](#17-open-questions--decisions)

---

## 1. Executive Summary

### What is the ClawdBot Task Queue?

A distributed task execution system with three layers:

1. **Task Queue** — A PostgreSQL-backed queue where tasks (with full AI prompts) are stored and tracked
2. **Task Manager** — A central Claude agent that orchestrates the queue: assigns tasks to workers, generates prompts, monitors progress, and communicates with humans via Telegram
3. **ClawdBots** — AI agent sessions (Claude Code or similar) running on any machine (your Mac, a friend's machine, cloud VMs) that receive task assignments with prompts, execute them, and report back

The system is built in three phases: **Phase 0** (basic queue + admin UI at `/admin/workers`), **Phase 1** (Task Manager agent + ClawdBots), **Phase 2** (autonomous scheduling + metrics).

The key insight: **ClawdBots are AI agents, not scripts.** Each task carries a full natural-language prompt that tells the ClawdBot exactly what to do. The Task Manager generates these prompts and intelligently assigns them to available workers.

### Why build this?

Today, InvestorCenter's data ingestion runs on fixed Kubernetes CronJobs — they fire on schedules, not on demand. There's no way to:
- Dynamically dispatch ad-hoc crawl tasks ("pull the last 30 days of $NVDA Reddit mentions")
- Distribute work across heterogeneous machines (your laptop, a friend's Mac, a cloud VM)
- Track task-level status with human visibility and intervention
- Have AI agents autonomously manage which tickers need data refreshes

This system bridges the gap between the existing scheduled pipelines and a flexible, human-and-agent-directed task execution platform.

---

## 2. Problem Statement

| Gap | Impact |
|-----|--------|
| CronJobs are schedule-only | Can't do ad-hoc "crawl X for ticker Y right now" |
| No distributed worker pool | All compute runs in the K8s cluster; can't leverage other machines |
| No task-level visibility | Can't tell if a specific ticker's data pull succeeded or failed |
| No human-in-the-loop | No way for humans to reprioritize, retry, or redirect mid-flight |
| No communication channel | System can't proactively notify humans of issues or completions |
| No agent orchestration | Humans must manually decide what to crawl and when |
| Hard-coded pipeline logic | Changing crawl behavior requires code changes and deploys |

---

## 3. Product Vision

```
Phase 0: "Humans create tasks via /admin/workers UI -> manually assign to ClawdBots -> ClawdBots execute"
Phase 1: "Humans add tasks -> Task Manager assigns them with prompts -> ClawdBots execute"
Phase 2: "Task Manager autonomously creates tasks based on data gaps + metrics track everything"
```

### Key Personas

| Persona | What it is | Phase |
|---------|-----------|-------|
| **Human Operator** | You and your team. Add tasks, monitor progress via `/admin/workers` UI. Give instructions via dashboard or Telegram | P0+ |
| **Task Manager** | A central Claude agent. Assigns tasks to workers, generates prompts, monitors health, communicates via Telegram | P1 |
| **ClawdBot Worker** | An AI agent session (Claude Code) on any machine. Receives a prompt, executes it, reports back | P0+ |
| **Autonomous Scheduler** | The Task Manager evolves to also create tasks based on data gaps, staleness, and priority | P2 |

### How It Differs From v1 of This Spec

| Aspect | v1 (Old) | v2 (Current) |
|--------|----------|-------------|
| Workers | Python CLI scripts with coded handlers | AI agents that receive prompts |
| Task dispatch | Workers poll for tasks (pull model) | Task Manager assigns tasks (push model) |
| Task definition | Structured params (`{ticker, subreddits, ...}`) | Full natural-language prompt + structured context |
| Adding new task types | Write a new Python handler class, deploy | Write a new prompt template, no deploy needed |
| Task Manager | Didn't exist (Phase 2 concept) | Central to the system from Phase 1 |

---

## 4. Phase 0: Basic Task Queue & Admin UI

Phase 0 is the MVP: a working task queue with a human-operated admin UI. No Task Manager agent yet — humans create tasks, manually assign them to ClawdBots, and ClawdBots pull their assignments via the API.

### 4.1 What's in Phase 0

- **Task table + CRUD API** — PostgreSQL `tasks` table with all the fields from the schema. Go endpoints for create, list, get, retry, cancel.
- **Worker table + registration** — Workers register and get API keys. Check-in endpoint returns any assigned task.
- **Admin UI at `/admin/workers`** — Reuse the existing admin workers page. Add a task queue tab/section with:
  - Task list view with status filters (pending/running/succeeded/failed)
  - Create task form (select type, fill params, or write a custom prompt)
  - Manual worker assignment (dropdown: "Assign to which ClawdBot?")
  - Task detail view (prompt, result, event timeline)
- **Prompt templates in DB** — Seed initial templates. Admin UI can edit them.
- **ClawdBot CLAUDE.md** — Workers check in, get assigned tasks, execute, report back.

### 4.2 Phase 0 Flow

```
+--------+         +-----------------+         +-------------+
| Human  |         |  /admin/workers |         |  ClawdBot   |
| (You)  |         |  (Next.js UI)   |         |  (Claude    |
|        |         |                 |         |   Code)     |
+---+----+         +--------+--------+         +------+------+
    |                       |                         |
    | 1. Create task        |                         |
    |  (pick type, params)  |                         |
    +---------------------> |                         |
    |                       |                         |
    | 2. Generate prompt    |                         |
    |  (template + params)  |                         |
    |  [done server-side]   |                         |
    |                       |                         |
    | 3. Assign to worker   |                         |
    |  (pick from dropdown) |                         |
    +---------------------> |                         |
    |                       |                         |
    |                       |  4. ClawdBot checks in  |
    |                       | <-----------------------+
    |                       |                         |
    |                       |  5. Return assignment   |
    |                       |    (task + prompt)      |
    |                       | +---------------------> |
    |                       |                         |
    |                       |  6. Execute prompt      |
    |                       |    (crawl, analyze...)  |
    |                       |                         |
    |                       |  7. Report results      |
    |                       | <-----------------------+
    |                       |                         |
    | 8. View results in UI |                         |
    | <---------------------+                         |
```

### 4.3 What Phase 0 Does NOT Include

- No Task Manager agent (humans do the orchestration)
- No Telegram bot (use the web UI only)
- No autonomous scheduling
- No smart routing or prompt rewriting on retry
- No metrics dashboard (just the task list)

### 4.4 Why Phase 0 Matters

- **Validates the schema** — We'll know if the task/worker tables need changes before building the agent
- **Validates the API** — ClawdBots can run end-to-end against real endpoints
- **Validates prompt templates** — We'll see what prompts work and what needs tweaking
- **Low risk** — Just a CRUD API + admin UI page, no AI agent complexity yet
- **Immediately useful** — Even without the Task Manager, humans can dispatch tasks to ClawdBots

---

## 5. Phase 1: Task Manager Agent & ClawdBots

Phase 1 adds the AI brain: a Task Manager agent that automatically assigns tasks, generates prompts, handles failures, and communicates via Telegram. This builds on the Phase 0 infrastructure.

### 5.1 Core Capabilities (Added Over Phase 0)

#### Task Queue
- **Task storage** -- PostgreSQL table with status tracking, prompt, context, and results
- **Status lifecycle** -- `pending` -> `assigned` -> `running` -> `succeeded` / `failed` / `cancelled`
- **Priority** -- Tasks have priority levels (0 = critical, 50 = normal, 100 = low)
- **Retry policy** -- Configurable retries with backoff; Task Manager can rewrite the prompt on retry
- **Tags** -- Ticker symbols, data sources, urgency labels for filtering and metrics

#### Task Manager (Central Agent)
- **Assignment** -- When a ClawdBot comes online ("I'm ready"), the Task Manager picks the best task and sends it the prompt
- **Prompt generation** -- Converts task intent + template into a full executable prompt for the ClawdBot
- **Monitoring** -- Tracks which ClawdBots are online, what they're working on, detects stalls
- **Communication** -- Sends Telegram notifications for completions, failures, and queue status
- **Smart routing** -- Considers worker capabilities, current load, and task requirements when assigning
- **Human interface** -- Receives commands via Telegram (`/status`, `/add reddit NVDA`, `/retry 123`)

#### Human Oversight (Enhanced)
- **Admin UI at `/admin/workers`** -- Same UI from Phase 0, now also shows Task Manager activity and agent decisions
- **Telegram bot** -- Real-time notifications + interactive commands for quick operations
- **Manual override** -- Cancel tasks, reprioritize, add notes, override Task Manager decisions

### 5.2 Task Types

| Task Type | Description | Example Prompt Summary |
|-----------|-------------|----------------------|
| `reddit_crawl` | Pull posts from Reddit for a ticker | "Use Arctic Shift API to fetch posts mentioning $NVDA from r/wallstreetbets. POST results to InvestorCenter API..." |
| `twitter_crawl` | Pull posts/mentions from X | "Search X for $AAPL mentions in the last 7 days. Extract text, engagement metrics..." |
| `news_crawl` | Crawl news articles for a ticker | "Fetch recent news articles about $TSLA from Google News / financial news sites..." |
| `sec_filing_crawl` | Pull specific SEC filings | "Download the latest 10-K filing for $MSFT from SEC EDGAR. Parse key financials..." |
| `ai_sentiment` | Run AI sentiment analysis | "Analyze these 500 Reddit posts about $NVDA. For each, classify sentiment as bullish/bearish/neutral..." |
| `ai_summary` | Generate AI analysis for a ticker | "Given $AAPL's recent financials, news, and social sentiment, write a 500-word investment analysis..." |
| `api_update` | Post processed data to internal API | "POST the following batch of records to the InvestorCenter API at endpoint /api/v1/reddit/posts..." |
| `custom` | Any ad-hoc task defined by prompt | Whatever the human writes |

### 5.3 User Flows (Phase 1)

#### Flow 1: Human Adds a Task via Admin UI
```
1. Human opens /admin/workers -> Task Queue tab
2. Selects task type (e.g., "Reddit Crawl") or "Custom"
3. For typed tasks: fills in parameters (ticker: NVDA, subreddits, date range)
   -> Task Manager auto-generates the full prompt from the template
   For custom: writes the prompt directly
4. Sets priority (normal)
5. Submits -> task status = "pending"
6. Task Manager automatically assigns when a ClawdBot checks in
```

#### Flow 2: Task Manager Assigns Work to a ClawdBot
```
1. A ClawdBot comes online: POST /api/v1/workers/checkin { worker_id, capabilities }
2. Task Manager receives the check-in notification
3. Task Manager queries pending tasks, picks the best match for this worker
4. Task Manager assigns the task: updates status -> "assigned", sets worker_id
5. ClawdBot receives the assignment response containing the full prompt
6. ClawdBot executes the prompt (crawls data, calls APIs, etc.)
7. ClawdBot reports back: POST /api/v1/tasks/{id}/complete { status, result }
8. Task Manager updates status -> "succeeded" / "failed"
9. Task Manager sends Telegram notification
10. If the ClawdBot is still online, it checks in again -> gets next task
```

#### Flow 3: Human Monitors via Telegram
```
Human: /status
Bot:   Queue: 12 pending, 3 running, 45 done, 2 failed
       Workers: macbook-john (busy, 2 tasks), cloud-vm-1 (idle)
Bot:   Retrying task #42: twitter_crawl for $AAPL (was: rate limited)
       Task Manager rewrote prompt with longer delays between requests
```

---

## 6. Phase 2: Autonomous Agent & Metrics

### 6.1 Task Manager Evolves Into Autonomous Scheduler

In Phase 2, the Task Manager gains the ability to **create tasks on its own**, not just assign human-created ones:

- **Data gap detection** -- Scans the database for tickers with stale or missing data (e.g., "AAPL hasn't had Reddit data pulled in 7 days")
- **Priority scoring** -- Prioritizes based on: watchlist popularity, trending tickers, earnings upcoming, data staleness
- **Auto-scheduling** -- Creates tasks automatically ("Pull Reddit data for top 50 watched tickers daily")
- **Adaptive frequency** -- Increases crawl frequency for trending/volatile tickers
- **Budget awareness** -- Respects rate limits and API costs (e.g., don't send 10k posts to ChatGPT in an hour)
- **Self-healing** -- Detects systematic failures and adapts (e.g., if Reddit API is down, backs off; rewrites prompts for tasks that failed)
- **Prompt improvement** -- Learns from failed tasks to generate better prompts over time

#### Agent Decision Loop
```
Every N minutes, the Task Manager:
  1. Query: Which tickers have the most watchers / highest IC score changes?
  2. Query: Which tickers have stale data (>X hours since last crawl)?
  3. Query: Are any tickers trending on social media that we're not tracking?
  4. Score each ticker's "data freshness need"
  5. Generate prompts and create tasks for top-priority tickers
  6. Review failed tasks -- retry with rewritten prompt or flag for human
  7. Send daily digest to Telegram
```

### 6.2 Metrics & Monitoring System

#### Dashboard Metrics
| Metric | Description |
|--------|-------------|
| **Tasks by status** | Pending / Running / Succeeded / Failed / Cancelled over time |
| **Tasks by type** | Breakdown of crawl vs AI vs API update tasks |
| **Ticker coverage** | Which tickers have been processed, when was last update |
| **Worker health** | Which ClawdBots are online, their throughput, error rates |
| **Data freshness** | Per-ticker: how stale is each data source |
| **Throughput** | Tasks completed per hour/day/week |
| **Error rate** | Failure rate by task type and data source |
| **Queue depth** | How many tasks are waiting, average wait time |
| **Agent decisions** | What the Task Manager decided and why (audit log) |

#### Ticker Coverage Matrix
```
           | Reddit | Twitter | News | SEC | Sentiment | IC Score |
  AAPL     |   OK   |   OK    |  OK  |  OK |    OK     |    OK    |  Last: 2h ago
  NVDA     |   OK   |  STALE  |  OK  |  OK |    OK     |    OK    |  Last: 6h ago
  TSLA     |   OK   |  MISS   |  OK  |  OK |   STALE   |    OK    |  Last: 1d ago
  GME      |  MISS  |  MISS   | MISS |  OK |   MISS    |   MISS   |  Never
```

---

## 7. Technical Architecture

### 7.1 Architecture Overview

```
+--------------------------------------------------------------------+
|                        HUMAN LAYER                                  |
|                                                                     |
|   +---------------+    +---------------+    +-------------------+   |
|   | Admin Web UI  |    |   Telegram    |    |  Slack (optional) |   |
|   | (Next.js)     |    |   Bot         |    |  Bot              |   |
|   +-------+-------+    +-------+-------+    +--------+----------+   |
|           |                     |                     |              |
+-----------+---------------------+---------------------+--------------+
            |                     |                     |
            v                     v                     v
+--------------------------------------------------------------------+
|                    TASK MANAGER LAYER                                |
|                    (Claude Agent)                                    |
|                                                                     |
|   The brain of the system. A Claude agent that:                     |
|   - Receives new tasks from humans (dashboard/Telegram)             |
|   - Generates prompts from templates + task parameters              |
|   - Assigns tasks to available ClawdBots                            |
|   - Monitors progress and handles failures                          |
|   - Communicates status via Telegram                                |
|   - [P2] Autonomously creates tasks based on data gaps              |
|                                                                     |
+---+-----------------------------+-----------------------------------+
    |                             |
    v                             v
+------------------+    +------------------+
| PostgreSQL       |    | Redis            |
| - tasks table    |    | - worker status  |
| - workers table  |    | - rate limits    |
| - task_events    |    | - session cache  |
| - prompt_templates|   |                  |
+------------------+    +------------------+
    ^                             ^
    |                             |
+---+-----------------------------+-----------------------------------+
|                      WORKER LAYER                                    |
|                                                                      |
|   +-------------+  +-------------+  +--------------+                 |
|   | ClawdBot #1 |  | ClawdBot #2 |  | ClawdBot #3  |    ...         |
|   | (Your Mac)  |  | (Friend's)  |  | (Cloud VM)   |                |
|   |             |  |             |  |              |                |
|   | AI agent    |  | AI agent    |  | AI agent     |                |
|   | session     |  | session     |  | session      |                |
|   | receives    |  | receives    |  | receives     |                |
|   | prompt,     |  | prompt,     |  | prompt,      |                |
|   | executes    |  | executes    |  | executes     |                |
|   +------+------+  +------+------+  +------+-------+                |
|          |                |                |                         |
|          v                v                v                         |
|   +------------------------------------------------------------+    |
|   | External Data Sources                                       |    |
|   | Reddit/Arctic Shift, X/Twitter, SEC EDGAR, News sites,     |    |
|   | ChatGPT/Claude API, InvestorCenter Internal API             |    |
|   +------------------------------------------------------------+    |
+----------------------------------------------------------------------+
```

### 6.2 Why This Architecture?

**Task Manager as a Claude Agent** (not a traditional service):
- Can generate and refine prompts dynamically -- the core differentiator
- Can reason about task failures and rewrite prompts for retries
- Can understand human instructions in natural language via Telegram
- In Phase 2, can make intelligent autonomous decisions about what to crawl
- The "business logic" lives in prompts, not code

**PostgreSQL as the queue** (not RabbitMQ/Redis Streams/SQS):
- Already have PostgreSQL, no new infrastructure
- Task volume is low-to-moderate (hundreds/day, not millions)
- Need durable, queryable task history for the dashboard and metrics
- `SELECT ... FOR UPDATE SKIP LOCKED` gives us reliable concurrent task claiming
- If we outgrow this, migrating to SQS later is straightforward

**Push model via Task Manager** (not worker polling):
- ClawdBots are session-based AI agents, not persistent daemons
- Task Manager can make intelligent routing decisions
- Can batch related tasks into a single prompt ("crawl Reddit for these 5 tickers")
- Task Manager knows worker context -- route based on capabilities, load, past success
- Simpler from the ClawdBot's perspective -- just receive prompt, execute, report

**Redis for coordination**:
- Worker online/offline status with TTL-based expiry
- Rate limit counters per data source (Reddit API: 60 req/min, etc.)
- Fast ephemeral state -- system works without Redis (falls back to DB)

### 6.3 Integration with Existing System

The task queue complements, not replaces, the existing CronJob pipelines:

| Concern | Current (CronJobs) | Task Queue |
|---------|-------------------|------------|
| Scheduled data pulls | Keep as-is | Not needed |
| Ad-hoc crawl requests | Not possible | Primary use case |
| Distributed execution | K8s only | Any machine |
| Task-level tracking | Job-level only | Per-task status |
| Human intervention | None | Dashboard + messaging |
| Changing behavior | Code deploy | Edit a prompt |

Over time, some CronJobs could be migrated to agent-scheduled tasks (Phase 2), but the CronJobs remain as a reliable baseline.

---

## 7. Database Schema

### 7.1 Core Tables

```sql
-- Prompt templates for each task type
CREATE TABLE prompt_templates (
    id              SERIAL PRIMARY KEY,
    task_type       VARCHAR(50) NOT NULL UNIQUE,   -- 'reddit_crawl', 'ai_sentiment', etc.
    template        TEXT NOT NULL,                  -- The prompt template with {{variables}}
    system_context  TEXT,                           -- Background context appended to every prompt
    version         INTEGER NOT NULL DEFAULT 1,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- The central task table
CREATE TABLE tasks (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Task definition
    task_type       VARCHAR(50) NOT NULL,           -- 'reddit_crawl', 'ai_sentiment', 'custom'
    priority        SMALLINT NOT NULL DEFAULT 50,   -- 0 (highest) to 100 (lowest)

    -- The prompt: this is what the ClawdBot actually receives and executes
    prompt          TEXT NOT NULL,                   -- Full natural-language prompt for the ClawdBot
    prompt_context  JSONB,                           -- Structured context (API keys ref, endpoints, schemas)

    -- Original intent (what the human/agent requested, before prompt generation)
    intent          TEXT,                            -- "Pull Reddit data for NVDA, last 30 days"
    params          JSONB,                           -- Structured params used to generate the prompt

    -- Lifecycle
    status          VARCHAR(20) NOT NULL DEFAULT 'pending',
                    -- pending, assigned, running, succeeded, failed, cancelled
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    assigned_at     TIMESTAMPTZ,
    started_at      TIMESTAMPTZ,
    completed_at    TIMESTAMPTZ,

    -- Worker assignment (set by Task Manager)
    worker_id       UUID REFERENCES workers(id),

    -- Results
    result          JSONB,                           -- Success: summary data. Failure: error details
    result_summary  TEXT,                            -- Human-readable summary from ClawdBot
    error_message   TEXT,

    -- Retry
    retry_count     SMALLINT NOT NULL DEFAULT 0,
    max_retries     SMALLINT NOT NULL DEFAULT 3,
    previous_prompt TEXT,                            -- Prompt from previous attempt (if retried)

    -- Metadata
    created_by      VARCHAR(100) NOT NULL,           -- 'human:username' or 'agent:task_manager'
    tags            TEXT[],                           -- ['NVDA', 'reddit', 'urgent']
    notes           TEXT,                             -- Human-added notes
    parent_task_id  UUID REFERENCES tasks(id),       -- For sub-task chains

    CONSTRAINT valid_status CHECK (status IN
        ('pending', 'assigned', 'running', 'succeeded', 'failed', 'cancelled'))
);

CREATE INDEX idx_tasks_pending ON tasks (priority, created_at) WHERE status = 'pending';
CREATE INDEX idx_tasks_status ON tasks (status);
CREATE INDEX idx_tasks_worker ON tasks (worker_id) WHERE status IN ('assigned', 'running');
CREATE INDEX idx_tasks_tags ON tasks USING GIN (tags);
CREATE INDEX idx_tasks_created_at ON tasks (created_at);
CREATE INDEX idx_tasks_type_status ON tasks (task_type, status);

-- Worker registration
CREATE TABLE workers (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            VARCHAR(100) NOT NULL,           -- 'macbook-pro-john', 'cloud-vm-1'
    capabilities    TEXT[] NOT NULL,                  -- ['reddit_crawl', 'ai_sentiment']
    max_concurrency SMALLINT NOT NULL DEFAULT 1,     -- How many tasks simultaneously

    -- Status
    status          VARCHAR(20) NOT NULL DEFAULT 'offline',
                    -- online, busy, offline
    last_checkin    TIMESTAMPTZ,
    current_tasks   SMALLINT NOT NULL DEFAULT 0,

    -- Auth
    api_key_hash    VARCHAR(256) NOT NULL,            -- Hashed API key for auth

    -- Metadata
    registered_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    metadata        JSONB,                            -- OS, IP, version, etc.

    CONSTRAINT valid_worker_status CHECK (status IN ('online', 'busy', 'offline'))
);

-- Task execution log (append-only audit trail)
CREATE TABLE task_events (
    id              BIGSERIAL PRIMARY KEY,
    task_id         UUID NOT NULL REFERENCES tasks(id),
    event_type      VARCHAR(30) NOT NULL,
                    -- 'created', 'prompt_generated', 'assigned', 'started',
                    -- 'progress', 'succeeded', 'failed', 'retried', 'cancelled'
    worker_id       UUID REFERENCES workers(id),
    details         JSONB,                            -- Event-specific data
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_task_events_task ON task_events (task_id, created_at);

-- Notification log
CREATE TABLE task_notifications (
    id              BIGSERIAL PRIMARY KEY,
    task_id         UUID REFERENCES tasks(id),
    channel         VARCHAR(20) NOT NULL,             -- 'telegram', 'slack', 'in_app'
    message         TEXT NOT NULL,
    sent_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    delivered       BOOLEAN DEFAULT FALSE
);
```

### 7.2 Phase 2 Additional Tables

```sql
-- Agent decisions audit log
CREATE TABLE agent_decisions (
    id              BIGSERIAL PRIMARY KEY,
    decision_type   VARCHAR(50) NOT NULL,             -- 'create_task', 'retry', 'escalate', 'rewrite_prompt'
    reasoning       TEXT NOT NULL,                    -- Why the Task Manager made this decision
    tickers         TEXT[],
    tasks_created   UUID[],
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Ticker data freshness tracking
CREATE TABLE ticker_data_freshness (
    ticker          VARCHAR(10) NOT NULL,
    data_source     VARCHAR(50) NOT NULL,             -- 'reddit', 'twitter', 'news', 'sec'
    last_success_at TIMESTAMPTZ,
    last_attempt_at TIMESTAMPTZ,
    last_error      TEXT,
    record_count    INTEGER DEFAULT 0,
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (ticker, data_source)
);

CREATE INDEX idx_freshness_stale ON ticker_data_freshness (last_success_at)
    WHERE last_success_at < NOW() - INTERVAL '24 hours';

-- Daily metrics snapshots
CREATE TABLE task_metrics_daily (
    date            DATE NOT NULL,
    task_type       VARCHAR(50) NOT NULL,
    tasks_created   INTEGER DEFAULT 0,
    tasks_succeeded INTEGER DEFAULT 0,
    tasks_failed    INTEGER DEFAULT 0,
    avg_duration_ms INTEGER,
    p95_duration_ms INTEGER,
    unique_tickers  INTEGER DEFAULT 0,
    PRIMARY KEY (date, task_type)
);
```

---

## 8. API Specification

### 8.1 Task API

All endpoints under `/api/v1/tasks`. Admin-authenticated unless noted.

#### Create Task
```
POST /api/v1/tasks
Authorization: Bearer <admin_jwt>

{
    "task_type": "reddit_crawl",
    "priority": 30,
    "params": {
        "ticker": "NVDA",
        "subreddits": ["wallstreetbets", "stocks", "investing"],
        "date_range": {"from": "2026-01-01", "to": "2026-02-08"},
        "limit": 1000
    },
    "tags": ["NVDA", "reddit"],
    "notes": "Need fresh Reddit data for NVDA earnings analysis"
}

Response 201:
{
    "id": "a1b2c3d4-...",
    "status": "pending",
    "prompt": "<auto-generated from template>",
    "created_at": "2026-02-08T10:00:00Z"
}
```

Note: The `prompt` field is auto-generated by the Task Manager from the prompt template + params. Humans can also provide a raw `prompt` directly for `custom` type tasks.

#### Create Custom Task (with direct prompt)
```
POST /api/v1/tasks
Authorization: Bearer <admin_jwt>

{
    "task_type": "custom",
    "priority": 20,
    "prompt": "Go to the SEC EDGAR full-text search and find all 8-K filings
               for $NVDA filed in the last 30 days that mention 'AI' or
               'artificial intelligence'. For each filing, extract the filing
               date, form type, and the relevant paragraph. POST results to
               https://api.investorcenter.ai/api/v1/sec/filings with format:
               {ticker, filing_date, form_type, excerpt, url}",
    "tags": ["NVDA", "sec", "custom"]
}
```

#### Create Batch Tasks
```
POST /api/v1/tasks/batch
Authorization: Bearer <admin_jwt>

{
    "task_type": "reddit_crawl",
    "tickers": ["NVDA", "AAPL", "TSLA", "MSFT", "GOOGL"],
    "params": {
        "subreddits": ["wallstreetbets", "stocks"],
        "date_range": {"from": "2026-01-01", "to": "2026-02-08"}
    }
}

Response 201:
{
    "created": 5,
    "task_ids": ["a1b2...", "c3d4...", "e5f6...", "g7h8...", "i9j0..."]
}
```

#### List Tasks
```
GET /api/v1/tasks?status=failed&task_type=reddit_crawl&ticker=NVDA&page=1&limit=20
Authorization: Bearer <admin_jwt>

Response 200:
{
    "tasks": [
        {
            "id": "a1b2c3d4-...",
            "task_type": "reddit_crawl",
            "status": "failed",
            "intent": "Pull Reddit data for NVDA, last 30 days",
            "tags": ["NVDA", "reddit"],
            "worker_id": "w-abc123",
            "error_message": "Rate limited by Reddit API",
            "retry_count": 1,
            "created_at": "2026-02-08T10:00:00Z",
            "completed_at": "2026-02-08T10:05:00Z"
        }
    ],
    "total": 42,
    "page": 1,
    "pages": 3
}
```

#### Get Task Detail (includes full prompt)
```
GET /api/v1/tasks/{id}
Authorization: Bearer <admin_jwt>

Response 200:
{
    "id": "a1b2c3d4-...",
    "task_type": "reddit_crawl",
    "status": "succeeded",
    "prompt": "<full prompt text>",
    "intent": "Pull Reddit data for NVDA",
    "params": { ... },
    "result": { "posts_collected": 847 },
    "result_summary": "Collected 847 posts from 3 subreddits...",
    "events": [
        {"event_type": "created", "created_at": "..."},
        {"event_type": "prompt_generated", "created_at": "..."},
        {"event_type": "assigned", "worker_id": "w-abc", "created_at": "..."},
        {"event_type": "succeeded", "created_at": "..."}
    ]
}
```

#### Retry Task
```
POST /api/v1/tasks/{id}/retry
Authorization: Bearer <admin_jwt>

Response 200:
{
    "id": "a1b2c3d4-...",
    "status": "pending",
    "retry_count": 2,
    "prompt": "<potentially rewritten prompt by Task Manager>"
}
```

#### Cancel Task
```
POST /api/v1/tasks/{id}/cancel
Authorization: Bearer <admin_jwt>

Response 200: { "cancelled": true }
```

### 8.2 Worker API

#### Check In (Worker announces availability, gets assignment)
```
POST /api/v1/workers/checkin
Authorization: X-Worker-Key <api_key>

{
    "worker_id": "w-abc123",
    "capabilities": ["reddit_crawl", "ai_sentiment"],
    "status": "online"
}

Response 200 (task assigned):
{
    "assignment": {
        "task_id": "a1b2c3d4-...",
        "task_type": "reddit_crawl",
        "prompt": "You are a data collection agent for InvestorCenter.ai...",
        "prompt_context": {
            "api_base_url": "https://api.investorcenter.ai",
            "api_endpoints": { ... },
            "data_formats": { ... }
        }
    }
}

Response 200 (no tasks available):
{
    "assignment": null,
    "message": "No pending tasks matching your capabilities",
    "retry_after_seconds": 30
}
```

This is the key endpoint. The ClawdBot checks in, and the Task Manager either assigns it a task (with the full prompt) or tells it to check back later.

#### Report Completion
```
POST /api/v1/tasks/{id}/complete
Authorization: X-Worker-Key <api_key>

{
    "worker_id": "w-abc123",
    "status": "succeeded",
    "result": {
        "posts_collected": 847,
        "posts_stored": 847,
        "api_calls_made": 12,
        "duration_seconds": 45
    },
    "result_summary": "Successfully collected 847 Reddit posts about $NVDA from r/wallstreetbets, r/stocks, and r/investing spanning Jan 2 - Feb 7 2026. All posts POSTed to InvestorCenter API."
}

Response 200: { "acknowledged": true, "next_checkin": true }
```

#### Report Failure
```
POST /api/v1/tasks/{id}/complete
Authorization: X-Worker-Key <api_key>

{
    "worker_id": "w-abc123",
    "status": "failed",
    "error_message": "Arctic Shift API returned 429 Too Many Requests after 342 posts",
    "result": {
        "posts_collected": 342,
        "posts_stored": 342
    },
    "result_summary": "Partially completed. Got 342/~800 posts before hitting rate limit."
}

Response 200: { "acknowledged": true }
```

#### Register Worker (one-time setup)
```
POST /api/v1/workers/register
Authorization: Bearer <admin_jwt>

{
    "name": "macbook-pro-john",
    "capabilities": ["reddit_crawl", "twitter_crawl", "ai_sentiment", "custom"],
    "max_concurrency": 1
}

Response 201:
{
    "id": "w-abc123",
    "api_key": "cwb_live_xxxxxxxxxxxxxxxx"   // Only shown once
}
```

#### List Workers
```
GET /api/v1/workers
Authorization: Bearer <admin_jwt>

Response 200:
{
    "workers": [
        {
            "id": "w-abc123",
            "name": "macbook-pro-john",
            "status": "busy",
            "capabilities": ["reddit_crawl", "ai_sentiment"],
            "current_tasks": 1,
            "last_checkin": "2026-02-08T10:04:55Z"
        }
    ]
}
```

### 8.3 Prompt Template API

#### List Templates
```
GET /api/v1/prompt-templates
Authorization: Bearer <admin_jwt>
```

#### Update Template
```
PUT /api/v1/prompt-templates/{task_type}
Authorization: Bearer <admin_jwt>

{
    "template": "You are a data collection agent...",
    "system_context": "InvestorCenter API base URL: ..."
}
```

---

## 9. Task Manager Design

### 9.1 What is the Task Manager?

The Task Manager is a **Claude agent** (using Anthropic's Agent SDK or similar) that serves as the brain of the system. It runs as a persistent service and handles:

1. **Prompt generation** -- When a new task is created with params, the Task Manager generates the full prompt
2. **Task assignment** -- When a ClawdBot checks in, the Task Manager decides which task to assign
3. **Failure handling** -- When a task fails, the Task Manager decides whether to retry (possibly with a rewritten prompt) or escalate
4. **Telegram interface** -- Receives and responds to human commands via Telegram
5. **[Phase 2] Autonomous scheduling** -- Creates tasks based on data freshness analysis

### 9.2 Task Manager as a Service

The Task Manager runs as a long-lived process (deployed in K8s or on a dedicated machine):

```
task-manager/
    main.py                 -- Entry point, starts the agent loop
    agent.py                -- Claude agent with tool definitions
    tools/
        task_queue.py       -- Tools: query tasks, create tasks, assign tasks, update status
        worker_registry.py  -- Tools: list workers, check health, track capabilities
        prompt_engine.py    -- Tools: load templates, generate prompts, rewrite failed prompts
        telegram.py         -- Tools: send messages, receive commands
        database.py         -- Tools: query freshness data, metrics [P2]
    templates/
        reddit_crawl.md     -- Prompt template for Reddit crawling
        twitter_crawl.md    -- Prompt template for Twitter crawling
        ai_sentiment.md     -- Prompt template for sentiment analysis
        ...
```

### 9.3 Task Manager Agent Loop

```
The Task Manager continuously:

1. CHECK for new tasks needing prompt generation
   -> Generate prompts from templates + params
   -> Store prompt on the task

2. CHECK for worker check-ins (via webhook or polling the API)
   -> Match worker capabilities to pending tasks
   -> Assign highest-priority matching task
   -> Include full prompt in the assignment

3. CHECK for completed tasks
   -> Update status, log events
   -> Update ticker_data_freshness [P2]
   -> Send Telegram notification

4. CHECK for failed tasks
   -> Decide: retry with rewritten prompt? Escalate to human?
   -> If retry: analyze the error, rewrite prompt to avoid the issue
   -> If escalate: send Telegram message asking human for guidance

5. CHECK for stale workers (no check-in in >5 minutes)
   -> Mark worker as offline
   -> Release any assigned tasks back to pending

6. [P2] CHECK for data gaps
   -> Query ticker_data_freshness
   -> Create tasks for stale tickers
   -> Log decisions to agent_decisions table

7. RESPOND to Telegram commands
   -> /status, /add, /retry, /workers, etc.
```

### 9.4 Assignment Logic

When a ClawdBot checks in, the Task Manager decides what to assign:

```
Input: worker {id, capabilities, max_concurrency, current_tasks}

1. If worker.current_tasks >= worker.max_concurrency: return null (worker is full)

2. Query pending tasks matching worker.capabilities, ordered by:
   - priority ASC (lower = more urgent)
   - created_at ASC (older first, FIFO within same priority)

3. Smart considerations:
   - If worker recently succeeded at task_type X, prefer assigning more of X (warm cache)
   - If a task has failed before, prefer assigning to a different worker
   - [P2] Batch multiple small tasks into one prompt if beneficial

4. Assign task: set status='assigned', worker_id, assigned_at

5. Return task with full prompt to worker
```

### 9.5 Prompt Rewriting on Retry

When a task fails, the Task Manager can analyze the error and rewrite the prompt:

```
Example:
  Original prompt: "Fetch Reddit posts for $NVDA from Arctic Shift API..."
  Error: "429 Too Many Requests after 342 posts"

  Task Manager reasoning:
  "The task hit a rate limit. I should rewrite the prompt to add delays
   between requests and reduce batch size."

  Rewritten prompt: "Fetch Reddit posts for $NVDA from Arctic Shift API.
   IMPORTANT: Add a 2-second delay between API requests to avoid rate
   limiting. Fetch in batches of 50 (not 100). If you get a 429 response,
   wait 60 seconds before retrying..."
```

This is a key advantage of the prompt-based architecture -- the Task Manager can adapt its instructions to the ClawdBot without any code changes.

---

## 10. ClawdBot Worker Design

### 10.1 What is a ClawdBot?

A ClawdBot is simply a **Claude Code session** (or similar AI agent) running on any machine. It does not need special software beyond the Claude CLI. The workflow:

```
1. Human starts Claude Code on their machine
2. Claude Code reads the CLAUDE.md in the clawdbot project directory
3. CLAUDE.md contains instructions: "You are a ClawdBot worker. Check in
   with the Task Manager at POST /api/v1/workers/checkin..."
4. Claude Code checks in, receives a task with a prompt
5. Claude Code executes the prompt (runs bash commands, writes scripts, calls APIs)
6. Claude Code reports results back via the API
7. Claude Code checks in again for the next task, or session ends
```

### 10.2 ClawdBot CLAUDE.md (System Prompt)

Each machine running a ClawdBot has a project directory with a `CLAUDE.md`:

```markdown
# ClawdBot Worker

You are a ClawdBot worker for InvestorCenter.ai. Your job is to execute
data collection and analysis tasks assigned by the Task Manager.

## Setup
- Worker ID: {{WORKER_ID}}
- API Key: stored in env var CLAWDBOT_API_KEY
- API Base: https://api.investorcenter.ai/api/v1

## Workflow
1. Check in with the Task Manager:
   curl -X POST $API_BASE/workers/checkin \
     -H "X-Worker-Key: $CLAWDBOT_API_KEY" \
     -d '{"worker_id": "{{WORKER_ID}}", "capabilities": [...], "status": "online"}'

2. If you receive a task assignment, execute the prompt in the assignment.
   The prompt will tell you exactly what to do.

3. When done, report results:
   curl -X POST $API_BASE/tasks/{task_id}/complete \
     -H "X-Worker-Key: $CLAWDBOT_API_KEY" \
     -d '{"worker_id": "{{WORKER_ID}}", "status": "succeeded", ...}'

4. Check in again for the next task.

## Important
- Always follow the prompt instructions exactly
- Report failures honestly with error details
- Include a result_summary in natural language
- Do not modify the InvestorCenter codebase
- You have access to: bash, curl, python, node on this machine
```

### 10.3 What Makes This Powerful

- **Zero deployment** -- No special software to install. Just Claude Code + a CLAUDE.md
- **Infinitely flexible** -- Any task that can be described in a prompt can be executed
- **Self-healing** -- If a task fails, the Task Manager rewrites the prompt for the retry
- **New task types for free** -- Adding a new task type is just adding a new prompt template
- **Human-readable** -- Every task has a readable prompt and result summary

---

## 11. Prompt System

### 11.1 Prompt Templates

Each task type has a prompt template stored in the database (`prompt_templates` table). Templates use `{{variable}}` placeholders.

#### Example: reddit_crawl template

```
You are a data collection agent for InvestorCenter.ai. Your task is to
collect Reddit posts about {{ticker}} ({{company_name}}).

## Your Task
Search for Reddit posts mentioning {{ticker}} or {{company_name}} in the
following subreddits: {{subreddits}}

Date range: {{date_from}} to {{date_to}}
Maximum posts: {{limit}}

## Data Source
Use the Arctic Shift API (https://arctic-shift.photon-reddit.com/api).
- Endpoint: /api/posts/search
- Params: subreddit, q (search query), after (epoch), before (epoch), limit

## Output Format
For each post, collect:
- external_id (Reddit post ID)
- subreddit
- title
- body (selftext)
- score (upvotes)
- num_comments
- author
- created_utc
- url

## Storing Results
POST each batch of posts (up to 100 at a time) to:
  {{api_base_url}}/api/v1/reddit/posts

Request body:
{
  "posts": [
    {
      "external_id": "...",
      "subreddit": "...",
      "title": "...",
      "body": "...",
      "score": 0,
      "num_comments": 0,
      "author": "...",
      "created_utc": "2026-01-15T10:00:00Z",
      "url": "...",
      "tickers": ["{{ticker}}"]
    }
  ]
}

Headers: { "X-Worker-Key": "$CLAWDBOT_API_KEY" }

## Completion
When done, report back with:
- Total posts collected
- Date range of posts actually found
- Any errors or issues encountered
```

### 11.2 Prompt Generation Flow

```
1. Human creates task: { task_type: "reddit_crawl", params: { ticker: "NVDA", ... } }

2. Task Manager loads the reddit_crawl template from prompt_templates table

3. Task Manager fills in variables:
   {{ticker}} -> "NVDA"
   {{company_name}} -> "NVIDIA Corporation" (looked up from stocks table)
   {{subreddits}} -> "wallstreetbets, stocks, investing"
   {{date_from}} -> "2026-01-01"
   {{date_to}} -> "2026-02-08"
   {{limit}} -> "1000"
   {{api_base_url}} -> "https://api.investorcenter.ai"

4. Task Manager stores the generated prompt on the task record

5. When a ClawdBot checks in, it receives this fully-rendered prompt
```

### 11.3 Why Prompts in the Database?

| Benefit | Explanation |
|---------|-------------|
| **No deploys to change behavior** | Edit the template in the DB or admin UI, takes effect immediately |
| **Audit trail** | Every task has the exact prompt that was executed, stored forever |
| **Retry with improvement** | Task Manager can rewrite the prompt for retries based on error analysis |
| **Human-readable** | Anyone can read the prompt and understand exactly what happened |
| **Custom tasks** | Humans can write arbitrary prompts for one-off tasks |
| **A/B testing** | Can test different prompt versions and compare success rates |

---

## 12. Communication Layer

### 12.1 Telegram Bot (Recommended for Phase 1)

**Why Telegram over Slack?**
- Free, no workspace needed, works on mobile
- Simple Bot API -- 5 minutes to set up
- Supports group chats (team notifications) and DMs (personal alerts)
- Rich formatting (markdown, inline buttons for quick actions)

#### Notification Types

| Event | Example Message |
|-------|---------|
| Task completed | `reddit_crawl for $NVDA completed -- 847 posts (45s)` |
| Task failed | `twitter_crawl for $AAPL failed -- Rate limit exceeded (retry 2/3)` |
| Worker online | `ClawdBot "macbook-john" connected (reddit, ai_sentiment)` |
| Worker offline | `ClawdBot "macbook-john" went offline (last seen 5m ago)` |
| Queue stalled | `12 tasks pending, 0 workers online` |
| Daily digest (P2) | `Daily: 200 tasks, 195 done, 5 failed -- Top: NVDA(45), AAPL(38)` |

#### Interactive Commands (via Telegram)

```
/status           -- Current queue status (pending/running/done/failed)
/workers          -- List online workers
/queue            -- Show pending tasks
/retry <task_id>  -- Retry a failed task
/add reddit NVDA  -- Quick-add a Reddit crawl task for NVDA
/add batch reddit NVDA,AAPL,TSLA  -- Batch add
/cancel <task_id> -- Cancel a task
/pause            -- Pause all workers (stop assigning new tasks)
/resume           -- Resume
```

The Task Manager handles all Telegram interactions -- it IS the Telegram bot.

### 12.2 Telegram Integration in Task Manager

The Task Manager includes Telegram as one of its tools:

```python
# Tool definition for the Task Manager agent
tools = [
    {
        "name": "send_telegram",
        "description": "Send a message to the InvestorCenter Telegram group",
        "input_schema": {
            "type": "object",
            "properties": {
                "message": {"type": "string"}
            }
        }
    },
    {
        "name": "check_telegram_messages",
        "description": "Check for new commands from the Telegram group",
        "input_schema": { ... }
    }
]
```

---

## 13. Security & Auth

### 13.1 Authentication Model

| Actor | Auth Method |
|-------|-------------|
| Admin (web UI) | JWT (existing auth system) |
| ClawdBot workers | API key per worker (`X-Worker-Key` header) |
| Task Manager | Internal service key |
| Telegram bot | Bot token + webhook secret |

### 13.2 Worker Security

- Each worker gets a unique API key on registration (shown once, stored hashed)
- Workers can ONLY: check in, report task completion
- Workers CANNOT: create tasks, cancel tasks, see other workers' tasks, list all tasks
- API keys can be revoked from admin dashboard
- Rate limiting on check-in endpoint

### 13.3 Prompt Security

- Prompts may reference API endpoints but never contain raw credentials
- ClawdBots use their own local environment variables for data source auth (Reddit API key, etc.)
- The `prompt_context` field provides endpoint URLs and data schemas, not secrets

---

## 14. Implementation Roadmap

### Phase 1a: Core Infrastructure (Week 1-2)

| Task | Effort | Description |
|------|--------|-------------|
| DB migration | S | Create tasks, workers, task_events, prompt_templates, task_notifications tables |
| Go API: Task CRUD | M | Create, list, get, cancel, retry endpoints |
| Go API: Worker endpoints | M | Register, check-in (with assignment logic), complete |
| Prompt template CRUD | S | API to manage prompt templates |
| Stale worker reaper | S | Background goroutine to mark offline workers, release assigned tasks |

### Phase 1b: Task Manager Agent (Week 2-3)

| Task | Effort | Description |
|------|--------|-------------|
| Task Manager scaffold | M | Python service using Claude Agent SDK, main loop |
| Prompt generation tool | M | Load templates, fill variables, store on task |
| Assignment logic | M | Match worker capabilities to tasks, smart routing |
| Failure handling | M | Analyze errors, decide retry vs escalate, rewrite prompts |
| Telegram integration | M | Send notifications, receive and parse commands |

### Phase 1c: ClawdBot & Dashboard (Week 3-4)

| Task | Effort | Description |
|------|--------|-------------|
| ClawdBot CLAUDE.md | S | Write the system prompt for ClawdBot workers |
| First prompt templates | M | reddit_crawl, ai_sentiment, api_update templates |
| Test end-to-end flow | M | Human creates task -> Task Manager assigns -> ClawdBot executes |
| Admin dashboard: task list | M | Next.js page: list tasks, filter, status badges |
| Admin dashboard: create task | M | Form to create tasks by type or custom prompt |
| Admin dashboard: task detail | S | View prompt, results, event timeline |

### Phase 1d: Polish (Week 4-5)

| Task | Effort | Description |
|------|--------|-------------|
| More prompt templates | M | twitter_crawl, news_crawl, sec_filing_crawl, ai_summary |
| Batch task creation | S | API + UI for creating tasks for multiple tickers at once |
| Telegram interactive commands | M | /status, /add, /retry, /workers, /cancel |
| Error handling edge cases | M | Worker crashes mid-task, network failures, duplicate claims |

### Phase 2a: Metrics (Week 6-7)

| Task | Effort | Description |
|------|--------|-------------|
| ticker_data_freshness tracking | S | Update on task completion |
| task_metrics_daily aggregation | S | Nightly rollup job |
| Metrics API endpoints | M | Task stats, coverage matrix, worker health |
| Admin dashboard: metrics page | M | Charts, coverage matrix, worker health cards |

### Phase 2b: Autonomous Scheduling (Week 7-9)

| Task | Effort | Description |
|------|--------|-------------|
| Data gap detection | M | Task Manager queries freshness, identifies stale tickers |
| Priority scoring | M | Rank tickers by watchlist popularity, staleness, trending |
| Auto task creation | M | Task Manager creates tasks for top-priority tickers |
| Agent decisions audit log | S | Log all autonomous decisions with reasoning |
| Budget/rate limit awareness | M | Don't exceed API costs or rate limits |
| Daily digest via Telegram | S | Summary of autonomous actions and results |

**S** = Small (< 1 day), **M** = Medium (1-3 days)

---

## 15. Open Questions & Decisions

### Needs Decision

| # | Question | Options | Recommendation |
|---|----------|---------|----------------|
| 1 | **Task Manager hosting** | K8s pod vs dedicated machine vs serverless | **K8s pod** -- consistent with existing infra, can use existing DB/Redis |
| 2 | **Task Manager SDK** | Anthropic Agent SDK vs custom agent loop | **Agent SDK** if available, otherwise custom loop with Claude API |
| 3 | **Primary notification channel** | Telegram vs Slack vs Discord | **Telegram** -- free, simple API, mobile-friendly |
| 4 | **ClawdBot agent runtime** | Claude Code vs custom agent vs any LLM | **Claude Code** for now, design API to be agent-agnostic |
| 5 | **Twitter/X data source** | Official API ($100/mo) vs scraping vs third-party | Needs research -- X API pricing may be prohibitive |
| 6 | **AI provider for analysis tasks** | Claude vs ChatGPT vs Gemini | Support multiple; ClawdBot uses whatever is in its local env |
| 7 | **Agent autonomy level (P2)** | Fully autonomous vs human-approval | Start with human-approval for expensive tasks, auto for crawls |

### Future Considerations

- **Task dependencies** -- "Run sentiment analysis after crawl completes" (DAG execution)
- **Cost tracking** -- Track API costs per task (LLM tokens, data source API calls)
- **Multi-tenant** -- If other teams want to use the queue
- **Webhook triggers** -- External events trigger tasks (e.g., SEC filing detected -> auto-crawl)
- **Worker auto-scaling** -- Spin up cloud ClawdBots when queue depth is high
- **Prompt versioning** -- Track which prompt version produced which results

---

## Appendix A: Comparison to Existing CronJobs

| CronJob | Keep or Migrate? | Notes |
|---------|-----------------|-------|
| `polygon-ticker-update` | **Keep** | Critical daily price data, reliable schedule |
| `polygon-volume-update` | **Keep** | Same as above |
| `sec-filing-daily-update` | **Keep** (migrate P2) | Could become agent-scheduled |
| `reddit-post-collector` | **Migrate P2** | Agent can schedule adaptively per ticker |
| `reddit-sentiment-collector` | **Migrate P2** | Run as follow-up task after crawl |
| `reddit-heatmap` | **Keep** | Simple aggregation, schedule is fine |
| `screener-refresh` | **Keep** | Daily aggregate, no benefit from queue |

## Appendix B: Example Task Lifecycle

```
 Time    Event                                           Status
 -----   -----                                           ------
 10:00   Human creates task via dashboard                 pending
 10:00   Task Manager generates prompt from template      pending (prompt set)
 10:02   ClawdBot "macbook-john" checks in                --
 10:02   Task Manager assigns task to macbook-john        assigned
 10:02   ClawdBot starts executing prompt                 running
 10:05   ClawdBot reports success (847 posts collected)   succeeded
 10:05   Telegram: "reddit_crawl $NVDA done -- 847 posts" --
 10:05   ClawdBot checks in again                         --
 10:05   Task Manager assigns next task                   ...
```

## Appendix C: Example Retry with Prompt Rewrite

```
 Time    Event                                           Status
 -----   -----                                           ------
 10:00   Task created: reddit_crawl for $NVDA             pending
 10:02   ClawdBot executes... hits rate limit at 342/800  failed
 10:02   Telegram: "reddit_crawl $NVDA failed -- 429 rate limit"

 10:02   Task Manager analyzes failure:
         "Error was 429 Too Many Requests. The original prompt did not
          include rate limiting instructions. I will rewrite the prompt
          to add delays between requests and smaller batch sizes."

 10:02   Task Manager rewrites prompt and retries          pending (retry 1)
         New prompt includes: "Add 2s delay between requests.
          Batch size 50. On 429, wait 60s and retry."

 10:05   ClawdBot #2 checks in, gets the retried task     assigned
 10:08   ClawdBot completes successfully (847 posts)       succeeded
 10:08   Telegram: "reddit_crawl $NVDA succeeded on retry -- 847 posts"
```
