"""Catalyst Detector Service for IC Score v2.1 Phase 3.

This module detects upcoming and recent catalysts that may impact
a stock's price or IC Score. Catalysts include:
- Earnings releases
- Analyst rating changes
- Insider trades
- Technical breakouts
- Dividend dates
- 52-week high/low proximity

Key features:
- Multiple detector types
- Upcoming vs recent catalysts
- Impact assessment (bullish/bearish/neutral)
"""

import logging
from abc import ABC, abstractmethod
from dataclasses import dataclass
from datetime import date, timedelta
from enum import Enum
from typing import Dict, List, Optional, Any

from sqlalchemy import text

logger = logging.getLogger(__name__)


class CatalystType(Enum):
    """Types of catalysts."""
    EARNINGS = "earnings"
    ANALYST_RATING = "analyst_rating"
    INSIDER_TRADE = "insider_trade"
    TECHNICAL_BREAKOUT = "technical_breakout"
    DIVIDEND_DATE = "dividend_date"
    FIFTY_TWO_WEEK = "52_week_high_low"
    NEWS_SENTIMENT = "news_sentiment"
    INSTITUTIONAL_CHANGE = "institutional_change"


class CatalystImpact(Enum):
    """Impact direction of catalyst."""
    BULLISH = "bullish"
    BEARISH = "bearish"
    NEUTRAL = "neutral"


@dataclass
class Catalyst:
    """A detected catalyst event."""
    event_type: CatalystType
    title: str
    description: Optional[str]
    event_date: Optional[date]
    icon: str
    impact: CatalystImpact
    confidence: float  # 0-1 confidence level
    days_until: int  # Negative = past, positive = future
    source: str
    metadata: Optional[Dict[str, Any]] = None


class CatalystDetector(ABC):
    """Base class for catalyst detectors."""

    @abstractmethod
    async def detect(self, session, ticker: str) -> Optional[Catalyst]:
        """Detect catalyst for a ticker.

        Args:
            session: Database session.
            ticker: Stock ticker symbol.

        Returns:
            Catalyst if detected, None otherwise.
        """
        pass


class EarningsDetector(CatalystDetector):
    """Detect upcoming or recent earnings releases."""

    ICON = "📊"
    LOOKBACK_DAYS = 7
    LOOKAHEAD_DAYS = 30

    async def detect(self, session, ticker: str) -> Optional[Catalyst]:
        """Detect earnings catalyst."""
        try:
            # Check for upcoming earnings (from SEC filings pattern)
            # Most companies report within ~45 days of quarter end
            query = text("""
                WITH last_filing AS (
                    SELECT MAX(filing_date) as last_date
                    FROM sec_filings
                    WHERE ticker = :ticker
                      AND form_type IN ('10-Q', '10-K')
                ),
                expected_next AS (
                    SELECT
                        CASE
                            WHEN last_date IS NOT NULL
                            THEN last_date + INTERVAL '90 days'
                            ELSE NULL
                        END as expected_date
                    FROM last_filing
                )
                SELECT expected_date
                FROM expected_next
                WHERE expected_date BETWEEN CURRENT_DATE - INTERVAL '7 days'
                  AND CURRENT_DATE + INTERVAL '45 days'
            """)

            result = await session.execute(query, {"ticker": ticker})
            row = result.fetchone()

            if row and row[0]:
                expected_date = row[0].date() if hasattr(row[0], 'date') else row[0]
                days_until = (expected_date - date.today()).days

                if days_until > 0:
                    title = f"Earnings expected in {days_until} days"
                elif days_until == 0:
                    title = "Earnings release today"
                else:
                    title = f"Earnings released {-days_until} days ago"

                return Catalyst(
                    event_type=CatalystType.EARNINGS,
                    title=title,
                    description="Quarterly earnings report",
                    event_date=expected_date,
                    icon=self.ICON,
                    impact=CatalystImpact.NEUTRAL,  # Unknown until results
                    confidence=0.7,
                    days_until=days_until,
                    source="sec_filings"
                )

            return None

        except Exception as e:
            logger.debug(f"Error detecting earnings for {ticker}: {e}")
            return None


