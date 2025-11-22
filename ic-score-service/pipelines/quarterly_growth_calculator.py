#!/usr/bin/env python3
"""Quarterly Growth Metrics Calculator Pipeline.

This script calculates quarter-over-quarter (QoQ) growth metrics by comparing
current quarter to previous quarter for key financial metrics:
- Revenue growth
- EPS growth
- EBITDA growth
- Net income growth

Usage:
    python quarterly_growth_calculator.py --limit 100    # Test on 100 stocks
    python quarterly_growth_calculator.py --all          # All stocks
    python quarterly_growth_calculator.py --ticker AAPL  # Single stock
"""

import argparse
import asyncio
import logging
import sys
from datetime import date, datetime
from pathlib import Path
from typing import Dict, List, Optional

# Add parent directory to path for imports
sys.path.insert(0, str(Path(__file__).parent.parent))

from sqlalchemy import text
from tqdm import tqdm

from database.database import get_database

# Setup logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s',
    handlers=[
        logging.StreamHandler(),
        logging.FileHandler('/app/logs/quarterly_growth_calculator.log')
    ]
)
logger = logging.getLogger(__name__)


class QuarterlyGrowthCalculator:
    """Calculator for quarter-over-quarter growth metrics."""

    def __init__(self):
        """Initialize the calculator."""
        self.db = get_database()

        # Track progress
        self.processed_count = 0
        self.calculated_count = 0
        self.skipped_count = 0
        self.error_count = 0

    async def get_tickers_to_process(
        self,
        limit: Optional[int] = None,
        ticker: Optional[str] = None
    ) -> List[str]:
        """Get list of tickers to process.

        Args:
            limit: Maximum number of tickers to process.
            ticker: Single ticker to process.

        Returns:
            List of ticker symbols.
        """
        async with self.db.session() as session:
            if ticker:
                return [ticker.upper()]

            query = text("""
                SELECT DISTINCT ticker
                FROM financials
                WHERE ticker NOT LIKE '%-%'
                  AND ticker NOT LIKE '%.%'
                  AND statement_type = '10-Q'
                ORDER BY ticker
                LIMIT :limit
            """)
            result = await session.execute(query, {"limit": limit or 10000})
            tickers = [row[0] for row in result.fetchall()]

        logger.info(f"Found {len(tickers)} tickers with quarterly data")
        return tickers

    async def get_last_two_quarters(self, ticker: str) -> Optional[tuple]:
        """Get last two quarters of financial data for a ticker.

        Args:
            ticker: Stock ticker symbol.

        Returns:
            Tuple of (current_quarter, previous_quarter) dicts, or None if insufficient data.
        """
        async with self.db.session() as session:
            query = text("""
                SELECT
                    id, period_end_date, fiscal_year, fiscal_quarter,
                    revenue, net_income, eps_diluted, ebitda
                FROM financials
                WHERE ticker = :ticker
                  AND statement_type = '10-Q'
                  AND fiscal_quarter IS NOT NULL
                ORDER BY period_end_date DESC
                LIMIT 2
            """)
            result = await session.execute(query, {"ticker": ticker})
            rows = result.fetchall()

            if len(rows) < 2:
                return None

            current_q = {
                'id': rows[0][0],
                'period_end_date': rows[0][1],
                'fiscal_year': rows[0][2],
                'fiscal_quarter': rows[0][3],
                'revenue': int(rows[0][4]) if rows[0][4] else None,
                'net_income': int(rows[0][5]) if rows[0][5] else None,
                'eps_diluted': float(rows[0][6]) if rows[0][6] else None,
                'ebitda': int(rows[0][7]) if rows[0][7] else None,
            }

            previous_q = {
                'id': rows[1][0],
                'period_end_date': rows[1][1],
                'fiscal_year': rows[1][2],
                'fiscal_quarter': rows[1][3],
                'revenue': int(rows[1][4]) if rows[1][4] else None,
                'net_income': int(rows[1][5]) if rows[1][5] else None,
                'eps_diluted': float(rows[1][6]) if rows[1][6] else None,
                'ebitda': int(rows[1][7]) if rows[1][7] else None,
            }

            return (current_q, previous_q)

    def calculate_qoq_growth(
        self,
        current: Optional[float],
        previous: Optional[float]
    ) -> Optional[float]:
        """Calculate quarter-over-quarter growth percentage.

        Args:
            current: Current quarter value.
            previous: Previous quarter value.

        Returns:
            Growth percentage, or None if calculation not possible.
        """
        if current is None or previous is None:
            return None

        # Handle edge cases
        if previous == 0:
            # Avoid division by zero
            return None

        # Negative to positive or positive to negative transitions
        if (previous < 0 and current > 0) or (previous > 0 and current < 0):
            # These transitions are hard to interpret meaningfully
            return None

        # Calculate growth
        growth = ((current - previous) / abs(previous)) * 100
        return round(growth, 2)

    async def calculate_growth_metrics(self, ticker: str) -> Optional[Dict]:
        """Calculate QoQ growth metrics for a ticker.

        Args:
            ticker: Stock ticker symbol.

        Returns:
            Dict with calculated growth metrics, or None if insufficient data.
        """
        quarters = await self.get_last_two_quarters(ticker)
        if not quarters:
            logger.debug(f"{ticker}: Need at least 2 quarters for QoQ calculation")
            return None

        current_q, previous_q = quarters

        # Calculate growth for each metric
        qoq_revenue_growth = self.calculate_qoq_growth(
            current_q['revenue'],
            previous_q['revenue']
        )

        qoq_eps_growth = self.calculate_qoq_growth(
            current_q['eps_diluted'],
            previous_q['eps_diluted']
        )

        qoq_ebitda_growth = self.calculate_qoq_growth(
            current_q['ebitda'],
            previous_q['ebitda']
        )

        qoq_net_income_growth = self.calculate_qoq_growth(
            current_q['net_income'],
            previous_q['net_income']
        )

        growth_data = {
            'ticker': ticker,
            'financial_id': current_q['id'],
            'period_end_date': current_q['period_end_date'],
            # Revenue
            'qoq_revenue_growth': qoq_revenue_growth,
            'revenue_current': current_q['revenue'],
            'revenue_previous': previous_q['revenue'],
            # EPS
            'qoq_eps_growth': qoq_eps_growth,
            'eps_current': current_q['eps_diluted'],
            'eps_previous': previous_q['eps_diluted'],
            # EBITDA
            'qoq_ebitda_growth': qoq_ebitda_growth,
            'ebitda_current': current_q['ebitda'],
            'ebitda_previous': previous_q['ebitda'],
            # Net Income
            'qoq_net_income_growth': qoq_net_income_growth,
            'net_income_current': current_q['net_income'],
            'net_income_previous': previous_q['net_income'],
            # Metadata
            'calculation_date': date.today(),
        }

        return growth_data

    async def store_growth_metrics(self, growth_data: Dict) -> bool:
        """Store quarterly growth metrics in database.

        Args:
            growth_data: Dict with calculated growth metrics.

        Returns:
            True if successful, False otherwise.
        """
        try:
            async with self.db.session() as session:
                query = text("""
                    INSERT INTO quarterly_growth_metrics (
                        ticker, financial_id, period_end_date,
                        qoq_revenue_growth, revenue_current, revenue_previous,
                        qoq_eps_growth, eps_current, eps_previous,
                        qoq_ebitda_growth, ebitda_current, ebitda_previous,
                        qoq_net_income_growth, net_income_current, net_income_previous,
                        calculation_date
                    ) VALUES (
                        :ticker, :financial_id, :period_end_date,
                        :qoq_revenue_growth, :revenue_current, :revenue_previous,
                        :qoq_eps_growth, :eps_current, :eps_previous,
                        :qoq_ebitda_growth, :ebitda_current, :ebitda_previous,
                        :qoq_net_income_growth, :net_income_current, :net_income_previous,
                        :calculation_date
                    )
                    ON CONFLICT (ticker, period_end_date)
                    DO UPDATE SET
                        financial_id = EXCLUDED.financial_id,
                        qoq_revenue_growth = EXCLUDED.qoq_revenue_growth,
                        revenue_current = EXCLUDED.revenue_current,
                        revenue_previous = EXCLUDED.revenue_previous,
                        qoq_eps_growth = EXCLUDED.qoq_eps_growth,
                        eps_current = EXCLUDED.eps_current,
                        eps_previous = EXCLUDED.eps_previous,
                        qoq_ebitda_growth = EXCLUDED.qoq_ebitda_growth,
                        ebitda_current = EXCLUDED.ebitda_current,
                        ebitda_previous = EXCLUDED.ebitda_previous,
                        qoq_net_income_growth = EXCLUDED.qoq_net_income_growth,
                        net_income_current = EXCLUDED.net_income_current,
                        net_income_previous = EXCLUDED.net_income_previous,
                        calculation_date = EXCLUDED.calculation_date
                """)

                await session.execute(query, growth_data)
                await session.commit()

                return True

        except Exception as e:
            logger.error(f"Error storing growth metrics for {growth_data['ticker']}: {e}", exc_info=True)
            return False

    async def process_ticker(self, ticker: str) -> bool:
        """Process a single ticker to calculate growth metrics.

        Args:
            ticker: Stock ticker symbol.

        Returns:
            True if successful, False otherwise.
        """
        try:
            # Calculate growth metrics
            growth_data = await self.calculate_growth_metrics(ticker)

            if not growth_data:
                logger.debug(f"{ticker}: Insufficient data for growth calculation")
                return False

            # Store in database
            success = await self.store_growth_metrics(growth_data)

            if success:
                logger.info(
                    f"{ticker}: QoQ Growth - "
                    f"Revenue: {growth_data.get('qoq_revenue_growth')}%, "
                    f"EPS: {growth_data.get('qoq_eps_growth')}%, "
                    f"EBITDA: {growth_data.get('qoq_ebitda_growth')}%, "
                    f"Net Income: {growth_data.get('qoq_net_income_growth')}%"
                )

            return success

        except Exception as e:
            logger.error(f"{ticker}: Error processing: {e}", exc_info=True)
            return False

    async def process_batch(
        self,
        tickers: List[str],
        batch_size: int = 50
    ):
        """Process tickers in batches with progress tracking.

        Args:
            tickers: List of ticker symbols.
            batch_size: Number of tickers to process concurrently.
        """
        progress_bar = tqdm(total=len(tickers), desc="Calculating QoQ Growth")

        for i in range(0, len(tickers), batch_size):
            batch = tickers[i:i + batch_size]

            # Process batch concurrently
            tasks = [self.process_ticker(ticker) for ticker in batch]
            results = await asyncio.gather(*tasks, return_exceptions=True)

            # Update counters
            for ticker, result in zip(batch, results):
                self.processed_count += 1

                if isinstance(result, Exception):
                    logger.error(f"{ticker}: Exception - {result}")
                    self.error_count += 1
                elif result:
                    self.calculated_count += 1
                else:
                    self.skipped_count += 1

                progress_bar.update(1)
                progress_bar.set_postfix({
                    'calculated': self.calculated_count,
                    'errors': self.error_count,
                    'skipped': self.skipped_count
                })

        progress_bar.close()

    async def run(
        self,
        limit: Optional[int] = None,
        ticker: Optional[str] = None,
        all_tickers: bool = False
    ):
        """Run the quarterly growth calculator pipeline.

        Args:
            limit: Limit number of tickers to process.
            ticker: Process single ticker.
            all_tickers: Process all tickers with quarterly data.
        """
        start_time = datetime.now()
        logger.info("=" * 80)
        logger.info("Quarterly Growth Metrics Calculator")
        logger.info("=" * 80)

        # Determine limit
        if all_tickers:
            limit = None
        elif ticker:
            limit = 1
        elif limit is None:
            limit = 100  # Default safe limit for testing

        # Get tickers to process
        tickers = await self.get_tickers_to_process(limit=limit, ticker=ticker)

        if not tickers:
            logger.info("No tickers to process")
            return

        logger.info(f"Processing {len(tickers)} tickers...")

        # Process in batches
        await self.process_batch(tickers, batch_size=50)

        # Print summary
        duration = datetime.now() - start_time
        logger.info("=" * 80)
        logger.info("QoQ Growth Calculation Complete")
        logger.info("=" * 80)
        logger.info(f"Duration: {duration}")
        logger.info(f"Processed: {self.processed_count}")
        logger.info(f"Calculated: {self.calculated_count}")
        logger.info(f"Skipped: {self.skipped_count}")
        logger.info(f"Errors: {self.error_count}")
        if self.processed_count > 0:
            logger.info(f"Success Rate: {(self.calculated_count/self.processed_count*100):.1f}%")


def main():
    """Main entry point for CLI."""
    parser = argparse.ArgumentParser(
        description='Calculate quarter-over-quarter growth metrics for stocks',
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  # Test on 10 tickers
  python quarterly_growth_calculator.py --limit 10

  # Process all tickers
  python quarterly_growth_calculator.py --all

  # Process single ticker
  python quarterly_growth_calculator.py --ticker AAPL
        """
    )

    parser.add_argument(
        '--limit',
        type=int,
        help='Limit number of tickers to process (default: 100)'
    )
    parser.add_argument(
        '--ticker',
        type=str,
        help='Process single ticker symbol'
    )
    parser.add_argument(
        '--all',
        action='store_true',
        help='Process all tickers with quarterly data'
    )

    args = parser.parse_args()

    # Validate arguments
    if args.ticker and (args.all or args.limit):
        parser.error("--ticker cannot be used with --all or --limit")

    # Run calculator
    calculator = QuarterlyGrowthCalculator()
    asyncio.run(calculator.run(
        limit=args.limit,
        ticker=args.ticker,
        all_tickers=args.all
    ))


if __name__ == '__main__':
    main()
