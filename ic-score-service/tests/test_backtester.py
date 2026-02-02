"""Unit tests for IC Score Backtester."""

import pytest
from unittest.mock import AsyncMock, MagicMock, patch
from datetime import date, timedelta
from typing import Dict, List

import sys
from pathlib import Path
sys.path.insert(0, str(Path(__file__).parent.parent))

from backtesting.backtester import (
    ICScoreBacktester,
    BacktestConfig,
    BacktestResults,
    PeriodResult,
    RebalanceFrequency,
)


class TestBacktestConfig:
    """Test BacktestConfig dataclass."""

    def test_default_config(self):
        """Test default configuration values."""
        config = BacktestConfig(
            start_date=date(2020, 1, 1),
            end_date=date(2024, 1, 1)
        )

        assert config.rebalance_frequency == RebalanceFrequency.MONTHLY
        assert config.universe == "sp500"
        assert config.benchmark == "SPY"
        assert config.transaction_cost_bps == 10.0
        assert config.slippage_bps == 5.0
        assert config.use_smoothed_scores is True
        assert config.exclude_financials is False
        assert config.exclude_utilities is False

    def test_custom_config(self):
        """Test custom configuration values."""
        config = BacktestConfig(
            start_date=date(2019, 1, 1),
            end_date=date(2024, 6, 1),
            rebalance_frequency=RebalanceFrequency.QUARTERLY,
            universe="sp1500",
            benchmark="QQQ",
            transaction_cost_bps=15.0,
            slippage_bps=10.0,
            exclude_financials=True,
            exclude_utilities=True,
            use_smoothed_scores=False,
        )

        assert config.rebalance_frequency == RebalanceFrequency.QUARTERLY
        assert config.universe == "sp1500"
        assert config.benchmark == "QQQ"
        assert config.transaction_cost_bps == 15.0
        assert config.exclude_financials is True


