# Technical Specification: System Architecture
## InvestorCenter.ai

**Version:** 1.0
**Date:** November 12, 2025
**Status:** Draft

---

## Table of Contents

1. [Overview](#overview)
2. [System Architecture](#system-architecture)
3. [Technology Stack](#technology-stack)
4. [Data Infrastructure](#data-infrastructure)
5. [API Architecture](#api-architecture)
6. [Database Schema](#database-schema)
7. [Security & Compliance](#security--compliance)
8. [Performance & Scalability](#performance--scalability)
9. [Deployment & DevOps](#deployment--devops)

---

## Overview

This document defines the technical architecture for InvestorCenter.ai, covering all systems, infrastructure, data flows, and implementation details needed to build a production-ready stock analysis platform.

### Design Principles

1. **Scalability:** Handle 10,000+ concurrent users, 5,000+ stocks
2. **Performance:** Sub-second page loads, real-time data updates
3. **Reliability:** 99.9% uptime, graceful degradation
4. **Maintainability:** Modular architecture, clean separation of concerns
5. **Cost-Efficiency:** Optimize infrastructure costs, use free data sources
6. **Security:** HTTPS, authentication, authorization, data encryption

---

## System Architecture

### High-Level Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                        CLIENT LAYER                         │
├─────────────────────────────────────────────────────────────┤
│  Web App (Next.js)  │  iOS App (Swift)  │  Android (Kotlin)│
│  React, TailwindCSS │  SwiftUI         │  Jetpack Compose│
└──────────────┬──────────────┬──────────────┬───────────────┘
               │              │              │
               └──────────────┴──────────────┘
                              │
                    ┌─────────▼─────────┐
                    │   CDN (CloudFlare)│
                    │   Static Assets   │
                    └─────────┬─────────┘
                              │
┌─────────────────────────────▼──────────────────────────────┐
│                      API GATEWAY LAYER                      │
├─────────────────────────────────────────────────────────────┤
│  Load Balancer (AWS ELB / nginx)                           │
│  API Gateway (Kong / AWS API Gateway)                       │
│  - Rate Limiting                                            │
│  - Authentication (JWT)                                     │
│  - Request Routing                                          │
└─────────────────────────────┬───────────────────────────────┘
                              │
┌─────────────────────────────▼──────────────────────────────┐
│                    APPLICATION LAYER                        │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌────────────┐  ┌────────────┐  ┌────────────┐           │
│  │ Web API    │  │ Data Proc. │  │ Real-Time  │           │
│  │ (FastAPI)  │  │ (Python)   │  │ (WebSocket)│           │
│  │            │  │            │  │            │           │
│  │ Stock Data │  │ IC Score   │  │ Price      │           │
│  │ User Mgmt  │  │ Calc       │  │ Updates    │           │
│  │ Portfolio  │  │ Sentiment  │  │ Alerts     │           │
│  └────────────┘  └────────────┘  └────────────┘           │
│                                                             │
│  ┌────────────┐  ┌────────────┐  ┌────────────┐           │
│  │ ML Service │  │ News       │  │ Chart      │           │
│  │ (Python)   │  │ Aggregator │  │ Service    │           │
│  │            │  │            │  │            │           │
│  │ Pattern    │  │ NLP        │  │ Indicator  │           │
│  │ Detection  │  │ Sentiment  │  │ Calc       │           │
│  └────────────┘  └────────────┘  └────────────┘           │
└─────────────────────────────┬───────────────────────────────┘
                              │
┌─────────────────────────────▼──────────────────────────────┐
│                       CACHE LAYER                           │
├─────────────────────────────────────────────────────────────┤
│  Redis Cluster                                              │
│  - Session Storage                                          │
│  - IC Scores Cache (TTL: 24h)                              │
│  - Price Data Cache (TTL: 1min market hours, 1h off hours) │
│  - API Response Cache                                       │
└─────────────────────────────┬───────────────────────────────┘
                              │
┌─────────────────────────────▼──────────────────────────────┐
│                       DATA LAYER                            │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌──────────────────┐  ┌─────────────────┐                │
│  │ PostgreSQL       │  │ TimescaleDB     │                │
│  │ (Primary DB)     │  │ (Time-Series)   │                │
│  │                  │  │                 │                │
│  │ - Users          │  │ - Price History │                │
│  │ - Portfolios     │  │ - IC Scores     │                │
│  │ - Watchlists     │  │ - Indicators    │                │
│  │ - Companies      │  │ - Metrics       │                │
│  │ - Financials     │  │                 │                │
│  └──────────────────┘  └─────────────────┘                │
│                                                             │
│  ┌──────────────────┐  ┌─────────────────┐                │
│  │ Elasticsearch    │  │ S3 / Object     │                │
│  │ (Search)         │  │ Storage         │                │
│  │                  │  │                 │                │
│  │ - Stock Search   │  │ - Reports (PDF) │                │
│  │ - News Articles  │  │ - Exports       │                │
│  │ - Screener Index │  │ - Backups       │                │
│  └──────────────────┘  └─────────────────┘                │
└─────────────────────────────┬───────────────────────────────┘
                              │
┌─────────────────────────────▼──────────────────────────────┐
│                     DATA INGESTION LAYER                    │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌────────────┐  ┌────────────┐  ┌────────────┐           │
│  │ SEC EDGAR  │  │ Market     │  │ News       │           │
│  │ Scraper    │  │ Data Feed  │  │ Scrapers   │           │
│  │ (Python)   │  │ (Polygon)  │  │ (Various)  │           │
│  └────────────┘  └────────────┘  └────────────┘           │
│                                                             │
│  ┌────────────┐  ┌────────────┐  ┌────────────┐           │
│  │ Social     │  │ Analyst    │  │ Economic   │           │
│  │ Media API  │  │ Data API   │  │ Data API   │           │
│  │ (Reddit)   │  │ (Various)  │  │ (FRED)     │           │
│  └────────────┘  └────────────┘  └────────────┘           │
│                                                             │
└─────────────────────────────┬───────────────────────────────┘
                              │
┌─────────────────────────────▼──────────────────────────────┐
│                       QUEUE LAYER                           │
├─────────────────────────────────────────────────────────────┤
│  Redis Queue / Celery                                       │
│  - Background Jobs                                          │
│  - Scheduled Tasks (cron)                                   │
│  - Data Processing Pipelines                                │
│  - Alert Processing                                         │
└─────────────────────────────────────────────────────────────┘
```

### Microservices Architecture

**Core Services:**

1. **Web API Service**
   - RESTful API for all client requests
   - User authentication/authorization
   - CRUD operations (portfolios, watchlists, alerts)
   - Proxy to other services

2. **Data Processing Service**
   - IC Score calculation engine
   - Factor score computation
   - Batch processing jobs
   - Historical data backfill

3. **Real-Time Service**
   - WebSocket server for live updates
   - Price streaming during market hours
   - Alert notifications
   - Live news feed

4. **ML Service**
   - Pattern recognition (chart patterns)
   - Sentiment analysis (NLP)
   - Stock recommendations
   - Predictive models

5. **News Aggregation Service**
   - Scrape news from multiple sources
   - NLP sentiment analysis
   - Deduplication
   - Relevance scoring

6. **Chart Service**
   - Technical indicator calculations
   - Chart data generation
   - Pattern detection
   - Drawing tools persistence

7. **Notification Service**
   - Email notifications
   - Push notifications
   - SMS (premium)
   - Webhook delivery

---

## Technology Stack

### Frontend

**Web Application:**
- **Framework:** Next.js 14+ (React 18+)
- **Language:** TypeScript
- **UI Library:** Tailwind CSS + shadcn/ui
- **Charts:** TradingView Lightweight Charts or Apache ECharts
- **State Management:** Zustand or React Context
- **Data Fetching:** TanStack Query (React Query)
- **Forms:** React Hook Form + Zod validation
- **Tables:** TanStack Table (React Table)

**Mobile Apps:**
- **iOS:** Swift, SwiftUI
- **Android:** Kotlin, Jetpack Compose
- **Shared Logic:** Consider React Native or Flutter for code reuse (alternative)

### Backend

**API Server:**
- **Framework:** FastAPI (Python) or Express (Node.js)
  - Recommended: **FastAPI** (better performance, type safety, async)
- **Language:** Python 3.11+
- **Async:** asyncio, uvicorn
- **Validation:** Pydantic
- **ORM:** SQLAlchemy 2.0 (async)
- **Migrations:** Alembic

**Data Processing:**
- **Language:** Python 3.11+
- **Libraries:** pandas, numpy, scipy
- **Parallel Processing:** multiprocessing, concurrent.futures
- **Scheduling:** APScheduler or Celery Beat

### Data Storage

**Primary Database:**
- **Database:** PostgreSQL 15+
- **Extensions:**
  - TimescaleDB (time-series data)
  - pgvector (vector embeddings for ML)
- **Connection Pooling:** pgBouncer
- **Replication:** Master-Slave (read replicas)

**Cache:**
- **Cache:** Redis 7.x
- **Use Cases:** Session storage, API responses, frequently accessed data
- **Cluster:** Redis Cluster for high availability

**Search:**
- **Engine:** Elasticsearch 8.x or TypeSense
  - Recommended: **TypeSense** (simpler, cheaper for our scale)
- **Use Cases:** Stock search, news search, screener indexing

**Object Storage:**
- **Storage:** AWS S3, Google Cloud Storage, or MinIO (self-hosted)
- **Use Cases:** PDF reports, CSV exports, backups, user uploads

### Machine Learning & Data Science

**ML Framework:**
- **Libraries:** scikit-learn, XGBoost, LightGBM
- **Deep Learning:** PyTorch (if needed)
- **NLP:** Hugging Face Transformers, FinBERT
- **Time Series:** Prophet, statsmodels

**ML Ops:**
- **Experiment Tracking:** Weights & Biases or MLflow
- **Model Serving:** FastAPI endpoints
- **Model Storage:** S3 or DVC

### Infrastructure & DevOps

**Cloud Provider:**
- **Options:** AWS, Google Cloud Platform, Azure
- **Recommended:** **AWS** (most mature, best documentation)

**Containerization:**
- **Docker:** All services containerized
- **Orchestration:** Kubernetes (EKS) or Docker Swarm
  - For MVP: Docker Swarm (simpler)
  - For scale: Kubernetes

**CI/CD:**
- **Version Control:** GitHub
- **CI/CD:** GitHub Actions
- **Testing:** pytest (Python), Jest (TypeScript)
- **Code Quality:** Black, Ruff, ESLint, Prettier

**Monitoring & Logging:**
- **Metrics:** Prometheus + Grafana
- **Logging:** ELK Stack (Elasticsearch, Logstash, Kibana) or Loki
- **APM:** Datadog, New Relic, or Sentry
- **Uptime:** UptimeRobot or Pingdom

**CDN:**
- **CDN:** CloudFlare (free tier initially)
- **Benefits:** Static asset caching, DDoS protection, SSL

---

## Data Infrastructure

### Data Sources

**FREE Data Sources:**

1. **SEC EDGAR (Free)**
   - Financial statements (10-K, 10-Q)
   - Insider trades (Form 4)
   - Institutional holdings (Form 13F)
   - Press releases (8-K)
   - API: https://www.sec.gov/edgar/searchedgar/accessing-edgar-data.htm

2. **Yahoo Finance (Free, unofficial)**
   - Historical prices
   - Real-time quotes (15-min delay)
   - Basic fundamentals
   - Library: yfinance (Python)

3. **Alpha Vantage (Free tier)**
   - Stock prices (500 calls/day free)
   - Technical indicators
   - Fundamental data
   - Economic indicators

4. **FRED (Federal Reserve Economic Data) (Free)**
   - Economic indicators (GDP, CPI, unemployment)
   - Interest rates
   - Treasury yields
   - API: https://fred.stlouisfed.org/docs/api/

5. **Reddit API (Free with limits)**
   - Social sentiment data
   - We already have this infrastructure!

6. **News APIs (Free tiers)**
   - NewsAPI.org (100 requests/day free)
   - GNews (100 requests/day free)
   - Web scraping (public articles)

**PAID Data Sources (Add as revenue grows):**

1. **Polygon.io** ($200-$600/month)
   - Real-time market data
   - Options data
   - Better coverage

2. **Finnhub** ($100-$500/month)
   - News aggregation
   - Earnings calendar
   - Analyst recommendations

3. **Quandl / Nasdaq Data Link** (varies)
   - Alternative data
   - High-quality fundamentals

### Data Pipelines

**Pipeline 1: Daily Fundamental Data**
```
Schedule: Daily at 6:00 AM ET (after market close data available)

1. Fetch SEC filings (new 10-K, 10-Q, 8-K)
2. Parse XBRL financial data
3. Calculate financial metrics
4. Store in PostgreSQL
5. Update IC Score factors
6. Recalculate IC Scores
7. Store historical scores
8. Invalidate cache
9. Trigger alerts (score changes)

Duration: ~2-4 hours
```

**Pipeline 2: Real-Time Price Data**
```
Schedule: During market hours (9:30 AM - 4:00 PM ET)

1. WebSocket connection to price feed
2. Receive tick-by-tick updates
3. Aggregate to 1-min bars
4. Store in TimescaleDB
5. Update momentum factor (IC Score)
6. Broadcast to connected clients (WebSocket)
7. Check price alerts

Latency: <1 second
```

**Pipeline 3: News Ingestion**
```
Schedule: Continuous (every 5 minutes)

1. Fetch news from APIs and scrapers
2. Deduplicate articles
3. Extract relevant tickers
4. Run NLP sentiment analysis
5. Calculate sentiment scores
6. Store in Elasticsearch + PostgreSQL
7. Update news sentiment factor (IC Score)
8. Trigger news alerts

Duration: ~30 seconds per batch
```

**Pipeline 4: Insider Trades**
```
Schedule: Daily at 8:00 PM ET

1. Fetch latest Form 4 filings from SEC
2. Parse XML data
3. Identify significant transactions
4. Calculate net insider activity
5. Update insider factor (IC Score)
6. Store in PostgreSQL
7. Trigger alerts (significant buying)

Duration: ~1 hour
```

**Pipeline 5: Institutional Holdings**
```
Schedule: Weekly (Sundays)

1. Fetch new 13F filings from SEC
2. Parse XML data
3. Identify position changes
4. Track notable investors
5. Update institutional factor (IC Score)
6. Store in PostgreSQL

Duration: ~2 hours (quarterly spike)
```

**Pipeline 6: Analyst Ratings**
```
Schedule: Hourly during market hours

1. Fetch analyst ratings from data provider
2. Calculate consensus
3. Track rating changes
4. Update analyst consensus factor (IC Score)
5. Store in PostgreSQL
6. Trigger alerts (upgrades/downgrades)

Duration: ~5 minutes
```

### Data Processing Architecture

```
┌────────────────────────────────────────────────────┐
│            Data Ingestion Layer                    │
├────────────────────────────────────────────────────┤
│                                                    │
│  Scrapers → API Clients → Webhooks                │
│                                                    │
│                     ↓                              │
│                                                    │
│  ┌──────────────────────────────────────────┐     │
│  │      Message Queue (Redis / RabbitMQ)    │     │
│  │                                          │     │
│  │  Topics:                                 │     │
│  │  - raw.prices                            │     │
│  │  - raw.news                              │     │
│  │  - raw.sec_filings                       │     │
│  │  - raw.social                            │     │
│  └──────────────┬───────────────────────────┘     │
│                 ↓                                  │
├────────────────────────────────────────────────────┤
│           Data Processing Layer                    │
├────────────────────────────────────────────────────┤
│                                                    │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐        │
│  │ Parser   │  │ Cleaner  │  │Enrichment│        │
│  │ Workers  │  │ Workers  │  │ Workers  │        │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘        │
│       │             │              │              │
│       └─────────────┴──────────────┘              │
│                     ↓                              │
│  ┌──────────────────────────────────────────┐     │
│  │        Processed Data Queue              │     │
│  │  Topics:                                 │     │
│  │  - processed.financials                  │     │
│  │  - processed.prices                      │     │
│  │  - processed.sentiment                   │     │
│  └──────────────┬───────────────────────────┘     │
├────────────────────────────────────────────────────┤
│           Calculation Layer                        │
├────────────────────────────────────────────────────┤
│                                                    │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐        │
│  │ Factor   │  │ IC Score │  │Indicator │        │
│  │ Calc     │  │ Engine   │  │ Calc     │        │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘        │
│       │             │              │              │
│       └─────────────┴──────────────┘              │
│                     ↓                              │
├────────────────────────────────────────────────────┤
│              Storage Layer                         │
├────────────────────────────────────────────────────┤
│                                                    │
│  PostgreSQL  │  TimescaleDB  │  Elasticsearch     │
│                                                    │
└────────────────────────────────────────────────────┘
```

### Scheduled Jobs

**Cron Schedule:**

```python
# Daily jobs
0 6 * * * - Calculate IC Scores (after market close data)
0 8 * * * - Fetch SEC insider trades
0 2 * * * - Database backup
0 3 * * * - Generate daily reports

# Hourly jobs (market hours)
0 9-16 * * 1-5 - Fetch analyst ratings
30 9-16 * * 1-5 - Update news sentiment

# Every 15 minutes (market hours)
*/15 9-16 * * 1-5 - Fetch news articles
*/15 9-16 * * 1-5 - Update technical indicators

# Weekly jobs
0 0 * * 0 - Fetch 13F filings
0 1 * * 0 - Clean old cache entries
0 4 * * 0 - Generate weekly analytics

# Monthly jobs
0 0 1 * * - Archive old data
0 2 1 * * - Generate monthly reports
```

---

## API Architecture

### REST API Endpoints

**Authentication:**
```
POST   /api/v1/auth/register
POST   /api/v1/auth/login
POST   /api/v1/auth/logout
POST   /api/v1/auth/refresh
GET    /api/v1/auth/me
```

**Stocks:**
```
GET    /api/v1/stocks
GET    /api/v1/stocks/{ticker}
GET    /api/v1/stocks/{ticker}/score
GET    /api/v1/stocks/{ticker}/factors
GET    /api/v1/stocks/{ticker}/history
GET    /api/v1/stocks/{ticker}/financials
GET    /api/v1/stocks/{ticker}/earnings
GET    /api/v1/stocks/{ticker}/analysts
GET    /api/v1/stocks/{ticker}/insider
GET    /api/v1/stocks/{ticker}/institutional
GET    /api/v1/stocks/{ticker}/news
GET    /api/v1/stocks/{ticker}/sentiment
GET    /api/v1/stocks/{ticker}/chart
POST   /api/v1/stocks/{ticker}/compare
```

**Screener:**
```
GET    /api/v1/screener
POST   /api/v1/screener/run
GET    /api/v1/screener/presets
POST   /api/v1/screener/save
```

**Watchlists:**
```
GET    /api/v1/watchlists
POST   /api/v1/watchlists
GET    /api/v1/watchlists/{id}
PUT    /api/v1/watchlists/{id}
DELETE /api/v1/watchlists/{id}
POST   /api/v1/watchlists/{id}/stocks
DELETE /api/v1/watchlists/{id}/stocks/{ticker}
```

**Portfolios:**
```
GET    /api/v1/portfolios
POST   /api/v1/portfolios
GET    /api/v1/portfolios/{id}
PUT    /api/v1/portfolios/{id}
DELETE /api/v1/portfolios/{id}
POST   /api/v1/portfolios/{id}/transactions
GET    /api/v1/portfolios/{id}/performance
GET    /api/v1/portfolios/{id}/analysis
```

**Alerts:**
```
GET    /api/v1/alerts
POST   /api/v1/alerts
GET    /api/v1/alerts/{id}
PUT    /api/v1/alerts/{id}
DELETE /api/v1/alerts/{id}
```

**News:**
```
GET    /api/v1/news
GET    /api/v1/news/{id}
GET    /api/v1/news/search
```

**Search:**
```
GET    /api/v1/search?q={query}
```

**User:**
```
GET    /api/v1/user/profile
PUT    /api/v1/user/profile
GET    /api/v1/user/subscription
POST   /api/v1/user/subscription
GET    /api/v1/user/preferences
PUT    /api/v1/user/preferences
```

**Reports:**
```
POST   /api/v1/reports/stock/{ticker}
POST   /api/v1/reports/portfolio/{id}
GET    /api/v1/reports/{report_id}
```

### WebSocket API

**Real-Time Connections:**
```
WS     /ws/prices/{ticker}
WS     /ws/portfolio/{id}
WS     /ws/alerts
WS     /ws/news
```

**Message Format:**
```json
{
  "type": "price_update",
  "ticker": "AAPL",
  "price": 175.50,
  "change": 2.50,
  "change_percent": 1.45,
  "timestamp": "2025-11-12T14:30:00Z"
}
```

### API Response Format

**Success Response:**
```json
{
  "success": true,
  "data": {
    "ticker": "AAPL",
    "score": 78,
    "factors": { ... }
  },
  "meta": {
    "timestamp": "2025-11-12T14:30:00Z",
    "cached": false,
    "ttl": 3600
  }
}
```

**Error Response:**
```json
{
  "success": false,
  "error": {
    "code": "INVALID_TICKER",
    "message": "Stock ticker not found",
    "details": {
      "ticker": "XYZ"
    }
  },
  "meta": {
    "timestamp": "2025-11-12T14:30:00Z"
  }
}
```

### Rate Limiting

**Tiers:**
- Free: 100 requests/day
- Premium: 10,000 requests/day
- Pro: 100,000 requests/day
- Enterprise: Unlimited

**Headers:**
```
X-RateLimit-Limit: 10000
X-RateLimit-Remaining: 9850
X-RateLimit-Reset: 1699824000
```

**Response (429):**
```json
{
  "success": false,
  "error": {
    "code": "RATE_LIMIT_EXCEEDED",
    "message": "API rate limit exceeded",
    "details": {
      "limit": 10000,
      "reset_at": "2025-11-13T00:00:00Z"
    }
  }
}
```

---

(continued in next file due to length)

**End of Technical Specification Part 1**
