"""Unit tests for the fetch module."""

import pytest
import pandas as pd
from unittest.mock import Mock, patch, mock_open
import requests

import sys
import os
sys.path.insert(0, os.path.join(os.path.dirname(__file__), '..'))

from us_tickers.fetch import (
    get_exchange_listed_tickers,
    _parse_nasdaq_listed,
    _parse_other_listed,
    _filter_and_clean_data,
    _create_session_with_retries,
    EXCHANGE_CODES,
)

# Import RuntimeError for error handling tests
from builtins import RuntimeError


class TestExchangeCodes:
    """Test exchange codes constants."""
    
    def test_exchange_codes_exist(self):
        """Test that expected exchange codes are defined."""
        assert "Q" in EXCHANGE_CODES
        assert "N" in EXCHANGE_CODES
        assert "A" in EXCHANGE_CODES
        assert "P" in EXCHANGE_CODES
        assert "Z" in EXCHANGE_CODES
    
    def test_exchange_codes_values(self):
        """Test exchange code descriptions."""
        assert EXCHANGE_CODES["Q"] == "Nasdaq"
        assert EXCHANGE_CODES["N"] == "NYSE"
        assert EXCHANGE_CODES["A"] == "NYSE American"


class TestSessionCreation:
    """Test session creation with retries."""
    
    def test_create_session_with_retries(self):
        """Test session creation returns valid session."""
        session = _create_session_with_retries()
        assert isinstance(session, requests.Session)
        assert session.timeout == 20


class TestNasdaqListedParsing:
    """Test parsing of nasdaqlisted.txt content."""
    
    def test_parse_nasdaq_listed_normal(self):
        """Test normal parsing of nasdaqlisted.txt."""
        content = """Symbol|Security Name|Market Category|Test Issue|Financial Status|Round Lot Size|ETF|NextShares
AAPL|APPLE INC|Q|N|N|100|N|N
MSFT|MICROSOFT CORP|Q|N|N|100|N|N
QQQ|INVESCO QQQ TRUST|Q|N|N|100|Y|N"""
        
        df = _parse_nasdaq_listed(content)
        
        assert len(df) == 3
        assert "Ticker" in df.columns
        assert "Exchange" in df.columns
        assert df.iloc[0]["Ticker"] == "AAPL"
        assert df.iloc[0]["Exchange"] == "Q"
        assert df.iloc[0]["ETF"] == "N"
    
    def test_parse_nasdaq_listed_with_footer(self):
        """Test parsing with footer row that should be removed."""
        content = """Symbol|Security Name|Market Category|Test Issue|Financial Status|Round Lot Size|ETF|NextShares
AAPL|APPLE INC|Q|N|N|100|N|N
File Creation Time: 01/01/2024 09:00:00"""
        
        df = _parse_nasdaq_listed(content)
        
        assert len(df) == 1
        assert df.iloc[0]["Ticker"] == "AAPL"
    
    def test_parse_nasdaq_listed_empty_content(self):
        """Test parsing empty content."""
        df = _parse_nasdaq_listed("")
        assert len(df) == 0
    
    def test_parse_nasdaq_listed_malformed(self):
        """Test parsing malformed content."""
        content = """Invalid line without pipes
Another invalid line"""
        
        df = _parse_nasdaq_listed(content)
        assert len(df) == 0
    
    def test_parse_nasdaq_listed_normalization(self):
        """Test string normalization."""
        content = """Symbol|Security Name|Market Category|Test Issue|Financial Status|Round Lot Size|ETF|NextShares
  aapl  |  APPLE INC  |Q|N|N|100|N|N"""
        
        df = _parse_nasdaq_listed(content)
        
        assert df.iloc[0]["Ticker"] == "AAPL"
        assert df.iloc[0]["Security Name"] == "APPLE INC"


