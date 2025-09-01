#!/usr/bin/env python3
"""
Tests for ticker_db_importer.py

Comprehensive test suite to validate ticker data transformation logic.
"""

import unittest
import pandas as pd
import tempfile
import os
import sys
from pathlib import Path

# Import the functions we want to test
from ticker_db_importer import (
    clean_security_name,
    should_include_ticker,
    transform_ticker_data,
    generate_sql_inserts,
    EXCHANGE_CODES
)


class TestCleanSecurityName(unittest.TestCase):
    """Test the clean_security_name function."""

    def test_basic_cleaning(self) -> None:
        """Test basic name cleaning."""
        # Remove common suffixes
        self.assertEqual(
            clean_security_name("APPLE INC. - COMMON STOCK"),
            "APPLE INC."
        )

        # Remove quotes
        self.assertEqual(
            clean_security_name('"MICROSOFT CORPORATION"'),
            "MICROSOFT CORPORATION"
        )

        # Handle empty/None
        self.assertEqual(clean_security_name(""), "")
        self.assertEqual(clean_security_name(None), "")

    def test_suffix_removal(self) -> None:
        """Test removal of various suffixes."""
        test_cases = [
            ("TESLA, INC. - COMMON STOCK", "TESLA, INC."),
            ("AMAZON.COM, INC. CLASS A COMMON STOCK", "AMAZON.COM, INC."),
            ("NVIDIA CORPORATION - ORDINARY SHARES", "NVIDIA CORPORATION"),
            ("ALIBABA GROUP - AMERICAN DEPOSITARY SHARES", "ALIBABA GROUP"),
        ]

        for input_name, expected in test_cases:
            with self.subTest(input_name=input_name):
                self.assertEqual(clean_security_name(input_name), expected)

    def test_whitespace_cleanup(self) -> None:
        """Test whitespace and punctuation cleanup."""
        self.assertEqual(
            clean_security_name("  COMPANY   NAME  .  "),
            "COMPANY NAME"
        )

        self.assertEqual(
            clean_security_name("MULTIPLE    SPACES"),
            "MULTIPLE SPACES"
        )

    def test_corporate_endings(self) -> None:
        """Test proper handling of corporate endings."""
        test_cases = [
            ("ACME INC", "ACME INC."),
            ("BETA CORP", "BETA CORP."),
            ("GAMMA LTD", "GAMMA LTD."),
            ("DELTA CORPORATION", "DELTA CORPORATION"),  # No change needed
        ]

        for input_name, expected in test_cases:
            with self.subTest(input_name=input_name):
                self.assertEqual(clean_security_name(input_name), expected)


