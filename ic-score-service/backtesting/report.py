"""
Backtest report generation for IC Score validation.

This module generates detailed reports from backtest results
including performance summaries, charts, and statistical analysis.
"""

from dataclasses import dataclass, asdict
from datetime import date, datetime
from typing import Dict, List, Optional, Any
import json
import logging

from .backtester import BacktestResults, BacktestConfig, PeriodResult
from .metrics import BacktestMetrics, PerformanceCalculator

logger = logging.getLogger(__name__)


@dataclass
class DecilePerformance:
    """Performance summary for a single decile."""
    decile: int
    total_return: float
    annualized_return: float
    volatility: float
    sharpe_ratio: float
    max_drawdown: float
    avg_score: float
    num_periods: int


@dataclass
class BacktestSummary:
    """High-level backtest summary for API/UI."""
    # Configuration
    start_date: str
    end_date: str
    rebalance_frequency: str
    universe: str
    benchmark: str
    num_periods: int

    # Key findings
    top_decile_cagr: float
    bottom_decile_cagr: float
    spread_cagr: float
    benchmark_cagr: float
    top_vs_benchmark: float

    # Statistical validity
    hit_rate: float  # % periods top beats bottom
    monotonicity_score: float
    information_ratio: float

    # Risk metrics
    top_decile_sharpe: float
    top_decile_max_dd: float
    bottom_decile_sharpe: float

    # Decile breakdown
    decile_performance: List[DecilePerformance]


