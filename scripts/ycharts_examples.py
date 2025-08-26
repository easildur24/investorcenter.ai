#!/usr/bin/env python3
"""
YCharts API Usage Examples

This file contains practical examples of how to use the YCharts API script
for common financial data queries.
"""

import datetime
import os

from ycharts_query import YChartsAPIClient


def example_latest_stock_prices():
    """Example: Get latest stock prices for major tech companies"""
    print("=== Example: Latest Stock Prices ===")

    # Initialize client (make sure to set your API key)
    api_key = os.getenv("YCHARTS_API_KEY", "your_api_key_here")
    client = YChartsAPIClient(api_key)

    # Get latest prices for tech stocks
    symbols = ["AAPL", "GOOGL", "MSFT", "AMZN", "TSLA"]
    metrics = ["price", "market_cap", "volume"]

    response = client.get_company_latest_data(symbols, metrics)
    formatted = client.format_response(response, "json")
    print(formatted)
    print("\n")


def example_historical_data():
    """Example: Get historical price data for the past 30 days"""
    print("=== Example: Historical Price Data ===")

    api_key = os.getenv("YCHARTS_API_KEY", "your_api_key_here")
    client = YChartsAPIClient(api_key)

    # Get 30 days of historical data
    end_date = datetime.datetime.now()
    start_date = end_date - datetime.timedelta(days=30)

    symbols = ["AAPL", "SPY"]  # Apple and S&P 500 ETF
    metrics = ["price"]

    response = client.get_company_historical_data(
        symbols, metrics, start_date, end_date
    )
    formatted = client.format_response(response, "json")
    print(formatted)
    print("\n")


def example_company_info():
    """Example: Get company information and descriptions"""
    print("=== Example: Company Information ===")

    api_key = os.getenv("YCHARTS_API_KEY", "your_api_key_here")
    client = YChartsAPIClient(api_key)

    symbols = ["AAPL", "GOOGL"]
    info_fields = ["description", "sector", "industry"]

    response = client.get_company_info(symbols, info_fields)
    formatted = client.format_response(response, "json")
    print(formatted)
    print("\n")


def example_mutual_funds():
    """Example: Get mutual fund data"""
    print("=== Example: Mutual Fund Data ===")

    api_key = os.getenv("YCHARTS_API_KEY", "your_api_key_here")
    client = YChartsAPIClient(api_key)

    # Common mutual fund symbols (you may need to adjust these based on YCharts availability)
    symbols = ["VTIAX", "VTSAX"]  # Vanguard funds
    metrics = ["price", "net_assets"]

    response = client.get_mutual_fund_data(symbols, metrics)
    formatted = client.format_response(response, "json")
    print(formatted)
    print("\n")


def example_economic_indicators():
    """Example: Get economic indicator data"""
    print("=== Example: Economic Indicators ===")

    api_key = os.getenv("YCHARTS_API_KEY", "your_api_key_here")
    client = YChartsAPIClient(api_key)

    # Economic indicators (adjust symbols based on YCharts availability)
    indicators = ["GDP", "UNEMPLOYMENT_RATE", "INFLATION_RATE"]

    response = client.get_indicator_data(indicators)
    formatted = client.format_response(response, "json")
    print(formatted)
    print("\n")


def run_all_examples():
    """Run all examples"""
    print("YCharts API Examples")
    print("=" * 50)
    print("Make sure to set your YCHARTS_API_KEY environment variable!")
    print("=" * 50)
    print()

    try:
        example_latest_stock_prices()
        example_historical_data()
        example_company_info()
        example_mutual_funds()
        example_economic_indicators()
    except Exception as e:
        print(f"Error running examples: {e}")
        print("Make sure you have:")
        print("1. Set your YCHARTS_API_KEY environment variable")
        print(
            "2. Installed required packages: pip install -r requirements.txt"
        )
        print("3. Valid YCharts API access")


if __name__ == "__main__":
    run_all_examples()
