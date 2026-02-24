# YCharts New Data Sources - Schema Documentation

**Date**: 2026-02-24  
**Purpose**: Design schemas for three new YCharts data sources to expand investment research capabilities

---

## Overview

Three new data sources to add to the YCharts scraping pipeline:

1. **Analyst Estimates** - Forward guidance and consensus data
2. **Valuation Ratios** - Historical and current valuation multiples
3. **Performance Metrics** - Returns, risk metrics, and peer comparisons

All three use the same authentication and browser patterns as existing financials and key stats scrapers.

---

## 1. Analyst Estimates

### URL Pattern
```
https://ycharts.com/companies/{TICKER}/estimates
```

### Views Available
- Annual (default)
- Quarterly

Toggle via "Annual" / "Quarterly" buttons at top of page.

### Data Sections

#### Current Period Estimates
Table with columns: Actual, Estimated, Surprise, 2026 Est., 30 Day Chg., YoY Growth

Metrics:
- EPS (non-GAAP)
- EPS GAAP
- Revenue
- EBITDA
- EBITDA Margin

#### Future Periods (2027, 2028, etc.)
For each metric:
- Mean estimate
- Standard deviation
- Number of analysts

#### Price Targets
- Current price
- Mean target
- Low target
- High target
- Standard deviation
- Upside percentage
- Number of estimates

#### Analyst Recommendations
- Buy count
- Outperform count
- Hold count
- Underperform count
- Sell count
- Consensus score (1-5 scale)
- Consensus rating (text: Buy, Outperform, Hold, etc.)
- Total analysts

#### Calendar
- Next announcement date (estimated)
- Last announcement date (actual)

#### Current Multiples (Snapshot)
- PE Ratio
- PE Ratio (Forward)
- PE Ratio (Forward 1y)
- PS Ratio
- PS Ratio (Forward)
- PS Ratio (Forward 1y)
- PEG Ratio

### Schema Location
`data-ingestion-service/schemas/ycharts/analyst_estimates.json`

### S3 Storage Path
```
ycharts/analyst_estimates/{TICKER}/{YYYY-MM-DD}.json
```

### Ingestion Endpoint
```
POST /api/v1/ingest/ycharts/analyst_estimates/{TICKER}
```

### Notes
- Capture both Annual and Quarterly views
- "As of" date shown at bottom of tables (e.g. "As of Feb. 20, 2026")
- Use this as the `as_of_date` field
- Estimates are point-in-time snapshots (should be captured daily/weekly to track revisions)

---

## 2. Valuation Ratios

### URL Pattern
```
https://ycharts.com/companies/{TICKER}/valuation
```

### Data Sections

#### Price Ratios (Current + Historical)
Table columns: TTM, Industry Avg, 3Y Median, 5Y Median, 10Y Median, Fwd 1Y

Metrics:
- PE Ratio
- PS Ratio
- Price to Book Value
- Price to Free Cash Flow
- PEG Ratio

#### Enterprise Ratios (Current + Historical)
Table columns: TTM, Industry Avg, 3Y Median, 5Y Median, 10Y Median

Metrics:
- EV to EBITDA
- EV to EBIT
- EV to Revenues
- EV to Free Cash Flow
- EV to Assets

#### Historical Time Series
Annual data tables (10+ years back to 2016 or earlier):

**Price Ratios by Year**:
- PE Ratio
- PS Ratio
- Price to Book Value
- Price to Free Cash Flow
- PEG Ratio

**Enterprise Ratios by Year**:
- EV to EBITDA
- EV to EBIT
- EV to Revenues
- EV to Free Cash Flow
- EV to Assets

Row format: Latest, 2025, 2024, 2023, 2022, 2021, 2020, 2019, 2018, 2017, 2016

### Schema Location
`data-ingestion-service/schemas/ycharts/valuation.json`

### S3 Storage Path
```
ycharts/valuation/{TICKER}/{YYYY-MM-DD}.json
```

### Ingestion Endpoint
```
POST /api/v1/ingest/ycharts/valuation/{TICKER}
```

### Notes
- "Latest" column = most recent TTM data
- Industry averages are YCharts-calculated based on sector
- Historical data goes back 10+ years for most metrics
- Some ratios may be "--" (null) if data unavailable
- Capture full time series in single request (all years at once)

---

## 3. Performance Metrics

### URL Pattern
```
https://ycharts.com/companies/{TICKER}/performance/price
```

### Data Sections

#### Periodic Total Returns
Time periods: 1M, 3M, 6M, 1Y, 3Y, 5Y, 10Y, 15Y, 20Y, All-Time

Returns are total returns (price + dividends) in percentage format.

#### Annual Total Returns vs Peers
Table with columns for each year (2016-2025, YTD)

Rows:
- Stock
- S&P 500 Total Return (benchmark)
- Peer companies (varies by stock)

Format: Percentage returns for each year

#### Risk Metrics
Available metrics:
- Alpha (3Y)
- Beta (3Y, 5Y)
- Annualized Standard Deviation of Monthly Returns (3Y)
- Historical Sharpe Ratio (3Y)
- Historical Sortino (3Y)
- Max Drawdown (3Y, 5Y)
- Monthly Value at Risk (VaR) 5% (3Y)

