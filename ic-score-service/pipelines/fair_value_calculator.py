#!/usr/bin/env python3
"""Fair Value Calculator Pipeline.

Calculates DCF-based intrinsic value estimates including:
- WACC (Weighted Average Cost of Capital)
- Two-stage DCF model
- Graham Number
- Earnings Power Value (EPV)

Usage:
    python fair_value_calculator.py --all          # Calculate for all tickers
    python fair_value_calculator.py --limit 100    # Test on 100 tickers
    python fair_value_calculator.py --ticker AAPL  # Single ticker
"""

import argparse
import asyncio
import logging
import math
import sys
from datetime import date, datetime
from decimal import Decimal, InvalidOperation
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
    ]
)
logger = logging.getLogger(__name__)


# =============================================================================
# WACC CALCULATIONS
# =============================================================================

def calculate_cost_of_equity(
    beta: Decimal,
    risk_free_rate: Decimal,
    market_risk_premium: Decimal = Decimal("0.055")
) -> Decimal:
    """Cost of Equity using CAPM.

    Re = Rf + beta * (Rm - Rf)

    Args:
        beta: Stock beta (volatility vs market)
        risk_free_rate: Risk-free rate (10Y Treasury)
        market_risk_premium: Market risk premium (~5.5% historical)

    Returns:
        Cost of equity as decimal (e.g., 0.10 for 10%)
    """
    return risk_free_rate + (beta * market_risk_premium)


def estimate_cost_of_debt(
    interest_expense: Optional[Decimal],
    total_debt: Optional[Decimal]
) -> Decimal:
    """Estimate cost of debt from financial statements.

    Cost of Debt = Interest Expense / Total Debt

    Args:
        interest_expense: Annual interest expense
        total_debt: Total debt (short + long term)

    Returns:
        Cost of debt as decimal, defaults to 5% if unavailable
    """
    if interest_expense and total_debt and total_debt > 0:
        implied_rate = interest_expense / total_debt
        # Sanity bounds: 2% to 15%
        return max(Decimal("0.02"), min(Decimal("0.15"), implied_rate))
    return Decimal("0.05")  # Default 5%


def calculate_wacc(
    cost_of_equity: Decimal,
    cost_of_debt: Decimal,
    total_debt: Decimal,
    market_cap: Decimal,
    tax_rate: Decimal = Decimal("0.21")
) -> Decimal:
    """Weighted Average Cost of Capital.

    WACC = (E/V * Re) + (D/V * Rd * (1-T))

    Args:
        cost_of_equity: Cost of equity (Re)
        cost_of_debt: Cost of debt (Rd)
        total_debt: Market value of debt (D)
        market_cap: Market value of equity (E)
        tax_rate: Corporate tax rate (T)

    Returns:
        WACC as decimal, bounded between 5% and 20%
    """
    total_value = market_cap + total_debt
    if total_value == 0:
        return Decimal("0.10")  # Default 10%

    equity_weight = market_cap / total_value
    debt_weight = total_debt / total_value

    after_tax_cost_of_debt = cost_of_debt * (1 - tax_rate)

    wacc = (equity_weight * cost_of_equity) + (debt_weight * after_tax_cost_of_debt)

    # Sanity bounds: 5% to 20%
    return max(Decimal("0.05"), min(Decimal("0.20"), wacc))


# =============================================================================
# DCF MODEL
# =============================================================================

