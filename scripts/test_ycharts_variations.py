#!/usr/bin/env python3
"""
Test YCharts API with different base URLs and endpoint variations
"""

import json
import os

import requests
from dotenv import load_dotenv

load_dotenv()


def test_base_url_variations():
    """Test different base URLs for YCharts API"""

    api_key = os.getenv("YCHARTS_API_KEY")

    if not api_key:
        print("❌ Error: YCHARTS_API_KEY not found")
        return

    print(f"🔑 Testing with API key: {api_key[:10]}...")

    # Different possible base URLs
    base_urls = [
        "https://api.ycharts.com/v1/series",
        "https://api.ycharts.com/v2/series",
        "https://api.ycharts.com/series",
        "https://ycharts.com/api/v1/series",
        "https://ycharts.com/api/series",
        "https://data.ycharts.com/v1/series",
        "https://data.ycharts.com/series",
        "https://api-v1.ycharts.com/series",
        "https://marketdata.ycharts.com/v1/series",
        "https://marketdata.ycharts.com/series",
    ]

    params = {"security_id": "AAPL", "metric_ids": "1day_return,52_week_high"}

    # Try with Bearer token (most common)
    headers = {
        "Authorization": f"Bearer {api_key}",
        "Content-Type": "application/json",
        "User-Agent": "YCharts-Python-Client/1.0",
    }

    for base_url in base_urls:
        print(f"\n🌐 Testing: {base_url}")

        try:
            response = requests.get(
                base_url, headers=headers, params=params, timeout=10
            )

            status = response.status_code
            print(f"   📊 Status: {status}")

            if status == 200:
                print("   ✅ SUCCESS!")
                try:
                    data = response.json()
                    print("   📈 Data preview:")
                    print(json.dumps(data, indent=2)[:500])
                except:
                    print("   📄 Response text:")
                    print(response.text[:300])
                return True

            elif status == 401:
                print("   🔒 Unauthorized - check auth method")
            elif status == 403:
                print("   🚫 Forbidden - endpoint exists but access denied")
            elif status == 404:
                print("   ❌ Not found")
            elif status == 400:
                print("   ❓ Bad request")
                print(f"   Response: {response.text[:200]}")
            else:
                print(f"   ❓ Status {status}: {response.text[:100]}")

        except requests.exceptions.Timeout:
            print("   ⏰ Timeout")
        except requests.exceptions.ConnectionError:
            print("   🌐 Connection error")
        except Exception as e:
            print(f"   ❌ Error: {e}")

    return False


def test_alternative_endpoints():
    """Test alternative endpoint structures"""

    api_key = os.getenv("YCHARTS_API_KEY")

    print("\n" + "=" * 60)
    print("🔄 Testing alternative endpoint structures...")

    # Different endpoint patterns
    endpoints = [
        # REST-style endpoints
        (
            "https://api.ycharts.com/v1/securities/AAPL/metrics",
            {"metrics": "1day_return,52_week_high"},
        ),
        (
            "https://api.ycharts.com/v1/securities/AAPL",
            {"metrics": "1day_return,52_week_high"},
        ),
        (
            "https://api.ycharts.com/v1/companies/AAPL/series",
            {"metrics": "1day_return,52_week_high"},
        ),
        (
            "https://api.ycharts.com/v1/data",
            {"security_id": "AAPL", "metrics": "1day_return,52_week_high"},
        ),
        # Query-based endpoints
        (
            "https://api.ycharts.com/v1/query",
            {"security": "AAPL", "metrics": "1day_return,52_week_high"},
        ),
        (
            "https://api.ycharts.com/v1/timeseries",
            {"symbol": "AAPL", "metrics": "1day_return,52_week_high"},
        ),
        # GraphQL style
        (
            "https://api.ycharts.com/graphql",
            {
                "query": '{ security(id: "AAPL") { metrics(ids: ["1day_return", "52_week_high"]) } }'
            },
        ),
    ]

    headers = {
        "Authorization": f"Bearer {api_key}",
        "Content-Type": "application/json",
    }

    for url, params in endpoints:
        print(f"\n🌐 Testing: {url}")
        print(f"   📋 Params: {params}")

        try:
            response = requests.get(
                url, headers=headers, params=params, timeout=10
            )
            status = response.status_code
            print(f"   📊 Status: {status}")

            if status not in [404]:  # Show any non-404 responses
                print(f"   📄 Response preview: {response.text[:200]}")

        except Exception as e:
            print(f"   ❌ Error: {e}")


def test_with_api_key_formats():
    """Test different API key formats"""

    api_key = os.getenv("YCHARTS_API_KEY")

    print("\n" + "=" * 60)
    print("🔑 Testing different API key formats...")

    # Test if the API key needs to be formatted differently
    key_formats = [
        f"Bearer {api_key}",
        f"Token {api_key}",
        f"ApiKey {api_key}",
        f"YCharts {api_key}",
        api_key,
    ]

    url = "https://api.ycharts.com/v1/series"
    params = {"security_id": "AAPL", "metric_ids": "1day_return"}

    for key_format in key_formats:
        print(f"\n🔐 Testing auth format: {key_format[:20]}...")

        headers = {
            "Authorization": key_format,
            "Content-Type": "application/json",
        }

        try:
            response = requests.get(
                url, headers=headers, params=params, timeout=10
            )
            print(f"   📊 Status: {response.status_code}")

            if response.status_code not in [404, 401]:
                print(f"   📄 Response: {response.text[:200]}")

        except Exception as e:
            print(f"   ❌ Error: {e}")


if __name__ == "__main__":
    print("🚀 Testing YCharts API Variations")
    print("=" * 60)

    success = test_base_url_variations()

    if not success:
        test_alternative_endpoints()
        test_with_api_key_formats()

    print("\n" + "=" * 60)
    print("🔍 Summary:")
    print("- Your API key is properly formatted and loaded")
    print("- We need to find the correct base URL and endpoint structure")
    print("- Consider checking YCharts documentation or contacting support")
    print("- The API might require account activation or specific permissions")