class TestOtherListedParsing:
    """Test parsing of otherlisted.txt content."""
    
    def test_parse_other_listed_normal(self):
        """Test normal parsing of otherlisted.txt."""
        content = """ACT Symbol|Security Name|Exchange|CQS Symbol|ETF|Round Lot Size|Test Issue|NASDAQ Symbol|NextShares
HIMS|HIMS & HERS HEALTH INC|N|HIMS|N|100|N|HIMS|N
BRK.A|BERKSHIRE HATHAWAY INC|N|BRK.A|N|1|N|BRK.A|N"""
        
        df = _parse_other_listed(content)
        
        assert len(df) == 2
        assert "Ticker" in df.columns
        assert df.iloc[0]["Ticker"] == "HIMS"
        assert df.iloc[0]["Exchange"] == "N"
    
    def test_parse_other_listed_with_footer(self):
        """Test parsing with footer row that should be removed."""
        content = """ACT Symbol|Security Name|Exchange|CQS Symbol|ETF|Round Lot Size|Test Issue|NASDAQ Symbol|NextShares
HIMS|HIMS & HERS HEALTH INC|N|HIMS|N|100|N|HIMS|N
File Creation Time: 01/01/2024 09:00:00"""
        
        df = _parse_other_listed(content)
        
        assert len(df) == 1
        assert df.iloc[0]["Ticker"] == "HIMS"
    
    def test_parse_other_listed_empty_content(self):
        """Test parsing empty content."""
        df = _parse_other_listed("")
        assert len(df) == 0


class TestDataFiltering:
    """Test data filtering and cleaning."""
    
    def test_filter_by_exchanges(self):
        """Test filtering by exchange codes."""
        df = pd.DataFrame({
            "Ticker": ["AAPL", "HIMS", "SPY"],
            "Exchange": ["Q", "N", "P"],
            "ETF": ["N", "N", "Y"],
            "Test Issue": ["N", "N", "N"]
        })
        
        result = _filter_and_clean_data(df, ("Q", "N"), False, False)
        
        assert len(result) == 2
        assert "AAPL" in result["Ticker"].values
        assert "HIMS" in result["Ticker"].values
        assert "SPY" not in result["Ticker"].values
    
    def test_filter_etfs(self):
        """Test ETF filtering."""
        df = pd.DataFrame({
            "Ticker": ["AAPL", "SPY", "QQQ"],
            "Exchange": ["Q", "P", "Q"],
            "ETF": ["N", "Y", "Y"],
            "Test Issue": ["N", "N", "N"]
        })
        
        # Exclude ETFs
        result = _filter_and_clean_data(df, ("Q", "P"), False, False)
        assert len(result) == 1
        assert result.iloc[0]["Ticker"] == "AAPL"
        
        # Include ETFs
        result = _filter_and_clean_data(df, ("Q", "P"), True, False)
        assert len(result) == 3
    
    def test_filter_test_issues(self):
        """Test test issue filtering."""
        df = pd.DataFrame({
            "Ticker": ["AAPL", "TEST1", "TEST2"],
            "Exchange": ["Q", "N", "Q"],
            "ETF": ["N", "N", "N"],
            "Test Issue": ["N", "Y", "Y"]
        })
        
        # Exclude test issues
        result = _filter_and_clean_data(df, ("Q", "N"), False, False)
        assert len(result) == 1
        assert result.iloc[0]["Ticker"] == "AAPL"
        
        # Include test issues
        result = _filter_and_clean_data(df, ("Q", "N"), False, True)
        assert len(result) == 3
    
    def test_deduplication_and_sorting(self):
        """Test deduplication and sorting."""
        df = pd.DataFrame({
            "Ticker": ["MSFT", "AAPL", "AAPL", "GOOGL"],
            "Exchange": ["Q", "Q", "Q", "Q"],
            "ETF": ["N", "N", "N", "N"],
            "Test Issue": ["N", "N", "N", "N"]
        })
        
        result = _filter_and_clean_data(df, ("Q",), False, False)
        
        assert len(result) == 3  # After deduplication
        assert result.iloc[0]["Ticker"] == "AAPL"  # First after sorting
        assert result.iloc[1]["Ticker"] == "GOOGL"
        assert result.iloc[2]["Ticker"] == "MSFT"


