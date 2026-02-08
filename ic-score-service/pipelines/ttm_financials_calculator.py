#!/usr/bin/env python3
"""Calculate Trailing Twelve Months (TTM) financial metrics with correct EPS handling.

This script calculates TTM financials using the correct methodology for EPS:
1. If a recent annual 10-K exists (filed within last 3 months), use annual EPS directly
2. Otherwise, calculate TTM EPS from cumulative quarterly data:
   - Quarterly 10-Q EPS values are CUMULATIVE (year-to-date), not individual quarters
   - TTM EPS = Most recent Q EPS + (Previous Annual EPS - Previous same Q EPS)

For other metrics (revenue, cash flow, etc.), we sum the last 4 quarters.
Balance sheet items are taken from the most recent quarter.

Usage:
    python ttm_financials_calculator_v2.py --all          # Calculate for all tickers
    python ttm_financials_calculator_v2.py --limit 100    # Test on 100 tickers
    python ttm_financials_calculator_v2.py --ticker AAPL  # Single ticker
"""

import argparse
import asyncio
import logging
import sys
from datetime import date, datetime, timedelta
from pathlib import Path
from typing import Dict, List, Optional, Tuple

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
        logging.FileHandler('/app/logs/ttm_financials_calculator.log')
    ]
)
logger = logging.getLogger(__name__)


