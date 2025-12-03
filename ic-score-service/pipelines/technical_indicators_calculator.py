#!/usr/bin/env python3
"""Technical Indicators Calculator Pipeline.

This script fetches price data from Polygon.io and calculates technical
indicators (RSI, MACD, SMA, EMA, Bollinger Bands) for stocks.

Usage:
    python technical_indicators_calculator.py --limit 100    # Test on 100 stocks
    python technical_indicators_calculator.py --all          # All stocks
    python technical_indicators_calculator.py --ticker AAPL  # Single stock
"""

import argparse
import asyncio
import logging
import os
import sys
from datetime import datetime, timedelta, timezone
from pathlib import Path
from typing import List, Optional, Dict, Any

# Add parent directory to path for imports
sys.path.insert(0, str(Path(__file__).parent.parent))

import pandas as pd
import talib
from sqlalchemy import text
from sqlalchemy.dialects.postgresql import insert as pg_insert
from sqlalchemy.ext.asyncio import AsyncSession
from tqdm import tqdm

from database.database import get_database
from models import TechnicalIndicator

# Setup logging with configurable log directory
LOG_DIR = os.environ.get('LOG_DIR', '/app/logs')
Path(LOG_DIR).mkdir(parents=True, exist_ok=True)

logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s',
    handlers=[
        logging.StreamHandler(),
        logging.FileHandler(os.path.join(LOG_DIR, 'technical_indicators_calculator.log'))
    ]
)
logger = logging.getLogger(__name__)


