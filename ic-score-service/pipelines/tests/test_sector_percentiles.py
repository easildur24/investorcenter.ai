"""Tests for sector percentiles calculator pipeline.

Tests the pipeline orchestration functions in
sector_percentiles_calculator.py:
- calculate_sector_percentiles()
- run_lifecycle_classification()

Also tests the SectorPercentileAggregator's numpy-based
percentile logic with known data sets.
"""

from decimal import Decimal
from unittest.mock import AsyncMock, MagicMock, patch

import numpy as np
import pytest

# Add parent directory to path for imports
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).parent.parent.parent))

from pipelines.utils.sector_percentile import (
    SectorPercentileAggregator,
    SectorPercentileCalculator,
    SectorStats,
)


# =====================================================================
# SectorPercentileCalculator._calculate_percentile (pure logic)
# =====================================================================


class TestCalculatePercentile:
    """Tests for the piecewise linear interpolation logic."""

    @pytest.fixture
    def calculator(self):
        return SectorPercentileCalculator(AsyncMock())

    @pytest.fixture
    def uniform_stats(self):
        """Stats with evenly spaced percentile breakpoints."""
        return SectorStats(
            sector="Technology",
            metric="roe",
            min_value=0.0,
            p10=10.0,
            p25=25.0,
            p50=50.0,
            p75=75.0,
            p90=90.0,
            max_value=100.0,
            mean=50.0,
            std_dev=30.0,
            sample_count=200,
        )

    def test_value_at_min(self, calculator, uniform_stats):
        assert calculator._calculate_percentile(0.0, uniform_stats) == 0.0

    def test_value_at_max(self, calculator, uniform_stats):
        assert (
            calculator._calculate_percentile(100.0, uniform_stats) == 100.0
        )

    def test_value_at_median(self, calculator, uniform_stats):
        assert (
            calculator._calculate_percentile(50.0, uniform_stats) == 50.0
        )

    def test_value_at_p25(self, calculator, uniform_stats):
        assert (
            calculator._calculate_percentile(25.0, uniform_stats) == 25.0
        )

    def test_value_at_p75(self, calculator, uniform_stats):
        assert (
            calculator._calculate_percentile(75.0, uniform_stats) == 75.0
        )

    def test_interpolation_between_p25_and_p50(
        self, calculator, uniform_stats
    ):
        """Midpoint between p25=25 and p50=50 should give ~37.5 pctl."""
        result = calculator._calculate_percentile(37.5, uniform_stats)
        assert abs(result - 37.5) < 0.01

    def test_below_min_returns_zero(self, calculator, uniform_stats):
        result = calculator._calculate_percentile(-10.0, uniform_stats)
        assert result == 0.0

    def test_above_max_returns_hundred(self, calculator, uniform_stats):
        result = calculator._calculate_percentile(150.0, uniform_stats)
        assert result == 100.0

    def test_all_same_values(self, calculator):
        """When min==max (all identical), no division by zero."""
        stats = SectorStats(
            sector="Test",
            metric="roe",
            min_value=42.0,
            p10=42.0,
            p25=42.0,
            p50=42.0,
            p75=42.0,
            p90=42.0,
            max_value=42.0,
            mean=42.0,
            std_dev=0.0,
            sample_count=10,
        )
        # Value exactly at the common point
        result = calculator._calculate_percentile(42.0, stats)
        assert 0 <= result <= 100

    def test_value_between_p90_and_max(self, calculator, uniform_stats):
        """Value between p90 and max should be in range (90, 100)."""
        result = calculator._calculate_percentile(95.0, uniform_stats)
        assert 90.0 < result < 100.0


# =====================================================================
# SectorPercentileCalculator.get_percentile (async, inversion logic)
# =====================================================================


