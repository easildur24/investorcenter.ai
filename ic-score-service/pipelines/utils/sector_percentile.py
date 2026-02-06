"""Sector-relative percentile calculator for IC Score v2.1.

This module provides utilities for calculating sector-relative percentiles
for stock metrics, enabling more meaningful comparisons within industry groups.
"""
from typing import Dict, List, Optional, Set
from dataclasses import dataclass
from decimal import Decimal
import logging
import numpy as np
from sqlalchemy import select, text
from sqlalchemy.ext.asyncio import AsyncSession

from models import SectorPercentile

logger = logging.getLogger(__name__)


@dataclass
class SectorStats:
    """Distribution statistics for a metric within a sector."""
    sector: str
    metric: str
    min_value: float
    p10: float
    p25: float
    p50: float  # median
    p75: float
    p90: float
    max_value: float
    mean: float
    std_dev: float
    sample_count: int


class SectorPercentileCalculator:
    """Calculate sector-relative percentiles for stock metrics.

    This class handles the mapping of raw metric values to percentile scores
    within a sector context. For valuation metrics (lower is better), the
    percentile is inverted so higher scores always indicate "better" values.
    """

    # Metrics where lower values are better (inverted scoring)
    LOWER_IS_BETTER: Set[str] = {
        'pe_ratio', 'ps_ratio', 'pb_ratio', 'ev_ebitda', 'peg_ratio',
        'debt_to_equity', 'net_debt_to_ebitda', 'interest_coverage_inv',
        'current_ratio_inv', 'quick_ratio_inv'
    }

    # All metrics tracked for sector percentiles
    TRACKED_METRICS: List[str] = [
        # Valuation (lower is better)
        'pe_ratio', 'ps_ratio', 'pb_ratio', 'ev_ebitda', 'peg_ratio',
        # Profitability (higher is better)
        'roe', 'roa', 'roic', 'gross_margin', 'operating_margin', 'net_margin',
        # Growth (higher is better)
        'revenue_growth_yoy', 'earnings_growth_yoy', 'eps_growth_yoy',
        # Financial Health
        'current_ratio', 'quick_ratio', 'debt_to_equity', 'interest_coverage',
        # Efficiency
        'asset_turnover', 'inventory_turnover', 'receivables_turnover',
        # Market
        'dividend_yield', 'free_cash_flow_yield', 'earnings_yield',
    ]

    def __init__(self, session: AsyncSession):
        """Initialize calculator with database session.

        Args:
            session: Async SQLAlchemy session for database access
        """
        self.session = session
        self._cache: Dict[str, SectorStats] = {}

    async def get_percentile(
        self,
        sector: str,
        metric: str,
        value: float
    ) -> Optional[float]:
        """Get percentile score (0-100) for a value within its sector.

        Returns higher scores for "better" values:
        - For most metrics: higher value = higher percentile
        - For valuation metrics: lower value = higher percentile (inverted)

        Args:
            sector: GICS sector name (e.g., "Technology", "Healthcare")
            metric: Metric name (e.g., "pe_ratio", "roe")
            value: Raw metric value to score

        Returns:
            Percentile score 0-100, or None if sector stats unavailable
        """
        if value is None:
            return None

        stats = await self._get_sector_stats(sector, metric)
        if not stats:
            logger.warning(f"No sector stats for {sector}/{metric}")
            return None

        # Calculate raw percentile based on distribution
        raw_pct = self._calculate_percentile(value, stats)

        # Invert for "lower is better" metrics so higher score = better
        if metric in self.LOWER_IS_BETTER:
            return 100.0 - raw_pct

        return raw_pct

    def _calculate_percentile(self, value: float, stats: SectorStats) -> float:
        """Interpolate percentile based on distribution statistics.

        Uses piecewise linear interpolation between known percentile points
        for smoother distribution than simple ranking.

        Args:
            value: Raw value to score
            stats: Distribution statistics for interpolation

        Returns:
            Raw percentile 0-100 (not inverted)
        """
        # Handle edge cases
        if value <= stats.min_value:
            return 0.0
        if value >= stats.max_value:
            return 100.0

        # Piecewise linear interpolation between percentile points
        breakpoints = [
            (stats.min_value, 0),
            (stats.p10, 10),
            (stats.p25, 25),
            (stats.p50, 50),
            (stats.p75, 75),
            (stats.p90, 90),
            (stats.max_value, 100),
        ]

        for i in range(len(breakpoints) - 1):
            low_val, low_pct = breakpoints[i]
            high_val, high_pct = breakpoints[i + 1]

            if low_val <= value <= high_val:
                # Avoid division by zero
                if high_val == low_val:
                    return low_pct

                # Linear interpolation
                ratio = (value - low_val) / (high_val - low_val)
                return low_pct + ratio * (high_pct - low_pct)

        # Fallback (shouldn't reach here)
        return 50.0

    async def _get_sector_stats(self, sector: str, metric: str) -> Optional[SectorStats]:
        """Fetch sector statistics from database or cache.

        Uses materialized view for fast lookups. Cache is per-request
        to avoid stale data across calculation runs.

        Args:
            sector: Sector name
            metric: Metric name

        Returns:
            SectorStats object or None if not found
        """
        cache_key = f"{sector}:{metric}"

        if cache_key in self._cache:
            return self._cache[cache_key]

        # Query from materialized view for latest stats
        query = text("""
            SELECT
                sector, metric_name,
                min_value, p10_value, p25_value, p50_value,
                p75_value, p90_value, max_value,
                mean_value, std_dev, sample_count
            FROM mv_latest_sector_percentiles
            WHERE sector = :sector AND metric_name = :metric
        """)

        result = await self.session.execute(
            query,
            {"sector": sector, "metric": metric}
        )
        row = result.fetchone()

        if not row:
            return None

        stats = SectorStats(
            sector=row.sector,
            metric=row.metric_name,
            min_value=float(row.min_value) if row.min_value else 0,
            p10=float(row.p10_value) if row.p10_value else 0,
            p25=float(row.p25_value) if row.p25_value else 0,
            p50=float(row.p50_value) if row.p50_value else 0,
            p75=float(row.p75_value) if row.p75_value else 0,
            p90=float(row.p90_value) if row.p90_value else 0,
            max_value=float(row.max_value) if row.max_value else 0,
            mean=float(row.mean_value) if row.mean_value else 0,
            std_dev=float(row.std_dev) if row.std_dev else 0,
            sample_count=row.sample_count or 0,
        )

        self._cache[cache_key] = stats
        return stats

    async def get_sector_rank(
        self,
        sector: str,
        ticker: str,
        overall_score: float
    ) -> tuple[int, int]:
        """Get a stock's rank within its sector by IC Score.

        Args:
            sector: GICS sector name
            ticker: Stock ticker symbol
            overall_score: Stock's overall IC Score

        Returns:
            Tuple of (rank, total_in_sector)
        """
        query = text("""
            SELECT
                COUNT(*) FILTER (WHERE overall_score > :score) + 1 as rank,
                COUNT(*) as total
            FROM ic_scores ics
            JOIN companies c ON ics.ticker = c.ticker
            WHERE c.sector = :sector
            AND ics.date = (SELECT MAX(date) FROM ic_scores WHERE ticker = ics.ticker)
        """)

        result = await self.session.execute(
            query,
            {"sector": sector, "score": overall_score}
        )
        row = result.fetchone()

        if row:
            return int(row.rank), int(row.total)
        return 0, 0

    def clear_cache(self):
        """Clear the statistics cache. Call between batch runs."""
        self._cache.clear()


