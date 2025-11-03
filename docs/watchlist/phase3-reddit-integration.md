# Phase 3 Heatmap - Reddit Data Integration

## Overview

The Phase 3 Custom Heatmap Visualization now includes full integration with your existing Reddit trending stocks data from the `reddit_heatmap_daily` table.

## Reddit Data Currently Available

### Database Tables

You already have Reddit data implemented and populated:

```sql
-- Table: reddit_heatmap_daily
-- Records: 59 (as of 2025-11-02)
-- Data Source: ApeWisdom API

SELECT
    ticker_symbol,
    avg_rank,              -- Reddit ranking (lower = more trending)
    total_mentions,        -- Number of Reddit mentions
    popularity_score,      -- Calculated score (0-100)
    trend_direction,       -- "rising", "falling", "stable"
    date
FROM reddit_heatmap_daily
ORDER BY popularity_score DESC;
```

### Sample Data

Top trending stocks from your current data:

| Symbol | Rank | Mentions | Popularity | Trend   |
|--------|------|----------|------------|---------|
| BYND   | 1.0  | 363      | 100.0      | rising  |
| ASST   | 2.0  | 185      | 100.0      | rising  |
| SPY    | 3.0  | 69       | 57.8       | stable  |
| DTE    | 4.0  | 47       | 48.3       | rising  |
| TSLA   | 7.0  | 40       | 44.6       | falling |
| GME    | 6.0  | 45       | 47.1       | rising  |

## New Heatmap Metrics Using Reddit Data

### 1. Size Metrics (Tile Size)

#### **Reddit Mentions**
- **Value**: `total_mentions` from reddit_heatmap_daily
- **Display**: "363 mentions"
- **Use Case**: Show which stocks are being discussed most on Reddit
- **Larger tiles** = More mentions

#### **Reddit Popularity**
- **Value**: `popularity_score` (0-100)
- **Formula**: `(mentions × 0.4) + (upvotes/100 × 0.3) + ((101 - rank) × 0.3)`
- **Display**: "100 score"
- **Use Case**: Composite measure of Reddit buzz
- **Larger tiles** = Higher popularity score

### 2. Color Metrics (Tile Color)

#### **Reddit Rank**
- **Value**: Inverted rank (101 - rank) for color scale
- **Display**: "#1", "#5", "#20"
- **Use Case**: Show trending ranking on Reddit (lower number = higher trending)
- **Color Scale**:
  - **Green**: Rank 1-10 (top trending)
  - **Yellow**: Rank 11-30
  - **Red**: Rank 31+

#### **Reddit Trend**
- **Values**: "rising" (+10), "stable" (0), "falling" (-10)
- **Display**: "↑ Rising", "→ Stable", "↓ Falling"
- **Use Case**: Show momentum of Reddit interest
- **Color Scale**:
  - **Green**: Rising
  - **Gray**: Stable
  - **Red**: Falling

## Heatmap Configuration Examples

### Example 1: Reddit Buzz Heatmap
```json
{
  "name": "Reddit Trending",
  "size_metric": "reddit_mentions",
  "color_metric": "reddit_rank",
  "time_period": "1D",
  "color_scheme": "red_green"
}
```
**Result**: Large tiles = many mentions, Green tiles = top ranking

### Example 2: Reddit Momentum Heatmap
```json
{
  "name": "Reddit Momentum",
  "size_metric": "reddit_popularity",
  "color_metric": "reddit_trend",
  "time_period": "1D",
  "color_scheme": "red_green"
}
```
**Result**: Large tiles = high popularity, Green tiles = rising interest

### Example 3: Hybrid Price + Reddit Heatmap
```json
{
  "name": "Price vs Reddit",
  "size_metric": "market_cap",
  "color_metric": "reddit_rank",
  "time_period": "1D",
  "color_scheme": "blue_red"
}
```
**Result**: Large tiles = large cap, Blue tiles = trending on Reddit

## Tooltip Display

When hovering over a heatmap tile, Reddit data is displayed:

```
TSLA - Tesla Inc.
──────────────────
Price:          $250.45
Change:         +5.20 (+2.12%)
Market Cap:     $800.2B
Volume:         120.5M

────── Reddit ──────
Reddit Rank:    #7
Reddit Mentions: 40
Reddit Score:   44.6/100
Reddit Trend:   ↓ falling (-3)
```

## Data Flow

```
┌─────────────────────────┐
│  reddit_heatmap_daily   │
│  (59 records)           │
└───────────┬─────────────┘
            │
            │ LEFT JOIN LATERAL
            │ (get latest data per ticker)
            ↓
┌─────────────────────────┐
│ GetWatchListItemsWithData│
│ (Phase 2 function)      │
└───────────┬─────────────┘
            │
            │ Returns: WatchListItemWithData
            │ (includes Reddit fields)
            ↓
┌─────────────────────────┐
│   HeatmapService        │
│   GenerateHeatmapData() │
└───────────┬─────────────┘
            │
            │ Populates: HeatmapTile
            │ (reddit_rank, reddit_mentions, etc.)
            ↓
┌─────────────────────────┐
│  GET /api/v1/watchlists │
│  /:id/heatmap           │
└───────────┬─────────────┘
            │
            │ JSON response
            ↓
┌─────────────────────────┐
│  WatchListHeatmap.tsx   │
│  (D3.js treemap)        │
└─────────────────────────┘
```

