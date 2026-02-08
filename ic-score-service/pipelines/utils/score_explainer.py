"""Score Explainer Service for IC Score v2.1 Phase 3.

This module generates human-readable explanations for IC Score changes
and provides granular confidence information about data availability.

Key features:
- Factor-level change breakdown
- Human-readable explanations
- Granular confidence metrics
- Data freshness tracking
"""

import logging
from dataclasses import dataclass, field
from datetime import date, datetime, timedelta
from typing import Dict, List, Optional, Any, Tuple

from sqlalchemy import text

logger = logging.getLogger(__name__)


@dataclass
class ScoreChangeReason:
    """A reason for a score change."""
    factor: str
    previous_score: Optional[float]
    current_score: Optional[float]
    delta: float
    weight: float
    contribution: float  # delta * weight
    explanation: str


@dataclass
class FactorDataStatus:
    """Data availability status for a factor."""
    available: bool
    freshness: str  # 'fresh', 'recent', 'stale', 'missing'
    freshness_days: Optional[int]
    count: Optional[int]  # Number of data points
    warning: Optional[str]
    reason: Optional[str]


@dataclass
class GranularConfidence:
    """Granular confidence breakdown."""
    level: str  # 'High', 'Medium', 'Low'
    percentage: float
    factors: Dict[str, FactorDataStatus]
    warnings: List[str]


@dataclass
class ScoreChangeExplanation:
    """Complete explanation of score change."""
    ticker: str
    previous_score: Optional[float]
    current_score: float
    delta: float
    reasons: List[ScoreChangeReason]
    summary: str
    confidence: GranularConfidence