class TestShouldIncludeTicker(unittest.TestCase):
    """Test the should_include_ticker function."""

    def test_include_regular_stocks(self) -> None:
        """Test that regular stocks are included."""
        regular_stocks = [
            {"Ticker": "AAPL", "Security Name": "APPLE INC. - COMMON STOCK"},
            {"Ticker": "MSFT", "Security Name": "MICROSOFT CORPORATION"},
            {"Ticker": "GOOGL", "Security Name": "ALPHABET INC. - CLASS A COMMON STOCK"},
        ]

        for stock in regular_stocks:
            row = pd.Series(stock)
            with self.subTest(ticker=stock["Ticker"]):
                self.assertTrue(should_include_ticker(row))

    def test_exclude_warrants(self) -> None:
        """Test that warrants are excluded."""
        warrants = [
            {"Ticker": "AAPL.W", "Security Name": "APPLE INC. WARRANT"},
            {"Ticker": "MSFTW", "Security Name": "MICROSOFT WARRANT"},
            {"Ticker": "GOOGL.WS", "Security Name": "ALPHABET WARRANT"},
        ]

        for warrant in warrants:
            row = pd.Series(warrant)
            with self.subTest(ticker=warrant["Ticker"]):
                self.assertFalse(should_include_ticker(row))

    def test_exclude_preferred_stocks(self) -> None:
        """Test that preferred stocks are excluded."""
        preferred_stocks = [
            {"Ticker": "BACprA", "Security Name": "BANK OF AMERICA PREFERRED SERIES A"},
            {"Ticker": "JPMPD", "Security Name": "JP MORGAN PREFERRED STOCK"},
            {"Ticker": "WFCPA", "Security Name": "WELLS FARGO DEPOSITARY SHARES"},
        ]

        for preferred in preferred_stocks:
            row = pd.Series(preferred)
            with self.subTest(ticker=preferred["Ticker"]):
                self.assertFalse(should_include_ticker(row))

    def test_exclude_special_characters(self) -> None:
        """Test that tickers with special characters are excluded."""
        special_tickers = [
            {"Ticker": "BRK.A", "Security Name": "BERKSHIRE HATHAWAY CLASS A"},
            {"Ticker": "BRK$B", "Security Name": "BERKSHIRE HATHAWAY PREFERRED"},
            {"Ticker": "TEST^", "Security Name": "TEST COMPANY"},
        ]

        for ticker in special_tickers:
            row = pd.Series(ticker)
            with self.subTest(ticker=ticker["Ticker"]):
                self.assertFalse(should_include_ticker(row))

    def test_exclude_derivatives(self) -> None:
        """Test that derivatives are excluded by security name."""
        derivatives = [
            {"Ticker": "TESTX", "Security Name": "TEST COMPANY WARRANTS"},
            {"Ticker": "TESTY", "Security Name": "TEST COMPANY RIGHTS"},
            {"Ticker": "TESTZ", "Security Name": "TEST COMPANY UNITS"},
            {"Ticker": "TESTA", "Security Name": "TEST COMPANY NOTES DUE 2030"},
        ]

        for derivative in derivatives:
            row = pd.Series(derivative)
            with self.subTest(ticker=derivative["Ticker"]):
                self.assertFalse(should_include_ticker(row))

    def test_handle_empty_values(self) -> None:
        """Test handling of empty/null values."""
        empty_cases = [
            {"Ticker": "", "Security Name": "COMPANY NAME"},
            {"Ticker": None, "Security Name": "COMPANY NAME"},
        ]

        for case in empty_cases:
            row = pd.Series(case)
            with self.subTest(ticker=case["Ticker"]):
                self.assertFalse(should_include_ticker(row))


class TestTransformTickerData(unittest.TestCase):
    """Test the transform_ticker_data function."""

    def setUp(self) -> None:
        """Set up test data."""
        self.test_data = [
            # Valid stocks to include
            ["AAPL", "APPLE INC. - COMMON STOCK", "Q", "N", "N", "100", "N", "N", "Q", "", ""],
            ["MSFT", "MICROSOFT CORPORATION", "Q", "N", "N", "100", "N", "N", "Q", "", ""],
            ["BAC", "BANK OF AMERICA CORPORATION", "", "N", "", "100", "N", "", "N", "BAC", "BAC"],

            # Should be filtered out
            ["AAPL.W", "APPLE INC. WARRANT", "Q", "N", "N", "100", "N", "N", "Q", "", ""],
            ["BAC$A", "BANK OF AMERICA PREFERRED SERIES A", "", "N", "", "100", "N", "", "N", "BACPA", "BAC-A"],
        ]

        self.columns = [
            "Ticker", "Security Name", "Market Category", "Test Issue",
            "Financial Status", "Round Lot Size", "ETF", "NextShares",
            "Exchange", "CQS Symbol", "NASDAQ Symbol"
        ]

    def test_transform_with_sample_data(self) -> None:
        """Test transformation with sample data."""
        # Create temporary CSV file
        with tempfile.NamedTemporaryFile(mode='w', suffix='.csv', delete=False) as f:
            # Write header
            f.write(",".join(self.columns) + "\n")

            # Write test data
            for row in self.test_data:
                f.write(",".join(str(x) for x in row) + "\n")

            temp_file = f.name

        try:
            # Transform the data
            result_df = transform_ticker_data(temp_file)

            # Should have 3 records (2 filtered out)
            self.assertEqual(len(result_df), 3)

            # Check required columns exist
            expected_columns = ['symbol', 'name', 'exchange', 'country', 'currency']
            for col in expected_columns:
                self.assertIn(col, result_df.columns)

            # Check exchange mapping
            nasdaq_count = len(result_df[result_df['exchange'] == 'Nasdaq'])
            nyse_count = len(result_df[result_df['exchange'] == 'NYSE'])
            self.assertEqual(nasdaq_count, 2)  # AAPL, MSFT
            self.assertEqual(nyse_count, 1)    # BAC

            # Check country and currency defaults
            self.assertTrue(all(result_df['country'] == 'US'))
            self.assertTrue(all(result_df['currency'] == 'USD'))

        finally:
            # Clean up temporary file
            os.unlink(temp_file)


