# Reddit Integration - Phase 1: Ticker Heatmap Spec

## 1. Overview

Phase 1 focuses on building a **Ticker Heatmap** visualization showing the popularity of stocks on Reddit based on historical ranking data from ApeWisdom API. This provides immediate value to users without requiring complex Reddit API setup.

### Goals
- Collect historical ranking data from ApeWisdom API
- Store ranking trends in PostgreSQL
- Build an interactive heatmap showing ticker popularity over time
- Enable users to discover trending stocks and identify momentum shifts

### Non-Goals (Deferred to Later Phases)
- Direct Reddit post scraping
- AI-powered summarization
- Real-time sentiment analysis
- Individual post display

## 2. Data Source: ApeWisdom API

**API Endpoint**: `https://apewisdom.io/api/v1.0/filter/all-stocks/page/0`

**Response Structure**:
```json
{
  "results": [
    {
      "rank": 1,
      "ticker": "AAPL",
      "name": "Apple Inc.",
      "mentions": 342,
      "upvotes": 15234,
      "rank_24h_ago": 3,
      "mentions_24h_ago": 289
    }
  ],
  "updated_at": "2025-10-21T10:00:00Z"
}
```

**Key Fields**:
- `rank`: Current ranking by mentions
- `ticker`: Stock symbol
- `mentions`: Total mentions across tracked subreddits
- `upvotes`: Total upvotes on posts mentioning this ticker
- `rank_24h_ago`: Rank 24 hours prior (for trend detection)
- `mentions_24h_ago`: Mentions 24 hours prior

**Rate Limits**:
- Free tier: 100 requests/day
- Updates: Every hour
- Coverage: Top 100 stocks

**Subreddits Tracked by ApeWisdom**:
- r/wallstreetbets
- r/stocks
- r/investing
- r/StockMarket

## 3. Database Schema

### 3.1 Migration: `008_create_reddit_heatmap_tables.sql`

```sql
-- Reddit ranking snapshots from ApeWisdom
CREATE TABLE reddit_ticker_rankings (
    id BIGSERIAL PRIMARY KEY,
    ticker_symbol VARCHAR(10) NOT NULL,
    rank INT NOT NULL,
    mentions INT NOT NULL,
    upvotes INT DEFAULT 0,
    rank_24h_ago INT,
    mentions_24h_ago INT,
    snapshot_date DATE NOT NULL,
    snapshot_time TIMESTAMP NOT NULL,
    data_source VARCHAR(20) DEFAULT 'apewisdom',
    created_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(ticker_symbol, snapshot_date, data_source),
    FOREIGN KEY (ticker_symbol) REFERENCES tickers(symbol) ON DELETE CASCADE
);

-- Aggregated daily metrics for heatmap
CREATE TABLE reddit_heatmap_daily (
    id BIGSERIAL PRIMARY KEY,
    ticker_symbol VARCHAR(10) NOT NULL,
    date DATE NOT NULL,
    avg_rank DECIMAL(5,2),              -- Average rank for the day
    min_rank INT,                        -- Best (lowest) rank of the day
    max_rank INT,                        -- Worst (highest) rank of the day
    total_mentions INT,
    total_upvotes INT,
    rank_volatility DECIMAL(5,2),       -- Std deviation of rank
    trend_direction VARCHAR(10),         -- 'rising', 'falling', 'stable'
    popularity_score DECIMAL(8,2),      -- Calculated score for heatmap intensity
    data_source VARCHAR(20) DEFAULT 'apewisdom',
    calculated_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(ticker_symbol, date, data_source),
    FOREIGN KEY (ticker_symbol) REFERENCES tickers(symbol) ON DELETE CASCADE
);

-- Indexes for performance
CREATE INDEX idx_reddit_rankings_ticker_date ON reddit_ticker_rankings(ticker_symbol, snapshot_date DESC);
CREATE INDEX idx_reddit_rankings_date ON reddit_ticker_rankings(snapshot_date DESC, rank);
CREATE INDEX idx_reddit_heatmap_date ON reddit_heatmap_daily(date DESC, popularity_score DESC);
CREATE INDEX idx_reddit_heatmap_ticker ON reddit_heatmap_daily(ticker_symbol, date DESC);
```

### 3.2 Popularity Score Calculation

