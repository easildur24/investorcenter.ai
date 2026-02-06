"""Earnings Revisions Factor Calculator for IC Score v2.1.

This module calculates the Earnings Revisions factor, which measures
analyst estimate revisions to assess market sentiment and earnings momentum.

Components:
- Magnitude: How much have EPS estimates changed? (50% weight)
- Breadth: What percentage of analysts are raising estimates? (30% weight)
- Recency: Are recent revisions more positive than older ones? (20% weight)
"""

import logging
from dataclasses import dataclass
from datetime import date, timedelta
from typing import Dict, Optional, Any

from sqlalchemy import text

logger = logging.getLogger(__name__)


@dataclass
class EarningsRevisionsResult:
    """Result of earnings revisions calculation."""
    score: float
    magnitude_score: float
    breadth_score: float
    recency_score: float
    metrics: Dict[str, Any]


class EarningsRevisionsCalculator:
    """Calculate Earnings Revisions factor score.

    The Earnings Revisions factor measures the trend in analyst EPS estimates:
    - Rising estimates indicate improving business outlook
    - Falling estimates suggest deteriorating fundamentals
    - Breadth and recency help assess conviction

    Weight: 8% of total IC Score
    """

    WEIGHT = 0.08  # 8% of total score

    # Component weights
    MAGNITUDE_WEIGHT = 0.50
    BREADTH_WEIGHT = 0.30
    RECENCY_WEIGHT = 0.20

    # Scoring parameters
    MAGNITUDE_MAX_CHANGE = 0.15  # +/- 15% change = max/min score

    def __init__(self, session):
        """Initialize calculator with database session.

        Args:
            session: SQLAlchemy async session for database queries.
        """
        self.session = session

    async def fetch_eps_estimates(self, ticker: str) -> Optional[Dict[str, Any]]:
        """Fetch EPS estimate data for a ticker.

        Args:
            ticker: Stock ticker symbol.

        Returns:
            Dictionary with estimate data or None if not available.
        """
        try:
            # Get current fiscal year and next year estimates
            query = text("""
                SELECT
                    fiscal_year,
                    fiscal_quarter,
                    consensus_eps,
                    num_analysts,
                    high_estimate,
                    low_estimate,
                    estimate_30d_ago,
                    estimate_60d_ago,
                    estimate_90d_ago,
                    upgrades_30d,
                    downgrades_30d,
                    upgrades_60d,
                    downgrades_60d,
                    upgrades_90d,
                    downgrades_90d,
                    revision_pct_30d,
                    revision_pct_60d,
                    revision_pct_90d,
                    fetched_at
                FROM eps_estimates
                WHERE ticker = :ticker
                  AND fiscal_quarter IS NULL  -- Annual estimates
                ORDER BY fiscal_year DESC
                LIMIT 2
            """)

            result = await self.session.execute(query, {"ticker": ticker})
            rows = result.fetchall()

            if not rows:
                return None

            # Use the next fiscal year estimates (most relevant)
            current_year = date.today().year
            target_row = None

            for row in rows:
                if row[0] >= current_year:
                    target_row = row
                    break

            if not target_row:
                target_row = rows[0]

            return {
                'fiscal_year': target_row[0],
                'fiscal_quarter': target_row[1],
                'consensus_eps': float(target_row[2]) if target_row[2] else None,
                'num_analysts': target_row[3],
                'high_estimate': float(target_row[4]) if target_row[4] else None,
                'low_estimate': float(target_row[5]) if target_row[5] else None,
                'estimate_30d_ago': float(target_row[6]) if target_row[6] else None,
                'estimate_60d_ago': float(target_row[7]) if target_row[7] else None,
                'estimate_90d_ago': float(target_row[8]) if target_row[8] else None,
                'upgrades_30d': target_row[9] or 0,
                'downgrades_30d': target_row[10] or 0,
                'upgrades_60d': target_row[11] or 0,
                'downgrades_60d': target_row[12] or 0,
                'upgrades_90d': target_row[13] or 0,
                'downgrades_90d': target_row[14] or 0,
                'revision_pct_30d': float(target_row[15]) if target_row[15] else None,
                'revision_pct_60d': float(target_row[16]) if target_row[16] else None,
                'revision_pct_90d': float(target_row[17]) if target_row[17] else None,
            }

        except Exception as e:
            logger.error(f"Error fetching EPS estimates for {ticker}: {e}")
            return None

    def _calculate_magnitude_score(self, data: Dict[str, Any]) -> float:
        """Calculate magnitude score based on % change in consensus EPS.

        Higher score = more positive estimate revisions.

        Scale:
        - -15% change = 0 (very negative)
        - 0% change = 50 (neutral)
        - +15% change = 100 (very positive)

        Args:
            data: EPS estimate data dictionary.

        Returns:
            Magnitude score 0-100.
        """
        current = data.get('consensus_eps')

        # Try different time horizons (prefer longer for stability)
        prior = data.get('estimate_90d_ago') or data.get('estimate_60d_ago') or data.get('estimate_30d_ago')

        if not current or not prior or prior == 0:
            return 50  # Neutral if no data

        change_pct = (current - prior) / abs(prior)

        # Scale: -15% = 0, 0% = 50, +15% = 100
        score = 50 + (change_pct / self.MAGNITUDE_MAX_CHANGE) * 50

        return max(0, min(100, score))

    def _calculate_breadth_score(self, data: Dict[str, Any]) -> float:
        """Calculate breadth score based on ratio of upgrades vs downgrades.

        Higher score = more analysts raising estimates.

        Scale:
        - All downgrades = 0
        - Equal up/down = 50
        - All upgrades = 100

        Args:
            data: EPS estimate data dictionary.

        Returns:
            Breadth score 0-100.
        """
        # Use 90-day data for better sample size
        upgrades = data.get('upgrades_90d', 0)
        downgrades = data.get('downgrades_90d', 0)

        # Fall back to shorter periods if no data
        if upgrades == 0 and downgrades == 0:
            upgrades = data.get('upgrades_60d', 0)
            downgrades = data.get('downgrades_60d', 0)

        if upgrades == 0 and downgrades == 0:
            upgrades = data.get('upgrades_30d', 0)
            downgrades = data.get('downgrades_30d', 0)

        total_revisions = upgrades + downgrades

        if total_revisions == 0:
            return 50  # Neutral if no revisions

        # Calculate upgrade ratio
        upgrade_ratio = upgrades / total_revisions

        # Scale: 0% upgrades = 0, 50% = 50, 100% = 100
        score = upgrade_ratio * 100

        return max(0, min(100, score))

    def _calculate_recency_score(self, data: Dict[str, Any]) -> float:
        """Calculate recency score to detect acceleration in revisions.

        Higher score = more recent revisions are more positive.

        Compares 30-day revision trend to 90-day trend:
        - Recent more positive than longer-term = bullish acceleration
        - Recent more negative than longer-term = bearish momentum

        Args:
            data: EPS estimate data dictionary.

        Returns:
            Recency score 0-100.
        """
        rev_30d = data.get('revision_pct_30d')
        rev_90d = data.get('revision_pct_90d')

        if rev_30d is None and rev_90d is None:
            return 50  # Neutral if no data

        if rev_30d is None:
            rev_30d = 0
        if rev_90d is None:
            rev_90d = 0

        # Calculate acceleration (positive = improving trend)
        # If 30d revision is better than 90d, that's positive
        acceleration = rev_30d - rev_90d

        # Scale: -10% acceleration = 0, 0% = 50, +10% = 100
        score = 50 + (acceleration / 0.10) * 50

        return max(0, min(100, score))

    async def calculate(self, ticker: str) -> Optional[EarningsRevisionsResult]:
        """Calculate complete Earnings Revisions factor score.

        Args:
            ticker: Stock ticker symbol.

        Returns:
            EarningsRevisionsResult with score and components, or None.
        """
        data = await self.fetch_eps_estimates(ticker)

        if not data or data.get('consensus_eps') is None:
            logger.debug(f"{ticker}: No EPS estimate data available")
            return None

        # Calculate component scores
        magnitude_score = self._calculate_magnitude_score(data)
        breadth_score = self._calculate_breadth_score(data)
        recency_score = self._calculate_recency_score(data)

        # Calculate weighted overall score
        overall_score = (
            magnitude_score * self.MAGNITUDE_WEIGHT +
            breadth_score * self.BREADTH_WEIGHT +
            recency_score * self.RECENCY_WEIGHT
        )

        # Build metrics for transparency
        metrics = {
            'consensus_eps': data.get('consensus_eps'),
            'num_analysts': data.get('num_analysts'),
            'revision_pct_90d': data.get('revision_pct_90d'),
            'revision_pct_30d': data.get('revision_pct_30d'),
            'upgrades_90d': data.get('upgrades_90d'),
            'downgrades_90d': data.get('downgrades_90d'),
            'estimate_spread': None,
        }

        # Calculate estimate spread (high - low) as % of consensus
        high = data.get('high_estimate')
        low = data.get('low_estimate')
        consensus = data.get('consensus_eps')

        if high and low and consensus and consensus != 0:
            metrics['estimate_spread'] = round((high - low) / abs(consensus) * 100, 2)

        return EarningsRevisionsResult(
            score=round(overall_score, 2),
            magnitude_score=round(magnitude_score, 2),
            breadth_score=round(breadth_score, 2),
            recency_score=round(recency_score, 2),
            metrics=metrics
        )
