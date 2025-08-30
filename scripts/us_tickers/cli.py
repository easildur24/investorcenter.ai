"""Command line interface for US Tickers package."""

import argparse
import logging
import sys
from pathlib import Path
from typing import List

import pandas as pd

from .config import config
from .fetch import get_exchange_listed_tickers


def setup_logging(verbose: bool) -> None:
    """Setup logging configuration."""
    level = logging.DEBUG if verbose else logging.INFO

    # Clear any existing handlers and set level
    root_logger = logging.getLogger()
    root_logger.handlers.clear()
    root_logger.setLevel(level)

    # Add handler
    handler = logging.StreamHandler(sys.stderr)
    handler.setLevel(level)
    formatter = logging.Formatter(
        "%(asctime)s - %(name)s - %(levelname)s - %(message)s"
    )
    handler.setFormatter(formatter)
    root_logger.addHandler(handler)


def parse_exchanges(exchange_arg: str) -> List[str]:
    """Parse exchange codes from comma-separated string."""
    if not exchange_arg:
        return ["Q", "N"]

    exchanges = [ex.strip().upper() for ex in exchange_arg.split(",")]
    return exchanges


def save_output(
    df: pd.DataFrame, output_path: str, output_format: str
) -> None:
    """Save DataFrame to output file in specified format."""
    output_file = Path(output_path)

    # Ensure output directory exists
    output_file.parent.mkdir(parents=True, exist_ok=True)

    try:
        if output_format.lower() == "csv":
            df.to_csv(output_file, index=False)
            print(f"Saved {len(df)} tickers to {output_file}")
        elif output_format.lower() == "json":
            df.to_json(output_file, orient="records", indent=2)
            print(f"Saved {len(df)} tickers to {output_file}")
        else:
            raise ValueError(f"Unsupported output format: {output_format}")
    except Exception as e:
        print(f"Error saving output to {output_file}: {e}", file=sys.stderr)
        raise  # Re-raise instead of sys.exit for better testing


def main() -> None:
    """Main CLI entry point."""
    parser = argparse.ArgumentParser(
        description="Download and merge Nasdaq + NYSE tickers",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Exchange Codes:
  Q  - Nasdaq
  N  - NYSE
  A  - NYSE American
  P  - NYSE Arca
  Z  - Cboe

Examples:
  tickers fetch --exchanges Q,N --format csv --out tickers.csv
  tickers fetch --exchanges Q --include-etfs --format json
    --out nasdaq_etfs.json
  tickers fetch --exchanges N,A --include-test-issues --format csv
    --out nyse_plus_amex.csv
        """,
    )

    subparsers = parser.add_subparsers(
        dest="command", help="Available commands"
    )

    # Fetch command
    fetch_parser = subparsers.add_parser(
        "fetch", help="Fetch tickers from exchanges"
    )

    fetch_parser.add_argument(
        "--exchanges",
        type=str,
        default=",".join(config.default_exchanges or ["Q", "N"]),
        help=(
            f"Comma-separated exchange codes "
            f"(default: {','.join(config.default_exchanges or ['Q', 'N'])})"
        ),
    )

    fetch_parser.add_argument(
        "--include-etfs",
        action="store_true",
        help="Include ETFs in the output",
    )

    fetch_parser.add_argument(
        "--include-test-issues",
        action="store_true",
        help="Include test issues in the output",
    )

    fetch_parser.add_argument(
        "--format",
        choices=["csv", "json"],
        default="csv",
        help="Output format (default: csv)",
    )

    fetch_parser.add_argument(
        "--out", type=str, required=True, help="Output file path"
    )

    fetch_parser.add_argument(
        "--verbose", action="store_true", help="Enable verbose logging"
    )

    # Parse arguments
    args = parser.parse_args()

    if not args.command:
        parser.print_help()
        sys.exit(1)

    if args.command == "fetch":
        # Setup logging
        setup_logging(args.verbose)
        logger = logging.getLogger(__name__)

        try:
            # Parse exchanges
            exchanges = parse_exchanges(args.exchanges)
            logger.info(f"Fetching tickers for exchanges: {exchanges}")

            # Fetch tickers
            df = get_exchange_listed_tickers(
                exchanges=tuple(exchanges),
                include_etfs=args.include_etfs,
                include_test_issues=args.include_test_issues,
            )

            # Extract ticker list for display
            ticker_list = df["Ticker"].tolist()

            # Display summary
            print(f"Fetched {len(ticker_list)} tickers:")
            print(f"  Exchanges: {', '.join(exchanges)}")
            print(f"  Include ETFs: {args.include_etfs}")
            print(f"  Include test issues: {args.include_test_issues}")

            # Show first few tickers
            if ticker_list:
                print(f"  First 10 tickers: {', '.join(ticker_list[:10])}")
                if len(ticker_list) > 10:
                    print(f"  ... and {len(ticker_list) - 10} more")

            # Save output
            save_output(df, args.out, args.format)

        except KeyboardInterrupt:
            print("\nOperation cancelled by user", file=sys.stderr)
            sys.exit(130)
        except Exception as e:
            logger.error(f"Error fetching tickers: {e}")
            print(f"Error: {e}", file=sys.stderr)
            sys.exit(1)


if __name__ == "__main__":
    main()
