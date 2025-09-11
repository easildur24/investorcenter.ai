# SEC EDGAR API Research Findings

## Successfully Tested Endpoints

### 1. Company Tickers Mapping ✅
- **URL**: `https://www.sec.gov/files/company_tickers.json`
- **Purpose**: Maps ticker symbols to CIK (Central Index Key)
- **Data Available**: 10,125+ companies
- **Key Fields**: ticker, CIK, company name
- **Example**: AAPL → CIK 0000320193

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

## Implementation Strategy

### Phase 1: Basic Infrastructure
1. **CIK Mapping**
   - Download company_tickers.json
   - Map our stock symbols to CIKs
   - Store in database for quick lookup

2. **Filing Metadata**
   - Fetch submissions for each CIK
   - Store filing dates, types, accession numbers
   - Track which filings we've processed

### Phase 2: Data Collection
1. **10-K/10-Q Focus**
   - Filter for these filing types
   - Download and parse documents
   - Extract key sections

2. **Financial Metrics**
   - Use company facts API for structured data
   - Store standardized metrics
   - Track year-over-year changes

### Phase 3: Incremental Updates
1. **Daily Checks**
   - Check for new filings via submissions API
   - Process only new filings since last check
   - Update database with new data

2. **Monitoring**
   - Track API usage
   - Monitor for failures
   - Alert on new filings for watched stocks

## Data Storage Recommendations

### Tables Needed
1. **cik_mapping**: symbol → CIK
2. **sec_filings**: Filing metadata
3. **filing_content**: Extracted text and metrics
4. **filing_facts**: Structured XBRL data
5. **sync_status**: Track last successful sync

### Caching Strategy
- Cache CIK mappings (changes rarely)
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

## Sample Implementation Code

```python
class SECFilingFetcher:
    def __init__(self):
        self.headers = {
            'User-Agent': 'InvestorCenter (contact@investorcenter.ai)'
        }
        self.rate_limiter = RateLimiter(10)  # 10 req/sec
    
    def get_cik(self, symbol):
        # Map symbol to CIK
        pass
    
    def get_filings(self, cik, filing_type='10-K'):
        # Fetch filing list
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
- ✅ Map ticker symbols to CIKs
- ✅ Fetch filing metadata
- ✅ Download filing documents
- ✅ Extract structured XBRL data
- ⏳ Parse filing text sections
- ⏳ Store in database
- ⏳ Setup incremental updates