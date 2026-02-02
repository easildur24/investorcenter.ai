"""Tests for sector percentile calculator and aggregator.

These tests verify the sector-relative scoring functionality
introduced in IC Score v2.1.
"""
import pytest
from decimal import Decimal
from unittest.mock import AsyncMock, MagicMock, patch

import numpy as np

# Add parent directory to path for imports
import sys
from pathlib import Path
sys.path.insert(0, str(Path(__file__).parent.parent))

from pipelines.utils.sector_percentile import (
    SectorPercentileCalculator,
    SectorPercentileAggregator,
    SectorStats,
)


class TestSectorStats:
    """Tests for SectorStats dataclass."""

    def test_sector_stats_creation(self):
        """Test creating SectorStats with valid data."""
        stats = SectorStats(
            sector="Technology",
            metric="pe_ratio",
            min_value=5.0,
            p10=10.0,
            p25=15.0,
            p50=20.0,
            p75=30.0,
            p90=40.0,
            max_value=100.0,
            mean=25.0,
            std_dev=15.0,
            sample_count=150,
        )

        assert stats.sector == "Technology"
        assert stats.metric == "pe_ratio"
        assert stats.p50 == 20.0
        assert stats.sample_count == 150


class TestSectorPercentileCalculator:
    """Tests for SectorPercentileCalculator."""

    @pytest.fixture
    def mock_session(self):
        """Create a mock database session."""
        return AsyncMock()

    @pytest.fixture
    def calculator(self, mock_session):
        """Create calculator instance with mock session."""
        return SectorPercentileCalculator(mock_session)

    def test_lower_is_better_metrics(self, calculator):
        """Verify correct metrics are marked as lower-is-better."""
        assert 'pe_ratio' in calculator.LOWER_IS_BETTER
        assert 'ps_ratio' in calculator.LOWER_IS_BETTER
        assert 'pb_ratio' in calculator.LOWER_IS_BETTER
        assert 'debt_to_equity' in calculator.LOWER_IS_BETTER

        # These should NOT be lower-is-better
        assert 'roe' not in calculator.LOWER_IS_BETTER
        assert 'net_margin' not in calculator.LOWER_IS_BETTER

    def test_calculate_percentile_at_min(self, calculator):
        """Test percentile calculation at minimum value."""
        stats = SectorStats(
            sector="Tech", metric="pe_ratio",
            min_value=5.0, p10=10.0, p25=15.0, p50=20.0,
            p75=30.0, p90=40.0, max_value=100.0,
            mean=25.0, std_dev=15.0, sample_count=100
        )

        result = calculator._calculate_percentile(5.0, stats)
        assert result == 0.0

    def test_calculate_percentile_at_max(self, calculator):
        """Test percentile calculation at maximum value."""
        stats = SectorStats(
            sector="Tech", metric="pe_ratio",
            min_value=5.0, p10=10.0, p25=15.0, p50=20.0,
            p75=30.0, p90=40.0, max_value=100.0,
            mean=25.0, std_dev=15.0, sample_count=100
        )

        result = calculator._calculate_percentile(100.0, stats)
        assert result == 100.0

    def test_calculate_percentile_at_median(self, calculator):
        """Test percentile calculation at median value."""
        stats = SectorStats(
            sector="Tech", metric="pe_ratio",
            min_value=5.0, p10=10.0, p25=15.0, p50=20.0,
            p75=30.0, p90=40.0, max_value=100.0,
            mean=25.0, std_dev=15.0, sample_count=100
        )

        result = calculator._calculate_percentile(20.0, stats)
        assert result == 50.0

    def test_calculate_percentile_interpolation(self, calculator):
        """Test linear interpolation between percentile points."""
        stats = SectorStats(
            sector="Tech", metric="pe_ratio",
            min_value=0.0, p10=10.0, p25=25.0, p50=50.0,
            p75=75.0, p90=90.0, max_value=100.0,
            mean=50.0, std_dev=25.0, sample_count=100
        )

        # Value between p25 and p50
        result = calculator._calculate_percentile(37.5, stats)
        assert 25 < result < 50
        assert abs(result - 37.5) < 1  # Should be close to 37.5 with linear interpolation

    def test_calculate_percentile_below_min(self, calculator):
        """Test percentile calculation for value below minimum."""
        stats = SectorStats(
            sector="Tech", metric="pe_ratio",
            min_value=5.0, p10=10.0, p25=15.0, p50=20.0,
            p75=30.0, p90=40.0, max_value=100.0,
            mean=25.0, std_dev=15.0, sample_count=100
        )

        result = calculator._calculate_percentile(1.0, stats)
        assert result == 0.0

    def test_calculate_percentile_above_max(self, calculator):
        """Test percentile calculation for value above maximum."""
        stats = SectorStats(
            sector="Tech", metric="pe_ratio",
            min_value=5.0, p10=10.0, p25=15.0, p50=20.0,
            p75=30.0, p90=40.0, max_value=100.0,
            mean=25.0, std_dev=15.0, sample_count=100
        )

        result = calculator._calculate_percentile(150.0, stats)
        assert result == 100.0

    @pytest.mark.asyncio
    async def test_get_percentile_lower_is_better(self, calculator):
        """Test that lower-is-better metrics are inverted."""
        # Mock the sector stats retrieval
        mock_stats = SectorStats(
            sector="Technology", metric="pe_ratio",
            min_value=5.0, p10=10.0, p25=15.0, p50=20.0,
            p75=30.0, p90=40.0, max_value=100.0,
            mean=25.0, std_dev=15.0, sample_count=100
        )
        calculator._cache["Technology:pe_ratio"] = mock_stats

        # Low P/E (value at p10) should get HIGH score (inverted)
        result = await calculator.get_percentile("Technology", "pe_ratio", 10.0)
        assert result is not None
        assert result > 80  # Low P/E = high score

        # High P/E (value at p90) should get LOW score (inverted)
        result = await calculator.get_percentile("Technology", "pe_ratio", 40.0)
        assert result is not None
        assert result < 20  # High P/E = low score

    @pytest.mark.asyncio
    async def test_get_percentile_higher_is_better(self, calculator):
        """Test that higher-is-better metrics are NOT inverted."""
        mock_stats = SectorStats(
            sector="Technology", metric="roe",
            min_value=0.0, p10=5.0, p25=10.0, p50=15.0,
            p75=20.0, p90=30.0, max_value=50.0,
            mean=15.0, std_dev=10.0, sample_count=100
        )
        calculator._cache["Technology:roe"] = mock_stats

        # High ROE should get HIGH score (not inverted)
        result = await calculator.get_percentile("Technology", "roe", 30.0)
        assert result is not None
        assert result > 80

        # Low ROE should get LOW score (not inverted)
        result = await calculator.get_percentile("Technology", "roe", 5.0)
        assert result is not None
        assert result < 20

    @pytest.mark.asyncio
    async def test_get_percentile_none_value(self, calculator):
        """Test that None values return None."""
        result = await calculator.get_percentile("Technology", "pe_ratio", None)
        assert result is None

    def test_clear_cache(self, calculator):
        """Test cache clearing functionality."""
        calculator._cache["test_key"] = "test_value"
        assert len(calculator._cache) == 1

        calculator.clear_cache()
        assert len(calculator._cache) == 0