def calculate_dcf_fair_value(
    fcf_ttm: Decimal,
    growth_rate_high: Decimal,
    growth_rate_terminal: Decimal,
    wacc: Decimal,
    shares_outstanding: int,
    net_debt: Decimal,
    projection_years: int = 10
) -> Tuple[Optional[Decimal], Dict]:
    """Two-Stage DCF Model.

    Stage 1 (Years 1-5): High growth period at growth_rate_high
    Stage 2 (Years 6-10): Fade to terminal growth rate
    Terminal Value: Gordon Growth Model

    Args:
        fcf_ttm: Trailing twelve months free cash flow
        growth_rate_high: First 5 years growth rate
        growth_rate_terminal: Perpetuity growth (typically 2-3%)
        wacc: Weighted average cost of capital
        shares_outstanding: Number of shares outstanding
        net_debt: Total Debt - Cash
        projection_years: Number of years to project

    Returns:
        Tuple of (fair_value_per_share, calculation_details)
    """
    if fcf_ttm <= 0:
        return None, {"error": "Negative FCF - DCF not applicable"}

    if wacc <= growth_rate_terminal:
        return None, {"error": "WACC must be greater than terminal growth"}

    if shares_outstanding <= 0:
        return None, {"error": "Invalid shares outstanding"}

    projected_fcf = []
    current_fcf = float(fcf_ttm)

    # Stage 1: High growth (Years 1-5)
    high_growth = float(growth_rate_high)
    for year in range(1, 6):
        current_fcf = current_fcf * (1 + high_growth)
        pv_factor = 1 / ((1 + float(wacc)) ** year)
        pv_fcf = current_fcf * pv_factor
        projected_fcf.append({
            "year": year,
            "fcf": current_fcf,
            "pv_fcf": pv_fcf,
            "growth_rate": high_growth * 100
        })

    # Stage 2: Transition to terminal growth (Years 6-10)
    terminal_growth = float(growth_rate_terminal)
    for year in range(6, projection_years + 1):
        # Linear fade from high growth to terminal growth
        fade = (projection_years - year) / (projection_years - 5)
        blended_growth = terminal_growth + (high_growth - terminal_growth) * fade

        current_fcf = current_fcf * (1 + blended_growth)
        pv_factor = 1 / ((1 + float(wacc)) ** year)
        pv_fcf = current_fcf * pv_factor
        projected_fcf.append({
            "year": year,
            "fcf": current_fcf,
            "pv_fcf": pv_fcf,
            "growth_rate": blended_growth * 100
        })

    # Sum of discounted FCFs
    sum_pv_fcf = sum(p["pv_fcf"] for p in projected_fcf)

    # Terminal Value (Gordon Growth Model)
    # TV = FCF_n+1 / (WACC - g)
    terminal_fcf = current_fcf * (1 + terminal_growth)
    terminal_value = terminal_fcf / (float(wacc) - terminal_growth)
    pv_terminal = terminal_value / ((1 + float(wacc)) ** projection_years)

    # Enterprise Value
    enterprise_value = sum_pv_fcf + pv_terminal

    # Equity Value = Enterprise Value - Net Debt
    equity_value = enterprise_value - float(net_debt)

    if equity_value <= 0:
        return None, {"error": "Negative equity value - company may be distressed"}

    # Per Share Value
    fair_value_per_share = Decimal(str(round(equity_value / shares_outstanding, 2)))

    details = {
        "inputs": {
            "fcf_ttm": float(fcf_ttm),
            "growth_rate_high": high_growth * 100,
            "growth_rate_terminal": terminal_growth * 100,
            "wacc": float(wacc) * 100,
            "shares_outstanding": shares_outstanding,
            "net_debt": float(net_debt)
        },
        "projected_fcf": projected_fcf,
        "terminal_value": terminal_value,
        "pv_terminal_value": pv_terminal,
        "sum_pv_fcf": sum_pv_fcf,
        "enterprise_value": enterprise_value,
        "equity_value": equity_value,
        "fair_value_per_share": float(fair_value_per_share)
    }

    return fair_value_per_share, details


# =============================================================================
# ALTERNATIVE VALUATION METHODS
# =============================================================================

