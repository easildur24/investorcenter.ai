#!/bin/bash

# Test script for Polygon.io Tickers API
# Usage: ./test_polygon_tickers.sh YOUR_API_KEY

if [ $# -eq 0 ]; then
    echo "Usage: $0 <polygon-api-key>"
    echo "Example: $0 YOUR_POLYGON_API_KEY"
    exit 1
fi

API_KEY="$1"
BASE_URL="https://api.polygon.io/v3/reference/tickers"

echo "================================================"
echo "Testing Polygon.io Tickers API"
echo "================================================"

# First, get all ticker types
echo -e "\n0. FETCHING ALL TICKER TYPES..."
echo "--------------------------------"
curl -s "https://api.polygon.io/v3/reference/tickers/types?apikey=${API_KEY}" | python3 -m json.tool

# Test 1: US Stocks
echo -e "\n1. FETCHING US STOCKS (limit=2)..."
echo "--------------------------------"
curl -s "${BASE_URL}?market=stocks&locale=us&active=true&limit=2&apikey=${API_KEY}" | python3 -m json.tool

# Test 2: Crypto
echo -e "\n2. FETCHING CRYPTO (limit=2)..."
echo "--------------------------------"
curl -s "${BASE_URL}?market=crypto&active=true&limit=2&apikey=${API_KEY}" | python3 -m json.tool

# Test 3: ETFs - Try multiple approaches
echo -e "\n3a. FETCHING ETFs using type=ETF (limit=2)..."
echo "--------------------------------"
curl -s "${BASE_URL}?market=stocks&type=ETF&active=true&limit=2&apikey=${API_KEY}" | python3 -m json.tool

echo -e "\n3b. FETCHING ETFs using type=ETP (limit=2)..."
echo "--------------------------------"
curl -s "${BASE_URL}?market=stocks&type=ETP&active=true&limit=2&apikey=${API_KEY}" | python3 -m json.tool

echo -e "\n3c. FETCHING ETFs using type=ETN (limit=2)..."
echo "--------------------------------"
curl -s "${BASE_URL}?market=stocks&type=ETN&active=true&limit=2&apikey=${API_KEY}" | python3 -m json.tool

echo -e "\n3d. FETCHING ETFs using search=ETF in ticker (limit=5)..."
echo "--------------------------------"
curl -s "${BASE_URL}?market=stocks&search=ETF&active=true&limit=5&apikey=${API_KEY}" | python3 -m json.tool

# Test 4: Indices
echo -e "\n4. FETCHING INDICES (limit=2)..."
echo "--------------------------------"
curl -s "${BASE_URL}?market=indices&active=true&limit=2&apikey=${API_KEY}" | python3 -m json.tool

# Test 5: Get count of each type
echo -e "\n5. SUMMARY - Total counts for each type:"
echo "--------------------------------"

echo -n "US Stocks: "
curl -s "${BASE_URL}?market=stocks&locale=us&active=true&limit=1&apikey=${API_KEY}" | python3 -c "import sys, json; print(json.load(sys.stdin).get('count', 'N/A'))"

echo -n "Crypto: "
curl -s "${BASE_URL}?market=crypto&active=true&limit=1&apikey=${API_KEY}" | python3 -c "import sys, json; print(json.load(sys.stdin).get('count', 'N/A'))"

echo -n "ETFs (type=ETF): "
curl -s "${BASE_URL}?market=stocks&type=ETF&active=true&limit=1&apikey=${API_KEY}" | python3 -c "import sys, json; print(json.load(sys.stdin).get('count', 'N/A'))"

echo -n "ETPs (type=ETP): "
curl -s "${BASE_URL}?market=stocks&type=ETP&active=true&limit=1&apikey=${API_KEY}" | python3 -c "import sys, json; print(json.load(sys.stdin).get('count', 'N/A'))"

echo -n "ETNs (type=ETN): "
curl -s "${BASE_URL}?market=stocks&type=ETN&active=true&limit=1&apikey=${API_KEY}" | python3 -c "import sys, json; print(json.load(sys.stdin).get('count', 'N/A'))"

echo -n "Indices: "
curl -s "${BASE_URL}?market=indices&active=true&limit=1&apikey=${API_KEY}" | python3 -c "import sys, json; print(json.load(sys.stdin).get('count', 'N/A'))"

echo -e "\n================================================"
echo "Test complete!"
echo "================================================"