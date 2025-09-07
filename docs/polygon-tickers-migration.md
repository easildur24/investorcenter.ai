# Polygon.io Tickers API Migration Plan

## Overview
Migrating from multiple exchange-specific ticker sources to Polygon.io's unified `/v3/reference/tickers` API endpoint.

## API Endpoint
- **Base URL**: `https://api.polygon.io/v3/reference/tickers`
- **Authentication**: API key via `apikey` query parameter

## Expected Response Schema

### Response Structure
```json
{
  "status": "OK",
  "count": 10000,  // Total matching tickers
  "next_url": "https://api.polygon.io/v3/reference/tickers?cursor=...",  // Pagination
  "request_id": "...",
  "results": [
    {
      // Core Fields (ALL asset types)
      "ticker": "AAPL",                    // Symbol
      "name": "Apple Inc.",                 // Company/Asset name
      "market": "stocks",                   // stocks, crypto, fx, otc, indices
      "locale": "us",                       // us, global
      "active": true,                       // Currently trading
      "currency_name": "usd",               // Trading currency
      
      // Stock/ETF specific fields
      "type": "CS",                         // CS=Common Stock, ETF, ADRC, etc.
      "primary_exchange": "XNAS",           // Primary exchange code
      "cik": "0000320193",                  // SEC CIK number
      "composite_figi": "BBG000B9XRY4",     // FIGI identifier
      "share_class_figi": "BBG001S5N8V8",   // Share class FIGI
      "last_updated_utc": "2024-01-15",     // Last update date
      
      // Crypto specific fields
      "base_currency_symbol": "BTC",        // For crypto pairs
      "base_currency_name": "Bitcoin",
      "quote_currency_symbol": "USD",
      "quote_currency_name": "US Dollar",
      
      // Index specific fields
      "type": "INDEX",                      // For indices
      
      // Additional metadata
      "delisted_utc": null,                 // Delisting date if applicable
      "phone_number": "(408) 996-1010",     // Company phone
      "address": {
        "address1": "One Apple Park Way",
        "city": "Cupertino",
        "state": "CA",
        "postal_code": "95014"
      },
      "sic_code": "3571",                   // SIC classification
      "sic_description": "Electronic Computers",
      "ticker_root": "AAPL",                // Root ticker for options
      "homepage_url": "https://www.apple.com",
      "total_employees": 164000,
      "list_date": "1980-12-12",            // IPO date
      "branding": {
        "logo_url": "https://...",
        "icon_url": "https://..."
      },
      "market_cap": 2950000000000,          // Market capitalization
      "weighted_shares_outstanding": 15441900000
    }
  ]
}
```

## Asset Type Filters

### 1. US Stocks
```bash
?market=stocks&locale=us&type=CS&active=true
```
- Returns common stocks listed on US exchanges
- Includes NYSE, NASDAQ, AMEX

### 2. ETFs
```bash
?market=stocks&type=ETF&active=true
```
- Returns all Exchange Traded Funds
- Includes sector ETFs, index ETFs, commodity ETFs

### 3. Crypto
```bash
?market=crypto&active=true
```
- Returns cryptocurrency pairs
- Format: "X:BTCUSD" (exchange:pair)

### 4. Indices
```bash
?market=indices&active=true
```
- Returns market indices
- Examples: "I:SPX" (S&P 500), "I:DJI" (Dow Jones)

## Database Schema Mapping

### Current `stocks` Table Mapping
```sql
-- Polygon API -> stocks table
ticker          -> symbol
name            -> name
primary_exchange -> exchange
sic_description -> sector (needs mapping)
locale + market -> country (e.g., "us" + "stocks" = "USA")
currency_name   -> currency
market_cap      -> market_cap
homepage_url    -> website
-- New fields to add:
type            -> asset_type (NEW COLUMN: 'stock', 'etf', 'crypto', 'index')
cik             -> cik (NEW COLUMN)
list_date       -> ipo_date (NEW COLUMN)
```