class TechnicalIndicatorsCalculator:
    """Calculator for technical indicators using price data."""

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

        Only returns tickers that have price data in stock_prices table,
        excluding index tickers (I:*) and other unsupported asset types.

        Args:
            limit: Maximum number of stocks to process.
            ticker: Single ticker to process.

        Returns:
            List of ticker symbols.
        """
        async with self.db.session() as session:
            if ticker:
                return [ticker.upper()]

            # Query tickers that have at least 20 days of price data (minimum for indicators)
            # Exclude index tickers (I:*), crypto (X:*), and other special tickers
            query = text("""
                SELECT DISTINCT sp.ticker
                FROM stock_prices sp
                WHERE sp.interval = '1day'
                  AND sp.ticker NOT LIKE 'I:%'
                  AND sp.ticker NOT LIKE 'X:%'
                  AND sp.ticker NOT LIKE '%-%'
                GROUP BY sp.ticker
                HAVING COUNT(*) >= 20
                ORDER BY sp.ticker
                LIMIT :limit
            """)
            result = await session.execute(query, {"limit": limit or 10000})
            tickers = [row[0] for row in result.fetchall()]

        logger.info(f"Found {len(tickers)} stocks with sufficient price data to process")
        return tickers

    def calculate_technical_indicators(self, df: pd.DataFrame) -> Dict[str, Any]:
        """Calculate technical indicators from price data.

        Args:
            df: DataFrame with OHLCV data (columns: date, open, high, low, close, volume).

        Returns:
            Dictionary of technical indicators.
        """
        if df.empty or len(df) < 20:
            logger.warning("Insufficient data for technical indicators")
            return {}

        # Ensure data is sorted by date
        df = df.sort_values('date').reset_index(drop=True)

        try:
            # Ensure required columns exist
            required_cols = ['close']
            for col in required_cols:
                if col not in df.columns:
                    logger.error(f"Missing required column: {col}")
                    return {}

            # Convert to numpy arrays for TA-Lib (must be float64)
            close_prices = df['close'].astype('float64').values

            # Handle optional volume column (some data sources may not have it)
            has_volume = 'volume' in df.columns and df['volume'].notna().any()
            volume_data = df['volume'].astype('float64').values if has_volume else None

            # RSI (14-day)
            rsi = talib.RSI(close_prices, timeperiod=14)
            current_rsi = rsi[-1] if len(rsi) > 0 and not pd.isna(rsi[-1]) else None

            # MACD (12, 26, 9)
            macd, macd_signal, macd_hist = talib.MACD(close_prices, fastperiod=12, slowperiod=26, signalperiod=9)
            current_macd = macd[-1] if len(macd) > 0 and not pd.isna(macd[-1]) else None
            current_macd_signal = macd_signal[-1] if len(macd_signal) > 0 and not pd.isna(macd_signal[-1]) else None
            current_macd_hist = macd_hist[-1] if len(macd_hist) > 0 and not pd.isna(macd_hist[-1]) else None

            # SMAs (50-day, 200-day)
            sma_50 = talib.SMA(close_prices, timeperiod=50)
            sma_200 = talib.SMA(close_prices, timeperiod=200)
            current_sma_50 = sma_50[-1] if len(sma_50) > 0 and not pd.isna(sma_50[-1]) else None
            current_sma_200 = sma_200[-1] if len(sma_200) > 0 and not pd.isna(sma_200[-1]) else None

            # EMAs (12-day, 26-day)
            ema_12 = talib.EMA(close_prices, timeperiod=12)
            ema_26 = talib.EMA(close_prices, timeperiod=26)
            current_ema_12 = ema_12[-1] if len(ema_12) > 0 and not pd.isna(ema_12[-1]) else None
            current_ema_26 = ema_26[-1] if len(ema_26) > 0 and not pd.isna(ema_26[-1]) else None

            # Bollinger Bands (20-day, 2 std dev)
            bb_upper, bb_middle, bb_lower = talib.BBANDS(close_prices, timeperiod=20, nbdevup=2, nbdevdn=2, matype=0)
            current_bb_upper = bb_upper[-1] if len(bb_upper) > 0 and not pd.isna(bb_upper[-1]) else None
            current_bb_middle = bb_middle[-1] if len(bb_middle) > 0 and not pd.isna(bb_middle[-1]) else None
            current_bb_lower = bb_lower[-1] if len(bb_lower) > 0 and not pd.isna(bb_lower[-1]) else None

            # Volume moving average (20-day) - only if volume data is available
            current_volume_ma = None
            if volume_data is not None:
                volume_ma = talib.SMA(volume_data, timeperiod=20)
                current_volume_ma = volume_ma[-1] if len(volume_ma) > 0 and not pd.isna(volume_ma[-1]) else None

            # Momentum metrics
            current_price = df['close'].iloc[-1]

            # Calculate returns over different periods with validation
            momentum = {}

            def safe_return(current: float, historical: float) -> Optional[float]:
                """Calculate return with validation for zero/NaN values."""
                if pd.isna(historical) or pd.isna(current) or historical <= 0:
                    return None
                return ((current / historical) - 1) * 100

            if len(df) >= 252:
                ret = safe_return(current_price, df['close'].iloc[-252])
                if ret is not None:
                    momentum['12m_return'] = ret
            if len(df) >= 126:
                ret = safe_return(current_price, df['close'].iloc[-126])
                if ret is not None:
                    momentum['6m_return'] = ret
            if len(df) >= 63:
                ret = safe_return(current_price, df['close'].iloc[-63])
                if ret is not None:
                    momentum['3m_return'] = ret
            if len(df) >= 21:
                ret = safe_return(current_price, df['close'].iloc[-21])
                if ret is not None:
                    momentum['1m_return'] = ret

            return {
                'rsi': current_rsi,
                'macd': current_macd,
                'macd_signal': current_macd_signal,
                'macd_histogram': current_macd_hist,
                'sma_50': current_sma_50,
                'sma_200': current_sma_200,
                'ema_12': current_ema_12,
                'ema_26': current_ema_26,
                'bb_upper': current_bb_upper,
                'bb_middle': current_bb_middle,
                'bb_lower': current_bb_lower,
                'volume_ma_20': current_volume_ma,
                'current_price': current_price,
                **momentum
            }

        except Exception as e:
            logger.error(f"Error calculating indicators: {e}", exc_info=True)
            return {}

    async def store_price_data(
        self,
        ticker: str,
        prices: List[Dict[str, Any]],
        session: AsyncSession
    ) -> bool:
        """Store price data in stock_prices table.

        Args:
            ticker: Stock ticker symbol.
            prices: List of OHLCV price dictionaries.
            session: Database session.

        Returns:
            True if successful.
        """
        try:
            records = []
            for price in prices:
                # Convert date to datetime with timezone
                price_date = price.get('date')
                if isinstance(price_date, str):
                    price_datetime = datetime.strptime(price_date, '%Y-%m-%d')
                else:
                    price_datetime = datetime.combine(price_date, datetime.min.time())

                # Add timezone info
                price_datetime = price_datetime.replace(tzinfo=timezone.utc)

                record = {
                    'time': price_datetime,
                    'ticker': ticker,
                    'open': price.get('o'),
                    'high': price.get('h'),
                    'low': price.get('l'),
                    'close': price.get('c'),
                    'volume': price.get('v'),
                    'vwap': price.get('vw'),
                    'interval': '1day'
                }
                records.append(record)

            if not records:
                return False

            # Insert with ON CONFLICT DO UPDATE (use __table__ for Core-style insert)
            # Note: stock_prices has unique constraint on (ticker, time, interval)
            stmt = pg_insert(StockPrice.__table__).values(records)
            stmt = stmt.on_conflict_do_update(
                index_elements=['ticker', 'time', 'interval'],
                set_={
                    'open': stmt.excluded.open,
                    'high': stmt.excluded.high,
                    'low': stmt.excluded.low,
                    'close': stmt.excluded.close,
                    'volume': stmt.excluded.volume,
                    'vwap': stmt.excluded.vwap,
                }
            )

            await session.execute(stmt)
            return True

        except Exception as e:
            logger.error(f"{ticker}: Error storing price data: {e}", exc_info=True)
            await session.rollback()
            return False

    async def store_indicators(
        self,
        ticker: str,
        indicators: Dict[str, Any],
        session: AsyncSession
    ) -> bool:
        """Store technical indicators in technical_indicators table.

        Args:
            ticker: Stock ticker symbol.
            indicators: Dictionary of indicator values.
            session: Database session.

        Returns:
            True if successful.
        """
        try:
            current_time = datetime.now(timezone.utc)
            records = []

            for indicator_name, value in indicators.items():
                if value is not None and pd.notna(value):
                    record = {
                        'time': current_time,
                        'ticker': ticker,
                        'indicator_name': indicator_name,
                        'value': float(value),
                        'metadata': {'calculated_at': current_time.isoformat()}
                    }
                    records.append(record)

            if not records:
                return False

            # Insert with ON CONFLICT DO UPDATE (use __table__ for Core-style insert)
            stmt = pg_insert(TechnicalIndicator.__table__).values(records)
            stmt = stmt.on_conflict_do_update(
                index_elements=['time', 'ticker', 'indicator_name'],
                set_={
                    'value': stmt.excluded.value,
                    'metadata': stmt.excluded.metadata,
                }
            )

            await session.execute(stmt)
            return True

        except Exception as e:
            logger.error(f"{ticker}: Error storing indicators: {e}", exc_info=True)
            await session.rollback()
            return False

    async def get_prices_from_db(self, ticker: str, session: AsyncSession, days: int = 1150) -> pd.DataFrame:
        """Fetch price data from stock_prices table.

        Args:
            ticker: Stock ticker symbol.
            session: Database session.
            days: Number of days of data to fetch.

        Returns:
            DataFrame with OHLCV data.
        """
        query = text("""
            SELECT time as date, open, high, low, close, volume
            FROM stock_prices
            WHERE ticker = :ticker
              AND interval = '1day'
            ORDER BY time DESC
            LIMIT :days
        """)
        result = await session.execute(query, {"ticker": ticker, "days": days})
        rows = result.fetchall()

        if not rows:
            return pd.DataFrame()

        df = pd.DataFrame(rows, columns=['date', 'open', 'high', 'low', 'close', 'volume'])
        return df

    async def process_ticker(self, ticker: str, session: AsyncSession) -> bool:
        """Process a single ticker: fetch prices, calculate indicators, store data.

        Args:
            ticker: Stock ticker symbol.
            session: Database session.

        Returns:
            True if successful.
        """
        try:
            # Read prices from database (already backfilled with 10 years of data)
            df = await self.get_prices_from_db(ticker, session, days=1150)

            if df.empty or len(df) < 20:
                logger.warning(f"{ticker}: Insufficient price data in database")
                return False

            # Calculate technical indicators
            indicators = self.calculate_technical_indicators(df)

            if not indicators:
                logger.warning(f"{ticker}: No indicators calculated")
                return False

            # Note: Skip storing price data - we already have 15M+ rows from historical_price_backfill
            # The technical indicators calculator only needs to read prices and store indicators

            # Store indicators
            success = await self.store_indicators(ticker, indicators, session)
            if not success:
                logger.warning(f"{ticker}: Failed to store indicators")
                return False

            await session.commit()

            logger.info(f"{ticker}: Successfully processed {len(df)} days, {len(indicators)} indicators")
            return True

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
        logger.info("Technical Indicators Calculator Pipeline")
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
        description='Calculate technical indicators for stocks',
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  # Test on 10 stocks (default)
  python technical_indicators_calculator.py

  # Test on 100 stocks
  python technical_indicators_calculator.py --limit 100

  # Process all stocks
  python technical_indicators_calculator.py --all

  # Process single stock
  python technical_indicators_calculator.py --ticker AAPL
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
    calculator = TechnicalIndicatorsCalculator()
    asyncio.run(calculator.run(
        limit=args.limit,
        ticker=args.ticker,
        all_stocks=args.all
    ))


if __name__ == '__main__':
    main()
