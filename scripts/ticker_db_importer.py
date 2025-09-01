#!/usr/bin/env python3
"""
Ticker Database Importer

This script transforms the downloaded ticker CSV data to match the database schema
and prepares it for insertion into the PostgreSQL stocks table.
"""

import argparse
import re
import sys
from pathlib import Path
from typing import Dict, List, Optional

import pandas as pd

# Exchange code mapping from fetch.py
EXCHANGE_CODES = {
    "Q": "Nasdaq",
    "N": "NYSE",
    "A": "NYSE American",
    "P": "NYSE Arca",
    "Z": "Cboe",
}


def clean_security_name(name: str) -> str:
    """
    Clean security name by removing unnecessary text and formatting.

    Args:
        name: Raw security name from CSV

    Returns:
        Cleaned security name
    """
    if not name:
        return ""

    # Remove quotes
    name = name.strip('"')

    # Remove common suffixes that don't add value (order matters - longer first)
    suffixes_to_remove = [
        " - CLASS A COMMON STOCK",
        " CLASS A COMMON STOCK",
        " - CLASS B COMMON STOCK",
        " CLASS B COMMON STOCK",
        " - COMMON STOCK",
        " COMMON STOCK",
        " - ORDINARY SHARES",
        " ORDINARY SHARES",
        " - AMERICAN DEPOSITARY SHARES",
        " AMERICAN DEPOSITARY SHARES",
        " - ADS",
        " ADS",
    ]

    # Apply suffix removal (case insensitive)
    name_upper = name.upper()
    for suffix in suffixes_to_remove:
        if name_upper.endswith(suffix.upper()):
            name = name[: len(name) - len(suffix)]
            break

    # Clean up extra whitespace and trailing punctuation
    name = re.sub(r"\s+", " ", name.strip())
    name = name.rstrip(" .,")

    # Handle special cases
    if name.endswith(" INC"):
        name = name + "."
    elif name.endswith(" CORP"):
        name = name + "."
    elif name.endswith(" LTD"):
        name = name + "."

    return name


def should_include_ticker(row: pd.Series) -> bool:
    """
    Determine if a ticker should be included in the database.

    Args:
        row: DataFrame row containing ticker data

    Returns:
        True if ticker should be included
    """
    ticker = row["Ticker"]
    security_name = row["Security Name"]

    # Skip if no ticker
    if not ticker or pd.isna(ticker):
        return False

    # Skip tickers with special characters (warrants, rights, units, etc.)
    if any(char in ticker for char in [".", "$", "^", "+"]):
        return False

    # Skip if ticker contains common warrant/rights indicators
    warrant_indicators = ["W", "R", "U", "WS", "RT", "WT"]
    if any(ticker.endswith(indicator) for indicator in warrant_indicators):
        return False

    # Skip if security name indicates derivatives
    derivative_indicators = [
        "WARRANT",
        "RIGHTS",
        "UNITS",
        "PREFERRED",
        "NOTES",
        "TRUST",
        "DEPOSITARY SHARES",
        "SUBORDINATED",
        "CUMULATIVE",
    ]
    security_upper = security_name.upper()
    if any(indicator in security_upper for indicator in derivative_indicators):
        return False

    # Include everything else
    return True


def transform_ticker_data(csv_path: str) -> pd.DataFrame:
    """
    Transform ticker CSV data to match database schema.

    Args:
        csv_path: Path to the ticker CSV file

    Returns:
        DataFrame ready for database insertion
    """
    print(f"Reading ticker data from: {csv_path}")
    df = pd.read_csv(csv_path)

    print(f"Original dataset: {len(df)} records")

    # Filter to include only relevant tickers
    df_filtered = df[df.apply(should_include_ticker, axis=1)].copy()
    print(f"After filtering: {len(df_filtered)} records")

    # Transform the data
    transformed_data = []

    for _, row in df_filtered.iterrows():
        ticker = row["Ticker"].strip().upper()
        raw_name = row["Security Name"]
        exchange_code = row["Exchange"]

        # Clean the security name
        clean_name = clean_security_name(raw_name)

        # Convert exchange code to full name
        exchange_name = EXCHANGE_CODES.get(exchange_code, exchange_code)

        # Create the record for database insertion
        record = {
            "symbol": ticker,
            "name": clean_name,
            "exchange": exchange_name,
            "sector": None,  # To be populated later
            "industry": None,  # To be populated later
            "country": "US",
            "currency": "USD",
            "market_cap": None,  # To be populated later
            "description": None,  # To be populated later
            "website": None,  # To be populated later
        }

        transformed_data.append(record)

    result_df = pd.DataFrame(transformed_data)

    print(f"\nTransformation summary:")
    print(f"- Total records: {len(result_df)}")
    print(f"- Exchange distribution:")
    print(result_df["exchange"].value_counts())

    return result_df


