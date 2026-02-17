# InvestorCenter.AI — QA Execution Plan

**Target: Mid-2026 Launch · Competing with YCharts & Koyfin**
**Created: February 2026**

---

## Current State Summary

| Service | Tests | Coverage | Grade |
|---------|-------|----------|-------|
| Frontend (Next.js) | 420 | 71% lib, **0% components** | C |
| Backend (Go) | 794 | 21.8% overall, **4.1% database** | D+ |
| IC Score (Python) | 474 | 55% overall, **0% on 12/19 pipelines** | C+ |
| E2E | 0 | 0% | F |
| CronJob QA | 0 | 0% | F |
| **Total** | **~1,688** | — | **C-** |

**Target State by Launch:**

| Service | Tests | Coverage | Grade |
|---------|-------|----------|-------|
| Frontend | 800+ | 45%+ overall | B |
| Backend | 1,200+ | 40%+ overall | B |
| IC Score | 700+ | 70%+ overall | B+ |
| E2E | 30+ | Critical flows | B |
| CronJob QA | 50+ | All pipelines | B+ |
| **Total** | **~2,800+** | — | **B+** |

---

## Phase 0: Quick Wins (Week 1 — 3 days)

Zero-risk improvements that lock in existing quality and catch regressions immediately.

### 0.1 Raise CI Coverage Gates

Current thresholds are below actual coverage — a pipeline could lose 30% coverage and still pass.

| Service | Current Gate | Actual Coverage | New Gate |
|---------|-------------|-----------------|----------|
| Frontend | 60% | 71% | **65%** |
| Backend | 15% | 21.8% | **20%** |
| IC Score | 45% | 55% | **50%** |

**File:** `.github/workflows/ci.yml`
**Effort:** 15 minutes

### 0.2 Add Dependency Vulnerability Scanning to CI

```yaml
# Add to ci.yml
- name: Python dependency audit
  run: pip-audit --strict

- name: Go vulnerability check
  run: govulncheck ./...

- name: NPM audit
  run: npm audit --audit-level=high
```

**Effort:** 1 hour

### 0.3 Add golangci-lint (Replace bare go vet)

`go vet` catches ~20% of what golangci-lint catches. Add errcheck (unchecked errors), staticcheck (bug patterns), gosec (security), and ineffassign.

**File:** `.golangci.yml` + update CI
**Effort:** 2 hours

### 0.4 Add Prettier for Frontend

No JS/TS/CSS formatter exists. Inconsistent formatting across 124 source files.

```bash
npx prettier --write "app/**/*.{ts,tsx}" "components/**/*.{ts,tsx}" "lib/**/*.{ts,tsx}"
```

**Effort:** 1 hour

---

## Phase 1: Data Accuracy Foundation (Weeks 1–3)

**Why first:** For a financial data platform, wrong numbers are an existential threat. Users will forgive a UI glitch but won't forgive showing the wrong P/E ratio for AAPL. This phase tests the data pipelines that generate every number on the site.

### 1.1 Test TTM Financials Calculator (0% → 80%)

**Risk:** CRITICAL. Known bug documented in CLAUDE.md — SEC quarterly data is cumulative YTD, not standalone. The TTM calculator's "sum 4 quarters" fallback produces incorrect results. This feeds into IC Score, fair value, and valuation ratios.

**Tests to write (~20 tests):**

```
File: ic-score-service/pipelines/tests/test_ttm_financials.py

Core Logic:
- test_ttm_from_four_standalone_quarters (happy path)
- test_ttm_from_annual_report_only (10-K fallback)
- test_q4_derivation_from_fy_minus_q3 (FY - Q3_cumulative = Q4)
- test_cumulative_ytd_detection (flag cumulative quarters)
- test_ttm_eps_with_stock_split_adjustment
- test_ttm_revenue_basic_calculation
- test_ttm_with_missing_quarters (should skip, not produce wrong data)
- test_ttm_with_restated_financials (prefer latest filing)

Edge Cases:
- test_company_with_fiscal_year_not_december (e.g., AAPL Sept FY)
- test_company_with_only_two_quarters_available
- test_negative_eps_to_positive_transition
- test_zero_revenue_quarter (valid for some biotechs)
- test_currency_consistency_across_quarters

Golden File Tests:
- test_aapl_ttm_revenue_known_value (validate against known Apple TTM)
- test_msft_ttm_eps_known_value
- test_bank_ttm_with_nii_revenue_synthesis
```

**Effort:** 4 days
**Dependency:** None

### 1.2 Test Fair Value Calculator (0% → 70%)

**Risk:** CRITICAL. 276 lines of DCF model with zero tests. This is a headline feature — showing intrinsic value estimates.

**Tests to write (~25 tests):**

