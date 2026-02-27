"""Tests for pipeline dependency checker.

Comprehensive tests covering the dependency graph structure, execution
ordering, upstream dependency resolution, cycle detection, freshness
checks, and edge cases with mock database sessions.
"""

from datetime import date, datetime, timedelta, timezone
from unittest.mock import AsyncMock, MagicMock, patch

import pytest

from pipelines.utils.dependency_checker import (
    PIPELINE_DEPENDENCIES,
    PIPELINE_FRESHNESS_COLUMN,
    PIPELINE_OUTPUT_TABLE,
    DependencyCheckResult,
    check_upstream_freshness,
    get_all_pipelines,
    get_all_upstream_dependencies,
    get_execution_order,
    get_upstream_dependencies,
    has_cycle,
)


# =====================================================================
# TestDependencyCheckResult
# =====================================================================


class TestDependencyCheckResult:
    """Tests for the DependencyCheckResult dataclass."""

    def test_default_values(self):
        """Default values are sensible."""
        result = DependencyCheckResult(is_ready=True)
        assert result.is_ready is True
        assert result.stale_dependencies == []
        assert result.details == {}

    def test_custom_values(self):
        """Can set custom stale_dependencies and details."""
        result = DependencyCheckResult(
            is_ready=False,
            stale_dependencies=["a", "b"],
            details={"a": None, "b": datetime(2024, 1, 1)},
        )
        assert result.is_ready is False
        assert len(result.stale_dependencies) == 2
        assert result.details["a"] is None

    def test_not_ready_with_stale(self):
        result = DependencyCheckResult(
            is_ready=False,
            stale_dependencies=["sec_financials"],
        )
        assert not result.is_ready
        assert "sec_financials" in result.stale_dependencies

    def test_ready_with_details(self):
        ts = datetime(2025, 1, 15, 12, 0, 0)
        result = DependencyCheckResult(
            is_ready=True,
            details={"ttm_financials": ts},
        )
        assert result.is_ready
        assert result.details["ttm_financials"] == ts


# =====================================================================
# TestDependencyGraph
# =====================================================================


class TestDependencyGraph:
    def test_no_cycles(self):
        """The dependency graph must be a DAG (no cycles)."""
        assert has_cycle() is False

    def test_all_pipelines_present(self):
        """All 17 pipelines must be present in the graph."""
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

    def test_ttm_financials_depends_on_sec(self):
        assert PIPELINE_DEPENDENCIES["ttm_financials"] == [
            "sec_financials"
        ]

    def test_fundamental_metrics_depends_on_ttm(self):
        assert PIPELINE_DEPENDENCIES["fundamental_metrics"] == [
            "ttm_financials"
        ]

    def test_risk_metrics_deps(self):
        deps = set(PIPELINE_DEPENDENCIES["risk_metrics"])
        assert deps == {"daily_price_update", "benchmark_data"}

    def test_valuation_ratios_deps(self):
        deps = set(PIPELINE_DEPENDENCIES["valuation_ratios"])
        assert deps == {"daily_price_update", "ttm_financials"}


# =====================================================================
# TestGetAllPipelines
# =====================================================================


class TestGetAllPipelines:
    def test_returns_list(self):
        result = get_all_pipelines()
        assert isinstance(result, list)

    def test_returns_all_pipeline_names(self):
        result = get_all_pipelines()
        assert set(result) == set(PIPELINE_DEPENDENCIES.keys())

    def test_count_matches(self):
        result = get_all_pipelines()
        assert len(result) == len(PIPELINE_DEPENDENCIES)

    def test_all_elements_are_strings(self):
        for pipeline in get_all_pipelines():
            assert isinstance(pipeline, str)


# =====================================================================
# TestGetUpstreamDependencies
# =====================================================================