class TestGenerateSqlInserts(unittest.TestCase):
    """Test the generate_sql_inserts function."""

    def test_sql_generation(self) -> None:
        """Test SQL generation with sample data."""
        test_data = pd.DataFrame([
            {
                'symbol': 'AAPL',
                'name': 'APPLE INC.',
                'exchange': 'Nasdaq',
                'sector': None,
                'industry': None,
                'country': 'US',
                'currency': 'USD',
                'market_cap': None,
                'description': None,
                'website': None,
            },
            {
                'symbol': 'MSFT',
                'name': "MICROSOFT CORP.",
                'exchange': 'Nasdaq',
                'sector': None,
                'industry': None,
                'country': 'US',
                'currency': 'USD',
                'market_cap': None,
                'description': None,
                'website': None,
            }
        ])

        sql = generate_sql_inserts(test_data)

        # Check that SQL contains expected elements
        self.assertIn("INSERT INTO stocks", sql)
        self.assertIn("'AAPL'", sql)
        self.assertIn("'MSFT'", sql)
        self.assertIn("'APPLE INC.'", sql)
        self.assertIn("'Nasdaq'", sql)
        self.assertIn("ON CONFLICT (symbol) DO NOTHING", sql)

    def test_sql_escaping(self) -> None:
        """Test that single quotes are properly escaped in SQL."""
        test_data = pd.DataFrame([
            {
                'symbol': 'TEST',
                'name': "O'REILLY AUTOMOTIVE",  # Contains single quote
                'exchange': 'NYSE',
                'sector': None,
                'industry': None,
                'country': 'US',
                'currency': 'USD',
                'market_cap': None,
                'description': None,
                'website': None,
            }
        ])

        sql = generate_sql_inserts(test_data)

        # Should escape single quote as double single quote
        self.assertIn("O''REILLY AUTOMOTIVE", sql)


class TestExchangeMapping(unittest.TestCase):
    """Test exchange code mappings."""

    def test_exchange_codes_complete(self) -> None:
        """Test that all expected exchange codes are defined."""
        expected_codes = {"Q", "N", "A", "P", "Z"}
        self.assertEqual(set(EXCHANGE_CODES.keys()), expected_codes)

    def test_exchange_names(self) -> None:
        """Test exchange code to name mapping."""
        expected_mappings = {
            "Q": "Nasdaq",
            "N": "NYSE",
            "A": "NYSE American",
            "P": "NYSE Arca",
            "Z": "Cboe",
        }

        self.assertEqual(EXCHANGE_CODES, expected_mappings)


def run_tests() -> None:
    """Run all tests."""
    # Create test suite
    loader = unittest.TestLoader()
    suite = unittest.TestSuite()

    # Add test classes
    test_classes = [
        TestCleanSecurityName,
        TestShouldIncludeTicker,
        TestTransformTickerData,
        TestGenerateSqlInserts,
        TestExchangeMapping,
    ]

    for test_class in test_classes:
        tests = loader.loadTestsFromTestCase(test_class)
        suite.addTests(tests)

    # Run tests
    runner = unittest.TextTestRunner(verbosity=2)
    result = runner.run(suite)

    return result.wasSuccessful()


if __name__ == "__main__":
    success = run_tests()
    sys.exit(0 if success else 1)