class ScoreExplainer:
    """Service for explaining IC Score changes.

    Analyzes factor-level changes and generates human-readable
    explanations for why a stock's IC Score changed.
    """

    # Factor weights for v2.1 (only factors with database columns)
    FACTOR_WEIGHTS = {
        'value': 0.15,
        'growth': 0.15,
        'profitability': 0.15,
        'financial_health': 0.12,
        'momentum': 0.12,
        'technical': 0.10,
        'analyst_consensus': 0.06,  # Part of smart_money
        'insider_activity': 0.05,   # Part of smart_money
        'institutional': 0.05,      # Part of smart_money
        'news_sentiment': 0.05,
    }

    # Minimum delta to report as significant
    SIGNIFICANT_DELTA = 3.0

    # Explanation templates
    EXPLANATIONS = {
        'value': {
            'positive': "Stock appears more undervalued vs sector peers",
            'negative': "Stock appears more overvalued vs sector peers",
        },
        'growth': {
            'positive': "Revenue and earnings growth improved",
            'negative': "Growth metrics declined",
        },
        'profitability': {
            'positive': "Profitability margins improved",
            'negative': "Profitability margins contracted",
        },
        'financial_health': {
            'positive': "Balance sheet strength improved",
            'negative': "Debt or liquidity metrics worsened",
        },
        'momentum': {
            'positive': "Price momentum strengthened",
            'negative': "Price momentum weakened",
        },
        'technical': {
            'positive': "Technical indicators turned bullish",
            'negative': "Technical indicators turned bearish",
        },
        'analyst_consensus': {
            'positive': "Analyst ratings upgraded",
            'negative': "Analyst ratings downgraded",
        },
        'insider_activity': {
            'positive': "Insider buying increased",
            'negative': "Insider selling increased",
        },
        'institutional': {
            'positive': "Institutional ownership increased",
            'negative': "Institutional ownership decreased",
        },
        'news_sentiment': {
            'positive': "News sentiment improved",
            'negative': "News sentiment worsened",
        },
    }

    def __init__(self, session):
        """Initialize explainer with database session."""
        self.session = session

    async def explain_change(
        self,
        ticker: str,
        current_scores: Dict[str, Optional[float]],
        previous_scores: Optional[Dict[str, Optional[float]]] = None
    ) -> ScoreChangeExplanation:
        """Generate explanation for score change.

        Args:
            ticker: Stock ticker symbol.
            current_scores: Current factor scores dict.
            previous_scores: Previous factor scores dict (optional, will fetch).

        Returns:
            ScoreChangeExplanation with reasons and summary.
        """
        # Fetch previous scores if not provided
        if previous_scores is None:
            previous_scores = await self._get_previous_scores(ticker)

        current_overall = current_scores.get('overall_score', 0)
        previous_overall = previous_scores.get('overall_score') if previous_scores else None

        delta = current_overall - (previous_overall or current_overall)

        # Calculate factor-level changes
        reasons = []
        for factor, weight in self.FACTOR_WEIGHTS.items():
            current_factor = current_scores.get(f'{factor}_score')
            previous_factor = previous_scores.get(f'{factor}_score') if previous_scores else None

            if current_factor is None and previous_factor is None:
                continue

            # Handle None values
            curr_val = current_factor if current_factor is not None else 50
            prev_val = previous_factor if previous_factor is not None else 50

            factor_delta = curr_val - prev_val

            # Only report significant changes
            if abs(factor_delta) >= self.SIGNIFICANT_DELTA:
                explanation = self._get_explanation(factor, factor_delta)
                contribution = factor_delta * weight

                reasons.append(ScoreChangeReason(
                    factor=factor,
                    previous_score=previous_factor,
                    current_score=current_factor,
                    delta=round(factor_delta, 2),
                    weight=weight,
                    contribution=round(contribution, 2),
                    explanation=explanation
                ))

        # Sort by absolute contribution
        reasons.sort(key=lambda x: abs(x.contribution), reverse=True)

        # Generate summary
        summary = self._generate_summary(ticker, delta, reasons)

        # Get confidence
        confidence = await self.get_granular_confidence(ticker, current_scores)

        return ScoreChangeExplanation(
            ticker=ticker,
            previous_score=previous_overall,
            current_score=current_overall,
            delta=round(delta, 2),
            reasons=reasons[:5],  # Top 5 reasons
            summary=summary,
            confidence=confidence
        )

    def _get_explanation(self, factor: str, delta: float) -> str:
        """Get human-readable explanation for factor change."""
        templates = self.EXPLANATIONS.get(factor, {
            'positive': f"{factor.replace('_', ' ').title()} improved",
            'negative': f"{factor.replace('_', ' ').title()} declined"
        })

        if delta > 0:
            return templates['positive']
        else:
            return templates['negative']

    def _generate_summary(
        self,
        ticker: str,
        delta: float,
        reasons: List[ScoreChangeReason]
    ) -> str:
        """Generate summary of score change."""
        if not reasons:
            if abs(delta) < 0.5:
                return f"{ticker}'s IC Score is unchanged"
            elif delta > 0:
                return f"{ticker}'s IC Score improved slightly"
            else:
                return f"{ticker}'s IC Score declined slightly"

        # Get top contributor
        top_reason = reasons[0]

        if delta > 3:
            direction = "improved significantly"
        elif delta > 0:
            direction = "improved"
        elif delta < -3:
            direction = "declined significantly"
        else:
            direction = "declined"

        return (
            f"{ticker}'s IC Score {direction} ({delta:+.1f} points), "
            f"primarily due to {top_reason.factor.replace('_', ' ')} "
            f"({top_reason.delta:+.1f})"
        )

    async def _get_previous_scores(
        self,
        ticker: str
    ) -> Optional[Dict[str, Optional[float]]]:
        """Get previous day's scores from database."""
        try:
            query = text("""
                SELECT
                    overall_score,
                    value_score,
                    growth_score,
                    profitability_score,
                    financial_health_score,
                    momentum_score,
                    analyst_consensus_score,
                    insider_activity_score,
                    institutional_score,
                    news_sentiment_score,
                    technical_score
                FROM ic_scores
                WHERE ticker = :ticker
                  AND date < CURRENT_DATE
                ORDER BY date DESC
                LIMIT 1
            """)

            result = await self.session.execute(query, {"ticker": ticker})
            row = result.fetchone()

            if not row:
                return None

            return {
                'overall_score': float(row[0]) if row[0] else None,
                'value_score': float(row[1]) if row[1] else None,
                'growth_score': float(row[2]) if row[2] else None,
                'profitability_score': float(row[3]) if row[3] else None,
                'financial_health_score': float(row[4]) if row[4] else None,
                'momentum_score': float(row[5]) if row[5] else None,
                'analyst_consensus_score': float(row[6]) if row[6] else None,
                'insider_activity_score': float(row[7]) if row[7] else None,
                'institutional_score': float(row[8]) if row[8] else None,
                'news_sentiment_score': float(row[9]) if row[9] else None,
                'technical_score': float(row[10]) if row[10] else None,
            }

        except Exception as e:
            logger.error(f"Error fetching previous scores for {ticker}: {e}")
            return None

    async def get_granular_confidence(
        self,
        ticker: str,
        current_scores: Dict[str, Optional[float]]
    ) -> GranularConfidence:
        """Get granular confidence breakdown.

        Args:
            ticker: Stock ticker symbol.
            current_scores: Current factor scores dict.

        Returns:
            GranularConfidence with per-factor status.
        """
        factors = {}
        warnings = []
        available_count = 0
        total_factors = len(self.FACTOR_WEIGHTS)

        for factor in self.FACTOR_WEIGHTS.keys():
            score_key = f'{factor}_score'
            score = current_scores.get(score_key)

            if score is not None:
                available_count += 1
                freshness = await self._get_factor_freshness(ticker, factor)
                factors[factor] = FactorDataStatus(
                    available=True,
                    freshness=freshness['status'],
                    freshness_days=freshness['days'],
                    count=freshness.get('count'),
                    warning=freshness.get('warning'),
                    reason=None
                )

                if freshness.get('warning'):
                    warnings.append(f"{factor}: {freshness['warning']}")
            else:
                factors[factor] = FactorDataStatus(
                    available=False,
                    freshness='missing',
                    freshness_days=None,
                    count=0,
                    warning=None,
                    reason=self._get_missing_reason(factor)
                )

        # Calculate overall confidence
        percentage = (available_count / total_factors) * 100

        if percentage >= 90:
            level = 'High'
        elif percentage >= 70:
            level = 'Medium'
        else:
            level = 'Low'

        return GranularConfidence(
            level=level,
            percentage=round(percentage, 1),
            factors=factors,
            warnings=warnings
        )

    async def _get_factor_freshness(
        self,
        ticker: str,
        factor: str
    ) -> Dict[str, Any]:
        """Check data freshness for a factor."""
        # Map factors to their data sources
        source_queries = {
            'value': ("valuation_ratios", "calculation_date"),
            'growth': ("fundamental_metrics_extended", "calculation_date"),
            'profitability': ("financials", "period_end_date"),
            'financial_health': ("financials", "period_end_date"),
            'momentum': ("technical_indicators", "time"),
            'technical': ("technical_indicators", "time"),
            'analyst_consensus': ("analyst_ratings", "rating_date"),
            'insider_activity': ("insider_trades", "transaction_date"),
            'institutional': ("institutional_holdings", "filing_date"),
            'news_sentiment': ("news_articles", "published_at"),
        }

        if factor not in source_queries:
            return {'status': 'unknown', 'days': None}

        table, date_col = source_queries[factor]

        try:
            query = text(f"""
                SELECT MAX({date_col}) as latest_date, COUNT(*) as count
                FROM {table}
                WHERE ticker = :ticker
            """)

            result = await self.session.execute(query, {"ticker": ticker})
            row = result.fetchone()

            if not row or not row[0]:
                return {'status': 'missing', 'days': None, 'count': 0}

            latest_date = row[0]
            if hasattr(latest_date, 'date'):
                latest_date = latest_date.date()
            elif isinstance(latest_date, datetime):
                latest_date = latest_date.date()

            days_old = (date.today() - latest_date).days
            count = row[1]

            if days_old <= 1:
                status = 'fresh'
                warning = None
            elif days_old <= 7:
                status = 'recent'
                warning = None
            elif days_old <= 30:
                status = 'stale'
                warning = f"Data is {days_old} days old"
            else:
                status = 'stale'
                warning = f"Data is {days_old} days old (may be outdated)"

            return {
                'status': status,
                'days': days_old,
                'count': count,
                'warning': warning
            }

        except Exception as e:
            logger.debug(f"Error checking freshness for {factor}: {e}")
            return {'status': 'unknown', 'days': None}

    def _get_missing_reason(self, factor: str) -> str:
        """Get reason why a factor might be missing."""
        reasons = {
            'value': "Missing valuation data (P/E, P/B, P/S)",
            'growth': "Insufficient historical financial data",
            'profitability': "Missing profitability metrics",
            'financial_health': "Missing balance sheet data",
            'momentum': "Missing price data",
            'technical': "Missing technical indicators",
            'analyst_consensus': "No analyst coverage",
            'insider_activity': "No insider trades reported",
            'institutional': "No institutional holdings data",
            'news_sentiment': "No recent news articles",
        }
        return reasons.get(factor, "Data not available")

    async def store_score_change(
        self,
        ticker: str,
        explanation: ScoreChangeExplanation,
        trigger_events: Optional[List[str]] = None,
        smoothing_applied: bool = False
    ) -> bool:
        """Store score change record in database."""
        try:
            factor_changes = [
                {
                    'factor': r.factor,
                    'delta': r.delta,
                    'contribution': r.contribution,
                    'explanation': r.explanation
                }
                for r in explanation.reasons
            ]

            query = text("""
                INSERT INTO ic_score_changes (
                    ticker, calculated_at,
                    previous_score, current_score, delta,
                    factor_changes, trigger_events, smoothing_applied
                ) VALUES (
                    :ticker, CURRENT_DATE,
                    :previous_score, :current_score, :delta,
                    :factor_changes, :trigger_events, :smoothing_applied
                )
            """)

            await self.session.execute(query, {
                "ticker": ticker,
                "previous_score": explanation.previous_score,
                "current_score": explanation.current_score,
                "delta": explanation.delta,
                "factor_changes": factor_changes,
                "trigger_events": trigger_events or [],
                "smoothing_applied": smoothing_applied
            })

            return True

        except Exception as e:
            logger.error(f"Error storing score change for {ticker}: {e}")
            return False