class AnalystRatingDetector(CatalystDetector):
    """Detect recent analyst rating changes."""

    ICON_UPGRADE = "📈"
    ICON_DOWNGRADE = "📉"
    LOOKBACK_DAYS = 7

    async def detect(self, session, ticker: str) -> Optional[Catalyst]:
        """Detect analyst rating catalyst."""
        try:
            query = text("""
                SELECT rating, previous_rating, analyst_firm, rating_date
                FROM analyst_ratings
                WHERE ticker = :ticker
                  AND rating_date >= CURRENT_DATE - INTERVAL :lookback
                  AND previous_rating IS NOT NULL
                  AND rating != previous_rating
                ORDER BY rating_date DESC
                LIMIT 1
            """)

            result = await session.execute(
                query,
                {"ticker": ticker, "lookback": f"{self.LOOKBACK_DAYS} days"}
            )
            row = result.fetchone()

            if row:
                new_rating = row[0]
                prev_rating = row[1]
                firm = row[2]
                rating_date = row[3]

                days_until = (rating_date - date.today()).days

                # Determine if upgrade or downgrade
                upgrade_keywords = ['buy', 'outperform', 'overweight', 'strong buy']
                downgrade_keywords = ['sell', 'underperform', 'underweight', 'reduce']

                new_positive = any(k in (new_rating or '').lower() for k in upgrade_keywords)
                new_negative = any(k in (new_rating or '').lower() for k in downgrade_keywords)
                prev_positive = any(k in (prev_rating or '').lower() for k in upgrade_keywords)

                if new_positive and not prev_positive:
                    impact = CatalystImpact.BULLISH
                    icon = self.ICON_UPGRADE
                    action = "upgraded"
                elif new_negative:
                    impact = CatalystImpact.BEARISH
                    icon = self.ICON_DOWNGRADE
                    action = "downgraded"
                else:
                    impact = CatalystImpact.NEUTRAL
                    icon = self.ICON_UPGRADE
                    action = "changed rating"

                return Catalyst(
                    event_type=CatalystType.ANALYST_RATING,
                    title=f"{firm} {action} to {new_rating}",
                    description=f"From {prev_rating}",
                    event_date=rating_date,
                    icon=icon,
                    impact=impact,
                    confidence=0.85,
                    days_until=days_until,
                    source="analyst_ratings",
                    metadata={"firm": firm, "new": new_rating, "prev": prev_rating}
                )

            return None

        except Exception as e:
            logger.debug(f"Error detecting analyst rating for {ticker}: {e}")
            return None


class InsiderTradeDetector(CatalystDetector):
    """Detect significant insider trades."""

    ICON_BUY = "🛒"
    ICON_SELL = "💰"
    LOOKBACK_DAYS = 14
    MIN_VALUE = 50000  # $50k minimum

    async def detect(self, session, ticker: str) -> Optional[Catalyst]:
        """Detect insider trade catalyst."""
        try:
            query = text("""
                SELECT
                    transaction_type,
                    insider_name,
                    insider_title,
                    SUM(total_value) as total_value,
                    MAX(transaction_date) as latest_date
                FROM insider_trades
                WHERE ticker = :ticker
                  AND transaction_date >= CURRENT_DATE - INTERVAL :lookback
                  AND total_value >= :min_value
                GROUP BY transaction_type, insider_name, insider_title
                ORDER BY total_value DESC
                LIMIT 1
            """)

            result = await session.execute(
                query,
                {
                    "ticker": ticker,
                    "lookback": f"{self.LOOKBACK_DAYS} days",
                    "min_value": self.MIN_VALUE
                }
            )
            row = result.fetchone()

            if row:
                trans_type = row[0]
                name = row[1]
                title = row[2]
                value = row[3]
                trans_date = row[4]

                days_until = (trans_date - date.today()).days

                is_buy = trans_type and 'buy' in trans_type.lower()

                return Catalyst(
                    event_type=CatalystType.INSIDER_TRADE,
                    title=f"{name or 'Insider'} {'bought' if is_buy else 'sold'} ${value:,.0f}",
                    description=f"{title or 'Executive'} transaction",
                    event_date=trans_date,
                    icon=self.ICON_BUY if is_buy else self.ICON_SELL,
                    impact=CatalystImpact.BULLISH if is_buy else CatalystImpact.BEARISH,
                    confidence=0.75,
                    days_until=days_until,
                    source="insider_trades",
                    metadata={"name": name, "title": title, "value": float(value)}
                )

            return None

        except Exception as e:
            logger.debug(f"Error detecting insider trade for {ticker}: {e}")
            return None