## Implementation Checklist

### Backend Changes

- [x] Reddit tables exist (`reddit_heatmap_daily`)
- [x] Reddit data populated (59 records)
- [x] Reddit handlers exist (`GetRedditHeatmap`, `GetTickerRedditHistory`)
- [ ] Update `GetWatchListItemsWithData()` to JOIN reddit_heatmap_daily
- [ ] Add Reddit fields to `WatchListItemWithData` model
- [ ] Update `calculateSizeValue()` for reddit_mentions and reddit_popularity
- [ ] Update `calculateColorValue()` for reddit_rank and reddit_trend
- [ ] Add Reddit options to validation rules

### Frontend Changes

- [ ] Add Reddit options to size metric dropdown
- [ ] Add Reddit options to color metric dropdown
- [ ] Add Reddit data to tooltip display
- [ ] Update TypeScript interfaces with Reddit fields
- [ ] Update API client types

### Testing

- [ ] Test reddit_mentions as size metric
- [ ] Test reddit_popularity as size metric
- [ ] Test reddit_rank as color metric
- [ ] Test reddit_trend as color metric
- [ ] Verify tooltip shows Reddit data correctly
- [ ] Test with stocks that have NO Reddit data (should handle gracefully)
- [ ] Test save/load configurations with Reddit metrics

## Sample Use Cases

### 1. WallStreetBets Watch List
**User Goal**: Track meme stocks trending on Reddit
**Configuration**:
- Size: `reddit_mentions` (bigger = more discussion)
- Color: `reddit_trend` (green = rising interest)
- Stocks: GME, AMC, TSLA, PLTR, etc.

### 2. Contrarian Strategy
**User Goal**: Find stocks losing Reddit interest but fundamentally strong
**Configuration**:
- Size: `market_cap` (bigger = larger company)
- Color: `reddit_rank` (red = losing Reddit buzz)
- Filter: Only stocks with rank > 50

### 3. Momentum Trading
**User Goal**: Find stocks gaining both price and Reddit momentum
**Configuration**:
- Size: `reddit_popularity`
- Color: `price_change_pct`
- Look for: Large green tiles (high popularity + price gain)

## Data Freshness

Your Reddit data is currently populated with mock data. For production:

### Current State
- **Source**: Mock SQL script (`insert_mock_reddit_data.sql`)
- **Frequency**: One-time insert
- **Coverage**: 20 tickers over 7 days

### Production Recommendation
- **Source**: ApeWisdom API (already referenced in migration)
- **Frequency**: Hourly or daily updates via CronJob
- **Implementation**: See `scripts/deploy-reddit-collector.sh` and `Dockerfile.reddit-collector`

## API Endpoints Available

### Get Reddit Heatmap (Standalone)
```bash
GET /api/v1/reddit/heatmap?days=7&top=50

Response:
{
  "data": [
    {
      "tickerSymbol": "BYND",
      "avgRank": 4.0,
      "totalMentions": 1343,
      "popularityScore": 85.0,
      "trendDirection": "rising"
    }
  ]
}
```

### Get Ticker Reddit History
```bash
GET /api/v1/reddit/ticker/TSLA/history?days=30

Response:
{
  "data": {
    "tickerSymbol": "TSLA",
    "history": [...],
    "summary": {
      "avgPopularity": 46.0,
      "avgRank": 5.5,
      "bestRank": 3,
      "totalMentions": 340
    }
  }
}
```

### Get Watchlist Heatmap with Reddit Data
```bash
GET /api/v1/watchlists/:id/heatmap

Response:
{
  "tiles": [
    {
      "symbol": "TSLA",
      "reddit_rank": 7,
      "reddit_mentions": 40,
      "reddit_popularity": 44.6,
      "reddit_trend": "falling",
      "reddit_rank_change": -3
    }
  ]
}
```

## Summary

**You already have:**
- ✅ Reddit database tables (reddit_heatmap_daily)
- ✅ 59 records of Reddit data for 20 tickers
- ✅ Reddit API handlers
- ✅ Reddit data collection infrastructure

**What's needed for Phase 3:**
- 🔧 JOIN Reddit data in watchlist queries
- 🔧 Add Reddit metrics to heatmap service calculations
- 🔧 Update frontend UI with Reddit options
- 🔧 Display Reddit data in tooltips

**Result**: Your watchlist heatmaps will show which stocks are trending on Reddit, with full customization of size/color based on Reddit metrics!
