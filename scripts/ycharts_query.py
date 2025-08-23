#!/usr/bin/env python3
"""
YCharts API Query Script

This script provides a comprehensive interface to query the YCharts API
using the pycharts library. It supports fetching company data, mutual fund data,
and economic indicators.

Requirements:
    pip install git+https://github.com/ycharts/pycharts.git
    pip install requests pandas python-dotenv

Usage:
    python ycharts_query.py --help
"""

import argparse
import datetime
import json
import os
import sys
from typing import List, Dict, Any, Optional
import logging

try:
    from pycharts import CompanyClient, MutualFundClient, IndicatorClient
except ImportError:
    print("Error: pycharts library not found. Install it with:")
    print("pip install git+https://github.com/ycharts/pycharts.git")
    sys.exit(1)

try:
    import pandas as pd
    PANDAS_AVAILABLE = True
except ImportError:
    PANDAS_AVAILABLE = False
    print("Warning: pandas not available. Install with 'pip install pandas' for better data formatting.")

try:
    from dotenv import load_dotenv
    load_dotenv()
    DOTENV_AVAILABLE = True
except ImportError:
    DOTENV_AVAILABLE = False


class YChartsAPIClient:
    """
    A comprehensive client for querying YCharts API
    """
    
    def __init__(self, api_key: str):
        """
        Initialize the YCharts API client
        
        Args:
            api_key (str): Your YCharts API access token
        """
        self.api_key = api_key
        self.company_client = CompanyClient(api_key)
        self.mutual_fund_client = MutualFundClient(api_key)
        self.indicator_client = IndicatorClient(api_key)
        
        # Setup logging
        logging.basicConfig(level=logging.INFO)
        self.logger = logging.getLogger(__name__)
    
    def get_company_latest_data(self, symbols: List[str], metrics: List[str]) -> Dict[str, Any]:
        """
        Get latest data points for companies
        
        Args:
            symbols (List[str]): List of company symbols (e.g., ['AAPL', 'GOOGL'])
            metrics (List[str]): List of metrics to fetch (e.g., ['price', 'market_cap'])
        
        Returns:
            Dict: API response data
        """
        try:
            self.logger.info(f"Fetching latest data for symbols: {symbols}, metrics: {metrics}")
            response = self.company_client.get_points(symbols, metrics)
            return response
        except Exception as e:
            self.logger.error(f"Error fetching company latest data: {e}")
            return {"error": str(e)}
    
    def get_company_historical_data(self, symbols: List[str], metrics: List[str], 
                                  start_date: datetime.datetime, 
                                  end_date: datetime.datetime) -> Dict[str, Any]:
        """
        Get historical data series for companies
        
        Args:
            symbols (List[str]): List of company symbols
            metrics (List[str]): List of metrics to fetch
            start_date (datetime): Start date for the data
            end_date (datetime): End date for the data
        
        Returns:
            Dict: API response data
        """
        try:
            self.logger.info(f"Fetching historical data for symbols: {symbols}, metrics: {metrics}")
            self.logger.info(f"Date range: {start_date.date()} to {end_date.date()}")
            response = self.company_client.get_series(
                symbols, metrics, 
                query_start_date=start_date, 
                query_end_date=end_date
            )
            return response
        except Exception as e:
            self.logger.error(f"Error fetching company historical data: {e}")
            return {"error": str(e)}
    
    def get_company_info(self, symbols: List[str], info_fields: List[str]) -> Dict[str, Any]:
        """
        Get company information
        
        Args:
            symbols (List[str]): List of company symbols
            info_fields (List[str]): List of info fields (e.g., ['description', 'sector'])
        
        Returns:
            Dict: API response data
        """
        try:
            self.logger.info(f"Fetching company info for symbols: {symbols}, fields: {info_fields}")
            response = self.company_client.get_info(symbols, info_fields)
            return response
        except Exception as e:
            self.logger.error(f"Error fetching company info: {e}")
            return {"error": str(e)}
    
    def get_mutual_fund_data(self, symbols: List[str], metrics: List[str]) -> Dict[str, Any]:
        """
        Get mutual fund data
        
        Args:
            symbols (List[str]): List of mutual fund symbols
            metrics (List[str]): List of metrics to fetch
        
        Returns:
            Dict: API response data
        """
        try:
            self.logger.info(f"Fetching mutual fund data for symbols: {symbols}, metrics: {metrics}")
            response = self.mutual_fund_client.get_points(symbols, metrics)
            return response
        except Exception as e:
            self.logger.error(f"Error fetching mutual fund data: {e}")
            return {"error": str(e)}
    
    def get_indicator_data(self, indicators: List[str], 
                          start_date: Optional[datetime.datetime] = None,
                          end_date: Optional[datetime.datetime] = None) -> Dict[str, Any]:
        """
        Get economic indicator data
        
        Args:
            indicators (List[str]): List of indicator symbols
            start_date (Optional[datetime]): Start date for historical data
            end_date (Optional[datetime]): End date for historical data
        
        Returns:
            Dict: API response data
        """
        try:
            self.logger.info(f"Fetching indicator data for: {indicators}")
            if start_date and end_date:
                response = self.indicator_client.get_series(
                    indicators, 
                    query_start_date=start_date, 
                    query_end_date=end_date
                )
            else:
                response = self.indicator_client.get_points(indicators)
            return response
        except Exception as e:
            self.logger.error(f"Error fetching indicator data: {e}")
            return {"error": str(e)}
    
    def format_response(self, response: Dict[str, Any], output_format: str = "json") -> str:
        """
        Format the API response for display
        
        Args:
            response (Dict): API response data
            output_format (str): Output format ('json', 'csv', 'table')
        
        Returns:
            str: Formatted response
        """
        if "error" in response:
            return f"Error: {response['error']}"
        
        if output_format == "json":
            return json.dumps(response, indent=2, default=str)
        
        elif output_format == "csv" and PANDAS_AVAILABLE:
            try:
                # Try to convert to DataFrame for CSV output
                if isinstance(response, dict) and 'data' in response:
                    df = pd.DataFrame(response['data'])
                    return df.to_csv(index=False)
                else:
                    return json.dumps(response, indent=2, default=str)
            except Exception:
                return json.dumps(response, indent=2, default=str)
        
        elif output_format == "table" and PANDAS_AVAILABLE:
            try:
                # Try to convert to DataFrame for table output
                if isinstance(response, dict) and 'data' in response:
                    df = pd.DataFrame(response['data'])
                    return df.to_string(index=False)
                else:
                    return json.dumps(response, indent=2, default=str)
            except Exception:
                return json.dumps(response, indent=2, default=str)
        
        else:
            return json.dumps(response, indent=2, default=str)


