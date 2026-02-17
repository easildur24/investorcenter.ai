"""Tests for pipeline dependency checker.

Validates the dependency graph structure, execution ordering,
upstream dependency resolution, and freshness checks against
mock database sessions.
"""

from datetime import datetime, timedelta, timezone
from unittest.mock import AsyncMock, MagicMock

import pytest

from pipelines.utils.dependency_checker import (
    PIPELINE_DEPENDENCIES,
    PIPELINE_OUTPUT_TABLE,
    check_upstream_freshness,
    get_all_upstream_dependencies,
    get_execution_order,
    get_upstream_dependencies,
    has_cycle,
)


# =====================================================================
# TestDependencyGraph
# =====================================================================


class TestDependencyGraph:
    def test_no_cycles(self):
        """The dependency graph must be a DAG (no cycles)."""
        assert has_cycle() is False

    def test_all_pipelines_present(self):
        """All 16 pipelines must be present in the graph."""
        expected_pipelines = {
            "benchmark_data",
            "treasury_rates",
            "sec_financials",
            "sec_13f_holdings",
            "analyst_ratings",
            "insider_trades",
            "daily_price_update",
            "news_sentiment",
            "reddit_sentiment",
            "ttm_financials",
            "fundamental_metrics",
            "sector_percentiles",
            "fair_value",
            "technical_indicators",
            "valuation_ratios",
            "risk_metrics",
            "ic_score_calculator",
        }
        actual_pipelines = set(PIPELINE_DEPENDENCIES.keys())
        assert actual_pipelines == expected_pipelines

    def test_ingestion_pipelines_no_deps(self):
        """Ingestion pipelines have no upstream dependencies."""
        ingestion_pipelines = [
            "benchmark_data",
            "treasury_rates",
            "sec_financials",
            "sec_13f_holdings",
            "analyst_ratings",
            "insider_trades",
            "daily_price_update",
            "news_sentiment",
            "reddit_sentiment",
        ]
        for pipeline in ingestion_pipelines:
            assert PIPELINE_DEPENDENCIES[pipeline] == [], (
                f"{pipeline} should have no dependencies"
            )

    def test_ic_score_depends_on_all_calculators(self):
        """ic_score_calculator depends on all calculator pipelines."""
        ic_deps = set(
            PIPELINE_DEPENDENCIES["ic_score_calculator"]
        )
        expected_deps = {
            "fundamental_metrics",
            "sector_percentiles",
            "fair_value",
            "technical_indicators",
            "valuation_ratios",
            "risk_metrics",
        }
        assert ic_deps == expected_deps

    def test_all_deps_reference_valid_pipelines(self):
        """Every dependency listed must exist as a pipeline."""
        all_pipelines = set(PIPELINE_DEPENDENCIES.keys())
        for pipeline, deps in PIPELINE_DEPENDENCIES.items():
            for dep in deps:
                assert dep in all_pipelines, (
                    f"{pipeline} depends on '{dep}' which is "
                    f"not in PIPELINE_DEPENDENCIES"
                )


# =====================================================================
# TestGetExecutionOrder
# =====================================================================


class TestGetExecutionOrder:
    def test_returns_multiple_tiers(self):
        """Execution order has at least 3 tiers."""
        tiers = get_execution_order()
        assert len(tiers) >= 3

    def test_tier_0_has_no_deps(self):
        """First tier contains only pipelines with no deps."""
        tiers = get_execution_order()
        tier_0 = tiers[0]
        for pipeline in tier_0:
            assert PIPELINE_DEPENDENCIES[pipeline] == [], (
                f"{pipeline} in tier 0 but has dependencies: "
                f"{PIPELINE_DEPENDENCIES[pipeline]}"
            )

    def test_ic_score_in_last_tier(self):
        """ic_score_calculator must be in the final tier."""
        tiers = get_execution_order()
        last_tier = tiers[-1]
        assert "ic_score_calculator" in last_tier


# =====================================================================
# TestGetUpstreamDependencies
# =====================================================================


class TestGetUpstreamDependencies:
    def test_direct_deps(self):
        """fundamental_metrics depends directly on ttm_financials."""
        deps = get_upstream_dependencies("fundamental_metrics")
        assert deps == ["ttm_financials"]

    def test_transitive_deps(self):
        """ic_score_calculator transitively depends on many
        ingestion pipelines."""
        all_deps = get_all_upstream_dependencies(
            "ic_score_calculator"
        )
        expected_subset = {
            "sec_financials",
            "daily_price_update",
            "benchmark_data",
            "ttm_financials",
        }
        assert expected_subset.issubset(set(all_deps))

    def test_unknown_pipeline_raises(self):
        """Unknown pipeline name raises ValueError."""
        with pytest.raises(ValueError, match="Unknown pipeline"):
            get_upstream_dependencies("nonexistent")


# =====================================================================
# TestCheckUpstreamFreshness
# =====================================================================


class TestCheckUpstreamFreshness:
    @staticmethod
    def _make_mock_session(latest_timestamp):
        """Create a mock AsyncSession that returns a given
        timestamp from MAX() queries."""
        mock_row = MagicMock()
        mock_row.__getitem__ = lambda self, idx: latest_timestamp

        mock_result = MagicMock()
        mock_result.fetchone.return_value = mock_row

        mock_session = AsyncMock()
        mock_session.execute.return_value = mock_result
        return mock_session

    @pytest.mark.asyncio
    async def test_no_deps_always_ready(self):
        """Pipelines with no dependencies are always ready."""
        mock_session = AsyncMock()
        result = await check_upstream_freshness(
            "benchmark_data", mock_session
        )
        assert result.is_ready is True
        assert result.stale_dependencies == []

    @pytest.mark.asyncio
    async def test_fresh_data_is_ready(self):
        """Recent upstream data means the pipeline is ready."""
        recent_ts = datetime.now(timezone.utc).replace(
            tzinfo=None
        ) - timedelta(hours=1)
        mock_session = self._make_mock_session(recent_ts)

        result = await check_upstream_freshness(
            "fundamental_metrics", mock_session
        )
        assert result.is_ready is True
        assert result.stale_dependencies == []

    @pytest.mark.asyncio
    async def test_stale_data_not_ready(self):
        """Old upstream data means the pipeline is not ready."""
        old_ts = datetime.now(timezone.utc).replace(
            tzinfo=None
        ) - timedelta(hours=100)
        mock_session = self._make_mock_session(old_ts)

        result = await check_upstream_freshness(
            "fundamental_metrics", mock_session
        )
        assert result.is_ready is False
        assert "ttm_financials" in result.stale_dependencies
