"""Company lifecycle classification for IC Score v2.1.

This module provides utilities for classifying companies into lifecycle stages
and adjusting IC Score factor weights accordingly.
"""
from enum import Enum
from typing import Dict, Optional, Any
from dataclasses import dataclass
from decimal import Decimal
import logging
from sqlalchemy import text
from sqlalchemy.ext.asyncio import AsyncSession

logger = logging.getLogger(__name__)


class LifecycleStage(Enum):
    """Company lifecycle stages for weight adjustment."""
    HYPERGROWTH = "hypergrowth"  # >50% revenue growth, often pre-profit
    GROWTH = "growth"            # 20-50% revenue growth, scaling
    MATURE = "mature"            # Stable, consistent profitability
    VALUE = "value"              # Low PE, high margins, slow/no growth
    TURNAROUND = "turnaround"    # Declining revenue, restructuring


@dataclass
class ClassificationResult:
    """Result of lifecycle classification."""
    stage: LifecycleStage
    confidence: float  # 0-1 confidence in classification
    metrics_used: Dict[str, float]
    adjusted_weights: Dict[str, float]


class LifecycleClassifier:
    """Classify companies into lifecycle stages for weight adjustment.

    Different company types require different evaluation criteria:
    - Hypergrowth companies: Focus on growth metrics, discount profitability
    - Growth companies: Balance growth and emerging profitability
    - Mature companies: Focus on profitability, cash flow, stability
    - Value companies: Focus on valuation, dividend, capital return
    - Turnaround companies: Focus on financial health, momentum signals
    """

    # Base weights for IC Score factors (v2.1)
    BASE_WEIGHTS: Dict[str, float] = {
        # Quality (35%)
        'profitability': 0.12,
        'financial_health': 0.10,
        'growth': 0.13,
        # Valuation (30%)
        'value': 0.12,
        'intrinsic_value': 0.10,
        'historical_value': 0.08,
        # Signals (35%)
        'momentum': 0.10,
        'smart_money': 0.10,
        'earnings_revisions': 0.08,
        'technical': 0.07,
    }

    # Weight multipliers for each lifecycle stage
    WEIGHT_ADJUSTMENTS: Dict[LifecycleStage, Dict[str, float]] = {
        LifecycleStage.HYPERGROWTH: {
            'growth': 1.5,           # +50% weight on growth
            'momentum': 1.3,         # +30% momentum
            'earnings_revisions': 1.3,
            'profitability': 0.5,    # -50% profitability (often negative)
            'value': 0.4,            # -60% value (usually expensive)
            'intrinsic_value': 0.4,
            'historical_value': 0.4,
            'financial_health': 0.8,
        },
        LifecycleStage.GROWTH: {
            'growth': 1.3,           # +30% growth
            'momentum': 1.2,
            'earnings_revisions': 1.2,
            'profitability': 0.8,
            'value': 0.7,
            'intrinsic_value': 0.8,
            'historical_value': 0.8,
        },
        LifecycleStage.MATURE: {
            'profitability': 1.2,    # +20% profitability
            'financial_health': 1.2,
            'value': 1.1,
            'intrinsic_value': 1.1,
            'historical_value': 1.2,
            'growth': 0.7,           # -30% growth (not the focus)
            'momentum': 0.9,
        },
        LifecycleStage.VALUE: {
            'value': 1.4,            # +40% value metrics
            'intrinsic_value': 1.3,
            'historical_value': 1.3,
            'profitability': 1.2,
            'financial_health': 1.1,
            'growth': 0.5,           # -50% growth
            'momentum': 0.8,
        },
        LifecycleStage.TURNAROUND: {
            'financial_health': 1.4,  # +40% financial health (survival)
            'momentum': 1.3,          # +30% momentum (recovery signals)
            'smart_money': 1.3,       # Smart money spotting recovery
            'value': 1.2,
            'growth': 0.6,
            'profitability': 0.7,
        },
    }

    # Classification thresholds
    THRESHOLDS = {
        'hypergrowth_revenue_growth': 50.0,  # >50% YoY
        'growth_revenue_growth': 20.0,       # >20% YoY
        'turnaround_revenue_growth': -5.0,   # <-5% YoY
        'value_pe_threshold': 12.0,          # P/E < 12
        'value_margin_threshold': 5.0,       # Net margin > 5%
    }

    def __init__(self, session: Optional[AsyncSession] = None):
        """Initialize classifier with optional database session.

        Args:
            session: Async SQLAlchemy session for database persistence
        """
        self.session = session

    def classify(self, data: Dict[str, Any]) -> ClassificationResult:
        """Classify company based on financial metrics.

        Args:
            data: Dict containing:
                - revenue_growth_yoy: Year-over-year revenue growth %
                - net_margin: Net profit margin %
                - pe_ratio: Price-to-earnings ratio
                - market_cap: Market capitalization (optional)
                - earnings_growth_yoy: YoY earnings growth % (optional)

        Returns:
            ClassificationResult with stage, confidence, and adjusted weights
        """
        revenue_growth = data.get('revenue_growth_yoy') or 0
        net_margin = data.get('net_margin') or 0
        pe_ratio = data.get('pe_ratio') or 20
        earnings_growth = data.get('earnings_growth_yoy') or 0

        # Ensure numeric types
        revenue_growth = float(revenue_growth) if revenue_growth else 0
        net_margin = float(net_margin) if net_margin else 0
        pe_ratio = float(pe_ratio) if pe_ratio else 20
        earnings_growth = float(earnings_growth) if earnings_growth else 0

        # Classification logic with confidence scoring
        stage, confidence = self._determine_stage(
            revenue_growth, net_margin, pe_ratio, earnings_growth
        )

        # Calculate adjusted weights
        adjusted_weights = self.adjust_weights(self.BASE_WEIGHTS, stage)

        return ClassificationResult(
            stage=stage,
            confidence=confidence,
            metrics_used={
                'revenue_growth_yoy': revenue_growth,
                'net_margin': net_margin,
                'pe_ratio': pe_ratio,
                'earnings_growth_yoy': earnings_growth,
            },
            adjusted_weights=adjusted_weights,
        )

    def _determine_stage(
        self,
        revenue_growth: float,
        net_margin: float,
        pe_ratio: float,
        earnings_growth: float
    ) -> tuple[LifecycleStage, float]:
        """Determine lifecycle stage with confidence score.

        Returns:
            Tuple of (LifecycleStage, confidence 0-1)
        """
        t = self.THRESHOLDS

        # Hypergrowth: >50% revenue growth
        if revenue_growth > t['hypergrowth_revenue_growth']:
            confidence = min(1.0, (revenue_growth - 50) / 50 + 0.7)
            return LifecycleStage.HYPERGROWTH, confidence

        # Growth: 20-50% revenue growth
        if revenue_growth > t['growth_revenue_growth']:
            confidence = 0.6 + (revenue_growth - 20) / 60
            return LifecycleStage.GROWTH, confidence

        # Turnaround: negative revenue growth
        if revenue_growth < t['turnaround_revenue_growth']:
            confidence = min(1.0, abs(revenue_growth) / 20 + 0.5)
            return LifecycleStage.TURNAROUND, confidence

        # Value: low P/E with positive margins
        if pe_ratio < t['value_pe_threshold'] and net_margin > t['value_margin_threshold']:
            # Higher confidence if very low P/E and high margins
            pe_score = (12 - pe_ratio) / 12
            margin_score = min(net_margin / 20, 0.5)
            confidence = 0.5 + pe_score * 0.3 + margin_score
            return LifecycleStage.VALUE, min(1.0, confidence)

        # Default: Mature
        # Confidence based on how "typical" the metrics are
        confidence = 0.6
        if 0 < revenue_growth < 15 and net_margin > 0:
            confidence = 0.8
        return LifecycleStage.MATURE, confidence

    def adjust_weights(
        self,
        base_weights: Dict[str, float],
        stage: LifecycleStage
    ) -> Dict[str, float]:
        """Adjust factor weights based on lifecycle stage.

        Applies multipliers and normalizes to sum to 1.0.

        Args:
            base_weights: Base factor weights
            stage: Company lifecycle stage

        Returns:
            Adjusted weights normalized to sum to 1.0
        """
        adjustments = self.WEIGHT_ADJUSTMENTS.get(stage, {})

        adjusted = {}
        for factor, weight in base_weights.items():
            multiplier = adjustments.get(factor, 1.0)
            adjusted[factor] = weight * multiplier

        # Normalize to sum to 1.0
        total = sum(adjusted.values())
        if total > 0:
            adjusted = {k: v / total for k, v in adjusted.items()}

        return adjusted

    async def classify_and_store(
        self,
        ticker: str,
        data: Dict[str, Any]
    ) -> ClassificationResult:
        """Classify company and store result in database.

        Args:
            ticker: Stock ticker symbol
            data: Financial metrics for classification

        Returns:
            ClassificationResult
        """
        result = self.classify(data)

        # Only store if ticker fits in VARCHAR(10) column
        if self.session and len(ticker) <= 10:
            await self._store_classification(ticker, result, data)

        return result

    def _clamp_numeric(self, value: Any, min_val: float = -999999, max_val: float = 999999) -> Optional[float]:
        """Clamp numeric value to valid range for NUMERIC(10,4) field.

        Args:
            value: Value to clamp
            min_val: Minimum allowed value
            max_val: Maximum allowed value

        Returns:
            Clamped value or None if input is None/invalid
        """
        if value is None:
            return None
        try:
            float_val = float(value)
            if not (-1e10 < float_val < 1e10):  # Filter extreme outliers
                return None
            return max(min_val, min(max_val, float_val))
        except (ValueError, TypeError):
            return None

    async def _store_classification(
        self,
        ticker: str,
        result: ClassificationResult,
        data: Dict[str, Any]
    ):
        """Store classification result in database."""
        query = text("""
            INSERT INTO lifecycle_classifications (
                ticker, lifecycle_stage,
                revenue_growth_yoy, net_margin, pe_ratio, market_cap,
                weights_applied
            ) VALUES (
                :ticker, :stage,
                :revenue_growth, :net_margin, :pe_ratio, :market_cap,
                :weights
            )
            ON CONFLICT (ticker, classified_at)
            DO UPDATE SET
                lifecycle_stage = EXCLUDED.lifecycle_stage,
                revenue_growth_yoy = EXCLUDED.revenue_growth_yoy,
                net_margin = EXCLUDED.net_margin,
                pe_ratio = EXCLUDED.pe_ratio,
                market_cap = EXCLUDED.market_cap,
                weights_applied = EXCLUDED.weights_applied
        """)

        import json
        # Clamp values to valid ranges for NUMERIC(10,4) fields
        await self.session.execute(query, {
            "ticker": ticker,
            "stage": result.stage.value,
            "revenue_growth": self._clamp_numeric(data.get('revenue_growth_yoy'), -1000, 10000),
            "net_margin": self._clamp_numeric(data.get('net_margin'), -1000, 1000),
            "pe_ratio": self._clamp_numeric(data.get('pe_ratio'), -10000, 100000),
            "market_cap": data.get('market_cap'),
            "weights": json.dumps(result.adjusted_weights),
        })
        await self.session.commit()

    async def get_classification(self, ticker: str) -> Optional[ClassificationResult]:
        """Get latest classification for a ticker from database.

        Args:
            ticker: Stock ticker symbol

        Returns:
            ClassificationResult or None if not found
        """
        if not self.session:
            return None

        query = text("""
            SELECT lifecycle_stage, revenue_growth_yoy, net_margin,
                   pe_ratio, market_cap, weights_applied
            FROM lifecycle_classifications
            WHERE ticker = :ticker
            ORDER BY classified_at DESC
            LIMIT 1
        """)

        result = await self.session.execute(query, {"ticker": ticker})
        row = result.fetchone()

        if not row:
            return None

        import json
        weights = json.loads(row.weights_applied) if row.weights_applied else self.BASE_WEIGHTS

        return ClassificationResult(
            stage=LifecycleStage(row.lifecycle_stage),
            confidence=0.8,  # Stored classifications assumed confident
            metrics_used={
                'revenue_growth_yoy': float(row.revenue_growth_yoy) if row.revenue_growth_yoy else 0,
                'net_margin': float(row.net_margin) if row.net_margin else 0,
                'pe_ratio': float(row.pe_ratio) if row.pe_ratio else 0,
            },
            adjusted_weights=weights,
        )


def get_lifecycle_stage_description(stage: LifecycleStage) -> str:
    """Get human-readable description of lifecycle stage.

    Args:
        stage: Lifecycle stage enum

    Returns:
        Description string for UI display
    """
    descriptions = {
        LifecycleStage.HYPERGROWTH: "Hypergrowth company with >50% revenue growth. "
                                    "Focus on growth trajectory over current profitability.",
        LifecycleStage.GROWTH: "Growth company with 20-50% revenue growth. "
                               "Balancing expansion with emerging profitability.",
        LifecycleStage.MATURE: "Mature company with stable operations. "
                               "Focus on profitability, cash flow, and capital efficiency.",
        LifecycleStage.VALUE: "Value opportunity with low valuation and solid margins. "
                              "Focus on intrinsic value and dividend potential.",
        LifecycleStage.TURNAROUND: "Turnaround situation with declining revenue. "
                                   "Focus on financial health and recovery signals.",
    }
    return descriptions.get(stage, "Unknown lifecycle stage")