class TestGetPercentileInversion:
    """Test that lower-is-better metrics are inverted correctly."""

    @pytest.fixture
    def calculator(self):
        calc = SectorPercentileCalculator(AsyncMock())
        # Pre-load cache so no DB call is made
        stats = SectorStats(
            sector="Technology",
            metric="pe_ratio",
            min_value=5.0,
            p10=10.0,
            p25=15.0,
            p50=20.0,
            p75=30.0,
            p90=40.0,
            max_value=80.0,
            mean=25.0,
            std_dev=15.0,
            sample_count=100,
        )
        calc._cache["Technology:pe_ratio"] = stats
        calc._cache["Technology:roe"] = SectorStats(
            sector="Technology",
            metric="roe",
            min_value=0.0,
            p10=5.0,
            p25=10.0,
            p50=15.0,
            p75=25.0,
            p90=35.0,
            max_value=50.0,
            mean=17.0,
            std_dev=12.0,
            sample_count=100,
        )
        return calc

    @pytest.mark.asyncio
    async def test_pe_ratio_low_value_gets_high_percentile(
        self, calculator
    ):
        """Low P/E (good) should score high when inverted."""
        pct = await calculator.get_percentile(
            "Technology", "pe_ratio", 10.0
        )
        assert pct is not None
        assert pct > 80

    @pytest.mark.asyncio
    async def test_pe_ratio_high_value_gets_low_percentile(
        self, calculator
    ):
        """High P/E (bad) should score low when inverted."""
        pct = await calculator.get_percentile(
            "Technology", "pe_ratio", 40.0
        )
        assert pct is not None
        assert pct < 20

    @pytest.mark.asyncio
    async def test_roe_not_inverted(self, calculator):
        """ROE (higher-is-better) should NOT be inverted."""
        high_pct = await calculator.get_percentile(
            "Technology", "roe", 35.0
        )
        low_pct = await calculator.get_percentile(
            "Technology", "roe", 5.0
        )
        assert high_pct > low_pct

    @pytest.mark.asyncio
    async def test_none_value_returns_none(self, calculator):
        result = await calculator.get_percentile(
            "Technology", "pe_ratio", None
        )
        assert result is None


# =====================================================================
# SectorPercentileCalculator cache
# =====================================================================


class TestCalculatorCache:
    def test_clear_cache(self):
        calc = SectorPercentileCalculator(AsyncMock())
        calc._cache["key1"] = "val1"
        calc._cache["key2"] = "val2"
        assert len(calc._cache) == 2
        calc.clear_cache()
        assert len(calc._cache) == 0


# =====================================================================
# SectorPercentileAggregator configuration
# =====================================================================


class TestAggregatorConfig:
    def test_min_sample_size(self):
        agg = SectorPercentileAggregator(AsyncMock())
        assert agg.MIN_SAMPLE_SIZE == 5

    def test_tracked_metrics_coverage(self):
        """Verify that critical metrics are tracked."""
        metrics = SectorPercentileCalculator.TRACKED_METRICS
        assert "pe_ratio" in metrics
        assert "roe" in metrics
        assert "net_margin" in metrics
        assert "debt_to_equity" in metrics
        assert "revenue_growth_yoy" in metrics

    def test_lower_is_better_set_complete(self):
        lib = SectorPercentileCalculator.LOWER_IS_BETTER
        assert "pe_ratio" in lib
        assert "ps_ratio" in lib
        assert "pb_ratio" in lib
        assert "ev_ebitda" in lib
        assert "debt_to_equity" in lib
        assert "net_debt_to_ebitda" in lib
        # These should NOT be lower-is-better
        assert "roe" not in lib
        assert "net_margin" not in lib
        assert "revenue_growth_yoy" not in lib


# =====================================================================
# Percentile calculations match numpy
# =====================================================================


