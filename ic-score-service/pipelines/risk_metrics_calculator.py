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
import os
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

# Setup logging with configurable log directory
LOG_DIR = os.environ.get('LOG_DIR', '/app/logs')
Path(LOG_DIR).mkdir(parents=True, exist_ok=True)

logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s',
    handlers=[
        logging.StreamHandler(),
        logging.FileHandler(os.path.join(LOG_DIR, 'risk_metrics_calculator.log'))
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

    # Epsilon for floating point comparisons (avoids division by near-zero)
    EPSILON = 1e-10

    def __init__(self):
        """Initialize the calculator."""
        self.db = get_database()

        # Track progress
        self.processed_count = 0
        self.success_count = 0
        self.error_count = 0

    @staticmethod
    def validate_ticker(ticker: str) -> bool:
        """Validate ticker symbol format.

        Args:
            ticker: Ticker symbol to validate.

        Returns:
            True if valid, False otherwise.
        """
        import re
        # Valid tickers: 1-5 uppercase letters, optionally with . (e.g., BRK.A)
        pattern = r'^[A-Z]{1,5}(\.[A-Z])?$'
        return bool(re.match(pattern, ticker.upper()))

    async def get_stocks_to_process(
        self,
        limit: Optional[int] = None,
        ticker: Optional[str] = None
    ) -> List[str]:
        """Get list of stocks to process from database.

        Args:
            limit: Maximum number of stocks to process (must be positive).
            ticker: Single ticker to process.

        Returns:
            List of ticker symbols.

        Raises:
            ValueError: If limit is negative or ticker format is invalid.
        """
        # Input validation
        if limit is not None and limit < 0:
            raise ValueError(f"limit must be non-negative, got {limit}")

        if ticker:
            ticker_upper = ticker.upper()
            if not self.validate_ticker(ticker_upper):
                raise ValueError(f"Invalid ticker format: {ticker}")
            return [ticker_upper]

        async with self.db.session() as session:
            query = text("""
                SELECT symbol AS ticker
                FROM tickers
                WHERE symbol NOT LIKE '%-%'
                  AND active = true
                  AND asset_type = 'stock'
                ORDER BY symbol
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
        # Convert trading days to calendar days (252 trading days ≈ 365 calendar days)
        calendar_days = int(days * 365 / 252) + 100

        query = text("""
            SELECT
                DATE(time) as date,
                close
            FROM stock_prices
            WHERE ticker = :ticker
              AND interval = '1day'
              AND time >= NOW() - INTERVAL '1 day' * :days
            ORDER BY time
        """)

        result = await session.execute(query, {"ticker": ticker, "days": calendar_days})
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
        # Convert trading days to calendar days (252 trading days ≈ 365 calendar days)
        calendar_days = int(days * 365 / 252) + 100

        query = text("""
            SELECT
                DATE(time) as date,
                close
            FROM benchmark_returns
            WHERE symbol = :symbol
              AND time >= NOW() - INTERVAL '1 day' * :days
            ORDER BY time
        """)

        result = await session.execute(query, {"symbol": self.BENCHMARK_SYMBOL, "days": calendar_days})
        rows = result.fetchall()

        if not rows:
            return None

        df = pd.DataFrame(rows, columns=['date', 'close'])
        df['close'] = pd.to_numeric(df['close'], errors='coerce')
        df = df.dropna()

        return df

    async def get_risk_free_rate(self, trading_days: int, session: AsyncSession) -> float:
        """Get average 1-month Treasury rate over the lookback period.

        Args:
            trading_days: Number of trading days in the lookback period.
            session: Database session.

        Returns:
            Average annual risk-free rate as percentage (e.g., 4.5 for 4.5%).
        """
        # Convert trading days to calendar days (252 trading days ≈ 365 calendar days)
        calendar_days = int(trading_days * 365 / 252)

        query = text("""
            SELECT AVG(rate_1m) as avg_rate
            FROM treasury_rates
            WHERE rate_1m IS NOT NULL
              AND date >= NOW() - INTERVAL '1 day' * :days
        """)

        result = await session.execute(query, {"days": calendar_days})
        row = result.fetchone()

        if row and row[0]:
            return float(row[0])

        # Fallback: use historical average
        logger.warning("Using fallback risk-free rate of 4.9%")
        return 4.9

    def calculate_daily_returns(self, df: pd.DataFrame) -> pd.DataFrame:
        """Calculate daily returns from price data.

        Args:
            df: DataFrame with 'date' and 'close' columns.

        Returns:
            DataFrame with daily returns.
        """
        if df.empty:
            return pd.DataFrame()

        df = df.copy()
        df['date'] = pd.to_datetime(df['date'])
        df = df.sort_values('date')  # Ensure chronological order
        df = df.set_index('date')

        # Calculate daily percentage returns
        df['return'] = df['close'].pct_change() * 100

        # Drop first row (NaN from pct_change)
        df = df.dropna()

        return df.reset_index()

    def calculate_beta(
        self,
        stock_returns: pd.Series,
        benchmark_returns: pd.Series
    ) -> Optional[float]:
        """Calculate Beta (market correlation).

        Formula: Beta = Covariance(stock, benchmark) / Variance(benchmark)

        Args:
            stock_returns: Stock daily returns.
            benchmark_returns: Benchmark daily returns.

        Returns:
            Beta value or None if insufficient data.
        """
        if len(stock_returns) < 12 or len(benchmark_returns) < 12:
            return None

        try:
            covariance = np.cov(stock_returns, benchmark_returns)[0, 1]
            variance = np.var(benchmark_returns, ddof=1)

            if abs(variance) < self.EPSILON:
                return None

            beta = covariance / variance
            return float(beta)

        except Exception as e:
            logger.error(f"Error calculating beta: {e}", exc_info=True)
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
            logger.error(f"Error calculating alpha: {e}", exc_info=True)
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
        if abs(annualized_std) < self.EPSILON:
            return None

        try:
            sharpe = (annualized_return - risk_free_rate) / annualized_std
            return float(sharpe)

        except Exception as e:
            logger.error(f"Error calculating Sharpe ratio: {e}", exc_info=True)
            return None

    def calculate_sortino_ratio(
        self,
        annualized_return: float,
        daily_returns: pd.Series,
        risk_free_rate: float
    ) -> Tuple[Optional[float], Optional[float]]:
        """Calculate Sortino Ratio and downside deviation.

        Formula: Sortino = (R - R_f) / Downside_Deviation
        Downside_Deviation = Std dev of returns below 0%

        Args:
            annualized_return: Annualized return (%).
            daily_returns: Daily returns series.
            risk_free_rate: Risk-free rate (%).

        Returns:
            Tuple of (Sortino ratio, downside deviation) or (None, None).
        """
        try:
            # Calculate downside deviation (semi-deviation below 0%)
            # Use ALL periods in denominator (YCharts/Sortino standard formula)
            downside_returns = np.minimum(daily_returns, 0)  # Returns below 0%, else 0

            # Calculate semi-variance (mean of squared downside returns)
            semi_variance = np.mean(downside_returns ** 2)

            # Downside deviation = sqrt of semi-variance
            downside_dev_daily = np.sqrt(semi_variance)

            # Annualize downside deviation (daily to annual: multiply by sqrt(252))
            downside_dev_annual = downside_dev_daily * np.sqrt(252)

            if abs(downside_dev_annual) < self.EPSILON:
                return None, None

            # Calculate Sortino ratio
            sortino = (annualized_return - risk_free_rate) / downside_dev_annual

            return float(sortino), float(downside_dev_annual)

        except Exception as e:
            logger.error(f"Error calculating Sortino ratio: {e}", exc_info=True)
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
            logger.error(f"Error calculating max drawdown: {e}", exc_info=True)
            return None

    def calculate_var_5(self, daily_returns: pd.Series) -> Optional[float]:
        """Calculate Value at Risk at 5% confidence (parametric method).

        Formula: VaR_5% = -1.645 * Daily_Std_Dev

        Args:
            daily_returns: Daily returns series.

        Returns:
            VaR 5% (%) or None.
        """
        if len(daily_returns) < 60:  # Require at least 60 trading days
            return None

        try:
            # Calculate daily standard deviation
            daily_std = daily_returns.std()

            # VaR at 5% confidence (1.645 is the z-score for 95% confidence)
            var_5 = -1.645 * daily_std

            return float(var_5)

        except Exception as e:
            logger.error(f"Error calculating VaR: {e}", exc_info=True)
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
            logger.error(f"Error calculating annualized return: {e}", exc_info=True)
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

            # Get risk-free rate (average 1M Treasury over lookback period)
            risk_free_rate = await self.get_risk_free_rate(trading_days, session)

            # Calculate daily returns
            stock_daily = self.calculate_daily_returns(stock_df)
            benchmark_daily = self.calculate_daily_returns(benchmark_df)

            if stock_daily.empty or benchmark_daily.empty:
                return None

            # Align dates (inner join to match trading days)
            merged = pd.merge(
                stock_daily[['date', 'return']],
                benchmark_daily[['date', 'return']],
                on='date',
                suffixes=('_stock', '_benchmark')
            )

            # Sort by date descending and limit to exact number of trading days requested
            # This ensures we use the most recent N trading days, matching YCharts methodology
            merged = merged.sort_values('date', ascending=False).head(trading_days).sort_values('date')

            # Require at least 60 trading days (~3 months)
            if len(merged) < 60:
                logger.warning(f"{ticker}: Insufficient daily returns for {period} (got {len(merged)}, need 60+)")
                return None

            stock_returns = merged['return_stock']
            benchmark_returns = merged['return_benchmark']

            # Validate and clean NaN values from returns
            if stock_returns.isna().any() or benchmark_returns.isna().any():
                nan_count_stock = stock_returns.isna().sum()
                nan_count_benchmark = benchmark_returns.isna().sum()
                logger.warning(f"{ticker}: Found NaN values in returns (stock: {nan_count_stock}, benchmark: {nan_count_benchmark})")
                # Drop rows with NaN values
                valid_mask = ~(stock_returns.isna() | benchmark_returns.isna())
                stock_returns = stock_returns[valid_mask]
                benchmark_returns = benchmark_returns[valid_mask]

                # Re-check minimum data requirement after dropping NaN
                if len(stock_returns) < 60:
                    logger.warning(f"{ticker}: Insufficient data after dropping NaN for {period}")
                    return None

            # Validate no infinite values
            if np.isinf(stock_returns).any() or np.isinf(benchmark_returns).any():
                logger.warning(f"{ticker}: Found infinite values in returns, skipping")
                return None

            # Calculate annualized returns (YCharts method: average daily return × 252)
            stock_annualized_return = float(stock_returns.mean() * 252)
            benchmark_annualized_return = float(benchmark_returns.mean() * 252)

            # Calculate standard deviation (annualized from daily)
            daily_std = stock_returns.std()
            annualized_std = daily_std * np.sqrt(252)

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