class TechnicalBreakoutDetector(CatalystDetector):
    """Detect technical breakouts or breakdowns."""

    ICON_BREAKOUT = "🚀"
    ICON_BREAKDOWN = "⚠️"

    async def detect(self, session, ticker: str) -> Optional[Catalyst]:
        """Detect technical breakout catalyst."""
        try:
            query = text("""
                SELECT
                    value as current_price,
                    (SELECT value FROM technical_indicators
                     WHERE ticker = :ticker AND indicator_name = 'sma_50'
                     ORDER BY time DESC LIMIT 1) as sma_50,
                    (SELECT value FROM technical_indicators
                     WHERE ticker = :ticker AND indicator_name = 'sma_200'
                     ORDER BY time DESC LIMIT 1) as sma_200,
                    (SELECT value FROM technical_indicators
                     WHERE ticker = :ticker AND indicator_name = 'rsi'
                     ORDER BY time DESC LIMIT 1) as rsi
                FROM technical_indicators
                WHERE ticker = :ticker AND indicator_name = 'current_price'
                ORDER BY time DESC
                LIMIT 1
            """)

            result = await session.execute(query, {"ticker": ticker})
            row = result.fetchone()

            if row and row[0] and row[1]:
                price = float(row[0])
                sma_50 = float(row[1]) if row[1] else None
                sma_200 = float(row[2]) if row[2] else None
                rsi = float(row[3]) if row[3] else None

                # Check for golden cross (50 > 200)
                if sma_50 and sma_200 and sma_50 > sma_200:
                    if price > sma_50:
                        return Catalyst(
                            event_type=CatalystType.TECHNICAL_BREAKOUT,
                            title="Golden Cross: Price above 50/200 SMA",
                            description="Bullish technical pattern",
                            event_date=date.today(),
                            icon=self.ICON_BREAKOUT,
                            impact=CatalystImpact.BULLISH,
                            confidence=0.65,
                            days_until=0,
                            source="technical_indicators"
                        )

                # Check for death cross (50 < 200)
                if sma_50 and sma_200 and sma_50 < sma_200:
                    if price < sma_50:
                        return Catalyst(
                            event_type=CatalystType.TECHNICAL_BREAKOUT,
                            title="Death Cross: Price below 50/200 SMA",
                            description="Bearish technical pattern",
                            event_date=date.today(),
                            icon=self.ICON_BREAKDOWN,
                            impact=CatalystImpact.BEARISH,
                            confidence=0.65,
                            days_until=0,
                            source="technical_indicators"
                        )

                # Check RSI extremes
                if rsi:
                    if rsi < 30:
                        return Catalyst(
                            event_type=CatalystType.TECHNICAL_BREAKOUT,
                            title=f"RSI Oversold: {rsi:.1f}",
                            description="Potential reversal signal",
                            event_date=date.today(),
                            icon="📉",
                            impact=CatalystImpact.BULLISH,  # Oversold = potential bounce
                            confidence=0.60,
                            days_until=0,
                            source="technical_indicators"
                        )
                    elif rsi > 70:
                        return Catalyst(
                            event_type=CatalystType.TECHNICAL_BREAKOUT,
                            title=f"RSI Overbought: {rsi:.1f}",
                            description="Potential reversal signal",
                            event_date=date.today(),
                            icon="📈",
                            impact=CatalystImpact.BEARISH,  # Overbought = potential pullback
                            confidence=0.60,
                            days_until=0,
                            source="technical_indicators"
                        )

            return None

        except Exception as e:
            logger.debug(f"Error detecting technical breakout for {ticker}: {e}")
            return None


class DividendDateDetector(CatalystDetector):
    """Detect upcoming dividend dates."""

    ICON = "💵"
    LOOKAHEAD_DAYS = 30

    async def detect(self, session, ticker: str) -> Optional[Catalyst]:
        """Detect dividend date catalyst."""
        try:
            query = text("""
                SELECT ex_dividend_date, annual_dividend, dividend_yield
                FROM dividend_history
                WHERE ticker = :ticker
                  AND ex_dividend_date BETWEEN CURRENT_DATE AND CURRENT_DATE + INTERVAL :lookahead
                ORDER BY ex_dividend_date ASC
                LIMIT 1
            """)

            result = await session.execute(
                query,
                {"ticker": ticker, "lookahead": f"{self.LOOKAHEAD_DAYS} days"}
            )
            row = result.fetchone()

            if row and row[0]:
                ex_date = row[0]
                div_amount = row[1]
                div_yield = row[2]

                days_until = (ex_date - date.today()).days

                return Catalyst(
                    event_type=CatalystType.DIVIDEND_DATE,
                    title=f"Ex-dividend in {days_until} days",
                    description=f"Yield: {div_yield:.2f}%" if div_yield else None,
                    event_date=ex_date,
                    icon=self.ICON,
                    impact=CatalystImpact.NEUTRAL,
                    confidence=0.95,
                    days_until=days_until,
                    source="dividend_history",
                    metadata={"amount": float(div_amount) if div_amount else None}
                )

            return None

        except Exception as e:
            logger.debug(f"Error detecting dividend date for {ticker}: {e}")
            return None


