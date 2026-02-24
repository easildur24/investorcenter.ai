# Task Runner Skill

**Goal:** Pull next task from the task service, execute the corresponding skill, and report result.

## Prerequisites

1. **API credentials** — email and password for a user account on investorcenter.ai
2. **Repo cloned** — `~/.openclaw/workspace/investorcenter.ai` with latest `skills/` folder
3. **OpenClaw browser profile** available for skills that need browser automation

## Authentication

Get a JWT token:

```bash
curl -s -X POST https://investorcenter.ai/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email": "<EMAIL>", "password": "<PASSWORD>"}'
```

Response:
```json
{"access_token": "eyJ..."}
```

Store the token. It expires after 24 hours — if you get a 401, re-authenticate.

## Execution Flow

### Step 1: Claim Next Task

```bash
curl -s -X POST https://investorcenter.ai/api/v1/tasks/next \
  -H "Authorization: Bearer <TOKEN>" \
  -H "Content-Type: application/json"
```

**If no tasks are pending**, you'll get:
```json
{"error": "No pending tasks available"}
```
→ Exit gracefully. Queue is empty.

**If a task is available**, you'll get:
```json
{
  "success": true,
  "data": {
    "id": "<task-id>",
    "status": "in_progress",
    "task_type_id": 7,
    "task_type": {
      "name": "scrape_ycharts_keystats",
      "skill_path": "scrape-ycharts-keystats"
    },
    "params": {"ticker": "AAPL"},
    "claimed_by": "<your-user-id>"
  }
}
```

The task is now yours — no other bot can claim it.

### Step 2: Load the Skill

Read the skill file at:
```
skills/<skill_path>/SKILL.md
```

For example, if `skill_path` is `scrape-ycharts-keystats`, load:
```
~/.openclaw/workspace/investorcenter.ai/skills/scrape-ycharts-keystats/SKILL.md
```

This file contains the full instructions for executing the task.

### Step 3: Execute the Skill

Follow the instructions in the loaded SKILL.md, using `params` from the task as input.

For example, with `scrape_ycharts_keystats` and `params: {"ticker": "AAPL"}`:
1. Open `https://ycharts.com/companies/AAPL/key_stats` in OpenClaw browser
2. Wait for page load
3. Capture snapshot
4. Extract metrics using parse helpers
5. POST data to ingestion API

### Step 4: Update Task Status

**On success:**
```bash
curl -s -X PUT https://investorcenter.ai/api/v1/tasks/<task-id> \
  -H "Authorization: Bearer <TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{"status": "completed"}'
```

**On failure:**
```bash
curl -s -X PUT https://investorcenter.ai/api/v1/tasks/<task-id> \
  -H "Authorization: Bearer <TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{"status": "failed"}'
```

### Step 5: Exit

Done. One task executed.

## Error Handling

- **401 Unauthorized** → Re-authenticate (token expired), then retry
- **Skill file not found** → Mark task as `failed`, exit
- **Skill execution fails** → Mark task as `failed`, exit
- **Network error** → Wait 10 seconds, retry up to 3 times, then mark as `failed`

## Filtering by Task Type

To only grab tasks of a specific type:

```bash
curl -s -X POST https://investorcenter.ai/api/v1/tasks/next \
  -H "Authorization: Bearer <TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{"task_type": "scrape_ycharts_keystats"}'
```

## Summary

```
authenticate → claim next task → load skill → execute → update status → exit
```

The task-runner is a dispatcher. It doesn't know how to execute specific task types — it just loads the right skill and follows its instructions.