class TestMainFunction:
    """Test the main get_exchange_listed_tickers function."""
    
    @patch('us_tickers.fetch._download_file')
    def test_get_exchange_listed_tickers_success(self, mock_download):
        """Test successful ticker fetching."""
        # Mock download responses
        nasdaq_content = """Symbol|Security Name|Market Category|Test Issue|Financial Status|Round Lot Size|ETF|NextShares
AAPL|APPLE INC|Q|N|N|100|N|N
MSFT|MICROSOFT CORP|Q|N|N|100|N|N"""
        
        other_content = """ACT Symbol|Security Name|Exchange|CQS Symbol|ETF|Round Lot Size|Test Issue|NASDAQ Symbol|NextShares
HIMS|HIMS & HERS HEALTH INC|N|HIMS|N|100|N|HIMS|N"""
        
        mock_download.side_effect = [nasdaq_content, other_content]
        
        tickers, df = get_exchange_listed_tickers(exchanges=("Q", "N"))
        
        assert len(tickers) == 3
        assert "AAPL" in tickers
        assert "MSFT" in tickers
        assert "HIMS" in tickers
        assert len(df) == 3
    
    def test_get_exchange_listed_tickers_invalid_exchanges(self):
        """Test error handling for invalid exchange codes."""
        with pytest.raises(ValueError, match="Invalid exchange codes"):
            get_exchange_listed_tickers(exchanges=("INVALID",))
    
    @patch('us_tickers.fetch._download_file')
    def test_get_exchange_listed_tickers_defaults(self, mock_download):
        """Test default behavior (Q, N exchanges, no ETFs, no test issues)."""
        nasdaq_content = """Symbol|Security Name|Market Category|Test Issue|Financial Status|Round Lot Size|ETF|NextShares
AAPL|APPLE INC|Q|N|N|100|N|N
SPY|SPDR S&P 500 ETF|Q|N|N|100|Y|N
TEST|TEST ISSUE|Q|Y|N|100|N|N"""
        
        other_content = """ACT Symbol|Security Name|Exchange|CQS Symbol|ETF|Round Lot Size|Test Issue|NASDAQ Symbol|NextShares
HIMS|HIMS & HERS HEALTH INC|N|HIMS|N|100|N|HIMS|N"""
        
        mock_download.side_effect = [nasdaq_content, other_content]
        
        tickers, df = get_exchange_listed_tickers()
        
        # Should only include AAPL and HIMS (exclude SPY=ETF, TEST=test issue)
        assert len(tickers) == 2
        assert "AAPL" in tickers
        assert "HIMS" in tickers
        assert "SPY" not in tickers
        assert "TEST" not in tickers
    
    @patch('us_tickers.fetch._download_file')
    def test_get_exchange_listed_tickers_include_etfs(self, mock_download):
        """Test including ETFs."""
        nasdaq_content = """Symbol|Security Name|Market Category|Test Issue|Financial Status|Round Lot Size|ETF|NextShares
AAPL|APPLE INC|Q|N|N|100|N|N
SPY|SPDR S&P 500 ETF|Q|N|N|100|Y|N"""
        
        other_content = """ACT Symbol|Security Name|Exchange|CQS Symbol|ETF|Round Lot Size|Test Issue|NASDAQ Symbol|NextShares"""
        
        mock_download.side_effect = [nasdaq_content, other_content]
        
        tickers, df = get_exchange_listed_tickers(include_etfs=True)
        
        assert len(tickers) == 2
        assert "AAPL" in tickers
        assert "SPY" in tickers
    
    @patch('us_tickers.fetch._download_file')
    def test_get_exchange_listed_tickers_include_test_issues(self, mock_download):
        """Test including test issues."""
        nasdaq_content = """Symbol|Security Name|Market Category|Test Issue|Financial Status|Round Lot Size|ETF|NextShares
AAPL|APPLE INC|Q|N|N|100|N|N
TEST|TEST ISSUE|Q|Y|N|100|N|N"""
        
        other_content = """ACT Symbol|Security Name|Exchange|CQS Symbol|ETF|Round Lot Size|Test Issue|NASDAQ Symbol|NextShares"""
        
        mock_download.side_effect = [nasdaq_content, other_content]
        
        tickers, df = get_exchange_listed_tickers(include_test_issues=True)
        
        assert len(tickers) == 2
        assert "AAPL" in tickers
        assert "TEST" in tickers


class TestErrorHandling:
    """Test error handling scenarios."""
    
    @patch('us_tickers.fetch._download_file')
    def test_download_failure(self, mock_download):
        """Test handling of download failures."""
        mock_download.side_effect = requests.exceptions.RequestException("Network error")
        
        with pytest.raises(RuntimeError, match="Failed to fetch data from all sources"):
            get_exchange_listed_tickers()
    
    def test_empty_data_handling(self):
        """Test handling of empty data."""
        df = pd.DataFrame()
        result = _filter_and_clean_data(df, ("Q", "N"), False, False)
        assert len(result) == 0


if __name__ == "__main__":
    pytest.main([__file__])
