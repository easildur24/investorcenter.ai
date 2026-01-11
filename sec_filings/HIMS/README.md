# SEC Filing PE Ratio Calculator

This script analyzes SEC filing documents in XBRL format to extract earnings per share (EPS) data and calculate the Price-to-Earnings (PE) ratio.

## Features

- **Automatic EPS Extraction**: Parses XBRL-formatted SEC filings to extract earnings per share data
- **TTM Calculation**: Calculates trailing twelve months (TTM) EPS from quarterly data
- **PE Ratio Analysis**: Computes PE ratio and provides interpretation
- **Detailed Reporting**: Shows quarterly breakdown and analysis

## Usage

### Basic Usage (HIMS Example)
```bash
python3 calculate_pe_ratio.py
```

### General Usage
```bash
python3 calculate_pe_ratio.py [TICKER] [DOCUMENTS_DIRECTORY]
```

### Examples
```bash
# Analyze HIMS using documents in the current directory
python3 calculate_pe_ratio.py HIMS documents/

# Analyze any ticker with custom directory
python3 calculate_pe_ratio.py AAPL ../AAPL/filings/
```

## Requirements

- Python 3.6+
- No external dependencies required

## Input Files

The script expects SEC filing documents in XBRL HTML format (`.htm` files). These are typically:
- 10-Q (Quarterly Reports)
- 10-K (Annual Reports)

File naming should include the ticker symbol for automatic detection.

## Output

The script provides:

1. **Current Stock Price** (hardcoded for HIMS, manual input needed for others)
2. **TTM EPS** (calculated from the most recent 4 quarters)
3. **PE Ratio** (Price ÷ TTM EPS)
4. **Interpretation** (Low/Moderate/High/Very High PE analysis)
5. **Quarterly Breakdown** (Individual quarter EPS values)

### Sample Output
```
============================================================
PE RATIO ANALYSIS FOR HIMS
============================================================
Current Stock Price: $55.50
TTM EPS: $0.82
PE Ratio: 67.68
Analysis Date: 2025-09-14 10:17:22

Interpretation: Very High PE - Indicates very high growth expectations or potential overvaluation

This means investors are paying $67.68 for every $1 of annual earnings.

Quarterly EPS Breakdown (TTM):
----------------------------------------
Q2 2025: $0.19
Q1 2025: $0.22
Q3 2024: $0.35
Q2 2024: $0.06

Total TTM EPS: $0.82
```

## PE Ratio Interpretation Guide

- **< 15**: Low PE - May indicate undervaluation or slow growth expectations
- **15-25**: Moderate PE - Typical for established companies
- **25-50**: High PE - Indicates high growth expectations
- **> 50**: Very High PE - Indicates very high growth expectations or potential overvaluation

## How It Works

1. **File Discovery**: Scans the documents directory for SEC filing files containing the ticker symbol
2. **XBRL Parsing**: Extracts financial data using regex patterns to find US-GAAP elements:
   - `us-gaap:EarningsPerShare`
   - `us-gaap:NetIncome`
   - `us-gaap:WeightedAverageNumberOfSharesOutstanding`
3. **Quarter Identification**: Determines fiscal quarters from XBRL metadata or filename dates
4. **TTM Calculation**: Sums the most recent 4 quarters of EPS data
5. **PE Calculation**: Divides current stock price by TTM EPS

## Limitations

- **Stock Price**: Currently hardcoded for HIMS ($55.50). For other tickers, manual input or API integration needed
- **XBRL Format**: Only works with inline XBRL HTML format SEC filings
- **Data Quality**: Relies on consistent XBRL tagging in SEC documents

## Extending the Script

To add real-time stock price fetching:

1. Sign up for a financial data API (Alpha Vantage, IEX Cloud, etc.)
2. Replace the `get_current_stock_price()` method with API calls
3. Add your API key to the script

Example API integration:
```python
def get_current_stock_price(self) -> Optional[float]:
    api_key = "YOUR_API_KEY"
    url = f"https://api.example.com/quote/{self.ticker}"
    # Add API call implementation
```

## Files Analyzed (HIMS Example)

- `0001773751-24-000248_hims-20240630.htm` (Q2 2024)
- `0001773751-24-000339_hims-20240930.htm` (Q3 2024)
- `0001773751-25-000154_hims-20250331.htm` (Q1 2025)
- `0001773751-25-000250_hims-20250630.htm` (Q2 2025)

## Error Handling

The script handles:
- Missing or corrupted files
- Invalid XBRL data
- Zero or negative EPS values
- Missing stock price data

## Contributing

Feel free to enhance this script by:
- Adding real-time stock price API integration
- Supporting additional financial metrics
- Improving XBRL parsing accuracy
- Adding support for different filing formats
