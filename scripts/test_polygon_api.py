#!/usr/bin/env python3
"""
Test Polygon API ticker fetching
"""

import requests
import json
import sys
import os

API_KEY = os.getenv("POLYGON_API_KEY", "zapuIgaTVLJoanfEuimZYQ2xRlZmoU1m")
BASE_URL = "https://api.polygon.io/v3/reference/tickers"

def test_polygon_api():
    print("🧪 Testing Polygon Ticker API")
    print("=" * 50)
    print(f"API Key: {API_KEY[:10]}...{API_KEY[-4:]}\n")
    
    # Test 1: Fetch US Stocks
    print("1️⃣ Fetching US Stocks (limit 3)...")
    params = {
        "market": "stocks",
        "locale": "us",
        "type": "CS",
        "active": "true",
        "limit": 3,
        "apikey": API_KEY
    }
    
    response = requests.get(BASE_URL, params=params)
    if response.status_code == 200:
        data = response.json()
        if data.get("status") == "OK":
            print(f"✅ Found {len(data.get('results', []))} stocks:")
            for ticker in data.get("results", []):
                print(f"   {ticker['ticker']} - {ticker['name']}")
                print(f"      Type: {ticker.get('type', 'N/A')}, Exchange: {ticker.get('primary_exchange', 'N/A')}")
        else:
            print(f"❌ API Error: {data.get('error', 'Unknown')}")
    else:
        print(f"❌ HTTP Error: {response.status_code}")
    
    print()
    
    # Test 2: Fetch ETFs
    print("2️⃣ Fetching ETFs (limit 3)...")
    params = {
        "market": "stocks",
        "type": "ETF",
        "active": "true",
        "limit": 3,
        "apikey": API_KEY
    }
    
    response = requests.get(BASE_URL, params=params)
    if response.status_code == 200:
        data = response.json()
        if data.get("status") == "OK":
            print(f"✅ Found {len(data.get('results', []))} ETFs:")
            for ticker in data.get("results", []):
                print(f"   {ticker['ticker']} - {ticker['name']}")
                print(f"      Type: {ticker.get('type', 'N/A')}")
        else:
            print(f"❌ API Error: {data.get('error', 'Unknown')}")
    else:
        print(f"❌ HTTP Error: {response.status_code}")
    
    print()
    
    # Test 3: Check specific tickers
    print("3️⃣ Checking specific tickers...")
    for symbol in ["AAPL", "SPY"]:
        params = {
            "ticker": symbol,
            "apikey": API_KEY
        }
        response = requests.get(BASE_URL, params=params)
        if response.status_code == 200:
            data = response.json()
            if data.get("status") == "OK" and data.get("results"):
                ticker = data["results"][0]
                print(f"✅ {symbol}:")
                print(f"   Name: {ticker['name']}")
                print(f"   Type: {ticker.get('type', 'N/A')}")
                print(f"   Market: {ticker.get('market', 'N/A')}")
                print(f"   CIK: {ticker.get('cik', 'N/A')}")
            else:
                print(f"❌ {symbol}: Not found")
        else:
            print(f"❌ {symbol}: HTTP Error {response.status_code}")
        print()
    
    # Test 4: Show sample JSON
    print("4️⃣ Sample ticker JSON structure:")
    params = {
        "ticker": "AAPL",
        "apikey": API_KEY
    }
    response = requests.get(BASE_URL, params=params)
    if response.status_code == 200:
        data = response.json()
        if data.get("results"):
            print(json.dumps(data["results"][0], indent=2))
    
    print("\n✅ Test complete!")

if __name__ == "__main__":
    test_polygon_api()