```
File: ic-score-service/pipelines/tests/test_fair_value.py

DCF Model:
- test_basic_dcf_with_positive_fcf
- test_dcf_with_negative_fcf (no intrinsic value possible)
- test_dcf_discount_rate_sensitivity
- test_dcf_terminal_growth_rate_bounds (must be < discount rate)
- test_dcf_with_zero_growth (stable company)
- test_dcf_with_high_growth_company (30%+ revenue growth)

Valuation Multiples:
- test_pe_based_fair_value
- test_ps_based_fair_value
- test_ev_ebitda_based_fair_value
- test_blended_fair_value_from_multiple_methods

Edge Cases:
- test_company_with_no_fcf_history
- test_company_with_negative_earnings (pre-profit biotech)
- test_reit_fair_value (different model)
- test_bank_fair_value (book value approach)
- test_extreme_growth_rate_capping

Golden File Tests:
- test_aapl_fair_value_reasonable_range ($150-$300 as of 2025)
- test_negative_enterprise_value_handling
```

**Effort:** 5 days
**Dependency:** 1.1 (TTM data feeds fair value)

### 1.3 Test SEC Financials Ingestion (0% → 60%)

**Risk:** HIGH. Source data for everything downstream. Silent parsing errors cascade through all 19 pipelines.

**Tests to write (~15 tests):**

```
File: ic-score-service/pipelines/tests/test_sec_financials_ingestion.py

XBRL Parsing:
- test_parse_standard_10k_filing
- test_parse_standard_10q_filing
- test_parse_bank_filing_with_nii (industry-specific tags)
- test_parse_insurance_filing_with_premiums
- test_parse_reit_filing_with_real_estate_revenue
- test_dedup_multiple_filings_same_quarter

Resume Logic:
- test_resume_from_checkpoint
- test_resume_skips_already_processed

Error Handling:
- test_sec_api_rate_limit_retry
- test_sec_api_404_company_not_found
- test_malformed_xbrl_response
- test_missing_required_fields_in_filing

Data Integrity:
- test_fiscal_quarter_assignment_correctness
- test_period_end_date_validation
- test_no_duplicate_inserts_on_rerun
```

**Effort:** 4 days
**Dependency:** None

### 1.4 Pipeline Output Validation Framework

Create a reusable validation module that each pipeline calls before writing to DB. This catches bad data BEFORE it enters the system.

```
File: ic-score-service/pipelines/utils/data_validator.py

Validators:
- validate_ic_score_range(score) → 1-100, reject if outside
- validate_pe_ratio_range(pe) → warn if >10,000 or <-1,000
- validate_market_cap_positive(cap) → reject negative
- validate_price_not_stale(price, last_updated) → warn if >24h old
- validate_revenue_sign(revenue, sector) → flag negative unless biotech
- validate_ttm_consistency(q1, q2, q3, q4, ttm) → sum ≈ ttm

Integration:
- Call validators in each pipeline before DB write
- Log warnings for soft failures, raise for hard failures
- Track validation failure rates in cronjob_execution_logs
```

**Tests to write (~20 tests):**

```
File: ic-score-service/pipelines/tests/test_data_validator.py

- test_ic_score_within_range (1-100 passes)
- test_ic_score_above_100_rejected
- test_ic_score_below_0_rejected
- test_pe_ratio_extreme_value_warned
- test_negative_market_cap_rejected
- test_stale_price_detection
- test_ttm_consistency_check_passes
- test_ttm_consistency_check_fails_on_mismatch
- test_revenue_negative_flagged_for_non_biotech
- test_revenue_negative_allowed_for_biotech
```

**Effort:** 3 days
**Dependency:** None (can start immediately)

---

## Phase 2: CronJob QA Framework (Weeks 2–4)

**Why now:** You have 26 CronJobs running on staggered schedules with complex dependencies. A failure at 2 AM in `sec_financials` silently cascades to produce wrong IC Scores at midnight — and nobody knows until a user notices. This phase builds the safety net.

### 2.1 Pipeline Integration Test Harness

Create a test harness that runs each pipeline against a test database with known seed data and validates outputs.

```
Directory: ic-score-service/pipelines/tests/integration/

Architecture:
┌──────────────────┐     ┌──────────────────┐     ┌──────────────────┐
│   Seed Known     │ ──▶ │  Run Pipeline    │ ──▶ │ Validate Output  │
│   Test Data      │     │  (Same Code)     │     │ Against Expected │
└──────────────────┘     └──────────────────┘     └──────────────────┘

Seed Data:
- 10 representative companies (AAPL, MSFT, GOOGL, JPM, XOM, PFE, AMT, BRK.B, NVDA, TSLA)
- 2 years of daily prices
- 8 quarters of SEC financials
- Known analyst ratings
- Known insider trades
```

**Test files:**

