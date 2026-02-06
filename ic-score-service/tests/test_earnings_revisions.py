"""Unit tests for Earnings Revisions factor calculator."""

import pytest
from unittest.mock import AsyncMock, MagicMock, patch
from dataclasses import dataclass

import sys
from pathlib import Path
sys.path.insert(0, str(Path(__file__).parent.parent))

from pipelines.utils.earnings_revisions import EarningsRevisionsCalculator, EarningsRevisionsResult


class TestEarningsRevisionsCalculator:
    """Test cases for EarningsRevisionsCalculator."""

    @pytest.fixture
    def mock_session(self):
        """Create a mock database session."""
        session = AsyncMock()
        return session

    @pytest.fixture
    def calculator(self, mock_session):
        """Create a calculator instance with mocked session."""
        return EarningsRevisionsCalculator(mock_session)

    # ==================
    # Magnitude Score Tests
    # ==================

    def test_magnitude_score_positive_revision(self, calculator):
        """Test magnitude score with positive EPS revision."""
        data = {
            'consensus_eps': 5.0,
            'estimate_90d_ago': 4.5,  # +11% revision
        }
        score = calculator._calculate_magnitude_score(data)

        # +11% should be ~87 (50 + (0.11/0.15)*50)
        assert 85 < score < 90

    def test_magnitude_score_negative_revision(self, calculator):
        """Test magnitude score with negative EPS revision."""
        data = {
            'consensus_eps': 4.0,
            'estimate_90d_ago': 5.0,  # -20% revision
        }
        score = calculator._calculate_magnitude_score(data)

        # -20% should be clamped to 0 (50 + (-0.20/0.15)*50 = -17)
        assert score == 0

    def test_magnitude_score_no_change(self, calculator):
        """Test magnitude score with no revision."""
        data = {
            'consensus_eps': 5.0,
            'estimate_90d_ago': 5.0,
        }
        score = calculator._calculate_magnitude_score(data)

        # 0% change = 50 (neutral)
        assert score == 50

    def test_magnitude_score_missing_data(self, calculator):
        """Test magnitude score with missing estimate data."""
        data = {
            'consensus_eps': 5.0,
            'estimate_90d_ago': None,
        }
        score = calculator._calculate_magnitude_score(data)

        # Missing data = 50 (neutral)
        assert score == 50

    def test_magnitude_score_max_positive(self, calculator):
        """Test magnitude score caps at 100."""
        data = {
            'consensus_eps': 6.0,
            'estimate_90d_ago': 4.0,  # +50% revision
        }
        score = calculator._calculate_magnitude_score(data)

        # +50% should be capped at 100
        assert score == 100

    # ==================
    # Breadth Score Tests
    # ==================

    def test_breadth_score_all_upgrades(self, calculator):
        """Test breadth score when all analysts upgrade."""
        data = {
            'upgrades_90d': 10,
            'downgrades_90d': 0,
        }
        score = calculator._calculate_breadth_score(data)

        # 100% upgrades = 100
        assert score == 100

    def test_breadth_score_all_downgrades(self, calculator):
        """Test breadth score when all analysts downgrade."""
        data = {
            'upgrades_90d': 0,
            'downgrades_90d': 10,
        }
        score = calculator._calculate_breadth_score(data)

        # 0% upgrades = 0
        assert score == 0

    def test_breadth_score_mixed_revisions(self, calculator):
        """Test breadth score with mixed revisions."""
        data = {
            'upgrades_90d': 7,
            'downgrades_90d': 3,
        }
        score = calculator._calculate_breadth_score(data)

        # 70% upgrades = 70
        assert score == 70

    def test_breadth_score_no_revisions(self, calculator):
        """Test breadth score with no revisions."""
        data = {
            'upgrades_90d': 0,
            'downgrades_90d': 0,
        }
        score = calculator._calculate_breadth_score(data)

        # No revisions = 50 (neutral)
        assert score == 50

    def test_breadth_score_fallback_to_60d(self, calculator):
        """Test breadth score falls back to 60d data."""
        data = {
            'upgrades_90d': 0,
            'downgrades_90d': 0,
            'upgrades_60d': 5,
            'downgrades_60d': 5,
        }
        score = calculator._calculate_breadth_score(data)

        # 50% upgrades from 60d data = 50
        assert score == 50

    # ==================
    # Recency Score Tests
    # ==================

    def test_recency_score_positive_acceleration(self, calculator):
        """Test recency score with positive acceleration."""
        data = {
            'revision_pct_30d': 0.05,  # +5% recent
            'revision_pct_90d': 0.02,  # +2% overall
        }
        score = calculator._calculate_recency_score(data)

        # +3% acceleration should be positive
        # 50 + (0.03 / 0.10) * 50 = 65
        assert 63 < score < 67

    def test_recency_score_negative_acceleration(self, calculator):
        """Test recency score with negative acceleration."""
        data = {
            'revision_pct_30d': -0.02,  # -2% recent
            'revision_pct_90d': 0.05,   # +5% overall
        }
        score = calculator._calculate_recency_score(data)

        # -7% acceleration should be negative
        assert score < 50

    def test_recency_score_no_data(self, calculator):
        """Test recency score with missing data."""
        data = {
            'revision_pct_30d': None,
            'revision_pct_90d': None,
        }
        score = calculator._calculate_recency_score(data)

        # Missing data = 50 (neutral)
        assert score == 50

    # ==================
    # Overall Score Tests
    # ==================

    @pytest.mark.asyncio
    async def test_calculate_full_score(self, calculator, mock_session):
        """Test full score calculation with all components."""
        # Mock fetch_eps_estimates
        calculator.fetch_eps_estimates = AsyncMock(return_value={
            'consensus_eps': 5.0,
            'num_analysts': 20,
            'high_estimate': 5.5,
            'low_estimate': 4.5,
            'estimate_90d_ago': 4.8,  # +4.2% revision
            'upgrades_90d': 15,
            'downgrades_90d': 5,  # 75% upgrade ratio
            'revision_pct_30d': 0.03,
            'revision_pct_90d': 0.042,
        })

        result = await calculator.calculate('AAPL')

        assert result is not None
        assert isinstance(result, EarningsRevisionsResult)
        assert 0 <= result.score <= 100
        assert 0 <= result.magnitude_score <= 100
        assert 0 <= result.breadth_score <= 100
        assert 0 <= result.recency_score <= 100
        assert result.metrics['consensus_eps'] == 5.0
        assert result.metrics['num_analysts'] == 20

    @pytest.mark.asyncio
    async def test_calculate_missing_eps_data(self, calculator, mock_session):
        """Test calculation returns None when EPS data is missing."""
        calculator.fetch_eps_estimates = AsyncMock(return_value=None)

        result = await calculator.calculate('AAPL')

        assert result is None

    @pytest.mark.asyncio
    async def test_calculate_estimate_spread(self, calculator, mock_session):
        """Test estimate spread calculation."""
        calculator.fetch_eps_estimates = AsyncMock(return_value={
            'consensus_eps': 5.0,
            'num_analysts': 20,
            'high_estimate': 6.0,
            'low_estimate': 4.0,  # Spread = 2.0 / 5.0 = 40%
            'estimate_90d_ago': 5.0,
            'upgrades_90d': 10,
            'downgrades_90d': 10,
            'revision_pct_30d': 0,
            'revision_pct_90d': 0,
        })

        result = await calculator.calculate('AAPL')

        assert result is not None
        assert result.metrics['estimate_spread'] == 40.0  # 40%


class TestEarningsRevisionsEdgeCases:
    """Test edge cases and boundary conditions."""

    @pytest.fixture
    def calculator(self):
        return EarningsRevisionsCalculator(AsyncMock())

    def test_negative_eps_handling(self, calculator):
        """Test handling of negative EPS values."""
        data = {
            'consensus_eps': -1.0,
            'estimate_90d_ago': -1.5,  # Improvement (less negative)
        }
        score = calculator._calculate_magnitude_score(data)

        # Improvement from -1.5 to -1.0 should be positive
        # Change = (-1.0 - (-1.5)) / |-1.5| = 0.5 / 1.5 = 33%
        assert score > 50

    def test_zero_eps_prior(self, calculator):
        """Test handling of zero prior EPS."""
        data = {
            'consensus_eps': 1.0,
            'estimate_90d_ago': 0,
        }
        score = calculator._calculate_magnitude_score(data)

        # Division by zero should return neutral
        assert score == 50

    def test_extreme_revision(self, calculator):
        """Test handling of extreme revision percentages."""
        data = {
            'consensus_eps': 10.0,
            'estimate_90d_ago': 1.0,  # +900% revision
        }
        score = calculator._calculate_magnitude_score(data)

        # Should be capped at 100
        assert score == 100
