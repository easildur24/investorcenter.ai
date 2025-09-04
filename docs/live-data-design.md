# Live Stock Data Integration Design

## Overview
Design document for integrating live stock market data (quotes, volume, market cap) into the InvestorCenter.ai platform.

## Requirements
- **Data Points**: Live quotes, trading volume, market capitalization
- **Coverage**: 4,600+ US stocks
- **Latency**: 5-second lag acceptable
- **Cost**: Start with free tier, scalable to paid options
- **Storage**: PostgreSQL with caching layer

## Recommended Data Sources

### Primary: Alpha Vantage (Free Tier)
**Why Alpha Vantage?**
- Generous free tier (25-500 requests/day)
- Batch support (100 symbols per request)
- Official NASDAQ partnership
- Reliable data quality

**Free Tier Limits:**
- 25 API requests/day (some sources report 500/day)
- 5 requests/minute rate limit
- Supports batch requests (100 symbols max)

**Key Endpoints:**
- `GLOBAL_QUOTE`: Real-time price, volume, market cap
- `BATCH_STOCK_QUOTES`: Multiple symbols in one request

### Secondary: Finnhub (Free Tier)
**Why Finnhub?**
- 60 API calls/minute (generous rate limit)
- Real-time US stock data on free tier
- Good for on-demand updates

**Free Tier Limits:**
- 60 API calls/minute
- US market real-time data
- WebSocket available (limited)

## Architecture Design

### System Components
```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   Frontend  │────▶│  Go Backend │────▶│    Redis    │
│  (Next.js)  │     │   (Gin)     │     │   (Cache)   │
└─────────────┘     └─────────────┘     └─────────────┘
                            │                    │
                            ▼                    ▼
                    ┌─────────────┐     ┌─────────────┐
                    │ API Manager │────▶│ PostgreSQL  │
                    └─────────────┘     └─────────────┘
                            │
                    ┌───────┴────────┐
                    ▼                ▼
            ┌─────────────┐  ┌─────────────┐
            │Alpha Vantage│  │   Finnhub   │
            └─────────────┘  └─────────────┘
```

### Data Flow
1. **User Request** → Check Redis cache (5-min TTL)
2. **Cache Miss** → API Manager determines source
3. **API Selection**:
   - High-priority stocks → Finnhub
   - Batch updates → Alpha Vantage
4. **Store** → PostgreSQL + Update Redis cache
5. **Response** → Return cached or fresh data

## Implementation Strategy

### Phase 1: Free Tier Implementation

#### Tiered Update Strategy
**Tier 1: Top 100 Stocks (Most Active)**
- Update frequency: Every 5 minutes
- Source: Finnhub free tier
- Rationale: Most viewed/traded stocks need frequent updates

**Tier 2: Next 500 Stocks (Moderately Active)**
- Update frequency: Every 15 minutes
- Source: Finnhub free tier
- Rationale: Balance between freshness and API limits

**Tier 3: Remaining 4,000+ Stocks**
- Update frequency: Daily batch update
- Source: Alpha Vantage batch requests
- Rationale: Less active stocks, optimize API usage

#### API Usage Calculation
**Alpha Vantage:**
- 4,600 stocks ÷ 100 (batch size) = 46 requests
- Well within 25-500 daily request limit
- Schedule: Run once daily during off-hours

**Finnhub:**
- Top 600 stocks throughout the day
- 60 requests/minute available
- Distribute updates to avoid rate limits

### Phase 2: Scaling Strategy

#### When to Scale
- User base exceeds 1,000 active users
- Request volume exceeds free tier limits
- Need for more frequent updates

#### Scaling Options
1. **Alpha Vantage Premium** ($50-250/month)
   - Unlimited requests
   - Keep existing integration
   - Most cost-effective upgrade

2. **Polygon.io Starter** (~$200/month)
   - Better real-time coverage
   - WebSocket support
   - More comprehensive data

3. **Hybrid Approach**
   - Keep Finnhub free tier
   - Add paid tier for primary source
   - Maximize value per dollar

## Database Schema

### New Tables