```
test_pipeline_benchmark_data.py     — Validate SPY benchmark returns
test_pipeline_daily_prices.py       — Validate price ingestion and gap detection
test_pipeline_ttm_financials.py     — Validate TTM calculation with known Q data
test_pipeline_fundamental_metrics.py — Validate metric calculations for seed companies
test_pipeline_technical_indicators.py — Validate RSI/MACD/SMA for known price series
test_pipeline_valuation_ratios.py   — Validate PE/PB/PS with known financials + prices
test_pipeline_risk_metrics.py       — Validate beta/alpha/sharpe for known returns
test_pipeline_sector_percentiles.py — Validate ranking within sector
test_pipeline_ic_score.py           — End-to-end: seed data → IC Score output
```

**Each test follows the pattern:**

```python
@pytest.mark.asyncio
async def test_pipeline_daily_prices(test_db):
    """Run daily_price_update on seed data, validate outputs."""
    # 1. Seed: Insert known price history for AAPL (252 days)
    await seed_price_history(test_db, "AAPL", days=252)

    # 2. Execute: Run pipeline with mocked Polygon API returning known data
    with mock_polygon_response("AAPL", known_prices):
        await run_daily_price_update(test_db, tickers=["AAPL"])

    # 3. Validate: Check database has correct values
    prices = await get_prices(test_db, "AAPL")
    assert len(prices) == 253  # 252 + 1 new day
    assert prices[-1].close == Decimal("185.50")
    assert prices[-1].volume == 45_000_000
```

**Effort:** 8 days
**Dependency:** Phase 1.4 (validator framework)

### 2.2 CronJob Dependency Chain Validation

The dependency chain is the Achilles' heel. If `daily_price_update` (22:30 UTC) runs late, `technical_indicators` (23:00 UTC) runs on stale data. If `ttm_financials` (22:00 UTC) fails, `valuation_ratios` (23:30 UTC) produces wrong numbers.

**Implementation:**

```
File: ic-score-service/pipelines/utils/dependency_checker.py

PIPELINE_DEPENDENCIES = {
    "ic-score-calculator": {
        "requires": [
            "fundamental-metrics",
            "technical-indicators",
            "valuation-ratios",
            "risk-metrics",
            "sector-percentiles",
            "analyst-ratings",
        ],
        "max_staleness_hours": 26,  # Must have run in last 26h
    },
    "technical-indicators": {
        "requires": ["daily-price-update"],
        "max_staleness_hours": 6,
    },
    "valuation-ratios": {
        "requires": ["daily-price-update", "ttm-financials"],
        "max_staleness_hours": 6,
    },
    "risk-metrics": {
        "requires": ["daily-price-update", "benchmark-data"],
        "max_staleness_hours": 26,
    },
    "fundamental-metrics": {
        "requires": ["ttm-financials"],
        "max_staleness_hours": 26,
    },
    "sector-percentiles": {
        "requires": ["fundamental-metrics"],
        "max_staleness_hours": 26,
    },
    "fair-value": {
        "requires": ["fundamental-metrics"],
        "max_staleness_hours": 26,
    },
}

async def check_dependencies(pipeline_name: str, db) -> DependencyResult:
    """Check if all upstream pipelines have run successfully and recently."""
    deps = PIPELINE_DEPENDENCIES.get(pipeline_name, {})
    for dep in deps.get("requires", []):
        last_success = await get_last_success_time(db, dep)
        if last_success is None:
            return DependencyResult(ok=False, reason=f"{dep} has never succeeded")
        hours_ago = (now() - last_success).total_seconds() / 3600
        if hours_ago > deps["max_staleness_hours"]:
            return DependencyResult(
                ok=False,
                reason=f"{dep} last succeeded {hours_ago:.1f}h ago (max: {deps['max_staleness_hours']}h)"
            )
    return DependencyResult(ok=True)
```

**Tests (~12):**

```
File: ic-score-service/pipelines/tests/test_dependency_checker.py

- test_all_deps_met_returns_ok
- test_missing_upstream_blocks_pipeline
- test_stale_upstream_blocks_pipeline
- test_pipeline_with_no_deps_always_ok
- test_ic_score_requires_all_six_upstreams
- test_partial_deps_met_still_fails
- test_staleness_threshold_boundary (exactly at limit)
- test_weekend_staleness_extended (weekend-aware)
```

**Integration:** Add `check_dependencies()` call at the start of each pipeline's `main()`. If deps are stale, log a warning and exit with a specific exit code (e.g., 2 = dependency failure) that the cronjob-monitor can distinguish from runtime errors.

**Effort:** 4 days
**Dependency:** None

### 2.3 Post-Run Data Quality Checks

After each pipeline completes, run automated checks that verify the output looks reasonable.

```
File: ic-score-service/pipelines/utils/post_run_checks.py

Checks per pipeline:

daily_price_update:
  - At least 90% of active tickers have prices from today
  - No price is 0 or negative
  - No price changed >50% in one day (flag for manual review)
  - Volume is non-negative

ttm_financials:
  - TTM revenue ≈ sum of 4 quarters (within 5% tolerance)
  - No TTM EPS is >$1000 or <-$1000 (data error signal)
  - At least 80% of companies with SEC filings have TTM data

ic_score_calculator:
  - All scores are 1-100
  - Score distribution has reasonable standard deviation (15-30)
  - No more than 5% of scores changed by >20 points in one day
  - At least 90% of scored companies have all factor scores

screener_refresh:
  - Row count within 10% of previous refresh
  - No NULL values in required columns (symbol, name, sector)
  - Market cap values are positive
```