class BacktestReportGenerator:
    """
    Generate comprehensive reports from backtest results.

    Produces JSON summaries for API responses and detailed
    analysis for internal validation.
    """

    def __init__(self):
        self.perf_calc = PerformanceCalculator()

    def generate_summary(
        self,
        results: BacktestResults
    ) -> BacktestSummary:
        """
        Generate high-level summary from backtest results.

        This is suitable for API responses and UI display.
        """
        config = results.config

        # Calculate per-decile metrics
        decile_perf = self._calculate_decile_performance(results)

        # Get benchmark performance
        benchmark_returns = self._extract_benchmark_returns(results)
        benchmark_cagr = self._calculate_cagr(benchmark_returns)

        # Top and bottom decile stats
        top = decile_perf.get(1, self._empty_decile_perf(1))
        bottom = decile_perf.get(10, self._empty_decile_perf(10))

        return BacktestSummary(
            start_date=config.start_date.isoformat(),
            end_date=config.end_date.isoformat(),
            rebalance_frequency=config.rebalance_frequency.value,
            universe=config.universe,
            benchmark=config.benchmark,
            num_periods=len(set(pr.period_start for pr in results.period_results)),

            top_decile_cagr=top.annualized_return,
            bottom_decile_cagr=bottom.annualized_return,
            spread_cagr=top.annualized_return - bottom.annualized_return,
            benchmark_cagr=benchmark_cagr,
            top_vs_benchmark=top.annualized_return - benchmark_cagr,

            hit_rate=results.hit_rate,
            monotonicity_score=results.monotonicity_score,
            information_ratio=results.information_ratio,

            top_decile_sharpe=top.sharpe_ratio,
            top_decile_max_dd=top.max_drawdown,
            bottom_decile_sharpe=bottom.sharpe_ratio,

            decile_performance=list(decile_perf.values())
        )

    def generate_json_report(
        self,
        results: BacktestResults
    ) -> str:
        """Generate JSON report for API response."""
        summary = self.generate_summary(results)
        return json.dumps(asdict(summary), indent=2, default=str)

    def generate_detailed_report(
        self,
        results: BacktestResults
    ) -> Dict[str, Any]:
        """
        Generate detailed report for internal analysis.

        Includes period-by-period breakdown, sector analysis,
        and statistical tests.
        """
        summary = self.generate_summary(results)

        # Period-by-period returns for charting
        period_data = self._generate_period_data(results)

        # Cumulative returns by decile
        cumulative_returns = self._generate_cumulative_returns(results)

        # Sector breakdown
        sector_analysis = self._generate_sector_analysis(results)

        # Rolling metrics
        rolling_metrics = self._generate_rolling_metrics(results)

        # Statistical tests
        statistical_tests = self._run_statistical_tests(results)

        return {
            'summary': asdict(summary),
            'period_data': period_data,
            'cumulative_returns': cumulative_returns,
            'sector_analysis': sector_analysis,
            'rolling_metrics': rolling_metrics,
            'statistical_tests': statistical_tests,
            'generated_at': datetime.utcnow().isoformat(),
        }

    def _calculate_decile_performance(
        self,
        results: BacktestResults
    ) -> Dict[int, DecilePerformance]:
        """Calculate performance for each decile."""
        decile_perf: Dict[int, DecilePerformance] = {}

        for decile in range(1, 11):
            # Get returns for this decile
            returns = [
                pr.period_return
                for pr in results.period_results
                if pr.decile == decile
            ]

            if not returns:
                continue

            # Calculate metrics
            metrics = self.perf_calc.calculate_metrics(returns)

            # Average score
            avg_score = sum(
                pr.avg_score for pr in results.period_results
                if pr.decile == decile
            ) / len(returns)

            decile_perf[decile] = DecilePerformance(
                decile=decile,
                total_return=metrics.total_return,
                annualized_return=metrics.annualized_return,
                volatility=metrics.annualized_volatility,
                sharpe_ratio=metrics.sharpe_ratio,
                max_drawdown=metrics.max_drawdown,
                avg_score=avg_score,
                num_periods=len(returns)
            )

        return decile_perf

    def _extract_benchmark_returns(
        self,
        results: BacktestResults
    ) -> List[float]:
        """Extract unique benchmark returns by period."""
        period_benchmark: Dict[date, float] = {}

        for pr in results.period_results:
            if pr.period_start not in period_benchmark:
                period_benchmark[pr.period_start] = pr.benchmark_return

        return list(period_benchmark.values())

    def _calculate_cagr(self, returns: List[float]) -> float:
        """Calculate CAGR from returns."""
        if not returns:
            return 0.0

        total = 1.0
        for r in returns:
            total *= (1 + r)

        years = len(returns) / 12  # Assuming monthly
        if years <= 0 or total <= 0:
            return 0.0

        return (total ** (1 / years)) - 1

    def _empty_decile_perf(self, decile: int) -> DecilePerformance:
        """Return empty performance object."""
        return DecilePerformance(
            decile=decile,
            total_return=0,
            annualized_return=0,
            volatility=0,
            sharpe_ratio=0,
            max_drawdown=0,
            avg_score=0,
            num_periods=0
        )

    def _generate_period_data(
        self,
        results: BacktestResults
    ) -> List[Dict[str, Any]]:
        """Generate period-by-period data for charting."""
        # Group by period
        period_map: Dict[date, Dict[str, Any]] = {}

        for pr in results.period_results:
            if pr.period_start not in period_map:
                period_map[pr.period_start] = {
                    'date': pr.period_start.isoformat(),
                    'benchmark': pr.benchmark_return,
                }

            period_map[pr.period_start][f'd{pr.decile}'] = pr.period_return

        # Sort by date
        return [
            period_map[d]
            for d in sorted(period_map.keys())
        ]

    def _generate_cumulative_returns(
        self,
        results: BacktestResults
    ) -> Dict[str, List[Dict[str, Any]]]:
        """Generate cumulative return series for each decile."""
        cumulative: Dict[str, List[Dict[str, Any]]] = {}

        for decile in range(1, 11):
            returns = [
                (pr.period_start, pr.period_return)
                for pr in sorted(results.period_results, key=lambda x: x.period_start)
                if pr.decile == decile
            ]

            if not returns:
                continue

            series = []
            cum_value = 1.0

            for dt, ret in returns:
                cum_value *= (1 + ret)
                series.append({
                    'date': dt.isoformat(),
                    'value': cum_value,
                    'return': ret
                })

            cumulative[f'd{decile}'] = series

        # Add benchmark
        benchmark_returns = []
        seen_periods = set()

        for pr in sorted(results.period_results, key=lambda x: x.period_start):
            if pr.period_start not in seen_periods:
                seen_periods.add(pr.period_start)
                benchmark_returns.append((pr.period_start, pr.benchmark_return))

        cum_value = 1.0
        benchmark_series = []

        for dt, ret in benchmark_returns:
            cum_value *= (1 + ret)
            benchmark_series.append({
                'date': dt.isoformat(),
                'value': cum_value,
                'return': ret
            })

        cumulative['benchmark'] = benchmark_series

        return cumulative

    def _generate_sector_analysis(
        self,
        results: BacktestResults
    ) -> Dict[str, Any]:
        """Analyze performance by sector if available."""
        if not results.sector_results:
            return {}

        sector_data = {}

        for sector, decile_returns in results.sector_results.items():
            sector_metrics = {}

            for decile, total_return in decile_returns.items():
                sector_metrics[f'd{decile}'] = total_return

            sector_data[sector] = sector_metrics

        return sector_data

    def _generate_rolling_metrics(
        self,
        results: BacktestResults,
        window: int = 12  # 12 months
    ) -> Dict[str, List[Dict[str, Any]]]:
        """Generate rolling performance metrics."""
        rolling: Dict[str, List[Dict[str, Any]]] = {}

        for decile in [1, 5, 10]:  # Top, middle, bottom
            returns = [
                (pr.period_start, pr.period_return)
                for pr in sorted(results.period_results, key=lambda x: x.period_start)
                if pr.decile == decile
            ]

            if len(returns) < window:
                continue

            series = []

            for i in range(window, len(returns) + 1):
                window_returns = [r for _, r in returns[i-window:i]]
                window_date = returns[i-1][0]

                metrics = self.perf_calc.calculate_metrics(window_returns)

                series.append({
                    'date': window_date.isoformat(),
                    'rolling_return': metrics.total_return,
                    'rolling_sharpe': metrics.sharpe_ratio,
                    'rolling_volatility': metrics.annualized_volatility,
                })

            rolling[f'd{decile}'] = series

        return rolling

    def _run_statistical_tests(
        self,
        results: BacktestResults
    ) -> Dict[str, Any]:
        """Run statistical tests on backtest results."""
        tests = {}

        # Get returns for top and bottom deciles
        top_returns = [
            pr.period_return for pr in results.period_results
            if pr.decile == 1
        ]
        bottom_returns = [
            pr.period_return for pr in results.period_results
            if pr.decile == 10
        ]

        if not top_returns or not bottom_returns:
            return tests

        # T-test for difference in means
        try:
            t_stat, p_value = self._t_test(top_returns, bottom_returns)
            tests['t_test'] = {
                't_statistic': t_stat,
                'p_value': p_value,
                'significant_5pct': p_value < 0.05,
                'significant_1pct': p_value < 0.01,
            }
        except Exception as e:
            logger.warning(f"T-test failed: {e}")

        # Wilcoxon signed-rank test (non-parametric)
        try:
            # Pair returns by period
            period_pairs = self._pair_returns_by_period(
                results.period_results, 1, 10
            )
            if period_pairs:
                w_stat, w_p = self._wilcoxon_test(period_pairs)
                tests['wilcoxon_test'] = {
                    'w_statistic': w_stat,
                    'p_value': w_p,
                    'significant_5pct': w_p < 0.05,
                }
        except Exception as e:
            logger.warning(f"Wilcoxon test failed: {e}")

        # Binomial test for hit rate
        n_periods = len(set(pr.period_start for pr in results.period_results if pr.decile == 1))
        n_hits = int(results.hit_rate * n_periods)

        tests['binomial_test'] = {
            'n_periods': n_periods,
            'n_hits': n_hits,
            'hit_rate': results.hit_rate,
            'expected_if_random': 0.5,
            'excess_vs_random': results.hit_rate - 0.5,
        }

        return tests

    def _t_test(
        self,
        sample1: List[float],
        sample2: List[float]
    ) -> tuple:
        """Perform two-sample t-test."""
        import statistics

        n1, n2 = len(sample1), len(sample2)
        if n1 < 2 or n2 < 2:
            return 0.0, 1.0

        mean1 = statistics.mean(sample1)
        mean2 = statistics.mean(sample2)
        var1 = statistics.variance(sample1)
        var2 = statistics.variance(sample2)

        # Pooled standard error
        se = ((var1 / n1) + (var2 / n2)) ** 0.5

        if se == 0:
            return 0.0, 1.0

        t_stat = (mean1 - mean2) / se

        # Degrees of freedom (Welch's approximation)
        df = ((var1/n1 + var2/n2)**2 /
              ((var1/n1)**2/(n1-1) + (var2/n2)**2/(n2-1)))

        # Approximate p-value using normal distribution for large samples
        import math
        p_value = 2 * (1 - self._normal_cdf(abs(t_stat)))

        return t_stat, p_value

    def _wilcoxon_test(
        self,
        paired_returns: List[tuple]
    ) -> tuple:
        """Simplified Wilcoxon signed-rank test."""
        # Calculate differences
        diffs = [(top - bottom) for top, bottom in paired_returns if top != bottom]

        if not diffs:
            return 0.0, 1.0

        # Rank by absolute value
        ranked = sorted(enumerate(diffs), key=lambda x: abs(x[1]))

        # Calculate W statistic
        w_plus = sum(
            i + 1 for i, (_, d) in enumerate(ranked) if d > 0
        )
        w_minus = sum(
            i + 1 for i, (_, d) in enumerate(ranked) if d < 0
        )

        w_stat = min(w_plus, w_minus)
        n = len(diffs)

        # Approximate p-value for large n
        if n > 20:
            mean_w = n * (n + 1) / 4
            std_w = (n * (n + 1) * (2*n + 1) / 24) ** 0.5
            z = (w_stat - mean_w) / std_w if std_w > 0 else 0
            p_value = 2 * self._normal_cdf(-abs(z))
        else:
            # Small sample - return approximate p-value
            p_value = 0.05 if w_stat < n * (n+1) / 4 * 0.5 else 0.5

        return w_stat, p_value

    def _pair_returns_by_period(
        self,
        period_results: List[PeriodResult],
        decile1: int,
        decile2: int
    ) -> List[tuple]:
        """Pair returns from two deciles by period."""
        period_map: Dict[date, Dict[int, float]] = {}

        for pr in period_results:
            if pr.decile in (decile1, decile2):
                if pr.period_start not in period_map:
                    period_map[pr.period_start] = {}
                period_map[pr.period_start][pr.decile] = pr.period_return

        pairs = []
        for period_data in period_map.values():
            if decile1 in period_data and decile2 in period_data:
                pairs.append((period_data[decile1], period_data[decile2]))

        return pairs

    def _normal_cdf(self, x: float) -> float:
        """Approximate standard normal CDF."""
        import math
        return (1 + math.erf(x / math.sqrt(2))) / 2


