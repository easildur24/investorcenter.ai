#!/usr/bin/env python3
"""
Direct Ticker Database Importer

This script fetches ticker data from exchanges and imports it directly into the PostgreSQL database.
Perfect for periodic updates - only inserts new tickers, skips existing ones.

Usage:
    python ticker_import_to_db.py                    # Import Nasdaq + NYSE stocks
    python ticker_import_to_db.py --dry-run          # Preview what would be imported
    python ticker_import_to_db.py --exchanges Q      # Only Nasdaq
    python ticker_import_to_db.py --verbose          # Detailed logging
"""

import argparse
import logging
import os
import sys
from pathlib import Path

# Add the us_tickers module to Python path
sys.path.append(str(Path(__file__).parent))

from us_tickers import (get_exchange_listed_tickers, import_stocks_to_database,
                        test_database_connection, transform_for_database)
from us_tickers.database import get_database_stats


def setup_logging(verbose: bool = False) -> None:
    """Setup logging configuration."""
    level = logging.DEBUG if verbose else logging.INFO

    logging.basicConfig(
        level=level,
        format="%(asctime)s - %(name)s - %(levelname)s - %(message)s",
        handlers=[logging.StreamHandler(sys.stderr)],
    )


def parse_exchanges(exchange_arg: str) -> list:
    """Parse exchange codes from comma-separated string."""
    if not exchange_arg:
        return ["Q", "N"]

    exchanges = [ex.strip().upper() for ex in exchange_arg.split(",")]
    return exchanges


def main() -> None:
    """Main function for direct database import."""
    parser = argparse.ArgumentParser(
        description="Fetch tickers and import directly to database",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Exchange Codes:
  Q  - Nasdaq
  N  - NYSE
  A  - NYSE American
  P  - NYSE Arca
  Z  - Cboe

Examples:
  python ticker_import_to_db.py                           # Import Nasdaq + NYSE
  python ticker_import_to_db.py --dry-run                 # Preview mode
  python ticker_import_to_db.py --exchanges Q             # Only Nasdaq
  python ticker_import_to_db.py --batch-size 50 --verbose # Small batches with logging

Environment Variables Required:
  DB_HOST=localhost
  DB_PORT=5432
  DB_USER=investorcenter
  DB_PASSWORD=your_password
  DB_NAME=investorcenter_db
        """,
    )

    parser.add_argument(
        "--exchanges",
        type=str,
        default="Q,N",
        help="Comma-separated exchange codes (default: Q,N)",
    )

    parser.add_argument(
        "--include-etfs",
        action="store_true",
        help="Include ETFs in the import (usually not needed)",
    )

    parser.add_argument(
        "--include-test-issues",
        action="store_true",
        help="Include test issues in the import (usually not needed)",
    )

    parser.add_argument(
        "--batch-size",
        type=int,
        default=100,
        help="Number of records to import per batch (default: 100)",
    )

    parser.add_argument(
        "--dry-run",
        action="store_true",
        help="Preview what would be imported without actually importing",
    )

    parser.add_argument(
        "--verbose",
        action="store_true",
        help="Enable verbose logging",
    )

    args = parser.parse_args()

    # Setup logging
    setup_logging(args.verbose)
    logger = logging.getLogger(__name__)

    try:
        print("🚀 InvestorCenter Ticker Database Importer")
        print("=" * 50)

        # Test database connection first
        print("🔍 Testing database connection...")
        if not test_database_connection():
            print(
                "❌ Database connection failed. Please check your configuration.",
                file=sys.stderr,
            )
            print("\nTroubleshooting:")
            print("1. Make sure PostgreSQL is running")
            print("2. Check your .env file or environment variables:")
            print("   DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME")
            print("3. Verify database exists and migrations are run")
            print("4. Test with: psql -h $DB_HOST -U $DB_USER -d $DB_NAME")
            sys.exit(1)
        print("✅ Database connection successful!")

        # Show current database stats
        stats = get_database_stats()
        if stats:
            print(f"\n📊 Current database stats:")
            print(f"  Total stocks: {stats.get('total_stocks', 0)}")
            if stats.get("by_exchange"):
                for exchange, count in stats["by_exchange"].items():
                    print(f"  {exchange}: {count}")
            print(f"  Added in last 24h: {stats.get('added_last_24h', 0)}")

        # Parse exchanges
        exchanges = parse_exchanges(args.exchanges)
        logger.info(f"Fetching tickers for exchanges: {exchanges}")

        print(f"\n🔄 Fetching ticker data...")
        print(f"  Exchanges: {', '.join(exchanges)}")
        print(f"  Include ETFs: {args.include_etfs}")
        print(f"  Include test issues: {args.include_test_issues}")

        # Fetch raw ticker data
        raw_df = get_exchange_listed_tickers(
            exchanges=tuple(exchanges),
            include_etfs=args.include_etfs,
            include_test_issues=args.include_test_issues,
        )

        print(f"📥 Downloaded {len(raw_df)} raw ticker records")

        # Transform for database
        print("🔧 Transforming data for database...")
        transformed_df = transform_for_database(raw_df)

        if transformed_df.empty:
            print("⚠️  No valid tickers found after filtering")
            sys.exit(0)

        print(f"✨ {len(transformed_df)} tickers ready for import")

        # Show exchange distribution
        exchange_counts = transformed_df["exchange"].value_counts()
        for exchange, count in exchange_counts.items():
            print(f"  {exchange}: {count}")

        if args.dry_run:
            print(
                "\n🏃 Dry run mode - showing preview of data to be imported:"
            )
            preview_df = transformed_df.head(10)
            for _, row in preview_df.iterrows():
                print(
                    f"  {row['symbol']:6} | {row['name']:50} | {row['exchange']}"
                )
            if len(transformed_df) > 10:
                print(f"  ... and {len(transformed_df) - 10} more records")
            print("\n💡 To actually import, run without --dry-run flag")
            print(
                "📝 Note: ON CONFLICT DO NOTHING - existing stocks will be skipped"
            )
            sys.exit(0)

        # Import to database
        print(f"\n📊 Importing to database (batch size: {args.batch_size})...")
        print(
            "📝 Using ON CONFLICT DO NOTHING - existing stocks will be preserved"
        )

        inserted, skipped = import_stocks_to_database(
            transformed_df, args.batch_size
        )

        print(f"\n🎉 Import completed!")
        print(f"  ✅ Inserted: {inserted} new stocks")
        print(f"  ⏭️  Skipped: {skipped} existing stocks")
        print(f"  📈 Total processed: {inserted + skipped}")

        if inserted > 0:
            print(f"  🆕 {inserted} new companies added to InvestorCenter!")
        else:
            print("  ℹ️  Database is up to date - no new stocks found")

        # Show updated stats
        final_stats = get_database_stats()
        if final_stats:
            print(f"\n📊 Updated database stats:")
            print(f"  Total stocks: {final_stats.get('total_stocks', 0)}")
            if final_stats.get("by_exchange"):
                for exchange, count in final_stats["by_exchange"].items():
                    print(f"  {exchange}: {count}")

        print(
            f"\n🔄 For periodic updates, run this script again to get new listings"
        )

    except KeyboardInterrupt:
        print("\n\nOperation cancelled by user", file=sys.stderr)
        sys.exit(130)
    except Exception as e:
        logger.error(f"Error importing tickers: {e}")
        print(f"\n❌ Error: {e}", file=sys.stderr)
        sys.exit(1)


if __name__ == "__main__":
    main()
