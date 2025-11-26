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
    """Calculator for InvestorCenter proprietary IC Scores."""

    # Factor weights (must sum to 1.0)
    WEIGHTS = {
        'value': 0.12,
        'growth': 0.15,
        'profitability': 0.12,
        'financial_health': 0.10,
        'momentum': 0.08,
        'analyst_consensus': 0.10,
        'insider_activity': 0.08,
        'institutional': 0.10,
        'news_sentiment': 0.07,
        'technical': 0.08,
    }

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

    def __init__(self):
        """Initialize the IC Score calculator."""
        self.db = get_database()
        self.sector_percentiles = {}  # Cache for sector percentiles

        # Track progress
        self.processed_count = 0
        self.success_count = 0
        self.error_count = 0

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
                    SELECT ticker, sector
                    FROM stocks
                    WHERE ticker = :ticker
                """)
                result = await session.execute(query, {"ticker": ticker.upper()})
            else:
                where_clauses = ["ticker NOT LIKE '%-%'", "is_active = true"]
                params = {}

                if sector:
                    where_clauses.append("sector = :sector")
                    params['sector'] = sector

                if sp500:
                    where_clauses.append("is_sp500 = true")

                query_str = f"""
                    SELECT ticker, sector
                    FROM stocks
                    WHERE {' AND '.join(where_clauses)}
                    ORDER BY ticker
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
            query = text("""
                SELECT *
                FROM financials
                WHERE ticker = :ticker
                ORDER BY period_end_date DESC
                LIMIT 20
            """)
            result = await session.execute(query, {"ticker": ticker})
            rows = result.fetchall()

            if not rows:
                return None

            # Get latest quarterly and annual data
            latest = rows[0]._asdict() if rows else {}
            historical = [row._asdict() for row in rows]

            return {'latest': latest, 'historical': historical}

    async def fetch_technical_data(self, ticker: str) -> Optional[Dict[str, Any]]:
        """Fetch latest technical indicators for a stock."""
        async with self.db.session() as session:
            query = text("""
                SELECT indicator_name, value
                FROM technical_indicators
                WHERE ticker = :ticker
                  AND time >= NOW() - INTERVAL '7 days'
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
            # Get last 90 days of insider trades
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

            net_buying = 0
            for row in rows:
                if row[0] and 'buy' in row[0].lower():
                    net_buying += row[1] or 0
                elif row[0] and 'sell' in row[0].lower():
                    net_buying -= row[1] or 0

            return {'net_buying_90d': net_buying}

    async def fetch_news_sentiment_data(self, ticker: str) -> Optional[Dict[str, Any]]:
        """Fetch news sentiment data for a stock."""
        async with self.db.session() as session:
            # Get articles from last 30 days
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

            return {
                'article_count': row[0],
                'avg_sentiment': float(row[1]) if row[1] else 0,
                'positive_count': row[2] or 0,
                'negative_count': row[3] or 0,
                'neutral_count': row[4] or 0
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

            return {
                'total_shares': int(row[0]) if row[0] else 0,
                'total_value': float(row[1]) if row[1] else 0,
                'num_institutions': row[2] or 0,
                'prev_shares': int(prev_shares) if prev_shares else None
            }

    def calculate_value_score(self, financial_data: Dict[str, Any], sector_data: Dict) -> Tuple[Optional[float], Dict]:
        """Calculate value score based on P/E, P/B, P/S ratios vs sector median.

        Scoring methodology:
        - Lower valuation ratios = higher value scores
        - Scores normalized to 0-100 scale centered on benchmark values
        - P/E benchmark: ~15 (S&P 500 historical average)
        - P/B, P/S benchmarks: ~2 (typical fair value)
        """
        if not financial_data:
            return None, {}

        latest = financial_data.get('latest', {})
        metadata = {}

        # Get valuation metrics
        pe = latest.get('pe_ratio')
        pb = latest.get('pb_ratio')
        ps = latest.get('ps_ratio')

        if not any([pe, pb, ps]):
            return None, metadata

        # In production, compare against sector medians
        # For now, use simple percentile logic (lower is better for value)
        scores = []
        if pe and pe > 0:
            # Lower P/E is better, normalize to 0-100 scale
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

    def calculate_growth_score(self, financial_data: Dict[str, Any]) -> Tuple[Optional[float], Dict]:
        """Calculate growth score based on revenue, EPS, FCF growth.

        Scoring methodology:
        - 0% growth = 50 (neutral)
        - +20% growth = 100 (maximum score)
        - -20% growth = 0 (minimum score)
        """
        if not financial_data or len(financial_data.get('historical', [])) < 4:
            return None, {}

        historical = financial_data['historical']
        metadata = {}
        growth_scale = self.SCALE_FACTORS['growth_scale']

        # Calculate year-over-year growth rates
        scores = []

        # Revenue growth
        if historical[0].get('revenue') and historical[3].get('revenue'):
            rev_growth = ((historical[0]['revenue'] / historical[3]['revenue']) - 1) * 100
            # Normalize: 0% = 50, 20% = 100, -20% = 0
            rev_score = max(0, min(100, 50 + rev_growth * growth_scale))
            scores.append(rev_score)
            metadata['revenue_growth_yoy'] = rev_growth

        # EPS growth
        if historical[0].get('eps_diluted') and historical[3].get('eps_diluted'):
            if historical[3]['eps_diluted'] > 0:
                eps_growth = ((historical[0]['eps_diluted'] / historical[3]['eps_diluted']) - 1) * 100
                eps_score = max(0, min(100, 50 + eps_growth * growth_scale))
                scores.append(eps_score)
                metadata['eps_growth_yoy'] = eps_growth

        if not scores:
            return None, metadata

        return np.mean(scores), metadata

    def calculate_profitability_score(self, financial_data: Dict[str, Any]) -> Tuple[Optional[float], Dict]:
        """Calculate profitability score based on margins, ROE, ROA.

        Scoring methodology:
        - 0% margin/ROE = 0, 20% = 100
        - 0% ROA = 0, 10% = 100 (ROA typically lower than ROE)
        """
        if not financial_data:
            return None, {}

        latest = financial_data.get('latest', {})
        metadata = {}
        scores = []

        # Net margin
        if latest.get('net_margin'):
            margin = float(latest['net_margin'])
            margin_scale = self.SCALE_FACTORS['margin_scale']
            margin_score = max(0, min(100, margin * margin_scale))
            scores.append(margin_score)
            metadata['net_margin'] = margin

        # ROE
        if latest.get('roe'):
            roe = float(latest['roe'])
            roe_scale = self.SCALE_FACTORS['roe_scale']
            roe_score = max(0, min(100, roe * roe_scale))
            scores.append(roe_score)
            metadata['roe'] = roe

        # ROA
        if latest.get('roa'):
            roa = float(latest['roa'])
            roa_scale = self.SCALE_FACTORS['roa_scale']
            roa_score = max(0, min(100, roa * roa_scale))
            scores.append(roa_score)
            metadata['roa'] = roa

        if not scores:
            return None, metadata

        return np.mean(scores), metadata

    def calculate_financial_health_score(self, financial_data: Dict[str, Any]) -> Tuple[Optional[float], Dict]:
        """Calculate financial health score based on D/E, current ratio.

        Scoring methodology:
        - D/E: 0 = 100 (no debt), 2+ = 0 (high debt)
        - Current ratio: 2.0 = 100 (optimal), 0 or 5+ = 0 (extremes)
        """
        if not financial_data:
            return None, {}

        latest = financial_data.get('latest', {})
        metadata = {}
        scores = []

        # Debt to equity (lower is better)
        if latest.get('debt_to_equity') is not None:
            de = float(latest['debt_to_equity'])
            de_scale = self.SCALE_FACTORS['de_scale']
            de_score = max(0, min(100, 100 - de * de_scale))
            scores.append(de_score)
            metadata['debt_to_equity'] = de

        # Current ratio (optimal around 1.5-2.0)
        if latest.get('current_ratio'):
            cr = float(latest['current_ratio'])
            cr_optimal = self.SCALE_FACTORS['cr_optimal']
            cr_scale = self.SCALE_FACTORS['cr_scale']
            cr_score = max(0, min(100, 100 - abs(cr - cr_optimal) * cr_scale))
            scores.append(cr_score)
            metadata['current_ratio'] = cr

        if not scores:
            return None, metadata

        return np.mean(scores), metadata

    def calculate_momentum_score(self, technical_data: Dict[str, Any]) -> Tuple[Optional[float], Dict]:
        """Calculate momentum score based on price returns.

        Scoring methodology:
        - -20% return = 0, 0% = 50, +20% = 100
        """
        if not technical_data:
            return None, {}

        metadata = {}
        scores = []
        return_scale = self.SCALE_FACTORS['return_scale']

        # Use various period returns
        for period in ['1m_return', '3m_return', '6m_return', '12m_return']:
            if period in technical_data:
                ret = technical_data[period]
                # Normalize: -20% = 0, 0% = 50, 20% = 100
                score = max(0, min(100, 50 + ret * return_scale))
                scores.append(score)
                metadata[period] = ret

        if not scores:
            return None, metadata

        return np.mean(scores), metadata

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
        """Calculate news sentiment score based on article sentiment analysis."""
        if not news_data or news_data.get('article_count', 0) == 0:
            return None, {}

        metadata = {}
        scores = []

        # Average sentiment score (already 0-100 from sentiment analysis)
        avg_sentiment = news_data.get('avg_sentiment', 0)
        sentiment_score = max(0, min(100, avg_sentiment))
        scores.append(sentiment_score)
        metadata['avg_sentiment'] = avg_sentiment

        # Positive vs negative ratio
        positive_count = news_data.get('positive_count', 0)
        negative_count = news_data.get('negative_count', 0)
        total_articles = news_data.get('article_count', 0)

        if total_articles > 0:
            positive_ratio = (positive_count / total_articles) * 100
            metadata['positive_ratio'] = positive_ratio
            metadata['article_count'] = total_articles

        if not scores:
            return None, metadata

        return np.mean(scores), metadata

    def calculate_analyst_consensus_score(self, analyst_data: Optional[Dict[str, Any]]) -> Tuple[Optional[float], Dict]:
        """Calculate analyst consensus score based on buy/hold/sell ratings."""
        if not analyst_data or analyst_data.get('total_analysts', 0) == 0:
            return None, {}

        metadata = {}
        total_analysts = analyst_data['total_analysts']
        buy_count = analyst_data.get('buy_count', 0)
        hold_count = analyst_data.get('hold_count', 0)
        sell_count = analyst_data.get('sell_count', 0)

        # Calculate consensus score: 100 = all buy, 50 = all hold, 0 = all sell
        if total_analysts > 0:
            # Weight: Buy = 100, Hold = 50, Sell = 0
            weighted_score = ((buy_count * 100) + (hold_count * 50) + (sell_count * 0)) / total_analysts
            consensus_score = max(0, min(100, weighted_score))
        else:
            consensus_score = 50  # Neutral if no clear ratings

        metadata['total_analysts'] = total_analysts
        metadata['buy_count'] = buy_count
        metadata['hold_count'] = hold_count
        metadata['sell_count'] = sell_count
        metadata['avg_price_target'] = analyst_data.get('avg_price_target')

        return consensus_score, metadata

    def calculate_insider_activity_score(self, insider_data: Optional[Dict[str, Any]]) -> Tuple[Optional[float], Dict]:
        """Calculate insider activity score based on net buying/selling.

        Scoring methodology:
        - Heavy buying (+100k shares) = 100
        - Neutral (0 shares) = 50
        - Heavy selling (-100k shares) = 0
        """
        if not insider_data:
            return None, {}

        metadata = {}
        net_buying = insider_data.get('net_buying_90d', 0)
        insider_scale = self.SCALE_FACTORS['insider_scale']

        # Normalize: Heavy buying = 100, neutral = 50, heavy selling = 0
        if net_buying > 0:
            # Positive net buying
            score = min(100, 50 + (net_buying / insider_scale))
        elif net_buying < 0:
            # Net selling
            score = max(0, 50 + (net_buying / insider_scale))
        else:
            score = 50  # Neutral

        metadata['net_buying_90d'] = net_buying

        return score, metadata

    def calculate_institutional_score(self, institutional_data: Optional[Dict[str, Any]]) -> Tuple[Optional[float], Dict]:
        """Calculate institutional ownership score based on holdings changes."""
        if not institutional_data or institutional_data.get('num_institutions', 0) == 0:
            return None, {}

        metadata = {}
        num_institutions = institutional_data['num_institutions']
        total_shares = institutional_data.get('total_shares', 0)
        prev_shares = institutional_data.get('prev_shares')

        scores = []

        # Number of institutions (more is better, up to a point)
        # Normalize: 50 institutions = 50, 100+ = 100
        inst_score = min(100, (num_institutions / 100) * 100)
        scores.append(inst_score)
        metadata['num_institutions'] = num_institutions

        # Change in holdings (increasing = positive)
        if prev_shares is not None and prev_shares > 0:
            change_pct = ((total_shares - prev_shares) / prev_shares) * 100
            # Normalize: +10% = 100, 0% = 50, -10% = 0
            change_score = max(0, min(100, 50 + (change_pct * 5)))
            scores.append(change_score)
            metadata['holdings_change_pct'] = change_pct

        if not scores:
            return None, metadata

        return np.mean(scores), metadata

    async def calculate_ic_score(
        self,
        ticker: str,
        sector: Optional[str]
    ) -> Optional[Dict[str, Any]]:
        """Calculate complete IC Score for a stock.

        Args:
            ticker: Stock ticker symbol.
            sector: Company sector.

        Returns:
            IC Score data dictionary or None.
        """
        try:
            # Fetch all data sources
            financial_data = await self.fetch_financial_data(ticker)
            technical_data = await self.fetch_technical_data(ticker)
            insider_data = await self.fetch_insider_data(ticker)
            news_data = await self.fetch_news_sentiment_data(ticker)
            analyst_data = await self.fetch_analyst_data(ticker)
            institutional_data = await self.fetch_institutional_data(ticker)

            # Calculate individual factor scores
            factor_scores = {}
            factor_metadata = {}

            # Value score
            value_score, value_meta = self.calculate_value_score(financial_data, {})
            if value_score is not None:
                factor_scores['value'] = value_score
                factor_metadata['value'] = value_meta

            # Growth score
            growth_score, growth_meta = self.calculate_growth_score(financial_data)
            if growth_score is not None:
                factor_scores['growth'] = growth_score
                factor_metadata['growth'] = growth_meta

            # Profitability score
            profit_score, profit_meta = self.calculate_profitability_score(financial_data)
            if profit_score is not None:
                factor_scores['profitability'] = profit_score
                factor_metadata['profitability'] = profit_meta

            # Financial health score
            health_score, health_meta = self.calculate_financial_health_score(financial_data)
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

            # Analyst consensus score
            analyst_score, analyst_meta = self.calculate_analyst_consensus_score(analyst_data)
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

            # Calculate weighted overall score (only for available factors)
            total_weight = sum(self.WEIGHTS[factor] for factor in factor_scores.keys())
            overall_score = sum(
                float(factor_scores[factor]) * self.WEIGHTS[factor]
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

            # Build result
            result = {
                'ticker': ticker,
                'date': date.today(),
                'overall_score': round(overall_score, 2),
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
                'sector_percentile': None,  # Would calculate from sector distribution
                'confidence_level': confidence,
                'data_completeness': round(data_completeness, 2),
                'calculation_metadata': {
                    'factors': factor_metadata,
                    'weights_used': {k: self.WEIGHTS[k] for k in factor_scores.keys()},
                    'calculated_at': datetime.now().isoformat(),
                }
            }

            return result

        except Exception as e:
            logger.error(f"{ticker}: Error calculating IC Score: {e}", exc_info=True)
            return None

    async def store_ic_score(self, score_data: Dict[str, Any]) -> bool:
        """Store IC Score in database."""
        try:
            async with self.db.session() as session:
                stmt = pg_insert(ICScore).values(score_data)
                stmt = stmt.on_conflict_do_update(
                    index_elements=['ticker', 'date'],
                    set_={k: stmt.excluded[k] for k in score_data.keys() if k not in ['ticker', 'date']}
                )

                await session.execute(stmt)
                await session.commit()

                return True

        except Exception as e:
            logger.error(f"Error storing IC Score: {e}", exc_info=True)
            return False

    async def process_stocks(self, stocks: List[Dict[str, Any]], show_progress: bool = True):
        """Process a list of stocks."""
        progress_bar = tqdm(total=len(stocks), desc="Calculating IC Scores") if show_progress else None

        for stock in stocks:
            ticker = stock['ticker']
            sector = stock.get('sector')

            score_data = await self.calculate_ic_score(ticker, sector)

            if score_data:
                success = await self.store_ic_score(score_data)
                if success:
                    self.success_count += 1
                else:
                    self.error_count += 1
            else:
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

    args = parser.parse_args()

    if args.ticker and (args.all or args.limit or args.sector):
        parser.error("--ticker cannot be used with other filters")

    calculator = ICScoreCalculator()
    asyncio.run(calculator.run(
        limit=args.limit,
        ticker=args.ticker,
        sector=args.sector,
        all_stocks=args.all,
        sp500=args.sp500
    ))


if __name__ == '__main__':
    main()
