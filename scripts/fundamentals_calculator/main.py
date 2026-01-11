#!/usr/bin/env python3
"""
Fundamentals Calculator - Main Entry Point
Usage: python main.py TICKER
"""

import sys
from metrics import MetricsCalculator

def main():
    """Main entry point"""
    if len(sys.argv) != 2:
        print("Usage: python main.py TICKER")
        print("Example: python main.py AAPL")
        sys.exit(1)
    
    ticker = sys.argv[1].upper()
    
    print(f"🚀 CALCULATING FUNDAMENTALS FOR {ticker}")
    print("=" * 50)
    
    # Initialize calculator
    calculator = MetricsCalculator()
    
    # Calculate metrics
    results = calculator.calculate_all_metrics(ticker)
    
    if results:
        print(f"\n🎯 RESULTS FOR {ticker}:")
        print("-" * 30)
        
        for metric_name, value in results.items():
            if value is not None:
                if isinstance(value, (int, float)):
                    if abs(value) > 1000000000:
                        print(f"{metric_name}: ${value/1000000000:.2f}B")
                    else:
                        print(f"{metric_name}: ${value/1000000:.1f}M")
                else:
                    print(f"{metric_name}: {value}")
            else:
                print(f"{metric_name}: NOT CALCULATED")
        
        print(f"\n✅ Successfully calculated metrics for {ticker}")
    else:
        print(f"❌ Failed to calculate metrics for {ticker}")

if __name__ == "__main__":
    main()

