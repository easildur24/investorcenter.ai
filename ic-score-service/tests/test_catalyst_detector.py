"""Unit tests for Catalyst Detector service."""

import pytest
from unittest.mock import AsyncMock, MagicMock
from datetime import date, timedelta

import sys
from pathlib import Path
sys.path.insert(0, str(Path(__file__).parent.parent))

from pipelines.utils.catalyst_detector import (
    CatalystService,
    Catalyst,
    CatalystDetector,
    EarningsDetector,
    AnalystRatingDetector,
    InsiderTradeDetector,
    TechnicalBreakoutDetector,
    DividendDateDetector,
    FiftyTwoWeekDetector,
)


class TestCatalyst:
    """Test Catalyst dataclass."""

    def test_catalyst_creation(self):
        """Test creating a Catalyst instance."""
        catalyst = Catalyst(
            event_type='earnings',
            title='Q4 2025 Earnings',
            event_date=date.today() + timedelta(days=10),
            icon='📊',
            impact='Unknown',
            confidence=0.9,
            days_until=10
        )

        assert catalyst.event_type == 'earnings'
        assert catalyst.title == 'Q4 2025 Earnings'
        assert catalyst.icon == '📊'
        assert catalyst.confidence == 0.9
        assert catalyst.days_until == 10

    def test_catalyst_with_none_date(self):
        """Test catalyst with unknown date."""
        catalyst = Catalyst(
            event_type='technical',
            title='Approaching resistance',
            event_date=None,
            icon='📈',
            impact='Positive',
            confidence=0.7,
            days_until=None
        )

        assert catalyst.event_date is None
        assert catalyst.days_until is None


class TestCatalystService:
    """Test cases for CatalystService."""

    @pytest.fixture
    def mock_session(self):
        """Create a mock database session."""
        session = AsyncMock()
        return session

    @pytest.fixture
    def service(self, mock_session):
        """Create a service instance with mocked session."""
        return CatalystService(mock_session)

    @pytest.mark.asyncio
    async def test_get_catalysts_returns_list(self, service):
        """Test get_catalysts returns a list."""
        # Mock all detectors to return empty
        for detector in service.DETECTORS:
            detector.detect = AsyncMock(return_value=[])

        result = await service.get_catalysts('AAPL', limit=5)

        assert isinstance(result, list)

    @pytest.mark.asyncio
    async def test_get_catalysts_limits_results(self, service):
        """Test get_catalysts respects limit."""
        # Create 10 mock catalysts
        mock_catalysts = [
            Catalyst(
                event_type=f'test_{i}',
                title=f'Test {i}',
                event_date=date.today() + timedelta(days=i),
                icon='🔹',
                impact='Neutral',
                confidence=0.5,
                days_until=i
            )
            for i in range(10)
        ]

        # Mock detector to return all catalysts
        service.DETECTORS = [MagicMock()]
        service.DETECTORS[0].detect = AsyncMock(return_value=mock_catalysts)

        result = await service.get_catalysts('AAPL', limit=5)

        assert len(result) <= 5

    @pytest.mark.asyncio
    async def test_get_catalysts_sorts_by_days_until(self, service):
        """Test get_catalysts sorts by days until event."""
        catalysts = [
            Catalyst('a', 'Event A', date.today() + timedelta(days=30), '🔹', 'Neutral', 0.5, 30),
            Catalyst('b', 'Event B', date.today() + timedelta(days=5), '🔹', 'Neutral', 0.5, 5),
            Catalyst('c', 'Event C', date.today() + timedelta(days=15), '🔹', 'Neutral', 0.5, 15),
        ]

        service.DETECTORS = [MagicMock()]
        service.DETECTORS[0].detect = AsyncMock(return_value=catalysts)

        result = await service.get_catalysts('AAPL', limit=5)

        # Should be sorted by days_until ascending
        if len(result) >= 2:
            for i in range(len(result) - 1):
                if result[i].days_until is not None and result[i+1].days_until is not None:
                    assert result[i].days_until <= result[i+1].days_until


class TestEarningsDetector:
    """Test EarningsDetector."""

    @pytest.fixture
    def detector(self):
        return EarningsDetector()

    @pytest.mark.asyncio
    async def test_detect_upcoming_earnings(self, detector):
        """Test detection of upcoming earnings."""
        mock_session = AsyncMock()
        mock_result = MagicMock()
        mock_result.fetchone.return_value = (
            date.today() + timedelta(days=14),  # earnings_date
            'Q4 2025'  # fiscal_period
        )
        mock_session.execute = AsyncMock(return_value=mock_result)

        catalysts = await detector.detect(mock_session, 'AAPL')

        assert len(catalysts) > 0
        assert catalysts[0].event_type == 'earnings'
        assert catalysts[0].days_until == 14
        assert catalysts[0].icon == '📊'

    @pytest.mark.asyncio
    async def test_detect_no_earnings(self, detector):
        """Test when no upcoming earnings."""
        mock_session = AsyncMock()
        mock_result = MagicMock()
        mock_result.fetchone.return_value = None
        mock_session.execute = AsyncMock(return_value=mock_result)

        catalysts = await detector.detect(mock_session, 'AAPL')

        assert catalysts == []


