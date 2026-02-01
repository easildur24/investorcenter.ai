"""Score Stabilizer for IC Score v2.1 Phase 3.

This module implements score smoothing to prevent daily whipsaw
while still responding quickly to significant events.

Key features:
- Exponential smoothing with configurable alpha
- Event-based reset triggers (earnings, analyst changes, etc.)
- Minimum change threshold to reduce noise
"""

import logging
from dataclasses import dataclass
from datetime import date, timedelta
from enum import Enum
from typing import Dict, List, Optional, Tuple, Any

from sqlalchemy import text

logger = logging.getLogger(__name__)


class EventType(Enum):
    """Types of events that can trigger score reset."""
    EARNINGS_RELEASE = "earnings_release"
    ANALYST_RATING_CHANGE = "analyst_rating_change"
    INSIDER_TRADE_LARGE = "insider_trade_large"
    DIVIDEND_ANNOUNCEMENT = "dividend_announcement"
    ACQUISITION_NEWS = "acquisition_news"
    GUIDANCE_UPDATE = "guidance_update"
    STOCK_SPLIT = "stock_split"
    PRICE_BREAKOUT = "price_breakout"
    TECHNICAL_SIGNAL = "technical_signal"


@dataclass
class StabilizationResult:
    """Result of score stabilization."""
    final_score: float
    raw_score: float
    smoothing_applied: bool
    previous_score: Optional[float]
    detected_events: List[str]
    change_delta: float


@dataclass
class DetectedEvent:
    """A detected market event."""
    event_type: EventType
    event_date: date
    description: str
    impact_direction: str  # 'positive', 'negative', 'neutral'
    impact_magnitude: Optional[float] = None
    source: Optional[str] = None
    metadata: Optional[Dict[str, Any]] = None


