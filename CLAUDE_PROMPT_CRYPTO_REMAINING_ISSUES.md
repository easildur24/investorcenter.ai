# Prompt for Claude Code Web: Fix Remaining Crypto Issues

## Context

I've partially fixed cryptocurrency display issues in InvestorCenter.ai. The following issues remain and need your help to resolve:

### Current State
- ✅ FIXED: Fartcoin showing stock metrics (now hidden)
- ✅ FIXED: Bitcoin chart stuck loading (now returns empty state cleanly)
- ❌ NEEDS FIX: Bitcoin showing wrong price ($266.20 instead of real price ~$90,000+)
- ❌ NEEDS FIX: Crypto charts not implemented (currently showing empty state)

### Architecture Overview
- **Backend**: Go/Gin API at `backend/` (port 8080)
- **Frontend**: Next.js 14 at `app/` and `components/` (port 3000)
- **Crypto Data Source**: Redis cache populated by CronJob using CoinGecko API
- **Crypto Price Endpoint**: `/api/v1/tickers/:symbol` (backend checks Redis first)
- **Chart Endpoint**: `/api/v1/tickers/:symbol/chart?period=1Y`

### Current Branch
All work should be done on branch: `fix/crypto-display-issues`

---

## Issue 1: Bitcoin Price Showing $266.20 (Wrong Price)

### Problem
When visiting `/ticker/BTC`, the price displays as $266.20, which is completely incorrect. Bitcoin hasn't traded at this level since 2015. Current prices should be $30,000-$100,000+ range.

### Root Cause Analysis
From `backend/handlers/ticker_comprehensive.go` lines 68-77:

```go
if isCrypto {
    // For crypto, get price from Redis
    cryptoData, exists := getCryptoFromRedis(symbol)
    if exists {
        priceData = convertCryptoPriceToStockPrice(cryptoData)
        log.Printf("✓ Got crypto price for %s from Redis: $%.2f", symbol, cryptoData.CurrentPrice)
    } else {
        log.Printf("Failed to get crypto price for %s from Redis", symbol)
        priceData = generateMockPrice(symbol, stock)  // LINE 76 - FALLBACK TO MOCK!
    }
    marketStatus = "open" // Crypto markets are always open
    shouldUpdateRealtime = true
}
```

**The backend falls back to mock price generation when Redis doesn't have the data.**

### Investigation Steps You Should Take

1. **Check if Redis has BTC data**:
   ```bash
   kubectl exec -it -n investorcenter deployment/redis -- redis-cli
   GET crypto:quote:BTC
   KEYS crypto:quote:*
   ```

   Expected: Should return JSON with BTC price data like:
   ```json
   {"symbol":"BTC","current_price":89234.56,"price_change_24h":1234.56,...}
   ```

2. **Check if CronJob is running**:
   ```bash
   kubectl get cronjob -n investorcenter
   kubectl get jobs -n investorcenter | grep crypto
   kubectl logs -n investorcenter -l job-name=crypto-price-updater --tail=100
   ```

3. **Find the crypto updater script**:
   Look in `scripts/` directory for files like:
   - `coingecko_crypto_import.py`
   - `crypto_postgres_sync.py`
   - Any script that populates Redis with crypto prices

4. **Check symbol mapping**:
   The issue might be that:
   - Frontend sends "BTC"
   - Backend looks for "crypto:quote:BTC" in Redis
   - But Redis stores it as "crypto:quote:X:BTCUSD" or "crypto:quote:bitcoin"

   Check `backend/handlers/crypto_realtime_handlers.go` for the Redis key format.

### Tasks to Fix

**Task 1A**: Verify Redis crypto data population
- Find which script/CronJob populates Redis with crypto prices
- Ensure it's running on schedule
- Check for errors in CronJob logs
- Verify the script uses correct CoinGecko API

**Task 1B**: Fix symbol mapping if needed
- If Redis uses different symbol format (e.g., "bitcoin" instead of "BTC"), update:
  - `getCryptoFromRedis()` function in `backend/handlers/ticker_comprehensive.go`
  - Or update the script to use consistent symbols

**Task 1C**: Add fallback to CoinGecko API for real-time
If Redis is empty, instead of mock data, fetch directly from CoinGecko API:

```go
if !exists {
    log.Printf("Redis miss for %s, fetching from CoinGecko API", symbol)
    // Add CoinGecko API client call here
    // priceData = fetchFromCoinGecko(symbol)
}
```

**Expected Outcome**:
- BTC shows real current price (e.g., $89,234.56)
- Price updates in real-time from Redis
- No more mock prices for crypto

---

## Issue 2: Crypto Charts Not Implemented

### Problem
Crypto ticker pages show empty chart state with message "Loading Data - Please wait a moment."

From `components/ticker/CryptoTickerHeader.tsx` lines 171-177:

