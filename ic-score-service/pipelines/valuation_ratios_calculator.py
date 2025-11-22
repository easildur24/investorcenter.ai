#!/usr/bin/env python3
"""Valuation Ratios Calculator Pipeline.

This script calculates valuation ratios (P/E, P/B, P/S) by combining:
- Financial data from SEC filings (EPS, book value, shares outstanding)
- Latest stock prices from stock_prices table (EOD closing prices)

Ratios are timestamped with the calculation date and price used, ensuring
reproducibility and historical tracking.

Usage:
    python valuation_ratios_calculator.py --limit 100    # Test on 100 stocks
    python valuation_ratios_calculator.py --all          # All stocks
    python valuation_ratios_calculator.py --ticker AAPL  # Single stock
"""

import argparse
import asyncio
import logging
import sys
from datetime import datetime, date, timedelta
from decimal import Decimal
from pathlib import Path
from typing import List, Optional, Dict, Any

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
        logging.FileHandler('/app/logs/valuation_ratios_calculator.log')
    ]
)
logger = logging.getLogger(__name__)


class ValuationRatiosCalculator:
    """Calculator for stock valuation ratios using financial data + stock prices."""

    def __init__(self):
        """Initialize the calculator."""
        self.db = get_database()

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
        """Get list of stocks to process from database.

        Args:
            limit: Maximum number of stocks to process.
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
                  AND ticker IN (SELECT ticker FROM stock_prices WHERE time >= NOW() - INTERVAL '7 days')
                ORDER BY ticker
                LIMIT :limit
            """)
            result = await session.execute(query, {"limit": limit or 10000})
            tickers = [row[0] for row in result.fetchall()]

        logger.info(f"Found {len(tickers)} stocks with both financial and price data")
        return tickers

    async def get_latest_stock_price(self, ticker: str) -> Optional[Dict[str, Any]]:
        """Get latest closing price from stock_prices table.

        Args:
            ticker: Stock ticker symbol.

        Returns:
            Dictionary with price and date, or None if not found.
        """
        async with self.db.session() as session:
            query = text("""
                SELECT close, time
                FROM stock_prices
                WHERE ticker = :ticker
                  AND close IS NOT NULL
                ORDER BY time DESC
                LIMIT 1
            """)
            result = await session.execute(query, {"ticker": ticker})
            row = result.fetchone()

            if not row:
                return None

            return {
                'close': float(row[0]),
                'date': row[1].date() if hasattr(row[1], 'date') else row[1]
            }

    async def get_latest_financials(self, ticker: str) -> Optional[Dict[str, Any]]:
        """Get latest financial data for a stock.

        Args:
            ticker: Stock ticker symbol.

        Returns:
            Dictionary with financial data, or None if not found.
        """
        async with self.db.session() as session:
            query = text("""
                SELECT
                    id,
                    period_end_date,
                    fiscal_year,
                    fiscal_quarter,
                    eps_diluted,
                    shares_outstanding,
                    shareholders_equity,
                    revenue,
                    total_assets,
                    cash_and_equivalents,
                    short_term_debt,
                    long_term_debt,
                    total_liabilities
                FROM financials
                WHERE ticker = :ticker
                ORDER BY period_end_date DESC
                LIMIT 1
            """)
            result = await session.execute(query, {"ticker": ticker})
            row = result.fetchone()

            if not row:
                return None

            return {
                'id': row[0],
                'period_end_date': row[1],
                'fiscal_year': row[2],
                'fiscal_quarter': row[3],
                'eps_diluted': float(row[4]) if row[4] else None,
                'shares_outstanding': int(row[5]) if row[5] else None,
                'shareholders_equity': int(row[6]) if row[6] else None,
                'revenue': int(row[7]) if row[7] else None,
                'total_assets': int(row[8]) if row[8] else None,
                'cash_and_equivalents': int(row[9]) if row[9] else None,
                'short_term_debt': int(row[10]) if row[10] else None,
                'long_term_debt': int(row[11]) if row[11] else None,
                'total_liabilities': int(row[12]) if row[12] else None,
            }

    async def get_latest_ttm_financials(self, ticker: str) -> Optional[Dict[str, Any]]:
        """Get latest TTM financial data for EV/EBITDA calculation.

        Args:
            ticker: Stock ticker symbol.

        Returns:
            Dictionary with TTM financial data, or None if not found.
        """
        async with self.db.session() as session:
            query = text("""
                SELECT
                    ttm_ebitda,
                    cash_and_equivalents,
                    short_term_debt,
                    long_term_debt,
                    calculation_date
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
                'ttm_ebitda': int(row[0]) if row[0] else None,
                'cash_and_equivalents': int(row[1]) if row[1] else None,
                'short_term_debt': int(row[2]) if row[2] else None,
                'long_term_debt': int(row[3]) if row[3] else None,
                'calculation_date': row[4],
            }

    def calculate_ratios(
        self,
        ticker: str,
        price_data: Dict[str, Any],
        financial_data: Dict[str, Any],
        ttm_data: Optional[Dict[str, Any]] = None
    ) -> Optional[Dict[str, Any]]:
        """Calculate valuation ratios including EV/EBITDA.

        Args:
            ticker: Stock ticker symbol.
            price_data: Latest stock price data.
            financial_data: Latest financial statement data.
            ttm_data: TTM financial data (for EBITDA calculation).

        Returns:
            Dictionary of calculated ratios, or None if insufficient data.
        """
        try:
            stock_price = price_data['close']
            price_date = price_data['date']

            ratios = {
                'ticker': ticker,
                'financial_id': financial_data['id'],
                'valuation_calculation_date': price_date,
                'valuation_stock_price': round(stock_price, 2),
            }

            # Calculate P/E ratio
            eps = financial_data.get('eps_diluted')
            if eps and eps > 0:
                pe_ratio = stock_price / eps
                ratios['pe_ratio'] = round(pe_ratio, 2)
            else:
                ratios['pe_ratio'] = None

            # Calculate P/B ratio and current ratio (need shares outstanding)
            shares_outstanding = financial_data.get('shares_outstanding')
            shareholders_equity = financial_data.get('shareholders_equity')

            if shares_outstanding and shares_outstanding > 0:
                # Market cap
                market_cap = int(stock_price * shares_outstanding)
                ratios['market_cap'] = market_cap

                # P/B ratio
                if shareholders_equity and shareholders_equity > 0:
                    book_value_per_share = shareholders_equity / shares_outstanding
                    pb_ratio = stock_price / book_value_per_share
                    ratios['pb_ratio'] = round(pb_ratio, 2)
                else:
                    ratios['pb_ratio'] = None

                # P/S ratio
                revenue = financial_data.get('revenue')
                if revenue and revenue > 0:
                    ps_ratio = market_cap / revenue
                    ratios['ps_ratio'] = round(ps_ratio, 2)
                else:
                    ratios['ps_ratio'] = None
            else:
                ratios['market_cap'] = None
                ratios['pb_ratio'] = None
                ratios['ps_ratio'] = None

            # Calculate current ratio (not price-dependent, but useful to calculate here)
            total_assets = financial_data.get('total_assets')
            total_liabilities = financial_data.get('total_liabilities')
            cash = financial_data.get('cash_and_equivalents') or 0
            short_term_debt = financial_data.get('short_term_debt') or 0

            if total_assets and total_liabilities:
                current_assets = cash  # Simplified - should include other current assets
                current_liabilities = short_term_debt  # Simplified

                if current_liabilities and current_liabilities > 0:
                    # More accurate: use total assets as proxy if we don't have current assets breakdown
                    # This is a rough estimate
                    estimated_current_assets = total_assets * 0.4  # Assume 40% are current
                    estimated_current_liabilities = total_liabilities * 0.3  # Assume 30% are current

                    if estimated_current_liabilities > 0:
                        current_ratio = estimated_current_assets / estimated_current_liabilities
                        ratios['current_ratio'] = round(current_ratio, 2)
                    else:
                        ratios['current_ratio'] = None
                else:
                    ratios['current_ratio'] = None
            else:
                ratios['current_ratio'] = None

            # Calculate Enterprise Value and EV/EBITDA (if TTM data available)
            ratios['enterprise_value'] = None
            ratios['ttm_ev_ebitda_ratio'] = None

            if ttm_data and shares_outstanding and shares_outstanding > 0:
                market_cap = int(stock_price * shares_outstanding)
                cash = ttm_data.get('cash_and_equivalents') or 0
                short_term_debt = ttm_data.get('short_term_debt') or 0
                long_term_debt = ttm_data.get('long_term_debt') or 0
                total_debt = short_term_debt + long_term_debt

                # Enterprise Value = Market Cap + Total Debt - Cash
                enterprise_value = market_cap + total_debt - cash
                ratios['enterprise_value'] = enterprise_value

                # EV/EBITDA ratio
                ttm_ebitda = ttm_data.get('ttm_ebitda')
                if ttm_ebitda and ttm_ebitda > 0:
                    ev_ebitda_ratio = enterprise_value / ttm_ebitda
                    ratios['ttm_ev_ebitda_ratio'] = round(ev_ebitda_ratio, 2)

            # Check if we calculated at least one ratio
            if not any([ratios.get('pe_ratio'), ratios.get('pb_ratio'), ratios.get('ps_ratio')]):
                logger.warning(f"{ticker}: Could not calculate any valuation ratios")
                return None

            return ratios

        except Exception as e:
            logger.error(f"{ticker}: Error calculating ratios: {e}", exc_info=True)
            return None

    async def update_financials_with_ratios(self, ratios: Dict[str, Any]) -> bool:
        """Update financials table with calculated ratios.

        Args:
            ratios: Dictionary of calculated ratios.

        Returns:
            True if successful, False otherwise.
        """
        try:
            async with self.db.session() as session:
                # Build UPDATE statement
                update_fields = []
                params = {'financial_id': ratios['financial_id']}

                if ratios.get('pe_ratio') is not None:
                    update_fields.append("pe_ratio = :pe_ratio")
                    params['pe_ratio'] = ratios['pe_ratio']

                if ratios.get('pb_ratio') is not None:
                    update_fields.append("pb_ratio = :pb_ratio")
                    params['pb_ratio'] = ratios['pb_ratio']

                if ratios.get('ps_ratio') is not None:
                    update_fields.append("ps_ratio = :ps_ratio")
                    params['ps_ratio'] = ratios['ps_ratio']

                if ratios.get('current_ratio') is not None:
                    update_fields.append("current_ratio = :current_ratio")
                    params['current_ratio'] = ratios['current_ratio']

                if ratios.get('market_cap') is not None:
                    update_fields.append("market_cap = :market_cap")
                    params['market_cap'] = ratios['market_cap']

                if ratios.get('enterprise_value') is not None:
                    update_fields.append("enterprise_value = :enterprise_value")
                    params['enterprise_value'] = ratios['enterprise_value']

                if ratios.get('ttm_ev_ebitda_ratio') is not None:
                    update_fields.append("ttm_ev_ebitda_ratio = :ttm_ev_ebitda_ratio")
                    params['ttm_ev_ebitda_ratio'] = ratios['ttm_ev_ebitda_ratio']

                # Always update metadata
                update_fields.append("valuation_calculation_date = :calc_date")
                update_fields.append("valuation_stock_price = :stock_price")
                params['calc_date'] = ratios['valuation_calculation_date']
                params['stock_price'] = ratios['valuation_stock_price']

                query = text(f"""
                    UPDATE financials
                    SET {', '.join(update_fields)}
                    WHERE id = :financial_id
                """)

                await session.execute(query, params)
                await session.commit()

                return True

        except Exception as e:
            logger.error(f"Error updating financials: {e}", exc_info=True)
            return False

    async def process_ticker(self, ticker: str) -> bool:
        """Process a single ticker: calculate and store valuation ratios.

        Args:
            ticker: Stock ticker symbol.

        Returns:
            True if successful, False otherwise.
        """
        try:
            # Get latest stock price
            price_data = await self.get_latest_stock_price(ticker)
            if not price_data:
                logger.warning(f"{ticker}: No stock price data found")
                return False

            # Get latest financial data
            financial_data = await self.get_latest_financials(ticker)
            if not financial_data:
                logger.warning(f"{ticker}: No financial data found")
                return False

            # Get latest TTM financial data (for EV/EBITDA)
            ttm_data = await self.get_latest_ttm_financials(ticker)

            # Calculate ratios
            ratios = self.calculate_ratios(ticker, price_data, financial_data, ttm_data)
            if not ratios:
                return False

            # Update database
            success = await self.update_financials_with_ratios(ratios)

            if success:
                ev_ebitda_str = f", EV/EBITDA: {ratios.get('ttm_ev_ebitda_ratio')}" if ratios.get('ttm_ev_ebitda_ratio') else ""
                logger.info(
                    f"{ticker}: Updated ratios - "
                    f"P/E: {ratios.get('pe_ratio')}, "
                    f"P/B: {ratios.get('pb_ratio')}, "
                    f"P/S: {ratios.get('ps_ratio')}"
                    f"{ev_ebitda_str}, "
                    f"Price: ${ratios['valuation_stock_price']}"
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
        logger.info("Valuation Ratios Calculator Pipeline")
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
        logger.info("Processing Complete")
        logger.info("=" * 80)
        logger.info(f"Duration: {duration}")
        logger.info(f"Processed: {self.processed_count}")
        logger.info(f"Success: {self.success_count}")
        logger.info(f"Errors: {self.error_count}")
        if self.processed_count > 0:
            logger.info(f"Success Rate: {(self.success_count/self.processed_count*100):.1f}%")


def main():
    """Main entry point for CLI."""
    parser = argparse.ArgumentParser(
        description='Calculate valuation ratios for stocks',
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
