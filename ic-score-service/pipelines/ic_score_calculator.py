#!/usr/bin/env python3
"""IC Score Calculator Pipeline.

This script calculates InvestorCenter proprietary IC Scores (1-100) using 10
financial factors with weighted averaging and sector-relative percentiles.

Usage:
    python ic_score_calculator.py --ticker AAPL   # Single stock
    python ic_score_calculator.py --limit 100     # Test on 100 stocks
    python ic_score_calculator.py --all           # All stocks
    python ic_score_calculator.py --sector Technology  # All tech stocks
"""

import argparse
import asyncio
import logging
import os
import sys
from datetime import datetime, date, timedelta
from decimal import Decimal
from pathlib import Path
from typing import List, Optional, Dict, Any, Tuple

sys.path.insert(0, str(Path(__file__).parent.parent))

import numpy as np
import pandas as pd
from sqlalchemy import text, select
from sqlalchemy.dialects.postgresql import insert as pg_insert
from tqdm import tqdm

from database.database import get_database
from models import ICScore, Financial, TechnicalIndicator, InsiderTrade, InstitutionalHolding, AnalystRating, NewsArticle
from pipelines.utils.sector_percentile import SectorPercentileCalculator
from pipelines.utils.lifecycle import LifecycleClassifier, LifecycleStage
from pipelines.utils.earnings_revisions import EarningsRevisionsCalculator
from pipelines.utils.historical_valuation import HistoricalValuationCalculator
from pipelines.utils.dividend_quality import DividendQualityCalculator
# Phase 3 imports
from pipelines.utils.score_stabilizer import ScoreStabilizer, StabilizationResult, EventType
from pipelines.utils.peer_comparison import PeerComparisonService, PeerComparisonResult
from pipelines.utils.catalyst_detector import CatalystService, Catalyst
from pipelines.utils.score_explainer import ScoreExplainer, ScoreChangeExplanation

# Setup logging with configurable log directory
LOG_DIR = os.environ.get('LOG_DIR', '/app/logs')
Path(LOG_DIR).mkdir(parents=True, exist_ok=True)

logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s',
    handlers=[
        logging.StreamHandler(),
        logging.FileHandler(os.path.join(LOG_DIR, 'ic_score_calculator.log'))
    ]
)
logger = logging.getLogger(__name__)