@dataclass
class ChartData:
    """Data formatted for frontend charting."""
    labels: List[str]
    datasets: List[Dict[str, Any]]


class ChartDataGenerator:
    """Generate data formatted for frontend charts."""

    def generate_decile_bar_chart(
        self,
        summary: BacktestSummary
    ) -> ChartData:
        """Generate bar chart data for decile returns."""
        labels = [f"D{dp.decile}" for dp in summary.decile_performance]
        returns = [dp.annualized_return * 100 for dp in summary.decile_performance]

        # Color based on return
        colors = [
            '#10b981' if r > 0 else '#ef4444' for r in returns
        ]

        return ChartData(
            labels=labels,
            datasets=[{
                'label': 'Annualized Return (%)',
                'data': returns,
                'backgroundColor': colors,
            }]
        )

    def generate_cumulative_line_chart(
        self,
        cumulative_data: Dict[str, List[Dict[str, Any]]],
        deciles: List[int] = [1, 5, 10]
    ) -> ChartData:
        """Generate line chart data for cumulative returns."""
        # Get dates from first decile
        first_key = next(iter(cumulative_data.keys()))
        labels = [point['date'] for point in cumulative_data[first_key]]

        datasets = []

        colors = {
            1: '#10b981',  # Green for top
            5: '#6b7280',  # Gray for middle
            10: '#ef4444',  # Red for bottom
            'benchmark': '#3b82f6',  # Blue for benchmark
        }

        for decile in deciles:
            key = f'd{decile}'
            if key in cumulative_data:
                datasets.append({
                    'label': f'Decile {decile}',
                    'data': [point['value'] for point in cumulative_data[key]],
                    'borderColor': colors.get(decile, '#6b7280'),
                    'fill': False,
                })

        # Add benchmark
        if 'benchmark' in cumulative_data:
            datasets.append({
                'label': 'Benchmark',
                'data': [point['value'] for point in cumulative_data['benchmark']],
                'borderColor': colors['benchmark'],
                'borderDash': [5, 5],
                'fill': False,
            })

        return ChartData(labels=labels, datasets=datasets)

    def generate_spread_chart(
        self,
        period_data: List[Dict[str, Any]]
    ) -> ChartData:
        """Generate chart showing spread between top and bottom deciles."""
        labels = [p['date'] for p in period_data]

        spreads = []
        for p in period_data:
            d1 = p.get('d1', 0)
            d10 = p.get('d10', 0)
            spreads.append((d1 - d10) * 100)  # Convert to percentage

        return ChartData(
            labels=labels,
            datasets=[{
                'label': 'D1 - D10 Spread (%)',
                'data': spreads,
                'borderColor': '#8b5cf6',
                'backgroundColor': 'rgba(139, 92, 246, 0.1)',
                'fill': True,
            }]
        )