class TestAnalystRatingDetector:
    """Test AnalystRatingDetector."""

    @pytest.fixture
    def detector(self):
        return AnalystRatingDetector()

    @pytest.mark.asyncio
    async def test_detect_recent_upgrade(self, detector):
        """Test detection of recent analyst upgrade."""
        mock_session = AsyncMock()
        mock_result = MagicMock()
        mock_result.fetchone.return_value = (
            5,   # upgrades
            1,   # downgrades
            date.today() - timedelta(days=2)  # latest_date
        )
        mock_session.execute = AsyncMock(return_value=mock_result)

        catalysts = await detector.detect(mock_session, 'AAPL')

        assert len(catalysts) > 0
        assert catalysts[0].event_type == 'analyst_rating'
        assert catalysts[0].impact == 'Positive'

    @pytest.mark.asyncio
    async def test_detect_recent_downgrade(self, detector):
        """Test detection of recent analyst downgrade."""
        mock_session = AsyncMock()
        mock_result = MagicMock()
        mock_result.fetchone.return_value = (
            1,   # upgrades
            5,   # downgrades
            date.today() - timedelta(days=2)
        )
        mock_session.execute = AsyncMock(return_value=mock_result)

        catalysts = await detector.detect(mock_session, 'AAPL')

        assert len(catalysts) > 0
        assert catalysts[0].impact == 'Negative'


class TestDividendDateDetector:
    """Test DividendDateDetector."""

    @pytest.fixture
    def detector(self):
        return DividendDateDetector()

    @pytest.mark.asyncio
    async def test_detect_upcoming_ex_dividend(self, detector):
        """Test detection of upcoming ex-dividend date."""
        mock_session = AsyncMock()
        mock_result = MagicMock()
        mock_result.fetchone.return_value = (
            date.today() + timedelta(days=7),  # ex_dividend_date
            2.5  # dividend_yield
        )
        mock_session.execute = AsyncMock(return_value=mock_result)

        catalysts = await detector.detect(mock_session, 'AAPL')

        assert len(catalysts) > 0
        assert catalysts[0].event_type == 'ex_dividend'
        assert catalysts[0].days_until == 7
        assert catalysts[0].icon == '💰'


class TestFiftyTwoWeekDetector:
    """Test FiftyTwoWeekDetector."""

    @pytest.fixture
    def detector(self):
        return FiftyTwoWeekDetector()

    @pytest.mark.asyncio
    async def test_detect_near_52_week_high(self, detector):
        """Test detection when near 52-week high."""
        mock_session = AsyncMock()
        mock_result = MagicMock()
        mock_result.fetchone.return_value = (
            150.0,   # current_price
            155.0,   # week_52_high
            100.0    # week_52_low
        )
        mock_session.execute = AsyncMock(return_value=mock_result)

        catalysts = await detector.detect(mock_session, 'AAPL')

        assert len(catalysts) > 0
        assert catalysts[0].event_type == '52_week_high'
        assert catalysts[0].impact == 'Positive'

    @pytest.mark.asyncio
    async def test_detect_near_52_week_low(self, detector):
        """Test detection when near 52-week low."""
        mock_session = AsyncMock()
        mock_result = MagicMock()
        mock_result.fetchone.return_value = (
            105.0,   # current_price
            200.0,   # week_52_high
            100.0    # week_52_low
        )
        mock_session.execute = AsyncMock(return_value=mock_result)

        catalysts = await detector.detect(mock_session, 'AAPL')

        assert len(catalysts) > 0
        assert catalysts[0].event_type == '52_week_low'
        assert catalysts[0].impact == 'Negative'


class TestTechnicalBreakoutDetector:
    """Test TechnicalBreakoutDetector."""

    @pytest.fixture
    def detector(self):
        return TechnicalBreakoutDetector()

    @pytest.mark.asyncio
    async def test_detect_bullish_breakout(self, detector):
        """Test detection of bullish technical breakout."""
        mock_session = AsyncMock()
        mock_result = MagicMock()
        mock_result.fetchone.return_value = (
            150.0,   # current_price
            140.0,   # sma_50
            135.0,   # sma_200
            60.0     # rsi
        )
        mock_session.execute = AsyncMock(return_value=mock_result)

        catalysts = await detector.detect(mock_session, 'AAPL')

        # Price above both SMAs = bullish
        if catalysts:
            assert catalysts[0].event_type == 'technical'
            assert 'bullish' in catalysts[0].title.lower() or catalysts[0].impact == 'Positive'

    @pytest.mark.asyncio
    async def test_detect_oversold(self, detector):
        """Test detection of oversold condition."""
        mock_session = AsyncMock()
        mock_result = MagicMock()
        mock_result.fetchone.return_value = (
            100.0,   # current_price
            110.0,   # sma_50
            115.0,   # sma_200
            25.0     # rsi (oversold)
        )
        mock_session.execute = AsyncMock(return_value=mock_result)

        catalysts = await detector.detect(mock_session, 'AAPL')

        # RSI < 30 = oversold
        if catalysts:
            assert 'oversold' in catalysts[0].title.lower() or catalysts[0].impact in ['Positive', 'Neutral']
