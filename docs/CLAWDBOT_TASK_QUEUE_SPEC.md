# ClawdBot Task Queue System: Product & Technical Spec

> **Version**: 1.0
> **Date**: February 2026
> **Status**: Draft for Review

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [Problem Statement](#2-problem-statement)
3. [Product Vision](#3-product-vision)
4. [Phase 1: Task Queue & ClawdBot Workers](#4-phase-1-task-queue--clawdbot-workers)
5. [Phase 2: Agent Orchestration & Metrics](#5-phase-2-agent-orchestration--metrics)
6. [Technical Architecture](#6-technical-architecture)
7. [Database Schema](#7-database-schema)
8. [API Specification](#8-api-specification)
9. [ClawdBot Worker Design](#9-clawdbot-worker-design)
10. [Communication Layer](#10-communication-layer)
11. [Security & Auth](#11-security--auth)
12. [Implementation Roadmap](#12-implementation-roadmap)
13. [Open Questions & Decisions](#13-open-questions--decisions)

---

## 1. Executive Summary

### What is the ClawdBot Task Queue?

A distributed task execution system where human-added tasks (primarily data crawling and AI analysis) are placed into a queue, picked up by **ClawdBot workers** (machines running on personal Macs, friend machines, or cloud instances), executed, and reported back. Humans can monitor progress and intervene via a web dashboard and messaging channels (Telegram/Slack).

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

---

## 3. Product Vision

```
Phase 1: "Humans add tasks, ClawdBots execute them"
Phase 2: "An AI agent adds tasks, ClawdBots execute them, metrics track everything"
```

### Key Personas

| Persona | Description |
|---------|-------------|
| **Human Operator** | You and your team. Adds tasks, monitors progress, gives instructions |
| **ClawdBot Worker** | A machine (Mac, cloud VM, etc.) that polls the queue and executes tasks |
| **AI Agent** (Phase 2) | An autonomous agent that decides what tasks to create based on data gaps |

---

## 4. Phase 1: Task Queue & ClawdBot Workers

### 4.1 Core Capabilities

#### Task Management
- **Add tasks** — Humans create tasks via the admin dashboard or API
- **Task types** — Crawl (Reddit, X/Twitter, news sites, SEC, etc.), AI Analysis (send data to ChatGPT/Claude for summary), Internal API Update (post processed data to InvestorCenter API)
- **Task metadata** — Ticker symbol(s), data source, date range, priority, parameters
- **Task status lifecycle** — `pending` → `claimed` → `running` → `succeeded` / `failed` / `cancelled`
- **Retry policy** — Configurable retries with backoff per task type
- **Priority queue** — Tasks have priority levels (critical, high, normal, low)

#### ClawdBot Workers
- **Polling-based** — Workers poll the API for available tasks (simpler than push, works behind NATs/firewalls)
- **Heartbeat** — Workers send periodic heartbeats; stale claims are auto-released
- **Capability tags** — Each worker declares what task types it can handle (e.g., "reddit_crawl", "ai_analysis")
- **Concurrency** — Each worker can run N tasks concurrently (configurable)
- **Heterogeneous** — Workers can be your MacBook, a friend's machine, a cloud VM, a K8s pod

#### Human Oversight
- **Admin dashboard** — View all tasks, filter by status/type/ticker, retry failed tasks
- **Real-time updates** — Task status changes reflected immediately
- **Manual intervention** — Cancel running tasks, reprioritize, add notes
- **Notifications** — Telegram/Slack messages for completions, failures, and stalls

### 4.2 Task Types (Phase 1)

| Task Type | Description | Input | Output |
|-----------|-------------|-------|--------|
| `reddit_crawl` | Pull posts from Reddit subreddits for a ticker | `{ticker, subreddits[], date_range, limit}` | Posts stored in `reddit_posts_raw` |
| `twitter_crawl` | Pull posts/mentions from X (Twitter) | `{ticker, query, date_range, limit}` | Posts stored in `social_posts` |
| `news_crawl` | Crawl news articles for a ticker | `{ticker, sources[], date_range}` | Articles stored in `news_articles` |
| `sec_filing_crawl` | Pull specific SEC filings | `{ticker, filing_types[], date_range}` | Filings stored in `sec_filings` |
| `ai_sentiment` | Run AI sentiment analysis on collected posts | `{ticker, source, post_ids[]}` | Sentiment scores in `reddit_sentiment` / `social_sentiment` |
| `ai_summary` | Generate AI analysis summary for a ticker | `{ticker, data_sources[]}` | Summary stored in new `ai_analyses` table |
| `api_update` | Post processed data to InvestorCenter internal API | `{endpoint, method, payload}` | HTTP response logged |

### 4.3 User Flows

#### Adding a Task (Human)
```
1. Human opens Admin → Task Queue page
2. Selects task type (e.g., "Reddit Crawl")
3. Fills in parameters (ticker: NVDA, subreddits: [wallstreetbets, stocks], last 30 days)
4. Sets priority (normal)
5. Submits → task goes to "pending" state
6. (Optional) Adds batch of tickers at once
```

#### Worker Executing a Task (ClawdBot)
```
1. ClawdBot polls GET /api/v1/tasks/claim?capabilities=reddit_crawl,ai_sentiment
2. Server returns highest-priority pending task matching capabilities
3. Task status → "claimed", assigned to this worker
4. ClawdBot starts executing, status → "running"
5. ClawdBot sends heartbeats every 30s
6. On completion: POST /api/v1/tasks/{id}/complete with result summary
7. Task status → "succeeded" or "failed" (with error details)
8. Notification sent to Telegram: "✅ Reddit crawl for $NVDA completed — 847 posts collected"
```

#### Monitoring (Human)
```
1. Human opens Admin → Task Queue dashboard
2. Sees: 12 pending, 3 running, 45 succeeded, 2 failed
3. Clicks on failed task → sees error: "Rate limited by Reddit API"
4. Clicks "Retry" → task goes back to pending with retry_count + 1
5. Gets Telegram notification when retried task succeeds
```

---

## 5. Phase 2: Agent Orchestration & Metrics

### 5.1 AI Task Agent

An autonomous agent that manages the task queue intelligently:

- **Data gap detection** — Scans the database for tickers with stale or missing data (e.g., "AAPL hasn't had Reddit data pulled in 7 days")
- **Priority scoring** — Prioritizes based on: watchlist popularity, trending tickers, earnings upcoming, data staleness
- **Auto-scheduling** — Creates tasks automatically ("Pull Reddit data for top 50 watched tickers daily")
- **Adaptive scheduling** — Increases crawl frequency for trending/volatile tickers
- **Budget awareness** — Respects rate limits and API costs (e.g., don't send 10k posts to ChatGPT in an hour)
- **Self-healing** — Detects and retries systematic failures (e.g., if Reddit API is down, backs off and retries later)

#### Agent Decision Loop
```
Every N minutes:
  1. Query: Which tickers have the most watchers / highest IC score changes?
  2. Query: Which tickers have stale data (>X hours since last crawl)?
  3. Query: Are any tickers trending on social media that we're not tracking?
  4. Score each ticker's "data freshness need"
  5. Create tasks for top-priority tickers (respecting rate limits)
  6. Review failed tasks — retry with different strategy or flag for human
  7. Send daily digest to Telegram: "Today: 200 tasks created, 195 succeeded, 5 failed"
```

### 5.2 Metrics & Monitoring System

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
| **Agent decisions** | What the AI agent decided and why (audit log) |

#### Ticker Coverage Matrix
```
           | Reddit | Twitter | News | SEC | Sentiment | IC Score |
  AAPL     |   ✅   |   ✅    |  ✅  |  ✅ |    ✅     |    ✅    |  Last: 2h ago
  NVDA     |   ✅   |   ⚠️    |  ✅  |  ✅ |    ✅     |    ✅    |  Last: 6h ago
  TSLA     |   ✅   |   ❌    |  ✅  |  ✅ |    ⚠️     |    ✅    |  Last: 1d ago
  GME      |   ❌   |   ❌    |  ❌  |  ✅ |    ❌     |    ❌    |  Never
```

---

## 6. Technical Architecture

### 6.1 Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────┐
│                        HUMAN LAYER                                   │
│                                                                      │
│   ┌──────────────┐    ┌──────────────┐    ┌──────────────────────┐  │
│   │ Admin Web UI │    │   Telegram   │    │  Slack (optional)    │  │
│   │ (Next.js)    │    │   Bot        │    │  Bot                 │  │
│   └──────┬───────┘    └──────┬───────┘    └──────────┬───────────┘  │
│          │                   │                       │               │
└──────────┼───────────────────┼───────────────────────┼───────────────┘
           │                   │                       │
           ▼                   ▼                       ▼
┌─────────────────────────────────────────────────────────────────────┐
│                      API LAYER (Go Backend)                          │
│                                                                      │
│   ┌──────────────────────────────────────────────────────────────┐  │
│   │  Task Queue API                                               │  │
│   │  POST /tasks          — Create task                           │  │
│   │  GET  /tasks/claim    — Worker claims next task               │  │
│   │  POST /tasks/:id/heartbeat  — Worker heartbeat                │  │
│   │  POST /tasks/:id/complete   — Worker reports completion       │  │
│   │  GET  /tasks          — List/filter tasks                     │  │
│   │  POST /tasks/:id/cancel     — Cancel a task                   │  │
│   │  POST /tasks/:id/retry      — Retry a failed task             │  │
│   └──────────────────────────────────────────────────────────────┘  │
│                                                                      │
│   ┌──────────────────────────────────────────────────────────────┐  │
│   │  Worker API                                                   │  │
│   │  POST /workers/register     — Register a ClawdBot             │  │
│   │  POST /workers/:id/heartbeat — Worker liveness                │  │
│   │  GET  /workers              — List active workers             │  │
│   └──────────────────────────────────────────────────────────────┘  │
│                                                                      │
│   ┌──────────────────────────────────────────────────────────────┐  │
│   │  Metrics API (Phase 2)                                        │  │
│   │  GET /metrics/tasks         — Task metrics                    │  │
│   │  GET /metrics/coverage      — Ticker coverage matrix          │  │
│   │  GET /metrics/workers       — Worker health metrics           │  │
│   └──────────────────────────────────────────────────────────────┘  │
│                                                                      │
└────────────────────────────┬────────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────────┐
│                      DATA LAYER                                      │
│                                                                      │
│   ┌──────────────┐    ┌──────────────┐                              │
│   │ PostgreSQL   │    │ Redis        │                              │
│   │ (tasks,      │    │ (claim locks,│                              │
│   │  results,    │    │  rate limits, │                              │
│   │  metrics)    │    │  heartbeats) │                              │
│   └──────────────┘    └──────────────┘                              │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
           ▲                                       ▲
           │                                       │
┌──────────┼───────────────────────────────────────┼──────────────────┐
│          │          WORKER LAYER                  │                  │
│          │                                       │                  │
│   ┌──────┴──────┐  ┌─────────────┐  ┌───────────┴──┐              │
│   │ ClawdBot #1 │  │ ClawdBot #2 │  │ ClawdBot #3  │   ...        │
│   │ (Your Mac)  │  │ (Friend's)  │  │ (Cloud VM)   │              │
│   │             │  │             │  │              │              │
│   │ Capabilities│  │ Capabilities│  │ Capabilities │              │
│   │ - reddit    │  │ - twitter   │  │ - ai_analysis│              │
│   │ - news      │  │ - reddit    │  │ - reddit     │              │
│   │ - ai_anal.  │  │ - news      │  │ - sec_filing │              │
│   └─────────────┘  └─────────────┘  └──────────────┘              │
│          │                │                │                        │
│          ▼                ▼                ▼                        │
│   ┌─────────────────────────────────────────────────────────────┐  │
│   │ External Data Sources                                        │  │
│   │ Reddit API, X/Twitter API, SEC EDGAR, News sites,           │  │
│   │ ChatGPT/Claude API, InvestorCenter Internal API              │  │
│   └─────────────────────────────────────────────────────────────┘  │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

### 6.2 Why This Architecture?

**PostgreSQL as the queue** (not RabbitMQ/Redis Streams/SQS):
- We already have PostgreSQL, no new infrastructure
- Task volume is low-to-moderate (hundreds/day, not millions)
- Need durable, queryable task history for the dashboard and metrics
- `SELECT ... FOR UPDATE SKIP LOCKED` gives us reliable, concurrent task claiming
- If we outgrow this, migrating to SQS/Redis Streams later is straightforward

**Polling-based workers** (not WebSocket/push):
- Workers can be behind NATs, firewalls, home routers — no inbound connections needed
- Simpler to implement and debug
- Heartbeat + claim expiry handles worker crashes gracefully
- 5-10 second polling interval is fine for our task volume

**Redis for coordination** (not just PostgreSQL):
- Claim locks to prevent race conditions during concurrent claims
- Rate limit counters per data source (Reddit API: 60 req/min, etc.)
- Worker heartbeat tracking with TTL-based expiry
- Fast but not critical — system works without Redis (falls back to DB locks)

### 6.3 Integration with Existing System

The task queue complements, not replaces, the existing CronJob pipelines:

| Concern | Current (CronJobs) | Task Queue |
|---------|-------------------|------------|
| Scheduled data pulls | ✅ Keep as-is | Not needed |
| Ad-hoc crawl requests | ❌ Not possible | ✅ Primary use case |
| Distributed execution | ❌ K8s only | ✅ Any machine |
| Task-level tracking | ❌ Job-level only | ✅ Per-task status |
| Human intervention | ❌ None | ✅ Dashboard + messaging |

Over time, some CronJobs could be migrated to agent-scheduled tasks (Phase 2), but the CronJobs remain as a reliable baseline.

---

## 7. Database Schema

### 7.1 Core Tables

```sql
-- The central task table
CREATE TABLE tasks (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Task definition
    task_type       VARCHAR(50) NOT NULL,        -- 'reddit_crawl', 'ai_sentiment', etc.
    priority        SMALLINT NOT NULL DEFAULT 50, -- 0 (highest) to 100 (lowest)
    params          JSONB NOT NULL,               -- Type-specific parameters

    -- Lifecycle
    status          VARCHAR(20) NOT NULL DEFAULT 'pending',
                    -- pending, claimed, running, succeeded, failed, cancelled
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    claimed_at      TIMESTAMPTZ,
    started_at      TIMESTAMPTZ,
    completed_at    TIMESTAMPTZ,

    -- Worker assignment
    worker_id       UUID REFERENCES workers(id),
    last_heartbeat  TIMESTAMPTZ,

    -- Results
    result          JSONB,                        -- Success: summary data. Failure: error details
    error_message   TEXT,

    -- Retry
    retry_count     SMALLINT NOT NULL DEFAULT 0,
    max_retries     SMALLINT NOT NULL DEFAULT 3,

    -- Metadata
    created_by      VARCHAR(100) NOT NULL,        -- 'human:username' or 'agent:scheduler'
    tags            TEXT[],                        -- ['NVDA', 'reddit', 'urgent']
    notes           TEXT,                          -- Human-added notes
    parent_task_id  UUID REFERENCES tasks(id),    -- For sub-task chains

    -- Indexes
    CONSTRAINT valid_status CHECK (status IN
        ('pending', 'claimed', 'running', 'succeeded', 'failed', 'cancelled'))
);

CREATE INDEX idx_tasks_claimable ON tasks (priority, created_at)
    WHERE status = 'pending';
CREATE INDEX idx_tasks_status ON tasks (status);
CREATE INDEX idx_tasks_worker ON tasks (worker_id) WHERE status IN ('claimed', 'running');
CREATE INDEX idx_tasks_tags ON tasks USING GIN (tags);
CREATE INDEX idx_tasks_created_at ON tasks (created_at);
CREATE INDEX idx_tasks_type_status ON tasks (task_type, status);

-- Worker registration
CREATE TABLE workers (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            VARCHAR(100) NOT NULL,        -- 'macbook-pro-john', 'cloud-vm-1'
    capabilities    TEXT[] NOT NULL,               -- ['reddit_crawl', 'ai_sentiment']
    max_concurrency SMALLINT NOT NULL DEFAULT 2,

    -- Status
    status          VARCHAR(20) NOT NULL DEFAULT 'online',
                    -- online, idle, busy, offline
    last_heartbeat  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    current_tasks   SMALLINT NOT NULL DEFAULT 0,

    -- Auth
    api_key_hash    VARCHAR(256) NOT NULL,         -- Hashed API key for auth

    -- Metadata
    registered_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    metadata        JSONB,                         -- OS, IP, version, etc.

    CONSTRAINT valid_worker_status CHECK (status IN ('online', 'idle', 'busy', 'offline'))
);

-- Task execution log (append-only audit trail)
CREATE TABLE task_events (
    id              BIGSERIAL PRIMARY KEY,
    task_id         UUID NOT NULL REFERENCES tasks(id),
    event_type      VARCHAR(30) NOT NULL,          -- 'created', 'claimed', 'started',
                                                   -- 'heartbeat', 'progress', 'succeeded',
                                                   -- 'failed', 'retried', 'cancelled'
    worker_id       UUID REFERENCES workers(id),
    details         JSONB,                         -- Event-specific data
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_task_events_task ON task_events (task_id, created_at);

-- Notification log
CREATE TABLE task_notifications (
    id              BIGSERIAL PRIMARY KEY,
    task_id         UUID REFERENCES tasks(id),
    channel         VARCHAR(20) NOT NULL,          -- 'telegram', 'slack', 'in_app'
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
    decision_type   VARCHAR(50) NOT NULL,          -- 'create_task', 'retry', 'escalate'
    reasoning       TEXT NOT NULL,                 -- Why the agent made this decision
    tickers         TEXT[],
    tasks_created   UUID[],                        -- References to created tasks
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Ticker data freshness tracking
CREATE TABLE ticker_data_freshness (
    ticker          VARCHAR(10) NOT NULL,
    data_source     VARCHAR(50) NOT NULL,          -- 'reddit', 'twitter', 'news', 'sec'
    last_success_at TIMESTAMPTZ,
    last_attempt_at TIMESTAMPTZ,
    last_error      TEXT,
    record_count    INTEGER DEFAULT 0,             -- Total records collected
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

### 8.1 Task Queue API

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
    "created_at": "2026-02-08T10:00:00Z"
}
```

#### Create Batch Tasks
```
POST /api/v1/tasks/batch
Authorization: Bearer <admin_jwt>

{
    "tasks": [
        { "task_type": "reddit_crawl", "params": {"ticker": "NVDA", ...} },
        { "task_type": "reddit_crawl", "params": {"ticker": "AAPL", ...} },
        { "task_type": "reddit_crawl", "params": {"ticker": "TSLA", ...} }
    ]
}

Response 201:
{
    "created": 3,
    "task_ids": ["a1b2...", "c3d4...", "e5f6..."]
}
```

#### Claim Task (Worker)
```
POST /api/v1/tasks/claim
Authorization: X-Worker-Key <api_key>

{
    "worker_id": "w1-macbook-john",
    "capabilities": ["reddit_crawl", "ai_sentiment"]
}

Response 200:
{
    "id": "a1b2c3d4-...",
    "task_type": "reddit_crawl",
    "params": { ... },
    "claimed_at": "2026-02-08T10:05:00Z"
}

Response 204: (no tasks available)
```

#### Heartbeat (Worker)
```
POST /api/v1/tasks/{id}/heartbeat
Authorization: X-Worker-Key <api_key>

{
    "worker_id": "w1-macbook-john",
    "progress": {
        "posts_collected": 342,
        "estimated_total": 800,
        "percent": 42
    }
}

Response 200: { "continue": true }
Response 410: { "continue": false, "reason": "task_cancelled" }
```

#### Complete Task (Worker)
```
POST /api/v1/tasks/{id}/complete
Authorization: X-Worker-Key <api_key>

{
    "worker_id": "w1-macbook-john",
    "status": "succeeded",
    "result": {
        "posts_collected": 847,
        "posts_with_sentiment": 812,
        "date_range_actual": {"from": "2026-01-02", "to": "2026-02-07"},
        "duration_ms": 45200
    }
}

Response 200: { "acknowledged": true }
```

#### List Tasks
```
GET /api/v1/tasks?status=failed&task_type=reddit_crawl&ticker=NVDA&page=1&limit=20
Authorization: Bearer <admin_jwt>

Response 200:
{
    "tasks": [...],
    "total": 42,
    "page": 1,
    "pages": 3
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
    "retry_count": 2
}
```

#### Cancel Task
```
POST /api/v1/tasks/{id}/cancel
Authorization: Bearer <admin_jwt>

Response 200: { "cancelled": true }
```

### 8.2 Worker API

#### Register Worker
```
POST /api/v1/workers/register
Authorization: Bearer <admin_jwt>

{
    "name": "macbook-pro-john",
    "capabilities": ["reddit_crawl", "twitter_crawl", "ai_sentiment"],
    "max_concurrency": 3,
    "metadata": {
        "os": "macOS 15.3",
        "python": "3.11.5",
        "version": "0.1.0"
    }
}

Response 201:
{
    "id": "w-abc123...",
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
            "current_tasks": 2,
            "last_heartbeat": "2026-02-08T10:04:55Z"
        }
    ]
}
```

---

## 9. ClawdBot Worker Design

### 9.1 Worker Architecture

The ClawdBot worker is a **Python CLI application** that runs on any machine.

```
clawdbot/
├── __init__.py
├── cli.py                  # CLI entry point (click or argparse)
├── config.py               # Configuration (API URL, worker key, capabilities)
├── worker.py               # Main worker loop (poll → execute → report)
├── heartbeat.py            # Background heartbeat thread
├── handlers/               # One handler per task type
│   ├── __init__.py
│   ├── base.py             # Base handler interface
│   ├── reddit_crawl.py     # Reddit crawling logic
│   ├── twitter_crawl.py    # Twitter/X crawling logic
│   ├── news_crawl.py       # News site crawling
│   ├── sec_filing.py       # SEC EDGAR crawling
│   ├── ai_sentiment.py     # AI sentiment via ChatGPT/Claude
│   ├── ai_summary.py       # AI ticker summary
│   └── api_update.py       # POST to InvestorCenter API
├── notifier.py             # Send messages to Telegram/Slack
└── utils/
    ├── http.py             # HTTP client with retry
    ├── rate_limiter.py     # Per-source rate limiting
    └── logger.py           # Structured logging
```

### 9.2 Worker Lifecycle

```python
# Pseudocode for the main worker loop

async def run_worker(config):
    worker = await register_or_reconnect(config)
    heartbeat_task = start_heartbeat(worker)

    while True:
        if worker.current_tasks < worker.max_concurrency:
            task = await claim_task(worker)
            if task:
                asyncio.create_task(execute_task(worker, task))
            else:
                await asyncio.sleep(config.poll_interval)  # 5-10 seconds
        else:
            await asyncio.sleep(1)  # Wait for a slot to open

async def execute_task(worker, task):
    handler = get_handler(task.task_type)
    try:
        await send_heartbeat(task, {"status": "starting"})
        result = await handler.execute(task.params, heartbeat_callback)
        await complete_task(task.id, "succeeded", result)
        await notify(f"✅ {task.task_type} for {task.params.get('ticker', '?')} completed")
    except Exception as e:
        await complete_task(task.id, "failed", error=str(e))
        await notify(f"❌ {task.task_type} for {task.params.get('ticker', '?')} failed: {e}")
```

### 9.3 Running a ClawdBot

```bash
# Install
pip install clawdbot  # or: git clone && pip install -e .

# Configure
export CLAWDBOT_API_URL="https://api.investorcenter.ai"
export CLAWDBOT_API_KEY="cwb_live_xxxxxxxx"
export TELEGRAM_BOT_TOKEN="123456:ABC..."  # optional
export TELEGRAM_CHAT_ID="-100123456789"    # optional

# Run
clawdbot start --capabilities reddit_crawl,ai_sentiment --concurrency 2

# Or with a config file
clawdbot start --config ~/.clawdbot.yaml
```

### 9.4 Handler Interface

```python
class TaskHandler(ABC):
    """Base interface for all task handlers."""

    @abstractmethod
    async def execute(self, params: dict, heartbeat: Callable) -> dict:
        """
        Execute the task.

        Args:
            params: Task-specific parameters from the task definition
            heartbeat: Callback to send progress updates

        Returns:
            Result dict to be stored as task.result

        Raises:
            Exception: On failure (will be caught and reported)
        """
        pass

    @property
    @abstractmethod
    def task_type(self) -> str:
        """The task type this handler processes."""
        pass


class RedditCrawlHandler(TaskHandler):
    task_type = "reddit_crawl"

    async def execute(self, params, heartbeat):
        ticker = params["ticker"]
        subreddits = params["subreddits"]
        date_range = params["date_range"]

        posts_collected = 0
        for subreddit in subreddits:
            async for batch in self.fetch_posts(subreddit, ticker, date_range):
                posts_collected += len(batch)
                await self.store_posts(batch)
                await heartbeat({"posts_collected": posts_collected})

        # Post to InvestorCenter API
        await self.update_investorcenter(ticker, posts_collected)

        return {"posts_collected": posts_collected, "subreddits": subreddits}
```

---

## 10. Communication Layer

### 10.1 Telegram Bot (Recommended for Phase 1)

**Why Telegram over Slack?**
- Free, no workspace needed, works on mobile
- Simple Bot API — 5 minutes to set up
- Supports group chats (team notifications) and DMs (personal alerts)
- Rich formatting (markdown, inline buttons for quick actions)

#### Telegram Notification Types

| Event | Message |
|-------|---------|
| Task completed | `✅ reddit_crawl for $NVDA completed — 847 posts (45s)` |
| Task failed | `❌ twitter_crawl for $AAPL failed — Rate limit exceeded (retry 2/3)` |
| Worker online | `🤖 ClawdBot "macbook-john" connected (reddit, ai_sentiment)` |
| Worker offline | `⚠️ ClawdBot "macbook-john" went offline (last seen 5m ago)` |
| Queue stalled | `🔴 12 tasks pending, 0 workers online` |
| Daily digest (P2) | `📊 Daily: 200 tasks, 195✅ 5❌ — Top: NVDA(45), AAPL(38), TSLA(32)` |

#### Interactive Commands (via Telegram)

```
/status           — Current queue status (pending/running/done/failed)
/workers          — List online workers
/queue            — Show pending tasks
/retry <task_id>  — Retry a failed task
/add reddit NVDA  — Quick-add a Reddit crawl task for NVDA
/pause            — Pause all workers
/resume           — Resume all workers
```

### 10.2 Implementation

```python
# Lightweight Telegram integration
import httpx

class TelegramNotifier:
    def __init__(self, bot_token: str, chat_id: str):
        self.bot_token = bot_token
        self.chat_id = chat_id
        self.base_url = f"https://api.telegram.org/bot{bot_token}"

    async def send(self, message: str):
        async with httpx.AsyncClient() as client:
            await client.post(f"{self.base_url}/sendMessage", json={
                "chat_id": self.chat_id,
                "text": message,
                "parse_mode": "Markdown"
            })
```

---

## 11. Security & Auth

### 11.1 Authentication Model

| Actor | Auth Method |
|-------|-------------|
| Admin (web UI) | JWT (existing auth system) |
| ClawdBot workers | API key (per-worker, hashed in DB) |
| Telegram bot | Bot token + webhook secret |
| AI Agent (P2) | Internal service key |

### 11.2 Worker Security

- Each worker gets a unique API key on registration (shown once, stored hashed)
- Workers can ONLY: claim tasks, send heartbeats, report completion
- Workers CANNOT: create tasks, cancel tasks, access other workers' data
- API keys can be revoked from admin dashboard
- Rate limiting on claim endpoint to prevent abuse

### 11.3 Data Source Credentials

- API keys for Reddit, Twitter, SEC, ChatGPT etc. are stored as environment variables on the worker machine
- Workers never send these credentials to the server
- The task `params` specify *what* to crawl, the worker's local env has *how* to authenticate

---

## 12. Implementation Roadmap

### Phase 1a: Core Queue (Week 1-2)

| Task | Effort |
|------|--------|
| Database migration (tasks, workers, task_events tables) | S |
| Go API: CRUD tasks, claim, heartbeat, complete | M |
| Worker registration and API key auth | S |
| Stale claim reaper (background goroutine) | S |
| Basic admin UI: list tasks, create task form, retry button | M |

### Phase 1b: ClawdBot Worker (Week 2-3)

| Task | Effort |
|------|--------|
| Python worker scaffold (CLI, config, main loop) | M |
| `reddit_crawl` handler (adapt existing `fetcher.py`) | M |
| `ai_sentiment` handler (adapt existing `ai_processor.py`) | S |
| `api_update` handler | S |
| Heartbeat + progress reporting | S |
| Telegram notifier | S |

### Phase 1c: Polish & Deploy (Week 3-4)

| Task | Effort |
|------|--------|
| `twitter_crawl` handler | M |
| `news_crawl` handler | M |
| Batch task creation UI | S |
| Telegram interactive commands (/status, /retry) | M |
| Error handling, retry logic, rate limiting | M |
| Documentation + worker setup guide | S |

### Phase 2a: Metrics (Week 5-6)

| Task | Effort |
|------|--------|
| `ticker_data_freshness` table + tracking | S |
| `task_metrics_daily` aggregation job | S |
| Metrics API endpoints | M |
| Admin dashboard: metrics page, coverage matrix | M |

### Phase 2b: AI Agent (Week 6-8)

| Task | Effort |
|------|--------|
| Agent decision loop (Python service or CronJob) | L |
| Data gap detection queries | M |
| Priority scoring algorithm | M |
| `agent_decisions` audit log | S |
| Agent-created tasks feed into existing queue | S |
| Daily digest via Telegram | S |

**S** = Small (< 1 day), **M** = Medium (1-3 days), **L** = Large (3-5 days)

---

## 13. Open Questions & Decisions

### Needs Decision

| # | Question | Options | Recommendation |
|---|----------|---------|----------------|
| 1 | **Queue backend** | PostgreSQL `SKIP LOCKED` vs Redis Streams vs SQS | **PostgreSQL** — already have it, task volume is low, queryable history is valuable |
| 2 | **Primary notification channel** | Telegram vs Slack vs Discord | **Telegram** — free, simple API, mobile-friendly, no workspace needed |
| 3 | **Worker language** | Python vs Go vs Node | **Python** — most crawling libs, existing pipeline code is Python, team familiarity |
| 4 | **Worker distribution** | pip package vs Docker image vs binary | **pip install** for dev machines, **Docker image** for cloud — support both |
| 5 | **Twitter/X data source** | Official API ($100/mo basic) vs scraping vs third-party | Needs research — X API pricing may be prohibitive; alternatives exist |
| 6 | **AI provider for analysis** | ChatGPT (OpenAI) vs Claude vs Google Gemini | Could support multiple; start with whatever has existing API keys |
| 7 | **Agent autonomy level (P2)** | Fully autonomous vs human-approval-required | Start with **human-approval** for high-cost tasks (AI analysis), auto for crawls |

### Future Considerations

- **Task dependencies** — "Run sentiment analysis after crawl completes" (DAG execution)
- **Cost tracking** — Track API costs per task (OpenAI tokens, Reddit API calls)
- **Multi-tenant** — If other teams want to use the queue
- **Webhook triggers** — External events trigger tasks (e.g., SEC filing detected → auto-crawl)
- **Worker auto-scaling** — Spin up cloud workers when queue depth is high

---

## Appendix A: Comparison to Existing CronJobs

After the task queue is built, here's how the two systems coexist:

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
 Time    Event                                   Status
 ─────   ─────                                   ──────
 10:00   Human creates task via dashboard         pending
 10:02   ClawdBot "macbook-john" claims task      claimed
 10:02   Worker starts execution                  running
 10:03   Heartbeat: 200/800 posts collected       running (25%)
 10:04   Heartbeat: 500/800 posts collected       running (62%)
 10:05   Heartbeat: 800/800 posts collected       running (100%)
 10:05   Worker reports success                   succeeded
 10:05   Telegram: "✅ reddit_crawl NVDA done"     —
 10:05   Data posted to InvestorCenter API         —
```
