# OpenClaw Skills

Private skills for InvestorCenter.ai automation tasks.

## Skills

- **scrape-seekingalpha** — Extract ticker data, metrics, sentiment, and articles from SeekingAlpha
- **scrape-ycharts** — Pull financial data, charts, and comparisons from YCharts
- **reddit-summarizer** — Aggregate and summarize Reddit posts for market sentiment

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
