#!/usr/bin/env python3
"""Fundamental Metrics Calculator Pipeline.

Calculates extended financial metrics from existing data including:
- Phase 1: EBITDA Margin, EV/FCF, Growth Rates (YoY, 3Y/5Y CAGR)
- Phase 2: Dividend Metrics (Payout Ratio, Growth Rate, Consecutive Years)
- Phase 3: Leverage Metrics (Interest Coverage, Net Debt/EBITDA)

Usage:
    python fundamental_metrics_calculator.py --all          # Calculate for all tickers
    python fundamental_metrics_calculator.py --limit 100    # Test on 100 tickers
    python fundamental_metrics_calculator.py --ticker AAPL  # Single ticker
"""

import argparse
import asyncio
import logging
import sys
from datetime import date, datetime, timedelta
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
# PHASE 1: PROFITABILITY & GROWTH CALCULATIONS
# =============================================================================

def calculate_ebitda_margin(
    ebitda: Optional[Decimal],
    revenue: Optional[Decimal]
) -> Optional[Decimal]:
    """EBITDA Margin = EBITDA / Revenue x 100.

    Args:
        ebitda: EBITDA value (can be calculated as operating_income + depreciation)
        revenue: Total revenue

    Returns:
        EBITDA margin as percentage, or None if insufficient data
    """
    if not ebitda or not revenue or revenue == 0:
        return None
    return (ebitda / revenue) * 100


def calculate_ev_to_fcf(
    enterprise_value: Optional[Decimal],
    fcf: Optional[Decimal]
) -> Optional[Decimal]:
    """EV/FCF = Enterprise Value / Free Cash Flow.

    Args:
        enterprise_value: Enterprise value of the company
        fcf: Free cash flow (TTM)

    Returns:
        EV/FCF ratio, or None if FCF is non-positive or data missing
    """
    if not enterprise_value or not fcf or fcf <= 0:
        return None
    return enterprise_value / fcf


def calculate_yoy_growth(
    current: Optional[Decimal],
    prior: Optional[Decimal],
    metric_type: str = "revenue"
) -> Optional[Decimal]:
    """Year-over-Year Growth = (Current - Prior) / Prior x 100.

    Special handling for metrics that can be negative (EPS, FCF):
    - If signs change (loss to profit or vice versa), return None (not meaningful)
    - Revenue should always be positive, so no special handling needed

    Args:
        current: Current period value
        prior: Prior period value
        metric_type: Type of metric ("revenue", "eps", "fcf")

    Returns:
        YoY growth as percentage, or None if not meaningful

    Examples:
        - Revenue: $100M -> $120M = 20% growth
        - EPS: $2.00 -> $2.50 = 25% growth
        - EPS: -$1.00 -> $1.00 = None (turnaround, not meaningful)
        - EPS: -$2.00 -> -$1.00 = 50% (loss narrowing = improvement)
    """
    if current is None or prior is None or prior == 0:
        return None

    # For metrics that can be negative, check for sign changes
    if metric_type in ("eps", "fcf"):
        # Sign change = not meaningful for growth calculation
        if (current > 0 and prior < 0) or (current < 0 and prior > 0):
            return None
        # Both negative: calculate improvement in losses
        # e.g., -$2 -> -$1 is 50% improvement (loss narrowed)
        if current < 0 and prior < 0:
            return ((abs(prior) - abs(current)) / abs(prior)) * 100

    return ((current - prior) / prior) * 100


def calculate_cagr(
    start_value: Optional[Decimal],
    end_value: Optional[Decimal],
    years: int
) -> Optional[Decimal]:
    """Compound Annual Growth Rate = ((End/Start)^(1/years) - 1) x 100.

    Args:
        start_value: Value at start of period
        end_value: Value at end of period
        years: Number of years in period

    Returns:
        CAGR as percentage, or None if values invalid

    Example:
        Revenue grew from $100M to $150M over 3 years
        CAGR = ((150/100)^(1/3) - 1) x 100 = 14.47%
    """
    if not start_value or not end_value or start_value <= 0 or end_value <= 0 or years <= 0:
        return None
    try:
        ratio = float(end_value / start_value)
        cagr = (ratio ** (1 / years) - 1) * 100
        return Decimal(str(round(cagr, 4)))
    except (ValueError, InvalidOperation):
        return None


# =============================================================================
# PHASE 2: DIVIDEND CALCULATIONS
# =============================================================================

def calculate_payout_ratio(
    annual_dividend: Optional[Decimal],
    eps: Optional[Decimal]
) -> Optional[Decimal]:
    """Payout Ratio = (Annual Dividend per Share / EPS) x 100.

    Interpretation:
    - < 30%: Very conservative, room for dividend growth
    - 30-50%: Healthy, sustainable
    - 50-75%: Moderate risk
    - > 75%: High risk, may not be sustainable
    - > 100%: Paying more than earnings (using reserves/debt)

    Args:
        annual_dividend: Annual dividend per share
        eps: Earnings per share (TTM)

    Returns:
        Payout ratio as percentage, or None if EPS non-positive
    """
    if annual_dividend is None or eps is None or eps <= 0:
        return None
    return (annual_dividend / eps) * 100