The `popularity_score` determines heatmap cell color intensity:

```
popularity_score =
  (mentions × 0.4) +
  (upvotes / 100 × 0.3) +
  ((101 - avg_rank) × 0.3)

Normalized to 0-100 scale
```

**Color Mapping**:
- 80-100: Deep red (very hot)
- 60-79: Orange (hot)
- 40-59: Yellow (warm)
- 20-39: Light green (cool)
- 0-19: White/gray (cold)

### 3.3 Trend Direction Logic

```
if avg_rank_today < avg_rank_yesterday - 5:
    trend_direction = 'rising'   # Gaining popularity (rank decreasing)
elif avg_rank_today > avg_rank_yesterday + 5:
    trend_direction = 'falling'  # Losing popularity (rank increasing)
else:
    trend_direction = 'stable'
```

## 4. Backend Implementation

### 4.1 Python Data Collection Script

**New file**: `scripts/apewisdom_collector.py`

```python
"""
ApeWisdom Historical Ranking Collector

Fetches top 100 stock rankings from ApeWisdom API and stores in PostgreSQL.
Runs hourly via Kubernetes CronJob.

Usage:
    DB_HOST=localhost DB_USER=investorcenter DB_NAME=investorcenter_db \
    scripts/venv/bin/python scripts/apewisdom_collector.py --limit 100
"""

import requests
import psycopg2
from datetime import datetime, timezone
from typing import List, Dict, Optional
import time
import os
import argparse

class ApeWisdomCollector:
    BASE_URL = "https://apewisdom.io/api/v1.0/filter/all-stocks"

    def __init__(self, db_host: str, db_user: str, db_name: str, db_password: str):
        self.db_conn = psycopg2.connect(
            host=db_host,
            user=db_user,
            database=db_name,
            password=db_password
        )
        self.db_conn.autocommit = False

    def fetch_rankings(self, page: int = 0) -> Optional[Dict]:
        """Fetch rankings from ApeWisdom API"""
        url = f"{self.BASE_URL}/page/{page}"

        try:
            response = requests.get(url, timeout=10)
            response.raise_for_status()
            return response.json()
        except requests.RequestException as e:
            print(f"Error fetching page {page}: {e}")
            return None

    def store_ranking(self, ticker: str, data: Dict, snapshot_time: datetime):
        """Store single ticker ranking in database"""
        cursor = self.db_conn.cursor()

        try:
            # Check if ticker exists in tickers table
            cursor.execute(
                "SELECT symbol FROM tickers WHERE symbol = %s",
                (ticker,)
            )

            if not cursor.fetchone():
                print(f"Ticker {ticker} not found in database, skipping")
                return

            # Insert ranking data
            cursor.execute("""
                INSERT INTO reddit_ticker_rankings
                (ticker_symbol, rank, mentions, upvotes, rank_24h_ago,
                 mentions_24h_ago, snapshot_date, snapshot_time, data_source)
                VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s)
                ON CONFLICT (ticker_symbol, snapshot_date, data_source)
                DO UPDATE SET
                    rank = EXCLUDED.rank,
                    mentions = EXCLUDED.mentions,
                    upvotes = EXCLUDED.upvotes,
                    rank_24h_ago = EXCLUDED.rank_24h_ago,
                    mentions_24h_ago = EXCLUDED.mentions_24h_ago,
                    snapshot_time = EXCLUDED.snapshot_time
            """, (
                ticker,
                data.get('rank'),
                data.get('mentions', 0),
                data.get('upvotes', 0),
                data.get('rank_24h_ago'),
                data.get('mentions_24h_ago'),
                snapshot_time.date(),
                snapshot_time,
                'apewisdom'
            ))

        except Exception as e:
            print(f"Error storing {ticker}: {e}")
            raise

    def collect_all_rankings(self, max_pages: int = 5):
        """Collect rankings from all pages (up to top 100)"""
        snapshot_time = datetime.now(timezone.utc)
        stored_count = 0

        for page in range(max_pages):
            print(f"Fetching page {page}...")

            data = self.fetch_rankings(page)
            if not data or 'results' not in data:
                break

            results = data['results']
            if not results:
                break

            for item in results:
                ticker = item.get('ticker')
                if ticker:
                    try:
                        self.store_ranking(ticker, item, snapshot_time)
                        stored_count += 1
                    except Exception as e:
                        print(f"Failed to store {ticker}: {e}")
                        self.db_conn.rollback()
                        continue

            # Commit after each page
            self.db_conn.commit()
            print(f"Stored {len(results)} rankings from page {page}")

            # Rate limiting: 1 second between pages
            if page < max_pages - 1:
                time.sleep(1)

        print(f"Total rankings stored: {stored_count}")
        return stored_count

    def calculate_daily_metrics(self, target_date: Optional[datetime] = None):
        """Calculate aggregated daily metrics for heatmap"""
        if not target_date:
            target_date = datetime.now(timezone.utc).date()

        cursor = self.db_conn.cursor()

        cursor.execute("""
            INSERT INTO reddit_heatmap_daily
            (ticker_symbol, date, avg_rank, min_rank, max_rank,
             total_mentions, total_upvotes, rank_volatility,
             popularity_score, data_source)
            SELECT
                ticker_symbol,
                snapshot_date as date,
                AVG(rank) as avg_rank,
                MIN(rank) as min_rank,
                MAX(rank) as max_rank,
                SUM(mentions) as total_mentions,
                SUM(upvotes) as total_upvotes,
                STDDEV(rank) as rank_volatility,
                -- Popularity score calculation
                LEAST(100, (
                    (SUM(mentions) * 0.4) +
                    (SUM(upvotes) / 100.0 * 0.3) +
                    ((101 - AVG(rank)) * 0.3)
                )) as popularity_score,
                data_source
            FROM reddit_ticker_rankings
            WHERE snapshot_date = %s
            GROUP BY ticker_symbol, snapshot_date, data_source
            ON CONFLICT (ticker_symbol, date, data_source)
            DO UPDATE SET
                avg_rank = EXCLUDED.avg_rank,
                min_rank = EXCLUDED.min_rank,
                max_rank = EXCLUDED.max_rank,
                total_mentions = EXCLUDED.total_mentions,
                total_upvotes = EXCLUDED.total_upvotes,
                rank_volatility = EXCLUDED.rank_volatility,
                popularity_score = EXCLUDED.popularity_score,
                calculated_at = NOW()
        """, (target_date,))

        # Calculate trend direction
        cursor.execute("""
            UPDATE reddit_heatmap_daily AS today
            SET trend_direction = CASE
                WHEN today.avg_rank < yesterday.avg_rank - 5 THEN 'rising'
                WHEN today.avg_rank > yesterday.avg_rank + 5 THEN 'falling'
                ELSE 'stable'
            END
            FROM reddit_heatmap_daily AS yesterday
            WHERE today.ticker_symbol = yesterday.ticker_symbol
                AND today.date = %s
                AND yesterday.date = %s - INTERVAL '1 day'
                AND today.data_source = yesterday.data_source
        """, (target_date, target_date))

        self.db_conn.commit()
        print(f"Daily metrics calculated for {target_date}")

    def close(self):
        self.db_conn.close()


def main():
    parser = argparse.ArgumentParser(description='Collect ApeWisdom rankings')
    parser.add_argument('--limit', type=int, default=100,
                       help='Max tickers to collect (default: 100)')
    parser.add_argument('--calculate-metrics', action='store_true',
                       help='Calculate daily metrics after collection')
    args = parser.parse_args()

    # Database config from environment
    db_config = {
        'db_host': os.getenv('DB_HOST', 'localhost'),
        'db_user': os.getenv('DB_USER', 'investorcenter'),
        'db_name': os.getenv('DB_NAME', 'investorcenter_db'),
        'db_password': os.getenv('DB_PASSWORD', '')
    }

    collector = ApeWisdomCollector(**db_config)

    try:
        # Collect rankings (5 pages = ~100 tickers)
        max_pages = (args.limit + 19) // 20  # 20 per page
        collector.collect_all_rankings(max_pages=max_pages)

        # Calculate daily metrics if requested
        if args.calculate_metrics:
            collector.calculate_daily_metrics()

    finally:
        collector.close()


if __name__ == '__main__':
    main()
```

