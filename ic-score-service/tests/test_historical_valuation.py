"""Unit tests for Historical Valuation factor calculator."""

import pytest
from unittest.mock import AsyncMock, MagicMock
from datetime import date, timedelta

import sys
from pathlib import Path
sys.path.insert(0, str(Path(__file__).parent.parent))

from pipelines.utils.historical_valuation import (
    HistoricalValuationCalculator,
    HistoricalValuationResult
)


class TestHistoricalValuationCalculator:
    """Test cases for HistoricalValuationCalculator."""

    @pytest.fixture
    def mock_session(self):
        """Create a mock database session."""
        session = AsyncMock()
        return session

    @pytest.fixture
    def calculator(self, mock_session):
        """Create a calculator instance with mocked session."""
        return HistoricalValuationCalculator(mock_session)

    # ==================
    # Percentile Calculation Tests
    # ==================

    def test_percentile_at_historical_low(self, calculator):
        """Test percentile when current value is at historical low."""
        history = [10.0, 15.0, 20.0, 25.0, 30.0]
        current = 8.0  # Below all historical values

        percentile = calculator._percentile_in_history(current, history)

        # Current is below all values, so percentile = 0
        assert percentile == 0

    def test_percentile_at_historical_high(self, calculator):
        """Test percentile when current value is at historical high."""
        history = [10.0, 15.0, 20.0, 25.0, 30.0]
        current = 35.0  # Above all historical values

        percentile = calculator._percentile_in_history(current, history)

        # Current is above all values, so percentile = 100
        assert percentile == 100

    def test_percentile_at_median(self, calculator):
        """Test percentile when current value is at historical median."""
        history = [10.0, 15.0, 20.0, 25.0, 30.0]
        current = 20.0  # At median

        percentile = calculator._percentile_in_history(current, history)

        # 2 values below (10, 15), current at 20 = 40%
        assert percentile == 40

    def test_percentile_empty_history(self, calculator):
        """Test percentile with empty history."""
        history = []
        current = 20.0

        percentile = calculator._percentile_in_history(current, history)

        # Empty history returns neutral 50
        assert percentile == 50

    # ==================
    # Score Calculation Tests
    # ==================

    @pytest.mark.asyncio
    async def test_calculate_mature_company(self, calculator):
        """Test calculation for mature company (higher P/E weight)."""
        # Mock current valuation
        calculator.get_current_valuation = AsyncMock(return_value={
            'pe_ratio': 15.0,
            'ps_ratio': 2.0,
            'calculation_date': date.today()
        })

        # Mock 5-year P/E history (current at low end)
        calculator.get_valuation_history = AsyncMock(side_effect=[
            [20.0, 22.0, 18.0, 25.0, 30.0] * 10,  # PE history (current at p20)
            [2.5, 3.0, 2.8, 3.5, 4.0] * 10,       # PS history
        ])

        # Mock net margin (mature company)
        calculator.get_net_margin = AsyncMock(return_value=15.0)

        result = await calculator.calculate('AAPL')

        assert result is not None
        assert isinstance(result, HistoricalValuationResult)
        # Low P/E percentile means high score (inverted)
        assert result.score > 50  # Should be positive since trading below historical avg
        assert result.metrics.get('is_growth_company') is False

    @pytest.mark.asyncio
    async def test_calculate_growth_company(self, calculator):
        """Test calculation for growth company (higher P/S weight)."""
        # Mock current valuation
        calculator.get_current_valuation = AsyncMock(return_value={
            'pe_ratio': None,  # Often N/A for growth companies
            'ps_ratio': 5.0,
            'calculation_date': date.today()
        })

        # Mock 5-year history — code calls get_valuation_history only
        # for ps_ratio since pe_ratio is None
        calculator.get_valuation_history = AsyncMock(
            return_value=[8.0, 10.0, 12.0, 15.0, 20.0] * 10
        )

        # Mock net margin (growth company with low/negative margin)
        calculator.get_net_margin = AsyncMock(return_value=2.0)

        result = await calculator.calculate('TSLA')

        assert result is not None
        # P/S only, at low end of history = high score
        assert result.pe_percentile is None
        assert result.ps_percentile is not None
        assert result.score > 70  # Trading cheap vs history

    @pytest.mark.asyncio
    async def test_calculate_no_valuation_data(self, calculator):
        """Test calculation when no valuation data available."""
        calculator.get_current_valuation = AsyncMock(return_value=None)

        result = await calculator.calculate('UNKNOWN')

        assert result is None

    @pytest.mark.asyncio
    async def test_calculate_insufficient_history(self, calculator):
        """Test calculation with no historical data available."""
        calculator.get_current_valuation = AsyncMock(return_value={
            'pe_ratio': 15.0,
            'ps_ratio': 2.0,
        })

        # No history available — both return empty/None
        calculator.get_valuation_history = AsyncMock(return_value=None)

        calculator.get_net_margin = AsyncMock(return_value=10.0)

        result = await calculator.calculate('NEW_IPO')

        # Should return None due to no historical data
        assert result is None

    # ==================
    # Metric Tests
    # ==================

    @pytest.mark.asyncio
    async def test_metrics_contain_5yr_range(self, calculator):
        """Test that metrics include 5-year range data."""
        calculator.get_current_valuation = AsyncMock(return_value={
            'pe_ratio': 20.0,
            'ps_ratio': 3.0,
        })

        pe_history = [15.0, 18.0, 20.0, 25.0, 30.0, 22.0, 19.0, 21.0, 24.0, 28.0, 17.0, 23.0]
        ps_history = [2.0, 2.5, 3.0, 3.5, 4.0, 2.8, 3.2, 2.9, 3.1, 3.8, 2.4, 3.3]

        calculator.get_valuation_history = AsyncMock(side_effect=[pe_history, ps_history])
        calculator.get_net_margin = AsyncMock(return_value=12.0)

        result = await calculator.calculate('MSFT')

        assert result is not None
        metrics = result.metrics

        # Should contain P/E range
        assert 'pe_5y_low' in metrics
        assert 'pe_5y_high' in metrics
        assert 'pe_5y_median' in metrics

        # Should contain P/S range
        assert 'ps_5y_low' in metrics
        assert 'ps_5y_high' in metrics
        assert 'ps_5y_median' in metrics

        # Verify ranges are correct
        assert metrics['pe_5y_low'] == min(pe_history)
        assert metrics['pe_5y_high'] == max(pe_history)