def calculate_dividend_growth_rate(
    current_annual: Optional[Decimal],
    prior_annual: Optional[Decimal],
    years_diff: int = 1
) -> Optional[Decimal]:
    """Calculate dividend growth rate (CAGR if multi-year).

    Args:
        current_annual: Current year annual dividend
        prior_annual: Prior period annual dividend
        years_diff: Number of years between periods

    Returns:
        Dividend growth rate as percentage
    """
    if not current_annual or not prior_annual or prior_annual <= 0:
        return None
    if years_diff <= 0:
        return None
    if years_diff == 1:
        return ((current_annual - prior_annual) / prior_annual) * 100
    return calculate_cagr(prior_annual, current_annual, years_diff)


def count_consecutive_dividend_growth_years(dividend_history: List[Dict]) -> int:
    """Count consecutive years of dividend increases.

    Dividend Aristocrats: 25+ years of consecutive increases
    Dividend Champions: 25+ years
    Dividend Contenders: 10-24 years
    Dividend Challengers: 5-9 years

    Args:
        dividend_history: List of {year, amount} dicts sorted by year desc

    Returns:
        Number of consecutive years with dividend increases
    """
    if len(dividend_history) < 2:
        return 0

    # Sort by year descending (most recent first)
    sorted_history = sorted(dividend_history, key=lambda x: x["year"], reverse=True)

    consecutive_years = 0
    for i in range(len(sorted_history) - 1):
        current = sorted_history[i]["amount"]
        prior = sorted_history[i + 1]["amount"]

        # Check if years are consecutive (no gaps)
        if sorted_history[i]["year"] - sorted_history[i + 1]["year"] != 1:
            break

        # Check if dividend increased (strictly greater, not equal)
        if current is not None and prior is not None and current > prior:
            consecutive_years += 1
        else:
            # Dividend was cut or held flat - stop counting
            break

    return consecutive_years


# =============================================================================
# PHASE 3: LEVERAGE CALCULATIONS
# =============================================================================

def calculate_interest_coverage(
    operating_income: Optional[Decimal],
    interest_expense: Optional[Decimal]
) -> Optional[Decimal]:
    """Interest Coverage Ratio = Operating Income (EBIT) / Interest Expense.

    Also known as Times Interest Earned (TIE) ratio.

    Interpretation:
    - > 5: Excellent, very safe
    - 3-5: Good, healthy
    - 2-3: Adequate, some risk
    - 1-2: Poor, high risk
    - < 1: Cannot cover interest payments (distressed)

    Args:
        operating_income: Operating income (EBIT)
        interest_expense: Interest expense

    Returns:
        Interest coverage ratio, or None if data missing
    """
    if operating_income is None or interest_expense is None:
        return None
    if interest_expense <= 0:
        return None  # No interest expense = N/A (not infinite)

    return operating_income / interest_expense


def calculate_net_debt_to_ebitda(
    total_debt: Optional[Decimal],
    cash: Optional[Decimal],
    ebitda: Optional[Decimal]
) -> Optional[Decimal]:
    """Net Debt / EBITDA = (Total Debt - Cash) / EBITDA.

    Measures how many years of EBITDA needed to pay off debt.

    Interpretation:
    - < 1: Very low leverage, strong balance sheet
    - 1-2: Conservative leverage
    - 2-3: Moderate leverage
    - 3-4: Elevated leverage
    - > 4: High leverage, potential distress
    - Negative: More cash than debt (net cash position)

    Note: Industry context matters - utilities/REITs typically run higher leverage.

    Args:
        total_debt: Total debt (short + long term)
        cash: Cash and cash equivalents
        ebitda: EBITDA (TTM)

    Returns:
        Net Debt / EBITDA ratio, or None if EBITDA non-positive
    """
    if total_debt is None or cash is None or ebitda is None:
        return None
    if ebitda <= 0:
        return None  # Negative EBITDA makes ratio meaningless

    net_debt = total_debt - cash
    return net_debt / ebitda


# =============================================================================
# MAIN CALCULATOR CLASS
# =============================================================================

