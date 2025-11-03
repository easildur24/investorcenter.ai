# SEC Filing Data System - Complete Documentation

## Table of Contents
1. [System Overview](#system-overview)
2. [Architecture](#architecture)
3. [Components](#components)
4. [Data Flow](#data-flow)
5. [Database Schema](#database-schema)
6. [Kubernetes Jobs and Deployments](#kubernetes-jobs-and-deployments)
7. [Current Status](#current-status)
8. [Maintenance and Operations](#maintenance-and-operations)
9. [Troubleshooting](#troubleshooting)

---

## System Overview

The SEC Filing Data System is an automated pipeline that fetches, downloads, stores, and parses SEC filings (10-K, 10-Q, 8-K) for all tracked companies in the InvestorCenter platform. The system provides fundamental financial data to users for investment research.

### Key Capabilities

- **Automatic Filing Collection**: Continuously monitors SEC RSS feeds for new filings
- **Bulk Historical Download**: Fetches historical filings for all companies with CIK numbers
- **Cloud Storage**: Stores actual filing documents (HTML/PDF) in AWS S3
- **Data Parsing**: Extracts structured financial metrics from filings
- **Database Integration**: Links filings to ticker symbols and stores metadata

### Data Sources

- **SEC EDGAR API**: Fetches filing metadata via JSON API (data.sec.gov)
- **SEC RSS Feeds**: Real-time updates for newly published filings
- **Polygon.io**: Provides CIK numbers during ticker data import
- **SEC Archives**: Downloads actual filing documents (www.sec.gov/Archives)

---

## Architecture

```
                    ┌─────────────────────────────────────┐
                    │       SEC.GOV Data Sources          │
                    │  - EDGAR API (data.sec.gov)         │
                    │  - RSS Feeds (ATOM/XML)             │
                    │  - Archives (edgar/data/)           │
                    └──────────────┬──────────────────────┘
                                   │
                    ┌──────────────▼──────────────────────┐
                    │   InvestorCenter SEC Pipeline       │
                    │                                     │
                    │  ┌────────────────────────────┐    │
                    │  │ 1. SEC Filing Fetcher      │    │
                    │  │    (CronJob - Daily 2AM)   │    │
                    │  │    - Fetches metadata      │    │
                    │  │    - Uses CIK from DB      │    │
                    │  └──────────┬─────────────────┘    │
                    │             │                       │
                    │  ┌──────────▼─────────────────┐    │
                    │  │ 2. SEC Filing Downloader   │    │
                    │  │    (Runs after fetcher)    │    │
                    │  │    - Downloads HTML/PDF    │    │
                    │  │    - Uploads to S3         │    │
                    │  └──────────┬─────────────────┘    │
                    │             │                       │
                    │  ┌──────────▼─────────────────┐    │
                    │  │ 3. SEC Filing Parser       │    │
                    │  │    (Future - Not Active)   │    │
                    │  │    - Extracts financials   │    │
                    │  │    - Parses XBRL data      │    │
                    │  └──────────┬─────────────────┘    │
                    │             │                       │
                    │  ┌──────────▼─────────────────┐    │
                    │  │ 4. SEC RSS Updater         │    │
                    │  │    (Deployment - 24/7)     │    │
                    │  │    - Monitors RSS feeds    │    │
                    │  │    - Updates every 10min   │    │
                    │  └────────────────────────────┘    │
                    └──────────────┬──────────────────────┘
                                   │
                    ┌──────────────▼──────────────────────┐
                    │         Data Storage                │
                    │                                     │
                    │  ┌────────────┐  ┌──────────────┐  │
                    │  │ PostgreSQL │  │   AWS S3     │  │
                    │  │  Metadata  │  │  Documents   │  │
                    │  └────────────┘  └──────────────┘  │
                    └─────────────────────────────────────┘
```

---

## Components

### 1. SEC Filing Fetcher (`sec_filing_fetcher.py`)

**Purpose**: Fetches SEC filing metadata using CIK numbers from the database.

**What it does**:
- Reads all tickers with CIK numbers from the `tickers` table
- For each company, calls SEC EDGAR API: `https://data.sec.gov/submissions/CIK{cik}.json`
- Extracts metadata for 10-K, 10-Q, and amended versions (10-K/A, 10-Q/A)
- Saves filing metadata to `sec_filings` table
- Processes up to 100 filings per company
- Updates `sec_sync_status` table with last sync timestamp

**Rate Limiting**: 10 requests/second (100ms between requests) as required by SEC

**API Response Format**:

The SEC EDGAR API returns JSON with company information and recent filings:

```json
{
  "cik": "320193",
  "entityType": "operating",
  "sic": "3571",
  "sicDescription": "Electronic Computers",
  "name": "Apple Inc.",
  "tickers": ["AAPL"],
  "exchanges": ["Nasdaq"],
  "ein": "942404110",
  "description": "...",
  "category": "Large accelerated filer",
  "fiscalYearEnd": "0930",
  "filings": {
    "recent": {
      "accessionNumber": [
        "0000320193-23-000106",
        "0000320193-23-000077",
        "0000320193-23-000064"
      ],
      "filingDate": [
        "2023-11-03",
        "2023-08-04",
        "2023-05-05"
      ],
      "reportDate": [
        "2023-09-30",
        "2023-07-01",
        "2023-04-01"
      ],
      "acceptanceDateTime": [
        "2023-11-03T06:01:36.000Z",
        "2023-08-04T06:01:46.000Z",
        "2023-05-05T06:01:34.000Z"
      ],
      "act": ["34", "34", "34"],
      "form": ["10-K", "10-Q", "10-Q"],
      "fileNumber": ["001-36743", "001-36743", "001-36743"],
      "filmNumber": ["231383632", "231157090", "23933926"],
      "items": ["", "", ""],
      "size": [13221549, 11642555, 11476162],
      "isXBRL": [1, 1, 1],
      "isInlineXBRL": [1, 1, 1],
      "primaryDocument": [
        "aapl-20230930.htm",
        "aapl-20230701.htm",
        "aapl-20230401.htm"
      ],
      "primaryDocDescription": [
        "10-K - Annual report [Section 13 and 15(d)]",
        "10-Q - Quarterly report [Sections 13 or 15(d)]",
        "10-Q - Quarterly report [Sections 13 or 15(d)]"
      ]
    },
    "files": []
  }
}
```

**Key Fields Extracted**:
- `accessionNumber`: Unique filing identifier (e.g., "0000320193-23-000106")
- `filingDate`: Date filed with SEC (e.g., "2023-11-03")
- `reportDate`: Period end date (e.g., "2023-09-30" for fiscal year ending Sept 30)
- `form`: Filing type (e.g., "10-K", "10-Q")
- `primaryDocument`: Filename of main document (e.g., "aapl-20230930.htm")
- `size`: File size in bytes

**Key Features**:
- Skips duplicate filings (checks `accession_number`)
- Tracks companies by market cap (processes larger companies first)
- Records filing date, report date, accession number, primary document filename

**Usage**:
```bash
# Fetch filings for all companies
python scripts/sec_filing_fetcher.py

# Limit to 100 companies (for testing)
python scripts/sec_filing_fetcher.py 100
```

**Database Tables Used**:
- **Reads**: `tickers` (symbol, CIK)
- **Writes**: `sec_filings` (metadata), `sec_sync_status` (sync status)

---

### 2. SEC Filing Downloader (`sec_filing_downloader.py`)

**Purpose**: Downloads actual filing documents (HTML/PDF) from SEC EDGAR and uploads to S3.

**What it does**:
- Finds filings in `sec_filings` where `s3_key IS NULL` (not yet downloaded)
- Constructs EDGAR URLs from CIK, accession number, and primary document
  - Format: `https://www.sec.gov/Archives/edgar/data/{CIK}/{accession_no}/{primary_doc}`
- Downloads both HTML and PDF versions (if PDF exists)
- Uploads to S3 bucket: `investorcenter-sec-filings`
- Updates `sec_filings` with S3 key, file size, download timestamp
- Uses S3 STANDARD_IA storage class for cost savings

**S3 Structure**:
```
filings/
├── AAPL/
│   ├── 10-K/
│   │   ├── 2023/
│   │   │   ├── 2023-09-30_0000320193-23-000106.html
│   │   │   └── 2023-09-30_0000320193-23-000106.pdf
│   │   └── 2022/
│   │       └── ...
│   └── 10-Q/
│       └── ...
└── TSLA/
    └── ...
```

**S3 Features**:
- Versioning enabled
- Lifecycle policy: Move old versions to Glacier after 90 days
- Metadata includes: symbol, filing_type, filing_date, CIK, content hash
- Automatic bucket creation if not exists

**Usage**:
```bash
# Download all pending filings
python scripts/sec_filing_downloader.py

# Download only 50 filings (for testing)
python scripts/sec_filing_downloader.py 50
```

**Database Tables Used**:
- **Reads**: `sec_filings` (WHERE s3_key IS NULL)
- **Writes**: `sec_filings` (s3_key, document_url, download_date, file_size_bytes)

---

### 3. SEC RSS Updater (`sec_rss_updater.py`)

**Purpose**: Continuously monitors SEC RSS feeds for newly published filings and updates the database in real-time.

**What it does**:
- Monitors 6 SEC RSS/Atom feeds:
  - All filings (XBRL): `usgaap.rss.xml`
  - Company filings (latest 100)
  - Latest filings (latest 40)
  - Form 10-K feed
  - Form 10-Q feed
  - Form 8-K feed
- Runs continuously in a Kubernetes deployment
- Updates every 10 minutes
- Parses RSS entries to extract:
  - Company CIK
  - Form type
  - Filing date
  - Accession number
  - Filing URL
- Checks if company is tracked (CIK exists in `tickers` table)
- Saves new filings to `sec_filings`
- Triggers downloader for new filings

**RSS Entry Parsing**:
```
Title format: "10-K - APPLE INC (0000320193) (Filer)"
                ↓          ↓            ↓
            Form Type  Company Name   CIK
```

**Usage**:
```bash
# Run once and exit
python scripts/sec_rss_updater.py

# Run continuously (10-minute intervals)
python scripts/sec_rss_updater.py --continuous

# Run continuously (5-minute intervals)
python scripts/sec_rss_updater.py --continuous --interval 5
```

**Current Status**: ⚠️ Running but experiencing database connection issues (see [Current Status](#current-status))

---

### 4. SEC Filing Parser (`sec_filing_parser.py`)

**Purpose**: Extracts structured financial data from filing documents.

**What it does** (when active):
- Reads filing documents from S3
- Parses HTML to extract specific sections:
  - **10-K Sections**: Business, Risk Factors, MD&A, Financial Statements
  - **10-Q Sections**: Financial Statements, MD&A, Controls
- Extracts financial metrics:
  - Revenue, Net Income, EPS
  - Total Assets, Liabilities, Equity
  - Cash Flow, Free Cash Flow
  - Ratios (margins, ROE, ROA, debt-to-equity)
- Stores extracted data in `filing_content` table
- Uses BeautifulSoup for HTML parsing
- Pattern matching for section identification

**Section Patterns**:
```python
# Example 10-K patterns
'business': r'(?:ITEM\s+1[.\s]+BUSINESS|BUSINESS\s+OVERVIEW)'
'risk_factors': r'(?:ITEM\s+1A[.\s]+RISK\s+FACTORS)'
'mda': r'(?:ITEM\s+7[.\s]+MANAGEMENT[\'"]?S?\s+DISCUSSION)'
```

**Current Status**: 📝 Code exists but not currently active in production

---

## Data Flow

### Daily Batch Flow (CronJob)

```
2:00 AM UTC Daily
       │
       ▼
┌────────────────────┐
│ sec-filing-fetcher │ ──────┐
│  Fetches metadata  │       │
└────────────────────┘       │
       │                     │
       │ Saves to DB         │
       ▼                     │
┌────────────────────┐       │
│  sec_filings table │       │
│  (new records with │       │
│   s3_key = NULL)   │       │
└────────────────────┘       │
       │                     │
       ▼                     │
┌──────────────────────┐     │
│ sec-filing-downloader│ ◄───┘
│  Downloads documents │   (runs in same job)
└──────────────────────┘
       │
       ├──────────────┐
       ▼              ▼
┌──────────┐    ┌─────────┐
│   AWS S3 │    │  Update │
│Documents │    │ s3_key  │
└──────────┘    └─────────┘
```

### Real-time Flow (RSS Updater)

```
Every 10 minutes
       │
       ▼
┌────────────────────┐
│ Check 6 RSS Feeds  │
│  - Parse entries   │
│  - Extract CIK     │
└────────────────────┘
       │
       ▼
┌────────────────────────┐
│ Compare with existing  │
│ accession numbers      │
└────────────────────────┘
       │
       ├─── Already exists ──► Skip
       │
       ▼
┌────────────────────────┐
│ Check if CIK tracked   │
│ (exists in tickers)    │
└────────────────────────┘
       │
       ├─── Not tracked ──► Skip
       │
       ▼
┌────────────────────────┐
│ Save to sec_filings    │
└────────────────────────┘
       │
       ▼
┌────────────────────────┐
│ Trigger download       │
│ (calls downloader)     │
└────────────────────────┘
```

---

## Database Schema

### Tables Overview

1. **`sec_filings`** - Filing metadata and processing status
2. **`filing_content`** - Extracted content and financial metrics (future use)
3. **`xbrl_facts`** - Structured XBRL financial data (future use)
4. **`sec_sync_status`** - Synchronization tracking per company
5. **`cik_mapping`** - Symbol to CIK mapping (legacy, now using `tickers.cik`)

### Primary Table: `sec_filings`

Core table storing all SEC filing metadata:

```sql
CREATE TABLE sec_filings (
    id SERIAL PRIMARY KEY,
    ticker_id INTEGER REFERENCES tickers(id),
    symbol VARCHAR(20) NOT NULL,
    cik VARCHAR(10) NOT NULL,
    filing_type VARCHAR(20) NOT NULL,     -- '10-K', '10-Q', '8-K', etc.
    filing_date DATE NOT NULL,
    report_date DATE,                      -- Period end date
    accession_number VARCHAR(50) UNIQUE NOT NULL,

    -- Document references
    primary_document VARCHAR(255),         -- e.g., 'aapl-10k_20230930.htm'
    primary_doc_description VARCHAR(255),
    filing_url TEXT,                       -- SEC.gov URL

    -- S3 storage (added by downloader)
    s3_key VARCHAR(500),                   -- S3 path to document
    document_url VARCHAR(500),             -- EDGAR direct link
    download_date TIMESTAMP WITH TIME ZONE,
    file_size_bytes BIGINT,

    -- Processing flags
    is_processed BOOLEAN DEFAULT false,
    parsed_at TIMESTAMP WITH TIME ZONE,    -- When parsing completed

    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Key indexes
CREATE INDEX idx_filings_symbol ON sec_filings(symbol);
CREATE INDEX idx_filings_cik ON sec_filings(cik);
CREATE INDEX idx_filings_type ON sec_filings(filing_type);
CREATE INDEX idx_filings_date ON sec_filings(filing_date DESC);
```

**Key Fields**:
- **accession_number**: Unique SEC identifier (e.g., `0000320193-23-000106`)
- **s3_key**: S3 path (e.g., `filings/AAPL/10-K/2023/2023-09-30_0000320193-23-000106.html`)
- **cik**: Central Index Key (10 digits with leading zeros)

### Sync Status Table: `sec_sync_status`

Tracks synchronization status for each company:

```sql
CREATE TABLE sec_sync_status (
    id SERIAL PRIMARY KEY,
    symbol VARCHAR(20) UNIQUE NOT NULL,
    cik VARCHAR(10),

    -- Last sync timestamps
    last_10k_date DATE,
    last_10q_date DATE,
    last_8k_date DATE,
    last_filing_check TIMESTAMP WITH TIME ZONE,
    last_successful_sync TIMESTAMP WITH TIME ZONE,

    -- Statistics
    total_filings_count INTEGER DEFAULT 0,
    processed_filings_count INTEGER DEFAULT 0,
    failed_filings_count INTEGER DEFAULT 0,

    -- Configuration
    sync_enabled BOOLEAN DEFAULT true,
    sync_priority INTEGER DEFAULT 0,      -- Higher = more important

    -- Error tracking
    last_error TEXT,
    error_count INTEGER DEFAULT 0,

    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
```

### Extracted Data Table: `filing_content` (Future)

Stores parsed financial data from filings:

```sql
CREATE TABLE filing_content (
    id SERIAL PRIMARY KEY,
    filing_id INTEGER UNIQUE REFERENCES sec_filings(id) ON DELETE CASCADE,

    -- Extracted sections
    business_description TEXT,
    risk_factors TEXT,
    md_and_a TEXT,                        -- Management Discussion & Analysis
    financial_statements TEXT,

    -- Financial metrics
    revenue DECIMAL(20,2),
    revenue_yoy_change DECIMAL(10,4),
    net_income DECIMAL(20,2),
    earnings_per_share DECIMAL(10,4),
    total_assets DECIMAL(20,2),
    total_liabilities DECIMAL(20,2),

    -- Ratios
    gross_margin DECIMAL(10,4),
    operating_margin DECIMAL(10,4),
    net_margin DECIMAL(10,4),
    return_on_equity DECIMAL(10,4),
    debt_to_equity DECIMAL(10,4),

    -- Full text for search
    full_text TEXT,
    word_count INTEGER,
    extracted_at TIMESTAMP WITH TIME ZONE
);
```

---

## Kubernetes Jobs and Deployments

### 1. Daily CronJob: `sec-filing-daily-update`

**File**: `k8s/sec-filing-cronjob.yaml`

**Schedule**: Daily at 2:00 AM UTC (9:00 PM EST)

**Configuration**:
```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: sec-filing-daily-update
  namespace: investorcenter
spec:
  schedule: "0 2 * * *"  # 2 AM UTC daily
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: sec-filing-fetcher
            image: investorcenter/sec-filing-fetcher:v1
            command: ["python", "sec_filing_fetcher.py", "--days", "1"]

          - name: sec-filing-downloader
            image: investorcenter/sec-filing-downloader:v1
            env:
            - name: S3_BUCKET
              value: "investorcenter-sec-filings"
```

**Resources**:
- Memory: 256Mi request, 512Mi limit
- CPU: 100m request, 500m limit

**Job History**:
- Keeps last 3 successful jobs
- Keeps last 3 failed jobs

### 2. Deployment: `sec-rss-updater`

**File**: `k8s/sec-rss-updater-deployment.yaml`

**Type**: 24/7 running deployment (1 replica)

**Configuration**:
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: sec-rss-updater
  namespace: investorcenter
spec:
  replicas: 1
  template:
    spec:
      containers:
      - name: sec-rss-updater
        image: investorcenter/sec-rss-updater:v2
        env:
        - name: DB_HOST
          value: "postgres-service.investorcenter.svc.cluster.local"
        - name: S3_BUCKET
          value: "investorcenter-sec-filings"
```

**Probes**:
- **Liveness**: Python health check every 60s
- **Readiness**: Database connection check every 30s

**Current Image**: `360358043271.dkr.ecr.us-east-1.amazonaws.com/investorcenter/sec-rss-updater:v2`

### 3. Historical Backfill Jobs (Completed)

Completed one-time jobs for historical data:

- **`sec-filing-backfill`**: Fetched last 52 days of filings (completed 52 days ago)
- **`sec-filing-backfill-all`**: Fetched all historical filings (completed 52 days ago)
- **`sec-filing-download-all`**: Downloaded all documents to S3 (completed 51 days ago)
- **`sec-tables-migration`**: Created database tables (completed 52 days ago)

---

## Current Status

### ✅ Working Components

1. **Daily CronJob**: Successfully scheduled, will run at 2 AM UTC
2. **Database Schema**: All tables created and indexed
3. **S3 Bucket**: `investorcenter-sec-filings` configured with lifecycle policies
4. **Historical Data**: Initial backfill completed (52 days of filings downloaded)

### ⚠️ Issues and Warnings

#### Issue 1: SEC RSS Updater - Database Connection Timeout

**Symptoms**:
```
ERROR - Failed to get existing accession numbers: connection already closed
INFO - Found 0 existing filings in database
ERROR - Failed to get tracked CIKs: connection already closed
INFO - Tracking 0 companies
```

**Root Cause**: Database connection is timing out between update cycles (10-minute intervals). The connection is opened once when the deployment starts but isn't being refreshed.

**Impact**: RSS updater is polling feeds but not saving new filings to database.

**Fix Needed**: Reconnect to database at the start of each update cycle:
```python
def run_continuous(self, interval_minutes: int = 10):
    while True:
        # Reconnect to database each cycle
        if self.conn:
            self.conn.close()
        if not self.connect_db():
            logger.error("Failed to connect to database")
            time.sleep(60)
            continue

        # Process updates...
```

#### Issue 2: No Parsed Financial Data

**Status**: Parser code exists but is not actively running.

**Impact**: Financial metrics are not being extracted from downloaded filings.

**Next Steps**:
1. Activate parser as a Kubernetes job or deployment
2. Process existing downloaded filings
3. Set up automated parsing pipeline

---

## Maintenance and Operations

### Checking System Status

```bash
# Check CronJob status
kubectl get cronjobs -n investorcenter

# Check RSS updater deployment
kubectl get deployments -n investorcenter | grep sec

# View RSS updater logs
kubectl logs -n investorcenter deployment/sec-rss-updater --tail=100

# Check completed jobs
kubectl get jobs -n investorcenter | grep sec

# View CronJob history
kubectl get jobs -n investorcenter --selector=job-name=sec-filing-daily-update
```

### Database Queries

```sql
-- Check total filings count
SELECT COUNT(*) FROM sec_filings;

-- Filings by type
SELECT filing_type, COUNT(*)
FROM sec_filings
GROUP BY filing_type
ORDER BY COUNT(*) DESC;

-- Recent filings
SELECT symbol, filing_type, filing_date, accession_number
FROM sec_filings
ORDER BY filing_date DESC
LIMIT 10;

-- Download status
SELECT
    COUNT(*) FILTER (WHERE s3_key IS NOT NULL) as downloaded,
    COUNT(*) FILTER (WHERE s3_key IS NULL) as pending,
    COUNT(*) as total
FROM sec_filings;

-- Filings by company
SELECT symbol, COUNT(*) as filing_count
FROM sec_filings
GROUP BY symbol
ORDER BY COUNT(*) DESC
LIMIT 20;

-- Sync status
SELECT symbol, last_successful_sync, total_filings_count
FROM sec_sync_status
ORDER BY last_successful_sync DESC NULLS LAST
LIMIT 20;
```

### S3 Bucket Operations

```bash
# List bucket contents
aws s3 ls s3://investorcenter-sec-filings/filings/ --recursive --human-readable

# Get bucket size
aws s3 ls s3://investorcenter-sec-filings/filings/ --recursive --summarize | grep "Total Size"

# Count files
aws s3 ls s3://investorcenter-sec-filings/filings/ --recursive --summarize | grep "Total Objects"

# Check specific company's filings
aws s3 ls s3://investorcenter-sec-filings/filings/AAPL/ --recursive

# Download a specific filing
aws s3 cp s3://investorcenter-sec-filings/filings/AAPL/10-K/2023/2023-09-30_0000320193-23-000106.html .
```

### Manual Operations

```bash
# Run fetcher manually (100 companies)
kubectl run sec-fetcher-manual \
  --image=360358043271.dkr.ecr.us-east-1.amazonaws.com/investorcenter/sec-filing-fetcher:v1 \
  --restart=Never \
  --namespace=investorcenter \
  --env="DB_HOST=postgres-service.investorcenter.svc.cluster.local" \
  --command -- python sec_filing_fetcher.py 100

# Run downloader manually (50 filings)
kubectl run sec-downloader-manual \
  --image=360358043271.dkr.ecr.us-east-1.amazonaws.com/investorcenter/sec-filing-downloader:v1 \
  --restart=Never \
  --namespace=investorcenter \
  --env="S3_BUCKET=investorcenter-sec-filings" \
  --command -- python sec_filing_downloader.py 50
```

---

## Troubleshooting

### Problem: RSS Updater not saving new filings

**Symptoms**: Logs show "Found 0 existing filings", "Tracking 0 companies"

**Solution**: Fix database connection persistence (see Issue #1 above)

**Temporary Workaround**: Restart the deployment to reconnect:
```bash
kubectl rollout restart deployment/sec-rss-updater -n investorcenter
```

### Problem: High S3 storage costs

**Check current storage**:
```bash
aws s3 ls s3://investorcenter-sec-filings --summarize
```

**Solutions**:
1. Verify lifecycle policy is active (moves to Glacier after 90 days)
2. Delete old filing versions if not needed
3. Consider compressing HTML files before upload

### Problem: Missing filings for a company

**Diagnosis**:
```sql
-- Check if company has CIK
SELECT symbol, cik FROM tickers WHERE symbol = 'AAPL';

-- Check sync status
SELECT * FROM sec_sync_status WHERE symbol = 'AAPL';

-- Check filings
SELECT filing_type, filing_date, accession_number
FROM sec_filings
WHERE symbol = 'AAPL'
ORDER BY filing_date DESC;
```

**Solutions**:
1. If no CIK: Run CIK fetcher script to get CIK from Polygon
2. If has CIK but no filings: Check `last_error` in `sec_sync_status`
3. Manually trigger fetcher for that company

### Problem: Download failures

**Check logs**:
```bash
kubectl logs -n investorcenter -l job-name=sec-filing-daily-update -c sec-filing-downloader
```

**Common causes**:
1. SEC rate limiting (429 errors) - Script already handles this with delays
2. Network timeouts - Increase timeout in script
3. Invalid accession numbers - Check data quality in `sec_filings` table
4. S3 permission issues - Verify service account has S3 write permissions

---

## Future Improvements

### High Priority

1. **Fix RSS Updater Connection**: Implement database connection refresh
2. **Activate Parser**: Deploy filing parser to extract financial metrics
3. **Add API Endpoints**: Expose SEC data via REST API for frontend
4. **Implement Search**: Add full-text search across filing content

### Medium Priority

1. **XBRL Processing**: Parse structured XBRL financial data
2. **Email Alerts**: Notify on new 8-K filings (material events)
3. **Filing Comparison**: Compare filings year-over-year
4. **PDF OCR**: Extract text from PDF filings when HTML unavailable

### Low Priority

1. **Proxy Statement (DEF 14A)**: Download and parse proxy statements
2. **Insider Trading Forms**: Track Form 4 (insider trades)
3. **S-1 Analysis**: Monitor new IPO registrations
4. **International Filings**: Support foreign company filings (6-K, 20-F)

---

## SEC API Compliance

### Rate Limits

- **Limit**: 10 requests per second
- **Implementation**: 100ms delay between requests
- **Burst protection**: Single-threaded processing

### User-Agent Requirements

All requests include proper User-Agent:
```
InvestorCenter AI (contact@investorcenter.ai)
```

### Terms of Service

- Data used for research and educational purposes
- Proper attribution to SEC as data source
- No redistribution of raw filing documents
- Compliance with SEC's fair access policy

---

## Related Documentation

- [Polygon Ticker Import](/scripts/README_ycharts.md) - How CIK numbers are obtained
- [Database Migrations](/backend/migrations/006_create_sec_filing_tables.sql) - Schema definitions
- [Watchlist System](/docs/watchlist/) - How SEC data integrates with watchlists
- [Reddit Data](/docs/reddit-list-view-wireframe.md) - Social sentiment + SEC data

---

**Last Updated**: November 2025
**System Version**: v1.0
**Maintained By**: InvestorCenter Engineering Team
