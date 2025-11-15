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
import sys
from datetime import datetime, timedelta, timezone
from pathlib import Path
from typing import List, Optional, Dict, Any

# Add parent directory to path for imports
sys.path.insert(0, str(Path(__file__).parent.parent))

import pandas as pd
import pandas_ta as ta
from sqlalchemy import text
from sqlalchemy.dialects.postgresql import insert as pg_insert
from sqlalchemy.ext.asyncio import AsyncSession
from tqdm import tqdm

from database.database import get_database
from models import StockPrice, TechnicalIndicator
from pipelines.utils.polygon_client import PolygonClient

# Setup logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s',
    handlers=[
        logging.StreamHandler(),
        logging.FileHandler('technical_indicators_calculator.log')
    ]
)
logger = logging.getLogger(__name__)


class TechnicalIndicatorsCalculator:
    """Calculator for technical indicators using price data."""

    def __init__(self):
        """Initialize the calculator."""
        self.polygon_client = PolygonClient()
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
            # RSI (14-day)
            rsi = ta.rsi(df['close'], length=14)
            current_rsi = rsi.iloc[-1] if not rsi.empty else None

            # MACD (12, 26, 9)
            macd = ta.macd(df['close'], fast=12, slow=26, signal=9)
            current_macd = macd['MACD_12_26_9'].iloc[-1] if macd is not None else None
            current_macd_signal = macd['MACDs_12_26_9'].iloc[-1] if macd is not None else None
            current_macd_hist = macd['MACDh_12_26_9'].iloc[-1] if macd is not None else None

            # SMAs (50-day, 200-day)
            sma_50 = ta.sma(df['close'], length=50)
            sma_200 = ta.sma(df['close'], length=200)
            current_sma_50 = sma_50.iloc[-1] if not sma_50.empty else None
            current_sma_200 = sma_200.iloc[-1] if not sma_200.empty else None

            # EMAs (12-day, 26-day)
            ema_12 = ta.ema(df['close'], length=12)
            ema_26 = ta.ema(df['close'], length=26)
            current_ema_12 = ema_12.iloc[-1] if not ema_12.empty else None
            current_ema_26 = ema_26.iloc[-1] if not ema_26.empty else None

            # Bollinger Bands (20-day, 2 std dev)
            bbands = ta.bbands(df['close'], length=20, std=2)
            current_bb_upper = bbands['BBU_20_2.0'].iloc[-1] if bbands is not None else None
            current_bb_middle = bbands['BBM_20_2.0'].iloc[-1] if bbands is not None else None
            current_bb_lower = bbands['BBL_20_2.0'].iloc[-1] if bbands is not None else None

            # Volume moving average (20-day)
            volume_ma = ta.sma(df['volume'], length=20)
            current_volume_ma = volume_ma.iloc[-1] if not volume_ma.empty else None

            # Momentum metrics
            current_price = df['close'].iloc[-1]

            # Calculate returns over different periods
            momentum = {}
            if len(df) >= 252:
                momentum['12m_return'] = ((current_price / df['close'].iloc[-252]) - 1) * 100
            if len(df) >= 126:
                momentum['6m_return'] = ((current_price / df['close'].iloc[-126]) - 1) * 100
            if len(df) >= 63:
                momentum['3m_return'] = ((current_price / df['close'].iloc[-63]) - 1) * 100
            if len(df) >= 21:
                momentum['1m_return'] = ((current_price / df['close'].iloc[-21]) - 1) * 100

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

            # Insert with ON CONFLICT DO UPDATE
            stmt = pg_insert(StockPrice).values(records)
            stmt = stmt.on_conflict_do_update(
                index_elements=['time', 'ticker'],
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

            # Insert with ON CONFLICT DO UPDATE
            stmt = pg_insert(TechnicalIndicator).values(records)
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
            return False

    async def process_ticker(self, ticker: str, session: AsyncSession) -> bool:
        """Process a single ticker: fetch prices, calculate indicators, store data.

        Args:
            ticker: Stock ticker symbol.
            session: Database session.

        Returns:
            True if successful.
        """
        try:
            # Fetch 252 days of price data (1 trading year)
            prices = self.polygon_client.get_daily_prices(ticker, days=300)

            if not prices or len(prices) < 20:
                logger.warning(f"{ticker}: Insufficient price data")
                return False

            # Convert to DataFrame
            df = pd.DataFrame(prices)
            df.rename(columns={
                'o': 'open',
                'h': 'high',
                'l': 'low',
                'c': 'close',
                'v': 'volume'
            }, inplace=True)

            # Calculate technical indicators
            indicators = self.calculate_technical_indicators(df)

            if not indicators:
                logger.warning(f"{ticker}: No indicators calculated")
                return False

            # Store price data
            await self.store_price_data(ticker, prices, session)

            # Store indicators
            await self.store_indicators(ticker, indicators, session)

            await session.commit()

            logger.info(f"{ticker}: Successfully processed {len(prices)} days, {len(indicators)} indicators")
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

        # Close clients
        self.polygon_client.close()


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