class FundamentalMetricsCalculator:
    """Calculate extended fundamental metrics for all tickers."""

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

            # Get tickers with TTM financial data
            query = text("""
                SELECT DISTINCT ticker
                FROM ttm_financials
                WHERE ticker NOT LIKE '%-%'
                  AND ticker NOT LIKE '%.%'
                ORDER BY ticker
                LIMIT :limit
            """)

            result = await session.execute(query, {"limit": limit or 100000})
            rows = result.fetchall()

            tickers = [row[0] for row in rows]
            logger.info(f"Found {len(tickers)} tickers with TTM financial data")

            return tickers

    async def get_ttm_financials(self, ticker: str) -> Optional[Dict]:
        """Get most recent TTM financials for a ticker.

        Args:
            ticker: Stock ticker symbol.

        Returns:
            Dict with TTM financial data, or None if not found.
        """
        async with self.db.session() as session:
            query = text("""
                SELECT
                    ticker, calculation_date,
                    revenue, net_income, eps_basic, eps_diluted,
                    shares_outstanding, total_assets, total_liabilities,
                    shareholders_equity, cash_and_equivalents,
                    short_term_debt, long_term_debt,
                    operating_cash_flow, free_cash_flow, capex,
                    operating_income, gross_profit
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
                'ticker': row[0],
                'calculation_date': row[1],
                'revenue': Decimal(str(row[2])) if row[2] else None,
                'net_income': Decimal(str(row[3])) if row[3] else None,
                'eps_basic': Decimal(str(row[4])) if row[4] else None,
                'eps_diluted': Decimal(str(row[5])) if row[5] else None,
                'shares_outstanding': int(row[6]) if row[6] else None,
                'total_assets': Decimal(str(row[7])) if row[7] else None,
                'total_liabilities': Decimal(str(row[8])) if row[8] else None,
                'shareholders_equity': Decimal(str(row[9])) if row[9] else None,
                'cash_and_equivalents': Decimal(str(row[10])) if row[10] else None,
                'short_term_debt': Decimal(str(row[11])) if row[11] else None,
                'long_term_debt': Decimal(str(row[12])) if row[12] else None,
                'operating_cash_flow': Decimal(str(row[13])) if row[13] else None,
                'free_cash_flow': Decimal(str(row[14])) if row[14] else None,
                'capex': Decimal(str(row[15])) if row[15] else None,
                'operating_income': Decimal(str(row[16])) if row[16] else None,
                'gross_profit': Decimal(str(row[17])) if row[17] else None,
            }

    async def get_historical_annual_values(
        self,
        ticker: str,
        field: str,
        years_back: int = 5
    ) -> Dict[int, Decimal]:
        """Fetch annual values for a specific field going back N years.

        Args:
            ticker: Stock ticker symbol.
            field: Database field name to fetch.
            years_back: Number of years of history to fetch.

        Returns:
            Dict mapping fiscal_year -> value
        """
        async with self.db.session() as session:
            query = text(f"""
                SELECT fiscal_year, {field}
                FROM financials
                WHERE ticker = :ticker
                  AND statement_type = '10-K'
                  AND fiscal_quarter IS NULL
                  AND fiscal_year >= :min_year
                  AND {field} IS NOT NULL
                ORDER BY fiscal_year DESC
            """)

            result = await session.execute(query, {
                "ticker": ticker,
                "min_year": date.today().year - years_back
            })

            return {row[0]: Decimal(str(row[1])) for row in result.fetchall() if row[1] is not None}

    async def calculate_growth_rates(self, ticker: str) -> Dict:
        """Calculate all growth rate metrics for a ticker.

        Args:
            ticker: Stock ticker symbol.

        Returns:
            Dict with growth rate metrics.
        """
        # Get historical data
        revenue_history = await self.get_historical_annual_values(ticker, 'revenue', 5)
        eps_history = await self.get_historical_annual_values(ticker, 'eps_diluted', 5)
        fcf_history = await self.get_historical_annual_values(ticker, 'free_cash_flow', 5)

        current_year = date.today().year - 1  # Most recent complete fiscal year

        growth_rates = {}

        # Revenue Growth
        if current_year in revenue_history and (current_year - 1) in revenue_history:
            growth_rates['revenue_growth_yoy'] = calculate_yoy_growth(
                revenue_history[current_year],
                revenue_history[current_year - 1],
                metric_type="revenue"
            )

        if current_year in revenue_history and (current_year - 3) in revenue_history:
            growth_rates['revenue_growth_3y_cagr'] = calculate_cagr(
                revenue_history[current_year - 3],
                revenue_history[current_year],
                3
            )

        if current_year in revenue_history and (current_year - 5) in revenue_history:
            growth_rates['revenue_growth_5y_cagr'] = calculate_cagr(
                revenue_history[current_year - 5],
                revenue_history[current_year],
                5
            )

        # EPS Growth
        if current_year in eps_history and (current_year - 1) in eps_history:
            growth_rates['eps_growth_yoy'] = calculate_yoy_growth(
                eps_history[current_year],
                eps_history[current_year - 1],
                metric_type="eps"
            )

        if current_year in eps_history and (current_year - 3) in eps_history:
            growth_rates['eps_growth_3y_cagr'] = calculate_cagr(
                eps_history[current_year - 3],
                eps_history[current_year],
                3
            )

        if current_year in eps_history and (current_year - 5) in eps_history:
            growth_rates['eps_growth_5y_cagr'] = calculate_cagr(
                eps_history[current_year - 5],
                eps_history[current_year],
                5
            )

        # FCF Growth
        if current_year in fcf_history and (current_year - 1) in fcf_history:
            growth_rates['fcf_growth_yoy'] = calculate_yoy_growth(
                fcf_history[current_year],
                fcf_history[current_year - 1],
                metric_type="fcf"
            )

        return growth_rates

    async def get_dividend_history(
        self,
        ticker: str,
        years_back: int = 10
    ) -> List[Dict]:
        """Fetch dividend payment history for a ticker.

        Args:
            ticker: Stock ticker symbol.
            years_back: Number of years of history to fetch.

        Returns:
            List of {year, amount, payments} dicts sorted by year desc.
        """
        async with self.db.session() as session:
            query = text("""
                SELECT
                    EXTRACT(YEAR FROM ex_date)::int as year,
                    SUM(amount) as annual_dividend,
                    COUNT(*) as payment_count
                FROM dividends
                WHERE symbol = :ticker
                  AND ex_date >= :start_date
                  AND type = 'CD'
                GROUP BY EXTRACT(YEAR FROM ex_date)
                ORDER BY year DESC
            """)

            try:
                result = await session.execute(query, {
                    "ticker": ticker,
                    "start_date": date.today() - timedelta(days=365 * years_back)
                })
                return [
                    {"year": int(row[0]), "amount": Decimal(str(row[1])), "payments": int(row[2])}
                    for row in result.fetchall()
                ]
            except Exception:
                # Table might not exist or be empty
                return []

    async def calculate_dividend_metrics(
        self,
        ticker: str,
        ttm_eps: Optional[Decimal],
        current_price: Optional[Decimal]
    ) -> Dict:
        """Calculate all dividend-related metrics for a ticker.

        Args:
            ticker: Stock ticker symbol.
            ttm_eps: TTM earnings per share.
            current_price: Current stock price.

        Returns:
            Dict with dividend metrics.
        """
        dividend_history = await self.get_dividend_history(ticker, years_back=10)

        if not dividend_history:
            return {
                "dividend_yield": None,
                "payout_ratio": None,
                "dividend_growth_rate": None,
                "consecutive_dividend_years": 0
            }

        # Most recent annual dividend
        current_year = date.today().year
        current_year_dividend = None
        prior_year_dividend = None

        for record in dividend_history:
            if record["year"] == current_year - 1:
                current_year_dividend = record["amount"]
            elif record["year"] == current_year - 2:
                prior_year_dividend = record["amount"]

        # Use most recent if current year not complete
        if current_year_dividend is None and dividend_history:
            current_year_dividend = dividend_history[0]["amount"]

        # Dividend Yield = Annual Dividend / Current Price x 100
        dividend_yield = None
        if current_year_dividend and current_price and current_price > 0:
            dividend_yield = (current_year_dividend / current_price) * 100

        # Calculate 5-year dividend growth rate
        dividend_growth_rate = None
        if len(dividend_history) >= 2:
            oldest = dividend_history[-1]
            newest = dividend_history[0]
            years_diff = newest["year"] - oldest["year"]
            if years_diff > 0:
                dividend_growth_rate = calculate_dividend_growth_rate(
                    newest["amount"], oldest["amount"], years_diff
                )

        return {
            "dividend_yield": dividend_yield,
            "payout_ratio": calculate_payout_ratio(current_year_dividend, ttm_eps),
            "dividend_growth_rate": dividend_growth_rate,
            "consecutive_dividend_years": count_consecutive_dividend_growth_years(dividend_history)
        }

    async def calculate_leverage_metrics(self, ttm_data: Dict) -> Dict:
        """Calculate all leverage-related metrics for a ticker.

        Args:
            ttm_data: TTM financial data dict.

        Returns:
            Dict with leverage metrics.
        """
        if not ttm_data:
            return {
                "debt_to_equity": None,
                "interest_coverage": None,
                "net_debt_to_ebitda": None
            }

        # Calculate total debt
        total_debt = None
        if ttm_data.get('short_term_debt') is not None or ttm_data.get('long_term_debt') is not None:
            total_debt = (ttm_data.get('short_term_debt') or Decimal(0)) + \
                         (ttm_data.get('long_term_debt') or Decimal(0))

        # Calculate EBITDA (approximate as operating income if not available)
        ebitda = ttm_data.get('operating_income')

        # Debt to Equity
        debt_to_equity = None
        if total_debt and ttm_data.get('shareholders_equity') and ttm_data['shareholders_equity'] > 0:
            debt_to_equity = total_debt / ttm_data['shareholders_equity']

        return {
            "debt_to_equity": debt_to_equity,
            "interest_coverage": None,  # Requires interest expense from SEC
            "net_debt_to_ebitda": calculate_net_debt_to_ebitda(
                total_debt,
                ttm_data.get('cash_and_equivalents'),
                ebitda
            )
        }

    async def get_company_info(self, ticker: str) -> Optional[Dict]:
        """Get company sector and industry info.

        Args:
            ticker: Stock ticker symbol.

        Returns:
            Dict with company info, or None if not found.
        """
        async with self.db.session() as session:
            query = text("""
                SELECT ticker, sector, industry, market_cap
                FROM companies
                WHERE ticker = :ticker
            """)

            result = await session.execute(query, {"ticker": ticker})
            row = result.fetchone()

            if not row:
                return None

            return {
                'ticker': row[0],
                'sector': row[1],
                'industry': row[2],
                'market_cap': row[3],
            }

    async def get_current_price(self, ticker: str) -> Optional[Decimal]:
        """Get current stock price for a ticker.

        Args:
            ticker: Stock ticker symbol.

        Returns:
            Current stock price, or None if not available.
        """
        async with self.db.session() as session:
            query = text("""
                SELECT close
                FROM stock_prices
                WHERE ticker = :ticker
                ORDER BY time DESC
                LIMIT 1
            """)

            try:
                result = await session.execute(query, {"ticker": ticker})
                row = result.fetchone()
                return Decimal(str(row[0])) if row and row[0] else None
            except Exception:
                return None

    async def get_valuation_metrics(self, ticker: str) -> Dict:
        """Get valuation metrics from valuation_ratios table.

        Args:
            ticker: Stock ticker symbol.

        Returns:
            Dict with valuation metrics.
        """
        async with self.db.session() as session:
            query = text("""
                SELECT
                    stock_price, ttm_pe_ratio, ttm_pb_ratio, ttm_ps_ratio,
                    ttm_market_cap
                FROM valuation_ratios
                WHERE ticker = :ticker
                ORDER BY calculation_date DESC
                LIMIT 1
            """)

            try:
                result = await session.execute(query, {"ticker": ticker})
                row = result.fetchone()

                if not row:
                    return {}

                return {
                    'stock_price': Decimal(str(row[0])) if row[0] else None,
                    'pe_ratio': Decimal(str(row[1])) if row[1] else None,
                    'pb_ratio': Decimal(str(row[2])) if row[2] else None,
                    'ps_ratio': Decimal(str(row[3])) if row[3] else None,
                    'market_cap': int(row[4]) if row[4] else None,
                }
            except Exception:
                return {}

    async def calculate_all_metrics(self, ticker: str) -> Optional[Dict]:
        """Calculate all extended fundamental metrics for a ticker.

        Args:
            ticker: Stock ticker symbol.

        Returns:
            Dict with all calculated metrics, or None if insufficient data.
        """
        # Get TTM financials
        ttm_data = await self.get_ttm_financials(ticker)
        if not ttm_data:
            logger.debug(f"{ticker}: No TTM financials found")
            return None

        # Get valuation metrics
        valuation = await self.get_valuation_metrics(ticker)
        current_price = valuation.get('stock_price') or await self.get_current_price(ticker)

        # Calculate profitability margins from TTM data
        gross_margin = None
        operating_margin = None
        net_margin = None
        ebitda_margin = None

        if ttm_data.get('revenue') and ttm_data['revenue'] > 0:
            if ttm_data.get('gross_profit'):
                gross_margin = (ttm_data['gross_profit'] / ttm_data['revenue']) * 100
            if ttm_data.get('operating_income'):
                operating_margin = (ttm_data['operating_income'] / ttm_data['revenue']) * 100
                # Use operating income as proxy for EBITDA
                ebitda_margin = operating_margin
            if ttm_data.get('net_income'):
                net_margin = (ttm_data['net_income'] / ttm_data['revenue']) * 100

        # Calculate returns
        roe = None
        roa = None
        roic = None

        if ttm_data.get('net_income'):
            if ttm_data.get('shareholders_equity') and ttm_data['shareholders_equity'] > 0:
                roe = (ttm_data['net_income'] / ttm_data['shareholders_equity']) * 100
            if ttm_data.get('total_assets') and ttm_data['total_assets'] > 0:
                roa = (ttm_data['net_income'] / ttm_data['total_assets']) * 100

        # Calculate growth rates
        growth_rates = await self.calculate_growth_rates(ticker)

        # Calculate dividend metrics
        dividend_metrics = await self.calculate_dividend_metrics(
            ticker,
            ttm_data.get('eps_diluted'),
            current_price
        )

        # Calculate leverage metrics
        leverage_metrics = await self.calculate_leverage_metrics(ttm_data)

        # Calculate EV metrics
        enterprise_value = None
        ev_to_revenue = None
        ev_to_ebitda = None
        ev_to_fcf = None

        if valuation.get('market_cap') and ttm_data.get('short_term_debt') is not None:
            total_debt = (ttm_data.get('short_term_debt') or 0) + (ttm_data.get('long_term_debt') or 0)
            cash = ttm_data.get('cash_and_equivalents') or 0
            enterprise_value = Decimal(str(valuation['market_cap'])) + Decimal(str(total_debt)) - Decimal(str(cash))

            if enterprise_value > 0:
                if ttm_data.get('revenue') and ttm_data['revenue'] > 0:
                    ev_to_revenue = enterprise_value / ttm_data['revenue']
                if ttm_data.get('operating_income') and ttm_data['operating_income'] > 0:
                    ev_to_ebitda = enterprise_value / ttm_data['operating_income']
                if ttm_data.get('free_cash_flow') and ttm_data['free_cash_flow'] > 0:
                    ev_to_fcf = enterprise_value / ttm_data['free_cash_flow']

        # Compile all metrics
        metrics = {
            'ticker': ticker,
            'calculation_date': date.today(),

            # Profitability
            'gross_margin': gross_margin,
            'operating_margin': operating_margin,
            'net_margin': net_margin,
            'ebitda_margin': ebitda_margin,

            # Returns
            'roe': roe,
            'roa': roa,
            'roic': roic,

            # Growth Rates
            'revenue_growth_yoy': growth_rates.get('revenue_growth_yoy'),
            'revenue_growth_3y_cagr': growth_rates.get('revenue_growth_3y_cagr'),
            'revenue_growth_5y_cagr': growth_rates.get('revenue_growth_5y_cagr'),
            'eps_growth_yoy': growth_rates.get('eps_growth_yoy'),
            'eps_growth_3y_cagr': growth_rates.get('eps_growth_3y_cagr'),
            'eps_growth_5y_cagr': growth_rates.get('eps_growth_5y_cagr'),
            'fcf_growth_yoy': growth_rates.get('fcf_growth_yoy'),

            # Valuation
            'enterprise_value': enterprise_value,
            'ev_to_revenue': ev_to_revenue,
            'ev_to_ebitda': ev_to_ebitda,
            'ev_to_fcf': ev_to_fcf,

            # Liquidity (from TTM)
            'current_ratio': None,  # Need current assets/liabilities
            'quick_ratio': None,

            # Leverage
            'debt_to_equity': leverage_metrics.get('debt_to_equity'),
            'interest_coverage': leverage_metrics.get('interest_coverage'),
            'net_debt_to_ebitda': leverage_metrics.get('net_debt_to_ebitda'),

            # Dividends
            'dividend_yield': dividend_metrics.get('dividend_yield'),
            'payout_ratio': dividend_metrics.get('payout_ratio'),
            'dividend_growth_rate': dividend_metrics.get('dividend_growth_rate'),
            'consecutive_dividend_years': dividend_metrics.get('consecutive_dividend_years', 0),

            # Fair Value (Phase 5 - placeholder)
            'dcf_fair_value': None,
            'dcf_upside_percent': None,
            'graham_number': None,

            # Sector Comparisons (Phase 4 - placeholder)
            'pe_sector_percentile': None,
            'pb_sector_percentile': None,
            'roe_sector_percentile': None,
            'margin_sector_percentile': None,

            # Data Quality
            'data_quality_score': self._calculate_data_quality_score(ttm_data, growth_rates),
        }

        return metrics

    def _calculate_data_quality_score(self, ttm_data: Dict, growth_rates: Dict) -> Decimal:
        """Calculate data quality/completeness score (0-100).

        Args:
            ttm_data: TTM financial data.
            growth_rates: Growth rate metrics.

        Returns:
            Data quality score as percentage.
        """
        total_fields = 15
        filled_fields = 0

        # Check key TTM fields
        key_fields = ['revenue', 'net_income', 'eps_diluted', 'free_cash_flow',
                      'shareholders_equity', 'total_assets', 'operating_income']
        for field in key_fields:
            if ttm_data.get(field) is not None:
                filled_fields += 1

        # Check growth rate fields
        growth_fields = ['revenue_growth_yoy', 'eps_growth_yoy', 'revenue_growth_3y_cagr']
        for field in growth_fields:
            if growth_rates.get(field) is not None:
                filled_fields += 1

        # Additional points for having 5-year data
        if growth_rates.get('revenue_growth_5y_cagr') is not None:
            filled_fields += 2
        if growth_rates.get('eps_growth_5y_cagr') is not None:
            filled_fields += 2

        return Decimal(str(round((filled_fields / total_fields) * 100, 2)))

    async def store_metrics(self, metrics: Dict) -> bool:
        """Store calculated metrics in database.

        Args:
            metrics: Dict with calculated metrics.

        Returns:
            True if successful, False otherwise.
        """
        try:
            async with self.db.session() as session:
                # Convert Decimal values to float for storage
                def to_float(val):
                    if val is None:
                        return None
                    return float(val) if isinstance(val, Decimal) else val

                query = text("""
                    INSERT INTO fundamental_metrics_extended (
                        ticker, calculation_date,
                        gross_margin, operating_margin, net_margin, ebitda_margin,
                        roe, roa, roic,
                        revenue_growth_yoy, revenue_growth_3y_cagr, revenue_growth_5y_cagr,
                        eps_growth_yoy, eps_growth_3y_cagr, eps_growth_5y_cagr,
                        fcf_growth_yoy,
                        enterprise_value, ev_to_revenue, ev_to_ebitda, ev_to_fcf,
                        current_ratio, quick_ratio,
                        debt_to_equity, interest_coverage, net_debt_to_ebitda,
                        dividend_yield, payout_ratio, dividend_growth_rate,
                        consecutive_dividend_years,
                        dcf_fair_value, dcf_upside_percent, graham_number,
                        pe_sector_percentile, pb_sector_percentile,
                        roe_sector_percentile, margin_sector_percentile,
                        data_quality_score
                    ) VALUES (
                        :ticker, :calculation_date,
                        :gross_margin, :operating_margin, :net_margin, :ebitda_margin,
                        :roe, :roa, :roic,
                        :revenue_growth_yoy, :revenue_growth_3y_cagr, :revenue_growth_5y_cagr,
                        :eps_growth_yoy, :eps_growth_3y_cagr, :eps_growth_5y_cagr,
                        :fcf_growth_yoy,
                        :enterprise_value, :ev_to_revenue, :ev_to_ebitda, :ev_to_fcf,
                        :current_ratio, :quick_ratio,
                        :debt_to_equity, :interest_coverage, :net_debt_to_ebitda,
                        :dividend_yield, :payout_ratio, :dividend_growth_rate,
                        :consecutive_dividend_years,
                        :dcf_fair_value, :dcf_upside_percent, :graham_number,
                        :pe_sector_percentile, :pb_sector_percentile,
                        :roe_sector_percentile, :margin_sector_percentile,
                        :data_quality_score
                    )
                    ON CONFLICT (ticker, calculation_date)
                    DO UPDATE SET
                        gross_margin = EXCLUDED.gross_margin,
                        operating_margin = EXCLUDED.operating_margin,
                        net_margin = EXCLUDED.net_margin,
                        ebitda_margin = EXCLUDED.ebitda_margin,
                        roe = EXCLUDED.roe,
                        roa = EXCLUDED.roa,
                        roic = EXCLUDED.roic,
                        revenue_growth_yoy = EXCLUDED.revenue_growth_yoy,
                        revenue_growth_3y_cagr = EXCLUDED.revenue_growth_3y_cagr,
                        revenue_growth_5y_cagr = EXCLUDED.revenue_growth_5y_cagr,
                        eps_growth_yoy = EXCLUDED.eps_growth_yoy,
                        eps_growth_3y_cagr = EXCLUDED.eps_growth_3y_cagr,
                        eps_growth_5y_cagr = EXCLUDED.eps_growth_5y_cagr,
                        fcf_growth_yoy = EXCLUDED.fcf_growth_yoy,
                        enterprise_value = EXCLUDED.enterprise_value,
                        ev_to_revenue = EXCLUDED.ev_to_revenue,
                        ev_to_ebitda = EXCLUDED.ev_to_ebitda,
                        ev_to_fcf = EXCLUDED.ev_to_fcf,
                        current_ratio = EXCLUDED.current_ratio,
                        quick_ratio = EXCLUDED.quick_ratio,
                        debt_to_equity = EXCLUDED.debt_to_equity,
                        interest_coverage = EXCLUDED.interest_coverage,
                        net_debt_to_ebitda = EXCLUDED.net_debt_to_ebitda,
                        dividend_yield = EXCLUDED.dividend_yield,
                        payout_ratio = EXCLUDED.payout_ratio,
                        dividend_growth_rate = EXCLUDED.dividend_growth_rate,
                        consecutive_dividend_years = EXCLUDED.consecutive_dividend_years,
                        dcf_fair_value = EXCLUDED.dcf_fair_value,
                        dcf_upside_percent = EXCLUDED.dcf_upside_percent,
                        graham_number = EXCLUDED.graham_number,
                        pe_sector_percentile = EXCLUDED.pe_sector_percentile,
                        pb_sector_percentile = EXCLUDED.pb_sector_percentile,
                        roe_sector_percentile = EXCLUDED.roe_sector_percentile,
                        margin_sector_percentile = EXCLUDED.margin_sector_percentile,
                        data_quality_score = EXCLUDED.data_quality_score,
                        updated_at = CURRENT_TIMESTAMP
                """)

                params = {
                    'ticker': metrics['ticker'],
                    'calculation_date': metrics['calculation_date'],
                    'gross_margin': to_float(metrics.get('gross_margin')),
                    'operating_margin': to_float(metrics.get('operating_margin')),
                    'net_margin': to_float(metrics.get('net_margin')),
                    'ebitda_margin': to_float(metrics.get('ebitda_margin')),
                    'roe': to_float(metrics.get('roe')),
                    'roa': to_float(metrics.get('roa')),
                    'roic': to_float(metrics.get('roic')),
                    'revenue_growth_yoy': to_float(metrics.get('revenue_growth_yoy')),
                    'revenue_growth_3y_cagr': to_float(metrics.get('revenue_growth_3y_cagr')),
                    'revenue_growth_5y_cagr': to_float(metrics.get('revenue_growth_5y_cagr')),
                    'eps_growth_yoy': to_float(metrics.get('eps_growth_yoy')),
                    'eps_growth_3y_cagr': to_float(metrics.get('eps_growth_3y_cagr')),
                    'eps_growth_5y_cagr': to_float(metrics.get('eps_growth_5y_cagr')),
                    'fcf_growth_yoy': to_float(metrics.get('fcf_growth_yoy')),
                    'enterprise_value': to_float(metrics.get('enterprise_value')),
                    'ev_to_revenue': to_float(metrics.get('ev_to_revenue')),
                    'ev_to_ebitda': to_float(metrics.get('ev_to_ebitda')),
                    'ev_to_fcf': to_float(metrics.get('ev_to_fcf')),
                    'current_ratio': to_float(metrics.get('current_ratio')),
                    'quick_ratio': to_float(metrics.get('quick_ratio')),
                    'debt_to_equity': to_float(metrics.get('debt_to_equity')),
                    'interest_coverage': to_float(metrics.get('interest_coverage')),
                    'net_debt_to_ebitda': to_float(metrics.get('net_debt_to_ebitda')),
                    'dividend_yield': to_float(metrics.get('dividend_yield')),
                    'payout_ratio': to_float(metrics.get('payout_ratio')),
                    'dividend_growth_rate': to_float(metrics.get('dividend_growth_rate')),
                    'consecutive_dividend_years': metrics.get('consecutive_dividend_years', 0),
                    'dcf_fair_value': to_float(metrics.get('dcf_fair_value')),
                    'dcf_upside_percent': to_float(metrics.get('dcf_upside_percent')),
                    'graham_number': to_float(metrics.get('graham_number')),
                    'pe_sector_percentile': metrics.get('pe_sector_percentile'),
                    'pb_sector_percentile': metrics.get('pb_sector_percentile'),
                    'roe_sector_percentile': metrics.get('roe_sector_percentile'),
                    'margin_sector_percentile': metrics.get('margin_sector_percentile'),
                    'data_quality_score': to_float(metrics.get('data_quality_score')),
                }

                await session.execute(query, params)
                await session.commit()

                return True

        except Exception as e:
            logger.error(f"Error storing metrics for {metrics['ticker']}: {e}", exc_info=True)
            return False

    async def process_ticker(self, ticker: str) -> bool:
        """Process a single ticker to calculate all metrics.

        Args:
            ticker: Stock ticker symbol.

        Returns:
            True if successful, False otherwise.
        """
        try:
            # Calculate all metrics
            metrics = await self.calculate_all_metrics(ticker)

            if not metrics:
                logger.debug(f"{ticker}: Insufficient data")
                return False

            # Store in database
            success = await self.store_metrics(metrics)

            if success:
                logger.debug(
                    f"{ticker}: Calculated - ROE: {metrics.get('roe'):.1f}%, "
                    f"Rev Growth YoY: {metrics.get('revenue_growth_yoy')}%, "
                    f"Data Quality: {metrics.get('data_quality_score'):.0f}%"
                    if metrics.get('roe') and metrics.get('revenue_growth_yoy')
                    else f"{ticker}: Calculated with partial data"
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
        progress_bar = tqdm(total=len(tickers), desc="Calculating Fundamental Metrics")

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
        """Run the fundamental metrics calculation process.

        Args:
            limit: Limit number of tickers to process.
            ticker: Process single ticker.
            all_tickers: Process all tickers with sufficient data.
        """
        start_time = datetime.now()
        logger.info("=" * 80)
        logger.info("Fundamental Metrics Calculator (Phase 1-3)")
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
        logger.info("Calculation Complete")
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
        description='Calculate extended fundamental metrics (Phases 1-3)',
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  # Test on 10 tickers
  python fundamental_metrics_calculator.py --limit 10

  # Process all tickers
  python fundamental_metrics_calculator.py --all

  # Process single ticker
  python fundamental_metrics_calculator.py --ticker AAPL
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
        help='Process all tickers with TTM financial data'
    )

    args = parser.parse_args()

    # Validate arguments
    if args.ticker and (args.all or args.limit):
        parser.error("--ticker cannot be used with --all or --limit")

    # Run calculator
    calculator = FundamentalMetricsCalculator()
    asyncio.run(calculator.run(
        limit=args.limit,
        ticker=args.ticker,
        all_tickers=args.all
    ))


if __name__ == '__main__':
    main()