class ScoreStabilizer:
    """Apply score smoothing to prevent daily whipsaw.

    Uses exponential moving average (EMA) smoothing:
    - New score = ALPHA * raw_score + (1 - ALPHA) * previous_score
    - ALPHA = 0.7 means 70% weight on new score, 30% on previous

    Reset events bypass smoothing to quickly reflect material changes:
    - Earnings releases
    - Analyst rating changes
    - Large insider trades
    - Dividend announcements
    - M&A news
    - Guidance updates
    """

    # Smoothing parameters
    ALPHA = 0.7  # 70% new, 30% previous
    MIN_CHANGE_THRESHOLD = 0.5  # Minimum change to update score

    # Events that bypass smoothing (score resets immediately)
    RESET_EVENTS = {
        EventType.EARNINGS_RELEASE,
        EventType.ANALYST_RATING_CHANGE,
        EventType.INSIDER_TRADE_LARGE,
        EventType.DIVIDEND_ANNOUNCEMENT,
        EventType.ACQUISITION_NEWS,
        EventType.GUIDANCE_UPDATE,
    }

    # Thresholds for event detection
    LARGE_INSIDER_TRADE_THRESHOLD = 100000  # $100k
    SIGNIFICANT_PRICE_MOVE_PCT = 5.0  # 5%

    def __init__(self, session):
        """Initialize stabilizer with database session.

        Args:
            session: SQLAlchemy async session for database queries.
        """
        self.session = session

    async def stabilize(
        self,
        ticker: str,
        new_score: float,
        previous_score: Optional[float] = None,
        events: Optional[List[str]] = None
    ) -> StabilizationResult:
        """Apply smoothing to score unless significant event occurred.

        Args:
            ticker: Stock ticker symbol.
            new_score: Newly calculated raw score.
            previous_score: Previous day's score (optional, will fetch if not provided).
            events: Pre-detected events (optional, will detect if not provided).

        Returns:
            StabilizationResult with final score and metadata.
        """
        raw_score = new_score

        # Fetch previous score if not provided
        if previous_score is None:
            previous_score = await self._get_previous_score(ticker)

        # Detect events if not provided
        if events is None:
            detected_events = await self.detect_events(ticker)
            event_types = [e.event_type.value for e in detected_events]
        else:
            event_types = events
            detected_events = []

        # No previous score - use new score directly (first calculation)
        if previous_score is None:
            return StabilizationResult(
                final_score=round(new_score, 2),
                raw_score=raw_score,
                smoothing_applied=False,
                previous_score=None,
                detected_events=event_types,
                change_delta=0
            )

        # Check for reset events - bypass smoothing
        has_reset_event = any(
            EventType(e) in self.RESET_EVENTS
            for e in event_types
            if e in [et.value for et in EventType]
        )

        if has_reset_event:
            logger.debug(f"{ticker}: Reset event detected, bypassing smoothing")
            return StabilizationResult(
                final_score=round(new_score, 2),
                raw_score=raw_score,
                smoothing_applied=False,
                previous_score=previous_score,
                detected_events=event_types,
                change_delta=round(new_score - previous_score, 2)
            )

        # Apply exponential smoothing
        smoothed = self.ALPHA * new_score + (1 - self.ALPHA) * previous_score

        # Check if change exceeds minimum threshold
        change = smoothed - previous_score
        if abs(change) < self.MIN_CHANGE_THRESHOLD:
            # Change too small, keep previous score
            logger.debug(
                f"{ticker}: Change {change:.2f} below threshold, keeping previous score"
            )
            return StabilizationResult(
                final_score=round(previous_score, 2),
                raw_score=raw_score,
                smoothing_applied=True,
                previous_score=previous_score,
                detected_events=event_types,
                change_delta=0
            )

        # Apply smoothed score
        final_score = round(smoothed, 1)

        return StabilizationResult(
            final_score=final_score,
            raw_score=raw_score,
            smoothing_applied=True,
            previous_score=previous_score,
            detected_events=event_types,
            change_delta=round(final_score - previous_score, 2)
        )

    async def detect_events(
        self,
        ticker: str,
        since_date: Optional[date] = None
    ) -> List[DetectedEvent]:
        """Detect significant events since last calculation.

        Args:
            ticker: Stock ticker symbol.
            since_date: Look for events since this date (default: yesterday).

        Returns:
            List of detected events.
        """
        if since_date is None:
            since_date = date.today() - timedelta(days=1)

        events = []

        # Check earnings calendar
        earnings_event = await self._check_earnings(ticker, since_date)
        if earnings_event:
            events.append(earnings_event)

        # Check analyst ratings
        analyst_event = await self._check_analyst_ratings(ticker, since_date)
        if analyst_event:
            events.append(analyst_event)

        # Check insider trades
        insider_event = await self._check_insider_trades(ticker, since_date)
        if insider_event:
            events.append(insider_event)

        # Check dividend announcements
        dividend_event = await self._check_dividends(ticker, since_date)
        if dividend_event:
            events.append(dividend_event)

        # Check price breakouts
        price_event = await self._check_price_breakout(ticker, since_date)
        if price_event:
            events.append(price_event)

        return events

    async def _get_previous_score(self, ticker: str) -> Optional[float]:
        """Get the most recent IC Score for a ticker."""
        try:
            query = text("""
                SELECT overall_score
                FROM ic_scores
                WHERE ticker = :ticker
                  AND date < CURRENT_DATE
                ORDER BY date DESC
                LIMIT 1
            """)

            result = await self.session.execute(query, {"ticker": ticker})
            row = result.fetchone()

            if row and row[0]:
                return float(row[0])

            return None

        except Exception as e:
            logger.error(f"Error fetching previous score for {ticker}: {e}")
            return None

    async def _check_earnings(
        self,
        ticker: str,
        since_date: date
    ) -> Optional[DetectedEvent]:
        """Check for recent earnings release."""
        try:
            # Check SEC filings for 10-Q or 10-K
            query = text("""
                SELECT form_type, filing_date
                FROM sec_filings
                WHERE ticker = :ticker
                  AND filing_date >= :since_date
                  AND form_type IN ('10-Q', '10-K', '8-K')
                ORDER BY filing_date DESC
                LIMIT 1
            """)

            result = await self.session.execute(
                query, {"ticker": ticker, "since_date": since_date}
            )
            row = result.fetchone()

            if row:
                form_type = row[0]
                filing_date = row[1]

                return DetectedEvent(
                    event_type=EventType.EARNINGS_RELEASE,
                    event_date=filing_date,
                    description=f"{form_type} filing on {filing_date}",
                    impact_direction="neutral",
                    source="sec_filings"
                )

            return None

        except Exception as e:
            logger.debug(f"Error checking earnings for {ticker}: {e}")
            return None

    async def _check_analyst_ratings(
        self,
        ticker: str,
        since_date: date
    ) -> Optional[DetectedEvent]:
        """Check for recent analyst rating changes."""
        try:
            query = text("""
                SELECT rating, previous_rating, analyst_firm, rating_date
                FROM analyst_ratings
                WHERE ticker = :ticker
                  AND rating_date >= :since_date
                  AND previous_rating IS NOT NULL
                  AND rating != previous_rating
                ORDER BY rating_date DESC
                LIMIT 1
            """)

            result = await self.session.execute(
                query, {"ticker": ticker, "since_date": since_date}
            )
            row = result.fetchone()

            if row:
                new_rating = row[0]
                prev_rating = row[1]
                firm = row[2]
                rating_date = row[3]

                # Determine direction
                upgrade_keywords = ['buy', 'outperform', 'overweight']
                downgrade_keywords = ['sell', 'underperform', 'underweight']

                new_is_positive = any(k in (new_rating or '').lower() for k in upgrade_keywords)
                prev_is_positive = any(k in (prev_rating or '').lower() for k in upgrade_keywords)

                if new_is_positive and not prev_is_positive:
                    direction = "positive"
                elif not new_is_positive and prev_is_positive:
                    direction = "negative"
                else:
                    direction = "neutral"

                return DetectedEvent(
                    event_type=EventType.ANALYST_RATING_CHANGE,
                    event_date=rating_date,
                    description=f"{firm}: {prev_rating} → {new_rating}",
                    impact_direction=direction,
                    source="analyst_ratings",
                    metadata={"firm": firm, "new_rating": new_rating, "prev_rating": prev_rating}
                )

            return None

        except Exception as e:
            logger.debug(f"Error checking analyst ratings for {ticker}: {e}")
            return None

    async def _check_insider_trades(
        self,
        ticker: str,
        since_date: date
    ) -> Optional[DetectedEvent]:
        """Check for large insider trades."""
        try:
            query = text("""
                SELECT
                    transaction_type,
                    SUM(total_value) as total_value,
                    COUNT(*) as trade_count,
                    MAX(transaction_date) as latest_date
                FROM insider_trades
                WHERE ticker = :ticker
                  AND transaction_date >= :since_date
                  AND total_value > :threshold
                GROUP BY transaction_type
                ORDER BY total_value DESC
                LIMIT 1
            """)

            result = await self.session.execute(
                query,
                {
                    "ticker": ticker,
                    "since_date": since_date,
                    "threshold": self.LARGE_INSIDER_TRADE_THRESHOLD
                }
            )
            row = result.fetchone()

            if row:
                trans_type = row[0]
                total_value = row[1]
                trade_count = row[2]
                latest_date = row[3]

                is_buy = trans_type and 'buy' in trans_type.lower()
                direction = "positive" if is_buy else "negative"

                return DetectedEvent(
                    event_type=EventType.INSIDER_TRADE_LARGE,
                    event_date=latest_date,
                    description=f"Insider {trans_type}: ${total_value:,.0f} ({trade_count} trades)",
                    impact_direction=direction,
                    impact_magnitude=float(total_value),
                    source="insider_trades"
                )

            return None

        except Exception as e:
            logger.debug(f"Error checking insider trades for {ticker}: {e}")
            return None

    async def _check_dividends(
        self,
        ticker: str,
        since_date: date
    ) -> Optional[DetectedEvent]:
        """Check for dividend announcements."""
        try:
            query = text("""
                SELECT ex_dividend_date, annual_dividend, dividend_growth_yoy
                FROM dividend_history
                WHERE ticker = :ticker
                  AND ex_dividend_date >= :since_date
                ORDER BY ex_dividend_date DESC
                LIMIT 1
            """)

            result = await self.session.execute(
                query, {"ticker": ticker, "since_date": since_date}
            )
            row = result.fetchone()

            if row:
                ex_date = row[0]
                div_amount = row[1]
                growth = row[2]

                direction = "positive" if growth and float(growth) > 0 else "neutral"

                return DetectedEvent(
                    event_type=EventType.DIVIDEND_ANNOUNCEMENT,
                    event_date=ex_date,
                    description=f"Ex-dividend date: {ex_date}, ${div_amount}",
                    impact_direction=direction,
                    source="dividend_history"
                )

            return None

        except Exception as e:
            logger.debug(f"Error checking dividends for {ticker}: {e}")
            return None

    async def _check_price_breakout(
        self,
        ticker: str,
        since_date: date
    ) -> Optional[DetectedEvent]:
        """Check for significant price movements."""
        try:
            query = text("""
                SELECT
                    value as return_1d
                FROM technical_indicators
                WHERE ticker = :ticker
                  AND indicator_name = '1d_return'
                  AND time >= :since_date
                ORDER BY time DESC
                LIMIT 1
            """)

            result = await self.session.execute(
                query, {"ticker": ticker, "since_date": since_date}
            )
            row = result.fetchone()

            if row and row[0]:
                return_1d = float(row[0])

                if abs(return_1d) >= self.SIGNIFICANT_PRICE_MOVE_PCT:
                    direction = "positive" if return_1d > 0 else "negative"

                    return DetectedEvent(
                        event_type=EventType.PRICE_BREAKOUT,
                        event_date=date.today(),
                        description=f"Price moved {return_1d:+.1f}%",
                        impact_direction=direction,
                        impact_magnitude=abs(return_1d),
                        source="technical_indicators"
                    )

            return None

        except Exception as e:
            logger.debug(f"Error checking price breakout for {ticker}: {e}")
            return None

    async def store_event(self, ticker: str, event: DetectedEvent) -> bool:
        """Store a detected event in the database."""
        try:
            query = text("""
                INSERT INTO ic_score_events (
                    ticker, event_type, event_date, description,
                    impact_direction, impact_magnitude, source, metadata
                ) VALUES (
                    :ticker, :event_type, :event_date, :description,
                    :impact_direction, :impact_magnitude, :source, :metadata
                )
                ON CONFLICT DO NOTHING
            """)

            await self.session.execute(query, {
                "ticker": ticker,
                "event_type": event.event_type.value,
                "event_date": event.event_date,
                "description": event.description,
                "impact_direction": event.impact_direction,
                "impact_magnitude": event.impact_magnitude,
                "source": event.source,
                "metadata": event.metadata
            })

            return True

        except Exception as e:
            logger.error(f"Error storing event for {ticker}: {e}")
            return False