### 4.2 Go Backend API Endpoints

**New file**: `backend/handlers/reddit_heatmap_handlers.go`

```go
package handlers

import (
    "net/http"
    "strconv"
    "time"

    "github.com/gin-gonic/gin"
    "investorcenter/database"
    "investorcenter/models"
)

// GetRedditHeatmap returns heatmap data for specified date range
// GET /api/v1/reddit/heatmap?days=30&top=50
func GetRedditHeatmap(c *gin.Context) {
    daysStr := c.DefaultQuery("days", "30")
    topStr := c.DefaultQuery("top", "50")

    days, _ := strconv.Atoi(daysStr)
    topN, _ := strconv.Atoi(topStr)

    if days > 90 {
        days = 90 // Max 90 days
    }
    if topN > 100 {
        topN = 100 // Max 100 tickers
    }

    data, err := database.GetHeatmapData(days, topN)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{
            "error": "Failed to fetch heatmap data",
            "details": err.Error(),
        })
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "data": data,
        "days": days,
        "top_tickers": topN,
        "generated_at": time.Now(),
    })
}

// GetTickerRedditHistory returns Reddit ranking history for a ticker
// GET /api/v1/reddit/ticker/:symbol/history?days=30
func GetTickerRedditHistory(c *gin.Context) {
    symbol := c.Param("symbol")
    daysStr := c.DefaultQuery("days", "30")
    days, _ := strconv.Atoi(daysStr)

    history, err := database.GetTickerRedditHistory(symbol, days)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{
            "error": "Failed to fetch ticker history",
        })
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "ticker": symbol,
        "history": history,
        "days": days,
    })
}
```

