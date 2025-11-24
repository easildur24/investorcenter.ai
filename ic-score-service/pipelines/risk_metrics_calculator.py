#!/usr/bin/env python3
"""Risk Metrics Calculator Pipeline.

This script calculates risk-adjusted performance metrics for stocks:

Metrics Calculated:
1. Alpha: Excess return vs. benchmark adjusted for risk
2. Beta: Volatility relative to benchmark (market correlation)
3. Sharpe Ratio: Risk-adjusted return using total volatility
4. Sortino Ratio: Risk-adjusted return using downside volatility only
5. Standard Deviation: Annualized volatility
6. Maximum Drawdown: Largest peak-to-trough decline
7. VaR 5%: Value at Risk at 5% confidence level

Methodology based on YCharts:
- Monthly returns calculated from daily prices
- Risk-free rate: 1-month or 3-month Treasury yield
- Benchmark: S&P 500 (SPY)
- Periods: 1Y, 3Y (5Y when available)

Usage:
    python risk_metrics_calculator.py --limit 100  # Test on 100 stocks
    python risk_metrics_calculator.py --all        # All stocks
    python risk_metrics_calculator.py --ticker AAPL  # Single stock
"""

import argparse
import asyncio
import logging
import sys
from datetime import datetime, timedelta, timezone
from pathlib import Path
from typing import List, Optional, Dict, Any, Tuple

# Add parent directory to path for imports
sys.path.insert(0, str(Path(__file__).parent.parent))

import numpy as np
import pandas as pd
from scipy import stats
from sqlalchemy import text
from sqlalchemy.dialects.postgresql import insert as pg_insert
from sqlalchemy.ext.asyncio import AsyncSession
from tqdm import tqdm

from database.database import get_database
from models import RiskMetric

# Setup logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s',
    handlers=[
        logging.StreamHandler(),
        logging.FileHandler('/app/logs/risk_metrics_calculator.log')
    ]
)
logger = logging.getLogger(__name__)