class TestICScoreBacktester:
    """Test ICScoreBacktester class."""

    @pytest.fixture
    def mock_session(self):
        """Create a mock database session."""
        session = AsyncMock()
        return session

    @pytest.fixture
    def backtester(self, mock_session):
        """Create a backtester instance with mocked session."""
        return ICScoreBacktester(mock_session)

    @pytest.fixture
    def config(self):
        """Create a test configuration."""
        return BacktestConfig(
            start_date=date(2023, 1, 1),
            end_date=date(2023, 12, 31),
            rebalance_frequency=RebalanceFrequency.MONTHLY,
        )

    # ==================
    # Period Generation Tests
    # ==================

    def test_generate_periods_monthly(self, backtester):
        """Test generating monthly rebalance periods."""
        periods = backtester._generate_periods(
            start_date=date(2023, 1, 1),
            end_date=date(2023, 6, 30),
            frequency=RebalanceFrequency.MONTHLY
        )

        assert len(periods) == 6
        assert periods[0] == (date(2023, 1, 1), date(2023, 1, 31))
        assert periods[5] == (date(2023, 6, 1), date(2023, 6, 30))

    def test_generate_periods_weekly(self, backtester):
        """Test generating weekly rebalance periods."""
        periods = backtester._generate_periods(
            start_date=date(2023, 1, 1),
            end_date=date(2023, 1, 31),
            frequency=RebalanceFrequency.WEEKLY
        )

        # Should have 4-5 weekly periods in January
        assert len(periods) >= 4

    def test_generate_periods_quarterly(self, backtester):
        """Test generating quarterly rebalance periods."""
        periods = backtester._generate_periods(
            start_date=date(2023, 1, 1),
            end_date=date(2023, 12, 31),
            frequency=RebalanceFrequency.QUARTERLY
        )

        # Should have 4 quarterly periods
        assert len(periods) == 4

    def test_generate_periods_daily(self, backtester):
        """Test generating daily rebalance periods."""
        periods = backtester._generate_periods(
            start_date=date(2023, 1, 1),
            end_date=date(2023, 1, 10),
            frequency=RebalanceFrequency.DAILY
        )

        assert len(periods) == 10

    # ==================
    # Decile Portfolio Tests
    # ==================

    def test_create_decile_portfolios(self, backtester):
        """Test creating decile portfolios from scores."""
        # Create 100 stocks with scores
        scores = {f"TICK{i:03d}": 100 - i for i in range(100)}

        deciles = backtester._create_decile_portfolios(scores)

        assert len(deciles) == 10
        # Each decile should have 10 stocks
        for i in range(1, 11):
            assert len(deciles[i]) == 10

        # Top decile should have highest scored stocks
        assert "TICK000" in deciles[1]  # Score 100
        # Bottom decile should have lowest scored stocks
        assert "TICK099" in deciles[10]  # Score 1

    def test_create_decile_portfolios_small_universe(self, backtester):
        """Test decile creation with small universe."""
        scores = {"AAPL": 90, "MSFT": 80, "GOOGL": 70, "AMZN": 60, "META": 50}

        deciles = backtester._create_decile_portfolios(scores)

        # With 5 stocks, they get distributed across deciles
        total_stocks = sum(len(d) for d in deciles.values())
        assert total_stocks == 5

    def test_create_decile_portfolios_empty(self, backtester):
        """Test decile creation with empty scores."""
        deciles = backtester._create_decile_portfolios({})
        assert deciles == {}

    # ==================
    # Scoring Tests
    # ==================

    def test_score_growth(self, backtester):
        """Test growth scoring logic."""
        # High growth
        high_growth = backtester._score_growth({
            'revenue_growth_yoy': 20,  # 20%
            'eps_growth_yoy': 25,
        })
        assert high_growth > 0.7

        # Negative growth
        negative_growth = backtester._score_growth({
            'revenue_growth_yoy': -15,
            'eps_growth_yoy': -20,
        })
        assert negative_growth < 0.4

        # No data - should return neutral
        no_data = backtester._score_growth({})
        assert no_data == 0.5

    def test_score_value(self, backtester):
        """Test value scoring logic."""
        # Cheap valuation
        cheap = backtester._score_value({
            'pe_ratio': 10,
            'ps_ratio': 1,
            'pb_ratio': 0.5,
        })
        assert cheap > 0.7

        # Expensive valuation
        expensive = backtester._score_value({
            'pe_ratio': 50,
            'ps_ratio': 10,
            'pb_ratio': 8,
        })
        assert expensive < 0.3

        # No data
        no_data = backtester._score_value({})
        assert no_data == 0.5

    def test_score_profitability(self, backtester):
        """Test profitability scoring logic."""
        # High profitability
        high_profit = backtester._score_profitability({
            'net_margin': 25,
            'roe': 30,
        })
        assert high_profit > 0.6

        # Low profitability
        low_profit = backtester._score_profitability({
            'net_margin': 2,
            'roe': 5,
        })
        assert low_profit < 0.4

    def test_score_momentum(self, backtester):
        """Test momentum scoring logic."""
        # Strong momentum (prices ascending)
        prices = [100 * (1.001 ** i) for i in range(252)][::-1]
        strong_momentum = backtester._score_momentum(prices)
        assert strong_momentum > 0.5

        # Weak momentum (prices descending)
        prices_down = [100 * (0.999 ** i) for i in range(252)][::-1]
        weak_momentum = backtester._score_momentum(prices_down)
        assert weak_momentum < 0.5

        # Insufficient data
        short_prices = [100, 101, 102]
        insufficient = backtester._score_momentum(short_prices)
        assert insufficient == 0.5

    # ==================
    # Turnover Tests
    # ==================

    def test_calculate_turnover_full(self, backtester):
        """Test full turnover calculation."""
        previous = ["AAPL", "MSFT", "GOOGL"]
        current = ["META", "AMZN", "NFLX"]

        turnover = backtester._calculate_turnover(previous, current)
        assert turnover == 1.0  # Complete turnover

    def test_calculate_turnover_none(self, backtester):
        """Test zero turnover calculation."""
        holdings = ["AAPL", "MSFT", "GOOGL"]

        turnover = backtester._calculate_turnover(holdings, holdings)
        assert turnover == 0.0  # No turnover

    def test_calculate_turnover_partial(self, backtester):
        """Test partial turnover calculation."""
        previous = ["AAPL", "MSFT", "GOOGL"]
        current = ["AAPL", "MSFT", "META"]  # 1 of 3 changed

        turnover = backtester._calculate_turnover(previous, current)
        assert 0.3 < turnover < 0.4  # ~33% turnover

    def test_calculate_turnover_empty_previous(self, backtester):
        """Test turnover with no previous holdings."""
        current = ["AAPL", "MSFT", "GOOGL"]

        turnover = backtester._calculate_turnover([], current)
        assert turnover == 1.0  # Initial purchase = 100% turnover

    # ==================
    # Max Drawdown Tests
    # ==================

    def test_calculate_max_drawdown(self, backtester):
        """Test maximum drawdown calculation."""
        # Returns: up 10%, down 20%, up 5% = peak then valley
        returns = [0.10, -0.20, 0.05]

        max_dd = backtester._calculate_max_drawdown(returns)

        # After 10%: 1.10, peak = 1.10
        # After -20%: 1.10 * 0.80 = 0.88, dd = (1.10-0.88)/1.10 = 0.20
        assert 0.19 < max_dd < 0.21

    def test_calculate_max_drawdown_no_drawdown(self, backtester):
        """Test with no drawdown (always positive)."""
        returns = [0.05, 0.05, 0.05, 0.05]

        max_dd = backtester._calculate_max_drawdown(returns)
        assert max_dd == 0.0

    def test_calculate_max_drawdown_empty(self, backtester):
        """Test with empty returns."""
        max_dd = backtester._calculate_max_drawdown([])
        assert max_dd == 0.0

    # ==================
    # Monotonicity Tests
    # ==================

    def test_calculate_monotonicity_perfect(self, backtester):
        """Test perfect monotonicity score."""
        decile_returns = {
            1: 0.20,
            2: 0.18,
            3: 0.16,
            4: 0.14,
            5: 0.12,
            6: 0.10,
            7: 0.08,
            8: 0.06,
            9: 0.04,
            10: 0.02,
        }

        score = backtester._calculate_monotonicity(decile_returns)
        assert score == 1.0

    def test_calculate_monotonicity_random(self, backtester):
        """Test monotonicity with random ordering."""
        decile_returns = {
            1: 0.10,
            2: 0.15,  # Higher than D1 - violation
            3: 0.05,
            4: 0.20,  # Higher than D3 - violation
            5: 0.12,
        }

        score = backtester._calculate_monotonicity(decile_returns)
        assert 0 < score < 1.0

    def test_calculate_monotonicity_empty(self, backtester):
        """Test monotonicity with insufficient data."""
        score = backtester._calculate_monotonicity({1: 0.10})
        assert score == 0.0

    # ==================
    # Async Method Tests
    # ==================

    @pytest.mark.asyncio
    async def test_get_price_as_of(self, backtester, mock_session):
        """Test fetching historical price."""
        mock_result = MagicMock()
        mock_result.fetchone.return_value = (150.25,)
        mock_session.execute = AsyncMock(return_value=mock_result)

        price = await backtester._get_price_as_of("AAPL", date(2023, 6, 15))

        assert price == 150.25

    @pytest.mark.asyncio
    async def test_get_price_as_of_not_found(self, backtester, mock_session):
        """Test price lookup when not found."""
        mock_result = MagicMock()
        mock_result.fetchone.return_value = None
        mock_session.execute = AsyncMock(return_value=mock_result)

        price = await backtester._get_price_as_of("INVALID", date(2023, 6, 15))

        assert price is None

    @pytest.mark.asyncio
    async def test_get_stock_return(self, backtester, mock_session):
        """Test stock return calculation."""
        # Mock price lookups
        call_count = [0]
        def mock_fetchone():
            call_count[0] += 1
            if call_count[0] == 1:
                return (100.0,)  # Start price
            return (110.0,)  # End price

        mock_result = MagicMock()
        mock_result.fetchone = mock_fetchone
        mock_session.execute = AsyncMock(return_value=mock_result)

        ret = await backtester._get_stock_return(
            "AAPL",
            date(2023, 1, 1),
            date(2023, 6, 30)
        )

        # (110 - 100) / 100 = 0.10
        assert ret == 0.10

    @pytest.mark.asyncio
    async def test_calculate_portfolio_return(self, backtester, mock_session):
        """Test portfolio return calculation with costs."""
        # Mock returns for each ticker
        backtester._get_stock_return = AsyncMock(side_effect=[0.10, 0.05, 0.15])

        portfolio_return = await backtester._calculate_portfolio_return(
            tickers=["AAPL", "MSFT", "GOOGL"],
            start_date=date(2023, 1, 1),
            end_date=date(2023, 1, 31),
            transaction_cost_bps=10.0,
            slippage_bps=5.0
        )

        # Average return: (0.10 + 0.05 + 0.15) / 3 = 0.10
        # Cost adjustment: 15 bps = 0.0015
        # Net: 0.10 - 0.0015 = 0.0985
        assert 0.098 < portfolio_return < 0.099