**New file**: `backend/database/reddit_heatmap.go`

```go
package database

import (
    "time"
    "investorcenter/models"
)

// HeatmapDataPoint represents one cell in the heatmap
type HeatmapDataPoint struct {
    Ticker          string    `json:"ticker"`
    Date            string    `json:"date"`
    AvgRank         float64   `json:"avg_rank"`
    PopularityScore float64   `json:"popularity_score"`
    TotalMentions   int       `json:"total_mentions"`
    TrendDirection  string    `json:"trend_direction"`
}

// GetHeatmapData fetches heatmap data for top N tickers over last X days
func GetHeatmapData(days int, topN int) ([]HeatmapDataPoint, error) {
    query := `
        WITH top_tickers AS (
            SELECT ticker_symbol, SUM(total_mentions) as total
            FROM reddit_heatmap_daily
            WHERE date >= NOW() - INTERVAL '%d days'
            GROUP BY ticker_symbol
            ORDER BY total DESC
            LIMIT %d
        )
        SELECT
            h.ticker_symbol,
            h.date::text,
            h.avg_rank,
            h.popularity_score,
            h.total_mentions,
            COALESCE(h.trend_direction, 'stable') as trend_direction
        FROM reddit_heatmap_daily h
        INNER JOIN top_tickers t ON h.ticker_symbol = t.ticker_symbol
        WHERE h.date >= NOW() - INTERVAL '%d days'
        ORDER BY t.total DESC, h.date DESC
    `

    rows, err := DB.Query(fmt.Sprintf(query, days, topN, days))
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var data []HeatmapDataPoint
    for rows.Next() {
        var point HeatmapDataPoint
        err := rows.Scan(
            &point.Ticker,
            &point.Date,
            &point.AvgRank,
            &point.PopularityScore,
            &point.TotalMentions,
            &point.TrendDirection,
        )
        if err != nil {
            return nil, err
        }
        data = append(data, point)
    }

    return data, nil
}

// GetTickerRedditHistory returns daily metrics for a single ticker
func GetTickerRedditHistory(symbol string, days int) ([]HeatmapDataPoint, error) {
    query := `
        SELECT
            ticker_symbol,
            date::text,
            avg_rank,
            popularity_score,
            total_mentions,
            COALESCE(trend_direction, 'stable') as trend_direction
        FROM reddit_heatmap_daily
        WHERE ticker_symbol = $1
            AND date >= NOW() - INTERVAL '%d days'
        ORDER BY date DESC
    `

    rows, err := DB.Query(fmt.Sprintf(query, days), symbol)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var history []HeatmapDataPoint
    for rows.Next() {
        var point HeatmapDataPoint
        err := rows.Scan(
            &point.Ticker,
            &point.Date,
            &point.AvgRank,
            &point.PopularityScore,
            &point.TotalMentions,
            &point.TrendDirection,
        )
        if err != nil {
            return nil, err
        }
        history = append(history, point)
    }

    return history, nil
}
```