**Tests (~15):**

```
File: ic-score-service/pipelines/tests/test_post_run_checks.py

- test_price_coverage_check_passes
- test_price_coverage_check_fails_below_90_percent
- test_price_spike_detection (>50% change flagged)
- test_ttm_consistency_passes
- test_ttm_consistency_fails_on_mismatch
- test_ic_score_range_validation
- test_ic_score_distribution_check
- test_ic_score_volatility_check (>20pt change flagged)
- test_screener_row_count_stability
```

**Effort:** 4 days
**Dependency:** 2.1 (test harness for validation)

### 2.4 Alerting Implementation

The database schema for alerts already exists (migration 015), and the cronjob-monitor already tracks executions. What's missing is the notification sender.

```
File: cronjob-monitor/alerter.py

Implementation:
1. Query cronjob_alerts table for active alert configs
2. Check cronjob_execution_logs for trigger conditions:
   - "failure": consecutive_failures >= threshold
   - "timeout": duration > expected_duration * 1.5
   - "missed_schedule": no execution in 2x the cron interval
   - "performance_degradation": avg duration > p95 of last 7 days
3. Send notifications via configured channels:
   - Slack webhook (SLACK_WEBHOOK_URL env var)
   - Email via SMTP (reuse existing SendGrid config from app-secrets)

Alert Format (Slack):
🔴 CronJob FAILED: ic-score-calculator
- Status: Failed (3 consecutive failures)
- Last success: 2026-02-15 00:00 UTC (48h ago)
- Error: "Connection refused to database"
- Pod: ic-score-calculator-28462848-xxxxx
- Dashboard: https://investorcenter.ai/admin/cronjobs
```

**Run as:** Sidecar process in cronjob-monitor deployment, checking every 5 minutes.

**Effort:** 3 days
**Dependency:** Existing migration 015 tables

---

## Phase 3: Frontend Reliability (Weeks 3–5)

### 3.1 Add Zod Runtime Validation to API Client

The frontend currently trusts all API responses blindly. A backend bug that returns `null` for a price field would crash the UI silently.

```
File: lib/api/schemas.ts

import { z } from 'zod';

export const TickerPriceSchema = z.object({
  symbol: z.string(),
  price: z.number().positive(),
  change: z.number(),
  changePercent: z.number(),
  volume: z.number().int().nonnegative(),
  marketCap: z.number().nullable(),
});

export const ICScoreSchema = z.object({
  ticker: z.string(),
  date: z.string().date(),
  overall_score: z.number().min(1).max(100),
  rating: z.enum(["Strong Buy", "Buy", "Hold", "Sell", "Strong Sell"]),
  value_score: z.number().min(0).max(100).nullable(),
  growth_score: z.number().min(0).max(100).nullable(),
  // ... all factor scores
  confidence_level: z.enum(["Very High", "High", "Medium", "Low", "Very Low"]),
  data_completeness: z.number().min(0).max(1),
});

export const ScreenerRowSchema = z.object({
  symbol: z.string(),
  name: z.string(),
  sector: z.string(),
  market_cap: z.number().nullable(),
  pe_ratio: z.number().nullable(),
  ic_score: z.number().min(1).max(100).nullable(),
  // ... all screener columns
});
```

**Integration:** Validate responses in `lib/api/client.ts`:

```typescript
async get<T>(url: string, schema?: ZodSchema<T>): Promise<T> {
  const data = await this.rawGet(url);
  if (schema) {
    const result = schema.safeParse(data);
    if (!result.success) {
      console.error('API response validation failed:', result.error);
      // In development: throw error
      // In production: log to Sentry and return data anyway
    }
  }
  return data;
}
```

**Tests (~15):**

```
File: lib/api/__tests__/schemas.test.ts

- test_valid_ticker_price_passes
- test_negative_price_rejected
- test_missing_symbol_rejected
- test_ic_score_out_of_range_rejected
- test_invalid_rating_rejected
- test_null_optional_fields_pass
- test_screener_row_valid
```

**Effort:** 3 days
**Dependency:** None

### 3.2 Critical Component Tests

Focus on the components that display financial data — wrong rendering here directly misleads users.

**Priority components (16 tests files):**

