#!/usr/bin/env python3
"""
Simple YCharts API Test Script

This script tests your YCharts API key by making a direct HTTP request.
"""

import json
import os

import requests
from dotenv import load_dotenv

# Load environment variables
load_dotenv()


def test_ycharts_api():
    """Test the YCharts API with your API key"""

    # Get API key from environment
    api_key = os.getenv("YCHARTS_API_KEY")

    if not api_key:
        print("❌ Error: YCHARTS_API_KEY not found in .env file")
        print("Make sure your .env file contains:")
        print("YCHARTS_API_KEY=your_actual_api_key_here")
        return

    print(
        f"🔑 Using API key: {api_key[:10]}..."
        if len(api_key) > 10
        else f"🔑 Using API key: {api_key}"
    )

    # Try a simple API request
    # Note: This is a generic approach - YCharts API endpoints may vary
    headers = {
        "Authorization": f"Bearer {api_key}",
        "Content-Type": "application/json",
        "User-Agent": "YCharts-Test-Client/1.0",
    }

    # Common YCharts API patterns to try
    test_urls = [
        "https://api.ycharts.com/v1/companies/AAPL/data/price",
        "https://api.ycharts.com/v1/companies/AAPL/price",
        "https://api.ycharts.com/v1/data/companies/AAPL/price",
        "https://ycharts.com/api/v1/companies/AAPL/price",
    ]

    for url in test_urls:
        print(f"\n🌐 Testing URL: {url}")

        try:
            response = requests.get(url, headers=headers, timeout=10)

            print(f"📊 Status Code: {response.status_code}")
            print(f"📋 Headers: {dict(response.headers)}")

            if response.status_code == 200:
                print("✅ Success! API is working.")
                try:
                    data = response.json()
                    print("📈 Response Data:")
                    print(json.dumps(data, indent=2))
                except:
                    print("📄 Response Text:")
                    print(response.text[:500])
                return
            elif response.status_code == 401:
                print("❌ Authentication failed. Check your API key.")
            elif response.status_code == 403:
                print("❌ Access forbidden. Check your API permissions.")
            elif response.status_code == 404:
                print("❌ Endpoint not found. Trying next URL...")
            else:
                print(f"❌ Unexpected status: {response.status_code}")
                print(f"Response: {response.text[:200]}")

        except requests.exceptions.Timeout:
            print("⏰ Request timeout")
        except requests.exceptions.ConnectionError:
            print("🌐 Connection error")
        except Exception as e:
            print(f"❌ Error: {e}")

    print("\n🔍 If all URLs failed, you may need to:")
    print("1. Check YCharts API documentation for correct endpoints")
    print("2. Verify your API key is valid and active")
    print("3. Check if your account has the required permissions")
    print("4. Contact YCharts support for API access details")


if __name__ == "__main__":
    print("🚀 YCharts API Test")
    print("=" * 50)
    test_ycharts_api()