class TestGetUpstreamDependencies:
    def test_direct_deps(self):
        """fundamental_metrics depends directly on ttm_financials."""
        deps = get_upstream_dependencies("fundamental_metrics")
        assert deps == ["ttm_financials"]

    def test_ingestion_pipelines_return_empty(self):
        """All ingestion pipelines return empty dependency lists."""
        ingestion = [
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
        for pipeline in ingestion:
            deps = get_upstream_dependencies(pipeline)
            assert deps == [], f"{pipeline} should have no deps"

    def test_ttm_financials_depends_on_sec(self):
        deps = get_upstream_dependencies("ttm_financials")
        assert deps == ["sec_financials"]

    def test_risk_metrics_deps(self):
        deps = get_upstream_dependencies("risk_metrics")
        assert set(deps) == {"daily_price_update", "benchmark_data"}

    def test_valuation_ratios_deps(self):
        deps = get_upstream_dependencies("valuation_ratios")
        assert set(deps) == {"daily_price_update", "ttm_financials"}

    def test_ic_score_has_six_deps(self):
        deps = get_upstream_dependencies("ic_score_calculator")
        assert len(deps) == 6

    def test_unknown_pipeline_raises(self):
        """Unknown pipeline name raises ValueError."""
        with pytest.raises(ValueError, match="Unknown pipeline"):
            get_upstream_dependencies("nonexistent")

    def test_empty_string_raises(self):
        with pytest.raises(ValueError, match="Unknown pipeline"):
            get_upstream_dependencies("")


# =====================================================================
# TestGetAllUpstreamDependencies
# =====================================================================


class TestGetAllUpstreamDependencies:
    def test_unknown_pipeline_raises(self):
        with pytest.raises(ValueError, match="Unknown pipeline"):
            get_all_upstream_dependencies("does_not_exist")

    def test_ingestion_pipeline_has_no_transitive_deps(self):
        deps = get_all_upstream_dependencies("benchmark_data")
        assert deps == []

    def test_ttm_financials_transitive(self):
        """ttm_financials depends on sec_financials only."""
        deps = get_all_upstream_dependencies("ttm_financials")
        assert deps == ["sec_financials"]

    def test_fundamental_metrics_transitive(self):
        """fundamental_metrics transitively depends on sec_financials
        via ttm_financials."""
        deps = get_all_upstream_dependencies("fundamental_metrics")
        assert "ttm_financials" in deps
        assert "sec_financials" in deps
        assert len(deps) == 2

    def test_sector_percentiles_transitive(self):
        """sector_percentiles -> fundamental_metrics -> ttm_financials
        -> sec_financials."""
        deps = get_all_upstream_dependencies("sector_percentiles")
        assert "fundamental_metrics" in deps
        assert "ttm_financials" in deps
        assert "sec_financials" in deps

    def test_ic_score_has_all_ingestion_roots(self):
        """ic_score_calculator transitively depends on key ingestion
        pipelines."""
        deps = get_all_upstream_dependencies("ic_score_calculator")
        expected_ingestion = {
            "sec_financials",
            "daily_price_update",
            "benchmark_data",
        }
        assert expected_ingestion.issubset(set(deps))

    def test_ic_score_transitive_count(self):
        """ic_score_calculator should have many transitive deps."""
        deps = get_all_upstream_dependencies("ic_score_calculator")
        # Should include all calculator deps + their transitive deps
        assert len(deps) >= 8

    def test_result_is_sorted(self):
        deps = get_all_upstream_dependencies("ic_score_calculator")
        assert deps == sorted(deps)

    def test_no_duplicates(self):
        deps = get_all_upstream_dependencies("ic_score_calculator")
        assert len(deps) == len(set(deps))

    def test_pipeline_not_in_own_deps(self):
        """A pipeline should not appear in its own transitive deps."""
        for pipeline in PIPELINE_DEPENDENCIES:
            deps = get_all_upstream_dependencies(pipeline)
            assert pipeline not in deps, (
                f"{pipeline} should not be in its own transitive deps"
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

    def test_all_pipelines_placed(self):
        """Every pipeline appears in exactly one tier."""
        tiers = get_execution_order()
        all_placed = [p for tier in tiers for p in tier]
        assert set(all_placed) == set(PIPELINE_DEPENDENCIES.keys())
        # No duplicates
        assert len(all_placed) == len(set(all_placed))

    def test_deps_appear_before_dependents(self):
        """Every dependency appears in an earlier tier than its
        dependent."""
        tiers = get_execution_order()
        tier_index = {}
        for i, tier in enumerate(tiers):
            for p in tier:
                tier_index[p] = i

        for pipeline, deps in PIPELINE_DEPENDENCIES.items():
            for dep in deps:
                assert tier_index[dep] < tier_index[pipeline], (
                    f"{dep} (tier {tier_index[dep]}) should be "
                    f"before {pipeline} (tier {tier_index[pipeline]})"
                )

    def test_each_tier_is_sorted(self):
        """Each tier's pipeline list is sorted alphabetically."""
        tiers = get_execution_order()
        for tier in tiers:
            assert tier == sorted(tier)

    def test_ingestion_tier_count(self):
        """First tier should contain all 9 ingestion pipelines."""
        tiers = get_execution_order()
        assert len(tiers[0]) == 9


# =====================================================================
# TestHasCycle
# =====================================================================


class TestHasCycle:
    def test_no_cycle_in_real_graph(self):
        assert has_cycle() is False

    def test_cycle_detected(self):
        """has_cycle returns True when a cycle exists."""
        cyclic_deps = {
            "a": ["b"],
            "b": ["c"],
            "c": ["a"],
        }
        with patch(
            "pipelines.utils.dependency_checker.PIPELINE_DEPENDENCIES",
            cyclic_deps,
        ):
            assert has_cycle() is True

    def test_self_cycle_detected(self):
        """has_cycle returns True for self-referencing pipeline."""
        self_ref_deps = {
            "a": ["a"],
        }
        with patch(
            "pipelines.utils.dependency_checker.PIPELINE_DEPENDENCIES",
            self_ref_deps,
        ):
            assert has_cycle() is True

    def test_two_node_cycle(self):
        """Two pipelines that depend on each other."""
        cyclic_deps = {
            "a": ["b"],
            "b": ["a"],
        }
        with patch(
            "pipelines.utils.dependency_checker.PIPELINE_DEPENDENCIES",
            cyclic_deps,
        ):
            assert has_cycle() is True

    def test_partial_cycle(self):
        """Graph has some valid nodes and a cycle."""
        deps = {
            "root": [],
            "a": ["root"],
            "b": ["a", "c"],
            "c": ["b"],
        }
        with patch(
            "pipelines.utils.dependency_checker.PIPELINE_DEPENDENCIES",
            deps,
        ):
            assert has_cycle() is True

    def test_execution_order_raises_on_cycle(self):
        """get_execution_order should raise ValueError on cycle."""
        cyclic_deps = {
            "a": ["b"],
            "b": ["a"],
        }
        with patch(
            "pipelines.utils.dependency_checker.PIPELINE_DEPENDENCIES",
            cyclic_deps,
        ):
            with pytest.raises(ValueError, match="cycle"):
                get_execution_order()


# =====================================================================
# TestOutputTableMapping
# =====================================================================


class TestOutputTableMapping:
    def test_every_pipeline_has_output_table(self):
        """Every pipeline in the dependency graph has an output table
        mapping."""
        for pipeline in PIPELINE_DEPENDENCIES:
            assert pipeline in PIPELINE_OUTPUT_TABLE, (
                f"{pipeline} missing from PIPELINE_OUTPUT_TABLE"
            )

    def test_every_output_table_has_freshness_column(self):
        """Every output table has a freshness column defined."""
        for pipeline, table in PIPELINE_OUTPUT_TABLE.items():
            assert table in PIPELINE_FRESHNESS_COLUMN, (
                f"Table '{table}' (from pipeline '{pipeline}') "
                f"missing from PIPELINE_FRESHNESS_COLUMN"
            )

    def test_output_table_values_are_strings(self):
        for pipeline, table in PIPELINE_OUTPUT_TABLE.items():
            assert isinstance(table, str)
            assert len(table) > 0

    def test_freshness_column_values_are_strings(self):
        for table, col in PIPELINE_FRESHNESS_COLUMN.items():
            assert isinstance(col, str)
            assert len(col) > 0


# =====================================================================
# TestCheckUpstreamFreshness
# =====================================================================


class TestCheckUpstreamFreshness:
    @staticmethod
    def _make_mock_session(latest_timestamp):
        """Create a mock AsyncSession that returns a given timestamp
        from MAX() queries."""
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
    async def test_no_deps_doesnt_query_db(self):
        """For a pipeline with no deps, DB should not be queried."""
        mock_session = AsyncMock()
        await check_upstream_freshness(
            "sec_financials", mock_session
        )
        mock_session.execute.assert_not_awaited()

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

    @pytest.mark.asyncio
    async def test_null_latest_timestamp_is_stale(self):
        """Null latest timestamp means the dependency is stale."""
        mock_session = self._make_mock_session(None)

        result = await check_upstream_freshness(
            "fundamental_metrics", mock_session
        )
        assert result.is_ready is False
        assert "ttm_financials" in result.stale_dependencies

    @pytest.mark.asyncio
    async def test_db_exception_marks_stale(self):
        """Database exception marks the dependency as stale."""
        mock_session = AsyncMock()
        mock_session.execute.side_effect = Exception(
            "DB connection lost"
        )

        result = await check_upstream_freshness(
            "fundamental_metrics", mock_session
        )
        assert result.is_ready is False
        assert "ttm_financials" in result.stale_dependencies
        assert result.details["ttm_financials"] is None

    @pytest.mark.asyncio
    async def test_custom_max_age(self):
        """Custom max_age_hours changes freshness threshold."""
        # 5 hours ago -- stale with max_age=4, fresh with max_age=6
        ts = datetime.now(timezone.utc).replace(
            tzinfo=None
        ) - timedelta(hours=5)
        mock_session = self._make_mock_session(ts)

        result_stale = await check_upstream_freshness(
            "fundamental_metrics", mock_session, max_age_hours=4
        )
        assert result_stale.is_ready is False

        result_fresh = await check_upstream_freshness(
            "fundamental_metrics", mock_session, max_age_hours=6
        )
        assert result_fresh.is_ready is True

    @pytest.mark.asyncio
    async def test_multiple_deps_all_fresh(self):
        """Pipeline with multiple deps is ready when all are fresh."""
        recent_ts = datetime.now(timezone.utc).replace(
            tzinfo=None
        ) - timedelta(hours=1)
        mock_session = self._make_mock_session(recent_ts)

        result = await check_upstream_freshness(
            "ic_score_calculator", mock_session
        )
        assert result.is_ready is True
        assert result.stale_dependencies == []

    @pytest.mark.asyncio
    async def test_multiple_deps_one_stale(self):
        """If one dep out of many is stale, pipeline is not ready."""
        call_count = 0

        async def dynamic_execute(*args, **kwargs):
            nonlocal call_count
            call_count += 1
            mock_row = MagicMock()
            # Make every other call return a stale timestamp
            if call_count == 3:
                mock_row.__getitem__ = lambda self, idx: None
            else:
                recent = datetime.now(timezone.utc).replace(
                    tzinfo=None
                ) - timedelta(hours=1)
                mock_row.__getitem__ = lambda self, idx: recent

            mock_result = MagicMock()
            mock_result.fetchone.return_value = mock_row
            return mock_result

        mock_session = AsyncMock()
        mock_session.execute.side_effect = dynamic_execute

        result = await check_upstream_freshness(
            "ic_score_calculator", mock_session
        )
        assert result.is_ready is False
        assert len(result.stale_dependencies) >= 1

    @pytest.mark.asyncio
    async def test_details_populated(self):
        """Result details dict maps dependency names to timestamps."""
        recent_ts = datetime.now(timezone.utc).replace(
            tzinfo=None
        ) - timedelta(hours=1)
        mock_session = self._make_mock_session(recent_ts)

        result = await check_upstream_freshness(
            "fundamental_metrics", mock_session
        )
        assert "ttm_financials" in result.details
        assert result.details["ttm_financials"] == recent_ts

    @pytest.mark.asyncio
    async def test_details_populated_for_stale(self):
        """Stale deps should still have details entry."""
        mock_session = self._make_mock_session(None)

        result = await check_upstream_freshness(
            "fundamental_metrics", mock_session
        )
        assert "ttm_financials" in result.details
        assert result.details["ttm_financials"] is None

    @pytest.mark.asyncio
    async def test_date_object_handling(self):
        """A date object (not datetime) returned from DB should
        be handled correctly for freshness comparison."""
        # Use a recent date (today) to test the date-to-datetime
        # conversion path
        recent_date = date.today()
        mock_session = self._make_mock_session(recent_date)

        # The function converts date to datetime for comparison.
        # Today should be fresh if max_age_hours is generous.
        result = await check_upstream_freshness(
            "fundamental_metrics", mock_session, max_age_hours=48
        )
        # Today's date converted to midnight datetime should be
        # within 48 hours
        assert result.is_ready is True

    @pytest.mark.asyncio
    async def test_date_object_stale(self):
        """An old date object should be detected as stale."""
        old_date = date.today() - timedelta(days=10)
        mock_session = self._make_mock_session(old_date)

        result = await check_upstream_freshness(
            "fundamental_metrics", mock_session, max_age_hours=48
        )
        assert result.is_ready is False

    @pytest.mark.asyncio
    async def test_unknown_pipeline_raises(self):
        """Unknown pipeline should raise ValueError."""
        mock_session = AsyncMock()
        with pytest.raises(ValueError, match="Unknown pipeline"):
            await check_upstream_freshness(
                "nonexistent_pipeline", mock_session
            )

    @pytest.mark.asyncio
    async def test_all_ingestion_pipelines_always_ready(self):
        """Every ingestion pipeline should always be ready."""
        mock_session = AsyncMock()
        ingestion = [
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
        for pipeline in ingestion:
            result = await check_upstream_freshness(
                pipeline, mock_session
            )
            assert result.is_ready is True, (
                f"{pipeline} should always be ready"
            )

    @pytest.mark.asyncio
    async def test_empty_fetchone_result_is_stale(self):
        """If fetchone returns a row where row[0] is None,
        it should be marked stale."""
        mock_row = MagicMock()
        mock_row.__getitem__ = lambda self, idx: None

        mock_result = MagicMock()
        mock_result.fetchone.return_value = mock_row

        mock_session = AsyncMock()
        mock_session.execute.return_value = mock_result

        result = await check_upstream_freshness(
            "ttm_financials", mock_session
        )
        assert result.is_ready is False

    @pytest.mark.asyncio
    async def test_no_row_returned_is_stale(self):
        """If fetchone returns None (no rows), it should be stale."""
        mock_result = MagicMock()
        mock_result.fetchone.return_value = None

        mock_session = AsyncMock()
        mock_session.execute.return_value = mock_result

        result = await check_upstream_freshness(
            "ttm_financials", mock_session
        )
        assert result.is_ready is False
