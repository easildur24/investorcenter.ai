# OpenClaw Skills

Private skills for InvestorCenter.ai automation tasks.

## Skills

- **scrape-seekingalpha** — Extract ticker data, metrics, sentiment, and articles from SeekingAlpha
- **scrape-ycharts** — Pull financial data, charts, and comparisons from YCharts
- **scrape-ycharts-keystats** — Extract 100+ financial metrics from YCharts Key Stats page and upload to ingestion API
- **scrape-ycharts-financials** — Extract historical financial statements (income statement, balance sheet, cash flow) from YCharts and upload each period to ingestion API
- **scrape-ycharts-analyst-estimates** — Extract analyst estimates, price targets, and recommendations from YCharts estimates page
- **scrape-ycharts-valuation** — Extract current and historical valuation ratios from YCharts valuation page
- **scrape-ycharts-performance** — Extract stock performance, returns, risk metrics, and peer comparisons from YCharts performance page
- **reddit-summarizer** — Aggregate and summarize Reddit posts for market sentiment
- **data-ingestion** — Upload raw scraped data to S3 via the data ingestion API
- **task-runner** — Meta-skill: authenticate, pull tasks from queue, execute the corresponding skill, report results

## Usage

These skills are loaded by OpenClaw agents pulling from a task queue. Each skill contains:
- `SKILL.md` — Instructions and context
- Scripts/automation logic
- Selectors, rate limiting, error handling
- Output format specifications

## Installation

On each OpenClaw instance:

```bash
cd ~/.openclaw/workspace
git clone git@github.com:easildur24/investorcenter.ai.git
```

Or pull updates:

```bash
cd ~/.openclaw/workspace/investorcenter.ai
git pull
```

Agents can reference skills via:
```
/Users/larryli/.openclaw/workspace/investorcenter.ai/skills/<skill-name>/SKILL.md
```

---

*Private repo — not published to ClawHub.*
