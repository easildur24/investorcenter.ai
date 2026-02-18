"""Pipeline dependency checker.

Validates that upstream pipeline outputs are fresh before a
downstream pipeline runs.  Encodes the dependency graph from
the IC Score pipeline execution schedule.

Usage in a pipeline:
    from pipelines.utils.dependency_checker import (
        check_upstream_freshness,
    )
    result = await check_upstream_freshness(
        "fundamental_metrics", session
    )
    if not result.is_ready:
        logger.error(f"Stale: {result.stale_dependencies}")
        sys.exit(2)  # Distinct from runtime error
"""

from dataclasses import dataclass, field
from datetime import datetime, timedelta, timezone
from typing import Dict, List, Optional

from sqlalchemy import text
from sqlalchemy.ext.asyncio import AsyncSession

# ============================================================
# Pipeline dependency graph
# ============================================================

# Map from pipeline name -> list of upstream pipelines it depends on
PIPELINE_DEPENDENCIES: Dict[str, List[str]] = {
    # Ingestion pipelines (no upstream dependencies)
    "benchmark_data": [],
    "treasury_rates": [],
    "sec_financials": [],
    "sec_13f_holdings": [],
    "analyst_ratings": [],
    "insider_trades": [],
    "daily_price_update": [],
    "news_sentiment": [],
    "reddit_sentiment": [],
    # Calculator pipelines (have upstream dependencies)
    "ttm_financials": ["sec_financials"],
    "fundamental_metrics": ["ttm_financials"],
    "sector_percentiles": ["fundamental_metrics"],
    "fair_value": ["fundamental_metrics"],
    "technical_indicators": ["daily_price_update"],
    "valuation_ratios": [
        "daily_price_update",
        "ttm_financials",
    ],
    "risk_metrics": [
        "daily_price_update",
        "benchmark_data",
    ],
    # IC Score depends on all calculators
    "ic_score_calculator": [
        "fundamental_metrics",
        "sector_percentiles",
        "fair_value",
        "technical_indicators",
        "valuation_ratios",
        "risk_metrics",
    ],
}

# Table that each pipeline writes to (for freshness checks)
PIPELINE_OUTPUT_TABLE: Dict[str, str] = {
    "benchmark_data": "benchmark_returns",
    "treasury_rates": "treasury_rates",
    "sec_financials": "financials",
    "sec_13f_holdings": "institutional_holdings",
    "analyst_ratings": "analyst_ratings",
    "insider_trades": "insider_trades",
    "daily_price_update": "stock_prices",
    "news_sentiment": "news_articles",
    "reddit_sentiment": "reddit_posts",
    "ttm_financials": "ttm_financials",
    "fundamental_metrics": "fundamental_metrics_extended",
    "sector_percentiles": "fundamental_metrics_extended",
    "fair_value": "fundamental_metrics_extended",
    "technical_indicators": "technical_indicators",
    "valuation_ratios": "valuation_ratios",
    "risk_metrics": "risk_metrics",
    "ic_score_calculator": "ic_scores",
}

# Column to check for freshness (most recent timestamp)
PIPELINE_FRESHNESS_COLUMN: Dict[str, str] = {
    "benchmark_returns": "time",
    "treasury_rates": "date",
    "financials": "created_at",
    "institutional_holdings": "created_at",
    "analyst_ratings": "created_at",
    "insider_trades": "created_at",
    "daily_price_update": "time",
    "stock_prices": "time",
    "news_articles": "created_at",
    "reddit_posts": "created_at",
    "ttm_financials": "created_at",
    "fundamental_metrics_extended": "updated_at",
    "technical_indicators": "time",
    "valuation_ratios": "created_at",
    "risk_metrics": "time",
    "ic_scores": "created_at",
}


@dataclass
class DependencyCheckResult:
    """Result of a dependency freshness check."""

    is_ready: bool
    stale_dependencies: List[str] = field(default_factory=list)
    details: Dict[str, Optional[datetime]] = field(
        default_factory=dict
    )


def get_all_pipelines() -> List[str]:
    """Return all pipeline names."""
    return list(PIPELINE_DEPENDENCIES.keys())


def get_upstream_dependencies(pipeline_name: str) -> List[str]:
    """Return direct upstream dependencies for a pipeline."""
    if pipeline_name not in PIPELINE_DEPENDENCIES:
        raise ValueError(f"Unknown pipeline: {pipeline_name}")
    return PIPELINE_DEPENDENCIES[pipeline_name]