### Required Database Changes
```sql
-- Add new columns to stocks table
ALTER TABLE stocks ADD COLUMN asset_type VARCHAR(20) DEFAULT 'stock';
ALTER TABLE stocks ADD COLUMN cik VARCHAR(20);
ALTER TABLE stocks ADD COLUMN ipo_date DATE;
ALTER TABLE stocks ADD COLUMN logo_url TEXT;
ALTER TABLE stocks ADD COLUMN primary_exchange_code VARCHAR(10);

-- Create index for asset_type for faster filtering
CREATE INDEX idx_stocks_asset_type ON stocks(asset_type);
```

## Implementation Plan

### Phase 1: Database Schema Update
1. Add new columns to stocks table
2. Create migration script
3. Update Go models

### Phase 2: Polygon Client Enhancement
1. Add new types for Polygon ticker response
2. Create `GetAllTickers()` method with pagination
3. Add filters for different asset types

### Phase 3: Import Script Update
1. Replace exchange-specific fetching with Polygon API
2. Handle pagination (1000 tickers per request)
3. Map Polygon fields to database schema
4. Support for stocks, ETFs, crypto, indices

### Phase 4: Testing & Validation
1. Test with real API key
2. Verify data completeness
3. Compare with existing data

## Sample Go Implementation

```go
// PolygonTickerResponse represents the ticker list response
type PolygonTickersResponse struct {
    Status    string         `json:"status"`
    Count     int           `json:"count"`
    NextURL   string        `json:"next_url"`
    RequestID string        `json:"request_id"`
    Results   []PolygonTicker `json:"results"`
}

type PolygonTicker struct {
    Ticker                   string    `json:"ticker"`
    Name                     string    `json:"name"`
    Market                   string    `json:"market"`
    Locale                   string    `json:"locale"`
    Type                     string    `json:"type"`
    Active                   bool      `json:"active"`
    CurrencyName            string    `json:"currency_name"`
    CIK                     string    `json:"cik"`
    CompositeFigi           string    `json:"composite_figi"`
    ShareClassFigi          string    `json:"share_class_figi"`
    PrimaryExchange         string    `json:"primary_exchange"`
    LastUpdatedUTC          string    `json:"last_updated_utc"`
    DelistedUTC             *string   `json:"delisted_utc"`
    ListDate                string    `json:"list_date"`
    HomepageURL             string    `json:"homepage_url"`
    MarketCap               float64   `json:"market_cap"`
    TotalEmployees          int       `json:"total_employees"`
}

// GetAllTickers fetches all tickers with pagination
func (p *PolygonClient) GetAllTickers(assetType string) ([]PolygonTicker, error) {
    var allTickers []PolygonTicker
    nextURL := ""
    
    for {
        url := p.buildTickersURL(assetType, nextURL)
        resp, err := p.fetchTickers(url)
        if err != nil {
            return nil, err
        }
        
        allTickers = append(allTickers, resp.Results...)
        
        if resp.NextURL == "" {
            break
        }
        nextURL = resp.NextURL
    }
    
    return allTickers, nil
}
```

## Testing Commands

Run the test script with your API key:
```bash
./scripts/test_polygon_tickers.sh YOUR_POLYGON_API_KEY
```

This will show:
1. Sample data for stocks, crypto, ETFs, and indices
2. Total counts for each asset type
3. Response schema validation

## Benefits of Migration

1. **Single API Source**: Consolidate from multiple exchange APIs to one
2. **More Asset Types**: Support crypto, ETFs, and indices
3. **Better Data Quality**: Polygon provides clean, normalized data
4. **Real-time Updates**: Data is updated continuously
5. **Cost Efficiency**: Single API subscription vs multiple sources
6. **Pagination Support**: Handle large datasets efficiently

## Notes

- Polygon API has rate limits (check your subscription tier)
- Free tier is limited; paid tiers offer more requests
- Data is delayed by 15 minutes for free tier
- Real-time data available with paid subscriptions