class RiskMetricsCalculator:
    """Calculator for risk-adjusted performance metrics."""

    # Periods to calculate (in trading days)
    PERIODS = {
        '1Y': 252,   # 1 year
        '3Y': 756,   # 3 years
        # '5Y': 1260  # 5 years (add later)
    }

    # Benchmark symbol
    BENCHMARK_SYMBOL = 'SPY'

    def __init__(self):
        """Initialize the calculator."""
        self.db = get_database()

        # Track progress
        self.processed_count = 0
        self.success_count = 0
        self.error_count = 0

    async def get_stocks_to_process(
        self,
        limit: Optional[int] = None,
        ticker: Optional[str] = None
    ) -> List[str]:
        """Get list of stocks to process from database.

        Args:
            limit: Maximum number of stocks to process.
            ticker: Single ticker to process.

        Returns:
            List of ticker symbols.
        """
        async with self.db.session() as session:
            if ticker:
                return [ticker.upper()]

            query = text("""
                SELECT ticker
                FROM stocks
                WHERE ticker NOT LIKE '%-%'
                  AND is_active = true
                ORDER BY ticker
                LIMIT :limit
            """)
            result = await session.execute(query, {"limit": limit or 10000})
            tickers = [row[0] for row in result.fetchall()]

        logger.info(f"Found {len(tickers)} stocks to process")
        return tickers

    async def get_stock_prices(
        self,
        ticker: str,
        days: int,
        session: AsyncSession
    ) -> Optional[pd.DataFrame]:
        """Fetch historical stock prices.

        Args:
            ticker: Stock ticker symbol.
            days: Number of days to fetch.
            session: Database session.

        Returns:
            DataFrame with columns: date, close.
        """
        query = text("""
            SELECT
                DATE(time) as date,
                close
            FROM stock_prices
            WHERE ticker = :ticker
              AND interval = '1day'
              AND time >= NOW() - INTERVAL ':days days'
            ORDER BY time
        """)

        result = await session.execute(query, {"ticker": ticker, "days": days + 100})
        rows = result.fetchall()

        if not rows:
            return None

        df = pd.DataFrame(rows, columns=['date', 'close'])
        df['close'] = pd.to_numeric(df['close'], errors='coerce')
        df = df.dropna()

        return df

    async def get_benchmark_prices(
        self,
        days: int,
        session: AsyncSession
    ) -> Optional[pd.DataFrame]:
        """Fetch historical benchmark prices.

        Args:
            days: Number of days to fetch.
            session: Database session.

        Returns:
            DataFrame with columns: date, close.
        """
        query = text("""
            SELECT
                DATE(time) as date,
                close
            FROM benchmark_returns
            WHERE symbol = :symbol
              AND time >= NOW() - INTERVAL ':days days'
            ORDER BY time
        """)

        result = await session.execute(query, {"symbol": self.BENCHMARK_SYMBOL, "days": days + 100})
        rows = result.fetchall()

        if not rows:
            return None

        df = pd.DataFrame(rows, columns=['date', 'close'])
        df['close'] = pd.to_numeric(df['close'], errors='coerce')
        df = df.dropna()

        return df

    async def get_risk_free_rate(self, session: AsyncSession) -> float:
        """Get current risk-free rate (3-month Treasury).

        Args:
            session: Database session.

        Returns:
            Annual risk-free rate as percentage (e.g., 4.5 for 4.5%).
        """
        query = text("""
            SELECT rate_3m
            FROM treasury_rates
            WHERE rate_3m IS NOT NULL
            ORDER BY date DESC
            LIMIT 1
        """)

        result = await session.execute(query)
        row = result.fetchone()

        if row and row[0]:
            return float(row[0])

        # Fallback: use historical average
        logger.warning("Using fallback risk-free rate of 4.0%")
        return 4.0

    def calculate_monthly_returns(self, df: pd.DataFrame) -> pd.DataFrame:
        """Convert daily prices to monthly returns.

        Args:
            df: DataFrame with 'date' and 'close' columns.

        Returns:
            DataFrame with monthly returns.
        """
        if df.empty:
            return pd.DataFrame()

        df = df.copy()
        df['date'] = pd.to_datetime(df['date'])
        df = df.set_index('date')

        # Resample to month-end and take last close price
        monthly = df.resample('M').last()

        # Calculate percentage returns
        monthly['return'] = monthly['close'].pct_change() * 100

        # Drop first row (NaN)
        monthly = monthly.dropna()

        return monthly.reset_index()

    def calculate_beta(
        self,
        stock_returns: pd.Series,
        benchmark_returns: pd.Series
    ) -> Optional[float]:
        """Calculate Beta (market correlation).

        Formula: Beta = Covariance(stock, benchmark) / Variance(benchmark)

        Args:
            stock_returns: Stock monthly returns.
            benchmark_returns: Benchmark monthly returns.

        Returns:
            Beta value or None if insufficient data.
        """
        if len(stock_returns) < 12 or len(benchmark_returns) < 12:
            return None

        try:
            covariance = np.cov(stock_returns, benchmark_returns)[0, 1]
            variance = np.var(benchmark_returns, ddof=1)

            if variance == 0:
                return None

            beta = covariance / variance
            return float(beta)

        except Exception as e:
            logger.warning(f"Error calculating beta: {e}")
            return None

    def calculate_alpha(
        self,
        annualized_return: float,
        beta: float,
        benchmark_annualized_return: float,
        risk_free_rate: float
    ) -> Optional[float]:
        """Calculate Alpha (excess return vs benchmark).

        Formula: Alpha = R_stock - R_f - Beta * (R_benchmark - R_f)

        Args:
            annualized_return: Stock annualized return (%).
            beta: Stock beta.
            benchmark_annualized_return: Benchmark annualized return (%).
            risk_free_rate: Risk-free rate (%).

        Returns:
            Alpha (%) or None.
        """
        if beta is None:
            return None

        try:
            alpha = annualized_return - risk_free_rate - beta * (benchmark_annualized_return - risk_free_rate)
            return float(alpha)

        except Exception as e:
            logger.warning(f"Error calculating alpha: {e}")
            return None

    def calculate_sharpe_ratio(
        self,
        annualized_return: float,
        annualized_std: float,
        risk_free_rate: float
    ) -> Optional[float]:
        """Calculate Sharpe Ratio.

        Formula: Sharpe = (R - R_f) / σ

        Args:
            annualized_return: Annualized return (%).
            annualized_std: Annualized standard deviation (%).
            risk_free_rate: Risk-free rate (%).

        Returns:
            Sharpe ratio or None.
        """
        if annualized_std == 0:
            return None

        try:
            sharpe = (annualized_return - risk_free_rate) / annualized_std
            return float(sharpe)

        except Exception as e:
            logger.warning(f"Error calculating Sharpe ratio: {e}")
            return None

    def calculate_sortino_ratio(
        self,
        annualized_return: float,
        monthly_returns: pd.Series,
        risk_free_rate: float
    ) -> Tuple[Optional[float], Optional[float]]:
        """Calculate Sortino Ratio and downside deviation.

        Formula: Sortino = (R - R_f) / Downside_Deviation
        Downside_Deviation = Std dev of returns below 0%

        Args:
            annualized_return: Annualized return (%).
            monthly_returns: Monthly returns series.
            risk_free_rate: Risk-free rate (%).

        Returns:
            Tuple of (Sortino ratio, downside deviation) or (None, None).
        """
        try:
            # Get negative returns only
            negative_returns = monthly_returns[monthly_returns < 0]

            if len(negative_returns) < 2:
                return None, None

            # Calculate downside deviation (monthly)
            downside_dev_monthly = negative_returns.std()

            # Annualize downside deviation
            downside_dev_annual = downside_dev_monthly * np.sqrt(12)

            if downside_dev_annual == 0:
                return None, None

            # Calculate Sortino ratio
            sortino = (annualized_return - risk_free_rate) / downside_dev_annual

            return float(sortino), float(downside_dev_annual)

        except Exception as e:
            logger.warning(f"Error calculating Sortino ratio: {e}")
            return None, None

    def calculate_max_drawdown(self, df: pd.DataFrame) -> Optional[float]:
        """Calculate Maximum Drawdown.

        Formula: Max_Drawdown = (Trough - Peak) / Peak * 100

        Args:
            df: DataFrame with 'close' column.

        Returns:
            Maximum drawdown (%) or None.
        """
        if df.empty or len(df) < 2:
            return None

        try:
            prices = df['close'].values

            # Calculate cumulative maximum (peak)
            cummax = np.maximum.accumulate(prices)

            # Calculate drawdown at each point
            drawdown = (prices - cummax) / cummax * 100

            # Get maximum drawdown (most negative value)
            max_dd = float(np.min(drawdown))

            return max_dd

        except Exception as e:
            logger.warning(f"Error calculating max drawdown: {e}")
            return None

    def calculate_var_5(self, monthly_returns: pd.Series) -> Optional[float]:
        """Calculate Value at Risk at 5% confidence (parametric method).

        Formula: VaR_5% = -1.645 * Monthly_Std_Dev

        Args:
            monthly_returns: Monthly returns series.

        Returns:
            VaR 5% (%) or None.
        """
        if len(monthly_returns) < 12:
            return None

        try:
            # Calculate monthly standard deviation
            monthly_std = monthly_returns.std()

            # VaR at 5% confidence (1.645 is the z-score for 95% confidence)
            var_5 = -1.645 * monthly_std

            return float(var_5)

        except Exception as e:
            logger.warning(f"Error calculating VaR: {e}")
            return None

    def calculate_annualized_return(
        self,
        start_price: float,
        end_price: float,
        years: float
    ) -> float:
        """Calculate annualized return.

        Formula: Annualized_Return = ((End / Start) ^ (1 / Years) - 1) * 100

        Args:
            start_price: Starting price.
            end_price: Ending price.
            years: Number of years.

        Returns:
            Annualized return (%).
        """
        if start_price <= 0 or years <= 0:
            return 0.0

        try:
            annualized_return = ((end_price / start_price) ** (1 / years) - 1) * 100
            return float(annualized_return)

        except Exception as e:
            logger.warning(f"Error calculating annualized return: {e}")
            return 0.0

    async def calculate_metrics_for_period(
        self,
        ticker: str,
        period: str,
        trading_days: int,
        session: AsyncSession
    ) -> Optional[Dict[str, Any]]:
        """Calculate all risk metrics for a specific period.

        Args:
            ticker: Stock ticker.
            period: Period label (e.g., '1Y', '3Y').
            trading_days: Number of trading days in period.
            session: Database session.

        Returns:
            Dictionary of metrics or None if insufficient data.
        """
        try:
            # Fetch stock prices
            stock_df = await self.get_stock_prices(ticker, trading_days, session)

            if stock_df is None or len(stock_df) < trading_days * 0.7:
                logger.warning(f"{ticker}: Insufficient price data for {period}")
                return None

            # Fetch benchmark prices
            benchmark_df = await self.get_benchmark_prices(trading_days, session)

            if benchmark_df is None or len(benchmark_df) < trading_days * 0.7:
                logger.warning(f"{ticker}: Insufficient benchmark data for {period}")
                return None

            # Get risk-free rate
            risk_free_rate = await self.get_risk_free_rate(session)

            # Calculate monthly returns
            stock_monthly = self.calculate_monthly_returns(stock_df)
            benchmark_monthly = self.calculate_monthly_returns(benchmark_df)

            if stock_monthly.empty or benchmark_monthly.empty:
                return None

            # Align dates
            merged = pd.merge(
                stock_monthly[['date', 'return']],
                benchmark_monthly[['date', 'return']],
                on='date',
                suffixes=('_stock', '_benchmark')
            )

            if len(merged) < 12:
                logger.warning(f"{ticker}: Insufficient monthly returns for {period}")
                return None

            stock_returns = merged['return_stock']
            benchmark_returns = merged['return_benchmark']

            # Calculate annualized returns
            years = trading_days / 252
            stock_annualized_return = self.calculate_annualized_return(
                stock_df['close'].iloc[0],
                stock_df['close'].iloc[-1],
                years
            )
            benchmark_annualized_return = self.calculate_annualized_return(
                benchmark_df['close'].iloc[0],
                benchmark_df['close'].iloc[-1],
                years
            )

            # Calculate standard deviation (annualized)
            monthly_std = stock_returns.std()
            annualized_std = monthly_std * np.sqrt(12)

            # Calculate Beta
            beta = self.calculate_beta(stock_returns, benchmark_returns)

            # Calculate Alpha
            alpha = self.calculate_alpha(
                stock_annualized_return,
                beta,
                benchmark_annualized_return,
                risk_free_rate
            )

            # Calculate Sharpe Ratio
            sharpe = self.calculate_sharpe_ratio(
                stock_annualized_return,
                annualized_std,
                risk_free_rate
            )

            # Calculate Sortino Ratio
            sortino, downside_dev = self.calculate_sortino_ratio(
                stock_annualized_return,
                stock_returns,
                risk_free_rate
            )

            # Calculate Maximum Drawdown
            max_drawdown = self.calculate_max_drawdown(stock_df)

            # Calculate VaR 5%
            var_5 = self.calculate_var_5(stock_returns)

            # Return all metrics
            return {
                'period': period,
                'alpha': alpha,
                'beta': beta,
                'sharpe_ratio': sharpe,
                'sortino_ratio': sortino,
                'std_dev': float(annualized_std),
                'max_drawdown': max_drawdown,
                'var_5': var_5,
                'annualized_return': float(stock_annualized_return),
                'downside_deviation': downside_dev,
                'data_points': len(merged)
            }

        except Exception as e:
            logger.error(f"{ticker}: Error calculating metrics for {period}: {e}", exc_info=True)
            return None

    async def store_metrics(
        self,
        ticker: str,
        metrics: Dict[str, Any],
        session: AsyncSession
    ) -> bool:
        """Store risk metrics in database.

        Args:
            ticker: Stock ticker.
            metrics: Dictionary of metrics.
            session: Database session.

        Returns:
            True if successful.
        """
        try:
            current_time = datetime.now(timezone.utc)

            record = {
                'time': current_time,
                'ticker': ticker,
                'period': metrics['period'],
                'alpha': metrics.get('alpha'),
                'beta': metrics.get('beta'),
                'sharpe_ratio': metrics.get('sharpe_ratio'),
                'sortino_ratio': metrics.get('sortino_ratio'),
                'std_dev': metrics.get('std_dev'),
                'max_drawdown': metrics.get('max_drawdown'),
                'var_5': metrics.get('var_5'),
                'annualized_return': metrics.get('annualized_return'),
                'downside_deviation': metrics.get('downside_deviation'),
                'data_points': metrics.get('data_points')
            }

            # Insert with ON CONFLICT DO UPDATE
            stmt = pg_insert(RiskMetric.__table__).values([record])
            stmt = stmt.on_conflict_do_update(
                index_elements=['time', 'ticker', 'period'],
                set_={
                    'alpha': stmt.excluded.alpha,
                    'beta': stmt.excluded.beta,
                    'sharpe_ratio': stmt.excluded.sharpe_ratio,
                    'sortino_ratio': stmt.excluded.sortino_ratio,
                    'std_dev': stmt.excluded.std_dev,
                    'max_drawdown': stmt.excluded.max_drawdown,
                    'var_5': stmt.excluded.var_5,
                    'annualized_return': stmt.excluded.annualized_return,
                    'downside_deviation': stmt.excluded.downside_deviation,
                    'data_points': stmt.excluded.data_points,
                }
            )

            await session.execute(stmt)
            return True

        except Exception as e:
            logger.error(f"{ticker}: Error storing metrics: {e}", exc_info=True)
            return False

    async def process_ticker(self, ticker: str, session: AsyncSession) -> bool:
        """Process a single ticker: calculate and store risk metrics.

        Args:
            ticker: Stock ticker symbol.
            session: Database session.

        Returns:
            True if successful.
        """
        try:
            success_count = 0

            # Calculate metrics for each period
            for period, trading_days in self.PERIODS.items():
                metrics = await self.calculate_metrics_for_period(
                    ticker,
                    period,
                    trading_days,
                    session
                )

                if metrics:
                    stored = await self.store_metrics(ticker, metrics, session)
                    if stored:
                        success_count += 1

            if success_count > 0:
                await session.commit()
                logger.info(f"{ticker}: Successfully calculated {success_count} period(s)")
                return True
            else:
                logger.warning(f"{ticker}: No metrics calculated")
                return False

        except Exception as e:
            logger.error(f"{ticker}: Error processing: {e}", exc_info=True)
            await session.rollback()
            return False

    async def process_stocks(self, tickers: List[str], show_progress: bool = True):
        """Process a list of tickers.

        Args:
            tickers: List of ticker symbols.
            show_progress: Show progress bar.
        """
        progress_bar = tqdm(total=len(tickers), desc="Processing stocks") if show_progress else None

        for ticker in tickers:
            async with self.db.session() as session:
                success = await self.process_ticker(ticker, session)

            self.processed_count += 1
            if success:
                self.success_count += 1
            else:
                self.error_count += 1

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
        all_stocks: bool = False
    ):
        """Run the calculator pipeline.

        Args:
            limit: Limit number of stocks to process.
            ticker: Process single ticker.
            all_stocks: Process all stocks.
        """
        start_time = datetime.now()
        logger.info("=" * 80)
        logger.info("Risk Metrics Calculator Pipeline")
        logger.info("=" * 80)

        # Determine limit
        if all_stocks:
            limit = None
        elif ticker:
            limit = 1
        elif limit is None:
            limit = 10  # Default to 10 for safety

        # Get stocks to process
        tickers = await self.get_stocks_to_process(limit=limit, ticker=ticker)

        if not tickers:
            logger.warning("No stocks to process")
            return

        logger.info(f"Processing {len(tickers)} stocks...")

        # Process stocks
        await self.process_stocks(tickers)

        # Print summary
        duration = datetime.now() - start_time
        logger.info("=" * 80)
        logger.info("Processing Complete")
        logger.info("=" * 80)
        logger.info(f"Duration: {duration}")
        logger.info(f"Processed: {self.processed_count}")
        logger.info(f"Success: {self.success_count}")
        logger.info(f"Errors: {self.error_count}")
        logger.info(f"Success Rate: {(self.success_count/self.processed_count*100):.1f}%")


def main():
    """Main entry point for CLI."""
    parser = argparse.ArgumentParser(
        description='Calculate risk metrics for stocks',
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  # Test on 10 stocks (default)
  python risk_metrics_calculator.py

  # Test on 100 stocks
  python risk_metrics_calculator.py --limit 100

  # Process all stocks
  python risk_metrics_calculator.py --all

  # Process single stock
  python risk_metrics_calculator.py --ticker AAPL
        """
    )

    parser.add_argument(
        '--limit',
        type=int,
        help='Limit number of stocks to process (default: 10)'
    )
    parser.add_argument(
        '--ticker',
        type=str,
        help='Process single ticker symbol'
    )
    parser.add_argument(
        '--all',
        action='store_true',
        help='Process all stocks in database'
    )

    args = parser.parse_args()

    # Validate arguments
    if args.ticker and (args.all or args.limit):
        parser.error("--ticker cannot be used with --all or --limit")

    # Run pipeline
    calculator = RiskMetricsCalculator()
    asyncio.run(calculator.run(
        limit=args.limit,
        ticker=args.ticker,
        all_stocks=args.all
    ))


if __name__ == '__main__':
    main()