```typescript
{/* Chart placeholder - exact like CMC */}
<div className="mb-8">
  <h2 className="text-xl font-semibold text-gray-900 mb-4">{cryptoName} to USD Chart</h2>
  <div className="bg-gray-50 rounded-lg p-8 text-center">
    <div className="text-gray-500">Loading Data</div>
    <div className="text-gray-400 text-sm mt-1">Please wait a moment.</div>
  </div>
</div>
```

### Current Backend Implementation
From `backend/handlers/ticker_comprehensive.go` lines 199-223:

```go
// Handle crypto chart data separately
if isCrypto {
    log.Printf("⚠️  Chart requested for crypto %s - crypto charts not yet implemented", symbol)
    // For now, return empty chart data with a message
    // TODO: Implement crypto chart data from CoinGecko historical API or Redis time-series
    c.JSON(http.StatusOK, gin.H{
        "success": true,
        "data": gin.H{
            "symbol":      symbol,
            "period":      period,
            "dataPoints":  []models.ChartDataPoint{},
            "count":       0,
            "lastUpdated": time.Now().UTC(),
            "message":     "Crypto chart data coming soon",
        },
        // ... meta ...
    })
    return
}
```

### Chart Data Requirements

Looking at stock chart implementation, crypto charts need the same `ChartDataPoint` format:

```go
type ChartDataPoint struct {
    Timestamp string  `json:"timestamp"`  // ISO 8601 or Unix timestamp
    Price     float64 `json:"price"`
    Volume    float64 `json:"volume,omitempty"`
}
```

### Implementation Options

**Option 1: CoinGecko Historical Price API** (RECOMMENDED)
- Free tier allows 10-30 requests/minute
- Endpoint: `GET /coins/{id}/market_chart?vs_currency=usd&days={days}`
- Returns: Array of `[timestamp_ms, price]` tuples
- Available periods: 1, 7, 14, 30, 90, 180, 365, max

**Option 2: Polygon.io Crypto** (if you have subscription)
- Use Polygon crypto endpoints with "X:" prefix
- Example: `X:BTCUSD` for Bitcoin
- Already have Polygon client in codebase

**Option 3: Store in Redis Time-Series** (complex)
- Store historical data in Redis using time-series module
- Requires CronJob to continuously update
- More infrastructure overhead

### Tasks to Fix

**Task 2A**: Implement CoinGecko historical price fetching

1. **Create new service** at `backend/services/coingecko.go`:

```go
package services

import (
    "encoding/json"
    "fmt"
    "net/http"
    "time"
)

type CoinGeckoClient struct {
    BaseURL string
    Client  *http.Client
}

func NewCoinGeckoClient() *CoinGeckoClient {
    return &CoinGeckoClient{
        BaseURL: "https://api.coingecko.com/api/v3",
        Client:  &http.Client{Timeout: 10 * time.Second},
    }
}

// MapSymbolToCoinGeckoID maps ticker symbols to CoinGecko IDs
func (c *CoinGeckoClient) MapSymbolToCoinGeckoID(symbol string) string {
    symbolMap := map[string]string{
        "BTC":  "bitcoin",
        "ETH":  "ethereum",
        "SOL":  "solana",
        "ADA":  "cardano",
        "FARTCOIN": "fartcoin", // Add your crypto symbols here
        // ... add more
    }

    if id, ok := symbolMap[symbol]; ok {
        return id
    }
    // Default: lowercase symbol
    return strings.ToLower(symbol)
}

// GetMarketChart fetches historical price data
func (c *CoinGeckoClient) GetMarketChart(symbol string, days int) ([]models.ChartDataPoint, error) {
    coinID := c.MapSymbolToCoinGeckoID(symbol)
    url := fmt.Sprintf("%s/coins/%s/market_chart?vs_currency=usd&days=%d", c.BaseURL, coinID, days)

    resp, err := c.Client.Get(url)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    if resp.StatusCode != 200 {
        return nil, fmt.Errorf("CoinGecko API error: %d", resp.StatusCode)
    }

    var result struct {
        Prices [][]float64 `json:"prices"`  // [[timestamp_ms, price], ...]
    }

    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, err
    }

    // Convert to ChartDataPoint format
    dataPoints := make([]models.ChartDataPoint, len(result.Prices))
    for i, priceData := range result.Prices {
        timestamp := time.Unix(int64(priceData[0])/1000, 0)
        dataPoints[i] = models.ChartDataPoint{
            Timestamp: timestamp.Format(time.RFC3339),
            Price:     priceData[1],
        }
    }

    return dataPoints, nil
}
```

2. **Update `GetTickerChart()` in `backend/handlers/ticker_comprehensive.go`**:

Replace lines 199-223 with:

