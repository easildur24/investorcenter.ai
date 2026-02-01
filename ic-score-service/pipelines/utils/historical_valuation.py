"""Historical Valuation Factor Calculator for IC Score v2.1.

This module calculates the Historical Valuation factor, which compares
current valuation metrics to the company's own historical range.

Key insight: A stock trading at the low end of its own historical
valuation range may be undervalued relative to its typical pricing.
"""

import logging
from dataclasses import dataclass
from datetime import date, timedelta
from typing import Dict, List, Optional, Any

import numpy as np
from sqlalchemy import text

logger = logging.getLogger(__name__)


@dataclass
class HistoricalValuationResult:
    """Result of historical valuation calculation."""
    score: float
    pe_percentile: Optional[float]
    ps_percentile: Optional[float]
    metrics: Dict[str, Any]


class HistoricalValuationCalculator:
    """Calculate Historical Valuation factor score.

    Compares current valuation ratios to the company's 5-year history:
    - Trading at 5-year lows = potentially undervalued (high score)
    - Trading at 5-year highs = potentially overvalued (low score)

    For growth companies (low/negative margins), P/S is weighted more heavily.

    Weight: 8% of total IC Score
    """

    WEIGHT = 0.08  # 8% of total score
    HISTORY_YEARS = 5

    # Growth company threshold (net margin below this uses more P/S weight)
    GROWTH_MARGIN_THRESHOLD = 5.0

    # Weight split for growth vs mature companies
    PE_WEIGHT_MATURE = 0.70
    PS_WEIGHT_MATURE = 0.30
    PE_WEIGHT_GROWTH = 0.30
    PS_WEIGHT_GROWTH = 0.70

    def __init__(self, session):
        """Initialize calculator with database session.

        Args:
            session: SQLAlchemy async session for database queries.
        """
        self.session = session

    async def get_current_valuation(self, ticker: str) -> Optional[Dict[str, Any]]:
        """Get current valuation ratios.

        Args:
            ticker: Stock ticker symbol.

        Returns:
            Dictionary with current P/E, P/S, etc. or None.
        """
        try:
            query = text("""
                SELECT
                    ttm_pe_ratio,
                    ttm_ps_ratio,
                    ttm_pb_ratio,
                    stock_price,
                    calculation_date
                FROM valuation_ratios
                WHERE ticker = :ticker
                ORDER BY calculation_date DESC
                LIMIT 1
            """)

            result = await self.session.execute(query, {"ticker": ticker})
            row = result.fetchone()

            if not row:
                return None

            return {
                'pe_ratio': float(row[0]) if row[0] and row[0] > 0 else None,
                'ps_ratio': float(row[1]) if row[1] and row[1] > 0 else None,
                'pb_ratio': float(row[2]) if row[2] and row[2] > 0 else None,
                'stock_price': float(row[3]) if row[3] else None,
                'calculation_date': row[4],
            }

        except Exception as e:
            logger.error(f"Error fetching current valuation for {ticker}: {e}")
            return None

    async def get_valuation_history(
        self,
        ticker: str,
        metric: str,
        years: int = 5
    ) -> Optional[List[float]]:
        """Get historical valuation data points.

        Args:
            ticker: Stock ticker symbol.
            metric: Metric name (pe_ratio, ps_ratio, etc.).
            years: Number of years of history.

        Returns:
            List of historical values or None.
        """
        try:
            cutoff_date = date.today() - timedelta(days=years * 365)

            query = text(f"""
                SELECT {metric}
                FROM valuation_history
                WHERE ticker = :ticker
                  AND snapshot_date >= :cutoff_date
                  AND {metric} IS NOT NULL
                  AND {metric} > 0
                ORDER BY snapshot_date
            """)

            result = await self.session.execute(
                query,
                {"ticker": ticker, "cutoff_date": cutoff_date}
            )
            rows = result.fetchall()

            if not rows or len(rows) < 12:  # Need at least 12 months
                return None

            return [float(row[0]) for row in rows]

        except Exception as e:
            logger.error(f"Error fetching {metric} history for {ticker}: {e}")
            return None

    async def get_net_margin(self, ticker: str) -> Optional[float]:
        """Get current net margin to determine company type.

        Args:
            ticker: Stock ticker symbol.

        Returns:
            Net margin percentage or None.
        """
        try:
            query = text("""
                SELECT net_margin
                FROM financials
                WHERE ticker = :ticker
                ORDER BY period_end_date DESC
                LIMIT 1
            """)

            result = await self.session.execute(query, {"ticker": ticker})
            row = result.fetchone()

            if row and row[0] is not None:
                return float(row[0])

            return None

        except Exception as e:
            logger.error(f"Error fetching net margin for {ticker}: {e}")
            return None

    def _percentile_in_history(
        self,
        current_value: float,
        history: List[float]
    ) -> float:
        """Calculate where current value sits in historical distribution.

        Args:
            current_value: Current metric value.
            history: List of historical values.

        Returns:
            Percentile 0-100 (0 = lowest historically, 100 = highest).
        """
        if not history:
            return 50  # Neutral if no history

        # Count how many historical values are below current
        below_count = sum(1 for v in history if v < current_value)

        # Calculate percentile
        percentile = (below_count / len(history)) * 100

        return percentile

    async def calculate(self, ticker: str) -> Optional[HistoricalValuationResult]:
        """Calculate complete Historical Valuation factor score.

        Args:
            ticker: Stock ticker symbol.

        Returns:
            HistoricalValuationResult with score and metrics, or None.
        """
        # Get current valuation
        current = await self.get_current_valuation(ticker)
        if not current:
            logger.debug(f"{ticker}: No current valuation data")
            return None

        current_pe = current.get('pe_ratio')
        current_ps = current.get('ps_ratio')

        if current_pe is None and current_ps is None:
            logger.debug(f"{ticker}: No P/E or P/S ratio available")
            return None

        # Get historical data
        pe_history = None
        ps_history = None

        if current_pe:
            pe_history = await self.get_valuation_history(ticker, 'pe_ratio')

        if current_ps:
            ps_history = await self.get_valuation_history(ticker, 'ps_ratio')

        if not pe_history and not ps_history:
            logger.debug(f"{ticker}: Insufficient historical valuation data")
            return None

        # Calculate percentiles
        pe_percentile = None
        ps_percentile = None
        pe_score = None
        ps_score = None

        if pe_history and current_pe:
            pe_percentile = self._percentile_in_history(current_pe, pe_history)
            # Lower percentile = cheaper = higher score
            pe_score = 100 - pe_percentile

        if ps_history and current_ps:
            ps_percentile = self._percentile_in_history(current_ps, ps_history)
            # Lower percentile = cheaper = higher score
            ps_score = 100 - ps_percentile

        # Determine weights based on company type
        net_margin = await self.get_net_margin(ticker)
        is_growth = net_margin is not None and net_margin < self.GROWTH_MARGIN_THRESHOLD

        # Build metrics
        metrics = {
            'current_pe': current_pe,
            'current_ps': current_ps,
            'is_growth_company': is_growth,
            'net_margin': net_margin,
        }

        if pe_history:
            metrics['pe_5y_low'] = round(min(pe_history), 2)
            metrics['pe_5y_high'] = round(max(pe_history), 2)
            metrics['pe_5y_median'] = round(float(np.median(pe_history)), 2)
            metrics['pe_data_points'] = len(pe_history)

        if ps_history:
            metrics['ps_5y_low'] = round(min(ps_history), 2)
            metrics['ps_5y_high'] = round(max(ps_history), 2)
            metrics['ps_5y_median'] = round(float(np.median(ps_history)), 2)
            metrics['ps_data_points'] = len(ps_history)

        # Calculate overall score
        if pe_score is not None and ps_score is not None:
            if is_growth:
                # Growth company: weight P/S more
                overall_score = (
                    pe_score * self.PE_WEIGHT_GROWTH +
                    ps_score * self.PS_WEIGHT_GROWTH
                )
            else:
                # Mature company: weight P/E more
                overall_score = (
                    pe_score * self.PE_WEIGHT_MATURE +
                    ps_score * self.PS_WEIGHT_MATURE
                )
        elif pe_score is not None:
            overall_score = pe_score
        elif ps_score is not None:
            overall_score = ps_score
        else:
            return None

        return HistoricalValuationResult(
            score=round(overall_score, 2),
            pe_percentile=round(pe_percentile, 2) if pe_percentile is not None else None,
            ps_percentile=round(ps_percentile, 2) if ps_percentile is not None else None,
            metrics=metrics
        )
