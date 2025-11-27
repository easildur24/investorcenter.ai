#!/usr/bin/env python3
"""SEC Financial Statements Ingestion Pipeline.

This script fetches financial data from SEC EDGAR for US stocks and populates
the financials table. It retrieves 5 years of quarterly data for each stock.

Usage:
    python sec_financials_ingestion.py --limit 100    # Test on 100 stocks
    python sec_financials_ingestion.py --all          # All 4,600 stocks
    python sec_financials_ingestion.py --ticker AAPL  # Single stock
    python sec_financials_ingestion.py --resume       # Resume from last run
"""

import argparse
import asyncio
import logging
import sys
from datetime import datetime, date
from pathlib import Path
from typing import List, Optional, Dict, Any

# Add parent directory to path for imports
sys.path.insert(0, str(Path(__file__).parent.parent))

from sqlalchemy import text, select, insert
from sqlalchemy.dialects.postgresql import insert as pg_insert
from sqlalchemy.ext.asyncio import AsyncSession
from tqdm import tqdm

from database.database import get_database
from models import Financial
from pipelines.utils.sec_client import SECClient

# Setup logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s',
    handlers=[
        logging.StreamHandler(),
        logging.FileHandler('/app/logs/sec_financials_ingestion.log')
    ]
)
logger = logging.getLogger(__name__)


