#!/usr/bin/env python3
"""
ApeWisdom Historical Ranking Collector

Fetches top 100 stock rankings from ApeWisdom API and stores in PostgreSQL.
Can backfill historical data by simulating snapshots from the current data.

Usage:
    # Collect current snapshot
    DB_HOST=localhost DB_USER=edwardsun DB_NAME=investorcenter_db \\
    scripts/venv/bin/python scripts/apewisdom_collector.py --limit 100

    # Backfill 30 days of data (uses current data as proxy)
    DB_HOST=localhost DB_USER=edwardsun DB_NAME=investorcenter_db \\
    scripts/venv/bin/python scripts/apewisdom_collector.py \\
        --backfill-days 30 --limit 100

Dependencies: requests, psycopg2
"""

import argparse
import os
import sys
import time
from datetime import datetime, timedelta, timezone
from typing import Dict, List, Optional

import psycopg2
import requests


class ApeWisdomCollector:
    """Collects stock ranking data from ApeWisdom API"""

    BASE_URL = "https://apewisdom.io/api/v1.0/filter/all-stocks"

    def __init__(
        self, db_host: str, db_user: str, db_name: str, db_password: str
    ):
        """Initialize collector with database connection"""
        self.db_conn = psycopg2.connect(
            host=db_host,
            user=db_user,
            database=db_name,
            password=db_password,
        )
        self.db_conn.autocommit = False

    def fetch_rankings(self, page: int = 1) -> Optional[Dict]:
        """
        Fetch rankings from ApeWisdom API

        Args:
            page: Page number (1-based, ~100 results per page)

        Returns:
            JSON response dict or None on error
        """
        if page == 1:
            # First page doesn't need page parameter
            url = self.BASE_URL
        else:
            url = f"{self.BASE_URL}/{page}"

        try:
            print(f"Fetching {url}...")
            response = requests.get(url, timeout=15)
            response.raise_for_status()
            data = response.json()
            print(f"  ✓ Received {len(data.get('results', []))} results")
            return data
        except requests.RequestException as e:
            print(f"  ✗ Error fetching page {page}: {e}")
            return None

    def ticker_exists(self, ticker: str) -> bool:
        """Check if ticker exists in tickers table"""
        cursor = self.db_conn.cursor()
        try:
            cursor.execute(
                "SELECT symbol FROM tickers WHERE symbol = %s", (ticker,)
            )
            result = cursor.fetchone()
            return result is not None
        finally:
            cursor.close()

    def store_ranking(
        self, ticker: str, data: Dict, snapshot_time: datetime
    ) -> bool:
        """
        Store single ticker ranking in database

        Args:
            ticker: Stock symbol
            data: Ranking data from API
            snapshot_time: Timestamp for this snapshot

        Returns:
            True if stored successfully, False otherwise
        """
        cursor = self.db_conn.cursor()

        try:
            # Check if ticker exists
            if not self.ticker_exists(ticker):
                print(f"  ⊘ Skipping {ticker} (not in tickers table)")
                return False

            # Insert ranking data
            cursor.execute(
                """
                INSERT INTO reddit_ticker_rankings
                (ticker_symbol, rank, mentions, upvotes, rank_24h_ago,
                 mentions_24h_ago, snapshot_date, snapshot_time,
                 data_source)
                VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s)
                ON CONFLICT (ticker_symbol, snapshot_date, data_source)
                DO UPDATE SET
                    rank = EXCLUDED.rank,
                    mentions = EXCLUDED.mentions,
                    upvotes = EXCLUDED.upvotes,
                    rank_24h_ago = EXCLUDED.rank_24h_ago,
                    mentions_24h_ago = EXCLUDED.mentions_24h_ago,
                    snapshot_time = EXCLUDED.snapshot_time
            """,
                (
                    ticker,
                    data.get("rank"),
                    data.get("mentions", 0),
                    data.get("upvotes", 0),
                    data.get("rank_24h_ago"),
                    data.get("mentions_24h_ago"),
                    snapshot_time.date(),
                    snapshot_time,
                    "apewisdom",
                ),
            )
            return True

        except Exception as e:
            print(f"  ✗ Error storing {ticker}: {e}")
            raise
        finally:
            cursor.close()

    def collect_current_snapshot(self, max_pages: int = 1) -> int:
        """
        Collect current rankings from API

        Args:
            max_pages: Max pages to fetch (1 page = ~100 tickers)

        Returns:
            Number of rankings stored
        """
        snapshot_time = datetime.now(timezone.utc)
        stored_count = 0
        failed_count = 0

        print(f"\n📊 Collecting current snapshot at {snapshot_time}")
        print(f"   Fetching up to {max_pages} pages...")

        for page in range(1, max_pages + 1):
            data = self.fetch_rankings(page)
            if not data or "results" not in data:
                print(f"  ⊘ No more data at page {page}")
                break

            results = data["results"]
            if not results:
                break

            page_stored = 0
            for item in results:
                ticker = item.get("ticker")
                if not ticker:
                    continue

                try:
                    if self.store_ranking(ticker, item, snapshot_time):
                        page_stored += 1
                        stored_count += 1
                    else:
                        failed_count += 1
                except Exception as e:
                    print(f"  ✗ Failed to store {ticker}: {e}")
                    self.db_conn.rollback()
                    failed_count += 1
                    continue

            # Commit after each page
            self.db_conn.commit()
            print(f"  ✓ Page {page}: stored {page_stored} rankings")

            # Rate limiting: 1 second between pages
            if page < max_pages - 1:
                time.sleep(1)

        print(f"\n✓ Snapshot complete:")
        print(f"  • Stored: {stored_count} rankings")
        print(f"  • Skipped: {failed_count} (not in database)")
        return stored_count

    def backfill_historical_data(
        self, days: int, max_pages: int = 1
    ) -> int:
        """
        Backfill historical data by creating snapshots for past days

        NOTE: This uses current API data as a proxy for historical data.
        Actual historical data is not available from ApeWisdom free tier.
        This creates realistic-looking test data for development.

        Args:
            days: Number of days to backfill
            max_pages: Max pages per day (1 page = ~100 tickers)

        Returns:
            Total number of rankings stored
        """
        print(f"\n📅 Backfilling {days} days of historical data")
        print(
            "   (Using current data as proxy - not real historical data)\n"
        )

        total_stored = 0

        # Fetch current data once
        all_results = []
        for page in range(1, max_pages + 1):
            data = self.fetch_rankings(page)
            if data and "results" in data:
                all_results.extend(data["results"])
                time.sleep(1)  # Rate limiting

        if not all_results:
            print("✗ No data fetched from API")
            return 0

        print(f"✓ Fetched {len(all_results)} current rankings\n")

        # Create snapshots for each historical day
        for day_offset in range(days, 0, -1):
            snapshot_time = datetime.now(timezone.utc) - timedelta(
                days=day_offset
            )
            snapshot_date = snapshot_time.date()

            print(
                f"📅 Creating snapshot for {snapshot_date} "
                f"({day_offset} days ago)..."
            )

            day_stored = 0
            for item in all_results:
                ticker = item.get("ticker")
                if not ticker:
                    continue

                # Add some randomness to historical data
                # (vary mentions/rank slightly to simulate change over time)
                import random

                variance = random.uniform(0.8, 1.2)
                modified_item = {
                    "rank": item.get("rank"),
                    "mentions": int(item.get("mentions", 0) * variance),
                    "upvotes": int(item.get("upvotes", 0) * variance),
                    "rank_24h_ago": item.get("rank_24h_ago"),
                    "mentions_24h_ago": item.get("mentions_24h_ago"),
                }

                try:
                    if self.store_ranking(
                        ticker, modified_item, snapshot_time
                    ):
                        day_stored += 1
                except Exception as e:
                    print(f"  ✗ Failed: {ticker}: {e}")
                    self.db_conn.rollback()
                    continue

            self.db_conn.commit()
            total_stored += day_stored
            print(f"  ✓ Stored {day_stored} rankings\n")

        print(f"✓ Backfill complete: {total_stored} total rankings")
        return total_stored

    def calculate_daily_metrics(
        self, target_date: Optional[datetime] = None
    ):
        """
        Calculate aggregated daily metrics for heatmap

        Args:
            target_date: Date to calculate (defaults to today)
        """
        if not target_date:
            target_date = datetime.now(timezone.utc).date()

        print(f"\n📊 Calculating daily metrics for {target_date}...")

        cursor = self.db_conn.cursor()

        # Calculate aggregated metrics
        cursor.execute(
            """
            INSERT INTO reddit_heatmap_daily
            (ticker_symbol, date, avg_rank, min_rank, max_rank,
             total_mentions, total_upvotes, rank_volatility,
             popularity_score, data_source)
            SELECT
                ticker_symbol,
                snapshot_date as date,
                AVG(rank) as avg_rank,
                MIN(rank) as min_rank,
                MAX(rank) as max_rank,
                SUM(mentions) as total_mentions,
                SUM(upvotes) as total_upvotes,
                STDDEV(rank) as rank_volatility,
                -- Popularity score calculation
                LEAST(100, (
                    (SUM(mentions) * 0.4) +
                    (SUM(upvotes) / 100.0 * 0.3) +
                    ((101 - AVG(rank)) * 0.3)
                )) as popularity_score,
                data_source
            FROM reddit_ticker_rankings
            WHERE snapshot_date = %s
            GROUP BY ticker_symbol, snapshot_date, data_source
            ON CONFLICT (ticker_symbol, date, data_source)
            DO UPDATE SET
                avg_rank = EXCLUDED.avg_rank,
                min_rank = EXCLUDED.min_rank,
                max_rank = EXCLUDED.max_rank,
                total_mentions = EXCLUDED.total_mentions,
                total_upvotes = EXCLUDED.total_upvotes,
                rank_volatility = EXCLUDED.rank_volatility,
                popularity_score = EXCLUDED.popularity_score,
                calculated_at = NOW()
        """,
            (target_date,),
        )

        rows_affected = cursor.rowcount

        # Calculate trend direction (compare to yesterday)
        cursor.execute(
            """
            UPDATE reddit_heatmap_daily AS today
            SET trend_direction = CASE
                WHEN today.avg_rank < yesterday.avg_rank - 5
                    THEN 'rising'
                WHEN today.avg_rank > yesterday.avg_rank + 5
                    THEN 'falling'
                ELSE 'stable'
            END
            FROM reddit_heatmap_daily AS yesterday
            WHERE today.ticker_symbol = yesterday.ticker_symbol
                AND today.date = %s
                AND yesterday.date = %s - INTERVAL '1 day'
                AND today.data_source = yesterday.data_source
        """,
            (target_date, target_date),
        )

        self.db_conn.commit()
        cursor.close()

        print(f"  ✓ Calculated metrics for {rows_affected} tickers")

    def calculate_all_daily_metrics(self, days: int):
        """Calculate daily metrics for all historical days"""
        print(f"\n📊 Calculating daily metrics for all {days} days...")

        for day_offset in range(days, -1, -1):
            target_date = (
                datetime.now(timezone.utc) - timedelta(days=day_offset)
            ).date()
            self.calculate_daily_metrics(target_date)

        print("✓ All daily metrics calculated")

    def cleanup_old_data(self, retention_days: int = 30):
        """
        Delete data older than retention_days

        Args:
            retention_days: Number of days to keep (default: 30)
        """
        cutoff_date = (
            datetime.now(timezone.utc) - timedelta(days=retention_days)
        ).date()

        print(f"\n🗑️  Cleaning up data older than {cutoff_date}...")

        cursor = self.db_conn.cursor()

        # Delete old rankings
        cursor.execute(
            """
            DELETE FROM reddit_ticker_rankings
            WHERE snapshot_date < %s
        """,
            (cutoff_date,),
        )
        rankings_deleted = cursor.rowcount

        # Delete old daily metrics
        cursor.execute(
            """
            DELETE FROM reddit_heatmap_daily
            WHERE date < %s
        """,
            (cutoff_date,),
        )
        metrics_deleted = cursor.rowcount

        self.db_conn.commit()
        cursor.close()

        print(f"  ✓ Deleted {rankings_deleted} old ranking snapshots")
        print(f"  ✓ Deleted {metrics_deleted} old daily metrics")
        print(
            f"  ✓ Retained data from {cutoff_date} onwards "
            f"({retention_days} days)"
        )

    def close(self):
        """Close database connection"""
        self.db_conn.close()