```
File: components/__tests__/RealTimePriceHeader.test.tsx
- Renders price with correct decimal places
- Shows green/red for positive/negative change
- Displays "Market Closed" badge outside hours
- Handles undefined price (loading state)
- Updates on polling interval

File: components/__tests__/ICScoreGauge.test.tsx
- Renders score 1-100 with correct color band
- Strong Buy (80-100) = green
- Buy (60-79) = light green
- Hold (40-59) = yellow
- Sell (20-39) = orange
- Strong Sell (1-19) = red
- Handles null/undefined score

File: components/__tests__/FinancialTable.test.tsx
- Renders income statement with correct formatting
- Negative values in parentheses and red
- Large numbers abbreviated (1.5B, 250M)
- Handles missing quarters gracefully
- Sorts by fiscal year

File: components/__tests__/ICScoreScreener.test.tsx
- Renders table with all columns
- Sort by clicking column header
- Filter by sector dropdown
- Pagination next/prev
- Empty state when no results

File: components/__tests__/WatchListTable.test.tsx
- Renders watchlist items with real-time prices
- Add/remove ticker
- Empty watchlist state
```

**Effort:** 6 days
**Dependency:** None

### 3.3 Playwright E2E Tests

Install Playwright and test the 10 most critical user flows. These catch integration bugs between frontend, backend, and database.

```
Directory: e2e/

File: e2e/ticker-page.spec.ts
- Navigate to /ticker/AAPL → page loads, price displays, IC Score shows
- Switch tabs (Overview, Financials, Key Stats, Metrics, Technical, Risk)
- Chart renders with data points
- Real-time price updates within polling interval

File: e2e/screener.spec.ts
- Navigate to /screener → default results load
- Apply sector filter → results narrow
- Apply IC Score filter → results narrow
- Sort by market cap → order changes
- Click ticker → navigates to /ticker/[symbol]
- Pagination works

File: e2e/search.spec.ts
- Type "AAPL" in search bar → suggestions appear
- Click suggestion → navigates to /ticker/AAPL
- Type "Apple" → name search works
- No results for "ZZZZZ"

File: e2e/auth.spec.ts
- Sign up with valid credentials → redirects to home
- Login with valid credentials → JWT stored
- Login with invalid password → error message
- Access /watchlist without auth → redirected to login

File: e2e/watchlist.spec.ts
- Login → navigate to /watchlist → create new watchlist
- Add ticker to watchlist → appears in list
- Delete ticker from watchlist → removed
- Real-time prices update in watchlist view

File: e2e/ic-score.spec.ts
- Navigate to /ic-score → leaderboard loads
- Scores are 1-100 with correct rating text
- Click ticker → view IC Score detail with factor breakdown

File: e2e/crypto.spec.ts
- Navigate to /crypto → prices load
- Click on BTC → /ticker/X:BTCUSD loads
- Real-time prices update (crypto is 24/7)

File: e2e/market-overview.spec.ts
- Homepage shows market indices (S&P 500, NASDAQ, DOW)
- Top movers display with green/red change indicators

File: e2e/sentiment.spec.ts
- Navigate to /sentiment → trending tickers load
- Click ticker → sentiment detail page

File: e2e/mobile-responsive.spec.ts
- All critical pages render correctly at 375px width
- Navigation hamburger menu works
- Tables scroll horizontally on small screens
```

**CI Integration:**

```yaml
# .github/workflows/e2e.yml (separate workflow, runs on PR + main)
e2e-tests:
  runs-on: ubuntu-latest
  services:
    postgres:
      image: postgres:15
  steps:
    - uses: actions/checkout@v4
    - run: npx playwright install --with-deps
    - run: make dev &  # Start backend + frontend
    - run: npx playwright test
    - uses: actions/upload-artifact@v4
      if: failure()
      with:
        name: playwright-traces
        path: e2e/test-results/
```

**Effort:** 8 days
**Dependency:** 3.2 (component tests catch issues before E2E)

---

## Phase 4: Backend Hardening (Weeks 5–7)

### 4.1 Database Layer Tests (4.1% → 35%)

The database layer has 19 untested files (6,437 LOC). These are raw SQL queries that touch every table in the system. A typo in a WHERE clause could return wrong data to every user.

**Priority order by risk:**

```
File: backend/database/integration_test.go (extend existing)

Batch 1 — Financial Data (highest risk):
- TestIntegration_GetFinancials (income/balance/cashflow queries)
- TestIntegration_GetICScorePhase2 (factor scores query)
- TestIntegration_GetICScorePhase3 (final score aggregation)
- TestIntegration_GetSectorPercentiles

Batch 2 — User Data:
- TestIntegration_AlertsCRUD
- TestIntegration_NotificationsCRUD
- TestIntegration_SessionManagement
- TestIntegration_PasswordResetFlow

Batch 3 — Social/Sentiment:
- TestIntegration_SentimentLexicon
- TestIntegration_SocialPostsQuery
- TestIntegration_RedditHeatmapData

Batch 4 — Admin:
- TestIntegration_HeatmapData
- TestIntegration_VolumeData
- TestIntegration_SubscriptionData
```

**Effort:** 6 days
**Dependency:** Existing integration test harness

### 4.2 Handler Tests (18.7% → 40%)

