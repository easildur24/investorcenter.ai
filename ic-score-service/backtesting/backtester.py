"""
IC Score Backtester - Core backtesting framework for IC Score methodology.

This module implements point-in-time score calculation and portfolio backtesting
to validate the predictive power of IC Scores.
"""

from dataclasses import dataclass, field
from datetime import date, timedelta
from enum import Enum
from typing import Dict, List, Optional, Tuple, Any
import logging
from decimal import Decimal
import asyncio

from sqlalchemy.ext.asyncio import AsyncSession
from sqlalchemy import text

logger = logging.getLogger(__name__)


class RebalanceFrequency(Enum):
    """Portfolio rebalance frequency options."""
    DAILY = "daily"
    WEEKLY = "weekly"
    MONTHLY = "monthly"
    QUARTERLY = "quarterly"


@dataclass
class BacktestConfig:
    """Configuration for backtest run."""
    start_date: date
    end_date: date
    rebalance_frequency: RebalanceFrequency = RebalanceFrequency.MONTHLY
    universe: str = "sp500"  # sp500, sp1500, all
    min_market_cap: Optional[float] = None
    max_market_cap: Optional[float] = None
    sectors: Optional[List[str]] = None
    exclude_financials: bool = False
    exclude_utilities: bool = False
    transaction_cost_bps: float = 10.0  # 10 basis points
    slippage_bps: float = 5.0  # 5 basis points
    use_smoothed_scores: bool = True
    benchmark: str = "SPY"


@dataclass
class PeriodResult:
    """Results for a single backtest period."""
    period_start: date
    period_end: date
    decile: int
    holdings: List[str]
    num_holdings: int
    period_return: float
    benchmark_return: float
    excess_return: float
    avg_score: float
    turnover: float = 0.0


@dataclass
class BacktestResults:
    """Aggregated backtest results."""
    config: BacktestConfig
    period_results: List[PeriodResult]

    # Summary statistics
    total_return_by_decile: Dict[int, float] = field(default_factory=dict)
    annualized_return_by_decile: Dict[int, float] = field(default_factory=dict)
    sharpe_ratio_by_decile: Dict[int, float] = field(default_factory=dict)
    max_drawdown_by_decile: Dict[int, float] = field(default_factory=dict)

    # Spread metrics
    top_bottom_spread: float = 0.0
    top_vs_benchmark: float = 0.0
    information_ratio: float = 0.0
    hit_rate: float = 0.0  # % of periods where top decile beats bottom

    # Decile monotonicity
    monotonicity_score: float = 0.0  # 1.0 = perfect ordering

    # By-sector results
    sector_results: Dict[str, Dict[int, float]] = field(default_factory=dict)


