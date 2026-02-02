"""
Performance metrics for IC Score backtesting.

This module provides comprehensive performance analysis including
risk-adjusted returns, drawdowns, and statistical measures.
"""

from dataclasses import dataclass, field
from datetime import date
from typing import Dict, List, Optional, Tuple
import math
import statistics
import logging

logger = logging.getLogger(__name__)


@dataclass
class BacktestMetrics:
    """Comprehensive backtest performance metrics."""
    # Return metrics
    total_return: float = 0.0
    annualized_return: float = 0.0
    cagr: float = 0.0

    # Risk metrics
    volatility: float = 0.0
    annualized_volatility: float = 0.0
    downside_deviation: float = 0.0
    max_drawdown: float = 0.0
    avg_drawdown: float = 0.0
    max_drawdown_duration: int = 0  # days

    # Risk-adjusted metrics
    sharpe_ratio: float = 0.0
    sortino_ratio: float = 0.0
    calmar_ratio: float = 0.0
    information_ratio: float = 0.0
    treynor_ratio: float = 0.0

    # Benchmark comparison
    alpha: float = 0.0
    beta: float = 0.0
    correlation: float = 0.0
    tracking_error: float = 0.0
    excess_return: float = 0.0

    # Win/loss statistics
    win_rate: float = 0.0
    avg_win: float = 0.0
    avg_loss: float = 0.0
    profit_factor: float = 0.0
    payoff_ratio: float = 0.0
    best_period: float = 0.0
    worst_period: float = 0.0

    # Consistency metrics
    positive_periods: int = 0
    negative_periods: int = 0
    consecutive_wins: int = 0
    consecutive_losses: int = 0
    skewness: float = 0.0
    kurtosis: float = 0.0


@dataclass
class DrawdownInfo:
    """Information about a drawdown period."""
    start_date: date
    end_date: Optional[date]
    trough_date: date
    peak_value: float
    trough_value: float
    drawdown: float
    duration_days: int
    recovery_days: Optional[int]
    recovered: bool


