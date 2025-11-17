#!/usr/bin/env python3
"""Analyst Ratings Ingestion Pipeline.

This script fetches Wall Street analyst ratings and price targets from
Benzinga API (or other sources) and populates the analyst_ratings table.

Usage:
    python analyst_ratings_ingestion.py --limit 500    # Free tier daily limit
    python analyst_ratings_ingestion.py --ticker AAPL  # Single ticker
    python analyst_ratings_ingestion.py --sp500        # S&P 500 only
"""

import argparse
import asyncio
import logging
import os
import sys
from datetime import datetime, timedelta
from pathlib import Path
from typing import List, Optional, Dict, Any

sys.path.insert(0, str(Path(__file__).parent.parent))

import requests
from sqlalchemy import text
from sqlalchemy.dialects.postgresql import insert as pg_insert
from tqdm import tqdm

from database.database import get_database
from models import AnalystRating

# Setup logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s',
    handlers=[
        logging.StreamHandler(),
        logging.FileHandler('/app/logs/analyst_ratings_ingestion.log')
    ]
)
logger = logging.getLogger(__name__)


class AnalystRatingsIngestion:
    """Ingestion pipeline for analyst ratings and price targets."""

    # Rating to numeric mapping (1-5 scale)
    RATING_MAP = {
        'Strong Buy': 5.0,
        'Buy': 4.0,
        'Outperform': 4.0,
        'Hold': 3.0,
        'Neutral': 3.0,
        'Market Perform': 3.0,
        'Underperform': 2.0,
        'Sell': 2.0,
        'Strong Sell': 1.0,
    }

    def __init__(self, api_key: Optional[str] = None):
        """Initialize the ingestion pipeline.

        Args:
            api_key: Benzinga API key (optional, from environment if not provided).
        """
        self.api_key = api_key or os.getenv('BENZINGA_API_KEY')
        self.db = get_database()

        if not self.api_key:
            logger.warning("BENZINGA_API_KEY not set - using stub implementation")

        # Track progress
        self.processed_count = 0
        self.success_count = 0
        self.error_count = 0

    def fetch_ratings_benzinga(
        self,
        ticker: Optional[str] = None,
        limit: int = 500
    ) -> List[Dict[str, Any]]:
        """Fetch ratings from Benzinga API.

        Args:
            ticker: Stock ticker symbol (optional).
            limit: Maximum number of ratings to fetch.

        Returns:
            List of rating dictionaries.
        """
        if not self.api_key:
            logger.info("Benzinga API key not available - returning empty list")
            return []

        url = "https://api.benzinga.com/api/v2/calendar/ratings"

        params = {
            'token': self.api_key,
            'pagesize': min(limit, 500),  # Max 500 per request
        }

        if ticker:
            params['parameters[tickers]'] = ticker

        try:
            response = requests.get(url, params=params, timeout=30)
            response.raise_for_status()

            data = response.json()
            ratings = data.get('ratings', [])

            logger.info(f"Fetched {len(ratings)} ratings from Benzinga")
            return ratings

        except requests.exceptions.RequestException as e:
            logger.error(f"Error fetching ratings from Benzinga: {e}")
            return []
        except Exception as e:
            logger.exception(f"Unexpected error: {e}")
            return []

    def parse_rating(self, rating_data: Dict[str, Any]) -> Optional[Dict[str, Any]]:
        """Parse rating data from API response.

        Args:
            rating_data: Raw rating data dictionary.

        Returns:
            Parsed rating dictionary or None if invalid.
        """
        try:
            ticker = rating_data.get('ticker')
            if not ticker:
                return None

            # Parse date
            date_str = rating_data.get('date')
            if date_str:
                rating_date = datetime.strptime(date_str, '%Y-%m-%d').date()
            else:
                rating_date = datetime.now().date()

            # Extract rating info
            rating_current = rating_data.get('rating_current', '')
            rating_prior = rating_data.get('rating_prior')

            # Map to numeric
            rating_numeric = self.RATING_MAP.get(rating_current)

            # Extract price targets
            pt_current = rating_data.get('pt_current')
            pt_prior = rating_data.get('pt_prior')

            # Determine action
            action = rating_data.get('action', 'Unknown')
            if rating_prior and rating_current != rating_prior:
                if not rating_prior:
                    action = 'Initiated'
                elif self.RATING_MAP.get(rating_current, 0) > self.RATING_MAP.get(rating_prior, 0):
                    action = 'Upgraded'
                elif self.RATING_MAP.get(rating_current, 0) < self.RATING_MAP.get(rating_prior, 0):
                    action = 'Downgraded'
                else:
                    action = 'Reiterated'

            return {
                'ticker': ticker.upper(),
                'rating_date': rating_date,
                'analyst_name': rating_data.get('analyst', 'Unknown'),
                'analyst_firm': rating_data.get('analyst_firm', 'Unknown'),
                'rating': rating_current,
                'rating_numeric': rating_numeric,
                'price_target': pt_current,
                'prior_rating': rating_prior,
                'prior_price_target': pt_prior,
                'action': action,
                'notes': rating_data.get('notes'),
                'source': 'Benzinga',
            }

        except Exception as e:
            logger.error(f"Error parsing rating data: {e}")
            return None

    async def get_tickers_to_process(
        self,
        ticker: Optional[str] = None,
        sp500: bool = False,
        limit: Optional[int] = None
    ) -> List[str]:
        """Get list of tickers to fetch ratings for.

        Args:
            ticker: Single ticker to process.
            sp500: Only S&P 500 stocks.
            limit: Maximum number of tickers.

        Returns:
            List of ticker symbols.
        """
        if ticker:
            return [ticker.upper()]

        async with self.db.session() as session:
            query_str = """
                SELECT ticker
                FROM stocks
                WHERE ticker NOT LIKE '%-%'
                  AND is_active = true
            """

            if sp500:
                query_str += " AND is_sp500 = true"

            query_str += " ORDER BY ticker LIMIT :limit"

            query = text(query_str)
            result = await session.execute(query, {"limit": limit or 500})
            tickers = [row[0] for row in result.fetchall()]

        logger.info(f"Found {len(tickers)} tickers to process")
        return tickers

    async def store_ratings(self, ratings: List[Dict[str, Any]]) -> bool:
        """Store analyst ratings in database.

        Args:
            ratings: List of rating dictionaries.

        Returns:
            True if successful.
        """
        if not ratings:
            return False

        try:
            async with self.db.session() as session:
                # Insert with ON CONFLICT DO NOTHING (avoid duplicates)
                stmt = pg_insert(AnalystRating).values(ratings)
                stmt = stmt.on_conflict_do_nothing()

                await session.execute(stmt)
                await session.commit()

                logger.info(f"Stored {len(ratings)} analyst ratings")
                return True

        except Exception as e:
            logger.error(f"Error storing ratings: {e}", exc_info=True)
            return False

    async def run(
        self,
        limit: int = 500,
        ticker: Optional[str] = None,
        sp500: bool = False
    ):
        """Run the ingestion pipeline.

        Args:
            limit: Maximum number of ratings to fetch.
            ticker: Process single ticker.
            sp500: Only S&P 500 stocks.
        """
        start_time = datetime.now()
        logger.info("=" * 80)
        logger.info("Analyst Ratings Ingestion Pipeline")
        logger.info("=" * 80)

        if not self.api_key:
            logger.warning("=" * 80)
            logger.warning("BENZINGA_API_KEY environment variable not set")
            logger.warning("Analyst ratings ingestion requires a Benzinga API key")
            logger.warning("Sign up for free tier at: https://www.benzinga.com/apis")
            logger.warning("Set the key: export BENZINGA_API_KEY=your_key")
            logger.warning("=" * 80)
            logger.info("Exiting gracefully (no data ingested)")
            return

        if ticker:
            # Fetch ratings for single ticker
            ratings_data = self.fetch_ratings_benzinga(ticker=ticker, limit=limit)
        else:
            # Fetch ratings for multiple tickers
            tickers = await self.get_tickers_to_process(sp500=sp500, limit=min(limit, 100))

            ratings_data = []
            for tick in tqdm(tickers, desc="Fetching ratings"):
                ticker_ratings = self.fetch_ratings_benzinga(ticker=tick, limit=10)
                ratings_data.extend(ticker_ratings)

                if len(ratings_data) >= limit:
                    break

        # Parse ratings
        ratings = []
        for rating_data in ratings_data:
            parsed = self.parse_rating(rating_data)
            if parsed:
                ratings.append(parsed)

        self.processed_count = len(ratings_data)

        # Store ratings
        if ratings:
            success = await self.store_ratings(ratings)
            if success:
                self.success_count = len(ratings)
        else:
            logger.warning("No ratings to store")

        # Print summary
        duration = datetime.now() - start_time
        logger.info("=" * 80)
        logger.info("Ingestion Complete")
        logger.info("=" * 80)
        logger.info(f"Duration: {duration}")
        logger.info(f"Processed: {self.processed_count} ratings")
        logger.info(f"Stored: {self.success_count} ratings")


def main():
    """Main entry point for CLI."""
    parser = argparse.ArgumentParser(
        description='Ingest analyst ratings and price targets',
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  # Fetch 500 ratings (free tier daily limit)
  python analyst_ratings_ingestion.py --limit 500

  # Fetch ratings for single ticker
  python analyst_ratings_ingestion.py --ticker AAPL

  # Fetch ratings for S&P 500 stocks only
  python analyst_ratings_ingestion.py --sp500 --limit 500
        """
    )

    parser.add_argument(
        '--limit',
        type=int,
        default=500,
        help='Maximum number of ratings to fetch (default: 500)'
    )
    parser.add_argument(
        '--ticker',
        type=str,
        help='Process single ticker symbol'
    )
    parser.add_argument(
        '--sp500',
        action='store_true',
        help='Only fetch ratings for S&P 500 stocks'
    )

    args = parser.parse_args()

    # Run pipeline
    pipeline = AnalystRatingsIngestion()
    asyncio.run(pipeline.run(
        limit=args.limit,
        ticker=args.ticker,
        sp500=args.sp500
    ))


if __name__ == '__main__':
    main()
