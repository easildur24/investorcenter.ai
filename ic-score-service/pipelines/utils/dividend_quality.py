"""Dividend Quality Factor Calculator for IC Score v2.1.

This module calculates the optional Dividend Quality factor for
income-focused investors. It evaluates dividend sustainability,
growth, and yield relative to sector peers.

Components:
- Yield Score: Dividend yield vs sector peers (25%)
- Payout Score: Sustainable payout ratio (25%)
- Growth Score: Dividend growth history (25%)
- Streak Score: Consecutive years of dividend payments/increases (25%)
"""

import logging
from dataclasses import dataclass
from datetime import date
from typing import Dict, Optional, Any

from sqlalchemy import text

logger = logging.getLogger(__name__)


@dataclass
class DividendQualityResult:
    """Result of dividend quality calculation."""
    score: float
    yield_score: float
    payout_score: float
    growth_score: float
    streak_score: float
    metrics: Dict[str, Any]
    is_dividend_payer: bool


class DividendQualityCalculator:
    """Calculate optional Dividend Quality factor score.

    This factor is optional and only applies to dividend-paying stocks.
    When enabled (income mode), it adds 5% weight to the overall score.

    Components:
    - Yield: Dividend yield percentile within sector
    - Payout: Payout ratio sustainability (optimal 30-60%)
    - Growth: 5-year dividend CAGR
    - Streak: Consecutive years paying/increasing dividends

    Weight: 5% when enabled (optional factor)
    """

    WEIGHT = 0.05  # +5% when enabled

    # Minimum yield to be considered a dividend stock
    MIN_DIVIDEND_YIELD = 0.5  # 0.5%

    # Component weights
    YIELD_WEIGHT = 0.25
    PAYOUT_WEIGHT = 0.25
    GROWTH_WEIGHT = 0.25
    STREAK_WEIGHT = 0.25

    # Optimal payout ratio range
    PAYOUT_OPTIMAL_LOW = 30
    PAYOUT_OPTIMAL_HIGH = 60

    # Dividend streak thresholds
    DIVIDEND_KING_YEARS = 50
    DIVIDEND_ARISTOCRAT_YEARS = 25
    DIVIDEND_ACHIEVER_YEARS = 10

    def __init__(self, session, sector_calculator=None):
        """Initialize calculator with database session.

        Args:
            session: SQLAlchemy async session for database queries.
            sector_calculator: Optional SectorPercentileCalculator for
                sector-relative yield scoring.
        """
        self.session = session
        self.sector_calculator = sector_calculator

    async def fetch_dividend_data(self, ticker: str) -> Optional[Dict[str, Any]]:
        """Fetch dividend data for a ticker.

        Args:
            ticker: Stock ticker symbol.

        Returns:
            Dictionary with dividend data or None.
        """
        try:
            # Get most recent dividend data
            query = text("""
                SELECT
                    fiscal_year,
                    annual_dividend,
                    dividend_yield,
                    payout_ratio,
                    dividend_growth_yoy,
                    consecutive_years_paid,
                    consecutive_years_increased,
                    ex_dividend_date,
                    payment_date
                FROM dividend_history
                WHERE ticker = :ticker
                ORDER BY fiscal_year DESC
                LIMIT 5
            """)

            result = await self.session.execute(query, {"ticker": ticker})
            rows = result.fetchall()

            if not rows:
                return None

            latest = rows[0]

            # Calculate 5-year dividend CAGR if we have enough history
            dividend_cagr = None
            if len(rows) >= 5:
                current_div = float(rows[0][1]) if rows[0][1] else 0
                old_div = float(rows[4][1]) if rows[4][1] else 0
                if old_div > 0 and current_div > 0:
                    dividend_cagr = ((current_div / old_div) ** (1/4) - 1) * 100

            return {
                'fiscal_year': latest[0],
                'annual_dividend': float(latest[1]) if latest[1] else None,
                'dividend_yield': float(latest[2]) if latest[2] else None,
                'payout_ratio': float(latest[3]) if latest[3] else None,
                'dividend_growth_yoy': float(latest[4]) if latest[4] else None,
                'consecutive_years_paid': latest[5] or 0,
                'consecutive_years_increased': latest[6] or 0,
                'ex_dividend_date': latest[7],
                'payment_date': latest[8],
                'dividend_cagr_5y': dividend_cagr,
            }

        except Exception as e:
            logger.error(f"Error fetching dividend data for {ticker}: {e}")
            return None

    async def get_sector(self, ticker: str) -> Optional[str]:
        """Get sector for a ticker.

        Args:
            ticker: Stock ticker symbol.

        Returns:
            Sector name or None.
        """
        try:
            query = text("""
                SELECT sector FROM tickers WHERE symbol = :ticker
            """)
            result = await self.session.execute(query, {"ticker": ticker})
            row = result.fetchone()
            return row[0] if row else None
        except Exception as e:
            logger.error(f"Error fetching sector for {ticker}: {e}")
            return None

    def _calculate_yield_score(
        self,
        dividend_yield: float,
        sector_percentile: Optional[float]
    ) -> float:
        """Calculate yield score.

        Uses sector percentile if available, otherwise absolute scoring.

        Args:
            dividend_yield: Current dividend yield %.
            sector_percentile: Yield percentile within sector (0-100).

        Returns:
            Yield score 0-100.
        """
        if sector_percentile is not None:
            # Sector-relative: higher yield = higher percentile = higher score
            return sector_percentile

        # Absolute scoring fallback
        # 0% yield = 0, 2% = 50, 4%+ = 100
        if dividend_yield <= 0:
            return 0
        elif dividend_yield >= 4:
            return 100
        else:
            return min(100, dividend_yield * 25)

    def _calculate_payout_score(self, payout_ratio: Optional[float]) -> float:
        """Calculate payout ratio score.

        Optimal range: 30-60%
        - Too low (<20%): Company not sharing profits
        - Too high (>80%): Unsustainable, risk of cut

        Args:
            payout_ratio: Payout ratio percentage.

        Returns:
            Payout score 0-100.
        """
        if payout_ratio is None:
            return 50  # Neutral if unknown

        if payout_ratio < 0:
            return 0  # Negative earnings = risky

        if payout_ratio <= 20:
            # Too low - room to grow but not sharing
            return 40 + (payout_ratio / 20) * 20

        if payout_ratio <= self.PAYOUT_OPTIMAL_LOW:
            # Approaching optimal
            return 60 + ((payout_ratio - 20) / 10) * 20

        if payout_ratio <= self.PAYOUT_OPTIMAL_HIGH:
            # Optimal range
            return 100 - abs(payout_ratio - 45) / 15 * 20

        if payout_ratio <= 80:
            # Getting high but still sustainable
            return 80 - ((payout_ratio - 60) / 20) * 40

        # Very high - risky
        return max(0, 40 - (payout_ratio - 80) / 2)

    def _calculate_growth_score(
        self,
        dividend_cagr: Optional[float],
        dividend_growth_yoy: Optional[float]
    ) -> float:
        """Calculate dividend growth score.

        Args:
            dividend_cagr: 5-year dividend CAGR %.
            dividend_growth_yoy: Most recent YoY growth %.

        Returns:
            Growth score 0-100.
        """
        # Prefer CAGR if available
        growth = dividend_cagr if dividend_cagr is not None else dividend_growth_yoy

        if growth is None:
            return 50  # Neutral if unknown

        # Score: 0% growth = 50, +10% = 100, -10% = 0
        score = 50 + (growth / 10) * 50

        return max(0, min(100, score))

    def _calculate_streak_score(self, streak_years: int) -> float:
        """Calculate dividend streak score.

        Dividend Kings (50+ years): 100
        Dividend Aristocrats (25+ years): 90+
        Dividend Achievers (10+ years): 70+
        Others: proportional

        Args:
            streak_years: Consecutive years of dividend increases.

        Returns:
            Streak score 0-100.
        """
        if streak_years >= self.DIVIDEND_KING_YEARS:
            return 100  # Dividend King

        if streak_years >= self.DIVIDEND_ARISTOCRAT_YEARS:
            # Aristocrat tier: 90-100
            return 90 + (streak_years - 25) / 25 * 10

        if streak_years >= self.DIVIDEND_ACHIEVER_YEARS:
            # Achiever tier: 70-90
            return 70 + (streak_years - 10) / 15 * 20

        if streak_years >= 5:
            # Good track record: 50-70
            return 50 + (streak_years - 5) / 5 * 20

        # New or inconsistent: 0-50
        return streak_years * 10

    async def calculate(self, ticker: str) -> Optional[DividendQualityResult]:
        """Calculate complete Dividend Quality factor score.

        Args:
            ticker: Stock ticker symbol.

        Returns:
            DividendQualityResult with score and components, or None.
        """
        data = await self.fetch_dividend_data(ticker)

        if not data:
            logger.debug(f"{ticker}: No dividend data available")
            return DividendQualityResult(
                score=0,
                yield_score=0,
                payout_score=0,
                growth_score=0,
                streak_score=0,
                metrics={},
                is_dividend_payer=False
            )

        dividend_yield = data.get('dividend_yield')

        # Check if this is actually a dividend payer
        if not dividend_yield or dividend_yield < self.MIN_DIVIDEND_YIELD:
            return DividendQualityResult(
                score=0,
                yield_score=0,
                payout_score=0,
                growth_score=0,
                streak_score=0,
                metrics={'dividend_yield': dividend_yield},
                is_dividend_payer=False
            )

        # Get sector percentile for yield if available
        sector_percentile = None
        if self.sector_calculator:
            sector = await self.get_sector(ticker)
            if sector:
                sector_percentile = await self.sector_calculator.get_percentile(
                    sector, 'dividend_yield', dividend_yield
                )

        # Calculate component scores
        yield_score = self._calculate_yield_score(dividend_yield, sector_percentile)
        payout_score = self._calculate_payout_score(data.get('payout_ratio'))
        growth_score = self._calculate_growth_score(
            data.get('dividend_cagr_5y'),
            data.get('dividend_growth_yoy')
        )
        streak_score = self._calculate_streak_score(
            data.get('consecutive_years_increased', 0)
        )

        # Calculate weighted overall score
        overall_score = (
            yield_score * self.YIELD_WEIGHT +
            payout_score * self.PAYOUT_WEIGHT +
            growth_score * self.GROWTH_WEIGHT +
            streak_score * self.STREAK_WEIGHT
        )

        # Build metrics
        metrics = {
            'dividend_yield': dividend_yield,
            'payout_ratio': data.get('payout_ratio'),
            'dividend_cagr_5y': data.get('dividend_cagr_5y'),
            'dividend_growth_yoy': data.get('dividend_growth_yoy'),
            'consecutive_years_paid': data.get('consecutive_years_paid'),
            'consecutive_years_increased': data.get('consecutive_years_increased'),
            'ex_dividend_date': str(data.get('ex_dividend_date')) if data.get('ex_dividend_date') else None,
            'payment_date': str(data.get('payment_date')) if data.get('payment_date') else None,
        }

        # Determine dividend tier
        streak = data.get('consecutive_years_increased', 0)
        if streak >= self.DIVIDEND_KING_YEARS:
            metrics['dividend_tier'] = 'Dividend King'
        elif streak >= self.DIVIDEND_ARISTOCRAT_YEARS:
            metrics['dividend_tier'] = 'Dividend Aristocrat'
        elif streak >= self.DIVIDEND_ACHIEVER_YEARS:
            metrics['dividend_tier'] = 'Dividend Achiever'
        elif streak >= 5:
            metrics['dividend_tier'] = 'Dividend Payer'
        else:
            metrics['dividend_tier'] = 'New Dividend Payer'

        return DividendQualityResult(
            score=round(overall_score, 2),
            yield_score=round(yield_score, 2),
            payout_score=round(payout_score, 2),
            growth_score=round(growth_score, 2),
            streak_score=round(streak_score, 2),
            metrics=metrics,
            is_dividend_payer=True
        )