**Update**: `backend/main.go` - Add routes

```go
// Reddit heatmap endpoints
reddit := v1.Group("/reddit")
{
    reddit.GET("/heatmap", handlers.GetRedditHeatmap)
    reddit.GET("/ticker/:symbol/history", handlers.GetTickerRedditHistory)
}
```

## 5. Frontend: Ticker Heatmap Component

### 5.1 Heatmap Visualization

**New file**: `components/reddit/TickerHeatmap.tsx`

```typescript
'use client'

import React, { useEffect, useState } from 'react'
import { Tooltip } from '@/components/ui/Tooltip'

interface HeatmapCell {
  ticker: string
  date: string
  avg_rank: number
  popularity_score: number
  total_mentions: number
  trend_direction: 'rising' | 'falling' | 'stable'
}

interface HeatmapProps {
  days?: number
  topTickers?: number
}

export function TickerHeatmap({ days = 30, topTickers = 50 }: HeatmapProps) {
  const [data, setData] = useState<HeatmapCell[]>([])
  const [loading, setLoading] = useState(true)
  const [selectedTicker, setSelectedTicker] = useState<string | null>(null)

  useEffect(() => {
    fetchHeatmapData()
  }, [days, topTickers])

  const fetchHeatmapData = async () => {
    try {
      const response = await fetch(
        `/api/v1/reddit/heatmap?days=${days}&top=${topTickers}`
      )
      const result = await response.json()
      setData(result.data || [])
    } catch (error) {
      console.error('Failed to load heatmap:', error)
    } finally {
      setLoading(false)
    }
  }

  // Group data by ticker and date
  const groupedData = React.useMemo(() => {
    const tickers = Array.from(new Set(data.map(d => d.ticker)))
    const dates = Array.from(new Set(data.map(d => d.date))).sort()

    const grid: Record<string, Record<string, HeatmapCell>> = {}

    data.forEach(cell => {
      if (!grid[cell.ticker]) {
        grid[cell.ticker] = {}
      }
      grid[cell.ticker][cell.date] = cell
    })

    return { tickers, dates, grid }
  }, [data])

  const getColorForScore = (score: number): string => {
    if (score >= 80) return 'bg-red-600'
    if (score >= 60) return 'bg-orange-500'
    if (score >= 40) return 'bg-yellow-400'
    if (score >= 20) return 'bg-green-300'
    return 'bg-gray-200'
  }

  const getTrendIcon = (direction: string): string => {
    if (direction === 'rising') return '↗️'
    if (direction === 'falling') return '↘️'
    return '→'
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center h-96">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600" />
      </div>
    )
  }

  return (
    <div className="w-full overflow-x-auto">
      <div className="min-w-max">
        <div className="flex">
          {/* Ticker column */}
          <div className="flex flex-col sticky left-0 bg-white z-10 border-r">
            <div className="h-12 flex items-center px-4 font-semibold border-b">
              Ticker
            </div>
            {groupedData.tickers.map(ticker => (
              <div
                key={ticker}
                className="h-10 flex items-center px-4 border-b hover:bg-gray-50 cursor-pointer"
                onClick={() => setSelectedTicker(ticker)}
              >
                <span className="font-mono font-semibold">{ticker}</span>
              </div>
            ))}
          </div>

          {/* Date columns */}
          {groupedData.dates.map(date => (
            <div key={date} className="flex flex-col">
              <div className="h-12 flex items-center justify-center px-2 text-xs font-medium border-b bg-gray-50">
                {new Date(date).toLocaleDateString('en-US', {
                  month: 'short',
                  day: 'numeric'
                })}
              </div>
              {groupedData.tickers.map(ticker => {
                const cell = groupedData.grid[ticker]?.[date]

                return (
                  <Tooltip
                    key={`${ticker}-${date}`}
                    content={
                      cell ? (
                        <div className="text-xs">
                          <div className="font-semibold">{ticker}</div>
                          <div>Rank: {cell.avg_rank.toFixed(0)}</div>
                          <div>Mentions: {cell.total_mentions}</div>
                          <div>Score: {cell.popularity_score.toFixed(1)}</div>
                          <div>Trend: {getTrendIcon(cell.trend_direction)} {cell.trend_direction}</div>
                        </div>
                      ) : (
                        <div className="text-xs">No data</div>
                      )
                    }
                  >
                    <div
                      className={`
                        h-10 w-12 border-b border-r
                        ${cell ? getColorForScore(cell.popularity_score) : 'bg-gray-100'}
                        hover:opacity-80 cursor-pointer
                        transition-opacity
                      `}
                    />
                  </Tooltip>
                )
              })}
            </div>
          ))}
        </div>
      </div>

      {/* Legend */}
      <div className="mt-6 flex items-center gap-6">
        <div className="text-sm font-semibold">Popularity Score:</div>
        <div className="flex items-center gap-2">
          <div className="w-8 h-8 bg-gray-200" />
          <span className="text-xs">Cold (0-20)</span>
        </div>
        <div className="flex items-center gap-2">
          <div className="w-8 h-8 bg-green-300" />
          <span className="text-xs">Cool (20-40)</span>
        </div>
        <div className="flex items-center gap-2">
          <div className="w-8 h-8 bg-yellow-400" />
          <span className="text-xs">Warm (40-60)</span>
        </div>
        <div className="flex items-center gap-2">
          <div className="w-8 h-8 bg-orange-500" />
          <span className="text-xs">Hot (60-80)</span>
        </div>
        <div className="flex items-center gap-2">
          <div className="w-8 h-8 bg-red-600" />
          <span className="text-xs">Very Hot (80-100)</span>
        </div>
      </div>
    </div>
  )
}
```