class PerformanceCalculator:
    """
    Calculate comprehensive performance metrics for backtest results.

    Supports various risk-free rate assumptions and benchmark comparisons.
    """

    DEFAULT_RISK_FREE_RATE = 0.02  # 2% annual

    def __init__(
        self,
        risk_free_rate: float = DEFAULT_RISK_FREE_RATE,
        periods_per_year: int = 12  # Monthly rebalancing
    ):
        self.risk_free_rate = risk_free_rate
        self.periods_per_year = periods_per_year

    def calculate_metrics(
        self,
        returns: List[float],
        benchmark_returns: Optional[List[float]] = None,
        dates: Optional[List[date]] = None
    ) -> BacktestMetrics:
        """
        Calculate all performance metrics.

        Args:
            returns: List of period returns
            benchmark_returns: Optional benchmark returns for comparison
            dates: Optional dates for each return

        Returns:
            BacktestMetrics object with all calculations
        """
        if not returns:
            return BacktestMetrics()

        metrics = BacktestMetrics()

        # Basic return metrics
        metrics.total_return = self._calculate_total_return(returns)
        metrics.cagr = self._calculate_cagr(returns)
        metrics.annualized_return = metrics.cagr

        # Risk metrics
        metrics.volatility = self._calculate_volatility(returns)
        metrics.annualized_volatility = metrics.volatility * math.sqrt(self.periods_per_year)
        metrics.downside_deviation = self._calculate_downside_deviation(returns)

        # Drawdown analysis
        drawdowns = self._calculate_drawdown_series(returns)
        metrics.max_drawdown = max(drawdowns) if drawdowns else 0
        metrics.avg_drawdown = statistics.mean(drawdowns) if drawdowns else 0

        # Risk-adjusted returns
        metrics.sharpe_ratio = self._calculate_sharpe_ratio(returns)
        metrics.sortino_ratio = self._calculate_sortino_ratio(returns)
        metrics.calmar_ratio = self._calculate_calmar_ratio(returns, metrics.max_drawdown)

        # Win/loss stats
        wins = [r for r in returns if r > 0]
        losses = [r for r in returns if r < 0]

        metrics.positive_periods = len(wins)
        metrics.negative_periods = len(losses)
        metrics.win_rate = len(wins) / len(returns) if returns else 0

        metrics.avg_win = statistics.mean(wins) if wins else 0
        metrics.avg_loss = statistics.mean(losses) if losses else 0

        metrics.best_period = max(returns)
        metrics.worst_period = min(returns)

        total_wins = sum(wins) if wins else 0
        total_losses = abs(sum(losses)) if losses else 0
        metrics.profit_factor = total_wins / total_losses if total_losses > 0 else float('inf')

        metrics.payoff_ratio = abs(metrics.avg_win / metrics.avg_loss) if metrics.avg_loss != 0 else float('inf')

        # Consecutive wins/losses
        metrics.consecutive_wins, metrics.consecutive_losses = self._calculate_streaks(returns)

        # Higher moments
        if len(returns) > 2:
            metrics.skewness = self._calculate_skewness(returns)
            metrics.kurtosis = self._calculate_kurtosis(returns)

        # Benchmark comparison
        if benchmark_returns and len(benchmark_returns) == len(returns):
            metrics.alpha, metrics.beta = self._calculate_alpha_beta(
                returns, benchmark_returns
            )
            metrics.correlation = self._calculate_correlation(
                returns, benchmark_returns
            )
            metrics.tracking_error = self._calculate_tracking_error(
                returns, benchmark_returns
            )
            metrics.information_ratio = self._calculate_information_ratio(
                returns, benchmark_returns
            )

            benchmark_total = self._calculate_total_return(benchmark_returns)
            metrics.excess_return = metrics.total_return - benchmark_total

            if metrics.beta != 0:
                rf_period = self.risk_free_rate / self.periods_per_year
                excess_returns = [r - rf_period for r in returns]
                avg_excess = statistics.mean(excess_returns)
                metrics.treynor_ratio = avg_excess * self.periods_per_year / metrics.beta

        return metrics

    def _calculate_total_return(self, returns: List[float]) -> float:
        """Calculate compounded total return."""
        total = 1.0
        for r in returns:
            total *= (1 + r)
        return total - 1

    def _calculate_cagr(self, returns: List[float]) -> float:
        """Calculate Compound Annual Growth Rate."""
        if not returns:
            return 0.0

        total = 1.0
        for r in returns:
            total *= (1 + r)

        years = len(returns) / self.periods_per_year
        if years <= 0 or total <= 0:
            return 0.0

        return (total ** (1 / years)) - 1

    def _calculate_volatility(self, returns: List[float]) -> float:
        """Calculate standard deviation of returns."""
        if len(returns) < 2:
            return 0.0
        return statistics.stdev(returns)

    def _calculate_downside_deviation(
        self,
        returns: List[float],
        target: float = 0.0
    ) -> float:
        """Calculate downside deviation (semi-deviation below target)."""
        downside_returns = [
            (r - target) ** 2 for r in returns if r < target
        ]
        if not downside_returns:
            return 0.0
        return math.sqrt(sum(downside_returns) / len(downside_returns))

    def _calculate_drawdown_series(self, returns: List[float]) -> List[float]:
        """Calculate drawdown at each point in time."""
        drawdowns = []
        peak = 1.0
        cumulative = 1.0

        for r in returns:
            cumulative *= (1 + r)
            if cumulative > peak:
                peak = cumulative
            dd = (peak - cumulative) / peak if peak > 0 else 0
            drawdowns.append(dd)

        return drawdowns

    def analyze_drawdowns(
        self,
        returns: List[float],
        dates: List[date]
    ) -> List[DrawdownInfo]:
        """Analyze all drawdown periods."""
        if len(returns) != len(dates):
            raise ValueError("Returns and dates must have same length")

        drawdowns: List[DrawdownInfo] = []
        peak = 1.0
        cumulative = 1.0
        peak_date = dates[0]
        in_drawdown = False
        current_dd: Optional[DrawdownInfo] = None

        for i, (r, d) in enumerate(zip(returns, dates)):
            cumulative *= (1 + r)

            if cumulative > peak:
                # New peak - end any current drawdown
                if current_dd and in_drawdown:
                    current_dd.end_date = d
                    current_dd.recovered = True
                    current_dd.recovery_days = (d - current_dd.trough_date).days
                    drawdowns.append(current_dd)

                peak = cumulative
                peak_date = d
                in_drawdown = False
                current_dd = None

            elif cumulative < peak:
                dd = (peak - cumulative) / peak

                if not in_drawdown:
                    # Start new drawdown
                    in_drawdown = True
                    current_dd = DrawdownInfo(
                        start_date=peak_date,
                        end_date=None,
                        trough_date=d,
                        peak_value=peak,
                        trough_value=cumulative,
                        drawdown=dd,
                        duration_days=(d - peak_date).days,
                        recovery_days=None,
                        recovered=False
                    )
                elif current_dd and cumulative < current_dd.trough_value:
                    # Deeper trough
                    current_dd.trough_date = d
                    current_dd.trough_value = cumulative
                    current_dd.drawdown = dd
                    current_dd.duration_days = (d - current_dd.start_date).days

        # Handle ongoing drawdown at end
        if current_dd and in_drawdown:
            current_dd.end_date = dates[-1]
            drawdowns.append(current_dd)

        return drawdowns

    def _calculate_sharpe_ratio(self, returns: List[float]) -> float:
        """Calculate Sharpe Ratio."""
        if len(returns) < 2:
            return 0.0

        rf_period = self.risk_free_rate / self.periods_per_year
        excess_returns = [r - rf_period for r in returns]

        avg_excess = statistics.mean(excess_returns)
        std = statistics.stdev(excess_returns) if len(excess_returns) > 1 else 0

        if std == 0:
            return 0.0

        return avg_excess * math.sqrt(self.periods_per_year) / std

    def _calculate_sortino_ratio(self, returns: List[float]) -> float:
        """Calculate Sortino Ratio (uses downside deviation)."""
        if len(returns) < 2:
            return 0.0

        rf_period = self.risk_free_rate / self.periods_per_year
        excess_returns = [r - rf_period for r in returns]

        avg_excess = statistics.mean(excess_returns)
        downside = self._calculate_downside_deviation(excess_returns)

        if downside == 0:
            return 0.0

        return avg_excess * math.sqrt(self.periods_per_year) / downside

    def _calculate_calmar_ratio(
        self,
        returns: List[float],
        max_drawdown: float
    ) -> float:
        """Calculate Calmar Ratio (CAGR / Max Drawdown)."""
        if max_drawdown == 0:
            return 0.0

        cagr = self._calculate_cagr(returns)
        return cagr / max_drawdown

    def _calculate_alpha_beta(
        self,
        returns: List[float],
        benchmark_returns: List[float]
    ) -> Tuple[float, float]:
        """Calculate alpha and beta vs benchmark."""
        if len(returns) < 2 or len(returns) != len(benchmark_returns):
            return 0.0, 1.0

        # Calculate beta using covariance / variance
        mean_r = statistics.mean(returns)
        mean_b = statistics.mean(benchmark_returns)

        cov = sum(
            (r - mean_r) * (b - mean_b)
            for r, b in zip(returns, benchmark_returns)
        ) / len(returns)

        var_b = sum(
            (b - mean_b) ** 2 for b in benchmark_returns
        ) / len(benchmark_returns)

        beta = cov / var_b if var_b > 0 else 1.0

        # Alpha = portfolio return - (rf + beta * (market - rf))
        rf_period = self.risk_free_rate / self.periods_per_year
        alpha = mean_r - (rf_period + beta * (mean_b - rf_period))

        # Annualize alpha
        alpha_annual = alpha * self.periods_per_year

        return alpha_annual, beta

    def _calculate_correlation(
        self,
        returns: List[float],
        benchmark_returns: List[float]
    ) -> float:
        """Calculate correlation with benchmark."""
        if len(returns) < 2 or len(returns) != len(benchmark_returns):
            return 0.0

        mean_r = statistics.mean(returns)
        mean_b = statistics.mean(benchmark_returns)

        cov = sum(
            (r - mean_r) * (b - mean_b)
            for r, b in zip(returns, benchmark_returns)
        ) / len(returns)

        std_r = statistics.stdev(returns)
        std_b = statistics.stdev(benchmark_returns)

        if std_r == 0 or std_b == 0:
            return 0.0

        return cov / (std_r * std_b)

    def _calculate_tracking_error(
        self,
        returns: List[float],
        benchmark_returns: List[float]
    ) -> float:
        """Calculate tracking error (std of excess returns)."""
        if len(returns) != len(benchmark_returns):
            return 0.0

        excess = [r - b for r, b in zip(returns, benchmark_returns)]
        if len(excess) < 2:
            return 0.0

        return statistics.stdev(excess) * math.sqrt(self.periods_per_year)

    def _calculate_information_ratio(
        self,
        returns: List[float],
        benchmark_returns: List[float]
    ) -> float:
        """Calculate Information Ratio (excess return / tracking error)."""
        tracking_error = self._calculate_tracking_error(
            returns, benchmark_returns
        )
        if tracking_error == 0:
            return 0.0

        excess = [r - b for r, b in zip(returns, benchmark_returns)]
        avg_excess = statistics.mean(excess) * self.periods_per_year

        return avg_excess / tracking_error

    def _calculate_streaks(
        self,
        returns: List[float]
    ) -> Tuple[int, int]:
        """Calculate max consecutive wins and losses."""
        max_wins = 0
        max_losses = 0
        current_wins = 0
        current_losses = 0

        for r in returns:
            if r > 0:
                current_wins += 1
                current_losses = 0
                max_wins = max(max_wins, current_wins)
            elif r < 0:
                current_losses += 1
                current_wins = 0
                max_losses = max(max_losses, current_losses)
            else:
                current_wins = 0
                current_losses = 0

        return max_wins, max_losses

    def _calculate_skewness(self, returns: List[float]) -> float:
        """Calculate skewness of returns distribution."""
        if len(returns) < 3:
            return 0.0

        mean = statistics.mean(returns)
        std = statistics.stdev(returns)

        if std == 0:
            return 0.0

        n = len(returns)
        skew = sum((r - mean) ** 3 for r in returns) / n
        return skew / (std ** 3)

    def _calculate_kurtosis(self, returns: List[float]) -> float:
        """Calculate excess kurtosis of returns distribution."""
        if len(returns) < 4:
            return 0.0

        mean = statistics.mean(returns)
        std = statistics.stdev(returns)

        if std == 0:
            return 0.0

        n = len(returns)
        kurt = sum((r - mean) ** 4 for r in returns) / n
        return (kurt / (std ** 4)) - 3  # Excess kurtosis


