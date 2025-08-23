#!/usr/bin/env python3
"""
YCharts API Client - Final Version

This script is ready to use once you get the correct authentication 
details from YCharts support.

Usage:
    python3 ycharts_final.py --symbols AAPL --metrics price
"""

import os
import requests
import json
import argparse
from dotenv import load_dotenv

load_dotenv()

class YChartsClient:
    """YCharts API Client"""
    
    def __init__(self, api_key=None):
        self.api_key = api_key or os.getenv("YCHARTS_API_KEY")
        if not self.api_key:
            raise ValueError("API key required. Set YCHARTS_API_KEY in .env file.")
        
        # Base URL discovered from our testing
        self.base_url = "https://ycharts.com/api/v1"
        
        # Default headers - update these based on YCharts support guidance
        self.headers = {
            'Authorization': f'Bearer {self.api_key}',  # Update auth method as needed
            'Content-Type': 'application/json',
            'User-Agent': 'YCharts-Python-Client/1.0'
        }
    
    def get_series_data(self, security_id, metric_ids):
        """
        Get series data for a security
        
        Args:
            security_id (str): Security identifier (e.g., 'AAPL')
            metric_ids (str or list): Metric IDs (e.g., '1day_return,52_week_high')
        
        Returns:
            dict: API response data
        """
        if isinstance(metric_ids, list):
            metric_ids = ','.join(metric_ids)
        
        params = {
            'security_id': security_id,
            'metric_ids': metric_ids
        }
        
        return self._make_request('/series', params)
    
    def get_company_data(self, symbol, metrics=None):
        """
        Get company data
        
        Args:
            symbol (str): Company symbol (e.g., 'AAPL')
            metrics (list): List of metrics to fetch
        
        Returns:
            dict: API response data
        """
        endpoint = f'/companies/{symbol}'
        params = {}
        
        if metrics:
            params['metrics'] = ','.join(metrics) if isinstance(metrics, list) else metrics
        
        return self._make_request(endpoint, params)
    
    def _make_request(self, endpoint, params=None):
        """
        Make a request to the YCharts API
        
        Args:
            endpoint (str): API endpoint
            params (dict): Query parameters
        
        Returns:
            dict: API response or error information
        """
        url = f"{self.base_url}{endpoint}"
        
        try:
            response = requests.get(
                url,
                headers=self.headers,
                params=params or {},
                timeout=30
            )
            
            # Log the request for debugging
            print(f"🌐 Request: {response.url}")
            print(f"📊 Status: {response.status_code}")
            
            if response.status_code == 200:
                return response.json()
            elif response.status_code == 401:
                return {"error": "Unauthorized - check your API key"}
            elif response.status_code == 403:
                return {"error": "Forbidden - check account permissions or contact YCharts support"}
            elif response.status_code == 404:
                return {"error": "Not found - check endpoint URL"}
            elif response.status_code == 429:
                return {"error": "Rate limited - too many requests"}
            else:
                return {
                    "error": f"HTTP {response.status_code}",
                    "message": response.text[:500]
                }
                
        except requests.exceptions.Timeout:
            return {"error": "Request timeout"}
        except requests.exceptions.ConnectionError:
            return {"error": "Connection error"}
        except Exception as e:
            return {"error": f"Request failed: {str(e)}"}
    
    def test_connection(self):
        """Test the API connection"""
        print("🔍 Testing YCharts API connection...")
        print(f"🔑 API Key: {self.api_key[:10]}...")
        print(f"🌐 Base URL: {self.base_url}")
        
        # Try a simple request
        result = self.get_series_data('AAPL', '1day_return')
        
        if 'error' in result:
            print(f"❌ Connection failed: {result['error']}")
            print("\n💡 Next steps:")
            print("1. Contact YCharts support with your API key")
            print("2. Verify account permissions and status")
            print("3. Check if IP whitelisting is required")
            print("4. Ask for correct authentication method")
            return False
        else:
            print("✅ Connection successful!")
            print("📈 Sample data:")
            print(json.dumps(result, indent=2)[:500])
            return True

def main():
    """Main function for command line usage"""
    parser = argparse.ArgumentParser(description="YCharts API Client")
    parser.add_argument("--symbols", nargs="+", default=["AAPL"], help="Symbols to query")
    parser.add_argument("--metrics", nargs="+", default=["1day_return"], help="Metrics to fetch")
    parser.add_argument("--test", action="store_true", help="Test connection only")
    
    args = parser.parse_args()
    
    try:
        client = YChartsClient()
        
        if args.test:
            client.test_connection()
        else:
            for symbol in args.symbols:
                print(f"\n📊 Fetching data for {symbol}...")
                result = client.get_series_data(symbol, args.metrics)
                
                if 'error' in result:
                    print(f"❌ Error: {result['error']}")
                else:
                    print("✅ Success!")
                    print(json.dumps(result, indent=2))
                    
    except Exception as e:
        print(f"❌ Error: {e}")

if __name__ == "__main__":
    main()