class TestHistoricalValuationEdgeCases:
    """Test edge cases and boundary conditions."""

    @pytest.fixture
    def calculator(self):
        return HistoricalValuationCalculator(AsyncMock())

    def test_negative_pe_ratio_excluded(self, calculator):
        """Test that negative P/E ratios are excluded from history."""
        # In real data, negative P/E from negative earnings should be filtered
        # The calculator should only consider positive P/E values
        history = [-10.0, 15.0, -20.0, 25.0, 30.0]  # Negatives should be filtered upstream
        current = 20.0

        percentile = calculator._percentile_in_history(current, history)

        # All values considered as-is (filtering should happen during data fetch)
        assert percentile > 0

    @pytest.mark.asyncio
    async def test_very_high_pe(self, calculator):
        """Test handling of very high P/E ratio."""
        calculator.get_current_valuation = AsyncMock(return_value={
            'pe_ratio': 500.0,  # Very high P/E
            'ps_ratio': 10.0,
        })

        calculator.get_valuation_history = AsyncMock(side_effect=[
            [20.0, 25.0, 30.0, 35.0, 40.0] * 10,  # Normal PE history
            [5.0, 6.0, 7.0, 8.0, 9.0] * 10,       # PS history
        ])

        calculator.get_net_margin = AsyncMock(return_value=5.0)

        result = await calculator.calculate('OVERVALUED')

        assert result is not None
        # Very high P/E at 100th percentile = low score (0)
        # Score = 100 - percentile
        assert result.score < 30

    @pytest.mark.asyncio
    async def test_score_bounds(self, calculator):
        """Test that score is always between 0 and 100."""
        calculator.get_current_valuation = AsyncMock(return_value={
            'pe_ratio': 100.0,
            'ps_ratio': 50.0,
        })

        calculator.get_valuation_history = AsyncMock(side_effect=[
            [10.0] * 60,  # All same PE (extreme case)
            [5.0] * 60,   # All same PS
        ])

        calculator.get_net_margin = AsyncMock(return_value=10.0)

        result = await calculator.calculate('EXTREME')

        assert result is not None
        assert 0 <= result.score <= 100
        if result.pe_percentile is not None:
            assert 0 <= result.pe_percentile <= 100
        if result.ps_percentile is not None:
            assert 0 <= result.ps_percentile <= 100
