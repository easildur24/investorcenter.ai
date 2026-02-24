# Task Runner - Usage Guide

## Quick Start

### Via Cron (Recommended)

Set up a cron job that spawns an isolated session to run one task:

```bash
# Every 10 minutes during business hours
*/10 9-18 * * * /usr/local/bin/openclaw sessions spawn --agent main --task "Run next task from queue using task-runner skill" --cleanup delete
```

Or use the cron tool from within OpenClaw:

```javascript
{
  "name": "Task Runner",
  "schedule": {
    "kind": "cron",
    "expr": "*/10 9-18 * * *",  // Every 10 minutes, 9 AM - 6 PM
    "tz": "America/Los_Angeles"
  },
  "payload": {
    "kind": "agentTurn",
    "message": "Claim and execute next task from queue using task-runner skill. Single execution only - claim one task, run it, mark complete/failed, then exit."
  },
  "sessionTarget": "isolated",
  "enabled": true
}
```

### Manual Execution

From OpenClaw chat:

```
Run the next task from the task queue using the task-runner skill
```

Or via sessions_spawn:

```javascript
sessions_spawn({
  task: "Claim next task, execute it, and exit",
  cleanup: "delete"
})
```

## How It Works

1. **Cron triggers** → OpenClaw spawns isolated session
2. **Session reads** `task-runner/SKILL.md`
3. **Agent:**
   - Claims next task via API
   - Opens browser to YCharts page
   - Captures snapshot
   - Extracts 84 metrics
   - Ingests data
   - Marks task complete
   - **Exits**
4. **Cron waits** 10 minutes → repeat

## Why Isolated Sessions?

- **No context bloat:** Each task starts fresh
- **Clean failures:** Task errors don't crash main session
- **Token efficiency:** Only load skill context when needed
- **Parallel safe:** Multiple cron jobs won't conflict

## Monitoring

### Check Task Queue Status

```bash
curl -H "Authorization: Bearer <token>" \
  https://investorcenter.ai/api/v1/tasks?status=pending | jq '.data | length'
```

### View Recent Task Results

```bash
curl -H "Authorization: Bearer <token>" \
  https://investorcenter.ai/api/v1/tasks?status=completed&limit=10 | jq '.data[] | {ticker: .params.ticker, completed_at: .completed_at}'
```

### Check S3 Ingestion

```bash
aws s3 ls s3://investorcenter-raw-data/ycharts/key_stats/ --recursive | tail -20
```

## Configuration

### Environment Variables (optional)

Create `~/.openclaw/workspace/.env`:

```bash
WORKER_EMAIL=nikola@investorcenter.ai
WORKER_PASSWORD=<password>
API_BASE_URL=https://investorcenter.ai/api/v1
BROWSER_PROFILE=openclaw
```

### Default Credentials

If no env vars set, uses credentials from `TOOLS.md`.

## Troubleshooting

### "Browser not available"

The `run_task.py` script needs OpenClaw browser access. Don't run it directly:

```bash
# ❌ This won't work:
python3 ~/.openclaw/workspace/skills/task-runner/run_task.py

# ✅ Do this instead:
# From OpenClaw chat or via cron/sessions_spawn
```

### "Task stuck in 'in_progress'"

If a session crashes mid-task:

```bash
# Reset task to pending
curl -X PUT https://investorcenter.ai/api/v1/tasks/{task_id} \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"status": "pending"}'
```

### "No tasks in queue"

Normal - queue is empty. Tasks are created separately via the task creation API.

## Task Creation (Separate Step)

Tasks are **not** created by this runner - they're created separately:

```bash
# Create YCharts scraping tasks for S&P 500
curl -X POST https://investorcenter.ai/api/v1/tasks/batch \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "tasks": [
      {"type": "ycharts_key_stats", "params": {"ticker": "AAPL"}},
      {"type": "ycharts_key_stats", "params": {"ticker": "MSFT"}},
      {"type": "ycharts_key_stats", "params": {"ticker": "GOOGL"}}
    ]
  }'
```

## Architecture

```
┌─────────────┐
│ Cron (10min)│
└──────┬──────┘
       │
       ▼
┌─────────────────────┐
│ OpenClaw Gateway    │
│ (sessions_spawn)    │
└──────┬──────────────┘
       │
       ▼
┌─────────────────────┐
│ Isolated Session    │
│ - Load SKILL.md     │
│ - Claim task API    │
│ - Browser control   │
│ - Extract metrics   │
│ - Ingest API        │
│ - Mark complete     │
│ - EXIT              │
└─────────────────────┘
```

## Files

- `SKILL.md` - Skill documentation (loaded by OpenClaw)
- `run_task.py` - Core logic (browser automation stub)
- `README.md` - This file
- `parse_helpers.py` - Symlink to `../scrape-ycharts-keystats/parse_helpers.py`

## Next Steps

1. ✅ Skill created
2. ⏳ Set up cron job (via `cron` tool)
3. ⏳ Create initial task batch
4. ⏳ Monitor first few runs
5. ⏳ Adjust cron frequency based on task volume