### Schema Location
`data-ingestion-service/schemas/ycharts/performance.json`

### S3 Storage Path
```
ycharts/performance/{TICKER}/{YYYY-MM-DD}.json
```

### Ingestion Endpoint
```
POST /api/v1/ingest/ycharts/performance/{TICKER}
```

### Notes
- Performance data is as of market close (check date at bottom of tables)
- Benchmark is typically "S&P 500 Total Return" but may vary
- Peer companies shown are YCharts-selected based on sector/industry
- Some stocks may not have full 20-year history ("--" for unavailable periods)
- Annual returns table shows year-by-year performance (not annualized)
- Periodic returns (3Y, 5Y, etc.) are annualized

---

## Implementation Checklist

For each data source, Claude Code needs to create:

### 1. Backend (Go)
- [ ] Handler in `data-ingestion-service/handlers/ycharts/{source}.go`
- [ ] Route registration in `main.go`
- [ ] Schema validation using the JSON schema
- [ ] S3 upload logic
- [ ] Database record creation (optional, for tracking)

### 2. Task Type (SQL migration)
- [ ] Add new task type to `task_types` table
- [ ] Name: `scrape_ycharts_analyst_estimates`, `scrape_ycharts_valuation`, `scrape_ycharts_performance`
- [ ] Skill path: `scrape-ycharts-analyst-estimates`, etc.
- [ ] Param schema: `{"ticker": "string"}`

### 3. Skill (SKILL.md)
- [ ] Create skill directory: `skills/scrape-ycharts-{source}/`
- [ ] Write `SKILL.md` with:
  - URL pattern
  - Prerequisites (same as financials)
  - Workflow steps (open browser, capture snapshot, extract data, POST to API)
  - Field mapping table (snapshot labels → JSON fields)
  - Parse helper usage
  - Browser cleanup (`browser.stop()`)
- [ ] Include the **CRITICAL** profile note for `profile="openclaw"`

### 4. Testing
- [ ] Manually test skill execution for one ticker (e.g. AAPL)
- [ ] Verify S3 upload
- [ ] Check JSON schema validation
- [ ] Add task to queue and test via task-runner

---

## Parsing Considerations

### Common Patterns

**Percentages**: Convert "7.41%" to decimal 0.0741
```python
parse_percentage("7.41%")  # returns 0.0741
```

**Numbers with B/M suffix**: Already handled by `parse_dollar_amount`
```python
parse_dollar_amount("465.18B")  # returns 465180000000
```

**Floats**: Use `parse_float` for ratios and per-share values
```python
parse_float("33.68")  # returns 33.68
```

**Dates**: Convert "Feb. 20, 2026" to "2026-02-20"
```python
# Manual parsing needed - not in parse_helpers yet
```

**"--" values**: Map to `null`/`None` in JSON

### Table Extraction

All three pages use similar table structures:
1. **Column headers** in first row (periods, metrics, etc.)
2. **Row labels** with clickable links
3. **Cell values** as text

Extract strategy:
1. Capture full snapshot (100k chars max)
2. Parse table structure (find rowgroups, rows, cells)
3. Map row labels to schema field names
4. Extract cell values column by column
5. Apply parse helpers based on data type

### Multiple Tables

**Analyst Estimates**: 5+ tables (Current Period, Future Periods, Price Targets, Recommendations, Calendar)
- Extract each table separately
- Map to different top-level schema objects

**Valuation**: 4 tables (Price Ratios summary, Price historical, Enterprise summary, Enterprise historical)
- Current ratios → `price_ratios` object
- Historical → `historical_price_ratios` array

**Performance**: 2 tables (Periodic Returns, Annual Returns)
- Periodic → `periodic_returns` object
- Annual → `annual_returns` array

---

## Priority Order

**Recommended implementation order**:

1. **Valuation** - Simplest structure (just tables of numbers)
2. **Performance** - Similar to valuation, adds peer comparison
3. **Analyst Estimates** - Most complex (multiple tables, different data types)

---

## Questions for Claude Code

1. Should we create a shared `scrape-ycharts-base` library for common browser/parsing logic?
2. Do we want separate task types or a unified `scrape_ycharts` task with a `data_source` parameter?
3. Should historical time series be stored as arrays in the JSON, or separate S3 files per year?
4. Do we need rate limiting between YCharts requests? (probably yes, but how much?)

---

## File Locations

**Schemas**:
- `data-ingestion-service/schemas/ycharts/analyst_estimates.json`
- `data-ingestion-service/schemas/ycharts/valuation.json`
- `data-ingestion-service/schemas/ycharts/performance.json`

**Skills** (to be created):
- `skills/scrape-ycharts-analyst-estimates/SKILL.md`
- `skills/scrape-ycharts-valuation/SKILL.md`
- `skills/scrape-ycharts-performance/SKILL.md`

**Handlers** (to be created):
- `data-ingestion-service/handlers/ycharts/analyst_estimates.go`
- `data-ingestion-service/handlers/ycharts/valuation.go`
- `data-ingestion-service/handlers/ycharts/performance.go`
