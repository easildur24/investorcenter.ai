#!/usr/bin/env python3
"""
Direct test of Polygon.io API to isolate the real issue
Tests the exact same endpoint our Go backend is using
"""

import requests
import time
import json
from datetime import datetime

# Your actual API key (same one the backend uses)
API_KEY = "Q9LhuSPrdj8Fqv9ejYqwXF6AKv7YAsWa"
BASE_URL = "https://api.polygon.io"

def test_polygon_endpoint(symbol, num_calls=5, delay=2):
    """Test the exact endpoint our backend uses"""
    
    # This is the EXACT same endpoint our Go backend calls
    url = f"{BASE_URL}/v2/aggs/ticker/{symbol}/prev?adjusted=true&apikey={API_KEY}"
    
    print(f"🧪 Testing Polygon API directly")
    print(f"📋 Endpoint: {url}")
    print(f"🔄 Making {num_calls} calls with {delay}s delay")
    print("=" * 60)
    
    results = []
    
    for i in range(1, num_calls + 1):
        print(f"\n📞 Call {i}/{num_calls} at {datetime.now().strftime('%H:%M:%S')}")
        
        try:
            start_time = time.time()
            response = requests.get(url, timeout=10)
            end_time = time.time()
            
            print(f"⏱️  Response time: {(end_time - start_time)*1000:.1f}ms")
            print(f"📊 Status Code: {response.status_code}")
            
            if response.status_code == 200:
                data = response.json()
                print(f"✅ Status: {data.get('status', 'UNKNOWN')}")
                
                if data.get('status') == 'OK' and data.get('results'):
                    result = data['results'][0]
                    price = result.get('c', 'N/A')  # Close price
                    volume = result.get('v', 'N/A')  # Volume
                    timestamp = result.get('t', 'N/A')  # Timestamp
                    
                    print(f"💰 Price: ${price}")
                    print(f"📈 Volume: {volume}")
                    print(f"🕐 Timestamp: {timestamp}")
                    
                    if timestamp != 'N/A':
                        dt = datetime.fromtimestamp(timestamp/1000)
                        print(f"📅 Date: {dt.strftime('%Y-%m-%d %H:%M:%S')}")
                else:
                    print(f"❌ No data in response: {data}")
                    
            else:
                print(f"❌ HTTP Error: {response.status_code}")
                try:
                    error_data = response.json()
                    print(f"🚫 Error: {error_data}")
                except:
                    print(f"🚫 Raw response: {response.text[:200]}")
            
            results.append({
                'call': i,
                'status_code': response.status_code,
                'success': response.status_code == 200,
                'response_time': (end_time - start_time) * 1000
            })
            
        except Exception as e:
            print(f"💥 Exception: {e}")
            results.append({
                'call': i,
                'status_code': 'ERROR',
                'success': False,
                'error': str(e)
            })
        
        # Wait between calls (except for last call)
        if i < num_calls:
            print(f"⏳ Waiting {delay} seconds...")
            time.sleep(delay)
    
    # Summary
    print("\n" + "=" * 60)
    print("📊 SUMMARY:")
    print("=" * 60)
    
    successful_calls = sum(1 for r in results if r.get('success', False))
    print(f"✅ Successful calls: {successful_calls}/{num_calls}")
    print(f"❌ Failed calls: {num_calls - successful_calls}/{num_calls}")
    
    if successful_calls > 0:
        avg_response_time = sum(r.get('response_time', 0) for r in results if r.get('success', False)) / successful_calls
        print(f"⚡ Avg response time: {avg_response_time:.1f}ms")
    
    print("\n📋 Detailed Results:")
    for r in results:
        status = "✅" if r.get('success', False) else "❌"
        print(f"{status} Call {r['call']}: {r['status_code']} ({r.get('response_time', 'N/A')}ms)")
    
    return results

def test_different_symbols():
    """Test different crypto symbols to see if it's symbol-specific"""
    symbols = ["X:BTCUSD", "X:ETHUSD", "X:ADAUSD"]
    
    print(f"\n🔍 Testing different symbols:")
    print("=" * 60)
    
    for symbol in symbols:
        print(f"\n🪙 Testing {symbol}:")
        url = f"{BASE_URL}/v2/aggs/ticker/{symbol}/prev?adjusted=true&apikey={API_KEY}"
        
        try:
            response = requests.get(url, timeout=5)
            print(f"   Status: {response.status_code}")
            
            if response.status_code == 200:
                data = response.json()
                print(f"   API Status: {data.get('status', 'UNKNOWN')}")
                if data.get('results'):
                    price = data['results'][0].get('c', 'N/A')
                    print(f"   Price: ${price}")
            else:
                try:
                    error = response.json()
                    print(f"   Error: {error}")
                except:
                    print(f"   Raw: {response.text[:100]}")
                    
        except Exception as e:
            print(f"   Exception: {e}")
        
        time.sleep(1)  # Small delay between symbols

if __name__ == "__main__":
    print("🚀 Direct Polygon.io API Test")
    print("Testing the EXACT same endpoint our Go backend uses")
    print("This will help identify if the issue is in our code or Polygon API")
    print("")
    
    # Test the main crypto symbol that's failing
    print("🎯 Primary Test: X:ETHUSD")
    test_polygon_endpoint("X:ETHUSD", num_calls=5, delay=2)
    
    # Test different symbols
    test_different_symbols()
    
    print("\n🎯 Conclusion:")
    print("If all calls succeed → Issue is in our Go backend code")
    print("If calls fail → Issue is with Polygon API/rate limits")
