# Technical Specification: Social Sentiment & Reddit Trends Redesign

**Product:** InvestorCenter.AI  
**Version:** 1.0  
**Date:** February 2026  
**Status:** Draft for Engineering Review  
**Author:** Engineering  
**Related PRD:** `docs/prd-social-sentiment-redesign.md`

---

## Table of Contents

1. [Overview & Scope](#1-overview--scope)
2. [System Architecture](#2-system-architecture)
3. [Data Models & Database Schema](#3-data-models--database-schema)
4. [OpenClaw Data Pipeline](#4-openclaw-data-pipeline)
5. [Sentiment Processing Engine](#5-sentiment-processing-engine)
6. [Backend API Design](#6-backend-api-design)
7. [Frontend Components](#7-frontend-components)
8. [Alerts System](#8-alerts-system)
9. [Caching & Performance](#9-caching--performance)
10. [Feature Gating & Tiering](#10-feature-gating--tiering)
11. [Bug Fixes (Phase 1)](#11-bug-fixes-phase-1)
12. [Testing Strategy](#12-testing-strategy)
13. [Deployment & Rollout](#13-deployment--rollout)
14. [Observability & Monitoring](#14-observability--monitoring)
15. [Design Decisions & ADRs](#15-design-decisions--adrs)

---

## 1. Overview & Scope

### 1.1 Purpose

This technical specification translates the Social Sentiment & Reddit Trends PRD into implementation-level detail. It defines the data models, API contracts, component architecture, processing pipelines, and testing requirements needed for engineering to execute Phases 1–4 of the redesign.

### 1.2 Scope Mapping

| PRD Section | Tech Spec Coverage |
|---|---|
| 5a. Redesigned Reddit Trends Page | Sections 6.1, 7.1, 7.2 |
| 5b. Enhanced Sentiment Detail Page | Sections 6.2, 7.3, 7.4 |
| 5c. Sentiment Widget on Ticker Pages | Section 7.5 |
| 5d. Sentiment-Based Alerts | Section 8 |
| 5e. Bug Fixes | Section 11 |
| 15. OpenClaw Data Pipeline | Section 4 |
| 16. Product Evolution | Sections 4.6, 5.5 (extension points) |

### 1.3 Assumptions

- InvestorCenter frontend is built with **React** (Next.js or similar SSR framework).
- Backend API layer is **Node.js** (Express/Fastify) or **Python** (FastAPI). This spec is language-agnostic but uses Python pseudocode for data pipeline and TypeScript for frontend examples.
- Primary database is **PostgreSQL** with potential use of **Redis** for caching and **TimescaleDB** extension (or standalone) for time-series sentiment data.
- Market data (price, % change) is already available via an existing internal API or third-party provider (e.g., Polygon.io, Alpha Vantage, Financial Modeling Prep).
- Authentication and user management are already implemented.
- OpenClaw is installed and configured on the server environment with cron-job scheduling capability.

### 1.4 Technology Stack Summary

```
┌─────────────────────────────────────────────────────────────┐
│ FRONTEND                                                     │
│ React / Next.js, TailwindCSS, Recharts/Lightweight Charts   │
│ React Query (TanStack Query) for data fetching + caching    │
├─────────────────────────────────────────────────────────────┤
│ BACKEND API                                                  │
│ FastAPI (Python) or Express/Fastify (Node.js)               │
│ REST endpoints + optional WebSocket for live refresh         │
├─────────────────────────────────────────────────────────────┤
│ DATA PIPELINE                                                │
│ OpenClaw (scheduled cron jobs) → Python processing scripts  │
│ FinBERT / Hugging Face Transformers for NLP sentiment        │
├─────────────────────────────────────────────────────────────┤
│ DATA STORES                                                  │
│ PostgreSQL (primary) + TimescaleDB (time-series)            │
│ Redis (caching, session, rate limiting)                      │
├─────────────────────────────────────────────────────────────┤
│ INFRASTRUCTURE                                               │
│ Docker Compose (dev) → single VPS or cloud VM (prod)        │
│ Nginx reverse proxy, Let's Encrypt SSL                       │
└─────────────────────────────────────────────────────────────┘
```

---

## 2. System Architecture

### 2.1 High-Level Architecture Diagram

```
                                ┌──────────────────┐
                                │   InvestorCenter  │
                                │   Frontend (SPA)  │
                                └────────┬─────────┘
                                         │ HTTPS
                                         ▼
                                ┌──────────────────┐
                                │   API Gateway /   │
                                │   Backend Server  │
                                └──┬──────┬────┬───┘
                                   │      │    │
                          ┌────────┘      │    └────────┐
                          ▼               ▼             ▼
                   ┌────────────┐  ┌───────────┐  ┌──────────┐
                   │ PostgreSQL │  │   Redis    │  │ Market   │
                   │ + Timescale│  │  Cache     │  │ Data API │
                   └──────┬─────┘  └───────────┘  └──────────┘
                          │
                          │ writes
                          │
              ┌───────────┴───────────┐
              │  Sentiment Processing  │
              │  Engine (batch job)    │
              └───────────┬───────────┘
                          │ reads raw posts
                          │
              ┌───────────┴───────────┐
              │  OpenClaw Data        │
              │  Pipeline (cron)      │
              └───────────────────────┘
                   │         │        │
                   ▼         ▼        ▼
              Reddit    StockTwits  Twitter/X
              (JSON)    (web_fetch) (web_fetch)
```

### 2.2 Data Flow

```
1. INGEST:  OpenClaw cron → scrapes Reddit/social → writes raw posts to `raw_posts` table
2. PROCESS: Sentiment engine (cron, every 15 min) → reads raw_posts → NLP classification
            → writes to `post_sentiment`, `ticker_sentiment_snapshots`, `trending_rankings`
3. SERVE:   API reads from computed tables + Redis cache → returns to frontend
4. ALERT:   Alert evaluator (cron, every 15 min) → reads latest sentiment snapshots
            → compares against user alert rules → dispatches notifications
```

---

## 3. Data Models & Database Schema

### 3.1 Entity Relationship Overview

```
raw_posts ──────┐
                 ├──▶ post_sentiment ──▶ ticker_sentiment_snapshots ──▶ trending_rankings
subreddits ─────┘                                                           │
                                                                            ▼
tickers ◀─────────────────────────────────────────────────────── trending_rankings
   │
   ├──▶ ticker_sentiment_history (time-series, TimescaleDB hypertable)
   │
   └──▶ user_sentiment_alerts ──▶ alert_notifications
```

### 3.2 Table Definitions

#### `subreddits`

Tracks configured data sources.

```sql
CREATE TABLE subreddits (
    id              SERIAL PRIMARY KEY,
    name            VARCHAR(100) NOT NULL UNIQUE,  -- e.g., 'wallstreetbets'
    display_name    VARCHAR(100) NOT NULL,         -- e.g., 'r/wallstreetbets'
    platform        VARCHAR(20) NOT NULL DEFAULT 'reddit',  -- 'reddit', 'stocktwits', 'twitter', 'news'
    scrape_frequency_min INTEGER NOT NULL DEFAULT 15,
    is_active       BOOLEAN NOT NULL DEFAULT true,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Seed data
INSERT INTO subreddits (name, display_name, platform, scrape_frequency_min) VALUES
    ('wallstreetbets', 'r/wallstreetbets', 'reddit', 10),
    ('stocks', 'r/stocks', 'reddit', 15),
    ('investing', 'r/investing', 'reddit', 15),
    ('options', 'r/options', 'reddit', 15);
```

#### `raw_posts`

Raw ingested data from OpenClaw. Append-only, high-volume.

```sql
CREATE TABLE raw_posts (
    id                  BIGSERIAL PRIMARY KEY,
    external_id         VARCHAR(50) NOT NULL,          -- Reddit post ID (e.g., 't3_abc123')
    subreddit_id        INTEGER NOT NULL REFERENCES subreddits(id),
    platform            VARCHAR(20) NOT NULL DEFAULT 'reddit',
    title               TEXT NOT NULL,
    body                TEXT,                           -- selftext, nullable for link posts
    author              VARCHAR(100),
    author_karma        INTEGER,                        -- for bot/spam filtering
    author_account_age_days INTEGER,                    -- for bot/spam filtering
    score               INTEGER NOT NULL DEFAULT 0,     -- upvotes - downvotes
    upvotes             INTEGER NOT NULL DEFAULT 0,
    num_comments        INTEGER NOT NULL DEFAULT 0,
    url                 TEXT,                           -- permalink to original post
    post_created_at     TIMESTAMPTZ NOT NULL,           -- when the post was made on Reddit
    scraped_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    is_processed        BOOLEAN NOT NULL DEFAULT false,
    UNIQUE (external_id, platform)
);

CREATE INDEX idx_raw_posts_unprocessed ON raw_posts (is_processed) WHERE is_processed = false;
CREATE INDEX idx_raw_posts_scraped_at ON raw_posts (scraped_at DESC);
CREATE INDEX idx_raw_posts_subreddit_id ON raw_posts (subreddit_id, scraped_at DESC);
```

#### `post_sentiment`

Processed sentiment for each post. One row per post per detected ticker.

```sql
CREATE TABLE post_sentiment (
    id              BIGSERIAL PRIMARY KEY,
    raw_post_id     BIGINT NOT NULL REFERENCES raw_posts(id),
    ticker          VARCHAR(10) NOT NULL,              -- detected ticker symbol
    sentiment_label VARCHAR(10) NOT NULL,              -- 'bullish', 'bearish', 'neutral'
    sentiment_score FLOAT NOT NULL,                    -- -1.0 (bearish) to +1.0 (bullish)
    confidence      FLOAT NOT NULL,                    -- 0.0 to 1.0, NLP model confidence
    processed_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (raw_post_id, ticker)
);

CREATE INDEX idx_post_sentiment_ticker ON post_sentiment (ticker, processed_at DESC);
CREATE INDEX idx_post_sentiment_label ON post_sentiment (ticker, sentiment_label);
```

#### `ticker_sentiment_snapshots`

Aggregated sentiment per ticker, computed every processing cycle. This is the primary table the API reads from.

```sql
CREATE TABLE ticker_sentiment_snapshots (
    id                      BIGSERIAL PRIMARY KEY,
    ticker                  VARCHAR(10) NOT NULL,
    snapshot_time           TIMESTAMPTZ NOT NULL,
    time_range              VARCHAR(10) NOT NULL,      -- '1d', '7d', '14d', '30d'
    
    -- Mention metrics
    mention_count           INTEGER NOT NULL DEFAULT 0,
    total_upvotes           INTEGER NOT NULL DEFAULT 0,
    total_comments          INTEGER NOT NULL DEFAULT 0,
    unique_posts            INTEGER NOT NULL DEFAULT 0,
    
    -- Sentiment breakdown
    bullish_count           INTEGER NOT NULL DEFAULT 0,
    neutral_count           INTEGER NOT NULL DEFAULT 0,
    bearish_count           INTEGER NOT NULL DEFAULT 0,
    bullish_pct             FLOAT NOT NULL DEFAULT 0,
    neutral_pct             FLOAT NOT NULL DEFAULT 0,
    bearish_pct             FLOAT NOT NULL DEFAULT 0,
    sentiment_score         FLOAT NOT NULL DEFAULT 0,  -- weighted avg: -1.0 to +1.0
    sentiment_label         VARCHAR(10) NOT NULL,       -- derived: 'bullish'/'bearish'/'neutral'
    
    -- Velocity metrics
    mention_velocity_1h     FLOAT,                     -- mentions per hour (last 1h vs prior 1h)
    sentiment_velocity_24h  FLOAT,                     -- sentiment delta over past 24h
    
    -- Composite score
    composite_score         FLOAT NOT NULL DEFAULT 0,  -- weighted(mentions, upvotes, comments, sentiment)
    
    -- Subreddit distribution (JSONB for flexibility)
    subreddit_distribution  JSONB,                     -- {"wallstreetbets": 45, "stocks": 30, ...}
    
    -- Ranking
    rank                    INTEGER,
    previous_rank           INTEGER,
    rank_change             INTEGER,                   -- computed as integer: previous_rank - rank
    
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    UNIQUE (ticker, snapshot_time, time_range)
);

CREATE INDEX idx_sentiment_snapshots_ticker ON ticker_sentiment_snapshots (ticker, time_range, snapshot_time DESC);
CREATE INDEX idx_sentiment_snapshots_rank ON ticker_sentiment_snapshots (time_range, snapshot_time DESC, rank ASC);
```

#### `ticker_sentiment_history` (TimescaleDB hypertable)

Time-series data for historical charts. Uses TimescaleDB for efficient time-range queries and automatic partitioning.

```sql
CREATE TABLE ticker_sentiment_history (
    time                TIMESTAMPTZ NOT NULL,
    ticker              VARCHAR(10) NOT NULL,
    sentiment_score     FLOAT NOT NULL,            -- -1.0 to +1.0
    bullish_pct         FLOAT NOT NULL,
    mention_count       INTEGER NOT NULL,
    composite_score     FLOAT NOT NULL
);

-- Convert to TimescaleDB hypertable (partition by day)
SELECT create_hypertable('ticker_sentiment_history', 'time', chunk_time_interval => INTERVAL '1 day');

CREATE INDEX idx_sentiment_history_ticker ON ticker_sentiment_history (ticker, time DESC);

-- Retention policy: keep 12 months of data, compress after 7 days
SELECT add_retention_policy('ticker_sentiment_history', INTERVAL '12 months');
SELECT add_compression_policy('ticker_sentiment_history', INTERVAL '7 days');
```

#### `user_sentiment_alerts`

User-configured alert rules.

```sql
CREATE TABLE user_sentiment_alerts (
    id              SERIAL PRIMARY KEY,
    user_id         INTEGER NOT NULL REFERENCES users(id),
    ticker          VARCHAR(10) NOT NULL,
    alert_type      VARCHAR(30) NOT NULL,              -- 'mention_spike', 'sentiment_shift', 'trending_entry', 'sentiment_velocity'
    
    -- Configuration (varies by type, stored as JSONB)
    config          JSONB NOT NULL,
    -- Examples:
    -- mention_spike:      {"threshold": 200, "window": "24h"}
    -- sentiment_shift:    {"direction": "bearish"}
    -- trending_entry:     {"top_n": 10}
    -- sentiment_velocity: {"delta_pct": -20, "window": "24h"}
    
    is_active       BOOLEAN NOT NULL DEFAULT true,
    last_triggered_at TIMESTAMPTZ,
    cooldown_minutes  INTEGER NOT NULL DEFAULT 60,     -- prevent spam
    delivery_channels TEXT[] NOT NULL DEFAULT '{in_app}', -- '{in_app, email}'
    
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_alerts_user ON user_sentiment_alerts (user_id, is_active);
CREATE INDEX idx_alerts_ticker ON user_sentiment_alerts (ticker, alert_type, is_active);
```

#### `alert_notifications`

Dispatched alert history.

```sql
CREATE TABLE alert_notifications (
    id              BIGSERIAL PRIMARY KEY,
    alert_id        INTEGER NOT NULL REFERENCES user_sentiment_alerts(id),
    user_id         INTEGER NOT NULL REFERENCES users(id),
    ticker          VARCHAR(10) NOT NULL,
    message         TEXT NOT NULL,
    channel         VARCHAR(20) NOT NULL,               -- 'in_app', 'email'
    is_read         BOOLEAN NOT NULL DEFAULT false,
    triggered_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    delivered_at    TIMESTAMPTZ
);

CREATE INDEX idx_notifications_user ON alert_notifications (user_id, is_read, triggered_at DESC);
```

### 3.3 Migration Strategy

All schema changes use versioned migration files (e.g., Alembic for Python, Knex/Prisma for Node). Migrations are applied in order and are idempotent.

```
migrations/
├── 001_create_subreddits.sql
├── 002_create_raw_posts.sql
├── 003_create_post_sentiment.sql
├── 004_create_ticker_sentiment_snapshots.sql
├── 005_create_ticker_sentiment_history_hypertable.sql
├── 006_create_user_sentiment_alerts.sql
├── 007_create_alert_notifications.sql
└── 008_seed_subreddits.sql
```

---

## 4. OpenClaw Data Pipeline

### 4.1 Pipeline Overview

The OpenClaw data pipeline is a scheduled batch process that scrapes social media posts and ingests them into the `raw_posts` table. It runs as a series of cron jobs on the host machine where OpenClaw is installed.

### 4.2 Scraper Script

A wrapper script around OpenClaw's reddit-scraper skill that handles output parsing, deduplication, and database insertion.

```python
# scripts/pipeline/scrape_reddit.py
"""
OpenClaw Reddit scraper wrapper.
Called by cron every 10-15 minutes per subreddit.
"""
import subprocess
import json
import sys
from datetime import datetime, timezone
from db import get_db_session
from models import RawPost, Subreddit

def scrape_subreddit(subreddit_name: str, sort: str = "hot", limit: int = 100) -> list[dict]:
    """Execute OpenClaw reddit-scraper and return parsed posts."""
    cmd = [
        "python3",
        "/root/.openclaw/skills/reddit-scraper/scripts/reddit_scraper.py",  # adjust path
        "--subreddit", subreddit_name,
        "--sort", sort,
        "--limit", str(limit),
        "--json"
    ]
    
    result = subprocess.run(cmd, capture_output=True, text=True, timeout=60)
    
    if result.returncode != 0:
        raise RuntimeError(f"OpenClaw scrape failed: {result.stderr}")
    
    return json.loads(result.stdout)


def ingest_posts(subreddit_name: str, posts: list[dict]):
    """Insert posts into raw_posts table, skipping duplicates."""
    session = get_db_session()
    subreddit = session.query(Subreddit).filter_by(name=subreddit_name).first()
    
    inserted = 0
    for post in posts:
        external_id = post.get("id") or post.get("name")  # Reddit post ID
        
        # Skip if already ingested
        exists = session.query(RawPost).filter_by(
            external_id=external_id, platform="reddit"
        ).first()
        if exists:
            # Update score/upvotes (these change over time)
            exists.score = post.get("score", exists.score)
            exists.upvotes = post.get("ups", exists.upvotes)
            exists.num_comments = post.get("num_comments", exists.num_comments)
            continue
        
        raw_post = RawPost(
            external_id=external_id,
            subreddit_id=subreddit.id,
            platform="reddit",
            title=post.get("title", ""),
            body=post.get("selftext", ""),
            author=post.get("author", "[deleted]"),
            author_karma=post.get("author_karma"),
            score=post.get("score", 0),
            upvotes=post.get("ups", 0),
            num_comments=post.get("num_comments", 0),
            url=f"https://reddit.com{post.get('permalink', '')}",
            post_created_at=datetime.fromtimestamp(
                post.get("created_utc", 0), tz=timezone.utc
            ),
        )
        session.add(raw_post)
        inserted += 1
    
    session.commit()
    print(f"[{subreddit_name}] Ingested {inserted} new posts, updated {len(posts) - inserted}")


def main():
    subreddit_name = sys.argv[1]  # e.g., "wallstreetbets"
    sort = sys.argv[2] if len(sys.argv) > 2 else "hot"
    
    posts = scrape_subreddit(subreddit_name, sort=sort)
    ingest_posts(subreddit_name, posts)


if __name__ == "__main__":
    main()
```

### 4.3 Cron Schedule

```cron
# /etc/cron.d/investorcenter-scraper

# High-priority subreddits: every 10 minutes
*/10 * * * * root python3 /opt/investorcenter/scripts/pipeline/scrape_reddit.py wallstreetbets hot 2>> /var/log/ic-scraper.log
*/10 * * * * root python3 /opt/investorcenter/scripts/pipeline/scrape_reddit.py wallstreetbets new 2>> /var/log/ic-scraper.log

# Medium-priority subreddits: every 15 minutes
*/15 * * * * root python3 /opt/investorcenter/scripts/pipeline/scrape_reddit.py stocks hot 2>> /var/log/ic-scraper.log
*/15 * * * * root python3 /opt/investorcenter/scripts/pipeline/scrape_reddit.py investing hot 2>> /var/log/ic-scraper.log
*/15 * * * * root python3 /opt/investorcenter/scripts/pipeline/scrape_reddit.py options hot 2>> /var/log/ic-scraper.log

# Phase 3+ subreddits: every 30 minutes
# */30 * * * * root python3 /opt/investorcenter/scripts/pipeline/scrape_reddit.py SecurityAnalysis hot 2>> /var/log/ic-scraper.log
# */30 * * * * root python3 /opt/investorcenter/scripts/pipeline/scrape_reddit.py ValueInvesting hot 2>> /var/log/ic-scraper.log

# Sentiment processing: every 15 minutes (runs after scrapers)
3,18,33,48 * * * * root python3 /opt/investorcenter/scripts/pipeline/process_sentiment.py 2>> /var/log/ic-sentiment.log

# Trending ranking computation: every 15 minutes (runs after sentiment)
6,21,36,51 * * * * root python3 /opt/investorcenter/scripts/pipeline/compute_rankings.py 2>> /var/log/ic-rankings.log

# Alert evaluation: every 15 minutes (runs after rankings)
9,24,39,54 * * * * root python3 /opt/investorcenter/scripts/pipeline/evaluate_alerts.py 2>> /var/log/ic-alerts.log
```

### 4.4 Ticker Extraction

Before sentiment analysis, posts must be scanned for ticker symbols.

```python
# scripts/pipeline/ticker_extractor.py
import re

# Pre-loaded set of valid tickers (refresh daily from market data API)
VALID_TICKERS: set[str] = load_valid_tickers()  # ~8,000 US equities

# Common false positives to exclude
FALSE_POSITIVES = {
    "I", "A", "AM", "PM", "IT", "AT", "ON", "OR", "AN", "ALL", "ARE", "FOR",
    "HAS", "HE", "CEO", "DD", "IMO", "YOLO", "FOMO", "FYI", "HODL", "EPS",
    "PE", "IPO", "GDP", "CPI", "SEC", "FBI", "AI", "WSB", "ETF", "OTC",
    "MACD", "RSI", "EMA", "SMA", "ATH", "ATL", "YTD", "QE", "FED", "FDIC",
    "USA", "UK", "EU", "IMF", "OP", "TL", "DR", "LOL", "WTF", "NFT",
}

# Patterns: $AAPL, AAPL (standalone uppercase 1-5 chars)
TICKER_PATTERN = re.compile(r'\$([A-Z]{1,5})\b|\b([A-Z]{1,5})\b')

def extract_tickers(title: str, body: str = "") -> list[str]:
    """Extract probable ticker symbols from post title and body."""
    text = f"{title} {body or ''}"
    matches = set()
    
    for match in TICKER_PATTERN.finditer(text):
        ticker = match.group(1) or match.group(2)
        if ticker in VALID_TICKERS and ticker not in FALSE_POSITIVES:
            matches.add(ticker)
    
    # Prioritize $-prefixed tickers (high confidence)
    dollar_prefixed = set()
    for match in re.finditer(r'\$([A-Z]{1,5})\b', text):
        dollar_prefixed.add(match.group(1))
    
    # If we found dollar-prefixed tickers, also include non-prefixed valid ones
    # If no dollar-prefixed, be more conservative (require 2+ char tickers)
    if not dollar_prefixed:
        matches = {t for t in matches if len(t) >= 2}
    
    return list(matches)
```

### 4.5 Pipeline Health Monitoring

```python
# scripts/pipeline/health_check.py
"""
Exposes /health/pipeline endpoint data.
Called by the backend API server to surface data freshness.
"""

def get_pipeline_health() -> dict:
    """Returns pipeline health status for the freshness indicator."""
    session = get_db_session()
    
    latest_scrape = session.execute(
        "SELECT MAX(scraped_at) FROM raw_posts"
    ).scalar()
    
    latest_snapshot = session.execute(
        "SELECT MAX(snapshot_time) FROM ticker_sentiment_snapshots"
    ).scalar()
    
    unprocessed_count = session.execute(
        "SELECT COUNT(*) FROM raw_posts WHERE is_processed = false"
    ).scalar()
    
    return {
        "last_scrape_at": latest_scrape.isoformat() if latest_scrape else None,
        "last_snapshot_at": latest_snapshot.isoformat() if latest_snapshot else None,
        "unprocessed_posts": unprocessed_count,
        "status": "healthy" if (
            latest_scrape and (datetime.now(timezone.utc) - latest_scrape).seconds < 1800
        ) else "stale"
    }
```

### 4.6 Extension Points for Multi-Source (Phase 4)

The `raw_posts` table and `subreddits` table are designed to be platform-agnostic via the `platform` column. Future scrapers for StockTwits, Twitter/X, and news follow the same pattern:

```python
# scripts/pipeline/scrape_stocktwits.py  (Phase 4)
def scrape_stocktwits(ticker: str) -> list[dict]:
    """Use OpenClaw web_fetch to scrape StockTwits public stream."""
    cmd = [
        "openclaw", "web_fetch",
        f"https://stocktwits.com/symbol/{ticker}",
        "--json"
    ]
    # ... parse HTML/JSON, extract messages, insert into raw_posts with platform='stocktwits'
```

---

## 5. Sentiment Processing Engine

### 5.1 NLP Model Selection

**Primary model:** FinBERT (ProsusAI/finbert) — a BERT model fine-tuned on financial text for sentiment classification (positive/negative/neutral).

**Why FinBERT:**
- Specifically trained on financial language (earnings calls, analyst reports, financial news).
- Open-source, runs locally (no API cost per inference).
- 3-class output maps directly to our bullish/neutral/bearish taxonomy.
- Inference time: ~10ms per post on CPU, ~2ms on GPU. At 5,000 posts per batch, a CPU-only run completes in under 60 seconds.

**Alternative for higher throughput (Phase 3+):** Distilled FinBERT or a commercial sentiment API (e.g., AWS Comprehend) if processing volume exceeds single-machine capacity.

### 5.2 Processing Script

```python
# scripts/pipeline/process_sentiment.py
"""
Batch sentiment processing job.
Runs every 15 minutes. Processes all unprocessed raw_posts.
"""
from transformers import AutoTokenizer, AutoModelForSequenceClassification
import torch
from ticker_extractor import extract_tickers

# Load model once at startup
MODEL_NAME = "ProsusAI/finbert"
tokenizer = AutoTokenizer.from_pretrained(MODEL_NAME)
model = AutoModelForSequenceClassification.from_pretrained(MODEL_NAME)
model.eval()

LABEL_MAP = {0: "bullish", 1: "neutral", 2: "bearish"}
SCORE_MAP = {"bullish": 1.0, "neutral": 0.0, "bearish": -1.0}


def classify_sentiment(text: str) -> tuple[str, float, float]:
    """Returns (label, score, confidence)."""
    inputs = tokenizer(text, return_tensors="pt", truncation=True, max_length=512)
    
    with torch.no_grad():
        outputs = model(**inputs)
    
    probs = torch.softmax(outputs.logits, dim=1).squeeze()
    predicted_class = torch.argmax(probs).item()
    confidence = probs[predicted_class].item()
    label = LABEL_MAP[predicted_class]
    score = SCORE_MAP[label] * confidence  # scale by confidence
    
    return label, score, confidence


def process_batch():
    session = get_db_session()
    
    # Fetch unprocessed posts in batches
    unprocessed = session.query(RawPost).filter_by(
        is_processed=False
    ).order_by(RawPost.scraped_at.asc()).limit(5000).all()
    
    if not unprocessed:
        print("No unprocessed posts. Exiting.")
        return
    
    for post in unprocessed:
        # Step 1: Extract tickers
        tickers = extract_tickers(post.title, post.body)
        
        if not tickers:
            post.is_processed = True
            continue
        
        # Step 2: Classify sentiment (use title + first 300 chars of body)
        text = post.title
        if post.body:
            text += " " + post.body[:300]
        
        label, score, confidence = classify_sentiment(text)
        
        # Step 3: Create post_sentiment rows (one per ticker mentioned)
        for ticker in tickers:
            ps = PostSentiment(
                raw_post_id=post.id,
                ticker=ticker,
                sentiment_label=label,
                sentiment_score=score,
                confidence=confidence,
            )
            session.add(ps)
        
        post.is_processed = True
    
    session.commit()
    print(f"Processed {len(unprocessed)} posts.")
```

### 5.3 Ranking Computation

```python
# scripts/pipeline/compute_rankings.py
"""
Computes trending rankings and aggregated sentiment snapshots.
Runs every 15 minutes after sentiment processing.
"""
from datetime import datetime, timedelta, timezone

TIME_RANGES = {
    "1d":  timedelta(days=1),
    "7d":  timedelta(days=7),
    "14d": timedelta(days=14),
    "30d": timedelta(days=30),
}

# Composite score weights
W_MENTIONS = 0.35
W_UPVOTES  = 0.25
W_COMMENTS = 0.15
W_SENTIMENT = 0.25

def compute_rankings_for_range(time_range: str, delta: timedelta):
    """Compute snapshot for a single time range."""
    session = get_db_session()
    now = datetime.now(timezone.utc)
    cutoff = now - delta
    
    # Aggregate per ticker
    query = """
        SELECT
            ps.ticker,
            COUNT(DISTINCT rp.id) AS mention_count,
            COALESCE(SUM(rp.upvotes), 0) AS total_upvotes,
            COALESCE(SUM(rp.num_comments), 0) AS total_comments,
            COUNT(DISTINCT rp.id) AS unique_posts,
            COUNT(*) FILTER (WHERE ps.sentiment_label = 'bullish') AS bullish_count,
            COUNT(*) FILTER (WHERE ps.sentiment_label = 'neutral') AS neutral_count,
            COUNT(*) FILTER (WHERE ps.sentiment_label = 'bearish') AS bearish_count,
            AVG(ps.sentiment_score) AS avg_sentiment_score,
            jsonb_object_agg(
                s.name,
                COUNT(rp.id)
            ) FILTER (WHERE s.name IS NOT NULL) AS subreddit_distribution
        FROM post_sentiment ps
        JOIN raw_posts rp ON rp.id = ps.raw_post_id
        JOIN subreddits s ON s.id = rp.subreddit_id
        WHERE rp.post_created_at >= :cutoff
        GROUP BY ps.ticker
        HAVING COUNT(DISTINCT rp.id) >= 3  -- minimum threshold
        ORDER BY COUNT(DISTINCT rp.id) DESC
    """
    
    results = session.execute(query, {"cutoff": cutoff}).fetchall()
    
    # Normalize metrics for composite score
    if not results:
        return
    
    max_mentions = max(r.mention_count for r in results) or 1
    max_upvotes = max(r.total_upvotes for r in results) or 1
    max_comments = max(r.total_comments for r in results) or 1
    
    # Get previous rankings for rank_change calculation
    prev_snapshot = session.execute("""
        SELECT ticker, rank FROM ticker_sentiment_snapshots
        WHERE time_range = :time_range
        AND snapshot_time = (
            SELECT MAX(snapshot_time) FROM ticker_sentiment_snapshots
            WHERE time_range = :time_range AND snapshot_time < :now
        )
    """, {"time_range": time_range, "now": now}).fetchall()
    
    prev_ranks = {row.ticker: row.rank for row in prev_snapshot}
    
    ranked = []
    for i, r in enumerate(results):
        total = r.bullish_count + r.neutral_count + r.bearish_count or 1
        
        composite = (
            W_MENTIONS * (r.mention_count / max_mentions) +
            W_UPVOTES * (r.total_upvotes / max_upvotes) +
            W_COMMENTS * (r.total_comments / max_comments) +
            W_SENTIMENT * ((r.avg_sentiment_score + 1) / 2)  # normalize -1..1 to 0..1
        )
        
        current_rank = i + 1
        prev_rank = prev_ranks.get(r.ticker)
        rank_change = (prev_rank - current_rank) if prev_rank else None  # positive = moved up
        
        # Determine label
        bullish_pct = r.bullish_count / total
        bearish_pct = r.bearish_count / total
        if bullish_pct > 0.5:
            label = "bullish"
        elif bearish_pct > 0.5:
            label = "bearish"
        else:
            label = "neutral"
        
        snapshot = TickerSentimentSnapshot(
            ticker=r.ticker,
            snapshot_time=now,
            time_range=time_range,
            mention_count=r.mention_count,
            total_upvotes=r.total_upvotes,
            total_comments=r.total_comments,
            unique_posts=r.unique_posts,
            bullish_count=r.bullish_count,
            neutral_count=r.neutral_count,
            bearish_count=r.bearish_count,
            bullish_pct=round(bullish_pct, 4),
            neutral_pct=round(r.neutral_count / total, 4),
            bearish_pct=round(bearish_pct, 4),
            sentiment_score=round(r.avg_sentiment_score, 4),
            sentiment_label=label,
            composite_score=round(composite, 4),
            subreddit_distribution=r.subreddit_distribution,
            rank=current_rank,
            previous_rank=prev_rank,
            rank_change=rank_change,
        )
        session.add(snapshot)
    
    # Also write to time-series hypertable for historical charts
    for r in results:
        total = r.bullish_count + r.neutral_count + r.bearish_count or 1
        session.execute("""
            INSERT INTO ticker_sentiment_history (time, ticker, sentiment_score, bullish_pct, mention_count, composite_score)
            VALUES (:time, :ticker, :score, :bullish_pct, :mentions, :composite)
            ON CONFLICT DO NOTHING
        """, {
            "time": now,
            "ticker": r.ticker,
            "score": round(r.avg_sentiment_score, 4),
            "bullish_pct": round(r.bullish_count / total, 4),
            "mentions": r.mention_count,
            "composite": round(composite, 4),
        })
    
    session.commit()


def main():
    for time_range, delta in TIME_RANGES.items():
        compute_rankings_for_range(time_range, delta)
    
    # Invalidate Redis cache
    redis_client.delete("trending:1d", "trending:7d", "trending:14d", "trending:30d")
    redis_client.set("pipeline:last_computed", datetime.now(timezone.utc).isoformat())

```

### 5.4 Bot & Spam Filtering

Applied during ingestion (Section 4.2) and/or during ranking computation:

```python
def is_likely_bot(post: RawPost) -> bool:
    """Flag posts from suspicious accounts for downweighting."""
    if post.author_account_age_days is not None and post.author_account_age_days < 7:
        return True
    if post.author_karma is not None and post.author_karma < 10:
        return True
    if post.author == "[deleted]":
        return True
    return False

def detect_anomalous_spike(ticker: str, current_mentions: int, historical_avg: float) -> bool:
    """Flag tickers with >5x historical average mentions (pump-and-dump signal)."""
    if historical_avg == 0:
        return current_mentions > 50  # new ticker threshold
    return current_mentions > (historical_avg * 5)
```

### 5.5 Extension: Multi-Source Composite Score (Phase 4)

When multiple platforms are active, the composite score formula extends:

```python
PLATFORM_WEIGHTS = {
    "reddit": 1.0,
    "stocktwits": 1.2,
    "twitter": 0.8,
    "news": 1.5,
}

def compute_composite_multi_source(ticker: str, platform_sentiments: dict) -> float:
    """
    platform_sentiments: {"reddit": {"score": 0.6, "count": 150}, "stocktwits": {"score": 0.8, "count": 80}, ...}
    """
    weighted_sum = 0
    weight_total = 0
    
    for platform, data in platform_sentiments.items():
        w = PLATFORM_WEIGHTS.get(platform, 1.0)
        weighted_sum += data["score"] * data["count"] * w
        weight_total += data["count"] * w
    
    return weighted_sum / weight_total if weight_total > 0 else 0
```

---

## 6. Backend API Design

### 6.1 Trending Endpoint

```
GET /api/v1/social/trending
```

**Query Parameters:**

| Parameter | Type | Default | Description |
|---|---|---|---|
| `time_range` | string | `1d` | `1d`, `7d`, `14d`, `30d` |
| `sector` | string | `all` | GICS sector filter |
| `sentiment` | string | `all` | `all`, `bullish`, `bearish`, `neutral` |
| `subreddit` | string | `all` | Subreddit source filter |
| `sort_by` | string | `composite_score` | `composite_score`, `mention_count`, `total_upvotes`, `sentiment_score`, `rank_change`, `price_change_pct` |
| `sort_order` | string | `desc` | `asc`, `desc` |
| `page` | integer | `1` | Page number |
| `per_page` | integer | `50` | `50` or `100` |

**Response:**

```json
{
  "data": [
    {
      "rank": 1,
      "ticker": "NVDA",
      "company_name": "NVIDIA Corporation",
      "sector": "Technology",
      "price": 134.52,
      "price_change_pct": 3.21,
      "mention_count": 487,
      "total_upvotes": 12340,
      "composite_score": 0.8743,
      "sentiment": {
        "label": "bullish",
        "score": 0.72,
        "bullish_pct": 0.68,
        "neutral_pct": 0.22,
        "bearish_pct": 0.10
      },
      "rank_change": 3,
      "previous_rank": 4,
      "sparkline_7d": [120, 145, 230, 310, 420, 487, 450],
      "subreddit_distribution": {
        "wallstreetbets": 245,
        "stocks": 120,
        "options": 82,
        "investing": 40
      },
      "velocity": {
        "mention_velocity_1h": 2.3,
        "sentiment_velocity_24h": 0.15
      }
    }
  ],
  "meta": {
    "total_results": 127,
    "page": 1,
    "per_page": 50,
    "time_range": "1d",
    "filters_applied": {"sector": "all", "sentiment": "all"},
    "last_updated": "2026-02-22T14:30:00Z",
    "pipeline_status": "healthy"
  }
}
```

**Implementation notes:**
- Read from `ticker_sentiment_snapshots` (latest snapshot for the requested time_range).
- Price and company_name come from the existing market data service (joined server-side or via parallel frontend request).
- Sparkline data comes from the last 7 entries in `ticker_sentiment_history`.
- Response is cached in Redis with a 5-minute TTL (invalidated when new rankings are computed).

### 6.2 Sentiment Detail Endpoint

```
GET /api/v1/social/sentiment/{ticker}
```

**Query Parameters:**

| Parameter | Type | Default | Description |
|---|---|---|---|
| `history_range` | string | `7d` | `7d`, `30d`, `90d` |
| `compare_ticker` | string | null | Optional second ticker for comparison mode (premium) |

**Response:**

```json
{
  "ticker": "NVDA",
  "current": {
    "sentiment_label": "bullish",
    "sentiment_score": 0.72,
    "bullish_pct": 0.68,
    "neutral_pct": 0.22,
    "bearish_pct": 0.10,
    "mention_count_24h": 487,
    "mention_count_7d": 2340,
    "sentiment_velocity_24h": 0.15,
    "subreddit_distribution": { "wallstreetbets": 245, "stocks": 120, "options": 82, "investing": 40 }
  },
  "history": [
    { "time": "2026-02-22T14:00:00Z", "sentiment_score": 0.72, "bullish_pct": 0.68, "mention_count": 45 },
    { "time": "2026-02-22T13:00:00Z", "sentiment_score": 0.65, "bullish_pct": 0.62, "mention_count": 38 }
  ],
  "comparison": null,
  "posts": {
    "total": 487,
    "preview": [
      {
        "id": "t3_abc123",
        "title": "NVDA earnings blowout — this is just the beginning",
        "body_preview": "Q4 results exceeded every estimate. Data center revenue up 40% YoY...",
        "subreddit": "wallstreetbets",
        "sentiment_label": "bullish",
        "score": 1523,
        "num_comments": 342,
        "url": "https://reddit.com/r/wallstreetbets/comments/abc123",
        "created_at": "2026-02-22T10:15:00Z"
      }
    ]
  }
}
```

### 6.3 Posts Feed Endpoint

```
GET /api/v1/social/sentiment/{ticker}/posts
```

**Query Parameters:**

| Parameter | Type | Default | Description |
|---|---|---|---|
| `sentiment` | string | `all` | `all`, `bullish`, `bearish`, `neutral` |
| `sort_by` | string | `score` | `score`, `created_at`, `num_comments` |
| `page` | integer | `1` | |
| `per_page` | integer | `20` | Max 50 |

### 6.4 Ticker Widget Endpoint

```
GET /api/v1/social/widget/{ticker}
```

Lightweight endpoint for the ticker page widget. Returns current sentiment + top 3 posts.

**Response:**

```json
{
  "ticker": "NVDA",
  "sentiment_label": "bullish",
  "sentiment_score": 0.72,
  "mention_count_24h": 487,
  "sentiment_velocity_24h": 0.15,
  "trend_direction": "up",
  "top_posts": [
    { "title": "NVDA earnings blowout", "subreddit": "wallstreetbets", "score": 1523, "sentiment_label": "bullish" },
    { "title": "NVDA price target raised to $180", "subreddit": "stocks", "score": 456, "sentiment_label": "bullish" },
    { "title": "Selling covered calls on NVDA", "subreddit": "options", "score": 234, "sentiment_label": "neutral" }
  ]
}
```

### 6.5 Alerts CRUD Endpoints

```
POST   /api/v1/social/alerts          — Create alert
GET    /api/v1/social/alerts          — List user's alerts
PUT    /api/v1/social/alerts/{id}     — Update alert
DELETE /api/v1/social/alerts/{id}     — Delete alert
GET    /api/v1/social/notifications   — List user's notifications (paginated)
PUT    /api/v1/social/notifications/{id}/read — Mark notification as read
```

**Create Alert Request:**

```json
{
  "ticker": "NVDA",
  "alert_type": "mention_spike",
  "config": {
    "threshold": 200,
    "window": "24h"
  },
  "delivery_channels": ["in_app", "email"]
}
```

### 6.6 Pipeline Health Endpoint

```
GET /api/v1/social/health
```

Returns pipeline freshness data for the frontend indicator. No auth required.

---

## 7. Frontend Components

### 7.1 Component Tree

```
<SocialTrendsPage>                         // /reddit
├── <FilterBar>
│   ├── <SectorFilter>
│   ├── <SentimentFilter>
│   ├── <TimeRangeFilter>
│   └── <SubredditFilter>
├── <ViewToggle>                            // Table / Heatmap
├── <DataFreshnessIndicator>
├── <TrendingTable>                         // Table view
│   ├── <TrendingTableHeader>               // sortable columns
│   └── <TrendingTableRow>                  // one per ticker
│       ├── <SentimentBadge>
│       ├── <RankChangePill>
│       ├── <Sparkline>
│       ├── <WatchlistToggle>
│       └── <RowExpandPanel>                // expandable: top 3 posts
│           └── <PostPreviewCard>
├── <SentimentHeatmap>                      // Heatmap view
│   └── <HeatmapTile>
└── <Pagination>

<SentimentDetailPage>                       // /sentiment?ticker=XXX
├── <SentimentHeader>
│   ├── <SentimentGauge>
│   ├── <SentimentBreakdownBar>
│   └── <VelocityIndicator>
├── <PriceChartWithSentimentOverlay>        // dual y-axis chart
│   ├── <TimeRangeSelector>
│   └── <OverlayToggle>
├── <ComparisonSelector>                    // premium: add second ticker
├── <SubredditDistributionChart>
├── <PostFeed>
│   ├── <PostFeedFilters>
│   ├── <PostCard>
│   │   └── <ThreadExpansion>
│   └── <PostFeedPagination>
└── <HistoricalAccuracyCallout>             // Phase 4

<TickerSentimentWidget>                     // embedded in /ticker/XXX
├── <SentimentBadge>
├── <TrendArrow>
├── <MentionCount>
├── <TopPostsList>
└── <FullAnalysisCTA>
```

### 7.2 Key Component: TrendingTableRow

```tsx
// components/social/TrendingTableRow.tsx

interface TrendingStock {
  rank: number;
  ticker: string;
  company_name: string;
  sector: string;
  price: number;
  price_change_pct: number;
  mention_count: number;
  total_upvotes: number;
  composite_score: number;
  sentiment: {
    label: 'bullish' | 'neutral' | 'bearish';
    score: number;
    bullish_pct: number;
    neutral_pct: number;
    bearish_pct: number;
  };
  rank_change: number | null;
  previous_rank: number | null;
  sparkline_7d: number[];
  subreddit_distribution: Record<string, number>;
}

const TrendingTableRow: React.FC<{ stock: TrendingStock }> = ({ stock }) => {
  const [expanded, setExpanded] = useState(false);

  return (
    <>
      <tr className="hover:bg-gray-50 cursor-pointer" onClick={() => setExpanded(!expanded)}>
        <td>{stock.rank}</td>
        <td>
          <Link href={`/ticker/${stock.ticker}`} className="font-semibold text-blue-600">
            {stock.ticker}
          </Link>
          <span className="text-gray-500 text-sm ml-2">{stock.company_name}</span>
        </td>
        <td>${stock.price.toFixed(2)}</td>
        <td className={stock.price_change_pct >= 0 ? 'text-green-600' : 'text-red-600'}>
          {stock.price_change_pct >= 0 ? '+' : ''}{stock.price_change_pct.toFixed(2)}%
        </td>
        <td>{stock.mention_count.toLocaleString()}</td>
        <td>{stock.total_upvotes.toLocaleString()}</td>
        <td>{stock.composite_score.toFixed(2)}</td>
        <td><SentimentBadge sentiment={stock.sentiment} /></td>
        <td><RankChangePill change={stock.rank_change} /></td>
        <td><Sparkline data={stock.sparkline_7d} width={80} height={24} /></td>
        <td><WatchlistToggle ticker={stock.ticker} /></td>
      </tr>
      {expanded && (
        <tr>
          <td colSpan={11}>
            <RowExpandPanel ticker={stock.ticker} />
          </td>
        </tr>
      )}
    </>
  );
};
```

### 7.3 Key Component: SentimentBadge

```tsx
// components/social/SentimentBadge.tsx

const BADGE_STYLES = {
  bullish: 'bg-green-100 text-green-800 border-green-200',
  neutral: 'bg-gray-100 text-gray-700 border-gray-200',
  bearish: 'bg-red-100 text-red-800 border-red-200',
};

const BADGE_ICONS = {
  bullish: '▲',
  neutral: '●',
  bearish: '▼',
};

interface SentimentBadgeProps {
  sentiment: { label: 'bullish' | 'neutral' | 'bearish'; bullish_pct: number };
}

const SentimentBadge: React.FC<SentimentBadgeProps> = ({ sentiment }) => (
  <span className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium border ${BADGE_STYLES[sentiment.label]}`}>
    <span className="mr-1">{BADGE_ICONS[sentiment.label]}</span>
    {sentiment.label.charAt(0).toUpperCase() + sentiment.label.slice(1)}
    <span className="ml-1 opacity-70">{Math.round(sentiment.bullish_pct * 100)}%</span>
  </span>
);
```

### 7.4 Key Component: RankChangePill

Addresses BUG-001 (floating-point display).

```tsx
// components/social/RankChangePill.tsx

interface RankChangePillProps {
  change: number | null;
}

const RankChangePill: React.FC<RankChangePillProps> = ({ change }) => {
  if (change === null || change === undefined) {
    return <span className="text-gray-400 text-sm">NEW</span>;
  }

  // CRITICAL: Math.round to eliminate floating-point display (BUG-001 fix)
  const rounded = Math.round(change);

  if (rounded === 0) {
    return <span className="text-gray-400 text-sm">—</span>;
  }

  const isUp = rounded > 0;
  return (
    <span className={`text-sm font-medium ${isUp ? 'text-green-600' : 'text-red-600'}`}>
      {isUp ? '↑' : '↓'}{Math.abs(rounded)}
    </span>
  );
};
```

### 7.5 Data Fetching Pattern

All social data endpoints use React Query (TanStack Query) with stale-while-revalidate caching:

```tsx
// hooks/useTrendingData.ts
import { useQuery } from '@tanstack/react-query';

interface TrendingFilters {
  timeRange: string;
  sector: string;
  sentiment: string;
  subreddit: string;
  sortBy: string;
  sortOrder: string;
  page: number;
  perPage: number;
}

export function useTrendingData(filters: TrendingFilters) {
  return useQuery({
    queryKey: ['social-trending', filters],
    queryFn: () => fetchTrending(filters),
    staleTime: 5 * 60 * 1000,        // 5 min stale time (matches Redis TTL)
    refetchInterval: 15 * 60 * 1000,  // auto-refetch every 15 min
    refetchIntervalInBackground: false,
    keepPreviousData: true,            // prevent flash on filter change
  });
}

// Manual refresh handler (for refresh button)
export function useRefreshTrending() {
  const queryClient = useQueryClient();
  return () => queryClient.invalidateQueries({ queryKey: ['social-trending'] });
}
```

### 7.6 DataFreshnessIndicator

```tsx
// components/social/DataFreshnessIndicator.tsx
import { useQuery } from '@tanstack/react-query';
import { formatDistanceToNow } from 'date-fns';

const DataFreshnessIndicator: React.FC<{ onRefresh: () => void }> = ({ onRefresh }) => {
  const { data } = useQuery({
    queryKey: ['pipeline-health'],
    queryFn: () => fetch('/api/v1/social/health').then(r => r.json()),
    refetchInterval: 60_000, // check every minute
  });

  if (!data) return null;

  const lastUpdated = new Date(data.last_snapshot_at);
  const minutesAgo = (Date.now() - lastUpdated.getTime()) / 60_000;
  const isStale = minutesAgo > 30;

  return (
    <div className={`flex items-center gap-2 text-sm ${isStale ? 'text-amber-600' : 'text-gray-500'}`}>
      <span className={`w-2 h-2 rounded-full ${isStale ? 'bg-amber-400' : 'bg-green-400'}`} />
      Updated {formatDistanceToNow(lastUpdated, { addSuffix: true })}
      {isStale && <span className="text-amber-600 font-medium">⚠ Data may be stale</span>}
      <button onClick={onRefresh} className="ml-2 text-blue-600 hover:underline text-xs">
        Refresh
      </button>
    </div>
  );
};
```

---

## 8. Alerts System

### 8.1 Alert Evaluation Engine

```python
# scripts/pipeline/evaluate_alerts.py
"""
Evaluates all active alerts against latest sentiment data.
Runs every 15 minutes after ranking computation.
"""
from datetime import datetime, timezone, timedelta

def evaluate_alerts():
    session = get_db_session()
    now = datetime.now(timezone.utc)
    
    active_alerts = session.query(UserSentimentAlert).filter_by(is_active=True).all()
    
    for alert in active_alerts:
        # Respect cooldown
        if alert.last_triggered_at:
            cooldown_until = alert.last_triggered_at + timedelta(minutes=alert.cooldown_minutes)
            if now < cooldown_until:
                continue
        
        triggered = False
        message = ""
        
        if alert.alert_type == "mention_spike":
            triggered, message = check_mention_spike(session, alert)
        elif alert.alert_type == "sentiment_shift":
            triggered, message = check_sentiment_shift(session, alert)
        elif alert.alert_type == "trending_entry":
            triggered, message = check_trending_entry(session, alert)
        elif alert.alert_type == "sentiment_velocity":
            triggered, message = check_sentiment_velocity(session, alert)
        
        if triggered:
            dispatch_notification(session, alert, message)
            alert.last_triggered_at = now
    
    session.commit()


def check_mention_spike(session, alert) -> tuple[bool, str]:
    config = alert.config
    threshold = config["threshold"]
    window = config["window"]  # "24h" or "7d"
    
    delta = timedelta(hours=24) if window == "24h" else timedelta(days=7)
    cutoff = datetime.now(timezone.utc) - delta
    
    count = session.execute("""
        SELECT COUNT(DISTINCT rp.id) FROM post_sentiment ps
        JOIN raw_posts rp ON rp.id = ps.raw_post_id
        WHERE ps.ticker = :ticker AND rp.post_created_at >= :cutoff
    """, {"ticker": alert.ticker, "cutoff": cutoff}).scalar()
    
    if count >= threshold:
        return True, f"{alert.ticker} has {count} mentions in the last {window} (threshold: {threshold})"
    return False, ""


def check_sentiment_shift(session, alert) -> tuple[bool, str]:
    config = alert.config
    target_direction = config["direction"]  # "bullish" or "bearish"
    
    latest = session.execute("""
        SELECT sentiment_label FROM ticker_sentiment_snapshots
        WHERE ticker = :ticker AND time_range = '1d'
        ORDER BY snapshot_time DESC LIMIT 1
    """, {"ticker": alert.ticker}).scalar()
    
    if latest == target_direction:
        return True, f"{alert.ticker} sentiment has shifted to {target_direction.upper()}"
    return False, ""


def check_trending_entry(session, alert) -> tuple[bool, str]:
    config = alert.config
    top_n = config["top_n"]
    
    latest_rank = session.execute("""
        SELECT rank FROM ticker_sentiment_snapshots
        WHERE ticker = :ticker AND time_range = '1d'
        ORDER BY snapshot_time DESC LIMIT 1
    """, {"ticker": alert.ticker}).scalar()
    
    if latest_rank is not None and latest_rank <= top_n:
        return True, f"{alert.ticker} has entered the Top {top_n} trending (currently #{latest_rank})"
    return False, ""


def check_sentiment_velocity(session, alert) -> tuple[bool, str]:
    config = alert.config
    delta_pct = config["delta_pct"]  # e.g., -20 for 20% drop
    
    # Compare latest vs 24h ago
    scores = session.execute("""
        SELECT sentiment_score, snapshot_time FROM ticker_sentiment_snapshots
        WHERE ticker = :ticker AND time_range = '1d'
        ORDER BY snapshot_time DESC LIMIT 2
    """, {"ticker": alert.ticker}).fetchall()
    
    if len(scores) < 2:
        return False, ""
    
    current = scores[0].sentiment_score
    previous = scores[1].sentiment_score
    
    if previous == 0:
        return False, ""
    
    change_pct = ((current - previous) / abs(previous)) * 100
    
    if delta_pct < 0 and change_pct <= delta_pct:  # negative threshold = dropping
        return True, f"{alert.ticker} sentiment dropped {abs(change_pct):.0f}% in the last period"
    elif delta_pct > 0 and change_pct >= delta_pct:
        return True, f"{alert.ticker} sentiment rose {change_pct:.0f}% in the last period"
    return False, ""


def dispatch_notification(session, alert, message: str):
    """Create notification record and dispatch via channels."""
    for channel in alert.delivery_channels:
        notification = AlertNotification(
            alert_id=alert.id,
            user_id=alert.user_id,
            ticker=alert.ticker,
            message=message,
            channel=channel,
        )
        session.add(notification)
        
        if channel == "email":
            send_alert_email(alert.user_id, alert.ticker, message)
        # in_app notifications are read from the DB by the frontend
```

### 8.2 Free-Tier Limit Enforcement

```python
# In the POST /api/v1/social/alerts handler:

MAX_FREE_ALERTS = 3

def create_alert(user_id: int, request: CreateAlertRequest):
    user = get_user(user_id)
    
    if not user.is_premium:
        active_count = session.query(UserSentimentAlert).filter_by(
            user_id=user_id, is_active=True
        ).count()
        
        if active_count >= MAX_FREE_ALERTS:
            raise HTTPException(
                status_code=403,
                detail=f"Free tier limited to {MAX_FREE_ALERTS} alerts. Upgrade to Premium for unlimited alerts."
            )
    
    # ... create alert
```

---

## 9. Caching & Performance

### 9.1 Redis Cache Strategy

| Cache Key Pattern | TTL | Invalidation Trigger |
|---|---|---|
| `trending:{time_range}:{filters_hash}` | 5 min | New ranking computation (cron) |
| `sentiment:{ticker}:{history_range}` | 5 min | New snapshot computation |
| `widget:{ticker}` | 5 min | New snapshot computation |
| `posts:{ticker}:{filters_hash}:{page}` | 10 min | New posts ingested |
| `pipeline:last_computed` | None | Overwritten each computation cycle |
| `pipeline:health` | 1 min | Health check endpoint |

### 9.2 Cache Implementation

```python
import hashlib
import json

def cache_key(prefix: str, params: dict) -> str:
    """Generate deterministic cache key from parameters."""
    param_str = json.dumps(params, sort_keys=True)
    hash_suffix = hashlib.md5(param_str.encode()).hexdigest()[:8]
    return f"{prefix}:{hash_suffix}"


def cached_response(key: str, ttl: int, compute_fn):
    """Generic cache-aside pattern."""
    cached = redis_client.get(key)
    if cached:
        return json.loads(cached)
    
    result = compute_fn()
    redis_client.setex(key, ttl, json.dumps(result, default=str))
    return result
```

### 9.3 Database Query Optimization

- All frequently-read tables have indexes on query patterns (defined in Section 3.2).
- The `ticker_sentiment_snapshots` table uses a composite unique index on `(ticker, snapshot_time, time_range)` for efficient latest-snapshot queries.
- TimescaleDB hypertable for `ticker_sentiment_history` provides automatic time-based partitioning and query optimization for range scans.
- `EXPLAIN ANALYZE` should be run on all API-facing queries during development to ensure index usage.

### 9.4 Performance Targets

| Endpoint | P50 Latency | P99 Latency | Cache Hit Rate Target |
|---|---|---|---|
| `GET /social/trending` | <100ms | <300ms | >80% |
| `GET /social/sentiment/{ticker}` | <150ms | <400ms | >70% |
| `GET /social/widget/{ticker}` | <50ms | <150ms | >90% |
| `GET /social/health` | <20ms | <50ms | >95% |

---

## 10. Feature Gating & Tiering

### 10.1 Middleware Pattern

```python
from functools import wraps

PREMIUM_FEATURES = {
    "sentiment_history_30d",
    "sentiment_history_90d",
    "price_chart_overlay",
    "comparison_mode",
    "full_post_feed",
    "unlimited_alerts",
    "historical_accuracy",
    "api_access",
}

def require_premium(feature: str):
    """Decorator to gate premium-only features."""
    def decorator(fn):
        @wraps(fn)
        async def wrapper(*args, **kwargs):
            user = get_current_user()
            if feature in PREMIUM_FEATURES and not user.is_premium:
                raise HTTPException(
                    status_code=403,
                    detail={
                        "error": "premium_required",
                        "feature": feature,
                        "message": f"Upgrade to Premium to access {feature.replace('_', ' ')}.",
                        "upgrade_url": "/pricing"
                    }
                )
            return await fn(*args, **kwargs)
        return wrapper
    return decorator


# Usage in endpoint:
@app.get("/api/v1/social/sentiment/{ticker}")
@require_premium("sentiment_history_90d")  # only gated if history_range=90d
async def get_sentiment_detail(ticker: str, history_range: str = "7d"):
    if history_range in ("30d", "90d"):
        require_premium("sentiment_history_" + history_range)
    # ...
```

### 10.2 Frontend Paywall Component

```tsx
// components/PremiumGate.tsx

const PremiumGate: React.FC<{ feature: string; children: React.ReactNode }> = ({ feature, children }) => {
  const { user } = useAuth();

  if (user?.is_premium) {
    return <>{children}</>;
  }

  return (
    <div className="relative">
      <div className="filter blur-sm pointer-events-none opacity-50">{children}</div>
      <div className="absolute inset-0 flex items-center justify-center">
        <div className="bg-white border rounded-lg shadow-lg p-6 text-center max-w-sm">
          <h3 className="font-semibold text-lg mb-2">Premium Feature</h3>
          <p className="text-gray-600 text-sm mb-4">
            Upgrade to unlock {feature.replace(/_/g, ' ')}.
          </p>
          <Link href="/pricing" className="bg-blue-600 text-white px-4 py-2 rounded hover:bg-blue-700">
            Upgrade Now
          </Link>
        </div>
      </div>
    </div>
  );
};
```

---

## 11. Bug Fixes (Phase 1)

### 11.1 BUG-001: Floating-Point Rank Change

**Root cause:** `rank_change` is computed as a division or difference of floats and rendered without rounding.

**Fix (backend):** Ensure `rank_change` is stored as `INTEGER` in the database (see Section 3.2). In `compute_rankings.py`, the value is already `previous_rank - current_rank`, both integers.

**Fix (frontend):** `RankChangePill` component (Section 7.4) applies `Math.round()` as a safety net.

**Regression test:**

```python
def test_rank_change_is_integer():
    """BUG-001: rank_change must always be an integer, never a float."""
    response = client.get("/api/v1/social/trending?time_range=1d")
    data = response.json()["data"]
    for item in data:
        if item["rank_change"] is not None:
            assert isinstance(item["rank_change"], int), \
                f"rank_change for {item['ticker']} is {type(item['rank_change'])}, expected int"
            assert "." not in str(item["rank_change"]), \
                f"rank_change for {item['ticker']} contains decimal: {item['rank_change']}"
```

### 11.2 BUG-002: Inconsistent Score Icon

**Root cause:** Conditional rendering: `if (rank <= 3) show chartIcon else show chatBubble`.

**Fix:** Remove the conditional. The `SentimentBadge` component (Section 7.3) renders identically for all rows — a colored pill with directional indicator and percentage, regardless of rank.

**Regression test:**

```tsx
// __tests__/SentimentBadge.test.tsx
describe('SentimentBadge', () => {
  it('renders identically for rank 1 and rank 50', () => {
    const sentiment = { label: 'bullish' as const, score: 0.72, bullish_pct: 0.68, neutral_pct: 0.22, bearish_pct: 0.10 };
    const { container: rank1 } = render(<SentimentBadge sentiment={sentiment} />);
    const { container: rank50 } = render(<SentimentBadge sentiment={sentiment} />);
    expect(rank1.innerHTML).toBe(rank50.innerHTML);
  });
  
  it('never renders a chart icon or chat bubble icon', () => {
    const { container } = render(<SentimentBadge sentiment={{ label: 'bullish', score: 0.5, bullish_pct: 0.6, neutral_pct: 0.3, bearish_pct: 0.1 }} />);
    expect(container.querySelector('svg')).toBeNull(); // no icon SVGs
    expect(container.textContent).toContain('Bullish');
  });
});
```

### 11.3 BUG-003: Missing Price and % Change

**Fix:** The trending API response (Section 6.1) includes `price` and `price_change_pct` fields. These are joined from the existing market data service during API response construction:

```python
# In the trending endpoint handler:
def enrich_with_market_data(trending_items: list[dict]) -> list[dict]:
    """Join market data onto trending results."""
    tickers = [item["ticker"] for item in trending_items]
    market_data = market_data_service.get_quotes(tickers)  # existing service
    
    for item in trending_items:
        quote = market_data.get(item["ticker"], {})
        item["price"] = quote.get("price", None)
        item["price_change_pct"] = quote.get("change_pct", None)
        item["company_name"] = quote.get("company_name", item["ticker"])
        item["sector"] = quote.get("sector", "Unknown")
    
    return trending_items
```

**Regression test:**

```python
def test_trending_includes_price_data():
    """BUG-003: Every trending row must include price and price_change_pct."""
    response = client.get("/api/v1/social/trending?time_range=1d")
    data = response.json()["data"]
    assert len(data) > 0
    for item in data:
        assert "price" in item, f"Missing price for {item['ticker']}"
        assert "price_change_pct" in item, f"Missing price_change_pct for {item['ticker']}"
        assert item["price"] is not None, f"Null price for {item['ticker']}"
```

### 11.4 BUG-004: Data Staleness

**Fix:** Combination of pipeline scheduling (Section 4.3), health endpoint (Section 4.5), and frontend `DataFreshnessIndicator` component (Section 7.6).

**Regression test:**

```python
def test_data_freshness_within_sla():
    """BUG-004: Data must be <30 min old during market hours."""
    response = client.get("/api/v1/social/health")
    health = response.json()
    
    last_snapshot = datetime.fromisoformat(health["last_snapshot_at"])
    age_minutes = (datetime.now(timezone.utc) - last_snapshot).total_seconds() / 60
    
    # During market hours (9:30-16:00 ET), data must be <30 min old
    assert health["status"] == "healthy"
    assert age_minutes < 30, f"Data is {age_minutes:.0f} minutes old, exceeds 30-min SLA"
```

---

## 12. Testing Strategy

### 12.1 Test Pyramid

```
            ┌─────────┐
            │  E2E    │  2-3 critical user flows
            │  Tests  │  (Playwright)
           ─┼─────────┼─
          ┌─┤ Integra-│  API endpoint tests, DB tests
          │ │  tion   │  pipeline end-to-end
         ─┼─┼─────────┼─
        ┌──┤│  Unit   │  Components, utils, pipeline funcs
        │  ││  Tests  │  ticker extraction, sentiment, scoring
        └──┘└─────────┘
```

### 12.2 Unit Tests

| Module | Test File | Coverage Target |
|---|---|---|
| Ticker extraction | `tests/unit/test_ticker_extractor.py` | >95% |
| Rank change formatting | `tests/unit/test_rank_change.tsx` | 100% |
| Sentiment badge rendering | `tests/unit/test_sentiment_badge.tsx` | 100% |
| Composite score calculation | `tests/unit/test_composite_score.py` | >90% |
| Alert rule evaluation | `tests/unit/test_alert_evaluation.py` | >90% |
| Cache key generation | `tests/unit/test_cache.py` | 100% |
| Bot/spam filtering | `tests/unit/test_bot_filter.py` | >90% |
| Feature gating middleware | `tests/unit/test_premium_gate.py` | 100% |

**Example unit tests:**

```python
# tests/unit/test_ticker_extractor.py

class TestTickerExtractor:
    def test_dollar_sign_tickers(self):
        assert extract_tickers("$AAPL is mooning") == ["AAPL"]

    def test_multiple_tickers(self):
        result = extract_tickers("$NVDA and $AMD are competing")
        assert set(result) == {"NVDA", "AMD"}

    def test_filters_false_positives(self):
        result = extract_tickers("I AM going to DD on IT")
        assert "I" not in result
        assert "AM" not in result
        assert "DD" not in result
        assert "IT" not in result

    def test_no_tickers_found(self):
        assert extract_tickers("This is just a random post about nothing") == []

    def test_case_sensitive(self):
        result = extract_tickers("aapl is not recognized, AAPL is")
        assert result == ["AAPL"]

    def test_single_char_requires_dollar_sign(self):
        # Single-char tickers (like $F for Ford) need $ prefix
        assert extract_tickers("F is great") == []
        assert extract_tickers("$F is great") == ["F"]

    def test_body_text_extraction(self):
        result = extract_tickers("Check this out", body="Looking at $TSLA and MSFT potential")
        assert set(result) == {"TSLA", "MSFT"}
```

```python
# tests/unit/test_composite_score.py

class TestCompositeScore:
    def test_score_range(self):
        """Composite score must be between 0 and 1."""
        score = compute_composite(mentions=100, upvotes=500, comments=50, sentiment=0.8,
                                   max_mentions=100, max_upvotes=500, max_comments=50)
        assert 0 <= score <= 1

    def test_higher_mentions_higher_score(self):
        score_low = compute_composite(mentions=10, upvotes=100, comments=10, sentiment=0.5,
                                       max_mentions=100, max_upvotes=100, max_comments=100)
        score_high = compute_composite(mentions=90, upvotes=100, comments=10, sentiment=0.5,
                                        max_mentions=100, max_upvotes=100, max_comments=100)
        assert score_high > score_low

    def test_sentiment_contributes(self):
        score_bearish = compute_composite(mentions=50, upvotes=100, comments=50, sentiment=-0.8,
                                           max_mentions=100, max_upvotes=100, max_comments=100)
        score_bullish = compute_composite(mentions=50, upvotes=100, comments=50, sentiment=0.8,
                                           max_mentions=100, max_upvotes=100, max_comments=100)
        assert score_bullish > score_bearish
```

### 12.3 Integration Tests

```python
# tests/integration/test_trending_api.py

class TestTrendingAPI:
    def test_trending_returns_paginated_results(self, client, seeded_db):
        response = client.get("/api/v1/social/trending?time_range=1d&page=1&per_page=50")
        assert response.status_code == 200
        data = response.json()
        assert len(data["data"]) <= 50
        assert data["meta"]["page"] == 1

    def test_trending_filter_by_sentiment(self, client, seeded_db):
        response = client.get("/api/v1/social/trending?sentiment=bullish")
        data = response.json()["data"]
        for item in data:
            assert item["sentiment"]["label"] == "bullish"

    def test_trending_sort_by_mentions(self, client, seeded_db):
        response = client.get("/api/v1/social/trending?sort_by=mention_count&sort_order=desc")
        data = response.json()["data"]
        mentions = [item["mention_count"] for item in data]
        assert mentions == sorted(mentions, reverse=True)

    def test_trending_includes_required_fields(self, client, seeded_db):
        response = client.get("/api/v1/social/trending")
        item = response.json()["data"][0]
        required_fields = [
            "rank", "ticker", "company_name", "price", "price_change_pct",
            "mention_count", "total_upvotes", "composite_score", "sentiment",
            "rank_change", "sparkline_7d"
        ]
        for field in required_fields:
            assert field in item, f"Missing required field: {field}"

    def test_meta_includes_freshness(self, client, seeded_db):
        response = client.get("/api/v1/social/trending")
        meta = response.json()["meta"]
        assert "last_updated" in meta
        assert "pipeline_status" in meta
```

```python
# tests/integration/test_pipeline_end_to_end.py

class TestPipelineE2E:
    def test_scrape_to_sentiment_flow(self, db_session):
        """Verify full pipeline: ingest → extract tickers → classify sentiment → compute rankings."""
        # Insert mock raw post
        post = RawPost(
            external_id="t3_test123",
            subreddit_id=1,
            platform="reddit",
            title="$NVDA just smashed earnings expectations!",
            body="Revenue up 40% YoY, data center is on fire",
            author="testuser",
            score=500,
            upvotes=500,
            num_comments=100,
            post_created_at=datetime.now(timezone.utc),
        )
        db_session.add(post)
        db_session.commit()

        # Run processing
        process_batch()

        # Verify post_sentiment was created
        ps = db_session.query(PostSentiment).filter_by(raw_post_id=post.id).first()
        assert ps is not None
        assert ps.ticker == "NVDA"
        assert ps.sentiment_label in ("bullish", "neutral", "bearish")
        assert -1 <= ps.sentiment_score <= 1

        # Run ranking computation
        compute_rankings_for_range("1d", timedelta(days=1))

        # Verify snapshot was created
        snapshot = db_session.query(TickerSentimentSnapshot).filter_by(
            ticker="NVDA", time_range="1d"
        ).order_by(TickerSentimentSnapshot.snapshot_time.desc()).first()
        assert snapshot is not None
        assert snapshot.mention_count >= 1
        assert isinstance(snapshot.rank_change, (int, type(None)))
```

### 12.4 E2E Tests (Playwright)

```typescript
// tests/e2e/social-trends.spec.ts
import { test, expect } from '@playwright/test';

test.describe('Social Trends Page', () => {
  test('loads trending table with required columns', async ({ page }) => {
    await page.goto('/reddit');
    
    // Table loads
    const table = page.locator('[data-testid="trending-table"]');
    await expect(table).toBeVisible();
    
    // Required column headers present
    for (const header of ['Rank', 'Ticker', 'Price', '% Change', 'Mentions', 'Sentiment', 'Rank Δ']) {
      await expect(page.locator(`th:has-text("${header}")`)).toBeVisible();
    }
    
    // Freshness indicator present
    await expect(page.locator('[data-testid="freshness-indicator"]')).toBeVisible();
    await expect(page.locator('[data-testid="freshness-indicator"]')).toContainText('Updated');
  });

  test('filter by sentiment shows only matching rows', async ({ page }) => {
    await page.goto('/reddit');
    await page.click('[data-testid="sentiment-filter"]');
    await page.click('text=Bullish');
    
    const badges = page.locator('[data-testid="sentiment-badge"]');
    const count = await badges.count();
    for (let i = 0; i < count; i++) {
      await expect(badges.nth(i)).toContainText('Bullish');
    }
  });

  test('row expand shows post preview', async ({ page }) => {
    await page.goto('/reddit');
    
    // Click expand on first row
    await page.locator('[data-testid="row-expand-btn"]').first().click();
    
    // Expand panel visible with posts
    const panel = page.locator('[data-testid="row-expand-panel"]').first();
    await expect(panel).toBeVisible();
    
    const posts = panel.locator('[data-testid="post-preview-card"]');
    await expect(posts).toHaveCount(3);
  });

  test('rank change never shows decimal values (BUG-001)', async ({ page }) => {
    await page.goto('/reddit');
    
    const rankChanges = page.locator('[data-testid="rank-change"]');
    const count = await rankChanges.count();
    
    for (let i = 0; i < count; i++) {
      const text = await rankChanges.nth(i).textContent();
      expect(text).not.toMatch(/\.\d/); // no decimal points
    }
  });
});
```

---

## 13. Deployment & Rollout

### 13.1 Phase 1 Deployment Checklist

```
Pre-deployment:
□ All migrations applied to staging DB
□ OpenClaw cron jobs configured and tested
□ FinBERT model downloaded and cached
□ Redis instance provisioned
□ TimescaleDB extension enabled
□ All BUG-001 through BUG-004 regression tests passing
□ API endpoints returning valid responses against staging data
□ Frontend build successful, no TypeScript errors

Deployment:
□ Run database migrations (001-008)
□ Deploy backend API with new endpoints
□ Deploy OpenClaw scraper scripts and cron schedule
□ Verify first scrape cycle completes successfully
□ Verify first sentiment processing cycle completes
□ Verify first ranking computation produces valid data
□ Deploy frontend with new Social Trends page
□ Verify DataFreshnessIndicator shows healthy status

Post-deployment:
□ Smoke test all filter/sort combinations
□ Verify no floating-point values in rank_change
□ Verify all rows show price and % change
□ Verify data refreshes within 15 minutes
□ Monitor error logs for 24 hours
□ Check Redis cache hit rates
```

### 13.2 Feature Flag Strategy

Use feature flags for progressive rollout:

```python
FEATURE_FLAGS = {
    "social_trends_v2": True,          # Phase 1: new trending table
    "sentiment_detail_v2": False,       # Phase 2: redesigned detail page
    "price_chart_overlay": False,       # Phase 2: sentiment on price chart
    "sentiment_alerts": False,          # Phase 3: alert system
    "heatmap_view": False,              # Phase 3: heatmap toggle
    "comparison_mode": False,           # Phase 3: two-ticker comparison
    "multi_source_sentiment": False,    # Phase 4: StockTwits + Twitter
    "historical_accuracy": False,       # Phase 4: backtested signal quality
    "sentiment_api": False,             # Phase 4: public API
}
```

### 13.3 Rollback Plan

If critical issues are discovered post-deploy:

1. **Frontend rollback:** Revert to previous build via deployment platform (Vercel/Netlify) or Docker tag.
2. **Backend rollback:** Feature flags disable new endpoints; old `/reddit` endpoint still functional.
3. **Pipeline rollback:** Stop cron jobs (`crontab -r /etc/cron.d/investorcenter-scraper`). Existing data remains intact.
4. **Database rollback:** Down migrations available for all schema changes (no destructive operations on existing tables).

---

## 14. Observability & Monitoring

### 14.1 Metrics to Track

| Metric | Source | Alert Threshold |
|---|---|---|
| Pipeline last_scrape age | Pipeline health check | >30 min → warning, >60 min → critical |
| Pipeline last_snapshot age | Pipeline health check | >30 min → warning, >60 min → critical |
| Unprocessed posts queue depth | `raw_posts WHERE is_processed = false` | >10,000 → warning |
| API endpoint P99 latency | Application metrics | >500ms → warning |
| API 5xx error rate | Application logs | >1% → critical |
| Redis cache hit rate | Redis INFO | <60% → warning |
| FinBERT inference time per batch | Pipeline logs | >120s per 5K batch → warning |
| Alert notification delivery failures | Alert system logs | Any failure → warning |
| Disk usage (raw_posts table) | Database monitoring | >80% allocated → warning |

### 14.2 Logging

All pipeline scripts log structured JSON to files under `/var/log/ic-*.log`:

```python
import logging
import json

logger = logging.getLogger("investorcenter.pipeline")

def log_event(event: str, **kwargs):
    logger.info(json.dumps({"event": event, "timestamp": datetime.now(timezone.utc).isoformat(), **kwargs}))

# Usage:
log_event("scrape_complete", subreddit="wallstreetbets", posts_ingested=47, duration_sec=3.2)
log_event("sentiment_batch_complete", posts_processed=5000, duration_sec=52.1)
log_event("ranking_computed", time_range="1d", tickers_ranked=127)
log_event("alert_triggered", user_id=42, ticker="NVDA", alert_type="mention_spike")
```

---

## 15. Design Decisions & ADRs

### ADR-001: PostgreSQL + TimescaleDB over Dedicated Time-Series DB

**Context:** Sentiment history data is time-series by nature. Options included InfluxDB, QuestDB, or TimescaleDB as a Postgres extension.

**Decision:** Use TimescaleDB extension on the existing PostgreSQL instance.

**Rationale:** Avoids introducing a new database technology. TimescaleDB provides hypertables, automatic partitioning, continuous aggregates, and built-in retention/compression policies while running inside Postgres. Queries use standard SQL. The team already knows Postgres. At our projected volume (<200K rows/day), a dedicated TSDB is unnecessary overhead.

### ADR-002: FinBERT for Sentiment Classification over Commercial API

**Context:** Options included FinBERT (open-source, local), AWS Comprehend, Google NLP, or OpenAI classification.

**Decision:** FinBERT running locally on CPU.

**Rationale:** Zero per-inference cost (critical for 5K–200K posts/day). Financial domain fine-tuning outperforms general-purpose sentiment models on stock-related text. Inference on CPU is fast enough for batch processing (10ms/post). No external API dependency. Trade-off: requires model download (~500MB) and Python ML dependencies.

### ADR-003: Batch Processing over Stream Processing

**Context:** Could process posts in real-time via a streaming framework (Kafka, Redis Streams) or in batch via cron.

**Decision:** Batch cron jobs every 15 minutes.

**Rationale:** The PRD targets 15-minute freshness, not real-time. Batch processing is dramatically simpler to implement, debug, and operate for a small team. Streaming introduces Kafka/broker infrastructure, exactly-once semantics complexity, and higher ops burden. The system can be migrated to streaming later if latency requirements tighten.

### ADR-004: OpenClaw over Direct HTTP Scraping

**Context:** Could build custom scrapers using requests/BeautifulSoup, use the Reddit official API, or leverage OpenClaw.

**Decision:** OpenClaw reddit-scraper skill as the primary data acquisition layer.

**Rationale:** OpenClaw provides battle-tested scraping with built-in retry logic, Firecrawl fallback for bot detection circumvention, 15-minute caching, and structured JSON output. Eliminates Reddit API key dependency and rate-limit concerns. The platform-agnostic architecture (web_fetch, Firecrawl) naturally extends to StockTwits, Twitter/X, and news in later phases without building new scrapers. Lower maintenance burden than custom code.

### ADR-005: Redis Cache-Aside over Application-Level Caching

**Context:** Could cache in application memory (LRU), use Redis, or use Postgres materialized views.

**Decision:** Redis with cache-aside pattern and explicit invalidation.

**Rationale:** Redis provides shared cache across multiple backend instances (future horizontal scaling). Explicit TTLs and invalidation on new ranking computation prevent stale data. Materialized views were considered for rankings but add refresh complexity; Redis is simpler for read-heavy, write-infrequent patterns. Application-memory cache doesn't survive restarts and can't be shared.

### ADR-006: Paginated Table over Infinite Scroll

**Context:** PRD specifies paginated table. This ADR records the technical rationale.

**Decision:** Server-side pagination with 50/100 toggle.

**Rationale:** Server-side pagination limits query result set size, keeps API response payloads small (<50KB), and enables efficient database queries with `LIMIT/OFFSET`. Infinite scroll requires cursor-based pagination, more complex frontend state management, and degrades performance with large datasets. Financial data tables benefit from deterministic positioning (users can bookmark "page 2, sorted by sentiment").

---

*— End of Technical Specification —*
