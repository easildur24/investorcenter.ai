# Phase 3 Heatmap Data Availability Analysis

## Current Database State

### Database Findings (as of 2025-11-02)
- **Tickers table**: 20 tickers (mostly test data)
- **Market cap in DB**: ❌ All NULL values
- **Volume data in DB**: ❌ Migration `005_add_volume_fields.sql` NOT applied yet
- **Historical prices in stock_prices table**: ❌ 0 rows
- **Average volume**: ❌ Not calculated or stored

### Database Schema Available
- `stocks.market_cap` column EXISTS but unpopulated
- Volume migration ready but NOT applied (would add: `volume`, `avg_volume_30d`, `avg_volume_90d`, `vwap`, etc.)

## Data Sources: Polygon.io API

### ✅ Available via Real-time API

All heatmap metrics CAN be fetched from Polygon.io API:

#### 1. **Size Metrics**
| Metric | API Source | Implementation |
|--------|------------|----------------|
| **Market Cap** | Ticker Details API + Snapshot | `GET /v3/reference/tickers/{symbol}` returns `weighted_shares_outstanding`<br>Market Cap = current_price × shares_outstanding |
| **Volume** | Snapshot API | `GET /v2/snapshot/locale/us/markets/stocks/tickers/{symbol}`<br>Returns `ticker.day.v` (today's volume) |
| **Avg Volume** | Aggregates API | `GET /v2/aggs/ticker/{symbol}/range/1/day/{from}/{to}`<br>Calculate average from last 30/90 days |

#### 2. **Color Metrics**
| Metric | API Source | Implementation |
|--------|------------|----------------|
| **Price Change %** | Snapshot API | `ticker.todaysChangePerc` (for 1D)<br>Historical aggregates for other periods |
| **Volume Change %** | Aggregates API | Compare today's volume vs average volume |

#### 3. **Time Periods**
| Period | API Endpoint | Data Points |
|--------|--------------|-------------|
| **1D** | Snapshot API | Real-time snapshot includes today's data |
| **1W** | Aggregates (7 days) | `range/1/day/{from}/{to}` |
| **1M** | Aggregates (30 days) | `range/1/day/{from}/{to}` |
| **3M** | Aggregates (90 days) | `range/1/day/{from}/{to}` |
| **6M** | Aggregates (180 days) | `range/1/day/{from}/{to}` |
| **YTD** | Aggregates (since Jan 1) | `range/1/day/{from}/{to}` |
| **1Y** | Aggregates (365 days) | `range/1/day/{from}/{to}` |
| **5Y** | Aggregates (weekly/monthly) | `range/1/week/{from}/{to}` |

### Existing Go Code (Already Implemented)

**File**: `backend/services/polygon.go`

```go
// ✅ Already implemented - Real-time stock data
func (p *PolygonClient) GetStockRealTimePrice(symbol string) (*models.StockPrice, error)
// Returns: Price, Volume, Change%, Open, High, Low

// ✅ Already implemented - Historical data
func (p *PolygonClient) GetHistoricalData(symbol string, timespan string, from string, to string) ([]models.ChartDataPoint, error)
// Returns: OHLCV data for any time period

// ✅ Already implemented - Ticker details
func (p *PolygonClient) GetTickerDetails(symbol string) (*TickerDetailsResponse, error)
// Returns: Company info, shares outstanding (for market cap calculation)

// ✅ Already implemented - Bulk snapshots (efficient for heatmaps!)
func (p *PolygonClient) GetBulkStockSnapshots() (*BulkStockSnapshotResponse, error)
// Returns: ALL US stocks in ONE API call
```

## Implementation Strategy for Phase 3

### Option A: Real-time Only (Recommended for MVP)

**Pros**:
- No database changes needed
- Always fresh data
- Leverages existing Polygon API code

**Cons**:
- API rate limits (5 calls/min on free tier)
- Slower initial load for large watchlists
- Polygon API costs at scale

**Implementation**:
```go
// For a watchlist with 20 stocks:
// 1. Fetch bulk snapshot (1 API call for ALL stocks)
// 2. Calculate market cap: price × shares_outstanding
// 3. Calculate metrics for selected time period
// 4. Return heatmap data structure
```

### Option B: Hybrid (Database Cache + Real-time)

**Pros**:
- Fast lookups for static data (market cap, avg volume)
- Reduced API calls
- Better performance at scale

**Cons**:
- Need to run volume migration
- Need data population scripts
- Need cache invalidation strategy

**Implementation**:
1. Apply migration `005_add_volume_fields.sql`
2. Run daily CronJob to update market cap and average volumes
3. Fetch only real-time price changes from API

### Option C: Pre-calculate Everything (Future Enhancement)

Store all historical data in PostgreSQL `stock_prices` table for instant queries.

## Recommended Approach for Phase 3

### Phase 3A (Initial Implementation)
**Use Real-time API Only**

```go
// New service method in backend/services/heatmap_service.go
func (hs *HeatmapService) GenerateHeatmapData(watchListID uuid.UUID, config HeatmapConfig) (*HeatmapData, error) {
    // 1. Get symbols from watch list
    symbols := hs.getWatchListSymbols(watchListID)

    // 2. Fetch bulk snapshot (1 API call for all stocks)
    polygonClient := NewPolygonClient()
    bulkData, err := polygonClient.GetBulkStockSnapshots()

    // 3. For each symbol in watchlist:
    tiles := []HeatmapTile{}
    for _, symbol := range symbols {
        snapshot := findInBulkData(bulkData, symbol)

        // Calculate size metric
        size := hs.calculateSizeValue(symbol, config.SizeMetric, snapshot)

        // Calculate color metric based on time period
        colorValue := hs.calculateColorValue(symbol, config.ColorMetric, config.TimePeriod, snapshot)

        tiles = append(tiles, HeatmapTile{
            Symbol: symbol,
            Name: snapshot.Name,
            Size: size,
            ColorValue: colorValue,
        })
    }

    return &HeatmapData{Tiles: tiles}, nil
}
```

### Phase 3B (Performance Optimization)
1. Apply volume migration
2. Add daily CronJob to populate market cap and avg volume
3. Use database for static metrics, API for dynamic data

## API Rate Limit Considerations

### Polygon.io Free Tier Limits
- 5 API calls per minute
- 100 API calls per day (for historical data)

### Efficient API Usage
```
Bulk Snapshot Strategy (Recommended):
- 1 API call = ALL ~4,600 US stocks
- Perfect for watchlists up to 50 stocks
- No rate limit issues

Individual Snapshot Strategy (Fallback):
- 1 API call per stock
- Only use if bulk fails
- Watch rate limits for large watchlists
```

## Data Freshness

| Data Type | Update Frequency | Source |
|-----------|------------------|--------|
| **Current Price** | Real-time (seconds) | Polygon Snapshot API |
| **Volume** | Real-time (seconds) | Polygon Snapshot API |
| **Market Cap** | Daily (calculated) | Price × Shares Outstanding |
| **Price Change %** | Real-time | Polygon Snapshot API |
| **Historical Data** | On-demand | Polygon Aggregates API (cached) |

## Conclusion

### ✅ YES - We Have All Required Data

**Summary**:
- All Phase 3 heatmap features are **fully supported**
- Data is available via **Polygon.io API** (already integrated)
- **No database changes required** for initial implementation
- Existing Go code in `polygon.go` provides all necessary methods
- Can implement Phase 3 heatmap **immediately** using real-time API approach

**Next Steps**:
1. Implement heatmap service using bulk snapshot API
2. Add caching layer for improved performance
3. (Optional) Apply volume migration and populate database for future optimization

The architecture is already in place - we just need to connect the dots!