class TestSectorPercentileAggregator:
    """Tests for SectorPercentileAggregator."""

    @pytest.fixture
    def mock_session(self):
        """Create a mock database session."""
        return AsyncMock()

    @pytest.fixture
    def aggregator(self, mock_session):
        """Create aggregator instance with mock session."""
        return SectorPercentileAggregator(mock_session)

    def test_min_sample_size(self, aggregator):
        """Test minimum sample size constant."""
        assert aggregator.MIN_SAMPLE_SIZE == 5

    def test_tracked_metrics_not_empty(self):
        """Verify tracked metrics list is populated."""
        assert len(SectorPercentileCalculator.TRACKED_METRICS) > 10
        assert 'pe_ratio' in SectorPercentileCalculator.TRACKED_METRICS
        assert 'roe' in SectorPercentileCalculator.TRACKED_METRICS


class TestPercentileCalculations:
    """Integration tests for percentile calculation accuracy."""

    def test_percentile_distribution(self):
        """Test that percentile calculations match numpy."""
        # Create sample data
        data = np.array([10, 15, 20, 25, 30, 35, 40, 45, 50, 55])

        # Create stats matching numpy calculations
        stats = SectorStats(
            sector="Test", metric="test_metric",
            min_value=float(np.min(data)),
            p10=float(np.percentile(data, 10)),
            p25=float(np.percentile(data, 25)),
            p50=float(np.percentile(data, 50)),
            p75=float(np.percentile(data, 75)),
            p90=float(np.percentile(data, 90)),
            max_value=float(np.max(data)),
            mean=float(np.mean(data)),
            std_dev=float(np.std(data)),
            sample_count=len(data)
        )

        calculator = SectorPercentileCalculator(AsyncMock())

        # Median value should give approximately 50th percentile
        median_pct = calculator._calculate_percentile(stats.p50, stats)
        assert abs(median_pct - 50) < 1

        # 25th percentile value should give approximately 25th percentile
        p25_pct = calculator._calculate_percentile(stats.p25, stats)
        assert abs(p25_pct - 25) < 1

    def test_edge_case_identical_percentiles(self):
        """Test handling when percentile values are identical."""
        # Edge case: all values are the same
        stats = SectorStats(
            sector="Test", metric="test_metric",
            min_value=50.0, p10=50.0, p25=50.0, p50=50.0,
            p75=50.0, p90=50.0, max_value=50.0,
            mean=50.0, std_dev=0.0, sample_count=10
        )

        calculator = SectorPercentileCalculator(AsyncMock())

        # Should handle without division by zero
        result = calculator._calculate_percentile(50.0, stats)
        assert result is not None
        assert 0 <= result <= 100


if __name__ == "__main__":
    pytest.main([__file__, "-v"])
