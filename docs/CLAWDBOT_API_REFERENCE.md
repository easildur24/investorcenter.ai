# ClawdBot Worker API Reference

> **Base URL**: `https://api.investorcenter.ai/api/v1`
> **Auth**: All endpoints require `Authorization: Bearer <jwt>` header
> **Worker requirement**: All `/worker/*` endpoints require the user to have `is_worker = true`

---

## Getting Started

This section walks you through setting up a new ClawdBot from scratch.

### Step 1: Get Your Worker Account

An admin creates your worker account from the [Admin Workers page](https://investorcenter.ai/admin/workers). They'll provide you with:

- **Email** — e.g. `genesis@investorcenter.ai`
- **Password** — set during registration

If you're the admin, go to **Admin > Workers > Register Worker** and fill in the email, password, and name for your new ClawdBot.

### Step 2: Set Up Your Project Directory

Create a folder anywhere on the machine that will run the ClawdBot:

```bash
mkdir ~/clawdbot && cd ~/clawdbot
```

Create a `.env` file with your credentials:

```bash
CLAWDBOT_EMAIL=genesis@investorcenter.ai
CLAWDBOT_PASSWORD=your-password
API_BASE=https://api.investorcenter.ai/api/v1
```

### Step 3: Add a CLAUDE.md

Create a `CLAUDE.md` in your project directory. This is the system prompt that tells Claude Code how to behave as a worker:

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
- Refresh the token before it expires (1 hour)

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

### Step 4: Verify Your Setup

Test that your credentials work by logging in:

```bash
curl -s -X POST https://api.investorcenter.ai/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email": "genesis@investorcenter.ai", "password": "your-password"}' \
  | jq .access_token
```

You should get back a JWT token. Then verify you can access the worker API:

```bash
TOKEN="<paste your token>"

curl -s https://api.investorcenter.ai/api/v1/worker/tasks \
  -H "Authorization: Bearer $TOKEN" \
  | jq .
```

### Step 5: Run Your ClawdBot

Start Claude Code in your project directory:

```bash
cd ~/clawdbot
claude
```

Tell it to start working:

```
Check for pending tasks and execute them.
```

Claude Code reads the `CLAUDE.md`, authenticates with the API, picks up any assigned tasks, reads the SOP, and starts executing.

### Step 6: Assign a Task

From the [Admin Workers page](https://investorcenter.ai/admin/workers), switch to the **Tasks** tab and click **New Task**:

1. Pick a **Task Type** (e.g. Reddit Crawl)
2. Fill in the **Parameters** that appear (e.g. ticker: NVDA, subreddits: ["wallstreetbets"], days: 30)
3. Give it a **Title** (e.g. "Reddit Crawl for NVDA")
4. **Assign** it to your ClawdBot worker
5. Set **Priority** and click **Create**

Your ClawdBot will pick it up on its next poll.

---

## Authentication

### Login

```
POST /auth/login
```

Authenticate with email/password to get a JWT token.

**Request body:**

```json
{
  "email": "genesis@investorcenter.ai",
  "password": "your-password"
}
```

**Response `200`:**

```json
{
  "access_token": "eyJhbG...",
  "refresh_token": "eyJhbG...",
  "expires_in": 3600,
  "user": {
    "id": "uuid",
    "email": "genesis@investorcenter.ai",
    "full_name": "Genesis"
  }
}
```

Use `access_token` as `Bearer <token>` on all subsequent requests. Tokens expire after `expires_in` seconds (1 hour). Use the refresh endpoint to get a new access token.

### Refresh Token

```
POST /auth/refresh
```

**Request body:**

```json
{
  "refresh_token": "eyJhbG..."
}
```

**Response `200`:** Same shape as login response with new tokens.

---

## Task Types

### List Task Types

```
GET /worker/task-types
```

Returns all active task types with their SOPs and parameter schemas.

**Response `200`:**

```json
{
  "success": true,
  "data": [
    {
      "id": 1,
      "name": "reddit_crawl",
      "label": "Reddit Crawl",
      "sop": "## Reddit Crawl SOP\n\n### Data Source\nUse the Arctic Shift API...",
      "param_schema": {
        "ticker": "string",
        "subreddits": "string[]",
        "days": "number"
      },
      "is_active": true,
      "created_at": "2026-02-09T06:00:00Z",
      "updated_at": "2026-02-09T06:00:00Z"
    }
  ]
}
```

### Get Task Type

```
GET /worker/task-types/:id
```

Get a single task type by ID, including its full SOP.

**Path params:**

| Param | Type | Description |
|-------|------|-------------|
| `id` | int | Task type ID |

**Response `200`:**

```json
{
  "success": true,
  "data": {
    "id": 1,
    "name": "reddit_crawl",
    "label": "Reddit Crawl",
    "sop": "## Reddit Crawl SOP\n\n...",
    "param_schema": { "ticker": "string", "subreddits": "string[]", "days": "number" },
    "is_active": true,
    "created_at": "2026-02-09T06:00:00Z",
    "updated_at": "2026-02-09T06:00:00Z"
  }
}
```

**Errors:**

| Status | Error | When |
|--------|-------|------|
| `404` | `"Task type not found"` | ID doesn't exist or is inactive |

---

## Tasks

### Get My Tasks

```
GET /worker/tasks
```

Get all tasks assigned to the authenticated worker. Results are ordered by priority (urgent first), then by creation date (newest first).

**Query params:**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `status` | string | No | Filter by status: `pending`, `in_progress`, `completed`, `failed` |

**Response `200`:**

```json
{
  "success": true,
  "data": [
    {
      "id": "a1b2c3d4-...",
      "title": "Reddit Crawl for NVDA",
      "description": "Crawl last 30 days of NVDA mentions",
      "assigned_to": "worker-user-uuid",
      "status": "pending",
      "priority": "high",
      "task_type_id": 1,
      "task_type": {
        "id": 1,
        "name": "reddit_crawl",
        "label": "Reddit Crawl",
        "sop": "## Reddit Crawl SOP\n\nUse the Arctic Shift API..."
      },
      "params": {
        "ticker": "NVDA",
        "subreddits": ["wallstreetbets", "stocks"],
        "days": 30
      },
      "result": null,
      "created_by": "admin-user-uuid",
      "created_at": "2026-02-09T10:00:00Z",
      "updated_at": "2026-02-09T10:00:00Z",
      "started_at": null,
      "completed_at": null
    }
  ]
}
```

The `task_type` object (with SOP) is included inline via a JOIN. The worker gets everything needed to execute the task in a single API call.

### Get Task

```
GET /worker/tasks/:id
```

Get a single task by ID. Must be assigned to the authenticated worker.

**Path params:**

| Param | Type | Description |
|-------|------|-------------|
| `id` | UUID | Task ID |

**Response `200`:** Same shape as a single item in the list above.

**Errors:**

| Status | Error | When |
|--------|-------|------|
| `404` | `"Task not found or not assigned to you"` | Task doesn't exist or is assigned to someone else |

### Update Task Status

```
PUT /worker/tasks/:id/status
```

Update the status of a task. Automatically sets timestamps:
- `in_progress` sets `started_at = NOW()`
- `completed` or `failed` sets `completed_at = NOW()`

**Path params:**

| Param | Type | Description |
|-------|------|-------------|
| `id` | UUID | Task ID |

**Request body:**

```json
{
  "status": "in_progress"
}
```

**Valid statuses:** `in_progress`, `completed`, `failed`

**Response `200`:**

```json
{
  "success": true,
  "data": {
    "id": "a1b2c3d4-...",
    "status": "in_progress",
    "started_at": "2026-02-09T10:02:00Z",
    ...
  }
}
```

**Errors:**

| Status | Error | When |
|--------|-------|------|
| `400` | `"Invalid status..."` | Status not one of the valid values |
| `404` | `"Task not found or not assigned to you"` | Task doesn't exist or wrong worker |

### Post Task Result

```
POST /worker/tasks/:id/result
```

Submit a structured JSON result for a task. The task must be in `in_progress` status.

**Path params:**

| Param | Type | Description |
|-------|------|-------------|
| `id` | UUID | Task ID |

**Request body:**

```json
{
  "result": {
    "posts_collected": 847,
    "posts_stored": 847,
    "subreddits_crawled": ["wallstreetbets", "stocks"],
    "date_range": "2026-01-10 to 2026-02-09",
    "api_calls_made": 12
  }
}
```

The `result` field accepts any valid JSON object. Structure it meaningfully for the task type.

**Response `200`:**

```json
{
  "success": true,
  "data": {
    "id": "a1b2c3d4-...",
    "result": { "posts_collected": 847, ... },
    ...
  }
}
```

**Errors:**

| Status | Error | When |
|--------|-------|------|
| `400` | `"Can only post results to tasks with status 'in_progress'"` | Task not in progress |
| `404` | `"Task not found or not assigned to you"` | Task doesn't exist or wrong worker |

---

## Task Updates

Progress log / comment thread on a task.

### Get Task Updates

```
GET /worker/tasks/:id/updates
```

Get all updates for a task, ordered oldest first.

**Path params:**

| Param | Type | Description |
|-------|------|-------------|
| `id` | UUID | Task ID |

**Response `200`:**

```json
{
  "success": true,
  "data": [
    {
      "id": "update-uuid",
      "task_id": "a1b2c3d4-...",
      "content": "Starting Arctic Shift API crawl for NVDA in r/wallstreetbets...",
      "created_by": "worker-user-uuid",
      "created_by_name": "Genesis",
      "created_at": "2026-02-09T10:03:00Z"
    },
    {
      "id": "update-uuid-2",
      "task_id": "a1b2c3d4-...",
      "content": "Collected 500 posts so far, continuing...",
      "created_by": "worker-user-uuid",
      "created_by_name": "Genesis",
      "created_at": "2026-02-09T10:05:00Z"
    }
  ]
}
```

**Errors:**

| Status | Error | When |
|--------|-------|------|
| `404` | `"Task not found or not assigned to you"` | Task doesn't exist or wrong worker |

### Post Task Update

```
POST /worker/tasks/:id/updates
```

Post a progress update or log entry to a task's comment thread. Visible to admins in the UI.

**Path params:**

| Param | Type | Description |
|-------|------|-------------|
| `id` | UUID | Task ID |

**Request body:**

```json
{
  "content": "Collected 500 posts from r/wallstreetbets, moving to r/stocks..."
}
```

**Response `201`:**

```json
{
  "success": true,
  "data": {
    "id": "new-update-uuid",
    "task_id": "a1b2c3d4-...",
    "content": "Collected 500 posts from r/wallstreetbets, moving to r/stocks...",
    "created_by": "worker-user-uuid",
    "created_at": "2026-02-09T10:05:00Z"
  }
}
```

**Errors:**

| Status | Error | When |
|--------|-------|------|
| `404` | `"Task not found or not assigned to you"` | Task doesn't exist or wrong worker |

---

## Heartbeat

### Send Heartbeat

```
POST /worker/heartbeat
```

Send a heartbeat to indicate the worker is online. Updates `last_activity_at` on the user record. Workers with activity within the last 5 minutes are shown as "online" in the admin UI.

**Request body:** None (empty body or `{}`)

**Response `200`:**

```json
{
  "success": true,
  "message": "Heartbeat received"
}
```

> **Note:** Every `/worker/*` endpoint also updates `last_activity_at`, so the heartbeat is only needed during idle periods between tasks.

---

## Typical Workflow

```
1. Login
   POST /auth/login  →  get access_token

2. Check for work
   GET /worker/tasks?status=pending  →  pick up assigned tasks
   GET /worker/tasks?status=in_progress  →  resume interrupted tasks

3. For each task:
   a. Read task_type.sop + params from the response
   b. PUT /worker/tasks/:id/status  {"status": "in_progress"}
   c. Execute the work described by SOP + params
   d. POST /worker/tasks/:id/updates  {"content": "progress..."}  (repeat as needed)
   e. POST /worker/tasks/:id/result  {"result": {...}}
   f. PUT /worker/tasks/:id/status  {"status": "completed"}
      (or "failed" if something went wrong)

4. Heartbeat every 2 minutes during idle:
   POST /worker/heartbeat

5. Repeat from step 2
```

---

## Error Responses

All errors follow this shape:

```json
{
  "error": "Human-readable error message"
}
```

| Status | Meaning |
|--------|---------|
| `400` | Bad request — invalid input or state |
| `401` | Not authenticated — missing or invalid JWT |
| `403` | Not authorized — user is not a worker (`is_worker = false`) |
| `404` | Not found — resource doesn't exist or not assigned to you |
| `500` | Server error |
| `503` | Database unavailable |