class FiftyTwoWeekDetector(CatalystDetector):
    """Detect proximity to 52-week high or low."""

    ICON_HIGH = "🏆"
    ICON_LOW = "🔻"
    THRESHOLD_PCT = 5  # Within 5% of high/low

    async def detect(self, session, ticker: str) -> Optional[Catalyst]:
        """Detect 52-week high/low catalyst."""
        try:
            query = text("""
                SELECT
                    (SELECT value FROM technical_indicators
                     WHERE ticker = :ticker AND indicator_name = 'current_price'
                     ORDER BY time DESC LIMIT 1) as current_price,
                    (SELECT value FROM technical_indicators
                     WHERE ticker = :ticker AND indicator_name = '52w_high'
                     ORDER BY time DESC LIMIT 1) as high_52w,
                    (SELECT value FROM technical_indicators
                     WHERE ticker = :ticker AND indicator_name = '52w_low'
                     ORDER BY time DESC LIMIT 1) as low_52w
            """)

            result = await session.execute(query, {"ticker": ticker})
            row = result.fetchone()

            if row and row[0] and row[1] and row[2]:
                price = float(row[0])
                high_52w = float(row[1])
                low_52w = float(row[2])

                # Check proximity to 52-week high
                pct_from_high = ((high_52w - price) / high_52w) * 100
                if pct_from_high <= self.THRESHOLD_PCT:
                    return Catalyst(
                        event_type=CatalystType.FIFTY_TWO_WEEK,
                        title=f"Near 52-week high ({pct_from_high:.1f}% away)",
                        description=f"52-week high: ${high_52w:.2f}",
                        event_date=date.today(),
                        icon=self.ICON_HIGH,
                        impact=CatalystImpact.BULLISH,
                        confidence=0.80,
                        days_until=0,
                        source="technical_indicators",
                        metadata={"high_52w": high_52w, "pct_from_high": pct_from_high}
                    )

                # Check proximity to 52-week low
                pct_from_low = ((price - low_52w) / low_52w) * 100
                if pct_from_low <= self.THRESHOLD_PCT:
                    return Catalyst(
                        event_type=CatalystType.FIFTY_TWO_WEEK,
                        title=f"Near 52-week low ({pct_from_low:.1f}% above)",
                        description=f"52-week low: ${low_52w:.2f}",
                        event_date=date.today(),
                        icon=self.ICON_LOW,
                        impact=CatalystImpact.BEARISH,
                        confidence=0.80,
                        days_until=0,
                        source="technical_indicators",
                        metadata={"low_52w": low_52w, "pct_from_low": pct_from_low}
                    )

            return None

        except Exception as e:
            logger.debug(f"Error detecting 52-week high/low for {ticker}: {e}")
            return None


class CatalystService:
    """Service for detecting all catalysts for a stock."""

    # All detector types
    DETECTORS = [
        EarningsDetector(),
        AnalystRatingDetector(),
        InsiderTradeDetector(),
        TechnicalBreakoutDetector(),
        DividendDateDetector(),
        FiftyTwoWeekDetector(),
    ]

    def __init__(self, session):
        """Initialize service with database session."""
        self.session = session

    async def get_catalysts(
        self,
        ticker: str,
        limit: int = 5
    ) -> List[Catalyst]:
        """Get all catalysts for a stock.

        Args:
            ticker: Stock ticker symbol.
            limit: Maximum number of catalysts to return.

        Returns:
            List of catalysts sorted by date.
        """
        catalysts = []

        for detector in self.DETECTORS:
            try:
                catalyst = await detector.detect(self.session, ticker)
                if catalyst:
                    catalysts.append(catalyst)
            except Exception as e:
                logger.warning(f"Error in {detector.__class__.__name__}: {e}")

        # Sort by days_until (upcoming first, then recent)
        catalysts.sort(key=lambda x: (x.days_until < 0, abs(x.days_until)))

        return catalysts[:limit]

    async def store_catalysts(
        self,
        ticker: str,
        catalysts: List[Catalyst]
    ) -> bool:
        """Store catalysts in database."""
        try:
            for catalyst in catalysts:
                query = text("""
                    INSERT INTO catalyst_events (
                        ticker, event_type, title, description,
                        event_date, icon, impact, confidence,
                        days_until, source, metadata, expires_at
                    ) VALUES (
                        :ticker, :event_type, :title, :description,
                        :event_date, :icon, :impact, :confidence,
                        :days_until, :source, :metadata,
                        CURRENT_DATE + INTERVAL '30 days'
                    )
                """)

                await self.session.execute(query, {
                    "ticker": ticker,
                    "event_type": catalyst.event_type.value,
                    "title": catalyst.title,
                    "description": catalyst.description,
                    "event_date": catalyst.event_date,
                    "icon": catalyst.icon,
                    "impact": catalyst.impact.value,
                    "confidence": catalyst.confidence,
                    "days_until": catalyst.days_until,
                    "source": catalyst.source,
                    "metadata": catalyst.metadata
                })

            return True

        except Exception as e:
            logger.error(f"Error storing catalysts for {ticker}: {e}")
            return False
