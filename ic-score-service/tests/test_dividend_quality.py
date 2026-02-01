"""Unit tests for Dividend Quality factor calculator."""

import pytest
from unittest.mock import AsyncMock, MagicMock
from datetime import date

import sys
from pathlib import Path
sys.path.insert(0, str(Path(__file__).parent.parent))

from pipelines.utils.dividend_quality import (
    DividendQualityCalculator,
    DividendQualityResult
)


class TestDividendQualityCalculator:
    """Test cases for DividendQualityCalculator."""

    @pytest.fixture
    def mock_session(self):
        """Create a mock database session."""
        session = AsyncMock()
        return session

    @pytest.fixture
    def calculator(self, mock_session):
        """Create a calculator instance with mocked session."""
        return DividendQualityCalculator(mock_session)

    # ==================
    # Yield Score Tests
    # ==================

    def test_yield_score_with_sector_percentile(self, calculator):
        """Test yield score uses sector percentile when available."""
        score = calculator._calculate_yield_score(
            dividend_yield=3.0,
            sector_percentile=75.0
        )

        # Should use sector percentile directly
        assert score == 75.0

    def test_yield_score_without_sector_data(self, calculator):
        """Test yield score fallback to absolute scoring."""
        # 0% yield = 0, 2% = 50, 4%+ = 100
        score = calculator._calculate_yield_score(
            dividend_yield=2.0,
            sector_percentile=None
        )

        # 2% yield = 50 (2 * 25)
        assert score == 50.0

    def test_yield_score_high_yield(self, calculator):
        """Test yield score for high-yield stock."""
        score = calculator._calculate_yield_score(
            dividend_yield=5.0,
            sector_percentile=None
        )

        # 5% should cap at 100 (4%+ = 100)
        assert score == 100.0

    def test_yield_score_zero_yield(self, calculator):
        """Test yield score for zero yield."""
        score = calculator._calculate_yield_score(
            dividend_yield=0,
            sector_percentile=None
        )

        assert score == 0

    # ==================
    # Payout Score Tests
    # ==================

    def test_payout_score_optimal_range(self, calculator):
        """Test payout score in optimal range (30-60%)."""
        # Center of optimal range = 45%
        score = calculator._calculate_payout_score(45.0)

        # Should be near 100
        assert score > 90

    def test_payout_score_low_payout(self, calculator):
        """Test payout score for low payout ratio."""
        score = calculator._calculate_payout_score(15.0)

        # Low payout (room to grow but not sharing much)
        assert 40 < score < 70

    def test_payout_score_high_payout(self, calculator):
        """Test payout score for high payout ratio."""
        score = calculator._calculate_payout_score(75.0)

        # High payout (sustainable concern)
        assert 40 < score < 80

    def test_payout_score_very_high(self, calculator):
        """Test payout score for unsustainably high payout."""
        score = calculator._calculate_payout_score(120.0)

        # >100% payout is risky
        assert score < 30

    def test_payout_score_negative(self, calculator):
        """Test payout score for negative payout (negative earnings)."""
        score = calculator._calculate_payout_score(-50.0)

        # Negative earnings = risky
        assert score == 0

    def test_payout_score_missing(self, calculator):
        """Test payout score when data is missing."""
        score = calculator._calculate_payout_score(None)

        # Missing = neutral
        assert score == 50

    # ==================
    # Growth Score Tests
    # ==================

    def test_growth_score_strong_growth(self, calculator):
        """Test growth score with strong dividend growth."""
        # 10% CAGR = 100
        score = calculator._calculate_growth_score(
            dividend_cagr=10.0,
            dividend_growth_yoy=8.0
        )

        assert score == 100

    def test_growth_score_moderate_growth(self, calculator):
        """Test growth score with moderate dividend growth."""
        # 5% CAGR = 75
        score = calculator._calculate_growth_score(
            dividend_cagr=5.0,
            dividend_growth_yoy=4.0
        )

        assert score == 75

    def test_growth_score_no_growth(self, calculator):
        """Test growth score with no dividend growth."""
        score = calculator._calculate_growth_score(
            dividend_cagr=0,
            dividend_growth_yoy=0
        )

        # 0% growth = 50 (neutral)
        assert score == 50

    def test_growth_score_negative(self, calculator):
        """Test growth score with dividend cut."""
        score = calculator._calculate_growth_score(
            dividend_cagr=-10.0,
            dividend_growth_yoy=-15.0
        )

        # Negative growth = 0
        assert score == 0

    def test_growth_score_prefers_cagr(self, calculator):
        """Test growth score prefers CAGR over YoY."""
        score = calculator._calculate_growth_score(
            dividend_cagr=8.0,  # Should use this
            dividend_growth_yoy=2.0
        )

        # CAGR 8% = 90
        assert score == 90

    def test_growth_score_fallback_to_yoy(self, calculator):
        """Test growth score falls back to YoY when CAGR unavailable."""
        score = calculator._calculate_growth_score(
            dividend_cagr=None,
            dividend_growth_yoy=6.0
        )

        # YoY 6% = 80
        assert score == 80

    # ==================
    # Streak Score Tests
    # ==================

    def test_streak_score_dividend_king(self, calculator):
        """Test streak score for Dividend King (50+ years)."""
        score = calculator._calculate_streak_score(55)

        # 50+ years = 100
        assert score == 100

    def test_streak_score_dividend_aristocrat(self, calculator):
        """Test streak score for Dividend Aristocrat (25+ years)."""
        score = calculator._calculate_streak_score(30)

        # 25-49 years = 90-100
        assert 90 <= score <= 100

    def test_streak_score_dividend_achiever(self, calculator):
        """Test streak score for Dividend Achiever (10+ years)."""
        score = calculator._calculate_streak_score(15)

        # 10-24 years = 70-90
        assert 70 <= score <= 90

    def test_streak_score_good_track_record(self, calculator):
        """Test streak score for good track record (5+ years)."""
        score = calculator._calculate_streak_score(7)

        # 5-9 years = 50-70
        assert 50 <= score <= 70

    def test_streak_score_new_dividend_payer(self, calculator):
        """Test streak score for new dividend payer."""
        score = calculator._calculate_streak_score(2)

        # <5 years = years * 10
        assert score == 20

    def test_streak_score_zero(self, calculator):
        """Test streak score for no streak."""
        score = calculator._calculate_streak_score(0)

        assert score == 0

    # ==================
    # Overall Score Tests
    # ==================

    @pytest.mark.asyncio
    async def test_calculate_dividend_aristocrat(self, calculator):
        """Test full calculation for Dividend Aristocrat."""
        calculator.fetch_dividend_data = AsyncMock(return_value={
            'dividend_yield': 2.5,
            'payout_ratio': 45.0,
            'dividend_cagr_5y': 7.0,
            'dividend_growth_yoy': 6.0,
            'consecutive_years_paid': 35,
            'consecutive_years_increased': 30,
            'ex_dividend_date': date.today(),
            'payment_date': date.today(),
        })

        calculator.get_sector = AsyncMock(return_value='Consumer Staples')
        calculator.sector_calculator = None  # No sector-relative scoring

        result = await calculator.calculate('KO')

        assert result is not None
        assert isinstance(result, DividendQualityResult)
        assert result.is_dividend_payer is True
        # High quality dividend stock
        assert result.score > 70
        assert result.metrics['dividend_tier'] == 'Dividend Aristocrat'

    @pytest.mark.asyncio
    async def test_calculate_non_dividend_stock(self, calculator):
        """Test calculation for non-dividend paying stock."""
        calculator.fetch_dividend_data = AsyncMock(return_value={
            'dividend_yield': 0,
            'payout_ratio': None,
            'consecutive_years_increased': 0,
        })

        result = await calculator.calculate('AMZN')

        assert result is not None
        assert result.is_dividend_payer is False
        assert result.score == 0

    @pytest.mark.asyncio
    async def test_calculate_low_yield_excluded(self, calculator):
        """Test that stocks with yield < 0.5% are excluded."""
        calculator.fetch_dividend_data = AsyncMock(return_value={
            'dividend_yield': 0.3,  # Below MIN_DIVIDEND_YIELD
            'payout_ratio': 20.0,
            'consecutive_years_increased': 5,
        })

        result = await calculator.calculate('LOWDIV')

        assert result is not None
        assert result.is_dividend_payer is False

    @pytest.mark.asyncio
    async def test_calculate_missing_data(self, calculator):
        """Test calculation with missing dividend data."""
        calculator.fetch_dividend_data = AsyncMock(return_value=None)

        result = await calculator.calculate('UNKNOWN')

        assert result is not None
        assert result.is_dividend_payer is False
        assert result.score == 0


