#!/usr/bin/env python3
"""
IC Score Validation Script: Current vs Proposed Design

Compares the current IC Score calculation with the proposed sector-relative,
lifecycle-aware design for 10 diverse stocks.

Usage:
    python validate_new_design.py
"""

import asyncio
import sys
from pathlib import Path
from typing import Dict, Any, Optional, List, Tuple
from datetime import datetime, timedelta
from decimal import Decimal

sys.path.insert(0, str(Path(__file__).parent.parent))

import numpy as np
from sqlalchemy import text
from database.database import get_database

# 10 diverse stocks for validation
VALIDATION_STOCKS = [
    # Large-cap Tech (Growth)
    {"ticker": "AAPL", "name": "Apple", "sector": "Technology", "type": "Mature Tech"},
    {"ticker": "NVDA", "name": "NVIDIA", "sector": "Technology", "type": "Hypergrowth"},
    # Value / Financials
    {"ticker": "JPM", "name": "JPMorgan", "sector": "Financial Services", "type": "Value"},
    {"ticker": "BRK.B", "name": "Berkshire", "sector": "Financial Services", "type": "Value"},
    # Healthcare
    {"ticker": "JNJ", "name": "Johnson & Johnson", "sector": "Healthcare", "type": "Defensive"},
    {"ticker": "UNH", "name": "UnitedHealth", "sector": "Healthcare", "type": "Growth"},
    # Consumer
    {"ticker": "AMZN", "name": "Amazon", "sector": "Consumer Cyclical", "type": "Growth"},
    {"ticker": "WMT", "name": "Walmart", "sector": "Consumer Defensive", "type": "Value"},
    # Energy / Utilities
    {"ticker": "XOM", "name": "Exxon Mobil", "sector": "Energy", "type": "Value/Dividend"},
    # Industrial
    {"ticker": "CAT", "name": "Caterpillar", "sector": "Industrials", "type": "Cyclical"},
]


