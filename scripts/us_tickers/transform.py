"""Data transformation utilities for ticker data."""

import logging
import re
from typing import Dict

import pandas as pd

from .fetch import EXCHANGE_CODES

logger = logging.getLogger(__name__)


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


def transform_for_database(df: pd.DataFrame) -> pd.DataFrame:
    """
    Transform ticker DataFrame for database import.

    Args:
        df: Raw ticker DataFrame from exchange

    Returns:
        Transformed DataFrame ready for database import
    """
    logger.info(f"Transforming {len(df)} ticker records for database import")

    # Filter to include only relevant tickers
    df_filtered = df[df.apply(should_include_ticker, axis=1)].copy()
    logger.info(f"After filtering: {len(df_filtered)} records")

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

    logger.info(f"Transformation complete: {len(result_df)} records ready")
    if len(result_df) > 0:
        exchange_counts = result_df["exchange"].value_counts()
        for exchange, count in exchange_counts.items():
            logger.info(f"  {exchange}: {count} stocks")

    return result_df
