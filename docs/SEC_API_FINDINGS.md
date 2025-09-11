# SEC EDGAR API Research Findings

## Successfully Tested Endpoints

### 1. ~~Company Tickers Mapping~~ (Not Needed - Using Polygon CIK)
- **URL**: `https://www.sec.gov/files/company_tickers.json`
- **Purpose**: Maps ticker symbols to CIK (Central Index Key)
- **Status**: **NOT NEEDED** - We already have CIK data from Polygon API in our database
- **Our Approach**: Use `tickers.cik` field populated by Polygon API
- **Example**: AAPL → CIK 0000320193 (already in our DB)

### 2. Company Submissions ✅
- **URL**: `https://data.sec.gov/submissions/CIK{cik}.json`
- **Purpose**: List all filings for a company
- **Data Available**: Recent 1000 filings + historical
- **Key Fields**: filing type, date, accession number, report date
- **Rate Limit**: 10 requests/second

### 3. Company Facts (XBRL) ✅
- **URL**: `https://data.sec.gov/api/xbrl/companyfacts/CIK{cik}.json`
- **Purpose**: Structured financial data from filings
- **Data Available**: 500+ financial metrics
- **Key Metrics Available**:
  - Revenues
  - NetIncomeLoss
  - EarningsPerShare
  - Assets/Liabilities
  - StockholdersEquity
  - Cash and equivalents

### 4. Filing Documents ✅
- **URL Pattern**: `https://www.sec.gov/Archives/edgar/data/{cik}/{accession_clean}/{accession}.txt`
- **Purpose**: Full text of actual filing
- **Format**: SGML/HTML/XBRL embedded
- **Size**: Can be large (10MB+ for 10-K)

## Key Learnings

### CIK Format
- CIK must be 10 digits with leading zeros
- Example: Apple CIK 320193 → 0000320193

### Filing Types Available
- **10-K**: Annual reports (comprehensive)
- **10-Q**: Quarterly reports
- **8-K**: Current reports (material events)
- **DEF 14A**: Proxy statements
- **All forms**: 424B2, S-1, etc.

### API Requirements
1. **User-Agent Header**: Must include email
   ```python
   'User-Agent': 'CompanyName (email@domain.com)'
   ```

2. **Rate Limiting**: 
   - 10 requests per second maximum
   - No authentication required
   - Free to use

3. **Data Formats**:
   - JSON for metadata and structured data
   - TXT/HTML for full documents
   - XBRL embedded in filings

## Implementation Strategy (Updated)

### Phase 1: Basic Infrastructure
1. **CIK Data Source**
   - ✅ **USE EXISTING**: CIK already stored in `tickers.cik` from Polygon API
   - No need to download company_tickers.json
   - Query our database: `SELECT symbol, cik FROM tickers WHERE cik IS NOT NULL`

2. **Filing Metadata Storage**
   - Fetch submissions for each CIK using SEC API
   - **Store in `sec_filings` table**:
     - Basic info: symbol, cik, filing_type (10-K, 10-Q)
     - Dates: filing_date, report_date
     - Identifiers: accession_number (unique SEC ID)
     - URLs: primary_document, filing_detail_url
     - Status: is_processed flag
   - Track processing in `sec_sync_status` table

### Phase 2: Data Collection & Storage
1. **10-K/10-Q Document Processing**
   - Filter for these filing types from submissions
   - Download full filing documents
   - **Store extracted content in `filing_content` table**:
     - Text sections: business_description, risk_factors, md_and_a
     - Financial data: revenue, net_income, earnings_per_share
     - Calculated ratios: gross_margin, return_on_equity, debt_to_equity
     - Full text for search functionality

2. **XBRL Financial Metrics**
   - Use company facts API for structured data
   - **Store in `xbrl_facts` table**:
     - fact_name: "Revenues", "NetIncomeLoss", etc.
     - value: numerical amount
     - fiscal_year & fiscal_period (Q1, Q2, FY)
     - unit: USD, shares, etc.
   - Track year-over-year changes

### Phase 3: Incremental Updates
1. **Daily Sync Process**
   - Query `sec_sync_status` for last sync dates
   - Check for new filings via submissions API
   - Process only new filings (compare accession_number)
   - **Update tracking in `sec_sync_status`**:
     - last_10k_date, last_10q_date
     - last_successful_sync timestamp
     - total_filings_count

2. **Monitoring & Error Handling**
   - Track API usage (10 req/sec limit)
   - Monitor processing failures
   - Store errors in sec_sync_status.last_error
   - Alert on new filings for high-priority stocks

## Data Storage Recommendations (Updated)

### Tables Needed
1. ~~**cik_mapping**~~ → **Not needed, use `tickers.cik` field**
2. **sec_filings**: Filing metadata
3. **filing_content**: Extracted text and metrics
4. **filing_facts**: Structured XBRL data (xbrl_facts table)
5. **sec_sync_status**: Track last successful sync

### Caching Strategy
- ✅ CIK data already cached in `tickers` table (updated by Polygon sync)
- Cache company facts (update daily)
- Store full filing text (never changes)

## Next Steps

1. **Build Filing Fetcher Service**
   - Python service with rate limiting
   - Batch processing capability
   - Error handling and retries

2. **Create Parser**
   - Extract key sections from 10-K/10-Q
   - Parse XBRL for financial data
   - Handle different filing formats

3. **Setup Incremental Updates**
   - Daily cron job
   - Check for new filings
   - Process and store updates

## Sample Implementation Code (Updated)

```python
class SECFilingFetcher:
    def __init__(self):
        self.headers = {
            'User-Agent': 'InvestorCenter (contact@investorcenter.ai)'
        }
        self.rate_limiter = RateLimiter(10)  # 10 req/sec
    
    def get_stocks_with_cik(self):
        # Query our database for stocks with CIK
        cursor.execute("""
            SELECT symbol, cik, name 
            FROM tickers 
            WHERE cik IS NOT NULL 
                AND asset_type IN ('stock', 'CS')
        """)
        return cursor.fetchall()
    
    def get_filings(self, cik, filing_type='10-K'):
        # Fetch filing list using CIK from our DB
        url = f"https://data.sec.gov/submissions/CIK{cik.zfill(10)}.json"
        pass
    
    def get_filing_content(self, cik, accession):
        # Download filing document
        pass
    
    def extract_metrics(self, cik):
        # Get XBRL facts
        pass
```

## Challenges to Address

1. **Large File Sizes**: 10-K can be 10MB+
2. **Parsing Complexity**: Multiple formats (HTML, XBRL, text)
3. **Rate Limiting**: Need careful request management
4. **Data Normalization**: Different companies report differently
5. **Historical Backfill**: Lots of data to process initially

## Success Metrics

- ✅ Successfully connect to all SEC APIs
- ✅ ~~Map ticker symbols to CIKs~~ → Using existing CIK from Polygon data
- ✅ Fetch filing metadata
- ✅ Download filing documents
- ✅ Extract structured XBRL data
- ⏳ Parse filing text sections
- ⏳ Store in database
- ⏳ Setup incremental updates

## Key Simplification

**Original Plan**: Fetch CIK mapping from SEC → Store in database → Use for filings

**New Plan**: Use existing CIK from `tickers` table (populated by Polygon) → Directly fetch filings

This eliminates an entire step and reduces complexity!