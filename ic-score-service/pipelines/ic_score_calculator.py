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

# Setup logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s',
    handlers=[
        logging.StreamHandler(),
        logging.FileHandler('ic_score_calculator.log')
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
                    SELECT s.ticker, c.sector
                    FROM stocks s
                    LEFT JOIN companies c ON s.ticker = c.ticker
                    WHERE s.ticker = :ticker
                """)
                result = await session.execute(query, {"ticker": ticker.upper()})
            else:
                where_clauses = ["s.ticker NOT LIKE '%-%'", "s.is_active = true"]
                params = {}

                if sector:
                    where_clauses.append("c.sector = :sector")
                    params['sector'] = sector

                if sp500:
                    where_clauses.append("s.is_sp500 = true")

                query_str = f"""
                    SELECT s.ticker, c.sector
                    FROM stocks s
                    LEFT JOIN companies c ON s.ticker = c.ticker
                    WHERE {' AND '.join(where_clauses)}
                    ORDER BY s.ticker
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

    def calculate_value_score(self, financial_data: Dict[str, Any], sector_data: Dict) -> Tuple[Optional[float], Dict]:
        """Calculate value score based on P/E, P/B, P/S ratios vs sector median."""
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
            pe_score = max(0, min(100, 100 - (pe - 15) * 2))  # Centered at P/E of 15
            scores.append(pe_score)
            metadata['pe_ratio'] = float(pe)

        if pb and pb > 0:
            pb_score = max(0, min(100, 100 - (pb - 2) * 20))  # Centered at P/B of 2
            scores.append(pb_score)
            metadata['pb_ratio'] = float(pb)

        if ps and ps > 0:
            ps_score = max(0, min(100, 100 - (ps - 2) * 20))  # Centered at P/S of 2
            scores.append(ps_score)
            metadata['ps_ratio'] = float(ps)

        if not scores:
            return None, metadata

        return np.mean(scores), metadata

    def calculate_growth_score(self, financial_data: Dict[str, Any]) -> Tuple[Optional[float], Dict]:
        """Calculate growth score based on revenue, EPS, FCF growth."""
        if not financial_data or len(financial_data.get('historical', [])) < 4:
            return None, {}

        historical = financial_data['historical']
        metadata = {}

        # Calculate year-over-year growth rates
        scores = []

        # Revenue growth
        if historical[0].get('revenue') and historical[3].get('revenue'):
            rev_growth = ((historical[0]['revenue'] / historical[3]['revenue']) - 1) * 100
            # Normalize: 0% = 50, 20% = 100, -20% = 0
            rev_score = max(0, min(100, 50 + rev_growth * 2.5))
            scores.append(rev_score)
            metadata['revenue_growth_yoy'] = rev_growth

        # EPS growth
        if historical[0].get('eps_diluted') and historical[3].get('eps_diluted'):
            if historical[3]['eps_diluted'] > 0:
                eps_growth = ((historical[0]['eps_diluted'] / historical[3]['eps_diluted']) - 1) * 100
                eps_score = max(0, min(100, 50 + eps_growth * 2.5))
                scores.append(eps_score)
                metadata['eps_growth_yoy'] = eps_growth

        if not scores:
            return None, metadata

        return np.mean(scores), metadata

    def calculate_profitability_score(self, financial_data: Dict[str, Any]) -> Tuple[Optional[float], Dict]:
        """Calculate profitability score based on margins, ROE, ROA."""
        if not financial_data:
            return None, {}

        latest = financial_data.get('latest', {})
        metadata = {}
        scores = []

        # Net margin
        if latest.get('net_margin'):
            margin = float(latest['net_margin'])
            # Normalize: 0% = 0, 20% = 100
            margin_score = max(0, min(100, margin * 5))
            scores.append(margin_score)
            metadata['net_margin'] = margin

        # ROE
        if latest.get('roe'):
            roe = float(latest['roe'])
            # Normalize: 0% = 0, 20% = 100
            roe_score = max(0, min(100, roe * 5))
            scores.append(roe_score)
            metadata['roe'] = roe

        # ROA
        if latest.get('roa'):
            roa = float(latest['roa'])
            roa_score = max(0, min(100, roa * 10))
            scores.append(roa_score)
            metadata['roa'] = roa

        if not scores:
            return None, metadata

        return np.mean(scores), metadata

    def calculate_financial_health_score(self, financial_data: Dict[str, Any]) -> Tuple[Optional[float], Dict]:
        """Calculate financial health score based on D/E, current ratio."""
        if not financial_data:
            return None, {}

        latest = financial_data.get('latest', {})
        metadata = {}
        scores = []

        # Debt to equity (lower is better)
        if latest.get('debt_to_equity') is not None:
            de = float(latest['debt_to_equity'])
            # Normalize: 0 = 100, 2 = 0
            de_score = max(0, min(100, 100 - de * 50))
            scores.append(de_score)
            metadata['debt_to_equity'] = de

        # Current ratio (optimal around 1.5-2.0)
        if latest.get('current_ratio'):
            cr = float(latest['current_ratio'])
            # Normalize: 2.0 = 100, 0 or 5+ = 0
            cr_score = max(0, min(100, 100 - abs(cr - 2.0) * 40))
            scores.append(cr_score)
            metadata['current_ratio'] = cr

        if not scores:
            return None, metadata

        return np.mean(scores), metadata

    def calculate_momentum_score(self, technical_data: Dict[str, Any]) -> Tuple[Optional[float], Dict]:
        """Calculate momentum score based on price returns."""
        if not technical_data:
            return None, {}

        metadata = {}
        scores = []

        # Use various period returns
        for period in ['1m_return', '3m_return', '6m_return', '12m_return']:
            if period in technical_data:
                ret = technical_data[period]
                # Normalize: -20% = 0, 0% = 50, 20% = 100
                score = max(0, min(100, 50 + ret * 2.5))
                scores.append(score)
                metadata[period] = ret

        if not scores:
            return None, metadata

        return np.mean(scores), metadata

    def calculate_technical_score(self, technical_data: Dict[str, Any]) -> Tuple[Optional[float], Dict]:
        """Calculate technical score based on RSI, MACD, trend."""
        if not technical_data:
            return None, {}

        metadata = {}
        scores = []

        # RSI (30-70 range is neutral, <30 oversold, >70 overbought)
        if 'rsi' in technical_data:
            rsi = technical_data['rsi']
            # Normalize: 30 = 0, 50 = 50, 70 = 100
            if rsi < 50:
                rsi_score = max(0, (rsi - 30) * 2.5)
            else:
                rsi_score = min(100, 50 + (rsi - 50) * 2.5)
            scores.append(rsi_score)
            metadata['rsi'] = rsi

        # MACD histogram (positive = bullish)
        if 'macd_histogram' in technical_data:
            macd_hist = technical_data['macd_histogram']
            # Simple normalization
            macd_score = 50 + min(50, max(-50, macd_hist * 10))
            scores.append(macd_score)
            metadata['macd_histogram'] = macd_hist

        # Trend (price vs SMA)
        if 'current_price' in technical_data and 'sma_50' in technical_data:
            price = technical_data['current_price']
            sma = technical_data['sma_50']
            if sma > 0:
                trend = ((price / sma) - 1) * 100
                trend_score = max(0, min(100, 50 + trend * 5))
                scores.append(trend_score)
                metadata['price_vs_sma50'] = trend

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

            # Calculate data completeness
            data_completeness = (len(factor_scores) / len(self.WEIGHTS)) * 100

            if not factor_scores:
                logger.warning(f"{ticker}: No factor scores calculated")
                return None

            # Calculate weighted overall score (only for available factors)
            total_weight = sum(self.WEIGHTS[factor] for factor in factor_scores.keys())
            overall_score = sum(
                factor_scores[factor] * self.WEIGHTS[factor]
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