def main():
    """Main function to handle command line arguments and execute API queries"""
    
    parser = argparse.ArgumentParser(description="Query YCharts API for financial data")
    parser.add_argument("--api-key", type=str, help="YCharts API key (or set YCHARTS_API_KEY env var)")
    parser.add_argument("--symbols", type=str, nargs="+", required=True, 
                       help="List of symbols to query (e.g., AAPL GOOGL)")
    parser.add_argument("--metrics", type=str, nargs="+", default=["price"], 
                       help="List of metrics to fetch (default: price)")
    parser.add_argument("--query-type", type=str, choices=["latest", "historical", "info", "mutual_fund", "indicator"], 
                       default="latest", help="Type of query to perform")
    parser.add_argument("--start-date", type=str, help="Start date for historical data (YYYY-MM-DD)")
    parser.add_argument("--end-date", type=str, help="End date for historical data (YYYY-MM-DD)")
    parser.add_argument("--days-back", type=int, default=30, 
                       help="Number of days back for historical data (default: 30)")
    parser.add_argument("--output-format", type=str, choices=["json", "csv", "table"], 
                       default="json", help="Output format")
    parser.add_argument("--output-file", type=str, help="Output file path (optional)")
    parser.add_argument("--info-fields", type=str, nargs="+", default=["description"], 
                       help="Info fields for company info queries")
    
    args = parser.parse_args()
    
    # Get API key from argument or environment variable
    api_key = args.api_key or os.getenv("YCHARTS_API_KEY")
    if not api_key:
        print("Error: API key required. Use --api-key argument or set YCHARTS_API_KEY environment variable.")
        sys.exit(1)
    
    # Initialize client
    client = YChartsAPIClient(api_key)
    
    # Parse dates if provided
    start_date = None
    end_date = None
    
    if args.start_date:
        start_date = datetime.datetime.strptime(args.start_date, "%Y-%m-%d")
    if args.end_date:
        end_date = datetime.datetime.strptime(args.end_date, "%Y-%m-%d")
    
    # If no dates provided but historical data requested, use days_back
    if args.query_type == "historical" and not start_date:
        end_date = datetime.datetime.now()
        start_date = end_date - datetime.timedelta(days=args.days_back)
    
    # Execute query based on type
    response = {}
    
    if args.query_type == "latest":
        response = client.get_company_latest_data(args.symbols, args.metrics)
    
    elif args.query_type == "historical":
        if not start_date or not end_date:
            print("Error: Historical queries require start and end dates")
            sys.exit(1)
        response = client.get_company_historical_data(args.symbols, args.metrics, start_date, end_date)
    
    elif args.query_type == "info":
        response = client.get_company_info(args.symbols, args.info_fields)
    
    elif args.query_type == "mutual_fund":
        response = client.get_mutual_fund_data(args.symbols, args.metrics)
    
    elif args.query_type == "indicator":
        response = client.get_indicator_data(args.symbols, start_date, end_date)
    
    # Format and output response
    formatted_response = client.format_response(response, args.output_format)
    
    if args.output_file:
        with open(args.output_file, 'w') as f:
            f.write(formatted_response)
        print(f"Output saved to {args.output_file}")
    else:
        print(formatted_response)


if __name__ == "__main__":
    main()
