#!/bin/bash

# Test script to detect how ETFs are represented in Polygon API
# Usage: ./test_etf_detection.sh YOUR_API_KEY

if [ $# -eq 0 ]; then
    echo "Usage: $0 <polygon-api-key>"
    exit 1
fi

API_KEY="$1"
BASE_URL="https://api.polygon.io/v3/reference/tickers"

echo "================================================"
echo "ETF Detection Test for Polygon.io API"
echo "================================================"

# Test well-known ETF symbols
ETFS=("SPY" "QQQ" "IWM" "VOO" "VTI" "GLD" "SLV" "EEM" "XLF" "XLE")

echo -e "\nChecking known ETF symbols individually:"
echo "----------------------------------------"

for etf in "${ETFS[@]}"; do
    echo -e "\n🔍 Checking $etf..."
    response=$(curl -s "${BASE_URL}?ticker=${etf}&apikey=${API_KEY}")
    
    # Extract key fields
    ticker=$(echo "$response" | python3 -c "import sys, json; data=json.load(sys.stdin); print(data['results'][0]['ticker'] if data.get('results') else 'NOT FOUND')" 2>/dev/null)
    name=$(echo "$response" | python3 -c "import sys, json; data=json.load(sys.stdin); print(data['results'][0]['name'] if data.get('results') else '')" 2>/dev/null)
    type=$(echo "$response" | python3 -c "import sys, json; data=json.load(sys.stdin); print(data['results'][0].get('type', 'N/A') if data.get('results') else '')" 2>/dev/null)
    market=$(echo "$response" | python3 -c "import sys, json; data=json.load(sys.stdin); print(data['results'][0].get('market', 'N/A') if data.get('results') else '')" 2>/dev/null)
    
    if [ "$ticker" != "NOT FOUND" ]; then
        echo "  ✓ Found: $ticker"
        echo "    Name: $name"
        echo "    Type: $type"
        echo "    Market: $market"
    else
        echo "  ✗ Not found"
    fi
done

echo -e "\n================================================"
echo "Searching for ETFs by name pattern:"
echo "================================================"

# Search for "ETF" in name
echo -e "\nSearching for 'ETF' in name (limit=10):"
curl -s "${BASE_URL}?search=ETF&market=stocks&active=true&limit=10&apikey=${API_KEY}" | \
    python3 -c "
import sys, json
data = json.load(sys.stdin)
if 'results' in data:
    for item in data['results']:
        print(f\"{item['ticker']:10} Type: {item.get('type', 'N/A'):8} Name: {item['name'][:50]}\")
else:
    print('No results or error:', data.get('status', 'Unknown'))
"

# Search for "Exchange Traded" in name
echo -e "\nSearching for 'Exchange Traded' in name (limit=10):"
curl -s "${BASE_URL}?search=Exchange%20Traded&market=stocks&active=true&limit=10&apikey=${API_KEY}" | \
    python3 -c "
import sys, json
data = json.load(sys.stdin)
if 'results' in data:
    for item in data['results']:
        print(f\"{item['ticker']:10} Type: {item.get('type', 'N/A'):8} Name: {item['name'][:50]}\")
else:
    print('No results or error:', data.get('status', 'Unknown'))
"

echo -e "\n================================================"
echo "Summary of findings will help determine:"
echo "- How ETFs are marked in 'type' field"
echo "- Whether they need special filtering"
echo "- If name patterns can identify them"
echo "================================================"