Focus on handlers that serve financial data — these are the API contract with the frontend.

```
File: backend/handlers/crypto_realtime_handlers_test.go
- Test Redis cache hit → return cached price
- Test Redis cache miss → fetch from CoinGecko → cache and return
- Test invalid symbol → 404
- Test Redis unavailable → graceful fallback

File: backend/handlers/keystats_handlers_test.go
- Test GET key stats → returns stored data
- Test POST key stats → validates and stores
- Test DELETE key stats → removes entry
- Test unauthorized access → 401

File: backend/handlers/volume_handlers_test.go
- Test real-time volume (?realtime=true)
- Test database volume (default)
- Test invalid symbol → 404

File: backend/handlers/cronjob_handlers_test.go
- Test overview endpoint returns all jobs
- Test history pagination
- Test metrics aggregation
- Test non-admin access → 403
```

**Effort:** 5 days
**Dependency:** None

### 4.3 Remaining Pipeline Tests

Test the 8 remaining untested pipelines (the ones from Phase 1 covered the 4 highest-risk ones).

```
Priority order:
1. daily_price_update.py         — prices feed everything
2. analyst_ratings_ingestion.py  — IC Score factor input
3. benchmark_data_ingestion.py   — risk metrics depend on this
4. treasury_rates_ingestion.py   — fair value discount rate
5. sec_13f_ingestion.py          — institutional holdings
6. sec_insider_trades_ingestion.py — smart money factor
7. news_sentiment_ingestion.py   — sentiment factor
8. historical_price_backfill.py  — historical data quality
```

**Each pipeline gets ~8-12 tests covering:**
- Successful ingestion with mocked external API
- API error handling (rate limit, timeout, 404)
- Data transformation correctness
- Database upsert idempotency (running twice = same result)
- Edge cases specific to each pipeline

**Effort:** 10 days
**Dependency:** Phase 1.4 (validator framework)

---

## Phase 5: CronJob Production Monitoring (Weeks 6–8)

### 5.1 Implement Slack Alerting

Wire up the existing `cronjob_alerts` table to real Slack notifications.

```
Implementation:
- SLACK_WEBHOOK_URL as Kubernetes secret
- Alert on: failure (3 consecutive), timeout, missed schedule
- Daily summary at 9 AM ET: all job statuses from last 24h
- Weekly report: success rates, average durations, trend comparison

Alert Priority:
🔴 P0 (immediate Slack):
  - ic_score_calculator failed
  - daily_price_update failed
  - postgres_backup failed
  - 3+ consecutive failures on any pipeline

🟡 P1 (batched, every 4 hours):
  - Single pipeline failure
  - Pipeline timeout (exceeded expected duration × 1.5)
  - Coverage dropped below threshold

🔵 P2 (daily digest):
  - Pipeline performance degradation (>p95 duration)
  - Partial data issues (coverage < 90%)
```

**Effort:** 3 days
**Dependency:** Phase 2.4

### 5.2 Data Freshness Dashboard

Create a lightweight data freshness check that runs every 15 minutes and exposes metrics to the admin dashboard.

```
Endpoint: GET /api/v1/admin/data-freshness

Response:
{
  "overall_status": "healthy" | "degraded" | "critical",
  "last_checked": "2026-02-17T12:00:00Z",
  "pipelines": {
    "daily_prices": {
      "last_updated": "2026-02-16T22:45:00Z",
      "freshness_hours": 13.25,
      "threshold_hours": 26,
      "status": "healthy",
      "coverage_percent": 98.5,
      "tickers_updated": 5423,
      "tickers_expected": 5500
    },
    "ic_scores": {
      "last_updated": "2026-02-17T00:30:00Z",
      "freshness_hours": 11.5,
      "threshold_hours": 26,
      "status": "healthy",
      "coverage_percent": 95.2,
      "tickers_scored": 4100,
      "tickers_expected": 4300
    },
    "screener_data": {
      "last_refreshed": "2026-02-16T23:46:00Z",
      "row_count": 9800,
      "previous_row_count": 9795,
      "status": "healthy"
    }
  }
}
```

**Integration with existing admin dashboard frontend.**

**Effort:** 4 days
**Dependency:** Phase 2.3 (post-run checks)

### 5.3 Synthetic Canary Tests

Run lightweight checks every 30 minutes that verify the user-facing data is correct by spot-checking known values.

```
File: ic-score-service/scripts/canary_test.py

Checks:
1. AAPL price is between $100 and $500 (sanity bounds)
2. AAPL IC Score is between 1 and 100
3. Screener returns >5000 results
4. At least 3 market indices have price data
5. S&P 500 index value is between 3000 and 10000
6. BTC price is between $10,000 and $500,000
7. Screener data was refreshed in last 26 hours
8. IC Scores were calculated in last 26 hours

Run as: Kubernetes CronJob every 30 minutes
Alert: Slack P0 if any check fails
```

**Effort:** 2 days
**Dependency:** Phase 5.1 (Slack alerting)