class ICScoreCalculator:
    """Calculator for InvestorCenter proprietary IC Scores.

    IC Score v2.1 Features:
    - Sector-relative scoring using percentiles
    - Lifecycle-aware weight adjustments
    - Phase 2 factors: earnings_revisions, historical_value, dividend_quality
    - Smart Money consolidation (analyst + insider + institutional)
    """

    # Base factor weights (sum to 1.0)
    # Can be adjusted by lifecycle classification
    WEIGHTS = {
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
        'smart_money': 0.10,  # Combined: analyst + insider + institutional
        'earnings_revisions': 0.08,
        'technical': 0.07,
    }

    # Feature flags
    USE_SECTOR_RELATIVE_SCORING = True
    USE_LIFECYCLE_WEIGHTS = True

    # Phase 2 feature flags
    USE_EARNINGS_REVISIONS = True
    USE_HISTORICAL_VALUATION = True
    USE_DIVIDEND_QUALITY = False  # Optional, enable for income mode

    # Phase 3 feature flags
    USE_SCORE_STABILIZATION = True   # Exponential smoothing to prevent whipsaw
    USE_PEER_COMPARISON = True       # Similar stock comparison
    USE_CATALYST_DETECTION = True    # Upcoming event detection
    USE_SCORE_EXPLANATIONS = True    # Human-readable explanations

    # Rating thresholds
    RATING_THRESHOLDS = {
        'Strong Buy': 80,
        'Buy': 65,
        'Hold': 50,
        'Underperform': 35,
        'Sell': 0,
    }

    # Minimum data completeness threshold (percentage)
    # Require at least 40% of factors (4/10) to calculate a score
    MIN_DATA_COMPLETENESS = 40.0

    # Core factors that should be present for reliable scoring
    CORE_FACTORS = {'value', 'growth', 'profitability', 'financial_health'}

    # Valuation benchmarks for scoring normalization
    # These represent "fair value" centers for scoring
    VALUATION_BENCHMARKS = {
        'pe_ratio': 15.0,    # S&P 500 historical average P/E
        'pb_ratio': 2.0,     # Typical fair value P/B
        'ps_ratio': 2.0,     # Typical fair value P/S
    }

    # Scoring scale factors
    # Higher factor = more sensitive to deviations from benchmark
    SCALE_FACTORS = {
        'pe_scale': 2.0,         # P/E deviation scaling
        'pb_scale': 20.0,        # P/B deviation scaling
        'ps_scale': 20.0,        # P/S deviation scaling
        'growth_scale': 2.5,     # Growth rate scaling (50 + growth * scale)
        'margin_scale': 5.0,     # Margin % to score (margin * scale)
        'roe_scale': 5.0,        # ROE % to score (roe * scale)
        'roa_scale': 10.0,       # ROA % to score (roa * scale)
        'de_scale': 50.0,        # D/E ratio scaling (100 - de * scale)
        'cr_optimal': 2.0,       # Optimal current ratio
        'cr_scale': 40.0,        # Current ratio deviation scaling
        'return_scale': 2.5,     # Return % to score adjustment
        'macd_scale': 10.0,      # MACD histogram scaling
        'trend_scale': 5.0,      # Price vs SMA scaling
        'insider_scale': 2000.0, # Shares to score scaling
    }

    def __init__(self, income_mode: bool = False):
        """Initialize the IC Score calculator.

        Args:
            income_mode: If True, include Dividend Quality factor for
                        income-focused analysis (+5% weight).
        """
        self.db = get_database()
        self.income_mode = income_mode

        # v2.1 components (initialized per-session)
        self._sector_calculator: Optional[SectorPercentileCalculator] = None
        self._lifecycle_classifier: Optional[LifecycleClassifier] = None

        # Phase 2 factor calculators (initialized per-session)
        self._earnings_revisions_calc: Optional[EarningsRevisionsCalculator] = None
        self._historical_valuation_calc: Optional[HistoricalValuationCalculator] = None
        self._dividend_quality_calc: Optional[DividendQualityCalculator] = None

        # Phase 3 components (initialized per-session)
        self._score_stabilizer: Optional[ScoreStabilizer] = None
        self._peer_comparison: Optional[PeerComparisonService] = None
        self._catalyst_detector: Optional[CatalystService] = None
        self._score_explainer: Optional[ScoreExplainer] = None

        # Track progress
        self.processed_count = 0
        self.success_count = 0
        self.error_count = 0

    async def _init_v2_components(self, session):
        """Initialize scoring components with session.

        Called once per calculation batch to set up sector percentile
        calculator, lifecycle classifier, and Phase 2 factor calculators.
        """
        if self.USE_SECTOR_RELATIVE_SCORING:
            self._sector_calculator = SectorPercentileCalculator(session)
        else:
            self._sector_calculator = None

        if self.USE_LIFECYCLE_WEIGHTS:
            self._lifecycle_classifier = LifecycleClassifier(session)
        else:
            self._lifecycle_classifier = None

        # Initialize Phase 2 factor calculators
        if self.USE_EARNINGS_REVISIONS:
            self._earnings_revisions_calc = EarningsRevisionsCalculator(session)

        if self.USE_HISTORICAL_VALUATION:
            self._historical_valuation_calc = HistoricalValuationCalculator(session)

        if self.USE_DIVIDEND_QUALITY or self.income_mode:
            self._dividend_quality_calc = DividendQualityCalculator(
                session,
                sector_calculator=self._sector_calculator
            )

        # Initialize Phase 3 components
        if self.USE_SCORE_STABILIZATION:
            self._score_stabilizer = ScoreStabilizer(session)

        if self.USE_PEER_COMPARISON:
            self._peer_comparison = PeerComparisonService(session)

        if self.USE_CATALYST_DETECTION:
            self._catalyst_detector = CatalystService(session)

        if self.USE_SCORE_EXPLANATIONS:
            self._score_explainer = ScoreExplainer(session)

    async def get_stocks_to_process(
        self,
        limit: Optional[int] = None,
        ticker: Optional[str] = None,
        sector: Optional[str] = None,
        sp500: bool = False
    ) -> List[Dict[str, Any]]:
        """Get list of stocks to process.

        Args:
            limit: Maximum number of stocks.
            ticker: Single ticker to process.
            sector: Filter by sector.
            sp500: Only S&P 500 stocks.

        Returns:
            List of stock dictionaries.
        """
        async with self.db.session() as session:
            if ticker:
                query = text("""
                    SELECT symbol AS ticker, sector
                    FROM tickers
                    WHERE symbol = :ticker
                """)
                result = await session.execute(query, {"ticker": ticker.upper()})
            else:
                # Only process stocks (not ETFs, indices, crypto, etc.)
                # Note: asset_type 'CS' = Common Stock, 'stock' is legacy value
                # Exclude index tickers (I:*) and crypto (X:*) which may be misclassified
                where_clauses = [
                    "symbol NOT LIKE '%-%'",
                    "symbol NOT LIKE 'I:%'",
                    "symbol NOT LIKE 'X:%'",
                    "active = true",
                    "asset_type IN ('CS', 'stock')"
                ]
                params = {}

                if sector:
                    where_clauses.append("sector = :sector")
                    params['sector'] = sector

                if sp500:
                    where_clauses.append("is_sp500 = true")

                query_str = f"""
                    SELECT symbol AS ticker, sector
                    FROM tickers
                    WHERE {' AND '.join(where_clauses)}
                    ORDER BY symbol
                    LIMIT :limit
                """
                params['limit'] = limit or 10000
                query = text(query_str)
                result = await session.execute(query, params)

            stocks = [{"ticker": row[0], "sector": row[1]} for row in result.fetchall()]

        logger.info(f"Found {len(stocks)} stocks to process")
        return stocks

    async def fetch_financial_data(self, ticker: str) -> Optional[Dict[str, Any]]:
        """Fetch latest financial data for a stock."""
        async with self.db.session() as session:
            # Prefer rows with actual metrics (net_margin not null)
            # Filter out future dates to handle data quality issues
            query = text("""
                SELECT *
                FROM financials
                WHERE ticker = :ticker
                  AND period_end_date <= CURRENT_DATE
                ORDER BY
                    period_end_date DESC,
                    CASE WHEN net_margin IS NOT NULL THEN 0 ELSE 1 END
                LIMIT 20
            """)
            result = await session.execute(query, {"ticker": ticker})
            rows = result.fetchall()

            if not rows:
                return None

            # Get latest row with actual metrics
            latest = rows[0]._asdict() if rows else {}
            historical = [row._asdict() for row in rows]

            return {'latest': latest, 'historical': historical}

    async def fetch_fundamental_metrics(self, ticker: str) -> Optional[Dict[str, Any]]:
        """Fetch latest fundamental metrics from fundamental_metrics_extended.

        This table contains pre-calculated growth rates, profitability metrics,
        and other derived values that are more reliable than calculating from
        raw financials.
        """
        async with self.db.session() as session:
            query = text("""
                SELECT
                    revenue_growth_yoy, eps_growth_yoy,
                    net_margin, roe, roa, roic,
                    gross_margin, operating_margin,
                    dividend_yield, payout_ratio,
                    debt_to_equity, current_ratio, quick_ratio,
                    ev_to_ebitda, ev_to_revenue,
                    calculation_date,
                    fcf_growth_yoy,
                    dcf_fair_value, dcf_upside_percent, graham_number
                FROM fundamental_metrics_extended
                WHERE ticker = :ticker
                ORDER BY calculation_date DESC
                LIMIT 1
            """)
            result = await session.execute(query, {"ticker": ticker})
            row = result.fetchone()

            if not row:
                return None

            return {
                'revenue_growth_yoy': float(row[0]) if row[0] is not None else None,
                'eps_growth_yoy': float(row[1]) if row[1] is not None else None,
                'net_margin': float(row[2]) if row[2] is not None else None,
                'roe': float(row[3]) if row[3] is not None else None,
                'roa': float(row[4]) if row[4] is not None else None,
                'roic': float(row[5]) if row[5] is not None else None,
                'gross_margin': float(row[6]) if row[6] is not None else None,
                'operating_margin': float(row[7]) if row[7] is not None else None,
                'dividend_yield': float(row[8]) if row[8] is not None else None,
                'payout_ratio': float(row[9]) if row[9] is not None else None,
                'debt_to_equity': float(row[10]) if row[10] is not None else None,
                'current_ratio': float(row[11]) if row[11] is not None else None,
                'quick_ratio': float(row[12]) if row[12] is not None else None,
                'ev_to_ebitda': float(row[13]) if row[13] is not None else None,
                'ev_to_revenue': float(row[14]) if row[14] is not None else None,
                'calculation_date': row[15],
                'fcf_growth_yoy': float(row[16]) if row[16] is not None else None,
                'dcf_fair_value': float(row[17]) if row[17] is not None else None,
                'dcf_upside_percent': float(row[18]) if row[18] is not None else None,
                'graham_number': float(row[19]) if row[19] is not None else None,
            }

    async def fetch_technical_data(self, ticker: str) -> Optional[Dict[str, Any]]:
        """Fetch latest technical indicators for a stock."""
        async with self.db.session() as session:
            query = text("""
                SELECT indicator_name, value
                FROM technical_indicators
                WHERE ticker = :ticker
                  AND time >= NOW() - INTERVAL '14 days'
                ORDER BY time DESC
                LIMIT 100
            """)
            result = await session.execute(query, {"ticker": ticker})
            rows = result.fetchall()

            if not rows:
                return None

            # Convert to dict
            indicators = {}
            for row in rows:
                if row[0] not in indicators:  # Take most recent value
                    indicators[row[0]] = float(row[1])

            return indicators

    async def fetch_insider_data(self, ticker: str) -> Optional[Dict[str, Any]]:
        """Fetch insider trading data for a stock."""
        async with self.db.session() as session:
            # Get last 90 days of insider trades with both share counts and dollar values
            query = text("""
                SELECT transaction_type, SUM(shares) as total_shares, SUM(total_value) as total_value
                FROM insider_trades
                WHERE ticker = :ticker
                  AND transaction_date >= NOW() - INTERVAL '90 days'
                GROUP BY transaction_type
            """)
            result = await session.execute(query, {"ticker": ticker})
            rows = result.fetchall()

            if not rows:
                return None

            net_buying_shares = 0
            net_buying_value = 0
            total_transactions = 0
            for row in rows:
                total_transactions += 1
                if row[0] and 'buy' in row[0].lower():
                    net_buying_shares += row[1] or 0
                    net_buying_value += float(row[2] or 0)
                elif row[0] and 'sell' in row[0].lower():
                    net_buying_shares -= row[1] or 0
                    net_buying_value -= float(row[2] or 0)

            return {
                'net_buying_90d': net_buying_shares,
                'net_buying_value_90d': net_buying_value,
                'total_transactions': total_transactions,
            }

    async def fetch_news_sentiment_data(self, ticker: str) -> Optional[Dict[str, Any]]:
        """Fetch news sentiment data for a stock with recency weighting."""
        async with self.db.session() as session:
            # Get overall 30-day stats
            query = text("""
                SELECT
                    COUNT(*) as article_count,
                    AVG(sentiment_score) as avg_sentiment,
                    SUM(CASE WHEN sentiment_label = 'Positive' THEN 1 ELSE 0 END) as positive_count,
                    SUM(CASE WHEN sentiment_label = 'Negative' THEN 1 ELSE 0 END) as negative_count,
                    SUM(CASE WHEN sentiment_label = 'Neutral' THEN 1 ELSE 0 END) as neutral_count
                FROM news_articles
                WHERE :ticker = ANY(tickers)
                  AND published_at >= NOW() - INTERVAL '30 days'
            """)
            result = await session.execute(query, {"ticker": ticker})
            row = result.fetchone()

            if not row or row[0] == 0:
                return None

            # Get recent 7-day sentiment for recency weighting
            recent_query = text("""
                SELECT
                    COUNT(*) as article_count,
                    AVG(sentiment_score) as avg_sentiment
                FROM news_articles
                WHERE :ticker = ANY(tickers)
                  AND published_at >= NOW() - INTERVAL '7 days'
            """)
            recent_result = await session.execute(recent_query, {"ticker": ticker})
            recent_row = recent_result.fetchone()

            return {
                'article_count': row[0],
                'avg_sentiment': float(row[1]) if row[1] else 0,
                'positive_count': row[2] or 0,
                'negative_count': row[3] or 0,
                'neutral_count': row[4] or 0,
                'recent_article_count': recent_row[0] if recent_row else 0,
                'recent_avg_sentiment': float(recent_row[1]) if recent_row and recent_row[1] else None,
            }

    async def fetch_analyst_data(self, ticker: str) -> Optional[Dict[str, Any]]:
        """Fetch analyst ratings data for a stock."""
        async with self.db.session() as session:
            # Get most recent analyst ratings
            query = text("""
                SELECT
                    rating,
                    price_target,
                    analyst_firm,
                    rating_date
                FROM analyst_ratings
                WHERE ticker = :ticker
                  AND rating_date >= NOW() - INTERVAL '90 days'
                ORDER BY rating_date DESC
                LIMIT 20
            """)
            result = await session.execute(query, {"ticker": ticker})
            rows = result.fetchall()

            if not rows:
                return None

            # Count buy/hold/sell ratings
            buy_count = sum(1 for row in rows if row[0] and any(x in row[0].lower() for x in ['buy', 'outperform', 'overweight']))
            hold_count = sum(1 for row in rows if row[0] and any(x in row[0].lower() for x in ['hold', 'neutral', 'equal']))
            sell_count = sum(1 for row in rows if row[0] and any(x in row[0].lower() for x in ['sell', 'underperform', 'underweight']))

            # Calculate average price target
            price_targets = [float(row[1]) for row in rows if row[1]]
            avg_price_target = np.mean(price_targets) if price_targets else None

            return {
                'total_analysts': len(rows),
                'buy_count': buy_count,
                'hold_count': hold_count,
                'sell_count': sell_count,
                'avg_price_target': avg_price_target
            }

    async def fetch_institutional_data(self, ticker: str) -> Optional[Dict[str, Any]]:
        """Fetch institutional holdings data for a stock."""
        async with self.db.session() as session:
            # Get most recent quarter holdings
            query = text("""
                SELECT
                    SUM(shares) as total_shares,
                    SUM(market_value) as total_value,
                    COUNT(DISTINCT institution_cik) as num_institutions
                FROM institutional_holdings
                WHERE ticker = :ticker
                  AND filing_date >= NOW() - INTERVAL '120 days'
            """)
            result = await session.execute(query, {"ticker": ticker})
            row = result.fetchone()

            if not row or row[0] is None:
                return None

            # Get change from previous quarter
            prev_query = text("""
                SELECT SUM(shares) as prev_shares
                FROM institutional_holdings
                WHERE ticker = :ticker
                  AND filing_date >= NOW() - INTERVAL '240 days'
                  AND filing_date < NOW() - INTERVAL '120 days'
            """)
            prev_result = await session.execute(prev_query, {"ticker": ticker})
            prev_row = prev_result.fetchone()

            prev_shares = prev_row[0] if prev_row and prev_row[0] else None

            # Get shares outstanding for ownership percentage
            shares_outstanding = None
            shares_query = text("""
                SELECT shares_outstanding_diluted
                FROM financials
                WHERE ticker = :ticker
                  AND shares_outstanding_diluted IS NOT NULL
                  AND period_end_date <= CURRENT_DATE
                ORDER BY period_end_date DESC
                LIMIT 1
            """)
            shares_result = await session.execute(shares_query, {"ticker": ticker})
            shares_row = shares_result.fetchone()
            if shares_row and shares_row[0]:
                shares_outstanding = int(shares_row[0])

            return {
                'total_shares': int(row[0]) if row[0] else 0,
                'total_value': float(row[1]) if row[1] else 0,
                'num_institutions': row[2] or 0,
                'prev_shares': int(prev_shares) if prev_shares else None,
                'shares_outstanding': shares_outstanding,
            }

    async def fetch_valuation_data(self, ticker: str) -> Optional[Dict[str, Any]]:
        """Fetch latest valuation ratios from valuation_ratios table.

        This table contains TTM-based P/E, P/B, P/S ratios calculated by
        the valuation_ratios_calculator pipeline using current stock prices.
        """
        async with self.db.session() as session:
            query = text("""
                SELECT
                    ttm_pe_ratio,
                    ttm_pb_ratio,
                    ttm_ps_ratio,
                    stock_price,
                    ttm_market_cap,
                    calculation_date
                FROM valuation_ratios
                WHERE ticker = :ticker
                ORDER BY calculation_date DESC
                LIMIT 1
            """)
            result = await session.execute(query, {"ticker": ticker})
            row = result.fetchone()

            if not row:
                return None

            return {
                'pe_ratio': float(row[0]) if row[0] else None,
                'pb_ratio': float(row[1]) if row[1] else None,
                'ps_ratio': float(row[2]) if row[2] else None,
                'stock_price': float(row[3]) if row[3] else None,
                'market_cap': int(row[4]) if row[4] else None,
                'calculation_date': row[5]
            }

    async def calculate_value_score(
        self,
        valuation_data: Optional[Dict[str, Any]],
        sector: Optional[str]
    ) -> Tuple[Optional[float], Dict]:
        """Calculate value score based on P/E, P/B, P/S ratios.

        v2.1 Scoring methodology (sector-relative):
        - Uses sector percentiles instead of absolute benchmarks
        - Lower valuation = higher percentile = higher score
        - Falls back to legacy scoring if sector data unavailable

        Legacy scoring methodology (absolute benchmarks):
        - P/E benchmark: 15 (S&P 500 historical average)
        - P/B, P/S benchmarks: 2 (typical fair value)

        Args:
            valuation_data: Dict with pe_ratio, pb_ratio, ps_ratio
            sector: GICS sector for percentile comparison
        """
        if not valuation_data:
            return None, {}

        metadata = {}
        scores = []

        pe = valuation_data.get('pe_ratio')
        pb = valuation_data.get('pb_ratio')
        ps = valuation_data.get('ps_ratio')

        if not any([pe, pb, ps]):
            return None, metadata

        # v2.1: Use sector-relative percentiles
        if self._sector_calculator and sector and self.USE_SECTOR_RELATIVE_SCORING:
            if pe and pe > 0:
                pe_pct = await self._sector_calculator.get_percentile(sector, 'pe_ratio', pe)
                if pe_pct is not None:
                    scores.append(pe_pct)
                    metadata['pe_ratio'] = float(pe)
                    metadata['pe_sector_percentile'] = pe_pct

            if pb and pb > 0:
                pb_pct = await self._sector_calculator.get_percentile(sector, 'pb_ratio', pb)
                if pb_pct is not None:
                    scores.append(pb_pct)
                    metadata['pb_ratio'] = float(pb)
                    metadata['pb_sector_percentile'] = pb_pct

            if ps and ps > 0:
                ps_pct = await self._sector_calculator.get_percentile(sector, 'ps_ratio', ps)
                if ps_pct is not None:
                    scores.append(ps_pct)
                    metadata['ps_ratio'] = float(ps)
                    metadata['ps_sector_percentile'] = ps_pct

            metadata['scoring_method'] = 'sector_relative'

        # Fallback or legacy: Use absolute benchmarks
        if not scores:
            metadata['scoring_method'] = 'absolute_benchmark'

            if pe and pe > 0:
                pe_benchmark = self.VALUATION_BENCHMARKS['pe_ratio']
                pe_scale = self.SCALE_FACTORS['pe_scale']
                pe_score = max(0, min(100, 100 - (pe - pe_benchmark) * pe_scale))
                scores.append(pe_score)
                metadata['pe_ratio'] = float(pe)

            if pb and pb > 0:
                pb_benchmark = self.VALUATION_BENCHMARKS['pb_ratio']
                pb_scale = self.SCALE_FACTORS['pb_scale']
                pb_score = max(0, min(100, 100 - (pb - pb_benchmark) * pb_scale))
                scores.append(pb_score)
                metadata['pb_ratio'] = float(pb)

            if ps and ps > 0:
                ps_benchmark = self.VALUATION_BENCHMARKS['ps_ratio']
                ps_scale = self.SCALE_FACTORS['ps_scale']
                ps_score = max(0, min(100, 100 - (ps - ps_benchmark) * ps_scale))
                scores.append(ps_score)
                metadata['ps_ratio'] = float(ps)

        if not scores:
            return None, metadata

        return np.mean(scores), metadata

    async def calculate_growth_score(
        self,
        fundamental_metrics: Optional[Dict[str, Any]],
        financial_data: Optional[Dict[str, Any]],
        sector: Optional[str]
    ) -> Tuple[Optional[float], Dict]:
        """Calculate growth score based on revenue, EPS, FCF growth.

        v2.1 Scoring methodology (sector-relative):
        - Uses pre-calculated growth rates from fundamental_metrics_extended
        - Uses sector percentiles for growth comparison
        - Higher growth vs sector = higher score

        Legacy scoring methodology:
        - 0% growth = 50 (neutral)
        - +20% growth = 100 (maximum score)
        - -20% growth = 0 (minimum score)
        """
        metadata = {}
        growth_scale = self.SCALE_FACTORS['growth_scale']
        scores = []

        # Prefer pre-calculated growth rates from fundamental_metrics_extended
        rev_growth = None
        eps_growth = None
        fcf_growth = None

        if fundamental_metrics:
            rev_growth = fundamental_metrics.get('revenue_growth_yoy')
            eps_growth = fundamental_metrics.get('eps_growth_yoy')
            fcf_growth = fundamental_metrics.get('fcf_growth_yoy')
            if rev_growth is not None:
                metadata['revenue_growth_yoy'] = rev_growth
            if eps_growth is not None:
                metadata['eps_growth_yoy'] = eps_growth
            if fcf_growth is not None:
                metadata['fcf_growth_yoy'] = fcf_growth

        # Fallback: Calculate from raw financials if pre-calculated not available
        if rev_growth is None and financial_data and len(financial_data.get('historical', [])) >= 4:
            historical = financial_data['historical']
            if historical[0].get('revenue') and historical[3].get('revenue'):
                rev_current = float(historical[0]['revenue'])
                rev_prior = float(historical[3]['revenue'])
                if rev_prior > 0:
                    rev_growth = ((rev_current / rev_prior) - 1) * 100
                    metadata['revenue_growth_yoy'] = rev_growth
                    metadata['growth_source'] = 'calculated'

        if rev_growth is None and eps_growth is None and fcf_growth is None:
            return None, metadata

        # v2.1: Use sector-relative percentiles
        if self._sector_calculator and sector and self.USE_SECTOR_RELATIVE_SCORING:
            if rev_growth is not None:
                rev_pct = await self._sector_calculator.get_percentile(
                    sector, 'revenue_growth_yoy', rev_growth
                )
                if rev_pct is not None:
                    scores.append(rev_pct)
                    metadata['revenue_growth_sector_percentile'] = rev_pct

            if eps_growth is not None:
                eps_pct = await self._sector_calculator.get_percentile(
                    sector, 'eps_growth_yoy', eps_growth
                )
                if eps_pct is not None:
                    scores.append(eps_pct)
                    metadata['eps_growth_sector_percentile'] = eps_pct

            if scores:
                metadata['scoring_method'] = 'sector_relative'

        # Fallback or legacy: Use absolute scaling
        if not scores:
            metadata['scoring_method'] = 'absolute_benchmark'

            if rev_growth is not None:
                rev_score = max(0, min(100, 50 + rev_growth * growth_scale))
                scores.append(rev_score)

            if eps_growth is not None:
                eps_score = max(0, min(100, 50 + eps_growth * growth_scale))
                scores.append(eps_score)

        # FCF growth as supplemental signal (always use absolute scaling since
        # sector percentiles for FCF growth aren't tracked yet)
        if fcf_growth is not None:
            fcf_score = max(0, min(100, 50 + fcf_growth * growth_scale))
            scores.append(fcf_score)
            metadata['fcf_growth_score'] = round(fcf_score, 2)

        if not scores:
            return None, metadata

        return np.mean(scores), metadata

    async def calculate_profitability_score(
        self,
        fundamental_metrics: Optional[Dict[str, Any]],
        financial_data: Optional[Dict[str, Any]],
        sector: Optional[str]
    ) -> Tuple[Optional[float], Dict]:
        """Calculate profitability score based on margins, ROE, ROA.

        v2.1 Scoring methodology (sector-relative):
        - Uses pre-calculated metrics from fundamental_metrics_extended
        - Uses sector percentiles for profitability comparison
        - Higher margins/ROE/ROA vs sector = higher score

        Legacy scoring methodology:
        - 0% margin/ROE = 0, 20% = 100
        - 0% ROA = 0, 10% = 100 (ROA typically lower than ROE)
        """
        metadata = {}
        scores = []

        # Prefer data from fundamental_metrics_extended
        margin = None
        gross_margin = None
        operating_margin = None
        roe = None
        roa = None

        if fundamental_metrics:
            margin = fundamental_metrics.get('net_margin')
            gross_margin = fundamental_metrics.get('gross_margin')
            operating_margin = fundamental_metrics.get('operating_margin')
            roe = fundamental_metrics.get('roe')
            roa = fundamental_metrics.get('roa')

        # Fallback to financials table
        if financial_data and (margin is None or roe is None or roa is None):
            latest = financial_data.get('latest', {})
            if margin is None and latest.get('net_margin'):
                margin = float(latest['net_margin'])
            if gross_margin is None and latest.get('gross_margin'):
                gross_margin = float(latest['gross_margin'])
            if operating_margin is None and latest.get('operating_margin'):
                operating_margin = float(latest['operating_margin'])
            if roe is None and latest.get('roe'):
                roe = float(latest['roe'])
            if roa is None and latest.get('roa'):
                roa = float(latest['roa'])

        if margin is None and roe is None and roa is None and gross_margin is None:
            return None, {}

        if margin is not None:
            metadata['net_margin'] = margin
        if gross_margin is not None:
            metadata['gross_margin'] = gross_margin
        if operating_margin is not None:
            metadata['operating_margin'] = operating_margin
        if roe is not None:
            metadata['roe'] = roe
        if roa is not None:
            metadata['roa'] = roa

        # v2.1: Use sector-relative percentiles
        if self._sector_calculator and sector and self.USE_SECTOR_RELATIVE_SCORING:
            if margin is not None:
                margin_pct = await self._sector_calculator.get_percentile(
                    sector, 'net_margin', margin
                )
                if margin_pct is not None:
                    scores.append(margin_pct)
                    metadata['net_margin_sector_percentile'] = margin_pct

            if gross_margin is not None:
                gm_pct = await self._sector_calculator.get_percentile(
                    sector, 'gross_margin', gross_margin
                )
                if gm_pct is not None:
                    scores.append(gm_pct)
                    metadata['gross_margin_sector_percentile'] = gm_pct

            if operating_margin is not None:
                om_pct = await self._sector_calculator.get_percentile(
                    sector, 'operating_margin', operating_margin
                )
                if om_pct is not None:
                    scores.append(om_pct)
                    metadata['operating_margin_sector_percentile'] = om_pct

            if roe is not None:
                roe_pct = await self._sector_calculator.get_percentile(sector, 'roe', roe)
                if roe_pct is not None:
                    scores.append(roe_pct)
                    metadata['roe_sector_percentile'] = roe_pct

            if roa is not None:
                roa_pct = await self._sector_calculator.get_percentile(sector, 'roa', roa)
                if roa_pct is not None:
                    scores.append(roa_pct)
                    metadata['roa_sector_percentile'] = roa_pct

            if scores:
                metadata['scoring_method'] = 'sector_relative'

        # Fallback or legacy: Use absolute scaling
        if not scores:
            metadata['scoring_method'] = 'absolute_benchmark'

            if margin is not None:
                margin_scale = self.SCALE_FACTORS['margin_scale']
                margin_score = max(0, min(100, margin * margin_scale))
                scores.append(margin_score)

            if gross_margin is not None:
                # Gross margin typically ranges 20-80%, scale accordingly
                gm_score = max(0, min(100, gross_margin * 1.5))
                scores.append(gm_score)

            if operating_margin is not None:
                # Operating margin typically ranges 5-30%
                om_score = max(0, min(100, operating_margin * 3.0))
                scores.append(om_score)

            if roe is not None:
                roe_scale = self.SCALE_FACTORS['roe_scale']
                roe_score = max(0, min(100, roe * roe_scale))
                scores.append(roe_score)

            if roa is not None:
                roa_scale = self.SCALE_FACTORS['roa_scale']
                roa_score = max(0, min(100, roa * roa_scale))
                scores.append(roa_score)

        if not scores:
            return None, metadata

        return np.mean(scores), metadata

    async def calculate_financial_health_score(
        self,
        fundamental_metrics: Optional[Dict[str, Any]],
        financial_data: Optional[Dict[str, Any]],
        sector: Optional[str] = None
    ) -> Tuple[Optional[float], Dict]:
        """Calculate financial health score based on D/E, current ratio, quick ratio.

        v2.1 Scoring methodology (sector-relative):
        - Uses sector percentiles for D/E and current ratio
        - Lower D/E vs sector = higher score
        - Current ratio closer to sector median = higher score

        Legacy scoring methodology (absolute benchmarks):
        - D/E: 0 = 100 (no debt), 2+ = 0 (high debt)
        - Current ratio: 2.0 = 100 (optimal), 0 or 5+ = 0 (extremes)
        """
        metadata = {}
        scores = []

        # Get debt_to_equity, current_ratio, quick_ratio from best source
        de = None
        cr = None
        qr = None

        if fundamental_metrics:
            de = fundamental_metrics.get('debt_to_equity')
            cr = fundamental_metrics.get('current_ratio')
            qr = fundamental_metrics.get('quick_ratio')

        # Fallback to financials table
        if financial_data:
            latest = financial_data.get('latest', {})
            if de is None and latest.get('debt_to_equity') is not None:
                de = float(latest['debt_to_equity'])
            if cr is None and latest.get('current_ratio'):
                cr = float(latest['current_ratio'])
            if qr is None and latest.get('quick_ratio'):
                qr = float(latest['quick_ratio'])

        if de is not None:
            metadata['debt_to_equity'] = de
        if cr is not None:
            metadata['current_ratio'] = cr
        if qr is not None:
            metadata['quick_ratio'] = qr

        # v2.1: Use sector-relative percentiles
        if self._sector_calculator and sector and self.USE_SECTOR_RELATIVE_SCORING:
            if de is not None:
                de_pct = await self._sector_calculator.get_percentile(
                    sector, 'debt_to_equity', de
                )
                if de_pct is not None:
                    scores.append(de_pct)
                    metadata['de_sector_percentile'] = de_pct

            if cr is not None:
                cr_pct = await self._sector_calculator.get_percentile(
                    sector, 'current_ratio', cr
                )
                if cr_pct is not None:
                    scores.append(cr_pct)
                    metadata['cr_sector_percentile'] = cr_pct

            if qr is not None:
                qr_pct = await self._sector_calculator.get_percentile(
                    sector, 'quick_ratio', qr
                )
                if qr_pct is not None:
                    scores.append(qr_pct)
                    metadata['qr_sector_percentile'] = qr_pct

            if scores:
                metadata['scoring_method'] = 'sector_relative'

        # Fallback or legacy: Use absolute scaling
        if not scores:
            metadata['scoring_method'] = 'absolute_benchmark'

            # Debt to equity (lower is better)
            if de is not None:
                de_scale = self.SCALE_FACTORS['de_scale']
                de_score = max(0, min(100, 100 - de * de_scale))
                scores.append(de_score)

            # Current ratio (optimal around 1.5-2.0)
            if cr is not None:
                cr_optimal = self.SCALE_FACTORS['cr_optimal']
                cr_scale = self.SCALE_FACTORS['cr_scale']
                cr_score = max(0, min(100, 100 - abs(cr - cr_optimal) * cr_scale))
                scores.append(cr_score)

            # Quick ratio (optimal around 1.0-1.5)
            if qr is not None:
                qr_score = max(0, min(100, 100 - abs(qr - 1.5) * 40))
                scores.append(qr_score)

        if not scores:
            return None, metadata

        return np.mean(scores), metadata

    def calculate_momentum_score(self, technical_data: Dict[str, Any]) -> Tuple[Optional[float], Dict]:
        """Calculate momentum score based on price returns with recency weighting.

        Scoring methodology:
        - -20% return = 0, 0% = 50, +20% = 100
        - Recent returns weighted more heavily (1m: 35%, 3m: 30%, 6m: 20%, 12m: 15%)
        """
        if not technical_data:
            return None, {}

        metadata = {}
        weighted_scores = []
        return_scale = self.SCALE_FACTORS['return_scale']

        # Recency-weighted periods: more recent returns matter more
        period_weights = {
            '1m_return': 0.35,
            '3m_return': 0.30,
            '6m_return': 0.20,
            '12m_return': 0.15,
        }

        for period, weight in period_weights.items():
            if period in technical_data:
                ret = technical_data[period]
                # Normalize: -20% = 0, 0% = 50, 20% = 100
                score = max(0, min(100, 50 + ret * return_scale))
                weighted_scores.append((score, weight))
                metadata[period] = ret

        if not weighted_scores:
            return None, metadata

        # Weighted average with recency bias
        total_weight = sum(w for _, w in weighted_scores)
        momentum_score = sum(s * w for s, w in weighted_scores) / total_weight

        return round(momentum_score, 2), metadata

    def calculate_technical_score(self, technical_data: Dict[str, Any]) -> Tuple[Optional[float], Dict]:
        """Calculate technical score based on RSI, MACD, trend.

        Scoring methodology:
        - RSI: 30 = 0 (oversold), 50 = 50 (neutral), 70 = 100 (overbought)
        - MACD: Positive histogram = bullish, negative = bearish
        - Trend: Price above SMA50 = bullish
        """
        if not technical_data:
            return None, {}

        metadata = {}
        scores = []
        return_scale = self.SCALE_FACTORS['return_scale']
        macd_scale = self.SCALE_FACTORS['macd_scale']
        trend_scale = self.SCALE_FACTORS['trend_scale']

        # RSI (30-70 range is neutral, <30 oversold, >70 overbought)
        if 'rsi' in technical_data:
            rsi = technical_data['rsi']
            # Normalize: 30 = 0, 50 = 50, 70 = 100
            if rsi < 50:
                rsi_score = max(0, (rsi - 30) * return_scale)
            else:
                rsi_score = min(100, 50 + (rsi - 50) * return_scale)
            scores.append(rsi_score)
            metadata['rsi'] = rsi

        # MACD histogram (positive = bullish)
        if 'macd_histogram' in technical_data:
            macd_hist = technical_data['macd_histogram']
            # Simple normalization
            macd_score = 50 + min(50, max(-50, macd_hist * macd_scale))
            scores.append(macd_score)
            metadata['macd_histogram'] = macd_hist

        # Trend (price vs SMA)
        if 'current_price' in technical_data and 'sma_50' in technical_data:
            price = technical_data['current_price']
            sma = technical_data['sma_50']
            if sma > 0:
                trend = ((price / sma) - 1) * 100
                trend_score = max(0, min(100, 50 + trend * trend_scale))
                scores.append(trend_score)
                metadata['price_vs_sma50'] = trend

        if not scores:
            return None, metadata

        return np.mean(scores), metadata

    def calculate_news_sentiment_score(self, news_data: Optional[Dict[str, Any]]) -> Tuple[Optional[float], Dict]:
        """Calculate news sentiment score with recency weighting and volume context.

        Scoring methodology:
        - Blends 30-day and 7-day sentiment (60% recent, 40% overall)
        - Incorporates positive/negative article ratio
        - Low article count reduces confidence but doesn't penalize score
        """
        if not news_data or news_data.get('article_count', 0) == 0:
            return None, {}

        metadata = {}
        scores = []

        # 30-day overall sentiment (already 0-100 from sentiment analysis)
        avg_sentiment = news_data.get('avg_sentiment', 0)
        overall_score = max(0, min(100, avg_sentiment))
        metadata['avg_sentiment_30d'] = avg_sentiment

        # 7-day recent sentiment (weighted more heavily)
        recent_sentiment = news_data.get('recent_avg_sentiment')

        if recent_sentiment is not None:
            recent_score = max(0, min(100, recent_sentiment))
            # Blend: 60% recent (7d), 40% overall (30d)
            blended_score = recent_score * 0.60 + overall_score * 0.40
            scores.append(blended_score)
            metadata['avg_sentiment_7d'] = recent_sentiment
            metadata['scoring_method'] = 'recency_weighted'
        else:
            scores.append(overall_score)
            metadata['scoring_method'] = 'overall_only'

        # Positive vs negative ratio as supplemental signal
        positive_count = news_data.get('positive_count', 0)
        negative_count = news_data.get('negative_count', 0)
        total_articles = news_data.get('article_count', 0)

        if total_articles > 0:
            positive_ratio = (positive_count / total_articles) * 100
            # Convert ratio to score: 100% positive = 100, 50/50 = 50, 100% negative = 0
            ratio_score = max(0, min(100, positive_ratio))
            scores.append(ratio_score)
            metadata['positive_ratio'] = round(positive_ratio, 1)
            metadata['article_count'] = total_articles

        if not scores:
            return None, metadata

        return round(np.mean(scores), 2), metadata

    def calculate_analyst_consensus_score(
        self,
        analyst_data: Optional[Dict[str, Any]],
        valuation_data: Optional[Dict[str, Any]] = None
    ) -> Tuple[Optional[float], Dict]:
        """Calculate analyst consensus score based on ratings and price target upside.

        Scoring methodology:
        - Rating consensus: Buy=100, Hold=50, Sell=0 (60% weight)
        - Price target upside: >30% upside=100, 0%=50, <-30%=0 (40% weight)
        """
        if not analyst_data or analyst_data.get('total_analysts', 0) == 0:
            return None, {}

        metadata = {}
        total_analysts = analyst_data['total_analysts']
        buy_count = analyst_data.get('buy_count', 0)
        hold_count = analyst_data.get('hold_count', 0)
        sell_count = analyst_data.get('sell_count', 0)

        # Rating consensus score: 100 = all buy, 50 = all hold, 0 = all sell
        if total_analysts > 0:
            weighted_score = ((buy_count * 100) + (hold_count * 50) + (sell_count * 0)) / total_analysts
            rating_score = max(0, min(100, weighted_score))
        else:
            rating_score = 50

        metadata['total_analysts'] = total_analysts
        metadata['buy_count'] = buy_count
        metadata['hold_count'] = hold_count
        metadata['sell_count'] = sell_count
        metadata['rating_score'] = round(rating_score, 2)

        # Price target upside score
        avg_price_target = analyst_data.get('avg_price_target')
        stock_price = valuation_data.get('stock_price') if valuation_data else None
        metadata['avg_price_target'] = avg_price_target

        if avg_price_target and stock_price and stock_price > 0:
            upside_pct = ((avg_price_target / stock_price) - 1) * 100
            # Scale: -30% = 0, 0% = 50, +30% = 100
            upside_score = max(0, min(100, 50 + (upside_pct / 30) * 50))
            metadata['price_target_upside_pct'] = round(upside_pct, 2)
            metadata['upside_score'] = round(upside_score, 2)

            # Blend: 60% rating consensus, 40% price target upside
            consensus_score = rating_score * 0.60 + upside_score * 0.40
        else:
            consensus_score = rating_score

        return round(consensus_score, 2), metadata

    def calculate_insider_activity_score(self, insider_data: Optional[Dict[str, Any]]) -> Tuple[Optional[float], Dict]:
        """Calculate insider activity score based on net buying/selling dollar value.

        Scoring methodology (dollar-value based):
        - Heavy buying (+$1M) = 100
        - Neutral ($0) = 50
        - Heavy selling (-$1M) = 0

        Uses dollar values instead of share counts for cross-stock comparability.
        """
        if not insider_data:
            return None, {}

        metadata = {}

        # Prefer dollar values for cross-stock comparability
        net_buying_value = insider_data.get('net_buying_value_90d', 0)
        net_buying_shares = insider_data.get('net_buying_90d', 0)

        # Scale by dollar value: $1M net buying = max score
        # This makes the score comparable across high/low priced stocks
        VALUE_SCALE = 1_000_000.0  # $1M for full 50-point swing

        if net_buying_value != 0:
            score = max(0, min(100, 50 + (net_buying_value / VALUE_SCALE) * 50))
        elif net_buying_shares != 0:
            # Fallback to share-based scoring if no dollar values
            insider_scale = self.SCALE_FACTORS['insider_scale']
            score = max(0, min(100, 50 + (net_buying_shares / insider_scale)))
        else:
            score = 50  # Neutral

        metadata['net_buying_90d'] = net_buying_shares
        metadata['net_buying_value_90d'] = net_buying_value

        return score, metadata

    def calculate_institutional_score(self, institutional_data: Optional[Dict[str, Any]]) -> Tuple[Optional[float], Dict]:
        """Calculate institutional ownership score based on holdings changes.

        Scoring methodology:
        - Institutional breadth: log-scaled institution count (40% weight)
        - Ownership level: institutional shares as % of outstanding (30% weight)
        - Holdings change: quarter-over-quarter change (30% weight)
        """
        if not institutional_data or institutional_data.get('num_institutions', 0) == 0:
            return None, {}

        metadata = {}
        num_institutions = institutional_data['num_institutions']
        total_shares = institutional_data.get('total_shares', 0)
        prev_shares = institutional_data.get('prev_shares')
        shares_outstanding = institutional_data.get('shares_outstanding')

        weighted_scores = []

        # Number of institutions (log-scaled for better differentiation)
        # log2(10) ~= 3.3 -> 33, log2(100) ~= 6.6 -> 66, log2(1000) ~= 10 -> 100
        if num_institutions > 0:
            inst_score = min(100, np.log2(num_institutions) * 10)
            weighted_scores.append((inst_score, 0.40))
            metadata['num_institutions'] = num_institutions

        # Institutional ownership as % of shares outstanding
        if shares_outstanding and shares_outstanding > 0 and total_shares > 0:
            ownership_pct = (total_shares / shares_outstanding) * 100
            # 0% = 0, 50% = 75, 80%+ = 100 (diminishing returns above 50%)
            if ownership_pct <= 50:
                ownership_score = ownership_pct * 1.5
            else:
                ownership_score = 75 + min(25, (ownership_pct - 50) * 0.5)
            ownership_score = max(0, min(100, ownership_score))
            weighted_scores.append((ownership_score, 0.30))
            metadata['institutional_ownership_pct'] = round(ownership_pct, 2)

        # Change in holdings (increasing = positive)
        if prev_shares is not None and prev_shares > 0:
            change_pct = ((total_shares - prev_shares) / prev_shares) * 100
            # Normalize: +10% = 100, 0% = 50, -10% = 0
            change_score = max(0, min(100, 50 + (change_pct * 5)))
            weighted_scores.append((change_score, 0.30))
            metadata['holdings_change_pct'] = round(change_pct, 2)

        if not weighted_scores:
            return None, metadata

        total_weight = sum(w for _, w in weighted_scores)
        final_score = sum(s * w for s, w in weighted_scores) / total_weight

        return round(final_score, 2), metadata

    def calculate_intrinsic_value_score(
        self,
        fundamental_metrics: Optional[Dict[str, Any]],
        valuation_data: Optional[Dict[str, Any]]
    ) -> Tuple[Optional[float], Dict]:
        """Calculate intrinsic value score using DCF fair value and Graham Number.

        Uses pre-calculated fair value estimates from fundamental_metrics_extended:
        - DCF upside: margin of safety vs current price (60% weight)
        - Graham Number upside: conservative value vs price (40% weight)

        Scoring:
        - Stock at 50%+ discount to fair value = 100
        - Stock at fair value = 50
        - Stock at 50%+ premium to fair value = 0
        """
        if not fundamental_metrics or not valuation_data:
            return None, {}

        metadata = {}
        scores = []
        stock_price = valuation_data.get('stock_price')

        if not stock_price or stock_price <= 0:
            return None, {}

        # DCF-based upside (pre-calculated as dcf_upside_percent)
        dcf_upside = fundamental_metrics.get('dcf_upside_percent')
        dcf_fair_value = fundamental_metrics.get('dcf_fair_value')

        if dcf_upside is not None and dcf_fair_value is not None and dcf_fair_value > 0:
            # Scale: -50% upside = 0, 0% = 50, +50% = 100
            dcf_score = max(0, min(100, 50 + dcf_upside))
            scores.append(('dcf', dcf_score, 0.60))
            metadata['dcf_fair_value'] = dcf_fair_value
            metadata['dcf_upside_pct'] = dcf_upside

        # Graham Number upside
        graham_number = fundamental_metrics.get('graham_number')
        if graham_number is not None and graham_number > 0:
            graham_upside = ((graham_number / stock_price) - 1) * 100
            # Scale: -50% upside = 0, 0% = 50, +50% = 100
            graham_score = max(0, min(100, 50 + graham_upside))
            scores.append(('graham', graham_score, 0.40))
            metadata['graham_number'] = graham_number
            metadata['graham_upside_pct'] = round(graham_upside, 2)

        if not scores:
            return None, metadata

        # Weighted average of available methods
        total_weight = sum(w for _, _, w in scores)
        weighted_score = sum(s * w for _, s, w in scores) / total_weight

        return round(weighted_score, 2), metadata

    async def calculate_ic_score(
        self,
        ticker: str,
        sector: Optional[str]
    ) -> Optional[Dict[str, Any]]:
        """Calculate complete IC Score for a stock.

        v2.1 Features:
        - Sector-relative scoring using percentiles
        - Lifecycle-aware weight adjustments
        - Enhanced metadata with sector context

        Args:
            ticker: Stock ticker symbol.
            sector: Company sector (GICS classification).

        Returns:
            IC Score data dictionary or None.
        """
        try:
            # Fetch all data sources
            financial_data = await self.fetch_financial_data(ticker)
            fundamental_metrics = await self.fetch_fundamental_metrics(ticker)
            valuation_data = await self.fetch_valuation_data(ticker)
            technical_data = await self.fetch_technical_data(ticker)
            insider_data = await self.fetch_insider_data(ticker)
            news_data = await self.fetch_news_sentiment_data(ticker)
            analyst_data = await self.fetch_analyst_data(ticker)
            institutional_data = await self.fetch_institutional_data(ticker)

            # Determine lifecycle stage and adjust weights
            lifecycle_stage = None
            weights_to_use = self.WEIGHTS  # Default weights

            if self._lifecycle_classifier:
                # Prefer fundamental_metrics for lifecycle classification (has pre-calculated values)
                lifecycle_data = {}
                if fundamental_metrics:
                    lifecycle_data = {
                        'revenue_growth_yoy': fundamental_metrics.get('revenue_growth_yoy'),
                        'net_margin': fundamental_metrics.get('net_margin'),
                        'pe_ratio': valuation_data.get('pe_ratio') if valuation_data else None,
                        'market_cap': valuation_data.get('market_cap') if valuation_data else None,
                    }
                elif financial_data:
                    latest = financial_data.get('latest', {})
                    lifecycle_data = {
                        'revenue_growth_yoy': latest.get('revenue_growth_yoy'),
                        'net_margin': latest.get('net_margin'),
                        'pe_ratio': valuation_data.get('pe_ratio') if valuation_data else None,
                        'market_cap': valuation_data.get('market_cap') if valuation_data else None,
                    }

                if lifecycle_data:
                    classification = self._lifecycle_classifier.classify(lifecycle_data)
                    lifecycle_stage = classification.stage.value
                    weights_to_use = classification.adjusted_weights
                    logger.debug(f"{ticker}: Lifecycle={lifecycle_stage}, weights adjusted")

            # Calculate individual factor scores
            factor_scores = {}
            factor_metadata = {}

            # Value score (uses sector percentiles in v2.1)
            value_score, value_meta = await self.calculate_value_score(valuation_data, sector)
            if value_score is not None:
                factor_scores['value'] = value_score
                factor_metadata['value'] = value_meta

            # Growth score (uses pre-calculated metrics from fundamental_metrics_extended)
            growth_score, growth_meta = await self.calculate_growth_score(fundamental_metrics, financial_data, sector)
            if growth_score is not None:
                factor_scores['growth'] = growth_score
                factor_metadata['growth'] = growth_meta

            # Profitability score (uses pre-calculated metrics from fundamental_metrics_extended)
            profit_score, profit_meta = await self.calculate_profitability_score(fundamental_metrics, financial_data, sector)
            if profit_score is not None:
                factor_scores['profitability'] = profit_score
                factor_metadata['profitability'] = profit_meta

            # Financial health score (uses sector-relative scoring in v2.1)
            health_score, health_meta = await self.calculate_financial_health_score(fundamental_metrics, financial_data, sector)
            if health_score is not None:
                factor_scores['financial_health'] = health_score
                factor_metadata['financial_health'] = health_meta

            # Momentum score
            momentum_score, momentum_meta = self.calculate_momentum_score(technical_data)
            if momentum_score is not None:
                factor_scores['momentum'] = momentum_score
                factor_metadata['momentum'] = momentum_meta

            # Technical score
            tech_score, tech_meta = self.calculate_technical_score(technical_data)
            if tech_score is not None:
                factor_scores['technical'] = tech_score
                factor_metadata['technical'] = tech_meta

            # News sentiment score
            news_score, news_meta = self.calculate_news_sentiment_score(news_data)
            if news_score is not None:
                factor_scores['news_sentiment'] = news_score
                factor_metadata['news_sentiment'] = news_meta

            # Analyst consensus score (now includes price target upside)
            analyst_score, analyst_meta = self.calculate_analyst_consensus_score(analyst_data, valuation_data)
            if analyst_score is not None:
                factor_scores['analyst_consensus'] = analyst_score
                factor_metadata['analyst_consensus'] = analyst_meta

            # Insider activity score
            insider_score, insider_meta = self.calculate_insider_activity_score(insider_data)
            if insider_score is not None:
                factor_scores['insider_activity'] = insider_score
                factor_metadata['insider_activity'] = insider_meta

            # Institutional score
            institutional_score, institutional_meta = self.calculate_institutional_score(institutional_data)
            if institutional_score is not None:
                factor_scores['institutional'] = institutional_score
                factor_metadata['institutional'] = institutional_meta

            # Intrinsic value score (uses DCF fair value and Graham Number)
            intrinsic_score, intrinsic_meta = self.calculate_intrinsic_value_score(fundamental_metrics, valuation_data)
            if intrinsic_score is not None:
                factor_scores['intrinsic_value'] = intrinsic_score
                factor_metadata['intrinsic_value'] = intrinsic_meta

            # Phase 2: Earnings Revisions factor
            if self._earnings_revisions_calc:
                earnings_rev_result = await self._earnings_revisions_calc.calculate(ticker)
                if earnings_rev_result:
                    factor_scores['earnings_revisions'] = earnings_rev_result.score
                    factor_metadata['earnings_revisions'] = {
                        'score': earnings_rev_result.score,
                        'magnitude_score': earnings_rev_result.magnitude_score,
                        'breadth_score': earnings_rev_result.breadth_score,
                        'recency_score': earnings_rev_result.recency_score,
                        **earnings_rev_result.metrics
                    }

            # Phase 2: Historical Valuation factor
            if self._historical_valuation_calc:
                hist_val_result = await self._historical_valuation_calc.calculate(ticker)
                if hist_val_result:
                    factor_scores['historical_value'] = hist_val_result.score
                    factor_metadata['historical_value'] = {
                        'score': hist_val_result.score,
                        'pe_percentile': hist_val_result.pe_percentile,
                        'ps_percentile': hist_val_result.ps_percentile,
                        **hist_val_result.metrics
                    }

            # Phase 2: Dividend Quality factor (optional, for income mode)
            if self._dividend_quality_calc and (self.income_mode or self.USE_DIVIDEND_QUALITY):
                div_quality_result = await self._dividend_quality_calc.calculate(ticker)
                if div_quality_result and div_quality_result.is_dividend_payer:
                    factor_scores['dividend_quality'] = div_quality_result.score
                    factor_metadata['dividend_quality'] = {
                        'score': div_quality_result.score,
                        'yield_score': div_quality_result.yield_score,
                        'payout_score': div_quality_result.payout_score,
                        'growth_score': div_quality_result.growth_score,
                        'streak_score': div_quality_result.streak_score,
                        **div_quality_result.metrics
                    }

            # Calculate data completeness
            data_completeness = (len(factor_scores) / len(self.WEIGHTS)) * 100

            if not factor_scores:
                logger.warning(f"{ticker}: No factor scores calculated")
                return None

            # Enforce minimum data completeness threshold
            if data_completeness < self.MIN_DATA_COMPLETENESS:
                logger.warning(
                    f"{ticker}: Data completeness {data_completeness:.1f}% below minimum "
                    f"threshold {self.MIN_DATA_COMPLETENESS}% (factors: {list(factor_scores.keys())})"
                )
                return None

            # Check if core financial factors are present for reliable scoring
            available_core = set(factor_scores.keys()) & self.CORE_FACTORS
            if len(available_core) < 2:
                logger.warning(
                    f"{ticker}: Insufficient core factors ({available_core}), need at least 2 of {self.CORE_FACTORS}"
                )
                return None

            # Calculate weighted overall score using lifecycle-adjusted weights
            # Map factor names to weights (supports both v2.0 legacy and v2.1)
            smart_money_weight = weights_to_use.get('smart_money', 0.10)
            technical_weight = weights_to_use.get('technical', 0.07)
            factor_weight_mapping = {
                'value': weights_to_use.get('value', 0.12),
                'growth': weights_to_use.get('growth', 0.13),
                'profitability': weights_to_use.get('profitability', 0.12),
                'financial_health': weights_to_use.get('financial_health', 0.10),
                'momentum': weights_to_use.get('momentum', 0.10),
                'technical': technical_weight * 0.60,             # 60% of technical allocation
                'news_sentiment': technical_weight * 0.40,        # 40% of technical allocation
                # Smart money sub-factors: split smart_money weight among the three
                'analyst_consensus': smart_money_weight * 0.4,    # 40% of smart money
                'insider_activity': smart_money_weight * 0.3,     # 30% of smart money
                'institutional': smart_money_weight * 0.3,        # 30% of smart money
                # Intrinsic value factor (was missing before - 10% weight)
                'intrinsic_value': weights_to_use.get('intrinsic_value', 0.10),
                # Phase 2 factors
                'earnings_revisions': weights_to_use.get('earnings_revisions', 0.08),
                'historical_value': weights_to_use.get('historical_value', 0.08),
                'dividend_quality': weights_to_use.get('dividend_quality', 0.05) if self.income_mode else 0,
            }

            total_weight = sum(
                factor_weight_mapping.get(factor, 0)
                for factor in factor_scores.keys()
            )
            overall_score = sum(
                float(factor_scores[factor]) * factor_weight_mapping.get(factor, 0)
                for factor in factor_scores.keys()
            ) / total_weight if total_weight > 0 else 0

            # Determine rating
            rating = 'Sell'
            for rating_name, threshold in sorted(self.RATING_THRESHOLDS.items(), key=lambda x: -x[1]):
                if overall_score >= threshold:
                    rating = rating_name
                    break

            # Determine confidence level
            if data_completeness >= 90:
                confidence = 'High'
            elif data_completeness >= 70:
                confidence = 'Medium'
            else:
                confidence = 'Low'

            # v2.1: Get sector rank
            sector_rank = None
            sector_total = None
            if self._sector_calculator and sector:
                sector_rank, sector_total = await self._sector_calculator.get_sector_rank(
                    sector, ticker, overall_score
                )

            # Phase 3: Apply score stabilization
            raw_score = overall_score
            previous_score = None
            smoothing_applied = False
            stabilization_events = []

            if self._score_stabilizer:
                # Detect any reset events
                detected_events = await self._score_stabilizer.detect_events(ticker)
                stabilization_events = detected_events

                # Get previous score and apply stabilization
                stabilization_result = await self._score_stabilizer.stabilize(
                    ticker=ticker,
                    new_score=overall_score,
                    events=detected_events
                )

                overall_score = stabilization_result.final_score
                previous_score = stabilization_result.previous_score
                smoothing_applied = stabilization_result.smoothing_applied

                if stabilization_result.previous_score:
                    logger.debug(
                        f"{ticker}: Score {stabilization_result.previous_score:.1f} -> {overall_score:.1f} "
                        f"(raw: {raw_score:.1f}, smoothing: {smoothing_applied})"
                    )

            # Build result with v2.1 enhancements
            result = {
                'ticker': ticker,
                'date': date.today(),
                'overall_score': round(overall_score, 2),
                'previous_score': round(previous_score, 2) if previous_score else None,
                'value_score': round(factor_scores.get('value'), 2) if 'value' in factor_scores else None,
                'growth_score': round(factor_scores.get('growth'), 2) if 'growth' in factor_scores else None,
                'profitability_score': round(factor_scores.get('profitability'), 2) if 'profitability' in factor_scores else None,
                'financial_health_score': round(factor_scores.get('financial_health'), 2) if 'financial_health' in factor_scores else None,
                'momentum_score': round(factor_scores.get('momentum'), 2) if 'momentum' in factor_scores else None,
                'analyst_consensus_score': round(factor_scores.get('analyst_consensus'), 2) if 'analyst_consensus' in factor_scores else None,
                'insider_activity_score': round(factor_scores.get('insider_activity'), 2) if 'insider_activity' in factor_scores else None,
                'institutional_score': round(factor_scores.get('institutional'), 2) if 'institutional' in factor_scores else None,
                'news_sentiment_score': round(factor_scores.get('news_sentiment'), 2) if 'news_sentiment' in factor_scores else None,
                'technical_score': round(factor_scores.get('technical'), 2) if 'technical' in factor_scores else None,
                'rating': rating,
                'sector_percentile': round((sector_total - sector_rank + 1) / sector_total * 100, 1) if sector_rank and sector_total else None,
                'confidence_level': confidence,
                'data_completeness': round(data_completeness, 2),
                # v2.1 new fields
                'lifecycle_stage': lifecycle_stage,
                'sector_rank': sector_rank,
                'sector_total': sector_total,
                'calculation_metadata': {
                    'factors': factor_metadata,
                    'weights_used': {k: round(factor_weight_mapping.get(k, 0), 4) for k in factor_scores.keys()},
                    'lifecycle_stage': lifecycle_stage,
                    'scoring_version': '2.2',
                    'income_mode': self.income_mode,
                    'calculated_at': datetime.now().isoformat(),
                    # Phase 3: Stabilization metadata
                    'raw_score': round(raw_score, 2),
                    'smoothing_applied': smoothing_applied,
                    'stabilization_events': [
                        {'type': e.event_type.value, 'date': str(e.event_date), 'description': e.description}
                        for e in stabilization_events
                    ] if stabilization_events else [],
                }
            }

            # Phase 3: Add peer comparison data
            if self._peer_comparison:
                try:
                    peer_result = await self._peer_comparison.get_peers(ticker, limit=5)
                    if peer_result:
                        result['peers'] = [
                            {
                                'ticker': p.ticker,
                                'company_name': p.company_name,
                                'ic_score': p.ic_score,
                                'similarity_score': round(p.similarity_score, 3),
                            }
                            for p in peer_result.peers
                        ]
                        result['peer_comparison'] = {
                            'avg_peer_score': round(peer_result.avg_peer_score, 2) if peer_result.avg_peer_score else None,
                            'sector_rank': peer_result.sector_rank,
                            'sector_total': peer_result.sector_total,
                            'vs_peers_delta': round(overall_score - peer_result.avg_peer_score, 2) if peer_result.avg_peer_score else None,
                        }
                except Exception as e:
                    logger.debug(f"{ticker}: Peer comparison error: {e}")

            # Phase 3: Add catalyst data
            if self._catalyst_detector:
                try:
                    catalysts = await self._catalyst_detector.get_catalysts(ticker, limit=5)
                    if catalysts:
                        result['catalysts'] = [
                            {
                                'event_type': c.event_type,
                                'title': c.title,
                                'event_date': str(c.event_date) if c.event_date else None,
                                'icon': c.icon,
                                'impact': c.impact,
                                'confidence': c.confidence,
                                'days_until': c.days_until,
                            }
                            for c in catalysts
                        ]
                except Exception as e:
                    logger.debug(f"{ticker}: Catalyst detection error: {e}")

            # Phase 3: Generate score explanation
            if self._score_explainer:
                try:
                    current_scores = {
                        'overall_score': overall_score,
                        **{f'{k}_score': v for k, v in factor_scores.items()}
                    }
                    explanation = await self._score_explainer.explain_change(
                        ticker=ticker,
                        current_scores=current_scores
                    )
                    if explanation:
                        result['explanation'] = {
                            'summary': explanation.summary,
                            'delta': explanation.delta,
                            'reasons': [
                                {
                                    'factor': r.factor,
                                    'delta': r.delta,
                                    'contribution': r.contribution,
                                    'explanation': r.explanation,
                                }
                                for r in explanation.reasons
                            ],
                            'confidence': {
                                'level': explanation.confidence.level,
                                'percentage': explanation.confidence.percentage,
                                'warnings': explanation.confidence.warnings,
                            }
                        }
                        # Update confidence level from granular confidence
                        result['confidence_level'] = explanation.confidence.level
                except Exception as e:
                    logger.debug(f"{ticker}: Score explanation error: {e}")

            return result

        except Exception as e:
            logger.error(f"{ticker}: Error calculating IC Score: {e}", exc_info=True)
            return None

    # Valid columns in ICScore model (used for filtering score_data before storing)
    IC_SCORE_COLUMNS = {
        'ticker', 'date', 'overall_score',
        'value_score', 'growth_score', 'profitability_score', 'financial_health_score',
        'momentum_score', 'analyst_consensus_score', 'insider_activity_score',
        'institutional_score', 'news_sentiment_score', 'technical_score',
        'rating', 'sector_percentile', 'confidence_level', 'data_completeness',
        'lifecycle_stage', 'raw_score', 'smoothing_applied', 'weights_used',
        'sector_rank', 'sector_total', 'calculation_metadata',
    }

    async def store_ic_score(self, score_data: Dict[str, Any]) -> bool:
        """Store IC Score in database."""
        try:
            # Filter to only include valid columns (exclude non-column keys like 'peers', 'previous_score', etc.)
            filtered_data = {k: v for k, v in score_data.items() if k in self.IC_SCORE_COLUMNS}

            async with self.db.session() as session:
                stmt = pg_insert(ICScore).values(filtered_data)
                stmt = stmt.on_conflict_do_update(
                    index_elements=['ticker', 'date'],
                    set_={k: stmt.excluded[k] for k in filtered_data.keys() if k not in ['ticker', 'date']}
                )

                await session.execute(stmt)
                await session.commit()

                return True

        except Exception as e:
            logger.error(f"Error storing IC Score: {e}", exc_info=True)
            return False

    async def process_stocks(self, stocks: List[Dict[str, Any]], show_progress: bool = True):
        """Process a list of stocks.

        v2.1: Initializes sector percentile calculator and lifecycle classifier
        within a database session context.

        Note: Each stock is processed in its own session to prevent transaction
        errors in one stock from affecting subsequent stocks.
        """
        progress_bar = tqdm(total=len(stocks), desc="Calculating IC Scores") if show_progress else None

        logger.info("Using IC Score v2.1 with sector-relative scoring")

        for stock in stocks:
            ticker = stock['ticker']
            sector = stock.get('sector')

            try:
                # Create a fresh session for each stock to prevent transaction cascade failures
                async with self.db.session() as session:
                    # Initialize v2 components with this session
                    await self._init_v2_components(session)

                    score_data = await self.calculate_ic_score(ticker, sector)

                if score_data:
                    success = await self.store_ic_score(score_data)
                    if success:
                        self.success_count += 1
                    else:
                        self.error_count += 1
                else:
                    self.error_count += 1
            except Exception as e:
                logger.error(f"{ticker}: Unhandled error: {e}")
                self.error_count += 1

            self.processed_count += 1

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
        sector: Optional[str] = None,
        all_stocks: bool = False,
        sp500: bool = False
    ):
        """Run the IC Score calculator."""
        start_time = datetime.now()
        logger.info("=" * 80)
        logger.info("IC Score Calculator Pipeline")
        logger.info("=" * 80)

        if all_stocks:
            limit = None
        elif ticker:
            limit = 1
        elif limit is None:
            limit = 10

        stocks = await self.get_stocks_to_process(
            limit=limit,
            ticker=ticker,
            sector=sector,
            sp500=sp500
        )

        if not stocks:
            logger.warning("No stocks to process")
            return

        logger.info(f"Processing {len(stocks)} stocks...")

        await self.process_stocks(stocks)

        duration = datetime.now() - start_time
        logger.info("=" * 80)
        logger.info("Calculation Complete")
        logger.info("=" * 80)
        logger.info(f"Duration: {duration}")
        logger.info(f"Processed: {self.processed_count}")
        logger.info(f"Success: {self.success_count}")
        logger.info(f"Errors: {self.error_count}")
        logger.info(f"Success Rate: {(self.success_count/self.processed_count*100):.1f}%")


def main():
    """Main entry point for CLI."""
    parser = argparse.ArgumentParser(
        description='Calculate IC Scores for stocks',
        formatter_class=argparse.RawDescriptionHelpFormatter
    )

    parser.add_argument('--limit', type=int, help='Limit number of stocks')
    parser.add_argument('--ticker', type=str, help='Single ticker symbol')
    parser.add_argument('--sector', type=str, help='Filter by sector')
    parser.add_argument('--all', action='store_true', help='Process all stocks')
    parser.add_argument('--sp500', action='store_true', help='Process S&P 500 only')
    parser.add_argument('--income-mode', action='store_true',
                        help='Enable income mode (include Dividend Quality factor)')

    args = parser.parse_args()

    if args.ticker and (args.all or args.limit or args.sector):
        parser.error("--ticker cannot be used with other filters")

    calculator = ICScoreCalculator(income_mode=args.income_mode)
    asyncio.run(calculator.run(
        limit=args.limit,
        ticker=args.ticker,
        sector=args.sector,
        all_stocks=args.all,
        sp500=args.sp500
    ))


if __name__ == '__main__':
    main()
