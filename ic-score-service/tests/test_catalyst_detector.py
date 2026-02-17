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
    CatalystType,
    CatalystImpact,
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
            event_type=CatalystType.EARNINGS,
            title='Q4 2025 Earnings',
            description='Quarterly earnings report',
            event_date=date.today() + timedelta(days=10),
            icon='📊',
            impact=CatalystImpact.NEUTRAL,
            confidence=0.9,
            days_until=10,
            source='sec_filings',
        )

        assert catalyst.event_type == CatalystType.EARNINGS
        assert catalyst.title == 'Q4 2025 Earnings'
        assert catalyst.icon == '📊'
        assert catalyst.confidence == 0.9
        assert catalyst.days_until == 10
        assert catalyst.source == 'sec_filings'
        assert catalyst.description == 'Quarterly earnings report'

    def test_catalyst_with_none_date(self):
        """Test catalyst with unknown date."""
        catalyst = Catalyst(
            event_type=CatalystType.TECHNICAL_BREAKOUT,
            title='Approaching resistance',
            description=None,
            event_date=None,
            icon='📈',
            impact=CatalystImpact.BULLISH,
            confidence=0.7,
            days_until=0,
            source='technical_indicators',
        )

        assert catalyst.event_date is None
        assert catalyst.description is None


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
        # Mock all detectors to return None (no catalyst)
        for detector in service.DETECTORS:
            detector.detect = AsyncMock(return_value=None)

        result = await service.get_catalysts('AAPL', limit=5)

        assert isinstance(result, list)

    @pytest.mark.asyncio
    async def test_get_catalysts_limits_results(self, service):
        """Test get_catalysts respects limit."""
        # Create 10 mock catalysts via 10 detectors that each return one
        mock_detectors = []
        for i in range(10):
            det = MagicMock()
            det.detect = AsyncMock(return_value=Catalyst(
                event_type=CatalystType.EARNINGS,
                title=f'Test {i}',
                description=f'Desc {i}',
                event_date=date.today() + timedelta(days=i + 1),
                icon='🔹',
                impact=CatalystImpact.NEUTRAL,
                confidence=0.5,
                days_until=i + 1,
                source='test',
            ))
            mock_detectors.append(det)

        service.DETECTORS = mock_detectors

        result = await service.get_catalysts('AAPL', limit=5)

        assert len(result) <= 5

    @pytest.mark.asyncio
    async def test_get_catalysts_sorts_by_days_until(self, service):
        """Test get_catalysts sorts by days until event."""
        det1 = MagicMock()
        det1.detect = AsyncMock(return_value=Catalyst(
            event_type=CatalystType.EARNINGS,
            title='Event A',
            description='Far',
            event_date=date.today() + timedelta(days=30),
            icon='🔹',
            impact=CatalystImpact.NEUTRAL,
            confidence=0.5,
            days_until=30,
            source='test',
        ))
        det2 = MagicMock()
        det2.detect = AsyncMock(return_value=Catalyst(
            event_type=CatalystType.ANALYST_RATING,
            title='Event B',
            description='Close',
            event_date=date.today() + timedelta(days=5),
            icon='🔹',
            impact=CatalystImpact.NEUTRAL,
            confidence=0.5,
            days_until=5,
            source='test',
        ))
        det3 = MagicMock()
        det3.detect = AsyncMock(return_value=Catalyst(
            event_type=CatalystType.DIVIDEND_DATE,
            title='Event C',
            description='Mid',
            event_date=date.today() + timedelta(days=15),
            icon='🔹',
            impact=CatalystImpact.NEUTRAL,
            confidence=0.5,
            days_until=15,
            source='test',
        ))

        service.DETECTORS = [det1, det2, det3]

        result = await service.get_catalysts('AAPL', limit=5)

        # Should be sorted by days_until ascending
        if len(result) >= 2:
            for i in range(len(result) - 1):
                assert result[i].days_until <= result[i + 1].days_until


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
        # The query returns a single expected_date column
        mock_result.fetchone.return_value = (
            date.today() + timedelta(days=14),
        )
        mock_session.execute = AsyncMock(return_value=mock_result)

        catalyst = await detector.detect(mock_session, 'AAPL')

        # detect() returns Optional[Catalyst], not a list
        assert catalyst is not None
        assert catalyst.event_type == CatalystType.EARNINGS
        assert catalyst.days_until == 14
        assert catalyst.icon == '📊'

    @pytest.mark.asyncio
    async def test_detect_no_earnings(self, detector):
        """Test when no upcoming earnings."""
        mock_session = AsyncMock()
        mock_result = MagicMock()
        mock_result.fetchone.return_value = None
        mock_session.execute = AsyncMock(return_value=mock_result)

        catalyst = await detector.detect(mock_session, 'AAPL')

        assert catalyst is None


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
        # Query returns: rating, previous_rating, analyst_firm, rating_date
        mock_result.fetchone.return_value = (
            'Buy',
            'Hold',
            'Goldman Sachs',
            date.today() - timedelta(days=2),
        )
        mock_session.execute = AsyncMock(return_value=mock_result)

        catalyst = await detector.detect(mock_session, 'AAPL')

        assert catalyst is not None
        assert catalyst.event_type == CatalystType.ANALYST_RATING
        assert catalyst.impact == CatalystImpact.BULLISH

    @pytest.mark.asyncio
    async def test_detect_recent_downgrade(self, detector):
        """Test detection of recent analyst downgrade."""
        mock_session = AsyncMock()
        mock_result = MagicMock()
        mock_result.fetchone.return_value = (
            'Sell',
            'Buy',
            'Morgan Stanley',
            date.today() - timedelta(days=2),
        )
        mock_session.execute = AsyncMock(return_value=mock_result)

        catalyst = await detector.detect(mock_session, 'AAPL')

        assert catalyst is not None
        assert catalyst.impact == CatalystImpact.BEARISH


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
        # Query returns: ex_dividend_date, annual_dividend, dividend_yield
        mock_result.fetchone.return_value = (
            date.today() + timedelta(days=7),
            3.28,
            2.5,
        )
        mock_session.execute = AsyncMock(return_value=mock_result)

        catalyst = await detector.detect(mock_session, 'AAPL')

        assert catalyst is not None
        assert catalyst.event_type == CatalystType.DIVIDEND_DATE
        assert catalyst.days_until == 7
        assert catalyst.icon == '💵'


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
        # Query returns: current_price, week_52_high, week_52_low
        mock_result.fetchone.return_value = (
            150.0,
            155.0,  # Within 5% threshold
            100.0,
        )
        mock_session.execute = AsyncMock(return_value=mock_result)

        catalyst = await detector.detect(mock_session, 'AAPL')

        assert catalyst is not None
        assert catalyst.event_type == CatalystType.FIFTY_TWO_WEEK
        assert catalyst.impact == CatalystImpact.BULLISH

    @pytest.mark.asyncio
    async def test_detect_near_52_week_low(self, detector):
        """Test detection when near 52-week low."""
        mock_session = AsyncMock()
        mock_result = MagicMock()
        mock_result.fetchone.return_value = (
            105.0,
            200.0,
            100.0,  # Within 5% threshold
        )
        mock_session.execute = AsyncMock(return_value=mock_result)

        catalyst = await detector.detect(mock_session, 'AAPL')

        assert catalyst is not None
        assert catalyst.event_type == CatalystType.FIFTY_TWO_WEEK
        assert catalyst.impact == CatalystImpact.BEARISH


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
        # Query returns: current_price, sma_50, sma_200, rsi
        mock_result.fetchone.return_value = (
            150.0,  # price
            140.0,  # sma_50 (price > sma_50)
            135.0,  # sma_200 (sma_50 > sma_200 = golden cross)
            60.0,   # rsi
        )
        mock_session.execute = AsyncMock(return_value=mock_result)

        catalyst = await detector.detect(mock_session, 'AAPL')

        # Price above both SMAs with golden cross = bullish
        assert catalyst is not None
        assert catalyst.event_type == CatalystType.TECHNICAL_BREAKOUT
        assert catalyst.impact == CatalystImpact.BULLISH

    @pytest.mark.asyncio
    async def test_detect_oversold(self, detector):
        """Test detection of oversold condition."""
        mock_session = AsyncMock()
        mock_result = MagicMock()
        mock_result.fetchone.return_value = (
            100.0,   # current_price
            110.0,   # sma_50 (price < sma_50)
            105.0,   # sma_200 (sma_50 > sma_200, no death cross)
            25.0,    # rsi < 30 = oversold
        )
        mock_session.execute = AsyncMock(return_value=mock_result)

        catalyst = await detector.detect(mock_session, 'AAPL')

        # RSI < 30 = oversold signal
        assert catalyst is not None
        assert 'RSI' in catalyst.title or 'Oversold' in catalyst.title
        assert catalyst.impact == CatalystImpact.BULLISH  # Oversold = bounce potential
