#!/usr/bin/env python3
"""Valuation Ratios Calculator Pipeline with Polygon API Integration.

This script calculates valuation ratios (P/E, P/B, P/S, P/FCF, EV/EBITDA) by combining:
- TTM financial data from ttm_financials table
- Latest stock prices from Polygon.io API (real-time)
- Shares outstanding from financials table

Ratios are stored in the valuation_ratios table with timestamps for historical tracking.

Usage:
    python valuation_ratios_calculator.py --limit 100    # Test on 100 stocks
    python valuation_ratios_calculator.py --all          # All stocks
    python valuation_ratios_calculator.py --ticker AAPL  # Single stock
"""

import argparse
import asyncio
import logging
import os
import sys
from datetime import datetime, date
from decimal import Decimal
from pathlib import Path
from typing import List, Optional, Dict, Any

# Add parent directory to path for imports
sys.path.insert(0, str(Path(__file__).parent.parent))

from sqlalchemy import text
from tqdm import tqdm

from database.database import get_database
from pipelines.utils.polygon_client import PolygonClient

# Setup logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s',
    handlers=[
        logging.StreamHandler(),
        logging.FileHandler('/app/logs/valuation_ratios_calculator.log')
    ]
)
logger = logging.getLogger(__name__)


class ValuationRatiosCalculator:
    """Calculator for stock valuation ratios using TTM financials + Polygon API prices."""

    def __init__(self):
        """Initialize the calculator."""
        self.db = get_database()

        # Initialize Polygon client
        api_key = os.getenv('POLYGON_API_KEY')
        if not api_key:
            logger.warning("POLYGON_API_KEY not set - will fail to fetch prices")
        self.polygon = PolygonClient(api_key) if api_key else None

        # Track progress
        self.processed_count = 0
        self.success_count = 0
        self.error_count = 0
        self.skipped_count = 0

    async def get_stocks_to_process(
        self,
        limit: Optional[int] = None,
        ticker: Optional[str] = None
    ) -> List[str]:
        """Get list of stocks to process from TTM financials table.

        Args:
            limit: Maximum number of stocks to process.
            ticker: Single ticker to process.

        Returns:
            List of ticker symbols.
        """
        async with self.db.session() as session:
            if ticker:
                return [ticker.upper()]

            # Get stocks that have TTM financials (our source of truth)
            query = text("""
                SELECT DISTINCT ticker
                FROM ttm_financials
                WHERE ticker NOT LIKE '%-%'
                  AND ticker NOT LIKE '%.%'
                ORDER BY ticker
                LIMIT :limit
            """)
            result = await session.execute(query, {"limit": limit or 10000})
            tickers = [row[0] for row in result.fetchall()]

        logger.info(f"Found {len(tickers)} stocks with TTM financial data")
        return tickers

    def get_polygon_price(self, ticker: str) -> Optional[Dict[str, Any]]:
        """Get latest stock price from Polygon API.

        Args:
            ticker: Stock ticker symbol.

        Returns:
            Dictionary with price and date, or None if not found.
        """
        if not self.polygon:
            logger.error("Polygon client not initialized")
            return None

        try:
            price_data = self.polygon.get_latest_price(ticker)

            if not price_data or not price_data.get('close'):
                logger.warning(f"{ticker}: No price data from Polygon")
                return None

            return {
                'close': float(price_data['close']),
                'date': price_data['date'] if isinstance(price_data['date'], date) else date.today()
            }

        except Exception as e:
            logger.error(f"{ticker}: Error fetching Polygon price: {e}")
            return None

    async def get_ttm_financials(self, ticker: str) -> Optional[Dict[str, Any]]:
        """Get latest TTM financial data for a stock.

        Args:
            ticker: Stock ticker symbol.

        Returns:
            Dictionary with TTM financial data, or None if not found.
        """
        async with self.db.session() as session:
            query = text("""
                SELECT
                    id,
                    calculation_date,
                    ttm_period_start,
                    ttm_period_end,
                    revenue,
                    net_income,
                    eps_diluted,
                    shares_outstanding,
                    shareholders_equity
                FROM ttm_financials
                WHERE ticker = :ticker
                ORDER BY calculation_date DESC
                LIMIT 1
            """)
            result = await session.execute(query, {"ticker": ticker})
            row = result.fetchone()

            if not row:
                return None

            return {
                'id': row[0],
                'calculation_date': row[1],
                'ttm_period_start': row[2],
                'ttm_period_end': row[3],
                'revenue': int(row[4]) if row[4] else None,
                'net_income': int(row[5]) if row[5] else None,
                'eps_diluted': float(row[6]) if row[6] else None,
                'shares_outstanding': int(row[7]) if row[7] else None,
                'shareholders_equity': int(row[8]) if row[8] else None,
            }

    def calculate_ratios(
        self,
        ticker: str,
        price_data: Dict[str, Any],
        ttm_data: Dict[str, Any]
    ) -> Optional[Dict[str, Any]]:
        """Calculate valuation ratios using Polygon price and TTM financials.

        Args:
            ticker: Stock ticker symbol.
            price_data: Latest stock price data from Polygon.
            ttm_data: TTM financial data.

        Returns:
            Dictionary of calculated ratios, or None if insufficient data.
        """
        try:
            stock_price = price_data['close']
            shares_outstanding = ttm_data.get('shares_outstanding')

            # Initialize ratios dictionary
            ratios = {
                'ticker': ticker,
                'ttm_financial_id': ttm_data['id'],
                'calculation_date': date.today(),
                'stock_price': round(stock_price, 2),
                'ttm_period_start': ttm_data.get('ttm_period_start'),
                'ttm_period_end': ttm_data.get('ttm_period_end'),
            }

            # Calculate market cap
            if shares_outstanding and shares_outstanding > 0:
                market_cap = int(stock_price * shares_outstanding)
                ratios['ttm_market_cap'] = market_cap
            else:
                logger.warning(f"{ticker}: No shares outstanding data")
                market_cap = None
                ratios['ttm_market_cap'] = None

            # P/E Ratio (using TTM EPS)
            eps = ttm_data.get('eps_diluted')
            if eps and eps > 0:
                pe_ratio = stock_price / eps
                ratios['ttm_pe_ratio'] = round(pe_ratio, 2)
            else:
                ratios['ttm_pe_ratio'] = None

            # P/B Ratio (Price to Book)
            shareholders_equity = ttm_data.get('shareholders_equity')
            if shares_outstanding and shareholders_equity and shareholders_equity > 0:
                book_value_per_share = shareholders_equity / shares_outstanding
                if book_value_per_share > 0:
                    pb_ratio = stock_price / book_value_per_share
                    ratios['ttm_pb_ratio'] = round(pb_ratio, 2)
                else:
                    ratios['ttm_pb_ratio'] = None
            else:
                ratios['ttm_pb_ratio'] = None

            # P/S Ratio (Price to Sales)
            revenue = ttm_data.get('revenue')
            if market_cap and revenue and revenue > 0:
                ps_ratio = market_cap / revenue
                ratios['ttm_ps_ratio'] = round(ps_ratio, 4)
            else:
                ratios['ttm_ps_ratio'] = None

            return ratios

        except Exception as e:
            logger.error(f"{ticker}: Error calculating ratios: {e}", exc_info=True)
            return None

    async def store_ratios(self, ratios: Dict[str, Any]) -> bool:
        """Store calculated ratios in valuation_ratios table.

        Args:
            ratios: Dictionary of calculated ratios.

        Returns:
            True if successful, False otherwise.
        """
        try:
            async with self.db.session() as session:
                # UPSERT: Insert or update if ticker already exists for today
                query = text("""
                    INSERT INTO valuation_ratios (
                        ticker,
                        ttm_financial_id,
                        calculation_date,
                        stock_price,
                        ttm_market_cap,
                        ttm_pe_ratio,
                        ttm_pb_ratio,
                        ttm_ps_ratio,
                        ttm_period_start,
                        ttm_period_end
                    ) VALUES (
                        :ticker,
                        :ttm_financial_id,
                        :calculation_date,
                        :stock_price,
                        :ttm_market_cap,
                        :ttm_pe_ratio,
                        :ttm_pb_ratio,
                        :ttm_ps_ratio,
                        :ttm_period_start,
                        :ttm_period_end
                    )
                    ON CONFLICT (ticker, calculation_date)
                    DO UPDATE SET
                        ttm_financial_id = EXCLUDED.ttm_financial_id,
                        stock_price = EXCLUDED.stock_price,
                        ttm_market_cap = EXCLUDED.ttm_market_cap,
                        ttm_pe_ratio = EXCLUDED.ttm_pe_ratio,
                        ttm_pb_ratio = EXCLUDED.ttm_pb_ratio,
                        ttm_ps_ratio = EXCLUDED.ttm_ps_ratio,
                        ttm_period_start = EXCLUDED.ttm_period_start,
                        ttm_period_end = EXCLUDED.ttm_period_end
                """)

                await session.execute(query, ratios)
                await session.commit()

                return True

        except Exception as e:
            logger.error(f"Error storing ratios: {e}", exc_info=True)
            return False

    async def process_ticker(self, ticker: str) -> bool:
        """Process a single ticker: calculate and store valuation ratios.

        Args:
            ticker: Stock ticker symbol.

        Returns:
            True if successful, False otherwise.
        """
        try:
            # Get TTM financial data
            ttm_data = await self.get_ttm_financials(ticker)
            if not ttm_data:
                logger.warning(f"{ticker}: No TTM financial data found")
                return False

            # Get latest stock price from Polygon
            price_data = self.get_polygon_price(ticker)
            if not price_data:
                logger.warning(f"{ticker}: No price data from Polygon")
                return False

            # Calculate ratios
            ratios = self.calculate_ratios(ticker, price_data, ttm_data)
            if not ratios:
                return False

            # Store in database
            success = await self.store_ratios(ratios)

            if success:
                logger.info(
                    f"{ticker}: Updated ratios - "
                    f"P/E: {ratios.get('ttm_pe_ratio')}, "
                    f"P/B: {ratios.get('ttm_pb_ratio')}, "
                    f"P/S: {ratios.get('ttm_ps_ratio')}, "
                    f"Market Cap: ${ratios.get('ttm_market_cap'):,} "
                    f"Price: ${ratios['stock_price']}"
                )

            return success

        except Exception as e:
            logger.error(f"{ticker}: Error processing: {e}", exc_info=True)
            return False

    async def process_stocks(self, tickers: List[str], show_progress: bool = True):
        """Process a list of tickers.

        Args:
            tickers: List of ticker symbols.
            show_progress: Show progress bar.
        """
        progress_bar = tqdm(total=len(tickers), desc="Calculating valuation ratios") if show_progress else None

        for ticker in tickers:
            success = await self.process_ticker(ticker)

            self.processed_count += 1
            if success:
                self.success_count += 1
            else:
                self.error_count += 1

            if progress_bar:
                progress_bar.update(1)
                progress_bar.set_postfix({
                    'success': self.success_count,
                    'errors': self.error_count
                })

        if progress_bar:
            progress_bar.close()

    async def run(
        self,
        limit: Optional[int] = None,
        ticker: Optional[str] = None,
        all_stocks: bool = False
    ):
        """Run the valuation ratios calculator pipeline.

        Args:
            limit: Limit number of stocks to process.
            ticker: Process single ticker.
            all_stocks: Process all stocks.
        """
        start_time = datetime.now()
        logger.info("=" * 80)
        logger.info("Valuation Ratios Calculator Pipeline (Polygon API + TTM Financials)")
        logger.info("=" * 80)

        # Determine limit
        if all_stocks:
            limit = None
        elif ticker:
            limit = 1
        elif limit is None:
            limit = 10  # Default to 10 for safety

        # Get stocks to process
        tickers = await self.get_stocks_to_process(limit=limit, ticker=ticker)

        if not tickers:
            logger.warning("No stocks to process")
            return

        logger.info(f"Processing {len(tickers)} stocks...")

        # Process stocks
        await self.process_stocks(tickers)

        # Print summary
        duration = datetime.now() - start_time
        logger.info("=" * 80)
        logger.info("Valuation Ratios Calculation Complete")
        logger.info("=" * 80)
        logger.info(f"Duration: {duration}")
        logger.info(f"Processed: {self.processed_count}")
        logger.info(f"Success: {self.success_count}")
        logger.info(f"Errors: {self.error_count}")
        if self.processed_count > 0:
            logger.info(f"Success Rate: {(self.success_count/self.processed_count*100):.1f}%")

    def __del__(self):
        """Cleanup Polygon client."""
        if hasattr(self, 'polygon') and self.polygon:
            self.polygon.close()


def main():
    """Main entry point for CLI."""
    parser = argparse.ArgumentParser(
        description='Calculate valuation ratios using Polygon API + TTM financials',
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  # Test on 10 stocks (default)
  python valuation_ratios_calculator.py

  # Test on 100 stocks
  python valuation_ratios_calculator.py --limit 100

  # Process all stocks
  python valuation_ratios_calculator.py --all

  # Process single stock
  python valuation_ratios_calculator.py --ticker AAPL
        """
    )

    parser.add_argument(
        '--limit',
        type=int,
        help='Limit number of stocks to process (default: 10)'
    )
    parser.add_argument(
        '--ticker',
        type=str,
        help='Process single ticker symbol'
    )
    parser.add_argument(
        '--all',
        action='store_true',
        help='Process all stocks in database'
    )

    args = parser.parse_args()

    # Validate arguments
    if args.ticker and (args.all or args.limit):
        parser.error("--ticker cannot be used with --all or --limit")

    # Run pipeline
    calculator = ValuationRatiosCalculator()
    asyncio.run(calculator.run(
        limit=args.limit,
        ticker=args.ticker,
        all_stocks=args.all
    ))


if __name__ == '__main__':
    main()