```sql
-- Live market data table
CREATE TABLE market_data_live (
    id SERIAL PRIMARY KEY,
    symbol VARCHAR(10) NOT NULL,
    price DECIMAL(10, 2),
    volume BIGINT,
    market_cap BIGINT,
    day_high DECIMAL(10, 2),
    day_low DECIMAL(10, 2),
    prev_close DECIMAL(10, 2),
    change_percent DECIMAL(5, 2),
    last_updated TIMESTAMP NOT NULL,
    data_source VARCHAR(20),
    FOREIGN KEY (symbol) REFERENCES stocks(symbol)
);

-- Create indexes for performance
CREATE INDEX idx_market_data_live_symbol ON market_data_live(symbol);
CREATE INDEX idx_market_data_live_updated ON market_data_live(last_updated);

-- API usage tracking
CREATE TABLE api_usage (
    id SERIAL PRIMARY KEY,
    api_provider VARCHAR(50) NOT NULL,
    endpoint VARCHAR(100),
    request_count INT DEFAULT 1,
    request_date DATE NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### Cache Strategy
**Redis Key Structure:**
```
quote:{symbol}          - Individual stock quote (TTL: 5 min)
volume:{symbol}         - Trading volume (TTL: 5 min)
market_cap:{symbol}     - Market cap (TTL: 5 min)
tier:1:stocks          - List of tier 1 stocks (TTL: 1 day)
tier:2:stocks          - List of tier 2 stocks (TTL: 1 day)
api:limits:{provider}   - API rate limit tracking (TTL: 1 min)
```

## API Endpoints (New)

### GET /api/v1/tickers/:symbol/live
Returns live quote data for a specific symbol
```json
{
  "symbol": "AAPL",
  "price": 150.25,
  "volume": 75000000,
  "marketCap": 2500000000000,
  "dayHigh": 151.50,
  "dayLow": 149.00,
  "prevClose": 149.80,
  "changePercent": 0.30,
  "lastUpdated": "2024-01-15T14:30:00Z",
  "dataSource": "finnhub"
}
```

### GET /api/v1/market/live
Batch endpoint for multiple symbols
```json
{
  "symbols": ["AAPL", "GOOGL", "MSFT"],
  "data": [...]
}
```

## Implementation Checklist

### Backend Tasks
- [ ] Create API client for Alpha Vantage
- [ ] Create API client for Finnhub
- [ ] Implement API Manager with rate limiting
- [ ] Setup Redis caching layer
- [ ] Create database migrations for new tables
- [ ] Implement tiered update scheduler
- [ ] Add new API endpoints
- [ ] Add error handling and retry logic
- [ ] Implement API usage tracking

### Frontend Tasks
- [ ] Create live data components
- [ ] Implement auto-refresh mechanism
- [ ] Add loading states for live data
- [ ] Create market status indicators
- [ ] Add data source attribution

### DevOps Tasks
- [ ] Add Redis to Docker Compose
- [ ] Configure environment variables for API keys
- [ ] Setup monitoring for API usage
- [ ] Create alerts for rate limit approaches
- [ ] Document API key setup process

## Configuration

### Environment Variables
```bash
# API Keys
ALPHA_VANTAGE_API_KEY=your_key_here
FINNHUB_API_KEY=your_key_here

# Redis Configuration
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_DB=0

# Update Intervals (seconds)
TIER1_UPDATE_INTERVAL=300    # 5 minutes
TIER2_UPDATE_INTERVAL=900    # 15 minutes
TIER3_UPDATE_INTERVAL=86400  # 24 hours

# Cache TTL (seconds)
QUOTE_CACHE_TTL=300          # 5 minutes
```

## Monitoring & Alerts

### Key Metrics
- API request count per provider
- Rate limit usage percentage
- Cache hit/miss ratio
- Data freshness by tier
- Failed request count
- Average response time

### Alert Thresholds
- API rate limit > 80% usage
- Failed requests > 5% in 5 minutes
- Cache hit ratio < 70%
- Data staleness > configured interval

## Security Considerations

- Store API keys in environment variables
- Never commit API keys to repository
- Implement request signing if available
- Use HTTPS for all API calls
- Rate limit client requests to prevent abuse
- Monitor for unusual usage patterns

## Testing Strategy

### Unit Tests
- API client mocking
- Cache layer testing
- Rate limiter logic
- Data transformation

### Integration Tests
- End-to-end data flow
- API fallback mechanisms
- Cache invalidation
- Database updates

### Load Testing
- Simulate concurrent user requests
- Test rate limit handling
- Verify cache performance
- Database connection pooling

## Success Metrics

### Phase 1 (Free Tier)
- Successfully update all 4,600 stocks daily
- Maintain < 5 second data freshness for top 100 stocks
- Zero API rate limit violations
- 90%+ cache hit ratio

### Phase 2 (Scaled)
- Support 1,000+ concurrent users
- Maintain < 1 second API response time
- 99.9% uptime for live data service
- Cost per user < $0.50/month

## Timeline

### Week 1
- Implement API clients
- Setup Redis caching
- Create database schema

### Week 2
- Build API Manager with rate limiting
- Implement tiered update logic
- Create new API endpoints

### Week 3
- Frontend integration
- Testing and optimization
- Documentation

### Week 4
- Production deployment
- Monitoring setup
- Performance tuning

## Future Enhancements

1. **WebSocket Support** - Real-time streaming for premium users
2. **Historical Data** - Store and analyze price history
3. **Technical Indicators** - Calculate RSI, MACD, etc.
4. **News Integration** - Correlate price movements with news
5. **Predictive Caching** - ML-based cache warming
6. **Multi-Region** - CDN for global users
7. **Alerts System** - Price/volume alerts for users