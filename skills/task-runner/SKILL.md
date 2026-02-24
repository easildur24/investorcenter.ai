# Task Runner

## What This Does

Grabs the next pending task from the queue, executes it, marks it complete/failed, then exits.

**One task per invocation. No loops. No scheduling.**

## How to Use

Just invoke the skill:

```
Grab and run the next task from the queue
```

The agent will:
1. Claim next task from `/api/v1/tasks/next`
2. Execute it (YCharts scraping, etc.)
3. Mark status: `completed` or `failed`
4. Exit

If queue is empty, exits gracefully.

## Task Types Supported

### `ycharts_key_stats`

Scrapes YCharts key statistics page for a ticker.

**Process:**
1. Open `https://ycharts.com/companies/{TICKER}/key_stats`
2. Wait 3 seconds for page load
3. Capture browser snapshot
4. Extract 84 metrics (revenue, PE ratio, cash flow, etc.)
5. POST to `/api/v1/ingest/ycharts/key_stats/{TICKER}`

**Metrics extracted:** All fields defined in `data-ingestion-service/schemas/ycharts/key_stats.json`

## Authentication

Uses worker credentials from `TOOLS.md`:
- Email: `nikola@investorcenter.ai`
- Gets fresh token on each run

## Error Handling

**Browser errors:**
- Tab not found → Reopen browser and retry
- Timeout → Fail task, exit

**API errors:**
- 401/403 → Re-authenticate once, then fail
- 400 → Schema error, fail task
- 500 → Server error, fail task

**Failed tasks stay in queue** as `failed` status (won't be retried automatically).

## Dependencies

- `scrape-ycharts-keystats/parse_helpers.py` - Metric parsing functions
- OpenClaw browser control
- Task queue API access

## Exit Codes

- `0` - Success (task completed or queue empty)
- `1` - Failure (task failed, will be marked as failed)

## Notes

This skill does **one thing**: execute one task then exit.

Scheduling (how often to run) is handled externally via cron or manual invocation.
