# S3 Bucket Structure

## Bucket: investorcenter-raw-data

Region: us-east-1  
Created: February 12, 2026

## Purpose

Storage for raw scraped data from external sources before ETL processing.

## Folder Structure

```
investorcenter-raw-data/
├── README.md                          ← This documentation
│
├── ycharts/                           ← YCharts data
│   ├── key_stats/                     ← Key Stats page (100+ metrics)
│   │   └── {TICKER}/
│   │       └── {YYYY-MM-DD}/
│   │           └── {YYYYMMDDTHHMMSSZ}.json
│   ├── financials/                    ← Income/Balance/Cash Flow (future)
│   ├── valuation/                     ← Valuation ratios (future)
│   └── performance/                   ← Performance metrics (future)
│
├── seekingalpha/                      ← SeekingAlpha data (future)
│   ├── ratings/
│   └── analysis/
│
└── sec_edgar/                         ← SEC filings (future)
    └── {TICKER}/
```

## Example Path

**YCharts Key Stats for NVDA scraped on Feb 12, 2026 at 8:30 PM PST:**

```
s3://investorcenter-raw-data/ycharts/key_stats/NVDA/2026-02-13/20260213T043000Z.json
                             ^^^^^^^ ^^^^^^^^^^ ^^^^ ^^^^^^^^^^^ ^^^^^^^^^^^^^^^^
                             source  data_type  tkr  UTC date    UTC timestamp
```

## File Format

All files are JSON with metadata wrapper:

```json
{
  "ticker": "NVDA",
  "collected_at": "2026-02-12T20:30:00-08:00",
  "source_url": "https://ycharts.com/companies/NVDA/key_stats/stats",
  "uploaded_by": "genesis@investorcenter.ai",
  "uploaded_at": "2026-02-13T04:30:15Z",
  "price": { ... },
  "income_statement": { ... },
  "balance_sheet": { ... },
  "cash_flow": { ... },
  "valuation": { ... },
  "performance": { ... },
  "estimates": { ... },
  "dividends": { ... },
  "risk_metrics": { ... },
  "management_effectiveness": { ... },
  "advanced_metrics": { ... },
  "liquidity_solvency": { ... },
  "employees": { ... }
}
```

## Data Pipeline

```
┌─────────────────┐
│ Scraper         │ (OpenClaw/Nikola)
└────────┬────────┘
         │ POST /ingest/ycharts/key_stats/NVDA
         ↓
┌──────────────────┐
│ Ingestion API    │ (validates + uploads)
└────────┬─────────┘
         │
         ↓
┌──────────────────┐
│ S3 Bucket        │ ← YOU ARE HERE
│ raw data storage │
└────────┬─────────┘
         │
         ↓
┌──────────────────┐
│ ETL Cronjob      │ (reads S3 → writes DB)
└────────┬─────────┘
         │
         ↓
┌──────────────────┐
│ PostgreSQL DB    │ (ycharts_key_stats table)
└────────┬─────────┘
         │
         ↓
┌──────────────────┐
│ Backend API      │ (serves to frontend)
└──────────────────┘
```

## Access Control

**Ingestion Service:**
- IAM Role: `data-ingestion-service-role`
- Permissions: `s3:PutObject`, `s3:GetObject`
- IRSA: Kubernetes ServiceAccount attached

**ETL Cronjobs:**
- IAM Role: `etl-cronjob-role`
- Permissions: `s3:GetObject`, `s3:ListBucket`

**Backend API:**
- No direct S3 access (reads from database)

## Lifecycle Policy (Future)

```
0-30 days:   S3 Standard (hot data)
31-90 days:  S3 Standard-IA (warm data)
91+ days:    S3 Glacier (cold archive)
```

## Cost Estimate

- **Storage:** $0.023/GB/month (S3 Standard)
- **Requests:** $0.005/1000 PUT requests
- **Expected:** ~$5-10/month for 1000 tickers scraped daily

## Monitoring

- CloudWatch metrics: Bucket size, request count
- Alarms: Unexpected upload failures, quota exceeded
- Logs: Data ingestion service logs all S3 operations

## Retention Policy

**Keep forever:**
- Raw data is cheap to store
- Allows reprocessing if ETL logic changes
- Compliance/audit trail
- Historical analysis

## Current Status

✅ Bucket created: `investorcenter-raw-data`  
✅ Folder structure initialized  
✅ Ingestion API configured  
✅ README uploaded to S3  
⏳ First data upload pending (test with NVDA)

---

Last Updated: 2026-02-13 00:54 PST
