#!/usr/bin/env python3
"""
Comprehensive YCharts API Test Script

This script tests different authentication methods and endpoint patterns
to find the correct way to access the YCharts API.
"""

import json
import os

import requests
from dotenv import load_dotenv

# Load environment variables
load_dotenv()


def test_ycharts_api_comprehensive():
    """Test YCharts API with different authentication methods and endpoints"""

    api_key = os.getenv("YCHARTS_API_KEY")

    if not api_key:
        print("❌ Error: YCHARTS_API_KEY not found in .env file")
        return

    print(f"🔑 Testing with API key: {api_key[:10]}...")

    # Different authentication methods to try
    auth_methods = [
        {
            "name": "Bearer Token",
            "headers": {
                "Authorization": f"Bearer {api_key}",
                "Content-Type": "application/json",
            },
        },
        {
            "name": "API Key Header",
            "headers": {
                "X-API-Key": api_key,
                "Content-Type": "application/json",
            },
        },
        {
            "name": "Authorization Header",
            "headers": {
                "Authorization": api_key,
                "Content-Type": "application/json",
            },
        },
        {
            "name": "Query Parameter",
            "headers": {"Content-Type": "application/json"},
            "params": {"api_key": api_key},
        },
        {
            "name": "Token Query Parameter",
            "headers": {"Content-Type": "application/json"},
            "params": {"token": api_key},
        },
    ]

    # Different endpoint patterns to try
    endpoints = [
        # Standard REST patterns
        "https://api.ycharts.com/v1/companies/AAPL",
        "https://api.ycharts.com/v1/securities/AAPL",
        "https://api.ycharts.com/v1/stocks/AAPL",
        "https://api.ycharts.com/companies/AAPL",
        "https://api.ycharts.com/securities/AAPL",
        # With data endpoints
        "https://api.ycharts.com/v1/companies/AAPL/data",
        "https://api.ycharts.com/v1/companies/AAPL/metrics",
        "https://api.ycharts.com/v1/companies/AAPL/price",
        # Alternative base URLs
        "https://ycharts.com/api/companies/AAPL",
        "https://ycharts.com/api/v1/companies/AAPL",
        # GraphQL endpoint (common pattern)
        "https://api.ycharts.com/graphql",
        "https://ycharts.com/graphql",
        # Generic data endpoints
        "https://api.ycharts.com/data",
        "https://api.ycharts.com/v1/data",
    ]

    success_found = False

    for auth_method in auth_methods:
        print(f"\n🔐 Testing authentication method: {auth_method['name']}")
        print("-" * 60)

        for endpoint in endpoints:
            print(f"🌐 Testing: {endpoint}")

            try:
                params = auth_method.get("params", {})
                response = requests.get(
                    endpoint,
                    headers=auth_method["headers"],
                    params=params,
                    timeout=10,
                )

                status_code = response.status_code
                print(f"   📊 Status: {status_code}")

                if status_code == 200:
                    print("   ✅ SUCCESS! Found working endpoint!")
                    print(f"   📋 Headers: {dict(response.headers)}")
                    try:
                        data = response.json()
                        print("   📈 Response Data:")
                        print(json.dumps(data, indent=4)[:1000])
                    except:
                        print("   📄 Response Text:")
                        print(response.text[:500])
                    success_found = True
                    return

                elif status_code == 401:
                    print("   🔒 Authentication required")
                elif status_code == 403:
                    print("   🚫 Access forbidden")
                elif status_code == 404:
                    print("   ❌ Not found")
                elif status_code == 429:
                    print("   ⏱️ Rate limited")
                else:
                    print(f"   ❓ Status {status_code}: {response.text[:100]}")

            except requests.exceptions.Timeout:
                print("   ⏰ Timeout")
            except requests.exceptions.ConnectionError:
                print("   🌐 Connection error")
            except Exception as e:
                print(f"   ❌ Error: {e}")

    if not success_found:
        print("\n" + "=" * 60)
        print("🔍 No working endpoints found. Next steps:")
        print("1. Check YCharts official API documentation")
        print("2. Verify your API key is active and has correct permissions")
        print("3. Contact YCharts support for API access details")
        print("4. Check if you need to use their Python SDK instead")
        print("\n💡 Your API key appears to be formatted correctly,")
        print("   so the issue is likely with endpoint URLs or auth method.")


def test_sdk_availability():
    """Test if YCharts provides an official SDK"""
    print("\n🔍 Checking for YCharts SDK availability...")

    try:
        import ycharts

        print("✅ YCharts SDK found!")
        return True
    except ImportError:
        print("❌ No YCharts SDK found")

    try:
        import pycharts

        print("✅ PyCharts library found!")
        return True
    except ImportError:
        print("❌ No PyCharts library found")

    return False


if __name__ == "__main__":
    print("🚀 Comprehensive YCharts API Test")
    print("=" * 60)

    # Test SDK availability first
    sdk_available = test_sdk_availability()

    if not sdk_available:
        print("\n📡 Testing direct API access...")
        test_ycharts_api_comprehensive()
    else:
        print("\n💡 Consider using the official SDK for better integration")