class DecileMetricsCalculator:
    """Calculate and compare metrics across deciles."""

    def __init__(self, performance_calc: Optional[PerformanceCalculator] = None):
        self.perf_calc = performance_calc or PerformanceCalculator()

    def calculate_decile_metrics(
        self,
        decile_returns: Dict[int, List[float]],
        benchmark_returns: Optional[List[float]] = None
    ) -> Dict[int, BacktestMetrics]:
        """Calculate metrics for each decile."""
        results = {}

        for decile, returns in decile_returns.items():
            results[decile] = self.perf_calc.calculate_metrics(
                returns, benchmark_returns
            )

        return results

    def calculate_spread_metrics(
        self,
        decile_metrics: Dict[int, BacktestMetrics]
    ) -> Dict[str, float]:
        """Calculate spread between top and bottom deciles."""
        if 1 not in decile_metrics or 10 not in decile_metrics:
            return {}

        top = decile_metrics[1]
        bottom = decile_metrics[10]

        return {
            'return_spread': top.total_return - bottom.total_return,
            'cagr_spread': top.cagr - bottom.cagr,
            'sharpe_spread': top.sharpe_ratio - bottom.sharpe_ratio,
            'volatility_spread': top.annualized_volatility - bottom.annualized_volatility,
            'max_dd_spread': top.max_drawdown - bottom.max_drawdown,
        }

    def calculate_monotonicity_metrics(
        self,
        decile_metrics: Dict[int, BacktestMetrics]
    ) -> Dict[str, float]:
        """Calculate how well metrics decrease monotonically across deciles."""
        metrics_to_check = ['cagr', 'sharpe_ratio', 'total_return']
        results = {}

        for metric_name in metrics_to_check:
            correct = 0
            total = 0

            deciles = sorted(decile_metrics.keys())
            for i, d1 in enumerate(deciles):
                for d2 in deciles[i+1:]:
                    total += 1
                    v1 = getattr(decile_metrics[d1], metric_name, 0)
                    v2 = getattr(decile_metrics[d2], metric_name, 0)
                    if v1 > v2:
                        correct += 1

            results[f'{metric_name}_monotonicity'] = correct / total if total > 0 else 0

        return results