### 5.2 New Page: Reddit Heatmap

**New file**: `app/reddit/heatmap/page.tsx`

```typescript
import { TickerHeatmap } from '@/components/reddit/TickerHeatmap'
import { Metadata } from 'next'

export const metadata: Metadata = {
  title: 'Reddit Ticker Heatmap | InvestorCenter.ai',
  description: 'Visualize stock popularity trends on Reddit communities',
}

export default function RedditHeatmapPage() {
  return (
    <div className="container mx-auto px-4 py-8">
      <div className="mb-8">
        <h1 className="text-3xl font-bold mb-2">Reddit Ticker Heatmap</h1>
        <p className="text-gray-600">
          Discover trending stocks based on Reddit community discussions.
          Darker colors indicate higher popularity.
        </p>
      </div>

      <div className="mb-6 flex items-center gap-4">
        <div className="text-sm text-gray-500">
          Data source: ApeWisdom (r/wallstreetbets, r/stocks, r/investing, r/StockMarket)
        </div>
        <div className="text-sm text-gray-500">
          Updated: Hourly
        </div>
      </div>

      <TickerHeatmap days={30} topTickers={50} />

      <div className="mt-8 bg-blue-50 border border-blue-200 rounded-lg p-4">
        <h3 className="font-semibold mb-2">How to Use</h3>
        <ul className="text-sm space-y-1 text-gray-700">
          <li>• Hover over cells to see detailed metrics</li>
          <li>• Click ticker symbols to view full history</li>
          <li>• Rising trends show increasing popularity (↗️)</li>
          <li>• Falling trends show decreasing popularity (↘️)</li>
        </ul>
      </div>
    </div>
  )
}
```

## 6. Kubernetes Automation

**New file**: `k8s/apewisdom-collector-cronjob.yaml`

```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: apewisdom-collector
  namespace: investorcenter
spec:
  schedule: "0 * * * *"  # Every hour at :00
  successfulJobsHistoryLimit: 3
  failedJobsHistoryLimit: 1
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: apewisdom-collector
            image: 360358043271.dkr.ecr.us-east-1.amazonaws.com/investorcenter/apewisdom-collector:latest
            env:
            - name: DB_HOST
              value: postgres-service
            - name: DB_USER
              valueFrom:
                secretKeyRef:
                  name: postgres-secret
                  key: username
            - name: DB_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: postgres-secret
                  key: password
            - name: DB_NAME
              value: investorcenter_db
            command:
            - /bin/sh
            - -c
            - |
              python3 /app/apewisdom_collector.py --limit 100 --calculate-metrics
          restartPolicy: OnFailure
```

