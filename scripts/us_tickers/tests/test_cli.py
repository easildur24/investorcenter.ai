"""Unit tests for the CLI module."""

import logging
import os
import tempfile
from typing import Any
from unittest.mock import patch

import pandas as pd
import pytest

from us_tickers.cli import parse_exchanges, save_output, setup_logging


class TestExchangeParsing:
    """Test exchange code parsing."""

    def test_parse_exchanges_default(self) -> None:
        """Test default exchange parsing."""
        exchanges = parse_exchanges("")
        assert exchanges == ["Q", "N"]

    def test_parse_exchanges_single(self) -> None:
        """Test single exchange parsing."""
        exchanges = parse_exchanges("Q")
        assert exchanges == ["Q"]

    def test_parse_exchanges_multiple(self) -> None:
        """Test multiple exchange parsing."""
        exchanges = parse_exchanges("Q,N,A")
        assert exchanges == ["Q", "N", "A"]

    def test_parse_exchanges_with_spaces(self) -> None:
        """Test exchange parsing with spaces."""
        exchanges = parse_exchanges(" Q , N , A ")
        assert exchanges == ["Q", "N", "A"]


class TestOutputSaving:
    """Test output file saving."""

    def test_save_csv_output(self) -> None:
        """Test saving CSV output."""
        df = pd.DataFrame(
            {
                "Ticker": ["AAPL", "MSFT"],
                "Exchange": ["Q", "Q"],
                "ETF": ["N", "N"],
            }
        )

        with tempfile.TemporaryDirectory() as temp_dir:
            output_path = os.path.join(temp_dir, "test.csv")
            save_output(df, output_path, "csv")

            # Verify file was created
            assert os.path.exists(output_path)

            # Verify content
            with open(output_path, "r") as f:
                content = f.read()
                assert "Ticker" in content
                assert "AAPL" in content
                assert "MSFT" in content

    def test_save_json_output(self) -> None:
        """Test saving JSON output."""
        df = pd.DataFrame(
            {
                "Ticker": ["AAPL", "MSFT"],
                "Exchange": ["Q", "Q"],
                "ETF": ["N", "N"],
            }
        )

        with tempfile.TemporaryDirectory() as temp_dir:
            output_path = os.path.join(temp_dir, "test.json")
            save_output(df, output_path, "json")

            # Verify file was created
            assert os.path.exists(output_path)

            # Verify content
            with open(output_path, "r") as f:
                content = f.read()
                assert "AAPL" in content
                assert "MSFT" in content

    def test_save_output_invalid_format(self) -> None:
        """Test error handling for invalid output format."""
        df = pd.DataFrame({"Ticker": ["AAPL"]})

        with pytest.raises(ValueError, match="Unsupported output format"):
            save_output(df, "test.txt", "txt")

    def test_save_output_create_directory(self) -> None:
        """Test that output directory is created if it doesn't exist."""
        df = pd.DataFrame({"Ticker": ["AAPL"]})

        with tempfile.TemporaryDirectory() as temp_dir:
            output_path = os.path.join(temp_dir, "newdir", "test.csv")
            save_output(df, output_path, "csv")

            # Verify directory and file were created
            assert os.path.exists(os.path.dirname(output_path))
            assert os.path.exists(output_path)


class TestLoggingSetup:
    """Test logging configuration."""

    def test_setup_logging_info_level(self) -> None:
        """Test logging setup with INFO level."""
        setup_logging(verbose=False)
        # Check that the root logger has INFO level
        assert logging.getLogger().level == logging.INFO

    def test_setup_logging_debug_level(self) -> None:
        """Test logging setup with DEBUG level."""
        setup_logging(verbose=True)
        # Check that the root logger has DEBUG level
        assert logging.getLogger().level == logging.DEBUG


class TestCLIArgumentParsing:
    """Test CLI argument parsing."""

    @patch("us_tickers.cli.get_exchange_listed_tickers")
    @patch("us_tickers.cli.save_output")
    def test_cli_fetch_command_defaults(self, mock_save: Any, mock_fetch: Any) -> None:
        """Test CLI fetch command with default arguments."""
        from us_tickers.cli import main

        # Mock the fetch function
        mock_fetch.return_value = (["AAPL", "MSFT"], pd.DataFrame())

        # Mock command line arguments
        with patch("sys.argv", ["tickers", "fetch", "--out", "test.csv"]):
            main()

        # Verify fetch was called with defaults
        mock_fetch.assert_called_once_with(
            exchanges=("Q", "N"), include_etfs=False, include_test_issues=False
        )

        # Verify save was called
        mock_save.assert_called_once()

    @patch("us_tickers.cli.get_exchange_listed_tickers")
    @patch("us_tickers.cli.save_output")
    def test_cli_fetch_command_with_options(self, mock_save: Any, mock_fetch: Any) -> None:
        """Test CLI fetch command with all options."""
        from us_tickers.cli import main

        # Mock the fetch function
        mock_fetch.return_value = (["AAPL", "SPY"], pd.DataFrame())

        # Mock command line arguments
        with patch(
            "sys.argv",
            [
                "tickers",
                "fetch",
                "--exchanges",
                "Q,P",
                "--include-etfs",
                "--include-test-issues",
                "--format",
                "json",
                "--out",
                "test.json",
                "--verbose",
            ],
        ):
            main()

        # Verify fetch was called with options
        mock_fetch.assert_called_once_with(
            exchanges=("Q", "P"), include_etfs=True, include_test_issues=True
        )

        # Verify save was called with JSON format
        mock_save.assert_called_once()

    def test_cli_no_command(self) -> None:
        """Test CLI with no command specified."""
        from us_tickers.cli import main

        with patch("sys.argv", ["tickers"]):
            with patch("sys.exit") as mock_exit:
                main()
                mock_exit.assert_called_once_with(1)


if __name__ == "__main__":
    pytest.main([__file__])