---

## Phase 6: Production Readiness (Weeks 7–8)

### 6.1 Container Security Scanning

```yaml
# Add to CI and deploy workflows
- name: Trivy image scan
  uses: aquasecurity/trivy-action@master
  with:
    image-ref: ${{ env.ECR_REGISTRY }}/investorcenter/backend:latest
    severity: CRITICAL,HIGH
    exit-code: 1  # Fail build on critical vulnerabilities
```

**Effort:** 0.5 days

### 6.2 Circuit Breaker for External APIs

Add circuit breaker pattern to Polygon, CoinGecko, and FMP clients so that API outages don't cascade into timeouts and pod kills.

```go
// backend/services/circuit_breaker.go
type CircuitBreaker struct {
    failureThreshold int      // Open circuit after N failures
    resetTimeout     time.Duration  // Try again after this duration
    state            State    // Closed, Open, HalfOpen
}

// Integration with Polygon client
func (p *PolygonClient) GetPrice(symbol string) (*Price, error) {
    return p.breaker.Execute(func() (interface{}, error) {
        return p.fetchPrice(symbol)
    })
}
```

**Effort:** 3 days

### 6.3 Load Testing

Use k6 to verify the platform handles expected traffic before launch.

```
File: load-tests/screener.js

Target: 100 concurrent users browsing the screener
Duration: 5 minutes sustained
Thresholds:
  - p95 response time < 500ms
  - Error rate < 1%
  - Throughput > 50 req/s

Scenarios:
1. Screener browsing (GET /api/v1/screener/stocks with various filters)
2. Ticker page load (GET /api/v1/tickers/:symbol + /price + /ic-score)
3. Search (GET /api/v1/markets/search?q=...)
4. Watchlist operations (authenticated CRUD)
```

**Effort:** 3 days

---

## CronJob QA Strategy — Detailed Approach

Given the complexity of 26 CronJobs with interdependencies, here is the layered QA strategy:

### Layer 1: Unit Tests (per pipeline)

**What:** Test each pipeline's data transformation logic in isolation with mocked external APIs and mocked database.

**Example:**

```python
# test_daily_price_update.py
async def test_price_transformation():
    """Mock Polygon API response → validate DB insert format."""
    raw_polygon = {"results": [{"t": 1708128000000, "c": 185.50, "v": 45000000}]}
    result = transform_polygon_prices(raw_polygon)
    assert result[0].close == Decimal("185.50")
    assert result[0].volume == 45_000_000
    assert result[0].date == date(2025, 2, 17)
```

**Coverage:** Every pipeline should have 10-20 unit tests.

### Layer 2: Integration Tests (pipeline + real DB)

**What:** Run the pipeline against a real PostgreSQL test database with seed data. Validate that the pipeline reads inputs correctly, transforms them, and writes correct outputs.

**Gating:** Runs in CI with `INTEGRATION_TEST_DB=true` (same pattern as existing DB integration tests).

**Example:**

```python
# test_pipeline_integration_ttm.py
async def test_ttm_calculation_end_to_end(test_db):
    """Seed 4 quarters of SEC data → run TTM calculator → validate TTM values."""
    await seed_quarterly_financials(test_db, "AAPL", [
        {"quarter": "Q1", "revenue": 100_000_000},
        {"quarter": "Q2", "revenue": 110_000_000},
        {"quarter": "Q3", "revenue": 120_000_000},
        {"quarter": "Q4", "revenue": 130_000_000},  # Derived from FY - Q3
    ])
    await run_ttm_calculator(test_db, tickers=["AAPL"])
    ttm = await get_ttm(test_db, "AAPL")
    assert ttm.revenue == 460_000_000  # Sum of 4 quarters
```

### Layer 3: Dependency Chain Tests

**What:** Validate that the execution order is correct and that each pipeline's output is fresh enough for its downstream consumers.

**Example:**

```python
# test_dependency_chain.py
def test_ic_score_dependencies_defined():
    """IC Score calculator must list all 6 upstream pipelines."""
    deps = PIPELINE_DEPENDENCIES["ic-score-calculator"]["requires"]
    assert "fundamental-metrics" in deps
    assert "technical-indicators" in deps
    assert "valuation-ratios" in deps
    assert "risk-metrics" in deps

def test_schedule_respects_dependencies():
    """Verify that each pipeline runs after its dependencies."""
    schedule = parse_all_cronjob_schedules()
    assert schedule["ttm-financials"].hour < schedule["valuation-ratios"].hour
    assert schedule["daily-price-update"].hour < schedule["technical-indicators"].hour
    assert schedule["benchmark-data"].hour < schedule["risk-metrics"].hour
```

### Layer 4: Post-Run Validation (production)

**What:** After each pipeline runs in production, validate the output before downstream pipelines consume it.

