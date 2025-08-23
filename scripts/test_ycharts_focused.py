#!/usr/bin/env python3
"""
Focused test on the working YCharts endpoints with 403 responses
"""

import os
import requests
import json
from dotenv import load_dotenv

load_dotenv()

def test_working_endpoints():
    """Focus on endpoints that returned 403 (exist but access denied)"""
    
    api_key = os.getenv("YCHARTS_API_KEY")
    
    if not api_key:
        print("❌ Error: YCHARTS_API_KEY not found")
        return
    
    print(f"🔑 API Key: {api_key[:10]}... (length: {len(api_key)})")
    
    # These endpoints returned 403, so they exist!
    working_endpoints = [
        "https://ycharts.com/api/v1/series",
        "https://ycharts.com/api/series"
    ]
    
    params = {
        'security_id': 'AAPL',
        'metric_ids': '1day_return,52_week_high'
    }
    
    # Try many different authentication methods
    auth_methods = [
        # Header-based authentication
        {'headers': {'Authorization': f'Bearer {api_key}'}},
        {'headers': {'Authorization': f'Token {api_key}'}},
        {'headers': {'Authorization': f'ApiKey {api_key}'}},
        {'headers': {'Authorization': f'YCharts {api_key}'}},
        {'headers': {'Authorization': api_key}},
        {'headers': {'X-API-Key': api_key}},
        {'headers': {'X-Auth-Token': api_key}},
        {'headers': {'YCharts-API-Key': api_key}},
        {'headers': {'YCharts-Token': api_key}},
        {'headers': {'API-Key': api_key}},
        {'headers': {'Token': api_key}},
        
        # Query parameter authentication
        {'params': {'api_key': api_key}},
        {'params': {'token': api_key}},
        {'params': {'key': api_key}},
        {'params': {'auth': api_key}},
        {'params': {'access_token': api_key}},
        {'params': {'ycharts_key': api_key}},
        
        # Basic auth (sometimes API keys go in username field)
        {'auth': (api_key, '')},
        {'auth': ('', api_key)},
        {'auth': (api_key, 'password')},
        
        # Combined approaches
        {'headers': {'Authorization': f'Bearer {api_key}'}, 'params': {'format': 'json'}},
        {'headers': {'X-API-Key': api_key}, 'params': {'format': 'json'}},
    ]
    
    for endpoint in working_endpoints:
        print(f"\n🌐 Testing endpoint: {endpoint}")
        print("=" * 60)
        
        for i, auth_method in enumerate(auth_methods):
            print(f"🔐 Auth method {i+1:2d}: ", end="")
            
            # Build request parameters
            request_params = {
                'url': endpoint,
                'timeout': 10,
                'params': params.copy()
            }
            
            # Add authentication
            if 'headers' in auth_method:
                request_params['headers'] = auth_method['headers'].copy()
                request_params['headers']['Content-Type'] = 'application/json'
                print(f"Headers: {list(auth_method['headers'].keys())}")
            
            if 'params' in auth_method:
                request_params['params'].update(auth_method['params'])
                print(f"Query params: {list(auth_method['params'].keys())}")
            
            if 'auth' in auth_method:
                request_params['auth'] = auth_method['auth']
                print(f"Basic auth: {auth_method['auth'][0][:10]}...")
            
            if not any(k in auth_method for k in ['headers', 'params', 'auth']):
                print("No auth method")
                continue
            
            try:
                response = requests.get(**request_params)
                status = response.status_code
                
                if status == 200:
                    print(f"   ✅ SUCCESS! Status: {status}")
                    try:
                        data = response.json()
                        print("   📈 Response data:")
                        print(json.dumps(data, indent=2)[:1000])
                    except:
                        print("   📄 Response text:")
                        print(response.text[:500])
                    return True
                    
                elif status == 401:
                    print(f"   🔒 Unauthorized ({status})")
                elif status == 403:
                    print(f"   🚫 Forbidden ({status}) - endpoint exists!")
                elif status == 400:
                    print(f"   ❓ Bad Request ({status})")
                    print(f"      Response: {response.text[:150]}")
                elif status == 404:
                    print(f"   ❌ Not Found ({status})")
                else:
                    print(f"   ❓ Status {status}")
                    if len(response.text) < 200:
                        print(f"      Response: {response.text}")
                        
            except requests.exceptions.Timeout:
                print("   ⏰ Timeout")
            except Exception as e:
                print(f"   ❌ Error: {e}")
    
    return False

def test_simple_endpoints():
    """Test simpler endpoints that might not require parameters"""
    
    api_key = os.getenv("YCHARTS_API_KEY")
    
    print("\n" + "="*60)
    print("🔍 Testing simple endpoints (no parameters)...")
    
    simple_endpoints = [
        "https://ycharts.com/api/v1/",
        "https://ycharts.com/api/",
        "https://ycharts.com/api/v1/status",
        "https://ycharts.com/api/status",
        "https://ycharts.com/api/v1/ping",
        "https://ycharts.com/api/ping",
        "https://ycharts.com/api/v1/user",
        "https://ycharts.com/api/user",
    ]
    
    headers = {'Authorization': f'Bearer {api_key}', 'Content-Type': 'application/json'}
    
    for endpoint in simple_endpoints:
        print(f"\n🌐 {endpoint}")
        try:
            response = requests.get(endpoint, headers=headers, timeout=10)
            print(f"   📊 Status: {response.status_code}")
            
            if response.status_code not in [404, 403]:
                print(f"   📄 Response: {response.text[:300]}")
                
        except Exception as e:
            print(f"   ❌ Error: {e}")

if __name__ == "__main__":
    print("🎯 Focused YCharts API Test")
    print("Testing endpoints that returned 403 (exist but access denied)")
    print("=" * 60)
    
    success = test_working_endpoints()
    
    if not success:
        test_simple_endpoints()
        
    print("\n" + "="*60)
    print("🔍 Key findings:")
    print("- Endpoints at ycharts.com/api/ exist (403 vs 404)")
    print("- Need to find correct authentication method")
    print("- API key format appears correct")
    print("- May need YCharts support for auth details")
