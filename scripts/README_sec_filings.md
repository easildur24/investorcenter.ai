# SEC Filings Downloader

A Python script to download SEC filings for any ticker using the SEC's EDGAR Full-Text Search API.

## Features

- Download SEC filings for any ticker (HIMS included)
- Filter by specific form types (10-K, 10-Q, 8-K, etc.)
- Download metadata only or full documents
- Automatic CIK lookup for known tickers
- Rate limiting to respect SEC servers
- Export to CSV and JSON formats
- Command-line interface

## Installation

The script uses dependencies already included in the project's `requirements.txt`:
- `requests` - for HTTP requests
- `pandas` - for data manipulation and CSV export

## Usage

### Basic Examples

```bash
# Download 10 most recent filings for HIMS (metadata only)
python scripts/sec_filings_downloader.py --ticker HIMS

# Download 5 most recent 10-K filings for HIMS
python scripts/sec_filings_downloader.py --ticker HIMS --form-type 10-K --count 5

# Download all form types (20 filings)
python scripts/sec_filings_downloader.py --ticker HIMS --all-forms --count 20

# Download actual filing documents (not just metadata)
python scripts/sec_filings_downloader.py --ticker HIMS --download-documents --count 3
```

### Command Line Options

- `--ticker`: Stock ticker symbol (required)
- `--form-type`: Specific form type (10-K, 10-Q, 8-K, etc.)
- `--all-forms`: Download all form types
- `--count`: Number of filings to download (default: 10)
- `--download-documents`: Download actual filing documents
- `--output-dir`: Output directory (default: sec_filings)
- `--user-agent`: Custom User-Agent string

### Output Structure

```
sec_filings/
└── HIMS/
    ├── HIMS_filings_metadata.csv    # Structured filing data
    ├── HIMS_filings_raw.json        # Raw API response
    └── documents/                    # Actual filing documents (if --download-documents used)
        ├── 0001773751-23-000123_hims-10k_20231231.htm
        └── ...
```

## SEC Filing Form Types

Common SEC filing types you might want to download:

- **10-K**: Annual report
- **10-Q**: Quarterly report
- **8-K**: Current report (material events)
- **DEF 14A**: Proxy statement
- **S-1**: Registration statement
- **S-3**: Registration statement (shelf registration)

## HIMS Specific Information

- **Company**: Hims & Hers Health, Inc.
- **CIK**: 0001773751
- **Ticker**: HIMS

## Rate Limiting

The script includes automatic rate limiting (0.1 seconds between requests) to be respectful to SEC servers. The SEC recommends:

1. Including a proper User-Agent header with contact information
2. Not making more than 10 requests per second
3. Using the API responsibly

## Error Handling

The script handles common errors:
- Network timeouts
- Invalid tickers
- Missing CIK information
- API rate limits
- File download failures

## Legal Notice

This script is for educational and research purposes. When using SEC data:

1. Comply with SEC's terms of service
2. Include proper attribution
3. Use a descriptive User-Agent with contact information
4. Respect rate limits

## Troubleshooting

### Common Issues

1. **"Could not find CIK for ticker"**: 
   - The ticker might be incorrect
   - The company might not be publicly traded
   - Try looking up the CIK manually on SEC.gov

2. **Network errors**:
   - Check internet connection
   - SEC servers might be temporarily unavailable
   - Try again with a smaller batch size

3. **Empty results**:
   - The company might not have the requested form type
   - Try without specifying a form type
   - Check if the ticker is correct