def get_all_upstream_dependencies(
    pipeline_name: str,
) -> List[str]:
    """Return ALL upstream dependencies (transitive closure).

    E.g., ic_score_calculator depends on fundamental_metrics,
    which depends on ttm_financials, which depends on
    sec_financials.
    """
    if pipeline_name not in PIPELINE_DEPENDENCIES:
        raise ValueError(f"Unknown pipeline: {pipeline_name}")

    visited = set()
    stack = list(PIPELINE_DEPENDENCIES[pipeline_name])

    while stack:
        dep = stack.pop()
        if dep not in visited:
            visited.add(dep)
            stack.extend(PIPELINE_DEPENDENCIES.get(dep, []))

    return sorted(visited)


def get_execution_order() -> List[List[str]]:
    """Return pipelines grouped by execution tier.

    Tier 0: no dependencies (can run first)
    Tier 1: depends only on Tier 0
    etc.

    Uses Kahn's algorithm for topological sort.
    """
    # Build in-degree counts
    in_degree: Dict[str, int] = {
        p: 0 for p in PIPELINE_DEPENDENCIES
    }
    for deps in PIPELINE_DEPENDENCIES.values():
        for dep in deps:
            if dep in in_degree:
                in_degree[dep] = in_degree.get(dep, 0)

    # Calculate actual in-degree from reverse mapping
    in_degree = {p: 0 for p in PIPELINE_DEPENDENCIES}
    for pipeline, deps in PIPELINE_DEPENDENCIES.items():
        for dep in deps:
            pass  # dep is upstream of pipeline

    # Simpler approach: BFS from roots
    remaining = dict(PIPELINE_DEPENDENCIES)
    tiers: List[List[str]] = []

    while remaining:
        # Find pipelines whose dependencies are all already placed
        placed = {
            p
            for tier in tiers
            for p in tier
        }
        current_tier = [
            p
            for p, deps in remaining.items()
            if all(d in placed for d in deps)
        ]

        if not current_tier:
            # Cycle detected
            raise ValueError(
                f"Dependency cycle detected among: "
                f"{list(remaining.keys())}"
            )

        tiers.append(sorted(current_tier))
        for p in current_tier:
            del remaining[p]

    return tiers


def has_cycle() -> bool:
    """Check if the dependency graph has any cycles."""
    try:
        get_execution_order()
        return False
    except ValueError:
        return True


async def check_upstream_freshness(
    pipeline_name: str,
    session: AsyncSession,
    max_age_hours: int = 48,
) -> DependencyCheckResult:
    """Check that all upstream dependencies have fresh data.

    Args:
        pipeline_name: The pipeline about to run.
        session: Async database session.
        max_age_hours: Maximum age in hours for data to be
            considered fresh.

    Returns:
        DependencyCheckResult with is_ready=True if all
        upstream data is fresh enough.
    """
    deps = get_upstream_dependencies(pipeline_name)

    if not deps:
        return DependencyCheckResult(is_ready=True)

    cutoff = datetime.now(timezone.utc).replace(
        tzinfo=None
    ) - timedelta(hours=max_age_hours)
    stale = []
    details: Dict[str, Optional[datetime]] = {}

    for dep in deps:
        table = PIPELINE_OUTPUT_TABLE.get(dep)
        ts_col = PIPELINE_FRESHNESS_COLUMN.get(
            table, "created_at"
        )

        if not table:
            stale.append(dep)
            details[dep] = None
            continue

        try:
            result = await session.execute(
                text(
                    f"SELECT MAX({ts_col}) as latest"
                    f" FROM {table}"
                )
            )
            row = result.fetchone()
            latest = row[0] if row and row[0] else None

            details[dep] = latest

            if latest is None:
                stale.append(dep)
            elif hasattr(latest, "replace"):
                # Convert date to datetime if needed
                if not hasattr(latest, "hour"):
                    latest = datetime.combine(
                        latest,
                        datetime.min.time(),
                    )
                if latest < cutoff:
                    stale.append(dep)
        except Exception:
            stale.append(dep)
            details[dep] = None

    return DependencyCheckResult(
        is_ready=len(stale) == 0,
        stale_dependencies=stale,
        details=details,
    )