class TTMFinancialsCalculator:
    """Calculate Trailing Twelve Months (TTM) financial metrics with correct EPS methodology."""

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
                return [ticker]

            # Get tickers with financial data (both quarterly and annual)
            query = text("""
                SELECT DISTINCT ticker
                FROM financials
                WHERE ticker NOT LIKE '%-%'
                  AND ticker NOT LIKE '%.%'
                ORDER BY ticker
                LIMIT :limit
            """)

            result = await session.execute(query, {"limit": limit or 100000})
            rows = result.fetchall()

            tickers = [row[0] for row in rows]
            logger.info(f"Found {len(tickers)} tickers with financial data")

            return tickers

    async def get_annual_10k(self, ticker: str) -> Optional[Dict]:
        """Get the most recent annual 10-K filing for a ticker.

        Args:
            ticker: Stock ticker symbol.

        Returns:
            Dict with annual financial data, or None if not found.
        """
        async with self.db.session() as session:
            # Filter out future dates to handle data quality issues
            query = text("""
                SELECT
                    id, ticker, period_end_date, fiscal_year, filing_date, statement_type,
                    revenue, net_income, eps_basic, eps_diluted, shares_outstanding,
                    total_assets, total_liabilities, shareholders_equity,
                    cash_and_equivalents, short_term_debt, long_term_debt,
                    operating_cash_flow, free_cash_flow, capex
                FROM financials
                WHERE ticker = :ticker
                  AND statement_type = '10-K'
                  AND fiscal_quarter IS NULL
                  AND period_end_date <= CURRENT_DATE
                ORDER BY period_end_date DESC
                LIMIT 1
            """)

            result = await session.execute(query, {"ticker": ticker})
            row = result.fetchone()

            if not row:
                return None

            return {
                'id': row[0],
                'ticker': row[1],
                'period_end_date': row[2],
                'fiscal_year': row[3],
                'filing_date': row[4],
                'statement_type': row[5],
                'revenue': int(row[6]) if row[6] else None,
                'net_income': int(row[7]) if row[7] else None,
                'eps_basic': float(row[8]) if row[8] else None,
                'eps_diluted': float(row[9]) if row[9] else None,
                'shares_outstanding': int(row[10]) if row[10] else None,
                'total_assets': int(row[11]) if row[11] else None,
                'total_liabilities': int(row[12]) if row[12] else None,
                'shareholders_equity': int(row[13]) if row[13] else None,
                'cash_and_equivalents': int(row[14]) if row[14] else None,
                'short_term_debt': int(row[15]) if row[15] else None,
                'long_term_debt': int(row[16]) if row[16] else None,
                'operating_cash_flow': int(row[17]) if row[17] else None,
                'free_cash_flow': int(row[18]) if row[18] else None,
                'capex': int(row[19]) if row[19] else None,
            }

    async def get_quarterly_data(self, ticker: str, limit: int = 5) -> List[Dict]:
        """Get recent quarterly 10-Q filings for a ticker.

        Args:
            ticker: Stock ticker symbol.
            limit: Number of quarters to retrieve.

        Returns:
            List of quarterly data dicts, ordered from newest to oldest.
        """
        async with self.db.session() as session:
            # Filter out future dates to handle data quality issues
            query = text("""
                SELECT
                    id, ticker, period_end_date, fiscal_year, fiscal_quarter, filing_date,
                    revenue, cost_of_revenue, gross_profit, operating_expenses,
                    operating_income, net_income, eps_basic, eps_diluted,
                    shares_outstanding, total_assets, total_liabilities,
                    shareholders_equity, cash_and_equivalents,
                    short_term_debt, long_term_debt,
                    operating_cash_flow, investing_cash_flow,
                    financing_cash_flow, free_cash_flow, capex
                FROM financials
                WHERE ticker = :ticker
                  AND statement_type = '10-Q'
                  AND fiscal_quarter IS NOT NULL
                  AND period_end_date <= CURRENT_DATE
                ORDER BY period_end_date DESC
                LIMIT :limit
            """)

            result = await session.execute(query, {"ticker": ticker, "limit": limit})
            rows = result.fetchall()

            quarters = []
            for row in rows:
                quarters.append({
                    'id': row[0],
                    'ticker': row[1],
                    'period_end_date': row[2],
                    'fiscal_year': row[3],
                    'fiscal_quarter': row[4],
                    'filing_date': row[5],
                    'revenue': int(row[6]) if row[6] else None,
                    'cost_of_revenue': int(row[7]) if row[7] else None,
                    'gross_profit': int(row[8]) if row[8] else None,
                    'operating_expenses': int(row[9]) if row[9] else None,
                    'operating_income': int(row[10]) if row[10] else None,
                    'net_income': int(row[11]) if row[11] else None,
                    'eps_basic': float(row[12]) if row[12] else None,
                    'eps_diluted': float(row[13]) if row[13] else None,
                    'shares_outstanding': int(row[14]) if row[14] else None,
                    'total_assets': int(row[15]) if row[15] else None,
                    'total_liabilities': int(row[16]) if row[16] else None,
                    'shareholders_equity': int(row[17]) if row[17] else None,
                    'cash_and_equivalents': int(row[18]) if row[18] else None,
                    'short_term_debt': int(row[19]) if row[19] else None,
                    'long_term_debt': int(row[20]) if row[20] else None,
                    'operating_cash_flow': int(row[21]) if row[21] else None,
                    'investing_cash_flow': int(row[22]) if row[22] else None,
                    'financing_cash_flow': int(row[23]) if row[23] else None,
                    'free_cash_flow': int(row[24]) if row[24] else None,
                    'capex': int(row[25]) if row[25] else None,
                })

            return quarters

    def calculate_ttm_eps(
        self,
        annual_10k: Optional[Dict],
        quarters: List[Dict]
    ) -> Tuple[Optional[float], Optional[float]]:
        """Calculate TTM EPS using the correct methodology.

        Methodology:
        1. If annual 10-K exists and was filed within last 3 months: Use annual EPS
        2. Otherwise: Calculate from cumulative quarterly data
           - Quarterly EPS in 10-Q are cumulative (year-to-date)
           - TTM EPS = Current Q (YTD) + Prior year Q4 EPS
           - Prior Q4 EPS = Prior Annual - Prior Q3 (YTD)

        Args:
            annual_10k: Most recent annual 10-K filing data.
            quarters: List of recent quarterly 10-Q filings (newest first).

        Returns:
            Tuple of (eps_basic, eps_diluted), or (None, None) if insufficient data.
        """
        # Strategy 1: Use recent annual 10-K if available
        if annual_10k:
            filing_date = annual_10k.get('filing_date')
            if filing_date:
                # Check if filed within last 3 months
                days_since_filing = (date.today() - filing_date).days
                if days_since_filing <= 90:  # 3 months
                    logger.debug(
                        f"{annual_10k['ticker']}: Using annual 10-K EPS "
                        f"(filed {days_since_filing} days ago)"
                    )
                    return (
                        annual_10k.get('eps_basic'),
                        annual_10k.get('eps_diluted')
                    )

        # Strategy 2: Calculate from cumulative quarterly data
        if not quarters or len(quarters) < 1:
            return (None, None)

        most_recent_q = quarters[0]

        # We need the most recent quarterly data (cumulative YTD)
        current_q_eps_basic = most_recent_q.get('eps_basic')
        current_q_eps_diluted = most_recent_q.get('eps_diluted')
        current_fiscal_quarter = most_recent_q.get('fiscal_quarter')

        # If we have Q4 data directly (which is rare), just use it
        if current_fiscal_quarter == 4:
            # Q4 cumulative = Full year
            return (current_q_eps_basic, current_q_eps_diluted)

        # For Q1, Q2, Q3: We need previous year's annual and corresponding quarter
        # TTM = Current Q (YTD) + Prior year Q4
        # Prior Q4 = Prior Annual - Prior same Q (YTD)

        # Find previous year's annual 10-K
        prev_annual_eps_basic = None
        prev_annual_eps_diluted = None

        if annual_10k:
            # Check if this is from prior fiscal year
            if annual_10k['fiscal_year'] == most_recent_q['fiscal_year'] - 1:
                prev_annual_eps_basic = annual_10k.get('eps_basic')
                prev_annual_eps_diluted = annual_10k.get('eps_diluted')

        # Find corresponding quarter from previous year (same fiscal quarter)
        prev_q_eps_basic = None
        prev_q_eps_diluted = None

        for q in quarters[1:]:  # Skip first (most recent)
            if q['fiscal_year'] == most_recent_q['fiscal_year'] - 1:
                if q['fiscal_quarter'] == current_fiscal_quarter:
                    prev_q_eps_basic = q.get('eps_basic')
                    prev_q_eps_diluted = q.get('eps_diluted')
                    break

        # Calculate Q4 from previous year, then add to current Q
        ttm_eps_basic = None
        ttm_eps_diluted = None

        if current_q_eps_basic is not None and prev_annual_eps_basic is not None and prev_q_eps_basic is not None:
            # Prior Q4 = Prior Annual - Prior same Q (cumulative)
            prior_q4_basic = prev_annual_eps_basic - prev_q_eps_basic
            # TTM = Current Q (cumulative) + Prior Q4
            ttm_eps_basic = round(current_q_eps_basic + prior_q4_basic, 4)

        if current_q_eps_diluted is not None and prev_annual_eps_diluted is not None and prev_q_eps_diluted is not None:
            prior_q4_diluted = prev_annual_eps_diluted - prev_q_eps_diluted
            ttm_eps_diluted = round(current_q_eps_diluted + prior_q4_diluted, 4)

        logger.debug(
            f"{most_recent_q['ticker']}: Calculated TTM EPS from Q{current_fiscal_quarter} + prior Q4"
        )

        return (ttm_eps_basic, ttm_eps_diluted)

    async def calculate_ttm_metrics(self, ticker: str) -> Optional[Dict]:
        """Calculate all TTM metrics for a ticker.

        Args:
            ticker: Stock ticker symbol.

        Returns:
            Dict with calculated TTM metrics, or None if insufficient data.
        """
        # Get annual 10-K
        annual_10k = await self.get_annual_10k(ticker)

        # Get quarterly 10-Q data (get 10 to ensure we have prior year data)
        # For companies with non-calendar fiscal years, we need more quarters
        # to find the matching quarter from the previous fiscal year
        quarters = await self.get_quarterly_data(ticker, limit=10)

        if not quarters:
            logger.debug(f"{ticker}: No quarterly data found")
            return None

        most_recent_q = quarters[0]

        # Calculate TTM EPS using correct methodology
        eps_basic, eps_diluted = self.calculate_ttm_eps(annual_10k, quarters)

        # FALLBACK: If complex EPS calculation failed but we have net_income and shares,
        # calculate EPS simply as net_income / shares_outstanding
        # This handles cases with fiscal year mismatches or missing prior year data
        if eps_diluted is None or eps_basic is None:
            # We'll calculate this after we get net_income from quarters
            needs_eps_fallback = True
        else:
            needs_eps_fallback = False

        # For revenue and other metrics: Sum last 4 quarters (if we have them)
        # OR use annual 10-K if recent
        revenue = None
        net_income = None
        operating_cash_flow = None
        free_cash_flow = None
        capex = None

        # Use annual 10-K if recent (within 3 months)
        use_annual = False
        if annual_10k and annual_10k.get('filing_date'):
            days_since_filing = (date.today() - annual_10k['filing_date']).days
            if days_since_filing <= 90:
                use_annual = True

        if use_annual and annual_10k:
            revenue = annual_10k.get('revenue')
            net_income = annual_10k.get('net_income')
            operating_cash_flow = annual_10k.get('operating_cash_flow')
            free_cash_flow = annual_10k.get('free_cash_flow')
            capex = annual_10k.get('capex')
        elif len(quarters) >= 4:
            # Sum last 4 quarters
            last_4_quarters = quarters[:4]

            revenue_values = [q['revenue'] for q in last_4_quarters if q['revenue'] is not None]
            if revenue_values:
                revenue = sum(revenue_values)

            net_income_values = [q['net_income'] for q in last_4_quarters if q['net_income'] is not None]
            if net_income_values:
                net_income = sum(net_income_values)

            ocf_values = [q['operating_cash_flow'] for q in last_4_quarters if q['operating_cash_flow'] is not None]
            if ocf_values:
                operating_cash_flow = sum(ocf_values)

            fcf_values = [q['free_cash_flow'] for q in last_4_quarters if q['free_cash_flow'] is not None]
            if fcf_values:
                free_cash_flow = sum(fcf_values)

            capex_values = [q['capex'] for q in last_4_quarters if q['capex'] is not None]
            if capex_values:
                capex = sum(capex_values)

        # Balance sheet items from most recent quarter
        shares_outstanding = most_recent_q.get('shares_outstanding')
        total_assets = most_recent_q.get('total_assets')
        total_liabilities = most_recent_q.get('total_liabilities')
        shareholders_equity = most_recent_q.get('shareholders_equity')
        cash_and_equivalents = most_recent_q.get('cash_and_equivalents')
        short_term_debt = most_recent_q.get('short_term_debt')
        long_term_debt = most_recent_q.get('long_term_debt')

        # Calculate 5-quarter average shareholders' equity (YCharts ROE methodology)
        # YCharts uses average of past 5 quarters to capture equity changes during the year
        avg_shareholders_equity_5q = None
        equity_values = [
            q['shareholders_equity'] for q in quarters[:5]
            if q.get('shareholders_equity') is not None
        ]
        if equity_values:
            avg_shareholders_equity_5q = int(sum(equity_values) / len(equity_values))
            logger.debug(
                f"{ticker}: 5Q avg equity calculated from {len(equity_values)} quarters: "
                f"${avg_shareholders_equity_5q:,} (latest: ${shareholders_equity:,})"
                if shareholders_equity else f"${avg_shareholders_equity_5q:,}"
            )

        # EPS SANITY CHECK: Detect stock split issues where EPS sign doesn't match net_income sign
        # This happens when historical EPS values weren't adjusted for stock splits
        if not needs_eps_fallback and net_income is not None and eps_diluted is not None:
            # If net_income and EPS have opposite signs, the calculation is wrong (likely stock split)
            if (net_income > 0 and eps_diluted < 0) or (net_income < 0 and eps_diluted > 0):
                logger.warning(
                    f"{ticker}: EPS sign mismatch detected (net_income={net_income:,}, eps={eps_diluted:.4f}) "
                    f"- likely stock split issue, will use fallback calculation"
                )
                needs_eps_fallback = True
                eps_basic = None
                eps_diluted = None

        # EPS FALLBACK: Calculate from net_income / shares_outstanding if needed
        if needs_eps_fallback and net_income is not None and shares_outstanding is not None and shares_outstanding > 0:
            calculated_eps = round(net_income / shares_outstanding, 4)
            if eps_basic is None:
                eps_basic = calculated_eps
            if eps_diluted is None:
                eps_diluted = calculated_eps
            logger.debug(
                f"{ticker}: Used fallback EPS calculation (net_income/shares): ${calculated_eps:.4f}"
            )

        # Determine TTM period
        if use_annual and annual_10k:
            ttm_period_end = annual_10k['period_end_date']
            ttm_period_start = date(annual_10k['fiscal_year'] - 1, ttm_period_end.month, ttm_period_end.day)
        elif len(quarters) >= 4:
            ttm_period_end = quarters[0]['period_end_date']
            ttm_period_start = quarters[3]['period_end_date']
        else:
            ttm_period_end = most_recent_q['period_end_date']
            ttm_period_start = most_recent_q['period_end_date']

        ttm_data = {
            'ticker': ticker,
            'calculation_date': date.today(),
            'ttm_period_start': ttm_period_start,
            'ttm_period_end': ttm_period_end,
            # Income Statement
            'revenue': revenue,
            'net_income': net_income,
            'eps_basic': eps_basic,
            'eps_diluted': eps_diluted,
            # Balance Sheet (from most recent quarter)
            'shares_outstanding': shares_outstanding,
            'total_assets': total_assets,
            'total_liabilities': total_liabilities,
            'shareholders_equity': shareholders_equity,
            'avg_shareholders_equity_5q': avg_shareholders_equity_5q,
            'cash_and_equivalents': cash_and_equivalents,
            'short_term_debt': short_term_debt,
            'long_term_debt': long_term_debt,
            # Cash Flow
            'operating_cash_flow': operating_cash_flow,
            'free_cash_flow': free_cash_flow,
            'capex': capex,
        }

        return ttm_data

    async def store_ttm_financials(self, ttm_data: Dict) -> bool:
        """Store TTM financial metrics in database.

        Args:
            ttm_data: Dict with calculated TTM metrics.

        Returns:
            True if successful, False otherwise.
        """
        try:
            async with self.db.session() as session:
                query = text("""
                    INSERT INTO ttm_financials (
                        ticker, calculation_date, ttm_period_start, ttm_period_end,
                        revenue, net_income, eps_basic, eps_diluted,
                        shares_outstanding, total_assets, total_liabilities,
                        shareholders_equity, avg_shareholders_equity_5q, cash_and_equivalents,
                        short_term_debt, long_term_debt,
                        operating_cash_flow, free_cash_flow, capex
                    ) VALUES (
                        :ticker, :calculation_date, :ttm_period_start, :ttm_period_end,
                        :revenue, :net_income, :eps_basic, :eps_diluted,
                        :shares_outstanding, :total_assets, :total_liabilities,
                        :shareholders_equity, :avg_shareholders_equity_5q, :cash_and_equivalents,
                        :short_term_debt, :long_term_debt,
                        :operating_cash_flow, :free_cash_flow, :capex
                    )
                    ON CONFLICT (ticker, calculation_date)
                    DO UPDATE SET
                        ttm_period_start = EXCLUDED.ttm_period_start,
                        ttm_period_end = EXCLUDED.ttm_period_end,
                        revenue = EXCLUDED.revenue,
                        net_income = EXCLUDED.net_income,
                        eps_basic = EXCLUDED.eps_basic,
                        eps_diluted = EXCLUDED.eps_diluted,
                        shares_outstanding = EXCLUDED.shares_outstanding,
                        total_assets = EXCLUDED.total_assets,
                        total_liabilities = EXCLUDED.total_liabilities,
                        shareholders_equity = EXCLUDED.shareholders_equity,
                        avg_shareholders_equity_5q = EXCLUDED.avg_shareholders_equity_5q,
                        cash_and_equivalents = EXCLUDED.cash_and_equivalents,
                        short_term_debt = EXCLUDED.short_term_debt,
                        long_term_debt = EXCLUDED.long_term_debt,
                        operating_cash_flow = EXCLUDED.operating_cash_flow,
                        free_cash_flow = EXCLUDED.free_cash_flow,
                        capex = EXCLUDED.capex
                """)

                await session.execute(query, ttm_data)
                await session.commit()

                return True

        except Exception as e:
            logger.error(f"Error storing TTM financials for {ttm_data['ticker']}: {e}", exc_info=True)
            return False

    async def process_ticker(self, ticker: str) -> bool:
        """Process a single ticker to calculate TTM metrics.

        Args:
            ticker: Stock ticker symbol.

        Returns:
            True if successful, False otherwise.
        """
        try:
            # Calculate TTM metrics
            ttm_data = await self.calculate_ttm_metrics(ticker)

            if not ttm_data:
                logger.debug(f"{ticker}: Insufficient data to calculate TTM metrics")
                return False

            # Store in database
            success = await self.store_ttm_financials(ttm_data)

            if success:
                eps = ttm_data.get('eps_diluted')
                revenue = ttm_data.get('revenue')
                eps_str = f"${eps}" if eps is not None else "N/A"
                revenue_str = f"${revenue:,}" if revenue is not None else "N/A"
                logger.debug(
                    f"{ticker}: TTM calculated - EPS: {eps_str}, Revenue: {revenue_str} "
                    f"({ttm_data['ttm_period_start']} to {ttm_data['ttm_period_end']})"
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
        progress_bar = tqdm(total=len(tickers), desc="Calculating TTM Financials")

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
        """Run the TTM financials calculation process.

        Args:
            limit: Limit number of tickers to process.
            ticker: Process single ticker.
            all_tickers: Process all tickers with sufficient data.
        """
        start_time = datetime.now()
        logger.info("=" * 80)
        logger.info("TTM Financials Calculator v2 (Correct EPS Methodology)")
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
        logger.info("TTM Calculation Complete")
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
        description='Calculate Trailing Twelve Months (TTM) financial metrics with correct EPS',
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  # Test on 10 tickers
  python ttm_financials_calculator_v2.py --limit 10

  # Process all tickers
  python ttm_financials_calculator_v2.py --all

  # Process single ticker
  python ttm_financials_calculator_v2.py --ticker AAPL
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
        help='Process all tickers with financial data'
    )

    args = parser.parse_args()

    # Validate arguments
    if args.ticker and (args.all or args.limit):
        parser.error("--ticker cannot be used with --all or --limit")

    # Run calculator
    calculator = TTMFinancialsCalculator()
    asyncio.run(calculator.run(
        limit=args.limit,
        ticker=args.ticker,
        all_tickers=args.all
    ))


if __name__ == '__main__':
    main()