class TestBacktestResults:
    """Test BacktestResults aggregation."""

    @pytest.fixture
    def mock_session(self):
        return AsyncMock()

    @pytest.fixture
    def backtester(self, mock_session):
        return ICScoreBacktester(mock_session)

    def test_aggregate_results(self, backtester):
        """Test result aggregation."""
        config = BacktestConfig(
            start_date=date(2023, 1, 1),
            end_date=date(2023, 12, 31),
        )

        # Create mock period results
        period_results = []
        for month in range(1, 13):
            for decile in range(1, 11):
                # Higher decile (worse score) = lower return
                base_return = 0.02 - (decile * 0.002)
                period_results.append(PeriodResult(
                    period_start=date(2023, month, 1),
                    period_end=date(2023, month, 28),
                    decile=decile,
                    holdings=[f"TICK{decile}A", f"TICK{decile}B"],
                    num_holdings=2,
                    period_return=base_return + (0.001 * month),
                    benchmark_return=0.01,
                    excess_return=base_return - 0.01,
                    avg_score=95 - (decile * 8),
                    turnover=0.1,
                ))

        results = backtester._aggregate_results(config, period_results)

        assert isinstance(results, BacktestResults)
        assert len(results.period_results) == 120  # 12 months * 10 deciles

        # Top decile should outperform bottom
        assert results.top_bottom_spread > 0

        # Monotonicity should be positive (returns decrease with decile)
        assert results.monotonicity_score > 0.5

        # Hit rate should be high (D1 beats D10 in most periods)
        assert results.hit_rate > 0.5