class ICScoreBacktester:
    """
    Backtest IC Score methodology against historical returns.

    Implements point-in-time score calculation to avoid look-ahead bias.
    Supports various rebalancing frequencies and universe filters.
    """

    def __init__(self, db_session: AsyncSession):
        self.db = db_session
        self._score_cache: Dict[Tuple[str, date], float] = {}
        self._price_cache: Dict[Tuple[str, date], float] = {}

    async def run_backtest(self, config: BacktestConfig) -> BacktestResults:
        """
        Run full historical backtest.

        Args:
            config: Backtest configuration

        Returns:
            BacktestResults with all metrics
        """
        logger.info(f"Starting backtest from {config.start_date} to {config.end_date}")

        period_results: List[PeriodResult] = []
        previous_holdings: Dict[int, List[str]] = {}

        # Generate rebalance periods
        periods = self._generate_periods(
            config.start_date,
            config.end_date,
            config.rebalance_frequency
        )

        logger.info(f"Generated {len(periods)} rebalance periods")

        for period_start, period_end in periods:
            try:
                # Get point-in-time scores
                scores = await self._calculate_point_in_time_scores(
                    period_start,
                    config
                )

                if not scores:
                    logger.warning(f"No scores available for {period_start}")
                    continue

                # Create decile portfolios
                deciles = self._create_decile_portfolios(scores)

                # Calculate returns for each decile
                benchmark_return = await self._get_benchmark_return(
                    config.benchmark,
                    period_start,
                    period_end
                )

                for decile, tickers in deciles.items():
                    portfolio_return = await self._calculate_portfolio_return(
                        tickers,
                        period_start,
                        period_end,
                        config.transaction_cost_bps,
                        config.slippage_bps
                    )

                    # Calculate turnover
                    prev = previous_holdings.get(decile, [])
                    turnover = self._calculate_turnover(prev, tickers)

                    # Average score for the decile
                    avg_score = sum(scores[t] for t in tickers) / len(tickers)

                    period_results.append(PeriodResult(
                        period_start=period_start,
                        period_end=period_end,
                        decile=decile,
                        holdings=tickers,
                        num_holdings=len(tickers),
                        period_return=portfolio_return,
                        benchmark_return=benchmark_return,
                        excess_return=portfolio_return - benchmark_return,
                        avg_score=avg_score,
                        turnover=turnover,
                    ))

                    previous_holdings[decile] = tickers

            except Exception as e:
                logger.error(f"Error processing period {period_start}: {e}")
                continue

        # Aggregate results
        return self._aggregate_results(config, period_results)

    def _generate_periods(
        self,
        start_date: date,
        end_date: date,
        frequency: RebalanceFrequency
    ) -> List[Tuple[date, date]]:
        """Generate rebalance periods based on frequency."""
        periods = []
        current = start_date

        while current < end_date:
            if frequency == RebalanceFrequency.DAILY:
                next_date = current + timedelta(days=1)
            elif frequency == RebalanceFrequency.WEEKLY:
                next_date = current + timedelta(weeks=1)
            elif frequency == RebalanceFrequency.MONTHLY:
                # Move to first of next month
                if current.month == 12:
                    next_date = date(current.year + 1, 1, 1)
                else:
                    next_date = date(current.year, current.month + 1, 1)
            elif frequency == RebalanceFrequency.QUARTERLY:
                # Move to first of next quarter
                quarter_month = ((current.month - 1) // 3 + 1) * 3 + 1
                if quarter_month > 12:
                    next_date = date(current.year + 1, quarter_month - 12, 1)
                else:
                    next_date = date(current.year, quarter_month, 1)
            else:
                raise ValueError(f"Unknown frequency: {frequency}")

            period_end = min(next_date - timedelta(days=1), end_date)
            periods.append((current, period_end))
            current = next_date

        return periods

    async def _calculate_point_in_time_scores(
        self,
        as_of_date: date,
        config: BacktestConfig
    ) -> Dict[str, float]:
        """
        Calculate IC Scores using only data available at as_of_date.

        This is critical for avoiding look-ahead bias.
        """
        scores = {}

        # Get universe of stocks as of that date
        stocks = await self._get_universe_as_of(as_of_date, config)

        logger.debug(f"Calculating scores for {len(stocks)} stocks as of {as_of_date}")

        for ticker in stocks:
            cache_key = (ticker, as_of_date)

            if cache_key in self._score_cache:
                scores[ticker] = self._score_cache[cache_key]
                continue

            try:
                score = await self._calculate_historical_score(ticker, as_of_date)
                if score is not None:
                    scores[ticker] = score
                    self._score_cache[cache_key] = score
            except Exception as e:
                logger.debug(f"Could not calculate score for {ticker} as of {as_of_date}: {e}")

        logger.debug(f"Calculated {len(scores)} valid scores")
        return scores

    async def _get_universe_as_of(
        self,
        as_of_date: date,
        config: BacktestConfig
    ) -> List[str]:
        """Get list of stocks in the universe as of a given date."""
        # Build query based on universe filter
        query = """
            SELECT DISTINCT c.ticker
            FROM companies c
            JOIN stock_prices sp ON c.ticker = sp.ticker
            WHERE sp.price_date <= :as_of_date
            AND sp.price_date >= :as_of_date - INTERVAL '7 days'
            AND c.is_active = true
        """
        params: Dict[str, Any] = {"as_of_date": as_of_date}

        # Apply universe filter
        if config.universe == "sp500":
            query += " AND c.is_sp500 = true"
        elif config.universe == "sp1500":
            query += " AND (c.is_sp500 = true OR c.is_sp400 = true OR c.is_sp600 = true)"

        # Apply market cap filter
        if config.min_market_cap:
            query += " AND c.market_cap >= :min_cap"
            params["min_cap"] = config.min_market_cap
        if config.max_market_cap:
            query += " AND c.market_cap <= :max_cap"
            params["max_cap"] = config.max_market_cap

        # Apply sector filter
        if config.exclude_financials:
            query += " AND c.sector != 'Financial Services'"
        if config.exclude_utilities:
            query += " AND c.sector != 'Utilities'"
        if config.sectors:
            query += " AND c.sector = ANY(:sectors)"
            params["sectors"] = config.sectors

        result = await self.db.execute(text(query), params)
        return [row[0] for row in result.fetchall()]

    async def _calculate_historical_score(
        self,
        ticker: str,
        as_of_date: date
    ) -> Optional[float]:
        """
        Calculate IC Score for a ticker using only data available as of the given date.

        This queries historical data tables with date filters to ensure
        no look-ahead bias.
        """
        # First, check if we have a stored historical score
        query = """
            SELECT overall_score
            FROM ic_scores
            WHERE ticker = :ticker
            AND calculated_at <= :as_of_date
            ORDER BY calculated_at DESC
            LIMIT 1
        """
        result = await self.db.execute(
            text(query),
            {"ticker": ticker, "as_of_date": as_of_date}
        )
        row = result.fetchone()

        if row and row[0] is not None:
            return float(row[0])

        # If no stored score, calculate from historical data
        # This is a simplified calculation - full implementation would
        # recalculate all factors using point-in-time data
        return await self._calculate_score_from_historical_data(ticker, as_of_date)

    async def _calculate_score_from_historical_data(
        self,
        ticker: str,
        as_of_date: date
    ) -> Optional[float]:
        """Calculate score from historical fundamental and price data."""
        # Get historical fundamentals
        fundamentals = await self._get_historical_fundamentals(ticker, as_of_date)
        if not fundamentals:
            return None

        # Get historical price data for momentum
        prices = await self._get_historical_prices(ticker, as_of_date)
        if not prices:
            return None

        # Simplified score calculation
        # Full implementation would use all factor calculators
        growth_score = self._score_growth(fundamentals)
        value_score = self._score_value(fundamentals)
        profitability_score = self._score_profitability(fundamentals)
        momentum_score = self._score_momentum(prices)

        # Weighted average
        weights = {
            'growth': 0.12,
            'value': 0.12,
            'profitability': 0.12,
            'momentum': 0.10,
        }

        total_weight = sum(weights.values())
        overall = (
            growth_score * weights['growth'] +
            value_score * weights['value'] +
            profitability_score * weights['profitability'] +
            momentum_score * weights['momentum']
        ) / total_weight * 100

        return max(0, min(100, overall))

    async def _get_historical_fundamentals(
        self,
        ticker: str,
        as_of_date: date
    ) -> Optional[Dict[str, Any]]:
        """Get fundamental data as of a given date."""
        query = """
            SELECT
                revenue_growth_yoy,
                eps_growth_yoy,
                net_margin,
                roe,
                pe_ratio,
                ps_ratio,
                pb_ratio
            FROM ttm_financials
            WHERE ticker = :ticker
            AND calculation_date <= :as_of_date
            ORDER BY calculation_date DESC
            LIMIT 1
        """
        result = await self.db.execute(
            text(query),
            {"ticker": ticker, "as_of_date": as_of_date}
        )
        row = result.fetchone()

        if not row:
            return None

        return {
            'revenue_growth_yoy': float(row[0]) if row[0] else None,
            'eps_growth_yoy': float(row[1]) if row[1] else None,
            'net_margin': float(row[2]) if row[2] else None,
            'roe': float(row[3]) if row[3] else None,
            'pe_ratio': float(row[4]) if row[4] else None,
            'ps_ratio': float(row[5]) if row[5] else None,
            'pb_ratio': float(row[6]) if row[6] else None,
        }

    async def _get_historical_prices(
        self,
        ticker: str,
        as_of_date: date
    ) -> Optional[List[float]]:
        """Get price history for momentum calculation."""
        query = """
            SELECT close_price
            FROM stock_prices
            WHERE ticker = :ticker
            AND price_date <= :as_of_date
            ORDER BY price_date DESC
            LIMIT 252
        """
        result = await self.db.execute(
            text(query),
            {"ticker": ticker, "as_of_date": as_of_date}
        )
        rows = result.fetchall()

        if len(rows) < 20:
            return None

        return [float(row[0]) for row in rows]

    def _score_growth(self, data: Dict[str, Any]) -> float:
        """Score growth metrics (0-1)."""
        score = 0.5

        rev_growth = data.get('revenue_growth_yoy')
        if rev_growth is not None:
            # 0% = 0.5, 20% = 0.8, -20% = 0.2
            score = 0.5 + (rev_growth / 100) * 1.5

        eps_growth = data.get('eps_growth_yoy')
        if eps_growth is not None:
            eps_score = 0.5 + (eps_growth / 100) * 1.5
            score = (score + eps_score) / 2

        return max(0, min(1, score))

    def _score_value(self, data: Dict[str, Any]) -> float:
        """Score valuation metrics (0-1)."""
        scores = []

        pe = data.get('pe_ratio')
        if pe and pe > 0:
            # PE < 15 = good, PE > 30 = bad
            pe_score = 1 - (pe - 15) / 30
            scores.append(max(0, min(1, pe_score)))

        ps = data.get('ps_ratio')
        if ps and ps > 0:
            ps_score = 1 - (ps - 2) / 8
            scores.append(max(0, min(1, ps_score)))

        pb = data.get('pb_ratio')
        if pb and pb > 0:
            pb_score = 1 - (pb - 1) / 5
            scores.append(max(0, min(1, pb_score)))

        return sum(scores) / len(scores) if scores else 0.5

    def _score_profitability(self, data: Dict[str, Any]) -> float:
        """Score profitability metrics (0-1)."""
        scores = []

        net_margin = data.get('net_margin')
        if net_margin is not None:
            # 10% margin = 0.5, 20% = 0.75
            margin_score = 0.25 + net_margin / 40
            scores.append(max(0, min(1, margin_score)))

        roe = data.get('roe')
        if roe is not None:
            # 15% ROE = 0.5, 25% = 0.75
            roe_score = roe / 50
            scores.append(max(0, min(1, roe_score)))

        return sum(scores) / len(scores) if scores else 0.5

    def _score_momentum(self, prices: List[float]) -> float:
        """Score price momentum (0-1)."""
        if len(prices) < 252:
            return 0.5

        # 12-month return minus 1-month return (skip recent momentum)
        current = prices[0]
        month_ago = prices[20] if len(prices) > 20 else prices[-1]
        year_ago = prices[251]

        if year_ago <= 0 or month_ago <= 0:
            return 0.5

        # 12-1 momentum
        twelve_month = (current / year_ago) - 1
        one_month = (current / month_ago) - 1
        momentum = twelve_month - one_month

        # Scale: -20% = 0, 0% = 0.5, +20% = 1.0
        return max(0, min(1, 0.5 + momentum * 2.5))

    def _create_decile_portfolios(
        self,
        scores: Dict[str, float]
    ) -> Dict[int, List[str]]:
        """Create 10 portfolios based on score deciles."""
        if not scores:
            return {}

        # Sort by score descending
        sorted_stocks = sorted(scores.items(), key=lambda x: x[1], reverse=True)

        n = len(sorted_stocks)
        decile_size = n // 10

        deciles: Dict[int, List[str]] = {i: [] for i in range(1, 11)}

        for i, (ticker, _) in enumerate(sorted_stocks):
            decile = min(10, (i // decile_size) + 1) if decile_size > 0 else 1
            deciles[decile].append(ticker)

        return deciles

    async def _calculate_portfolio_return(
        self,
        tickers: List[str],
        start_date: date,
        end_date: date,
        transaction_cost_bps: float,
        slippage_bps: float
    ) -> float:
        """Calculate equal-weighted portfolio return for a period."""
        if not tickers:
            return 0.0

        returns = []

        for ticker in tickers:
            ret = await self._get_stock_return(ticker, start_date, end_date)
            if ret is not None:
                returns.append(ret)

        if not returns:
            return 0.0

        # Equal-weighted return
        gross_return = sum(returns) / len(returns)

        # Apply transaction costs (assume full turnover)
        cost_adjustment = (transaction_cost_bps + slippage_bps) / 10000

        return gross_return - cost_adjustment

    async def _get_stock_return(
        self,
        ticker: str,
        start_date: date,
        end_date: date
    ) -> Optional[float]:
        """Get total return for a stock over a period."""
        cache_key_start = (ticker, start_date)
        cache_key_end = (ticker, end_date)

        # Try to get prices from cache or database
        start_price = self._price_cache.get(cache_key_start)
        end_price = self._price_cache.get(cache_key_end)

        if start_price is None:
            start_price = await self._get_price_as_of(ticker, start_date)
            if start_price:
                self._price_cache[cache_key_start] = start_price

        if end_price is None:
            end_price = await self._get_price_as_of(ticker, end_date)
            if end_price:
                self._price_cache[cache_key_end] = end_price

        if start_price and end_price and start_price > 0:
            return (end_price - start_price) / start_price

        return None

    async def _get_price_as_of(
        self,
        ticker: str,
        as_of_date: date
    ) -> Optional[float]:
        """Get closing price as of a date (or nearest prior date)."""
        query = """
            SELECT close_price
            FROM stock_prices
            WHERE ticker = :ticker
            AND price_date <= :as_of_date
            ORDER BY price_date DESC
            LIMIT 1
        """
        result = await self.db.execute(
            text(query),
            {"ticker": ticker, "as_of_date": as_of_date}
        )
        row = result.fetchone()

        return float(row[0]) if row else None

    async def _get_benchmark_return(
        self,
        benchmark: str,
        start_date: date,
        end_date: date
    ) -> float:
        """Get benchmark return for a period."""
        ret = await self._get_stock_return(benchmark, start_date, end_date)
        return ret if ret is not None else 0.0

    def _calculate_turnover(
        self,
        previous: List[str],
        current: List[str]
    ) -> float:
        """Calculate portfolio turnover."""
        if not previous:
            return 1.0

        prev_set = set(previous)
        curr_set = set(current)

        unchanged = len(prev_set & curr_set)
        total = max(len(prev_set), len(curr_set))

        if total == 0:
            return 0.0

        return 1 - (unchanged / total)

    def _aggregate_results(
        self,
        config: BacktestConfig,
        period_results: List[PeriodResult]
    ) -> BacktestResults:
        """Aggregate period results into summary statistics."""
        results = BacktestResults(
            config=config,
            period_results=period_results
        )

        # Group by decile
        decile_returns: Dict[int, List[float]] = {i: [] for i in range(1, 11)}

        for pr in period_results:
            decile_returns[pr.decile].append(pr.period_return)

        # Calculate metrics for each decile
        for decile in range(1, 11):
            returns = decile_returns[decile]
            if not returns:
                continue

            # Total return (compounded)
            total = 1.0
            for r in returns:
                total *= (1 + r)
            results.total_return_by_decile[decile] = total - 1

            # Annualized return
            years = (config.end_date - config.start_date).days / 365.25
            if years > 0:
                results.annualized_return_by_decile[decile] = (
                    (total ** (1 / years)) - 1
                )

            # Sharpe ratio (simplified)
            import statistics
            if len(returns) > 1:
                mean_ret = statistics.mean(returns)
                std_ret = statistics.stdev(returns)
                if std_ret > 0:
                    # Annualize based on frequency
                    periods_per_year = {
                        RebalanceFrequency.DAILY: 252,
                        RebalanceFrequency.WEEKLY: 52,
                        RebalanceFrequency.MONTHLY: 12,
                        RebalanceFrequency.QUARTERLY: 4,
                    }
                    periods = periods_per_year.get(config.rebalance_frequency, 12)
                    results.sharpe_ratio_by_decile[decile] = (
                        mean_ret * (periods ** 0.5) / std_ret
                    )

            # Max drawdown
            results.max_drawdown_by_decile[decile] = self._calculate_max_drawdown(returns)

        # Top vs bottom spread
        top_return = results.annualized_return_by_decile.get(1, 0)
        bottom_return = results.annualized_return_by_decile.get(10, 0)
        results.top_bottom_spread = top_return - bottom_return

        # Top vs benchmark
        benchmark_periods = [
            pr.benchmark_return for pr in period_results if pr.decile == 1
        ]
        if benchmark_periods:
            benchmark_total = 1.0
            for r in benchmark_periods:
                benchmark_total *= (1 + r)
            years = (config.end_date - config.start_date).days / 365.25
            benchmark_ann = (benchmark_total ** (1 / years)) - 1 if years > 0 else 0
            results.top_vs_benchmark = top_return - benchmark_ann

        # Hit rate (% of periods where D1 > D10)
        hit_count = 0
        total_periods = 0
        period_map: Dict[date, Dict[int, float]] = {}

        for pr in period_results:
            if pr.period_start not in period_map:
                period_map[pr.period_start] = {}
            period_map[pr.period_start][pr.decile] = pr.period_return

        for period_data in period_map.values():
            if 1 in period_data and 10 in period_data:
                total_periods += 1
                if period_data[1] > period_data[10]:
                    hit_count += 1

        results.hit_rate = hit_count / total_periods if total_periods > 0 else 0

        # Monotonicity score
        results.monotonicity_score = self._calculate_monotonicity(
            results.annualized_return_by_decile
        )

        return results

    def _calculate_max_drawdown(self, returns: List[float]) -> float:
        """Calculate maximum drawdown from a series of returns."""
        if not returns:
            return 0.0

        cumulative = 1.0
        peak = 1.0
        max_dd = 0.0

        for r in returns:
            cumulative *= (1 + r)
            if cumulative > peak:
                peak = cumulative
            dd = (peak - cumulative) / peak
            if dd > max_dd:
                max_dd = dd

        return max_dd

    def _calculate_monotonicity(
        self,
        decile_returns: Dict[int, float]
    ) -> float:
        """
        Calculate how well returns decrease monotonically across deciles.
        1.0 = perfect (D1 > D2 > ... > D10)
        0.0 = random
        """
        if len(decile_returns) < 2:
            return 0.0

        correct_orderings = 0
        total_pairs = 0

        deciles = sorted(decile_returns.keys())

        for i, d1 in enumerate(deciles):
            for d2 in deciles[i+1:]:
                total_pairs += 1
                # Lower decile should have higher return
                if decile_returns[d1] > decile_returns[d2]:
                    correct_orderings += 1

        return correct_orderings / total_pairs if total_pairs > 0 else 0
