# IC Score Ticker Coverage Analysis

**Date:** 2025-11-22
**Status:** Active Issue - 787 stocks missing financial data

---

## Executive Summary

The IC Score database has **80.2% coverage** of stocks with financial data. **787 stocks** (19.8%) have valid CIK numbers but are missing SEC financial data, preventing IC Score calculation.

###Quick Stats:
- **Total stocks in IC Score DB**: 4,083
- **Stocks with CIK**: 3,979 (97.5%)
- **Stocks with SEC financials**: 3,192 (80.2% of stocks with CIK)
- **Coverage gap**: 787 stocks

---

## Coverage Breakdown

| Data Type | Count | Coverage | Notes |
|-----------|-------|----------|-------|
| Total Stocks | 4,083 | 100% | IC Score database inventory |
| Stocks with CIK | 3,979 | 97.5% | Can retrieve SEC data |
| SEC Financials | 3,192 | 80.2% | Raw quarterly data |
| TTM Financials | 3,192 | 80.2% | Trailing 12-month data |
| Valuation Ratios | 3,179 | 79.9% | Price-based ratios |
| IC Scores | 4,081 | 102.6% | *Includes some non-CIK stocks |

---

## Root Causes

### 1. Foreign ADRs Not Supported (787 stocks)

**Problem**: The SEC financials pipeline only processes domestic US companies that file 10-K/10-Q forms. Foreign companies trading as American Depositary Receipts (ADRs) file different forms (20-F, 6-K) and are currently skipped.

**Affected Stocks (Sample)**:
- ASML (ASML Holding N.V.)
- AZN (AstraZeneca PLC)
- PDD (PDD Holdings Inc.)
- ARM (ARM Holdings PLC)
- SNY (Sanofi)
- NTES (NetEase, Inc.)
- TRI (Thomson Reuters Corp)
- ARGX (argenx SE)
- TCOM (Trip.com Group Limited)
- FER (Ferrovial SE)
- **ORCL was not in the sample - need to verify if it's in stocks table**

**Impact**: ~20% of stocks in the database cannot have IC Scores calculated.

### 2. Stocks Not in IC Score Database

**Problem**: Some stocks (like ORCL if confirmed missing) may not be in the IC Score `stocks` table at all, even though they exist in the main InvestorCenter database.

**Root Cause**: The IC Score database appears to have been seeded with a specific subset of stocks, possibly from an initial data load that excluded certain tickers.

---

## Monitoring & Detection

### Manual Coverage Check

Run this command to check current coverage:

```bash
kubectl exec -n investorcenter deploy/ic-score-api -- python3 -c "
import sys; sys.path.insert(0, '/app')
import asyncio
from sqlalchemy import text
from database.database import get_database

async def check():
    db = get_database()
    async with db.session() as session:
        total = (await session.execute(text('SELECT COUNT(*) FROM stocks WHERE cik IS NOT NULL'))).scalar()
        with_fin = (await session.execute(text('SELECT COUNT(DISTINCT ticker) FROM financials'))).scalar()
        print(f'Coverage: {with_fin}/{total} ({with_fin/total*100:.1f}%)')
        print(f'Missing: {total - with_fin} stocks')
asyncio.run(check())
"
```

### Recommended Monitoring

1. **Weekly Coverage Report**: Add to CronJob schedule
   - Check total stocks, CIK count, financials count
   - Alert if coverage drops below 75%
   - Track coverage trend over time

2. **Failed Ingestion Log**: Monitor SEC financials pipeline logs
   - Count stocks processed vs. skipped
   - Categorize skip reasons (no CIK, foreign ADR, API error)

3. **Admin Dashboard Widget**: Add coverage metrics to admin UI
   - Display current coverage percentage
   - Show count of stocks in each pipeline stage
   - List recently failed/skipped tickers

---

## Recommendations

### Short Term (Immediate)

1. **Document the ADR limitation** in user-facing documentation
2. **Add coverage metrics** to the Database Stats tab in admin dashboard
3. **Create alert** if coverage drops below 75%

### Medium Term (1-2 weeks)

1. **Add ADR support** to SEC financials pipeline
   - Implement 20-F/6-K form parsers
   - Handle different financial reporting standards (IFRS vs. US GAAP)
   - Test with sample ADRs (ASML, AZN, ARM)

2. **Sync missing stocks** from main DB to IC Score DB
   - Identify stocks in main DB but not in IC Score DB
   - Create sync script to add missing tickers with CIK numbers
   - Run as one-time backfill, then periodic sync

3. **Add monitoring dashboard**
   - Coverage percentage over time
   - Pipeline success rates
   - List of stocks without data + reason

### Long Term (1 month+)

1. **Automated ticker sync** between main DB and IC Score DB
   - Trigger on new ticker additions to main DB
   - Validate CIK exists before adding to IC Score
   - Auto-run SEC financials ingestion for new tickers

2. **Alternative data sources** for foreign companies
   - Integrate with international financial APIs
   - Support multiple financial reporting standards
   - Handle currency conversions

3. **Coverage SLOs** (Service Level Objectives)
   - Target: 95% coverage of stocks with valid CIK
   - Alert threshold: <90% coverage
   - Auto-retry failed ingestions

---

## How to Check Specific Ticker

To check if a specific ticker (e.g., ORCL) is missing:

```bash
kubectl exec -n investorcenter deploy/ic-score-api -- python3 -c "
import sys; sys.path.insert(0, '/app')
import asyncio
from sqlalchemy import text
from database.database import get_database

async def check_ticker(ticker):
    db = get_database()
    async with db.session() as session:
        stock = await session.execute(text('SELECT ticker, name, cik FROM stocks WHERE ticker = :t'), {'t': ticker})
        row = stock.fetchone()

        if not row:
            print(f'❌ {ticker} NOT in stocks table')
            return

        print(f'✓ {ticker} in stocks: {row[1]} (CIK: {row[2]})')

        fin = await session.execute(text('SELECT COUNT(*) FROM financials WHERE ticker = :t'), {'t': ticker})
        fin_count = fin.scalar()
        print(f'  SEC Financials: {fin_count} records')

        if fin_count == 0 and row[2]:
            print(f'  ⚠️ Has CIK but NO financials - likely foreign ADR or ingestion failed')

asyncio.run(check_ticker('ORCL'))
"
```

---

## Files & Components

### Data Pipeline
- `ic-score-service/pipelines/sec_financials_ingestion.py` - SEC financials ingestion
- `ic-score-service/pipelines/ttm_calculator.py` - TTM calculator
- `ic-score-service/pipelines/valuation_ratios_calculator.py` - Valuation ratios
- `ic-score-service/pipelines/ic_score_calculator.py` - IC Score calculation

### Database
- IC Score DB: Separate PostgreSQL database for IC Score data
- Main DB: InvestorCenter main database (different from IC Score)
- Tables: `stocks`, `financials`, `ttm_financials`, `valuation_ratios`, `ic_scores`

### Admin Interface
- `app/admin/dashboard/page.tsx` - Admin dashboard
- `backend/handlers/admin_handlers.go` - Admin API (connects to main DB, NOT IC Score DB)

---

## Action Items

- [ ] Verify if ORCL is in IC Score stocks table
- [ ] Add coverage metrics to admin dashboard
- [ ] Document ADR limitation in user docs
- [ ] Create script to sync missing tickers from main DB
- [ ] Implement ADR support (20-F/6-K forms)
- [ ] Add automated monitoring and alerts
- [ ] Set up weekly coverage reports

---

## Contact

For questions or updates on this issue, check the GitHub issues or contact the team.