class TestBacktesterEdgeCases:
    """Test edge cases and boundary conditions."""

    @pytest.fixture
    def mock_session(self):
        return AsyncMock()

    @pytest.fixture
    def backtester(self, mock_session):
        return ICScoreBacktester(mock_session)

    def test_period_generation_across_year(self, backtester):
        """Test period generation spanning multiple years."""
        periods = backtester._generate_periods(
            start_date=date(2022, 11, 1),
            end_date=date(2023, 2, 28),
            frequency=RebalanceFrequency.MONTHLY
        )

        assert len(periods) == 4
        # Should cross year boundary correctly
        assert periods[1][0].year == 2022
        assert periods[2][0].year == 2023

    def test_decile_creation_with_ties(self, backtester):
        """Test decile creation when scores have ties."""
        scores = {
            "A": 80, "B": 80, "C": 80,  # Tie
            "D": 70, "E": 70,
            "F": 60, "G": 50, "H": 40, "I": 30, "J": 20
        }

        deciles = backtester._create_decile_portfolios(scores)

        total = sum(len(d) for d in deciles.values())
        assert total == 10  # All stocks assigned

    @pytest.mark.asyncio
    async def test_portfolio_return_empty_tickers(self, backtester):
        """Test portfolio return with no tickers."""
        ret = await backtester._calculate_portfolio_return(
            tickers=[],
            start_date=date(2023, 1, 1),
            end_date=date(2023, 1, 31),
            transaction_cost_bps=10.0,
            slippage_bps=5.0
        )

        assert ret == 0.0

    @pytest.mark.asyncio
    async def test_portfolio_return_all_missing_prices(self, backtester, mock_session):
        """Test portfolio return when all prices are missing."""
        backtester._get_stock_return = AsyncMock(return_value=None)

        ret = await backtester._calculate_portfolio_return(
            tickers=["INVALID1", "INVALID2"],
            start_date=date(2023, 1, 1),
            end_date=date(2023, 1, 31),
            transaction_cost_bps=10.0,
            slippage_bps=5.0
        )

        assert ret == 0.0

    def test_score_growth_extreme_values(self, backtester):
        """Test growth scoring with extreme values."""
        # Extreme growth
        extreme_growth = backtester._score_growth({
            'revenue_growth_yoy': 100,
            'eps_growth_yoy': 200,
        })
        assert 0 <= extreme_growth <= 1

        # Extreme decline
        extreme_decline = backtester._score_growth({
            'revenue_growth_yoy': -80,
            'eps_growth_yoy': -100,
        })
        assert 0 <= extreme_decline <= 1

    def test_score_value_negative_ratios(self, backtester):
        """Test value scoring with negative ratios (loss-making companies)."""
        negative_pe = backtester._score_value({
            'pe_ratio': -10,  # Negative earnings
            'ps_ratio': 2,
            'pb_ratio': 1,
        })
        # Should handle gracefully - only use positive ratios
        assert 0 <= negative_pe <= 1