class TestDividendQualityEdgeCases:
    """Test edge cases and boundary conditions."""

    @pytest.fixture
    def calculator(self):
        return DividendQualityCalculator(AsyncMock())

    def test_payout_ratio_exactly_100(self, calculator):
        """Test payout ratio at exactly 100%."""
        score = calculator._calculate_payout_score(100.0)

        # 100% payout is risky but not terrible
        assert 20 < score < 50

    def test_streak_boundary_values(self, calculator):
        """Test streak score at boundary values."""
        # Test exact boundaries
        assert calculator._calculate_streak_score(5) >= 50
        assert calculator._calculate_streak_score(10) >= 70
        assert calculator._calculate_streak_score(25) >= 90
        assert calculator._calculate_streak_score(50) == 100

    @pytest.mark.asyncio
    async def test_score_bounds(self, calculator):
        """Test that all scores are bounded 0-100."""
        calculator.fetch_dividend_data = AsyncMock(return_value={
            'dividend_yield': 10.0,  # Very high
            'payout_ratio': 150.0,   # Unsustainable
            'dividend_cagr_5y': -20.0,  # Big cut
            'dividend_growth_yoy': -20.0,
            'consecutive_years_increased': 60,  # Dividend King
        })

        calculator.get_sector = AsyncMock(return_value='Utilities')

        result = await calculator.calculate('MIXED')

        assert result is not None
        assert 0 <= result.score <= 100
        assert 0 <= result.yield_score <= 100
        assert 0 <= result.payout_score <= 100
        assert 0 <= result.growth_score <= 100
        assert 0 <= result.streak_score <= 100

    @pytest.mark.asyncio
    async def test_with_sector_calculator(self, calculator):
        """Test calculation with sector-relative yield scoring."""
        mock_sector_calc = AsyncMock()
        mock_sector_calc.get_percentile = AsyncMock(return_value=85.0)

        calculator.sector_calculator = mock_sector_calc

        calculator.fetch_dividend_data = AsyncMock(return_value={
            'dividend_yield': 3.5,
            'payout_ratio': 50.0,
            'dividend_cagr_5y': 5.0,
            'consecutive_years_increased': 15,
        })

        calculator.get_sector = AsyncMock(return_value='Utilities')

        result = await calculator.calculate('DUK')

        assert result is not None
        # Yield score should use sector percentile (85)
        assert result.yield_score == 85.0