def generate_sql_inserts(
    df: pd.DataFrame, output_file: Optional[str] = None
) -> str:
    """
    Generate SQL INSERT statements for the transformed data.

    Args:
        df: Transformed DataFrame
        output_file: Optional file path to save SQL

    Returns:
        SQL INSERT statements as string
    """
    sql_statements = []

    # Add header comment
    sql_statements.append("-- Ticker data import")
    sql_statements.append("-- Generated from demo_tickers.csv")
    sql_statements.append("")

    # Create INSERT statements in batches of 100 for performance
    batch_size = 100
    total_batches = (len(df) + batch_size - 1) // batch_size

    for batch_num in range(total_batches):
        start_idx = batch_num * batch_size
        end_idx = min((batch_num + 1) * batch_size, len(df))
        batch_df = df.iloc[start_idx:end_idx]

        sql_statements.append(f"-- Batch {batch_num + 1}/{total_batches}")
        sql_statements.append(
            "INSERT INTO stocks (symbol, name, exchange, sector, industry, country, currency, market_cap, description, website)"
        )
        sql_statements.append("VALUES")

        values = []
        for _, row in batch_df.iterrows():
            # Escape single quotes in names
            clean_name = row["name"].replace("'", "''")
            value = f"('{row['symbol']}', '{clean_name}', '{row['exchange']}', NULL, NULL, '{row['country']}', '{row['currency']}', NULL, NULL, NULL)"
            values.append(value)

        sql_statements.append(",\n  ".join(values))
        sql_statements.append("ON CONFLICT (symbol) DO NOTHING;")
        sql_statements.append("")

    sql_content = "\n".join(sql_statements)

    if output_file:
        with open(output_file, "w") as f:
            f.write(sql_content)
        print(f"SQL statements saved to: {output_file}")

    return sql_content


def preview_data(df: pd.DataFrame, num_samples: int = 10) -> None:
    """Preview the transformed data."""
    print(f"\nPreview of transformed data (first {num_samples} records):")
    print("-" * 80)

    for i, (_, row) in enumerate(df.head(num_samples).iterrows()):
        print(
            f"{i+1:2d}. {row['symbol']:6s} | {row['name']:50s} | {row['exchange']}"
        )

    print("-" * 80)


def main():
    """Main function to run the ticker transformation."""
    parser = argparse.ArgumentParser(
        description="Transform ticker CSV for database import"
    )
    parser.add_argument(
        "--csv",
        default="scripts/us_tickers/demo_tickers.csv",
        help="Path to ticker CSV file",
    )
    parser.add_argument(
        "--output-sql",
        default="scripts/ticker_import.sql",
        help="Output SQL file path",
    )
    parser.add_argument(
        "--output-csv",
        default="scripts/transformed_tickers.csv",
        help="Output transformed CSV file path",
    )
    parser.add_argument(
        "--preview-only",
        action="store_true",
        help="Only preview data without generating files",
    )

    args = parser.parse_args()

    try:
        # Transform the data
        df_transformed = transform_ticker_data(args.csv)

        # Preview the data
        preview_data(df_transformed)

        if not args.preview_only:
            # Save transformed CSV
            df_transformed.to_csv(args.output_csv, index=False)
            print(f"Transformed CSV saved to: {args.output_csv}")

            # Generate SQL statements
            generate_sql_inserts(df_transformed, args.output_sql)

            print(f"\nFiles generated:")
            print(f"- Transformed CSV: {args.output_csv}")
            print(f"- SQL import file: {args.output_sql}")
            print(f"\nTotal tickers ready for import: {len(df_transformed)}")

    except Exception as e:
        print(f"Error: {e}")
        sys.exit(1)


if __name__ == "__main__":
    main()