```go
// Handle crypto chart data separately
if isCrypto {
    log.Printf("Fetching crypto chart data for %s", symbol)

    // Use CoinGecko for crypto historical data
    coinGeckoClient := services.NewCoinGeckoClient()

    // Convert period to days
    days := services.GetDaysFromPeriod(period)

    chartData, err := coinGeckoClient.GetMarketChart(symbol, days)
    if err != nil {
        log.Printf("Failed to get crypto chart data for %s: %v", symbol, err)
        // Return empty chart with error message
        c.JSON(http.StatusOK, gin.H{
            "success": true,
            "data": gin.H{
                "symbol":      symbol,
                "period":      period,
                "dataPoints":  []models.ChartDataPoint{},
                "count":       0,
                "lastUpdated": time.Now().UTC(),
                "error":       "Chart data temporarily unavailable",
            },
            "meta": gin.H{
                "symbol":    symbol,
                "period":    period,
                "isCrypto":  true,
                "timestamp": time.Now().UTC(),
            },
        })
        return
    }

    // Return crypto chart data
    c.JSON(http.StatusOK, gin.H{
        "success": true,
        "data": gin.H{
            "symbol":      symbol,
            "period":      period,
            "dataPoints":  chartData,
            "count":       len(chartData),
            "lastUpdated": time.Now().UTC(),
        },
        "meta": gin.H{
            "symbol":    symbol,
            "period":    period,
            "count":     len(chartData),
            "isCrypto":  true,
            "timestamp": time.Now().UTC(),
        },
    })
    return
}
```

**Task 2B**: Update frontend chart component

The `CryptoTickerHeader.tsx` chart placeholder (lines 171-177) should be replaced with an actual chart component. You can either:

1. Reuse the existing `HybridChart` component (used for stocks)
2. Create a new `CryptoChart` component

Example integration in `CryptoTickerHeader.tsx`:

```typescript
import HybridChart from './HybridChart';

// Inside component, fetch chart data
const [chartData, setChartData] = useState(null);

useEffect(() => {
  const fetchChart = async () => {
    const response = await fetch(`/api/v1/tickers/${symbol}/chart?period=1Y`);
    const data = await response.json();
    setChartData(data.data);
  };
  fetchChart();
}, [symbol]);

// Replace placeholder with:
{chartData ? (
  <div className="mb-8">
    <h2 className="text-xl font-semibold text-gray-900 mb-4">{cryptoName} to USD Chart</h2>
    <HybridChart
      symbol={symbol}
      initialData={chartData}
      currentPrice={parseFloat(currentPrice.price)}
    />
  </div>
) : (
  <div className="bg-gray-50 rounded-lg p-8 text-center">
    <div className="text-gray-500">Loading chart...</div>
  </div>
)}
```

**Expected Outcome**:
- Crypto ticker pages display functional price charts
- Charts show historical data from CoinGecko API
- Period selector works (1D, 1W, 1M, 1Y, etc.)
- Chart updates when crypto price changes

---

## Testing Instructions

After implementing fixes, test with these URLs:

1. **BTC (common crypto)**:
   - Visit: `http://localhost:3000/ticker/BTC`
   - Verify: Real price (not $266), working chart

2. **FARTCOIN (crypto in database)**:
   - Visit: `http://localhost:3000/ticker/FARTCOIN`
   - Verify: Real price from Redis, working chart

3. **AAPL (stock - regression test)**:
   - Visit: `http://localhost:3000/ticker/AAPL`
   - Verify: Still works, Polygon chart data

### Backend Logs to Monitor

```bash
# Watch backend logs for:
# - "✓ Got crypto price for BTC from Redis: $XXXXX" (success)
# - NOT "Failed to get crypto price" (indicates Redis miss)
# - "Fetching crypto chart data for BTC" (chart request)
# - NOT "crypto charts not yet implemented" (old message)

make dev
# or
cd backend && go run .
```

---

## Files You'll Need to Modify

1. **New file**: `backend/services/coingecko.go` (create this)
2. **Modify**: `backend/handlers/ticker_comprehensive.go` (lines 199-223)
3. **Modify**: `components/ticker/CryptoTickerHeader.tsx` (lines 171-177)
4. **Investigate**: Redis population script in `scripts/` directory
5. **Check**: Kubernetes CronJob manifests in `k8s/` directory

---

## Additional Context

- CoinGecko free API: https://www.coingecko.com/en/api/documentation
- Symbol mapping may need updates based on what's in your Redis
- The `HybridChart` component already exists and works for stocks, you can reuse it
- Consider adding caching for CoinGecko requests to avoid rate limits

---

## Success Criteria

✅ BTC shows real current price (e.g., $89,234.56), not mock price
✅ Crypto charts display historical price data
✅ Chart period selector works (1D, 1W, 1M, etc.)
✅ Stock pages still work (regression test)
✅ No errors in backend logs
✅ Fast performance (use caching if needed)

---

## Questions to Answer During Implementation

1. What Redis key format is used for crypto? (`crypto:quote:BTC` or `crypto:quote:bitcoin`?)
2. Is the crypto price CronJob running successfully?
3. Should we add CoinGecko API key for higher rate limits?
4. Do we need to add symbol mapping for all cryptos in the database?

Please investigate, implement the fixes, and provide a summary of what you found and changed.