def calculate_graham_number(
    eps: Optional[Decimal],
    book_value_per_share: Optional[Decimal]
) -> Optional[Decimal]:
    """Graham Number for defensive investors.

    Graham Number = sqrt(22.5 * EPS * Book Value per Share)

    Benjamin Graham's formula for maximum price a defensive investor
    should pay for a stock. Combines earnings and book value.

    The 22.5 multiplier comes from:
    - Maximum P/E of 15
    - Maximum P/B of 1.5
    - 15 * 1.5 = 22.5

    Args:
        eps: Earnings per share (positive)
        book_value_per_share: Book value per share (positive)

    Returns:
        Graham number fair value, or None if inputs invalid
    """
    if not eps or not book_value_per_share:
        return None
    if eps <= 0 or book_value_per_share <= 0:
        return None

    try:
        value = float(Decimal("22.5") * eps * book_value_per_share)
        return Decimal(str(round(math.sqrt(value), 2)))
    except (ValueError, InvalidOperation):
        return None


def calculate_earnings_power_value(
    normalized_ebit: Decimal,
    wacc: Decimal,
    net_debt: Decimal,
    shares_outstanding: int,
    tax_rate: Decimal = Decimal("0.21")
) -> Optional[Decimal]:
    """Earnings Power Value (EPV) - Bruce Greenwald's approach.

    EPV = (Normalized EBIT * (1 - Tax Rate)) / WACC - Net Debt

    Assumes no growth - values the company's current sustainable earning power
    as a perpetuity. More conservative than DCF because it ignores growth.

    Key principles:
    1. Use EBIT (not Net Income) to exclude financing decisions
    2. Apply tax rate to get after-tax operating earnings (NOPAT)
    3. Capitalize at WACC to get Enterprise Value
    4. Subtract Net Debt to get Equity Value

    Args:
        normalized_ebit: Normalized operating income
        wacc: Weighted average cost of capital
        net_debt: Total Debt - Cash
        shares_outstanding: Number of shares outstanding
        tax_rate: Corporate tax rate (default 21%)

    Returns:
        EPV fair value per share, or None if inputs invalid
    """
    if not normalized_ebit or normalized_ebit <= 0:
        return None
    if not wacc or wacc <= 0:
        return None
    if shares_outstanding <= 0:
        return None

    # After-tax operating earnings (NOPAT)
    after_tax_earnings = float(normalized_ebit) * (1 - float(tax_rate))

    # Enterprise Value = perpetuity of after-tax earnings
    enterprise_value = after_tax_earnings / float(wacc)

    # Equity Value = Enterprise Value - Net Debt
    equity_value = enterprise_value - float(net_debt)

    # Handle negative equity value (overleveraged companies)
    if equity_value <= 0:
        return None

    return Decimal(str(round(equity_value / shares_outstanding, 2)))


# =============================================================================
# MAIN CALCULATOR CLASS
# =============================================================================

