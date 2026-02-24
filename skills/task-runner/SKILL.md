# Task Runner Skill

## Purpose
Execute YCharts data scraping tasks from the task queue. Designed for **single-task execution** - claim one task, run it, exit. Managed by cron for reliability.

## Architecture

**Single-Execution Pattern:**
1. Claim next pending task
2. Execute task (scrape + ingest)
3. Mark complete/failed
4. Exit

**Why Single-Execution?**
- Cron manages scheduling (every 5-10 minutes)
- Sessions don't hang for hours
- Failures are isolated per-task
- Easy to monitor and restart

## Usage

### One-Shot Task Execution
```bash
# Claim next task, run it, exit
python3 ~/.openclaw/workspace/skills/task-runner/run_task.py
```

### Cron Setup (Recommended)
```bash
# Every 10 minutes during business hours (9 AM - 6 PM PST)
*/10 9-18 * * * cd ~/.openclaw/workspace && python3 skills/task-runner/run_task.py >> logs/task-runner.log 2>&1
```

## Task Workflow

### 1. Claim Task
```bash
POST /api/v1/tasks/next
Authorization: Bearer <token>
```

**Success:** Returns task object with `id`, `type`, `params.ticker`  
**Empty queue:** 204 No Content → exit gracefully

### 2. Execute Task

**For `ycharts_key_stats` tasks:**
1. Open browser: `https://ycharts.com/companies/{TICKER}/key_stats`
2. Wait 3 seconds for page load
3. Capture snapshot (maxChars: 100000)
4. Extract all 84 metrics from snapshot
5. Build ingestion payload
6. POST to `/api/v1/ingest/ycharts/key_stats/{TICKER}`

### 3. Mark Complete/Failed

**Success:**
```bash
PUT /api/v1/tasks/{task_id}
{"status": "completed"}
```

**Failure:**
```bash
PUT /api/v1/tasks/{task_id}
{"status": "failed", "error": "<error message>"}
```

## Files

- `SKILL.md` - This file
- `run_task.py` - Main execution script (single-task mode)
- `parse_helpers.py` - Data parsing utilities (from scrape-ycharts-keystats)

## Error Handling

**Browser errors:**
- Tab not found → Reopen browser
- Page load timeout → Retry once, then fail task

**Ingestion errors:**
- 401/403 → Token expired, re-authenticate
- 400 → Schema validation error, log and fail task
- 500 → Server error, fail task (will retry on next cron run)

**Task queue errors:**
- 204 No Content → Normal exit (queue empty)
- Network errors → Log and exit (cron will retry)

## Configuration

**Environment Variables:**
```bash
WORKER_EMAIL=nikola@investorcenter.ai
WORKER_PASSWORD=<from TOOLS.md>
API_BASE_URL=https://investorcenter.ai/api/v1
BROWSER_PROFILE=openclaw
```

**Defaults (if not set):**
- Uses credentials from TOOLS.md
- Browser: `openclaw` profile
- API: `https://investorcenter.ai/api/v1`

## Monitoring

**Success indicators:**
```bash
✅ TICKER: s3://path/to/file.json
Task <id> completed
```

**Failure indicators:**
```bash
❌ TICKER: <error message>
Task <id> failed: <reason>
```

**Check logs:**
```bash
tail -f ~/.openclaw/workspace/logs/task-runner.log
```

**Check queue status:**
```bash
curl -H "Authorization: Bearer <token>" \
  https://investorcenter.ai/api/v1/tasks?status=pending
```

## Integration with scrape-ycharts-keystats

This skill **depends on** the `scrape-ycharts-keystats` skill for:
- `parse_helpers.py` - Metric parsing functions
- Field mapping knowledge
- YCharts-specific date/number formats

**Import parse helpers:**
```python
import sys
sys.path.append('~/.openclaw/workspace/skills/scrape-ycharts-keystats')
from parse_helpers import parse_dollar_amount, parse_percentage, parse_float
```

## Task Types

Currently supports:
- `ycharts_key_stats` - Scrape YCharts key stats page for a ticker

**Future task types:**
- `ycharts_financials` - Income statement, balance sheet, cash flow
- `sec_filings` - 10-K, 10-Q filings
- `earnings_transcripts` - Quarterly earnings calls

## Best Practices

✅ **DO:**
- Exit after each task (let cron handle next run)
- Log all actions (timestamps, ticker, status)
- Mark tasks as failed if anything goes wrong
- Use fresh auth token for each run

❌ **DON'T:**
- Run in a loop (defeats the purpose)
- Retry failed tasks immediately (let cron handle it)
- Keep browser tabs open between runs
- Assume token is still valid

## Troubleshooting

**"No tasks in queue"**
- Normal - just means queue is empty
- Check if all tasks completed or failed

**"Browser tab not found"**
- Previous run didn't close browser properly
- Script will reopen - should auto-recover

**"401 Unauthorized"**
- Token expired (tokens last ~1 hour)
- Script auto-refreshes on each run

**"Task stuck in 'in_progress'"**
- Previous run crashed without marking complete/failed
- Manually reset: `PUT /api/v1/tasks/{id}` with `status=pending`

## Notes

- Created: 2026-02-23
- Replaces: Long-running loop-based task runner
- Pattern: Single-execution, cron-managed