**Implementation:** Each pipeline's `main()` calls `run_post_checks()` before logging success. If checks fail, the pipeline exits with code 2 (distinct from runtime error code 1), and the cronjob-monitor logs it as "completed with warnings."

### Layer 5: Synthetic Canary Tests (production)

**What:** Every 30 minutes, spot-check that user-facing data is reasonable. This catches issues that slip through all other layers.

**Implementation:** Lightweight script that queries the same API endpoints users hit and validates the responses against sanity bounds.

### Layer 6: Alerting & Dashboards (production)

**What:** Real-time notifications when anything goes wrong, plus a dashboard showing the health of all 26 CronJobs.

**Implementation:**
- Slack alerts for P0/P1 issues (Phase 5.1)
- Admin dashboard with freshness metrics (Phase 5.2)
- Weekly reliability report sent to team

---

## Timeline Summary

```
Week 1:  Phase 0 (Quick Wins) + Phase 1.4 (Validator) + Phase 1.1 (TTM Tests)
Week 2:  Phase 1.2 (Fair Value) + Phase 1.3 (SEC Ingestion) + Phase 2.2 (Dep Chain)
Week 3:  Phase 2.1 (Pipeline Harness) + Phase 3.1 (Zod Schemas) + Phase 2.4 (Alerting)
Week 4:  Phase 2.3 (Post-Run Checks) + Phase 3.2 (Component Tests)
Week 5:  Phase 3.3 (Playwright E2E)
Week 6:  Phase 4.1 (DB Layer Tests) + Phase 4.2 (Handler Tests) + Phase 5.1 (Slack)
Week 7:  Phase 4.3 (Remaining Pipelines) + Phase 5.2 (Freshness Dashboard)
Week 8:  Phase 5.3 (Canary Tests) + Phase 6 (Security, Circuit Breaker, Load Test)
```

**Total Effort:** ~8 weeks with 1 engineer full-time on QA

**Tests Added:** ~1,100 new tests across all services

**Critical Path:** Phase 1 (Data Accuracy) → Phase 2 (CronJob QA) → Phase 5 (Production Monitoring)
This order ensures the most dangerous gaps are closed first.

---

## Appendix: All 26 CronJobs with QA Status

| # | CronJob | Schedule | Tested | Monitor | Alert |
|---|---------|----------|--------|---------|-------|
| 1 | benchmark-data | 01:00 daily | ❌ | ✅ | ❌ |
| 2 | treasury-rates | 02:00 daily | ❌ | ✅ | ❌ |
| 3 | sec-financials | 02:00 Sun | ❌ | ✅ | ❌ |
| 4 | ticker-sync | 02:00 Sun | ❌ | ✅ | ❌ |
| 5 | 13f-holdings | Quarterly | ❌ | ✅ | ❌ |
| 6 | analyst-ratings | 04:00 daily | ❌ | ✅ | ❌ |
| 7 | fundamental-metrics | 05:00 daily | ✅ 64 tests | ✅ | ❌ |
| 8 | dividend-backfill | 06:00 daily | ❌ | ✅ | ❌ |
| 9 | polygon-ticker-update | 06:30 daily | ❌ | ✅ | ❌ |
| 10 | coverage-monitor | 08:00 daily | ❌ | ✅ | ❌ |
| 11 | insider-trades | Hourly M-F | ❌ | ✅ | ❌ |
| 12 | polygon-volume | 3x daily | ❌ | ✅ | ❌ |
| 13 | ttm-financials | 22:00 M-F | ❌ | ✅ | ❌ |
| 14 | daily-price-update | 22:30 M-F | ❌ | ✅ | ❌ |
| 15 | technical-indicators | 23:00 daily | ✅ 24 tests | ✅ | ❌ |
| 16 | valuation-ratios | 23:30 M-F | ✅ 9 tests | ✅ | ❌ |
| 17 | screener-refresh | 23:45 daily | ❌ | ✅ | ❌ |
| 18 | risk-metrics | 00:00 T-S | ✅ 38 tests | ✅ | ❌ |
| 19 | ic-score-calculator | 00:00 daily | ✅ 46 tests | ✅ | ❌ |
| 20 | sec-filing | 02:00 daily | ❌ | ✅ | ❌ |
| 21 | postgres-backup | 03:00 daily | ❌ | ✅ | ❌ |
| 22 | news-sentiment | Every 4h | ❌ | ✅ | ❌ |
| 23 | reddit-post-collector | Every 4h | ❌ | ✅ | ❌ |
| 24 | reddit-ai-processor | Hourly | ❌ | ✅ | ❌ |
| 25 | reddit-sentiment | Hourly | ❌ | ✅ | ❌ |
| 26 | reddit-heatmap | 02:00 daily | ❌ | ✅ | ❌ |

**After full execution of this plan:**

| # | CronJob | Tested | Monitor | Alert | Post-Check | Canary |
|---|---------|--------|---------|-------|------------|--------|
| All 26 | — | ✅ | ✅ | ✅ | ✅ | ✅ |