## 7. Implementation Plan

### Week 1: Database & Python Collector
- [ ] Create migration `008_create_reddit_heatmap_tables.sql`
- [ ] Run migration on local database
- [ ] Build `apewisdom_collector.py` script
- [ ] Test locally with `--limit 20` flag
- [ ] Verify data in `reddit_ticker_rankings` and `reddit_heatmap_daily`

### Week 2: Backend API
- [ ] Implement `reddit_heatmap_handlers.go`
- [ ] Implement `database/reddit_heatmap.go` queries
- [ ] Add routes to `main.go`
- [ ] Test API endpoints locally:
  - `GET /api/v1/reddit/heatmap?days=7&top=10`
  - `GET /api/v1/reddit/ticker/AAPL/history?days=7`

### Week 3: Frontend Heatmap
- [ ] Build `TickerHeatmap.tsx` component
- [ ] Create `/reddit/heatmap` page
- [ ] Implement tooltip component
- [ ] Add color legend
- [ ] Test responsive layout

### Week 4: Deployment & Automation
- [ ] Create Dockerfile for collector script
- [ ] Push to ECR: `investorcenter/apewisdom-collector:latest`
- [ ] Deploy Kubernetes CronJob
- [ ] Monitor first 24 hours of data collection
- [ ] Deploy frontend updates
- [ ] Add navigation link to heatmap page

## 8. API Endpoints Summary

### Get Heatmap Data
```
GET /api/v1/reddit/heatmap?days=30&top=50
```

Response:
```json
{
  "data": [
    {
      "ticker": "AAPL",
      "date": "2025-10-21",
      "avg_rank": 3.5,
      "popularity_score": 87.2,
      "total_mentions": 342,
      "trend_direction": "rising"
    }
  ],
  "days": 30,
  "top_tickers": 50,
  "generated_at": "2025-10-21T10:00:00Z"
}
```

### Get Ticker History
```
GET /api/v1/reddit/ticker/:symbol/history?days=30
```

Response:
```json
{
  "ticker": "AAPL",
  "history": [
    {
      "ticker": "AAPL",
      "date": "2025-10-21",
      "avg_rank": 3.5,
      "popularity_score": 87.2,
      "total_mentions": 342,
      "trend_direction": "rising"
    }
  ],
  "days": 30
}
```

## 9. Testing Plan

### Manual Testing
1. Run collector locally: `DB_HOST=localhost python3 scripts/apewisdom_collector.py --limit 20`
2. Verify database entries: `SELECT * FROM reddit_ticker_rankings ORDER BY snapshot_time DESC LIMIT 10;`
3. Check daily metrics: `SELECT * FROM reddit_heatmap_daily ORDER BY date DESC, popularity_score DESC;`
4. Test API: `curl http://localhost:8080/api/v1/reddit/heatmap?days=7&top=10`
5. Load frontend: `http://localhost:3000/reddit/heatmap`

### Production Monitoring
- CloudWatch logs for CronJob execution
- Database row counts: `SELECT COUNT(*) FROM reddit_ticker_rankings;`
- API response times (should be < 500ms)
- Frontend load time (should be < 2s)

## 10. Cost Estimate

- **ApeWisdom API**: Free (100 requests/day, using 24/day)
- **Compute**: Minimal (CronJob runs ~2 min/hour)
- **Storage**: ~50 MB/month for 90 days of data
- **Total**: ~$0/month (free tier)

## 11. Success Metrics

- **Data Collection**: 100 tickers collected hourly
- **API Performance**: < 500ms response time for heatmap endpoint
- **User Engagement**: Track page views on `/reddit/heatmap`
- **Data Quality**: > 95% successful CronJob runs

## 12. Future Enhancements (Phase 2+)

Once Phase 1 is stable:
- Add sentiment analysis from ApeWisdom (if available)
- Implement filtering by subreddit
- Add click-through to ticker detail pages
- Build "momentum" alerts (notify when ticker jumps 20+ ranks)
- Integrate with existing search functionality
- Add comparison view (compare 2 tickers side-by-side)

---

**Recommendation**: Start with Week 1-2 to build the foundation (database + collector + API), then validate data quality before proceeding to frontend visualization.