class SectorPercentileAggregator:
    """Aggregate and store sector percentile statistics.

    This class handles the daily calculation of sector-level distribution
    statistics for all tracked metrics. Designed to run as a scheduled job.
    """

    # Minimum samples required to calculate valid percentiles
    MIN_SAMPLE_SIZE = 5

    def __init__(self, session: AsyncSession):
        """Initialize aggregator with database session.

        Args:
            session: Async SQLAlchemy session for database access
        """
        self.session = session

    async def calculate_all_sectors(self) -> Dict[str, int]:
        """Calculate percentile stats for all sectors and metrics.

        Queries current fundamental metrics for all active companies,
        groups by sector, and calculates distribution statistics.

        Returns:
            Dict mapping sector names to number of metrics calculated
        """
        results = {}

        # Get list of active sectors
        sectors = await self._get_active_sectors()
        logger.info(f"Calculating percentiles for {len(sectors)} sectors")

        for sector in sectors:
            count = await self.calculate_sector(sector)
            results[sector] = count
            logger.info(f"Sector '{sector}': {count} metrics calculated")

        # Refresh materialized view
        await self._refresh_materialized_view()

        return results

    async def calculate_sector(self, sector: str) -> int:
        """Calculate percentile stats for a single sector.

        Args:
            sector: GICS sector name

        Returns:
            Number of metrics successfully calculated
        """
        metrics_calculated = 0

        for metric in SectorPercentileCalculator.TRACKED_METRICS:
            values = await self._get_metric_values(sector, metric)

            if len(values) < self.MIN_SAMPLE_SIZE:
                logger.debug(f"Skipping {sector}/{metric}: only {len(values)} samples")
                continue

            # Calculate distribution statistics
            arr = np.array(values)
            stats = SectorPercentile(
                sector=sector,
                metric_name=metric,
                min_value=Decimal(str(float(np.min(arr)))),
                p10_value=Decimal(str(float(np.percentile(arr, 10)))),
                p25_value=Decimal(str(float(np.percentile(arr, 25)))),
                p50_value=Decimal(str(float(np.percentile(arr, 50)))),
                p75_value=Decimal(str(float(np.percentile(arr, 75)))),
                p90_value=Decimal(str(float(np.percentile(arr, 90)))),
                max_value=Decimal(str(float(np.max(arr)))),
                mean_value=Decimal(str(float(np.mean(arr)))),
                std_dev=Decimal(str(float(np.std(arr)))),
                sample_count=len(values),
            )

            # Upsert into database
            await self._upsert_percentile(stats)
            metrics_calculated += 1

        return metrics_calculated

    async def _get_active_sectors(self) -> List[str]:
        """Get list of sectors with active companies."""
        query = text("""
            SELECT DISTINCT sector
            FROM companies
            WHERE sector IS NOT NULL
            AND is_active = true
            ORDER BY sector
        """)
        result = await self.session.execute(query)
        return [row[0] for row in result.fetchall()]

    async def _get_metric_values(self, sector: str, metric: str) -> List[float]:
        """Get all values for a metric within a sector.

        Uses the latest fundamental metrics for each company in the sector,
        filtering out NULL values and obvious outliers.
        """
        # Map metric names to database columns
        column_mapping = {
            'pe_ratio': 'f.pe_ratio',
            'ps_ratio': 'f.ps_ratio',
            'pb_ratio': 'f.pb_ratio',
            'roe': 'f.roe',
            'roa': 'f.roa',
            'roic': 'f.roic',
            'gross_margin': 'f.gross_margin',
            'operating_margin': 'f.operating_margin',
            'net_margin': 'f.net_margin',
            'debt_to_equity': 'f.debt_to_equity',
            'current_ratio': 'f.current_ratio',
            'quick_ratio': 'f.quick_ratio',
            'dividend_yield': 'vr.dividend_yield',
            'ev_ebitda': 'vr.ev_ebitda',
            'revenue_growth_yoy': 'fm.revenue_growth_yoy',
            'earnings_growth_yoy': 'fm.earnings_growth_yoy',
            'eps_growth_yoy': 'fm.eps_growth_yoy',
            'peg_ratio': 'vr.peg_ratio',
        }

        column = column_mapping.get(metric)
        if not column:
            return []

        # Determine which table to query based on metric
        if column.startswith('vr.'):
            table_join = """
                JOIN valuation_ratios vr ON c.ticker = vr.ticker
                AND vr.calculation_date = (
                    SELECT MAX(calculation_date) FROM valuation_ratios WHERE ticker = c.ticker
                )
            """
            col_ref = column
        elif column.startswith('fm.'):
            table_join = """
                JOIN fundamental_metrics fm ON c.ticker = fm.ticker
                AND fm.calculation_date = (
                    SELECT MAX(calculation_date) FROM fundamental_metrics WHERE ticker = c.ticker
                )
            """
            col_ref = column
        else:
            table_join = """
                JOIN financials f ON c.ticker = f.ticker
                AND f.period_end_date = (
                    SELECT MAX(period_end_date) FROM financials WHERE ticker = c.ticker
                )
            """
            col_ref = column

        query = text(f"""
            SELECT {col_ref} as value
            FROM companies c
            {table_join}
            WHERE c.sector = :sector
            AND c.is_active = true
            AND {col_ref} IS NOT NULL
            AND {col_ref} NOT IN ('NaN', 'Infinity', '-Infinity')
        """)

        result = await self.session.execute(query, {"sector": sector})
        values = [float(row.value) for row in result.fetchall()
                  if row.value is not None and np.isfinite(float(row.value))]

        # Remove extreme outliers (beyond 3 standard deviations)
        if len(values) > 10:
            arr = np.array(values)
            mean = np.mean(arr)
            std = np.std(arr)
            if std > 0:
                values = [v for v in values if abs(v - mean) <= 3 * std]

        return values

    async def _upsert_percentile(self, stats: SectorPercentile):
        """Insert or update percentile statistics."""
        query = text("""
            INSERT INTO sector_percentiles (
                sector, metric_name, calculated_at,
                min_value, p10_value, p25_value, p50_value,
                p75_value, p90_value, max_value,
                mean_value, std_dev, sample_count
            ) VALUES (
                :sector, :metric, CURRENT_DATE,
                :min_val, :p10, :p25, :p50,
                :p75, :p90, :max_val,
                :mean, :std, :count
            )
            ON CONFLICT (sector, metric_name, calculated_at)
            DO UPDATE SET
                min_value = EXCLUDED.min_value,
                p10_value = EXCLUDED.p10_value,
                p25_value = EXCLUDED.p25_value,
                p50_value = EXCLUDED.p50_value,
                p75_value = EXCLUDED.p75_value,
                p90_value = EXCLUDED.p90_value,
                max_value = EXCLUDED.max_value,
                mean_value = EXCLUDED.mean_value,
                std_dev = EXCLUDED.std_dev,
                sample_count = EXCLUDED.sample_count
        """)

        await self.session.execute(query, {
            "sector": stats.sector,
            "metric": stats.metric_name,
            "min_val": stats.min_value,
            "p10": stats.p10_value,
            "p25": stats.p25_value,
            "p50": stats.p50_value,
            "p75": stats.p75_value,
            "p90": stats.p90_value,
            "max_val": stats.max_value,
            "mean": stats.mean_value,
            "std": stats.std_dev,
            "count": stats.sample_count,
        })
        await self.session.commit()

    async def _refresh_materialized_view(self):
        """Refresh the materialized view for fast lookups."""
        await self.session.execute(
            text("REFRESH MATERIALIZED VIEW CONCURRENTLY mv_latest_sector_percentiles")
        )
        await self.session.commit()
        logger.info("Refreshed mv_latest_sector_percentiles")
