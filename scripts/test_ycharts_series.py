#!/usr/bin/env python3
"""
Test YCharts Series API Endpoint

Testing the specific endpoint structure provided by the user.
"""

import os
import requests
import json
from dotenv import load_dotenv

# Load environment variables
load_dotenv()

def test_ycharts_series_endpoint():
    """Test the YCharts series endpoint with different authentication methods"""
    
    api_key = os.getenv("YCHARTS_API_KEY")
    
    if not api_key:
        print("❌ Error: YCHARTS_API_KEY not found in .env file")
        return
    
    print(f"🔑 Testing with API key: {api_key[:10]}...")
    
    # The endpoint structure you provided
    base_url = "https://api.ycharts.com/v1/series"
    params = {
        'security_id': 'AAPL',
        'metric_ids': '1day_return,52_week_high'
    }
    
    # Different authentication methods to try
    auth_methods = [
        {
            "name": "Bearer Token",
            "headers": {
                'Authorization': f'Bearer {api_key}',
                'Content-Type': 'application/json'
            }
        },
        {
            "name": "API Key Header",
            "headers": {
                'X-API-Key': api_key,
                'Content-Type': 'application/json'
            }
        },
        {
            "name": "YCharts-API-Key Header",
            "headers": {
                'YCharts-API-Key': api_key,
                'Content-Type': 'application/json'
            }
        },
        {
            "name": "Authorization Header (Direct)",
            "headers": {
                'Authorization': api_key,
                'Content-Type': 'application/json'
            }
        },
        {
            "name": "Query Parameter",
            "headers": {'Content-Type': 'application/json'},
            "extra_params": {'api_key': api_key}
        },
        {
            "name": "Token Query Parameter",
            "headers": {'Content-Type': 'application/json'},
            "extra_params": {'token': api_key}
        },
        {
            "name": "Key Query Parameter",
            "headers": {'Content-Type': 'application/json'},
            "extra_params": {'key': api_key}
        }
    ]
    
    for auth_method in auth_methods:
        print(f"\n🔐 Testing authentication method: {auth_method['name']}")
        print("-" * 60)
        
        # Combine parameters
        test_params = params.copy()
        if 'extra_params' in auth_method:
            test_params.update(auth_method['extra_params'])
        
        print(f"🌐 URL: {base_url}")
        print(f"📋 Params: {test_params}")
        print(f"📋 Headers: {auth_method['headers']}")
        
        try:
            response = requests.get(
                base_url,
                headers=auth_method['headers'],
                params=test_params,
                timeout=15
            )
            
            status_code = response.status_code
            print(f"📊 Status Code: {status_code}")
            
            if status_code == 200:
                print("✅ SUCCESS! API call worked!")
                print(f"📋 Response Headers: {dict(response.headers)}")
                
                try:
                    data = response.json()
                    print("📈 Response Data:")
                    print(json.dumps(data, indent=2))
                    return True
                except json.JSONDecodeError:
                    print("📄 Response Text (not JSON):")
                    print(response.text[:1000])
                    return True
                    
            elif status_code == 401:
                print("🔒 401 Unauthorized - Invalid or missing authentication")
            elif status_code == 403:
                print("🚫 403 Forbidden - Valid auth but insufficient permissions")
            elif status_code == 404:
                print("❌ 404 Not Found - Endpoint doesn't exist")
            elif status_code == 400:
                print("❓ 400 Bad Request - Check parameters")
                print(f"Response: {response.text[:300]}")
            elif status_code == 429:
                print("⏱️ 429 Rate Limited - Too many requests")
            else:
                print(f"❓ Status {status_code}")
                print(f"Response: {response.text[:300]}")
                
        except requests.exceptions.Timeout:
            print("⏰ Request timeout")
        except requests.exceptions.ConnectionError:
            print("🌐 Connection error")
        except Exception as e:
            print(f"❌ Error: {e}")
    
    return False

def test_variations():
    """Test variations of the endpoint"""
    
    api_key = os.getenv("YCHARTS_API_KEY")
    
    print("\n" + "="*60)
    print("🔄 Testing endpoint variations...")
    
    # Test different parameter formats
    variations = [
        {
            "name": "Original format",
            "params": {
                'security_id': 'AAPL',
                'metric_ids': '1day_return,52_week_high'
            }
        },
        {
            "name": "Single metric",
            "params": {
                'security_id': 'AAPL',
                'metric_ids': '1day_return'
            }
        },
        {
            "name": "Different security",
            "params": {
                'security_id': 'MSFT',
                'metric_ids': '1day_return'
            }
        },
        {
            "name": "Array format",
            "params": {
                'security_id': 'AAPL',
                'metric_ids[]': ['1day_return', '52_week_high']
            }
        }
    ]
    
    # Use Bearer token as it's most common
    headers = {
        'Authorization': f'Bearer {api_key}',
        'Content-Type': 'application/json'
    }
    
    for variation in variations:
        print(f"\n📊 Testing: {variation['name']}")
        
        try:
            response = requests.get(
                "https://api.ycharts.com/v1/series",
                headers=headers,
                params=variation['params'],
                timeout=10
            )
            
            print(f"   Status: {response.status_code}")
            if response.status_code != 404:
                print(f"   Response preview: {response.text[:200]}")
                
        except Exception as e:
            print(f"   Error: {e}")

if __name__ == "__main__":
    print("🚀 Testing YCharts Series Endpoint")
    print("=" * 60)
    
    success = test_ycharts_series_endpoint()
    
    if not success:
        test_variations()
        
    print("\n" + "="*60)
    print("💡 If none worked, the API key might need:")
    print("   - Different authentication method")
    print("   - Account activation or permissions")
    print("   - Contact YCharts support for guidance")