class FairValueCalculator:
    """Calculate DCF and alternative fair value estimates for all tickers."""

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

            # Get tickers with fundamental metrics calculated
            query = text("""
                SELECT DISTINCT ticker
                FROM fundamental_metrics_extended
                WHERE calculation_date >= CURRENT_DATE - INTERVAL '7 days'
                  AND ticker NOT LIKE '%-%'
                  AND ticker NOT LIKE '%.%'
                ORDER BY ticker
                LIMIT :limit
            """)

            result = await session.execute(query, {"limit": limit or 100000})
            rows = result.fetchall()

            tickers = [row[0] for row in rows]
            logger.info(f"Found {len(tickers)} tickers with fundamental metrics")

            return tickers

    async def get_risk_free_rate(self) -> Decimal:
        """Get current 10-Year Treasury rate as risk-free rate.

        Returns:
            Risk-free rate as decimal, defaults to 4.5%
        """
        async with self.db.session() as session:
            query = text("""
                SELECT rate_10y
                FROM treasury_rates
                ORDER BY date DESC
                LIMIT 1
            """)

            try:
                result = await session.execute(query)
                row = result.fetchone()
                if row and row[0]:
                    # Treasury rates stored as percentages (e.g., 4.5 for 4.5%)
                    return Decimal(str(row[0])) / 100
            except Exception:
                pass

            return Decimal("0.045")  # Default 4.5%

    async def get_beta(self, ticker: str) -> Optional[Decimal]:
        """Get stock's beta from risk_metrics table.

        Args:
            ticker: Stock ticker symbol.

        Returns:
            Beta value, or None if not available.
        """
        async with self.db.session() as session:
            query = text("""
                SELECT beta
                FROM risk_metrics
                WHERE ticker = :ticker
                  AND period = '1Y'
                ORDER BY time DESC
                LIMIT 1
            """)

            try:
                result = await session.execute(query, {"ticker": ticker})
                row = result.fetchone()
                return Decimal(str(row[0])) if row and row[0] else None
            except Exception:
                return None

    async def get_financial_data(self, ticker: str) -> Optional[Dict]:
        """Get TTM financial data for fair value calculation.

        Args:
            ticker: Stock ticker symbol.

        Returns:
            Dict with financial data, or None if not available.
        """
        async with self.db.session() as session:
            query = text("""
                SELECT
                    t.ticker,
                    t.revenue,
                    t.net_income,
                    t.operating_income,
                    t.eps_diluted,
                    t.shares_outstanding,
                    t.cash_and_equivalents,
                    t.short_term_debt,
                    t.long_term_debt,
                    t.free_cash_flow,
                    t.shareholders_equity,
                    t.total_assets,
                    v.stock_price,
                    v.ttm_market_cap
                FROM ttm_financials t
                LEFT JOIN valuation_ratios v ON t.ticker = v.ticker
                    AND v.calculation_date = (
                        SELECT MAX(calculation_date)
                        FROM valuation_ratios
                        WHERE ticker = t.ticker
                    )
                WHERE t.ticker = :ticker
                ORDER BY t.calculation_date DESC
                LIMIT 1
            """)

            try:
                result = await session.execute(query, {"ticker": ticker})
                row = result.fetchone()

                if not row:
                    return None

                return {
                    'ticker': row[0],
                    'revenue': Decimal(str(row[1])) if row[1] else None,
                    'net_income': Decimal(str(row[2])) if row[2] else None,
                    'operating_income': Decimal(str(row[3])) if row[3] else None,
                    'eps_diluted': Decimal(str(row[4])) if row[4] else None,
                    'shares_outstanding': int(row[5]) if row[5] else None,
                    'cash': Decimal(str(row[6])) if row[6] else Decimal(0),
                    'short_term_debt': Decimal(str(row[7])) if row[7] else Decimal(0),
                    'long_term_debt': Decimal(str(row[8])) if row[8] else Decimal(0),
                    'free_cash_flow': Decimal(str(row[9])) if row[9] else None,
                    'shareholders_equity': Decimal(str(row[10])) if row[10] else None,
                    'total_assets': Decimal(str(row[11])) if row[11] else None,
                    'current_price': Decimal(str(row[12])) if row[12] else None,
                    'market_cap': Decimal(str(row[13])) if row[13] else None,
                }
            except Exception as e:
                logger.error(f"Error fetching financial data for {ticker}: {e}")
                return None

    async def get_growth_rate(self, ticker: str) -> Optional[Decimal]:
        """Get historical growth rate for DCF projection.

        Args:
            ticker: Stock ticker symbol.

        Returns:
            Growth rate as decimal, or default 5%.
        """
        async with self.db.session() as session:
            query = text("""
                SELECT
                    COALESCE(revenue_growth_5y_cagr, revenue_growth_3y_cagr, revenue_growth_yoy) / 100
                FROM fundamental_metrics_extended
                WHERE ticker = :ticker
                ORDER BY calculation_date DESC
                LIMIT 1
            """)

            try:
                result = await session.execute(query, {"ticker": ticker})
                row = result.fetchone()
                if row and row[0] is not None:
                    growth = Decimal(str(row[0]))
                    # Cap growth between -10% and 30%
                    return max(Decimal("-0.10"), min(Decimal("0.30"), growth))
            except Exception:
                pass

            return Decimal("0.05")  # Default 5%

    async def calculate_fair_values(self, ticker: str) -> Optional[Dict]:
        """Calculate all fair value estimates for a ticker.

        Args:
            ticker: Stock ticker symbol.

        Returns:
            Dict with fair value metrics, or None if insufficient data.
        """
        # Get financial data
        financial_data = await self.get_financial_data(ticker)
        if not financial_data:
            logger.debug(f"{ticker}: No financial data available")
            return None

        # Check for required fields
        shares_outstanding = financial_data.get('shares_outstanding')
        fcf = financial_data.get('free_cash_flow')
        current_price = financial_data.get('current_price')
        market_cap = financial_data.get('market_cap')

        if not shares_outstanding or shares_outstanding <= 0:
            logger.debug(f"{ticker}: Missing shares outstanding")
            return None

        # Get WACC components
        risk_free_rate = await self.get_risk_free_rate()
        beta = await self.get_beta(ticker) or Decimal("1.0")

        # Calculate cost of equity
        cost_of_equity = calculate_cost_of_equity(beta, risk_free_rate)

        # Calculate total debt and net debt
        total_debt = (financial_data.get('short_term_debt') or Decimal(0)) + \
                     (financial_data.get('long_term_debt') or Decimal(0))
        cash = financial_data.get('cash') or Decimal(0)
        net_debt = total_debt - cash

        # Estimate cost of debt
        cost_of_debt = Decimal("0.05")  # Default 5% (need interest expense for better estimate)

        # Calculate WACC
        if market_cap and market_cap > 0:
            wacc = calculate_wacc(cost_of_equity, cost_of_debt, total_debt, market_cap)
        else:
            wacc = cost_of_equity  # Use cost of equity if no debt info

        # Get growth rate for DCF
        growth_rate = await self.get_growth_rate(ticker)

        # Initialize results
        dcf_fair_value = None
        dcf_upside = None
        graham_number = None
        epv_fair_value = None

        # Calculate DCF fair value
        if fcf and fcf > 0:
            dcf_fair_value, dcf_details = calculate_dcf_fair_value(
                fcf_ttm=fcf,
                growth_rate_high=growth_rate,
                growth_rate_terminal=Decimal("0.025"),
                wacc=wacc,
                shares_outstanding=shares_outstanding,
                net_debt=net_debt
            )

        # Calculate DCF upside
        if dcf_fair_value and current_price and current_price > 0:
            dcf_upside = ((dcf_fair_value - current_price) / current_price) * 100

        # Calculate Graham Number
        eps = financial_data.get('eps_diluted')
        book_value_per_share = None
        if financial_data.get('shareholders_equity') and shares_outstanding:
            book_value_per_share = financial_data['shareholders_equity'] / shares_outstanding

        graham_number = calculate_graham_number(eps, book_value_per_share)

        # Calculate EPV
        operating_income = financial_data.get('operating_income')
        if operating_income and operating_income > 0:
            epv_fair_value = calculate_earnings_power_value(
                normalized_ebit=operating_income,
                wacc=wacc,
                net_debt=net_debt,
                shares_outstanding=shares_outstanding
            )

        return {
            'ticker': ticker,
            'calculation_date': date.today(),
            'dcf_fair_value': dcf_fair_value,
            'dcf_upside_percent': dcf_upside,
            'graham_number': graham_number,
            'epv_fair_value': epv_fair_value,
            'wacc': wacc * 100,  # Store as percentage
            'beta': beta,
            'cost_of_equity': cost_of_equity * 100,
            'cost_of_debt': cost_of_debt * 100,
            'current_price': current_price,
        }

    async def update_fair_value_metrics(self, fair_values: Dict) -> bool:
        """Update fair value metrics in fundamental_metrics_extended table.

        Args:
            fair_values: Dict with calculated fair values.

        Returns:
            True if successful, False otherwise.
        """
        try:
            async with self.db.session() as session:
                def to_float(val):
                    if val is None:
                        return None
                    return float(val) if isinstance(val, Decimal) else val

                query = text("""
                    UPDATE fundamental_metrics_extended
                    SET
                        dcf_fair_value = :dcf_fair_value,
                        dcf_upside_percent = :dcf_upside_percent,
                        graham_number = :graham_number,
                        epv_fair_value = :epv_fair_value,
                        wacc = :wacc,
                        beta = :beta,
                        cost_of_equity = :cost_of_equity,
                        cost_of_debt = :cost_of_debt,
                        updated_at = NOW()
                    WHERE ticker = :ticker
                      AND calculation_date = :calculation_date
                """)

                await session.execute(query, {
                    'ticker': fair_values['ticker'],
                    'calculation_date': fair_values['calculation_date'],
                    'dcf_fair_value': to_float(fair_values.get('dcf_fair_value')),
                    'dcf_upside_percent': to_float(fair_values.get('dcf_upside_percent')),
                    'graham_number': to_float(fair_values.get('graham_number')),
                    'epv_fair_value': to_float(fair_values.get('epv_fair_value')),
                    'wacc': to_float(fair_values.get('wacc')),
                    'beta': to_float(fair_values.get('beta')),
                    'cost_of_equity': to_float(fair_values.get('cost_of_equity')),
                    'cost_of_debt': to_float(fair_values.get('cost_of_debt')),
                })
                await session.commit()

                return True

        except Exception as e:
            logger.error(f"Error updating fair values for {fair_values['ticker']}: {e}")
            return False

    async def process_ticker(self, ticker: str) -> bool:
        """Process a single ticker to calculate fair values.

        Args:
            ticker: Stock ticker symbol.

        Returns:
            True if successful, False otherwise.
        """
        try:
            # Calculate fair values
            fair_values = await self.calculate_fair_values(ticker)

            if not fair_values:
                logger.debug(f"{ticker}: Could not calculate fair values")
                return False

            # Update in database
            success = await self.update_fair_value_metrics(fair_values)

            if success:
                dcf = fair_values.get('dcf_fair_value')
                upside = fair_values.get('dcf_upside_percent')
                graham = fair_values.get('graham_number')
                logger.debug(
                    f"{ticker}: DCF: ${dcf:.2f}, Upside: {upside:.1f}%, Graham: ${graham:.2f}"
                    if dcf and upside and graham
                    else f"{ticker}: Partial fair value calculated"
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
        progress_bar = tqdm(total=len(tickers), desc="Calculating Fair Values")

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
        """Run the fair value calculation process.

        Args:
            limit: Limit number of tickers to process.
            ticker: Process single ticker.
            all_tickers: Process all tickers with sufficient data.
        """
        start_time = datetime.now()
        logger.info("=" * 80)
        logger.info("Fair Value Calculator (Phase 5)")
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
        logger.info("Fair Value Calculation Complete")
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
        description='Calculate DCF fair value estimates (Phase 5)',
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  # Test on 10 tickers
  python fair_value_calculator.py --limit 10

  # Process all tickers
  python fair_value_calculator.py --all

  # Process single ticker
  python fair_value_calculator.py --ticker AAPL
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
        help='Process all tickers with fundamental metrics'
    )

    args = parser.parse_args()

    # Validate arguments
    if args.ticker and (args.all or args.limit):
        parser.error("--ticker cannot be used with --all or --limit")

    # Run calculator
    calculator = FairValueCalculator()
    asyncio.run(calculator.run(
        limit=args.limit,
        ticker=args.ticker,
        all_tickers=args.all
    ))


if __name__ == '__main__':
    main()
