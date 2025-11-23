#!/usr/bin/env python3
"""Sync tickers from tickers table to companies table for IC Score processing.

This script copies tickers from the main 'tickers' table to the IC Score 'companies' table.
It only syncs tickers that don't already exist in companies.

Usage:
    python sync_tickers_to_companies.py                    # Sync all missing tickers
    python sync_tickers_to_companies.py --limit 1000       # Sync first 1000 missing tickers
    python sync_tickers_to_companies.py --priority-only    # Sync only high-priority stocks (with CIK, US, active)
"""
import argparse
import asyncio
import sys
from datetime import datetime
from pathlib import Path

# Add parent directory to path
sys.path.insert(0, str(Path(__file__).parent.parent))

from sqlalchemy import text
from database.database import get_database


async def get_missing_tickers(session, limit: int = None, priority_only: bool = False):
    """Get tickers that exist in tickers table but not in companies table."""

    # Build query to find tickers not yet in companies
    query = """
        SELECT
            t.symbol,
            t.name,
            t.exchange,
            t.currency,
            t.cik,
            t.active,
            t.market,
            t.locale
        FROM tickers t
        LEFT JOIN companies c ON t.symbol = c.ticker
        WHERE c.ticker IS NULL
    """

    if priority_only:
        # Only sync high-priority tickers: active US stocks with CIK
        query += """
            AND t.active = TRUE
            AND t.cik IS NOT NULL
            AND t.cik != ''
            AND t.locale = 'us'
            AND t.market = 'stocks'
        """

    query += " ORDER BY t.symbol"

    if limit:
        query += f" LIMIT {limit}"

    result = await session.execute(text(query))
    return result.fetchall()


async def sync_ticker_to_company(session, ticker_data):
    """Insert a ticker into the companies table."""
    symbol, name, exchange, currency, cik, active, market, locale = ticker_data

    insert_query = text("""
        INSERT INTO companies (
            ticker,
            name,
            exchange,
            currency,
            cik,
            is_active,
            created_at,
            last_updated
        ) VALUES (
            :ticker,
            :name,
            :exchange,
            :currency,
            :cik,
            :is_active,
            NOW(),
            NOW()
        )
        ON CONFLICT (ticker) DO UPDATE SET
            cik = EXCLUDED.cik,
            last_updated = NOW()
    """)

    await session.execute(insert_query, {
        'ticker': symbol,
        'name': name or '',
        'exchange': exchange or '',
        'currency': (currency or 'USD').upper(),
        'cik': cik,
        'is_active': active
    })


async def sync_tickers(limit: int = None, priority_only: bool = False, dry_run: bool = False):
    """Main sync function."""
    db = get_database()

    print("=" * 80)
    print("TICKER TO COMPANIES SYNC")
    print("=" * 80)

    async with db.session() as session:
        # Get counts before sync
        companies_before = (await session.execute(text("SELECT COUNT(*) FROM companies"))).scalar()
        tickers_count = (await session.execute(text("SELECT COUNT(*) FROM tickers"))).scalar()

        print(f"\nBefore sync:")
        print(f"  Companies: {companies_before:,}")
        print(f"  Tickers: {tickers_count:,}")
        print(f"  Gap: {tickers_count - companies_before:,}")

        # Get missing tickers
        print(f"\nFetching missing tickers...")
        if priority_only:
            print("  Mode: Priority only (active US stocks with CIK)")
        else:
            print("  Mode: All missing tickers")

        missing = await get_missing_tickers(session, limit, priority_only)
        print(f"  Found {len(missing):,} tickers to sync")

        if not missing:
            print("\n✓ No tickers to sync - databases are in sync!")
            return

        if dry_run:
            print(f"\nDRY RUN - would sync {len(missing):,} tickers:")
            for i, ticker_data in enumerate(missing[:20]):
                symbol, name, exchange, currency, cik, active, market, locale = ticker_data
                print(f"  {symbol:6} - {name[:50]:50} (CIK: {cik or 'N/A':10}) {market}/{locale}")
            if len(missing) > 20:
                print(f"  ... and {len(missing) - 20:,} more")
            print("\nRun without --dry-run to perform actual sync")
            return

        # Sync tickers
        print(f"\nSyncing {len(missing):,} tickers to companies table...")
        synced = 0
        failed = 0

        for i, ticker_data in enumerate(missing):
            try:
                await sync_ticker_to_company(session, ticker_data)
                synced += 1

                if (i + 1) % 100 == 0:
                    await session.commit()
                    print(f"  Synced {i + 1:,}/{len(missing):,} tickers...")
            except Exception as e:
                failed += 1
                symbol = ticker_data[0]
                print(f"  ✗ Failed to sync {symbol}: {e}")

        # Final commit
        await session.commit()

        # Get counts after sync
        companies_after = (await session.execute(text("SELECT COUNT(*) FROM companies"))).scalar()

        print(f"\nSync complete:")
        print(f"  Successfully synced: {synced:,}")
        print(f"  Failed: {failed:,}")
        print(f"  Companies before: {companies_before:,}")
        print(f"  Companies after: {companies_after:,}")
        print(f"  New companies added: {companies_after - companies_before:,}")

        # Check specific high-value tickers
        print(f"\nVerifying specific tickers:")
        high_value_tickers = ['ORCL', 'NVDA', 'META', 'NFLX', 'AMD']
        for ticker in high_value_tickers:
            result = await session.execute(
                text("SELECT name FROM companies WHERE ticker = :ticker"),
                {'ticker': ticker}
            )
            row = result.fetchone()
            if row:
                print(f"  ✓ {ticker:6} - {row[0]}")
            else:
                print(f"  ✗ {ticker:6} - NOT FOUND")

    print("\n" + "=" * 80)


async def main():
    """Entry point."""
    parser = argparse.ArgumentParser(
        description='Sync tickers from tickers table to companies table',
        formatter_class=argparse.RawDescriptionHelpFormatter
    )

    parser.add_argument(
        '--limit',
        type=int,
        default=None,
        help='Limit number of tickers to sync (for testing)'
    )
    parser.add_argument(
        '--priority-only',
        action='store_true',
        help='Only sync priority tickers (active US stocks with CIK)'
    )
    parser.add_argument(
        '--dry-run',
        action='store_true',
        help='Show what would be synced without actually syncing'
    )

    args = parser.parse_args()

    try:
        await sync_tickers(
            limit=args.limit,
            priority_only=args.priority_only,
            dry_run=args.dry_run
        )
    except Exception as e:
        print(f"\n✗ Error during sync: {e}")
        import traceback
        traceback.print_exc()
        sys.exit(1)


if __name__ == '__main__':
    asyncio.run(main())