class ICScoreValidator:
    """Validates IC Score current vs proposed methodology."""

    # Current weights
    CURRENT_WEIGHTS = {
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

    # Proposed weights (reorganized)
    PROPOSED_WEIGHTS = {
        # Quality (40%)
        'growth': 0.15,
        'profitability': 0.13,
        'financial_health': 0.12,
        # Valuation (30%)
        'value': 0.15,
        'intrinsic_value': 0.15,
        # Signals (30%)
        'smart_money': 0.12,  # Combines analyst + insider + institutional
        'momentum': 0.10,
        'technical': 0.08,
        'sentiment': 0.08,  # news + earnings revisions
    }

    # Lifecycle weight adjustments
    LIFECYCLE_ADJUSTMENTS = {
        'hypergrowth': {'growth': 1.3, 'profitability': 0.7, 'value': 0.6, 'intrinsic_value': 0.6},
        'growth': {'growth': 1.2, 'profitability': 0.9, 'value': 0.8},
        'mature': {'profitability': 1.1, 'value': 1.1, 'financial_health': 1.1},
        'value': {'value': 1.3, 'profitability': 1.2, 'growth': 0.7},
        'turnaround': {'financial_health': 1.3, 'momentum': 1.2, 'growth': 0.8},
    }

    # Current methodology benchmarks
    VALUATION_BENCHMARKS = {
        'pe_ratio': 15.0,
        'pb_ratio': 2.0,
        'ps_ratio': 2.0,
    }

    def __init__(self):
        self.db = get_database()
        self.sector_stats = {}  # Cache sector statistics

    async def fetch_stock_data(self, ticker: str) -> Dict[str, Any]:
        """Fetch all data needed for IC Score calculation."""
        data = {'ticker': ticker}

        async with self.db.session() as session:
            # Fetch valuation ratios
            result = await session.execute(text("""
                SELECT ttm_pe_ratio, ttm_pb_ratio, ttm_ps_ratio, stock_price, ttm_market_cap
                FROM valuation_ratios
                WHERE ticker = :ticker
                ORDER BY calculation_date DESC LIMIT 1
            """), {"ticker": ticker})
            row = result.fetchone()
            if row:
                data['pe_ratio'] = float(row[0]) if row[0] else None
                data['pb_ratio'] = float(row[1]) if row[1] else None
                data['ps_ratio'] = float(row[2]) if row[2] else None
                data['price'] = float(row[3]) if row[3] else None
                data['market_cap'] = float(row[4]) if row[4] else None

            # Fetch financials
            result = await session.execute(text("""
                SELECT revenue, net_income, eps_diluted, net_margin, roe, roa,
                       debt_to_equity, current_ratio, free_cash_flow, gross_margin,
                       operating_margin
                FROM financials
                WHERE ticker = :ticker
                ORDER BY period_end_date DESC LIMIT 5
            """), {"ticker": ticker})
            rows = result.fetchall()
            if rows:
                latest = rows[0]
                data['revenue'] = float(latest[0]) if latest[0] else None
                data['net_income'] = float(latest[1]) if latest[1] else None
                data['eps'] = float(latest[2]) if latest[2] else None
                data['net_margin'] = float(latest[3]) if latest[3] else None
                data['roe'] = float(latest[4]) if latest[4] else None
                data['roa'] = float(latest[5]) if latest[5] else None
                data['debt_to_equity'] = float(latest[6]) if latest[6] else None
                data['current_ratio'] = float(latest[7]) if latest[7] else None
                data['fcf'] = float(latest[8]) if latest[8] else None
                data['gross_margin'] = float(latest[9]) if latest[9] else None
                data['operating_margin'] = float(latest[10]) if latest[10] else None

                # Calculate YoY growth if we have historical data
                if len(rows) >= 4:
                    prior = rows[3]
                    if latest[0] and prior[0] and float(prior[0]) > 0:
                        data['revenue_growth'] = ((float(latest[0]) / float(prior[0])) - 1) * 100
                    if latest[2] and prior[2] and float(prior[2]) > 0:
                        data['eps_growth'] = ((float(latest[2]) / float(prior[2])) - 1) * 100

            # Fetch technical indicators
            result = await session.execute(text("""
                SELECT indicator_name, value
                FROM technical_indicators
                WHERE ticker = :ticker AND time >= NOW() - INTERVAL '14 days'
                ORDER BY time DESC LIMIT 50
            """), {"ticker": ticker})
            indicators = {}
            for row in result.fetchall():
                if row[0] not in indicators:
                    indicators[row[0]] = float(row[1])
            data['rsi'] = indicators.get('rsi')
            data['macd_histogram'] = indicators.get('macd_histogram')
            data['return_1m'] = indicators.get('1m_return')
            data['return_3m'] = indicators.get('3m_return')
            data['return_6m'] = indicators.get('6m_return')
            data['return_12m'] = indicators.get('12m_return')

            # Fetch analyst data
            result = await session.execute(text("""
                SELECT rating, price_target
                FROM analyst_ratings
                WHERE ticker = :ticker AND rating_date >= NOW() - INTERVAL '90 days'
            """), {"ticker": ticker})
            ratings = result.fetchall()
            if ratings:
                buy_count = sum(1 for r in ratings if r[0] and any(x in r[0].lower() for x in ['buy', 'outperform']))
                sell_count = sum(1 for r in ratings if r[0] and any(x in r[0].lower() for x in ['sell', 'underperform']))
                hold_count = len(ratings) - buy_count - sell_count
                data['analyst_buy'] = buy_count
                data['analyst_hold'] = hold_count
                data['analyst_sell'] = sell_count
                data['analyst_total'] = len(ratings)
                price_targets = [float(r[1]) for r in ratings if r[1]]
                data['avg_price_target'] = np.mean(price_targets) if price_targets else None

            # Fetch insider data
            result = await session.execute(text("""
                SELECT transaction_type, SUM(shares) as total_shares
                FROM insider_trades
                WHERE ticker = :ticker AND transaction_date >= NOW() - INTERVAL '90 days'
                GROUP BY transaction_type
            """), {"ticker": ticker})
            net_buying = 0
            for row in result.fetchall():
                if row[0] and 'buy' in row[0].lower():
                    net_buying += row[1] or 0
                elif row[0] and 'sell' in row[0].lower():
                    net_buying -= row[1] or 0
            data['insider_net_buying'] = net_buying

            # Fetch institutional data
            result = await session.execute(text("""
                SELECT COUNT(DISTINCT institution_cik), SUM(shares)
                FROM institutional_holdings
                WHERE ticker = :ticker AND filing_date >= NOW() - INTERVAL '120 days'
            """), {"ticker": ticker})
            row = result.fetchone()
            data['num_institutions'] = row[0] if row else 0
            data['inst_shares'] = float(row[1]) if row and row[1] else 0

            # Fetch news sentiment
            result = await session.execute(text("""
                SELECT AVG(sentiment_score), COUNT(*)
                FROM news_articles
                WHERE :ticker = ANY(tickers) AND published_at >= NOW() - INTERVAL '30 days'
            """), {"ticker": ticker})
            row = result.fetchone()
            data['news_sentiment'] = float(row[0]) if row and row[0] else None
            data['news_count'] = row[1] if row else 0

            # Fetch sector info
            result = await session.execute(text("""
                SELECT sector FROM tickers WHERE symbol = :ticker
            """), {"ticker": ticker})
            row = result.fetchone()
            data['sector'] = row[0] if row else None

        return data

    async def fetch_sector_stats(self, sector: str) -> Dict[str, Dict[str, float]]:
        """Fetch sector statistics for percentile calculations."""
        if sector in self.sector_stats:
            return self.sector_stats[sector]

        stats = {}
        async with self.db.session() as session:
            # Get sector median valuation ratios
            result = await session.execute(text("""
                SELECT
                    PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY vr.ttm_pe_ratio) as median_pe,
                    PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY vr.ttm_pb_ratio) as median_pb,
                    PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY vr.ttm_ps_ratio) as median_ps,
                    PERCENTILE_CONT(0.25) WITHIN GROUP (ORDER BY vr.ttm_pe_ratio) as p25_pe,
                    PERCENTILE_CONT(0.75) WITHIN GROUP (ORDER BY vr.ttm_pe_ratio) as p75_pe
                FROM valuation_ratios vr
                JOIN tickers t ON vr.ticker = t.symbol
                WHERE t.sector = :sector
                  AND vr.ttm_pe_ratio > 0 AND vr.ttm_pe_ratio < 200
                  AND vr.calculation_date >= NOW() - INTERVAL '7 days'
            """), {"sector": sector})
            row = result.fetchone()
            if row:
                stats['pe'] = {'median': float(row[0]) if row[0] else 15, 'p25': float(row[3]) if row[3] else 10, 'p75': float(row[4]) if row[4] else 25}
                stats['pb'] = {'median': float(row[1]) if row[1] else 2}
                stats['ps'] = {'median': float(row[2]) if row[2] else 2}

            # Get sector profitability stats
            result = await session.execute(text("""
                SELECT
                    PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY f.net_margin) as median_margin,
                    PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY f.roe) as median_roe
                FROM financials f
                JOIN tickers t ON f.ticker = t.symbol
                WHERE t.sector = :sector
                  AND f.period_type = 'quarterly'
                ORDER BY f.period_end_date DESC
                LIMIT 500
            """), {"sector": sector})
            row = result.fetchone()
            if row:
                stats['net_margin'] = {'median': float(row[0]) if row[0] else 10}
                stats['roe'] = {'median': float(row[1]) if row[1] else 15}

        self.sector_stats[sector] = stats
        return stats

    def calculate_current_score(self, data: Dict[str, Any]) -> Tuple[float, Dict[str, float]]:
        """Calculate IC Score using CURRENT methodology."""
        scores = {}

        # Value Score (current: absolute benchmarks)
        if data.get('pe_ratio') and data['pe_ratio'] > 0:
            pe_score = max(0, min(100, 100 - (data['pe_ratio'] - 15) * 2.0))
            pb_score = max(0, min(100, 100 - (data.get('pb_ratio', 2) - 2) * 20)) if data.get('pb_ratio') else 50
            ps_score = max(0, min(100, 100 - (data.get('ps_ratio', 2) - 2) * 20)) if data.get('ps_ratio') else 50
            scores['value'] = np.mean([pe_score, pb_score, ps_score])

        # Growth Score
        growth_scores = []
        if data.get('revenue_growth') is not None:
            growth_scores.append(max(0, min(100, 50 + data['revenue_growth'] * 2.5)))
        if data.get('eps_growth') is not None:
            growth_scores.append(max(0, min(100, 50 + data['eps_growth'] * 2.5)))
        if growth_scores:
            scores['growth'] = np.mean(growth_scores)

        # Profitability Score
        prof_scores = []
        if data.get('net_margin') is not None:
            prof_scores.append(max(0, min(100, data['net_margin'] * 5)))
        if data.get('roe') is not None:
            prof_scores.append(max(0, min(100, data['roe'] * 5)))
        if data.get('roa') is not None:
            prof_scores.append(max(0, min(100, data['roa'] * 10)))
        if prof_scores:
            scores['profitability'] = np.mean(prof_scores)

        # Financial Health Score
        health_scores = []
        if data.get('debt_to_equity') is not None:
            health_scores.append(max(0, min(100, 100 - data['debt_to_equity'] * 50)))
        if data.get('current_ratio') is not None:
            health_scores.append(max(0, min(100, 100 - abs(data['current_ratio'] - 2.0) * 40)))
        if health_scores:
            scores['financial_health'] = np.mean(health_scores)

        # Momentum Score
        momentum_scores = []
        for ret in ['return_1m', 'return_3m', 'return_6m', 'return_12m']:
            if data.get(ret) is not None:
                momentum_scores.append(max(0, min(100, 50 + data[ret] * 2.5)))
        if momentum_scores:
            scores['momentum'] = np.mean(momentum_scores)

        # Technical Score
        if data.get('rsi') is not None:
            rsi = data['rsi']
            if rsi < 50:
                rsi_score = max(0, (rsi - 30) * 2.5)
            else:
                rsi_score = min(100, 50 + (rsi - 50) * 2.5)
            scores['technical'] = rsi_score

        # Analyst Score
        if data.get('analyst_total', 0) > 0:
            total = data['analyst_total']
            buy = data.get('analyst_buy', 0)
            hold = data.get('analyst_hold', 0)
            sell = data.get('analyst_sell', 0)
            scores['analyst_consensus'] = (buy * 100 + hold * 50 + sell * 0) / total

        # Insider Score
        if data.get('insider_net_buying') is not None:
            net = data['insider_net_buying']
            scores['insider_activity'] = max(0, min(100, 50 + net / 2000))

        # Institutional Score
        if data.get('num_institutions', 0) > 0:
            scores['institutional'] = min(100, (data['num_institutions'] / 100) * 100)

        # News Sentiment Score
        if data.get('news_sentiment') is not None:
            scores['news_sentiment'] = max(0, min(100, data['news_sentiment']))

        # Calculate weighted overall
        total_weight = sum(self.CURRENT_WEIGHTS[f] for f in scores.keys())
        if total_weight > 0:
            overall = sum(scores[f] * self.CURRENT_WEIGHTS[f] for f in scores.keys()) / total_weight
        else:
            overall = 0

        return overall, scores

    def sector_percentile(self, value: float, sector_stats: Dict, metric: str, lower_is_better: bool = False) -> float:
        """Calculate sector percentile score."""
        if metric not in sector_stats:
            return 50  # Neutral if no sector data

        median = sector_stats[metric].get('median', value)
        p25 = sector_stats[metric].get('p25', median * 0.7)
        p75 = sector_stats[metric].get('p75', median * 1.3)

        # Simple percentile estimation
        if value <= p25:
            pct = 25 * (value / p25) if p25 > 0 else 25
        elif value <= median:
            pct = 25 + 25 * ((value - p25) / (median - p25)) if median > p25 else 50
        elif value <= p75:
            pct = 50 + 25 * ((value - median) / (p75 - median)) if p75 > median else 75
        else:
            pct = min(100, 75 + 25 * ((value - p75) / p75)) if p75 > 0 else 100

        return 100 - pct if lower_is_better else pct

    def classify_lifecycle(self, data: Dict[str, Any]) -> str:
        """Classify company lifecycle stage."""
        revenue_growth = data.get('revenue_growth', 0) or 0
        net_margin = data.get('net_margin', 0) or 0
        pe_ratio = data.get('pe_ratio', 20) or 20

        if revenue_growth > 40:
            return 'hypergrowth'
        elif revenue_growth > 15:
            return 'growth'
        elif revenue_growth > 0 and net_margin > 5:
            return 'mature'
        elif pe_ratio < 12 and net_margin > 0:
            return 'value'
        elif revenue_growth < 0:
            return 'turnaround'
        else:
            return 'mature'

    async def calculate_proposed_score(self, data: Dict[str, Any]) -> Tuple[float, Dict[str, float], str]:
        """Calculate IC Score using PROPOSED methodology (sector-relative, lifecycle-aware)."""
        scores = {}
        sector = data.get('sector', 'Technology')
        sector_stats = await self.fetch_sector_stats(sector)

        # Classify lifecycle
        lifecycle = self.classify_lifecycle(data)

        # VALUE SCORE (Sector-relative) - Lower P/E is better
        if data.get('pe_ratio') and data['pe_ratio'] > 0:
            pe_pct = self.sector_percentile(data['pe_ratio'], sector_stats, 'pe', lower_is_better=True)
            pb_pct = self.sector_percentile(data.get('pb_ratio', 2), sector_stats, 'pb', lower_is_better=True) if data.get('pb_ratio') else 50
            ps_pct = self.sector_percentile(data.get('ps_ratio', 2), sector_stats, 'ps', lower_is_better=True) if data.get('ps_ratio') else 50
            scores['value'] = pe_pct * 0.4 + pb_pct * 0.3 + ps_pct * 0.3

        # INTRINSIC VALUE SCORE (DCF-based fair value comparison)
        if data.get('price') and data.get('avg_price_target'):
            margin_of_safety = (data['avg_price_target'] - data['price']) / data['avg_price_target']
            scores['intrinsic_value'] = max(0, min(100, 50 + margin_of_safety * 100))
        elif data.get('fcf') and data.get('market_cap') and data['market_cap'] > 0:
            fcf_yield = (data['fcf'] / data['market_cap']) * 100
            scores['intrinsic_value'] = max(0, min(100, 50 + fcf_yield * 5))

        # GROWTH SCORE (same as current but with sector context awareness)
        growth_scores = []
        if data.get('revenue_growth') is not None:
            # Sector-aware: tech expects higher growth
            base_expectation = 10 if sector in ['Technology', 'Healthcare'] else 5
            adjusted_growth = data['revenue_growth'] - base_expectation
            growth_scores.append(max(0, min(100, 50 + adjusted_growth * 2.5)))
        if data.get('eps_growth') is not None:
            growth_scores.append(max(0, min(100, 50 + data['eps_growth'] * 2.0)))
        if growth_scores:
            scores['growth'] = np.mean(growth_scores)

        # PROFITABILITY SCORE (Sector-relative)
        prof_scores = []
        if data.get('net_margin') is not None:
            prof_scores.append(self.sector_percentile(data['net_margin'], sector_stats, 'net_margin'))
        if data.get('roe') is not None:
            prof_scores.append(self.sector_percentile(data['roe'], sector_stats, 'roe'))
        if data.get('gross_margin') is not None:
            prof_scores.append(max(0, min(100, data['gross_margin'] * 2)))  # Gross margin 50% = 100
        if prof_scores:
            scores['profitability'] = np.mean(prof_scores)

        # FINANCIAL HEALTH SCORE (same logic, works across sectors)
        health_scores = []
        if data.get('debt_to_equity') is not None:
            health_scores.append(max(0, min(100, 100 - data['debt_to_equity'] * 40)))
        if data.get('current_ratio') is not None:
            # Optimal range scoring
            cr = data['current_ratio']
            if cr < 1:
                cr_score = cr * 50
            elif cr <= 2:
                cr_score = 50 + (cr - 1) * 50
            else:
                cr_score = max(50, 100 - (cr - 2) * 20)
            health_scores.append(cr_score)
        if health_scores:
            scores['financial_health'] = np.mean(health_scores)

        # SMART MONEY SCORE (Combined: Analyst + Insider + Institutional)
        smart_scores = []
        # Analyst consensus (40% of smart money)
        if data.get('analyst_total', 0) > 0:
            total = data['analyst_total']
            buy = data.get('analyst_buy', 0)
            hold = data.get('analyst_hold', 0)
            sell = data.get('analyst_sell', 0)
            smart_scores.append((buy * 100 + hold * 50 + sell * 0) / total * 0.4)
        # Insider activity (35% of smart money)
        if data.get('insider_net_buying') is not None:
            net = data['insider_net_buying']
            insider_score = max(0, min(100, 50 + net / 1000))
            smart_scores.append(insider_score * 0.35)
        # Institutional (25% of smart money)
        if data.get('num_institutions', 0) > 0:
            inst_score = min(100, (data['num_institutions'] / 50) * 100)
            smart_scores.append(inst_score * 0.25)
        if smart_scores:
            scores['smart_money'] = sum(smart_scores) / (0.4 + 0.35 + 0.25) * (len(smart_scores) / 3)

        # MOMENTUM SCORE (weighted by recency)
        momentum_scores = []
        weights = {'return_1m': 0.2, 'return_3m': 0.3, 'return_6m': 0.3, 'return_12m': 0.2}
        for ret, w in weights.items():
            if data.get(ret) is not None:
                momentum_scores.append((max(0, min(100, 50 + data[ret] * 2.0)), w))
        if momentum_scores:
            total_w = sum(w for _, w in momentum_scores)
            scores['momentum'] = sum(s * w for s, w in momentum_scores) / total_w

        # TECHNICAL SCORE
        if data.get('rsi') is not None:
            rsi = data['rsi']
            # RSI interpretation: oversold (<30) is bullish opportunity, overbought (>70) is bearish
            if rsi <= 30:
                rsi_score = 70 + (30 - rsi)  # Oversold = bullish signal
            elif rsi >= 70:
                rsi_score = 30 - (rsi - 70)  # Overbought = bearish signal
            else:
                rsi_score = 50  # Neutral
            scores['technical'] = max(0, min(100, rsi_score))

        # SENTIMENT SCORE (News + would include EPS revisions)
        if data.get('news_sentiment') is not None:
            scores['sentiment'] = max(0, min(100, (data['news_sentiment'] + 100) / 2))  # Convert -100 to 100 → 0 to 100

        # Apply lifecycle weight adjustments
        adjusted_weights = self.PROPOSED_WEIGHTS.copy()
        if lifecycle in self.LIFECYCLE_ADJUSTMENTS:
            for factor, multiplier in self.LIFECYCLE_ADJUSTMENTS[lifecycle].items():
                if factor in adjusted_weights:
                    adjusted_weights[factor] *= multiplier

        # Normalize weights
        total_base = sum(adjusted_weights.values())
        adjusted_weights = {k: v / total_base for k, v in adjusted_weights.items()}

        # Calculate weighted overall
        available_factors = [f for f in scores.keys() if f in adjusted_weights]
        total_weight = sum(adjusted_weights[f] for f in available_factors)
        if total_weight > 0:
            overall = sum(scores[f] * adjusted_weights[f] for f in available_factors) / total_weight
        else:
            overall = 0

        return overall, scores, lifecycle

    def get_rating(self, score: float) -> str:
        """Convert score to rating."""
        if score >= 80:
            return "Strong Buy"
        elif score >= 65:
            return "Buy"
        elif score >= 50:
            return "Hold"
        elif score >= 35:
            return "Sell"
        else:
            return "Strong Sell"

    async def validate(self):
        """Run validation on all test stocks."""
        print("=" * 100)
        print("IC SCORE VALIDATION: Current vs Proposed Design")
        print("=" * 100)
        print()

        results = []

        for stock in VALIDATION_STOCKS:
            ticker = stock['ticker']
            print(f"Processing {ticker} ({stock['name']})...")

            try:
                data = await self.fetch_stock_data(ticker)

                # Calculate both scores
                current_score, current_factors = self.calculate_current_score(data)
                proposed_score, proposed_factors, lifecycle = await self.calculate_proposed_score(data)

                results.append({
                    'ticker': ticker,
                    'name': stock['name'],
                    'sector': data.get('sector', stock['sector']),
                    'type': stock['type'],
                    'lifecycle': lifecycle,
                    'current_score': current_score,
                    'current_rating': self.get_rating(current_score),
                    'proposed_score': proposed_score,
                    'proposed_rating': self.get_rating(proposed_score),
                    'delta': proposed_score - current_score,
                    'current_factors': current_factors,
                    'proposed_factors': proposed_factors,
                    'pe_ratio': data.get('pe_ratio'),
                    'revenue_growth': data.get('revenue_growth'),
                    'net_margin': data.get('net_margin'),
                })
            except Exception as e:
                print(f"  Error: {e}")
                continue

        # Print comparison table
        print()
        print("=" * 100)
        print("COMPARISON RESULTS")
        print("=" * 100)
        print()
        print(f"{'Ticker':<8} {'Name':<20} {'Sector':<20} {'Lifecycle':<12} {'Current':<12} {'Proposed':<12} {'Delta':<8} {'Rating Change'}")
        print("-" * 100)

        for r in results:
            rating_change = ""
            if r['current_rating'] != r['proposed_rating']:
                rating_change = f"{r['current_rating'][:4]}→{r['proposed_rating'][:4]}"

            print(f"{r['ticker']:<8} {r['name']:<20} {r['sector'][:19]:<20} {r['lifecycle']:<12} "
                  f"{r['current_score']:>6.1f} ({r['current_rating'][:4]:>4}) "
                  f"{r['proposed_score']:>6.1f} ({r['proposed_rating'][:4]:>4}) "
                  f"{r['delta']:>+6.1f}  {rating_change}")

        # Print detailed factor breakdown
        print()
        print("=" * 100)
        print("DETAILED FACTOR BREAKDOWN")
        print("=" * 100)

        for r in results:
            print(f"\n{r['ticker']} - {r['name']} | Sector: {r['sector']} | Lifecycle: {r['lifecycle']}")
            print(f"  P/E: {r['pe_ratio']:.1f if r['pe_ratio'] else 'N/A'} | Rev Growth: {r['revenue_growth']:.1f}% | Net Margin: {r['net_margin']:.1f}%" if r['revenue_growth'] else f"  P/E: {r['pe_ratio']:.1f if r['pe_ratio'] else 'N/A'}")
            print("-" * 80)

            # Current factors
            print("  CURRENT FACTORS:")
            for f, s in sorted(r['current_factors'].items()):
                weight = self.CURRENT_WEIGHTS.get(f, 0) * 100
                print(f"    {f:<20}: {s:>6.1f}  (weight: {weight:.0f}%)")

            # Proposed factors
            print("  PROPOSED FACTORS:")
            for f, s in sorted(r['proposed_factors'].items()):
                weight = self.PROPOSED_WEIGHTS.get(f, 0) * 100
                print(f"    {f:<20}: {s:>6.1f}  (weight: {weight:.0f}%)")

        # Summary statistics
        print()
        print("=" * 100)
        print("SUMMARY STATISTICS")
        print("=" * 100)
        print()

        avg_current = np.mean([r['current_score'] for r in results])
        avg_proposed = np.mean([r['proposed_score'] for r in results])
        avg_delta = np.mean([r['delta'] for r in results])

        print(f"Average Current Score:  {avg_current:.1f}")
        print(f"Average Proposed Score: {avg_proposed:.1f}")
        print(f"Average Delta:          {avg_delta:+.1f}")
        print()

        # Count rating changes
        rating_changes = sum(1 for r in results if r['current_rating'] != r['proposed_rating'])
        print(f"Rating Changes: {rating_changes} out of {len(results)} stocks")

        # Lifecycle distribution
        print()
        print("Lifecycle Classification:")
        from collections import Counter
        lifecycle_counts = Counter(r['lifecycle'] for r in results)
        for lc, count in lifecycle_counts.most_common():
            print(f"  {lc}: {count}")


async def main():
    validator = ICScoreValidator()
    await validator.validate()


if __name__ == '__main__':
    asyncio.run(main())