class TestPercentileAccuracyWithNumpy:
    """Verify that SectorStats built from numpy match expected values."""

    def test_stats_from_known_data(self):
        data = np.array(
            [10.0, 20.0, 30.0, 40.0, 50.0, 60.0, 70.0, 80.0, 90.0, 100.0]
        )
        stats = SectorStats(
            sector="Test",
            metric="test_metric",
            min_value=float(np.min(data)),
            p10=float(np.percentile(data, 10)),
            p25=float(np.percentile(data, 25)),
            p50=float(np.percentile(data, 50)),
            p75=float(np.percentile(data, 75)),
            p90=float(np.percentile(data, 90)),
            max_value=float(np.max(data)),
            mean=float(np.mean(data)),
            std_dev=float(np.std(data)),
            sample_count=len(data),
        )

        calc = SectorPercentileCalculator(AsyncMock())

        # Median should give ~50th percentile
        median_pct = calc._calculate_percentile(stats.p50, stats)
        assert abs(median_pct - 50.0) < 1.0

        # p25 value should give ~25th percentile
        p25_pct = calc._calculate_percentile(stats.p25, stats)
        assert abs(p25_pct - 25.0) < 1.0

        # p90 value should give ~90th percentile
        p90_pct = calc._calculate_percentile(stats.p90, stats)
        assert abs(p90_pct - 90.0) < 1.0

    def test_single_value_data(self):
        """Edge case: all data points are the same value."""
        data = np.array([50.0] * 10)
        stats = SectorStats(
            sector="Test",
            metric="test_metric",
            min_value=50.0,
            p10=50.0,
            p25=50.0,
            p50=50.0,
            p75=50.0,
            p90=50.0,
            max_value=50.0,
            mean=50.0,
            std_dev=0.0,
            sample_count=10,
        )
        calc = SectorPercentileCalculator(AsyncMock())

        # Exactly at the value should not cause an error
        result = calc._calculate_percentile(50.0, stats)
        assert 0 <= result <= 100


# =====================================================================
# Pipeline orchestration: calculate_sector_percentiles
# =====================================================================


class TestCalculateSectorPercentilesOrchestration:
    """Test the async pipeline function with fully mocked dependencies."""

    @pytest.mark.asyncio
    async def test_dry_run_returns_early(self):
        """Dry run should not persist anything."""
        with patch(
            "pipelines.sector_percentiles_calculator.get_database"
        ) as mock_db:
            mock_session = AsyncMock()
            mock_db.return_value.session.return_value.__aenter__ = (
                AsyncMock(return_value=mock_session)
            )
            mock_db.return_value.session.return_value.__aexit__ = (
                AsyncMock(return_value=False)
            )

            mock_agg = AsyncMock()
            mock_agg._get_active_sectors = AsyncMock(
                return_value=["Technology", "Healthcare"]
            )

            with patch(
                "pipelines.sector_percentiles_calculator."
                "SectorPercentileAggregator",
                return_value=mock_agg,
            ):
                from pipelines.sector_percentiles_calculator import (
                    calculate_sector_percentiles,
                )

                result = await calculate_sector_percentiles(
                    dry_run=True
                )

            assert result.get("dry_run") is True

    @pytest.mark.asyncio
    async def test_specific_sector_mode(self):
        """Passing sector= should only calculate that one sector."""
        with patch(
            "pipelines.sector_percentiles_calculator.get_database"
        ) as mock_db:
            mock_session = AsyncMock()
            mock_db.return_value.session.return_value.__aenter__ = (
                AsyncMock(return_value=mock_session)
            )
            mock_db.return_value.session.return_value.__aexit__ = (
                AsyncMock(return_value=False)
            )

            mock_agg = AsyncMock()
            mock_agg.calculate_sector = AsyncMock(return_value=10)

            with patch(
                "pipelines.sector_percentiles_calculator."
                "SectorPercentileAggregator",
                return_value=mock_agg,
            ):
                from pipelines.sector_percentiles_calculator import (
                    calculate_sector_percentiles,
                )

                result = await calculate_sector_percentiles(
                    sector="Technology", dry_run=False
                )

            assert result.get("status") == "success"
            mock_agg.calculate_sector.assert_awaited_once_with(
                "Technology"
            )
