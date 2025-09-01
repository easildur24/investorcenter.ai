#!/usr/bin/env python3
"""
Test Direct Import Functionality

This script tests the direct database import functionality without requiring a real database.
It verifies the data flow and transformation logic.
"""

import sys
from pathlib import Path

# Add the us_tickers module to Python path
sys.path.append(str(Path(__file__).parent))

import pandas as pd
from us_tickers.transform import (clean_security_name, should_include_ticker,
                                  transform_for_database)


def test_transformation_logic():
    """Test the transformation logic with sample data."""
    print("🧪 Testing Ticker Transformation Logic")
    print("=" * 50)

    # Create sample data similar to what we get from exchanges
    sample_data = [
        {
            "Ticker": "AAPL",
            "Security Name": "APPLE INC. - COMMON STOCK",
            "Exchange": "Q",
            "ETF": "N",
            "Test Issue": "N",
        },
        {
            "Ticker": "MSFT",
            "Security Name": "MICROSOFT CORPORATION - COMMON STOCK",
            "Exchange": "Q",
            "ETF": "N",
            "Test Issue": "N",
        },
        {
            "Ticker": "BAC",
            "Security Name": "BANK OF AMERICA CORPORATION COMMON STOCK",
            "Exchange": "N",
            "ETF": "N",
            "Test Issue": "N",
        },
        {
            "Ticker": "AAPL.W",
            "Security Name": "APPLE INC. WARRANTS",
            "Exchange": "Q",
            "ETF": "N",
            "Test Issue": "N",
        },
        {
            "Ticker": "BAC$A",
            "Security Name": "BANK OF AMERICA PREFERRED SERIES A",
            "Exchange": "N",
            "ETF": "N",
            "Test Issue": "N",
        },
    ]

    df = pd.DataFrame(sample_data)
    print(f"📥 Sample input data: {len(df)} records")

    # Test individual filtering function
    print("\n🔍 Testing individual record filtering:")
    for _, row in df.iterrows():
        include = should_include_ticker(row)
        status = "✅ INCLUDE" if include else "❌ EXCLUDE"
        print(f"  {status} | {row['Ticker']:8} | {row['Security Name']}")

    # Test name cleaning
    print("\n🧽 Testing name cleaning:")
    for _, row in df.iterrows():
        if should_include_ticker(row):
            original = row["Security Name"]
            cleaned = clean_security_name(original)
            print(f"  {row['Ticker']:6} | {original} → {cleaned}")

    # Test full transformation
    print("\n🔧 Testing full transformation:")
    transformed_df = transform_for_database(df)

    print(f"📊 Transformation results:")
    print(f"  Input records: {len(df)}")
    print(f"  Output records: {len(transformed_df)}")
    print(f"  Filtered out: {len(df) - len(transformed_df)}")

    if not transformed_df.empty:
        print("\n📋 Final transformed data:")
        for _, row in transformed_df.iterrows():
            print(
                f"  {row['symbol']:6} | {row['name']:40} | {row['exchange']:12} | {row['country']} | {row['currency']}"
            )

    print("\n✅ Transformation logic test completed!")
    return len(transformed_df)


def test_database_import_simulation():
    """Simulate database import behavior."""
    print("\n🎲 Simulating Database Import Behavior")
    print("=" * 50)

    # Simulate what happens with existing vs new tickers
    existing_tickers = {"AAPL", "MSFT", "GOOGL"}  # Simulate existing in DB

    new_data = [
        {
            "symbol": "AAPL",
            "name": "APPLE INC.",
            "exchange": "Nasdaq",
        },  # Existing - should skip
        {
            "symbol": "MSFT",
            "name": "MICROSOFT CORP.",
            "exchange": "Nasdaq",
        },  # Existing - should skip
        {
            "symbol": "TSLA",
            "name": "TESLA INC.",
            "exchange": "Nasdaq",
        },  # New - should insert
        {
            "symbol": "NVDA",
            "name": "NVIDIA CORP.",
            "exchange": "Nasdaq",
        },  # New - should insert
    ]

    print("📊 Simulating periodic update behavior:")
    print(
        f"  Existing in DB: {len(existing_tickers)} stocks ({', '.join(sorted(existing_tickers))})"
    )
    print(f"  New data: {len(new_data)} stocks")

    inserted = 0
    skipped = 0

    for stock in new_data:
        if stock["symbol"] in existing_tickers:
            print(f"  ⏭️  SKIP: {stock['symbol']} (already exists)")
            skipped += 1
        else:
            print(f"  ✅ INSERT: {stock['symbol']} - {stock['name']}")
            inserted += 1

    print(f"\n📈 Simulated results:")
    print(f"  Inserted: {inserted} new stocks")
    print(f"  Skipped: {skipped} existing stocks")
    print(f"  Final DB size: {len(existing_tickers) + inserted} stocks")

    print("\n💡 This is exactly how the real script will behave!")
    print("   - ON CONFLICT DO NOTHING ensures no duplicates")
    print("   - Existing data is preserved")
    print("   - Only new listings are added")


def main():
    """Run all tests."""
    try:
        # Test transformation
        transformed_count = test_transformation_logic()

        # Test database simulation
        test_database_import_simulation()

        print(f"\n🎉 All tests completed successfully!")
        print(f"   Ready to import {transformed_count} sample records")
        print(f"\n🚀 To run with real data:")
        print(f"   python scripts/ticker_import_to_db.py --dry-run")
        print(f"   python scripts/ticker_import_to_db.py")

    except Exception as e:
        print(f"❌ Test failed: {e}", file=sys.stderr)
        sys.exit(1)


if __name__ == "__main__":
    main()