class SECFinancialsIngestion:
    """Ingestion pipeline for SEC financial statements."""

    def __init__(self, batch_size: int = 100):
        """Initialize the ingestion pipeline.

        Args:
            batch_size: Number of stocks to process in each batch.
        """
        self.batch_size = batch_size
        self.sec_client = SECClient()
        self.db = get_database()

        # Track progress
        self.processed_count = 0
        self.success_count = 0
        self.error_count = 0
        self.skipped_count = 0

    async def get_stocks_to_process(
        self,
        limit: Optional[int] = None,
        ticker: Optional[str] = None,
        resume: bool = False
    ) -> List[Dict[str, Any]]:
        """Get list of stocks to process from database.

        Args:
            limit: Maximum number of stocks to process.
            ticker: Single ticker to process.
            resume: Resume from stocks not yet processed.

        Returns:
            List of stock dictionaries with ticker and CIK.
        """
        async with self.db.session() as session:
            if ticker:
                # Single ticker
                query = text("""
                    SELECT symbol AS ticker, cik
                    FROM tickers
                    WHERE symbol = :ticker AND cik IS NOT NULL
                """)
                result = await session.execute(query, {"ticker": ticker.upper()})
            elif resume:
                # Resume from stocks without recent financial data
                query = text("""
                    SELECT s.symbol AS ticker, s.cik
                    FROM tickers s
                    LEFT JOIN (
                        SELECT DISTINCT ticker
                        FROM financials
                        WHERE created_at > NOW() - INTERVAL '7 days'
                    ) f ON s.symbol = f.ticker
                    WHERE s.cik IS NOT NULL
                      AND f.ticker IS NULL
                      AND s.symbol NOT LIKE '%-%'
                    ORDER BY s.symbol
                    LIMIT :limit
                """)
                result = await session.execute(query, {"limit": limit or 10000})
            else:
                # All stocks or limited
                query = text("""
                    SELECT symbol AS ticker, cik
                    FROM tickers
                    WHERE cik IS NOT NULL
                      AND symbol NOT LIKE '%-%'
                    ORDER BY symbol
                    LIMIT :limit
                """)
                result = await session.execute(query, {"limit": limit or 10000})

            stocks = [{"ticker": row[0], "cik": row[1]} for row in result.fetchall()]

        logger.info(f"Found {len(stocks)} stocks to process")
        return stocks

    async def fetch_and_store_financials(
        self,
        ticker: str,
        cik: str,
        session: AsyncSession
    ) -> bool:
        """Fetch financial data for a stock and store in database.

        Args:
            ticker: Stock ticker symbol.
            cik: Company CIK number.
            session: Database session.

        Returns:
            True if successful, False otherwise.
        """
        try:
            # Fetch financial data from SEC (5 years * 4 quarters = 20 periods)
            financials = self.sec_client.get_financials_for_ticker(cik, num_periods=20)

            if not financials:
                logger.warning(f"{ticker}: No financial data found")
                return False

            # Prepare records for insertion
            records = []
            for financial in financials:
                # Parse dates
                try:
                    period_end = datetime.strptime(financial['period_end_date'], '%Y-%m-%d').date()
                    filing_date = datetime.strptime(financial['filing_date'], '%Y-%m-%d').date()
                except Exception as e:
                    logger.error(f"{ticker}: Error parsing dates: {e}")
                    continue

                record = {
                    'ticker': ticker,
                    'filing_date': filing_date,
                    'period_end_date': period_end,
                    'fiscal_year': financial['fiscal_year'],
                    'fiscal_quarter': financial.get('fiscal_quarter'),
                    'statement_type': financial.get('statement_type'),

                    # Income Statement
                    'revenue': financial.get('revenue'),
                    'cost_of_revenue': financial.get('cost_of_revenue'),
                    'gross_profit': financial.get('gross_profit'),
                    'operating_expenses': financial.get('operating_expenses'),
                    'operating_income': financial.get('operating_income'),
                    'net_income': financial.get('net_income'),
                    'eps_basic': financial.get('eps_basic'),
                    'eps_diluted': financial.get('eps_diluted'),
                    'shares_outstanding': financial.get('shares_outstanding'),

                    # Balance Sheet
                    'total_assets': financial.get('total_assets'),
                    'total_liabilities': financial.get('total_liabilities'),
                    'shareholders_equity': financial.get('shareholders_equity'),
                    'cash_and_equivalents': financial.get('cash_and_equivalents'),
                    'short_term_debt': financial.get('short_term_debt'),
                    'long_term_debt': financial.get('long_term_debt'),

                    # Cash Flow
                    'operating_cash_flow': financial.get('operating_cash_flow'),
                    'investing_cash_flow': financial.get('investing_cash_flow'),
                    'financing_cash_flow': financial.get('financing_cash_flow'),
                    'free_cash_flow': financial.get('free_cash_flow'),
                    'capex': financial.get('capex'),

                    # Calculated Metrics
                    'pe_ratio': financial.get('pe_ratio'),
                    'pb_ratio': financial.get('pb_ratio'),
                    'ps_ratio': financial.get('ps_ratio'),
                    'debt_to_equity': financial.get('debt_to_equity'),
                    'current_ratio': financial.get('current_ratio'),
                    'quick_ratio': financial.get('quick_ratio'),
                    'roe': financial.get('roe'),
                    'roa': financial.get('roa'),
                    'roic': financial.get('roic'),
                    'gross_margin': financial.get('gross_margin'),
                    'operating_margin': financial.get('operating_margin'),
                    'net_margin': financial.get('net_margin'),

                    # Metadata
                    'raw_data': financial,
                }

                # Calculate EPS if not provided by SEC but we have the data
                net_income = record.get('net_income')
                shares_outstanding = record.get('shares_outstanding')

                if net_income is not None and shares_outstanding is not None and shares_outstanding > 0:
                    # Calculate EPS = Net Income / Shares Outstanding
                    calculated_eps = round(net_income / shares_outstanding, 4)

                    # Only override if SEC didn't provide EPS
                    if record.get('eps_basic') is None:
                        record['eps_basic'] = calculated_eps
                    if record.get('eps_diluted') is None:
                        record['eps_diluted'] = calculated_eps

                records.append(record)

            if not records:
                logger.warning(f"{ticker}: No valid records to insert")
                return False

            # De-duplicate records based on unique constraint (ticker, period_end_date, fiscal_quarter)
            # Keep the last occurrence (most recent filing typically has most accurate data)
            unique_records = {}
            for record in records:
                key = (record['ticker'], record['period_end_date'], record['fiscal_quarter'])
                unique_records[key] = record

            deduplicated_records = list(unique_records.values())
            logger.info(f"{ticker}: De-duplicated {len(records)} records to {len(deduplicated_records)}")

            # Insert with ON CONFLICT DO UPDATE to handle duplicates
            stmt = pg_insert(Financial).values(deduplicated_records)
            stmt = stmt.on_conflict_do_update(
                index_elements=['ticker', 'period_end_date', 'fiscal_quarter'],
                set_={
                    'revenue': stmt.excluded.revenue,
                    'net_income': stmt.excluded.net_income,
                    'total_assets': stmt.excluded.total_assets,
                    'shareholders_equity': stmt.excluded.shareholders_equity,
                    'operating_cash_flow': stmt.excluded.operating_cash_flow,
                    'gross_margin': stmt.excluded.gross_margin,
                    'operating_margin': stmt.excluded.operating_margin,
                    'net_margin': stmt.excluded.net_margin,
                    'roe': stmt.excluded.roe,
                    'roa': stmt.excluded.roa,
                    'debt_to_equity': stmt.excluded.debt_to_equity,
                    'eps_basic': stmt.excluded.eps_basic,
                    'eps_diluted': stmt.excluded.eps_diluted,
                    'raw_data': stmt.excluded.raw_data,
                }
            )

            await session.execute(stmt)
            await session.commit()

            logger.info(f"{ticker}: Successfully inserted {len(deduplicated_records)} financial records")
            return True

        except Exception as e:
            logger.error(f"{ticker}: Error processing financials: {e}", exc_info=True)
            await session.rollback()
            return False

    async def process_stocks(
        self,
        stocks: List[Dict[str, Any]],
        show_progress: bool = True
    ):
        """Process a list of stocks.

        Args:
            stocks: List of stock dictionaries with ticker and CIK.
            show_progress: Show progress bar.
        """
        progress_bar = tqdm(total=len(stocks), desc="Processing stocks") if show_progress else None

        for stock in stocks:
            ticker = stock['ticker']
            cik = stock['cik']

            async with self.db.session() as session:
                success = await self.fetch_and_store_financials(ticker, cik, session)

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
        all_stocks: bool = False,
        resume: bool = False
    ):
        """Run the ingestion pipeline.

        Args:
            limit: Limit number of stocks to process.
            ticker: Process single ticker.
            all_stocks: Process all stocks.
            resume: Resume from last run.
        """
        start_time = datetime.now()
        logger.info("=" * 80)
        logger.info("SEC Financial Statements Ingestion Pipeline")
        logger.info("=" * 80)

        # Determine limit
        if all_stocks:
            limit = None
        elif ticker:
            limit = 1
        elif limit is None:
            limit = 10  # Default to 10 for safety

        # Get stocks to process
        stocks = await self.get_stocks_to_process(limit=limit, ticker=ticker, resume=resume)

        if not stocks:
            logger.warning("No stocks to process")
            return

        logger.info(f"Processing {len(stocks)} stocks...")

        # Process stocks
        await self.process_stocks(stocks)

        # Print summary
        duration = datetime.now() - start_time
        logger.info("=" * 80)
        logger.info("Ingestion Complete")
        logger.info("=" * 80)
        logger.info(f"Duration: {duration}")
        logger.info(f"Processed: {self.processed_count}")
        logger.info(f"Success: {self.success_count}")
        logger.info(f"Errors: {self.error_count}")
        logger.info(f"Success Rate: {(self.success_count/self.processed_count*100):.1f}%")

        # Close SEC client
        self.sec_client.close()


def main():
    """Main entry point for CLI."""
    parser = argparse.ArgumentParser(
        description='Ingest SEC financial statements into database',
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  # Test on 10 stocks (default)
  python sec_financials_ingestion.py

  # Test on 100 stocks
  python sec_financials_ingestion.py --limit 100

  # Process all stocks
  python sec_financials_ingestion.py --all

  # Process single stock
  python sec_financials_ingestion.py --ticker AAPL

  # Resume from last run
  python sec_financials_ingestion.py --resume --limit 500
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
    parser.add_argument(
        '--resume',
        action='store_true',
        help='Resume from stocks without recent data'
    )

    args = parser.parse_args()

    # Validate arguments
    if args.ticker and (args.all or args.limit):
        parser.error("--ticker cannot be used with --all or --limit")

    # Run pipeline
    pipeline = SECFinancialsIngestion()
    asyncio.run(pipeline.run(
        limit=args.limit,
        ticker=args.ticker,
        all_stocks=args.all,
        resume=args.resume
    ))


if __name__ == '__main__':
    main()
