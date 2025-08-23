# YCharts API Query Script

This directory contains a comprehensive Python script for querying the YCharts API to fetch financial data including stock prices, company information, mutual fund data, and economic indicators.

## Files

- `ycharts_query.py` - Main script with YCharts API client
- `ycharts_examples.py` - Usage examples and sample queries
- `requirements.txt` - Python package dependencies
- `ycharts_env_example` - Environment variable template

## Setup

### 1. Install Dependencies

```bash
cd scripts
pip3 install -r requirements.txt
```

**Note:** On macOS, you may need to use `pip3` instead of `pip`.

### 2. Set Up API Key

You have two options for providing your YCharts API key:

**Option A: Environment Variable (Recommended)**
```bash
# Copy the example file
cp ycharts_env_example .env

# Edit .env and add your actual API key
YCHARTS_API_KEY=your_actual_api_key_here
```

**Option B: Command Line Argument**
```bash
python ycharts_query.py --api-key your_actual_api_key_here --symbols AAPL --metrics price
```

## Usage

### Command Line Interface

The script provides a comprehensive command-line interface:

```bash
# Get latest stock price
python ycharts_query.py --symbols AAPL --metrics price

# Get multiple metrics for multiple stocks
python ycharts_query.py --symbols AAPL GOOGL MSFT --metrics price market_cap volume

# Get historical data for the past 30 days
python ycharts_query.py --symbols AAPL --metrics price --query-type historical --days-back 30

# Get historical data for specific date range
python ycharts_query.py --symbols AAPL --metrics price --query-type historical --start-date 2024-01-01 --end-date 2024-01-31

# Get company information
python ycharts_query.py --symbols AAPL --query-type info --info-fields description sector industry

# Get mutual fund data
python ycharts_query.py --symbols VTIAX --metrics price net_assets --query-type mutual_fund

# Get economic indicators
python ycharts_query.py --symbols GDP UNEMPLOYMENT_RATE --query-type indicator

# Output to different formats
python ycharts_query.py --symbols AAPL --metrics price --output-format table
python ycharts_query.py --symbols AAPL --metrics price --output-format csv --output-file aapl_data.csv
```

### Python API

You can also use the `YChartsAPIClient` class directly in your Python code:

```python
from ycharts_query import YChartsAPIClient
import datetime

# Initialize client
client = YChartsAPIClient("your_api_key")

# Get latest data
response = client.get_company_latest_data(["AAPL", "GOOGL"], ["price", "market_cap"])

# Get historical data
end_date = datetime.datetime.now()
start_date = end_date - datetime.timedelta(days=30)
response = client.get_company_historical_data(["AAPL"], ["price"], start_date, end_date)

# Get company info
response = client.get_company_info(["AAPL"], ["description", "sector"])
```

## Available Query Types

1. **latest** - Get current/latest data points
2. **historical** - Get time series data over a date range
3. **info** - Get company information and metadata
4. **mutual_fund** - Get mutual fund specific data
5. **indicator** - Get economic indicator data

## Common Metrics

### Company Metrics
- `price` - Current stock price
- `market_cap` - Market capitalization
- `volume` - Trading volume
- `pe_ratio` - Price-to-earnings ratio
- `dividend_yield` - Dividend yield
- `beta` - Stock beta

### Company Info Fields
- `description` - Company description
- `sector` - Business sector
- `industry` - Industry classification
- `employees` - Number of employees
- `headquarters` - Company headquarters location

### Mutual Fund Metrics
- `price` - Fund price/NAV
- `net_assets` - Total net assets
- `expense_ratio` - Annual expense ratio
- `yield` - Fund yield

## Output Formats

- **json** (default) - Pretty-printed JSON
- **csv** - Comma-separated values (requires pandas)
- **table** - Formatted table view (requires pandas)

## Examples

Run the examples script to see the API in action:

```bash
python ycharts_examples.py
```

## Error Handling

The script includes comprehensive error handling:
- Invalid API keys
- Network connectivity issues
- Invalid symbols or metrics
- Date range errors
- Missing dependencies

## Requirements

- Python 3.7+
- YCharts API access token
- Internet connection

## Troubleshooting

1. **"pycharts library not found"**
   - Run: `pip install git+https://github.com/ycharts/pycharts.git`

2. **"API key required"**
   - Set the `YCHARTS_API_KEY` environment variable or use `--api-key` argument

3. **"Invalid symbol" errors**
   - Verify the symbol exists in YCharts database
   - Check symbol format (some may require exchange suffixes)

4. **Rate limiting**
   - YCharts may have rate limits; add delays between requests if needed

## Support

For YCharts API-specific questions:
- YCharts API Documentation: https://ycharts.com/api/
- YCharts Support: Contact YCharts directly for API access and support

For script issues:
- Check the error messages and logs
- Verify all dependencies are installed
- Ensure your API key is valid and has appropriate permissions