def main():
    """Main entry point"""
    parser = argparse.ArgumentParser(
        description="Collect ApeWisdom stock rankings"
    )
    parser.add_argument(
        "--limit",
        type=int,
        default=100,
        help="Max tickers to collect (default: 100)",
    )
    parser.add_argument(
        "--backfill-days",
        type=int,
        default=0,
        help="Backfill N days of historical data (default: 0)",
    )
    parser.add_argument(
        "--calculate-metrics",
        action="store_true",
        help="Calculate daily metrics after collection",
    )
    parser.add_argument(
        "--cleanup",
        action="store_true",
        help="Clean up data older than retention period",
    )
    parser.add_argument(
        "--retention-days",
        type=int,
        default=30,
        help="Number of days to retain data (default: 30)",
    )
    args = parser.parse_args()

    # Database config from environment
    db_config = {
        "db_host": os.getenv("DB_HOST", "localhost"),
        "db_user": os.getenv("DB_USER", "investorcenter"),
        "db_name": os.getenv("DB_NAME", "investorcenter_db"),
        "db_password": os.getenv("DB_PASSWORD", ""),
    }

    # Validate environment (password can be empty string)
    required_keys = ["db_host", "db_user", "db_name"]
    if not all(db_config.get(key) for key in required_keys):
        print("✗ Error: Missing database configuration")
        print("  Required: DB_HOST, DB_USER, DB_NAME")
        sys.exit(1)

    collector = ApeWisdomCollector(**db_config)

    try:
        # Calculate max pages (~100 per page)
        max_pages = max(1, (args.limit + 99) // 100)

        if args.backfill_days > 0:
            # Backfill mode: create historical snapshots
            collector.backfill_historical_data(
                args.backfill_days, max_pages
            )
            if args.calculate_metrics:
                collector.calculate_all_daily_metrics(args.backfill_days)
        else:
            # Normal mode: collect current snapshot
            collector.collect_current_snapshot(max_pages=max_pages)
            if args.calculate_metrics:
                collector.calculate_daily_metrics()

        # Cleanup old data if requested
        if args.cleanup:
            collector.cleanup_old_data(retention_days=args.retention_days)

        print("\n✓ Collection complete!")

    except KeyboardInterrupt:
        print("\n\n⊘ Interrupted by user")
        sys.exit(1)
    except Exception as e:
        print(f"\n✗ Fatal error: {e}")
        import traceback

        traceback.print_exc()
        sys.exit(1)
    finally:
        collector.close()


if __name__ == "__main__":
    main()
