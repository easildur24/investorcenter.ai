"""Tests for pure scoring methods in ic_score_calculator.py.

Phase 1A: No database dependencies. Tests only the synchronous scoring
methods that take data dicts and return (score, metadata) tuples.

The async methods (calculate_value_score, calculate_growth_score, etc.)
are also tested using their legacy/absolute-benchmark fallback paths by
disabling sector-relative scoring.
"""

import asyncio
import os
from unittest.mock import MagicMock, patch

import numpy as np
import pytest

# conftest.py sets LOG_DIR and patches logging.FileHandler before import
with patch("pipelines.ic_score_calculator.get_database"):
    from pipelines.ic_score_calculator import ICScoreCalculator


@pytest.fixture
def calc():
    """Create an ICScoreCalculator with mocked DB and disabled sector scoring."""
    with patch("pipelines.ic_score_calculator.get_database"):
        c = ICScoreCalculator()
    # Disable sector-relative scoring so async methods use fallback paths
    c.USE_SECTOR_RELATIVE_SCORING = False
    c._sector_calculator = None
    return c


# =====================================================================
# calculate_momentum_score
# =====================================================================


class TestCalculateMomentumScore:
    def test_all_periods(self, calc):
        tech_data = {
            "1m_return": 10.0,
            "3m_return": 5.0,
            "6m_return": 15.0,
            "12m_return": 20.0,
        }
        score, meta = calc.calculate_momentum_score(tech_data)
        assert score is not None
        assert 0 <= score <= 100
        # All positive returns -> score should be > 50
        assert score > 50
        assert "1m_return" in meta

    def test_all_negative(self, calc):
        tech_data = {
            "1m_return": -10.0,
            "3m_return": -15.0,
            "6m_return": -20.0,
            "12m_return": -25.0,
        }
        score, meta = calc.calculate_momentum_score(tech_data)
        assert score is not None
        assert score < 50

    def test_mixed_returns(self, calc):
        tech_data = {
            "1m_return": 10.0,
            "3m_return": -5.0,
        }
        score, meta = calc.calculate_momentum_score(tech_data)
        assert score is not None
        assert 0 <= score <= 100

    def test_empty_data(self, calc):
        score, meta = calc.calculate_momentum_score({})
        assert score is None

    def test_none_data(self, calc):
        score, meta = calc.calculate_momentum_score(None)
        assert score is None

    def test_no_matching_keys(self, calc):
        tech_data = {"rsi": 55, "macd_histogram": 0.5}
        score, meta = calc.calculate_momentum_score(tech_data)
        assert score is None

    def test_extreme_positive(self, calc):
        tech_data = {"1m_return": 50.0}
        score, meta = calc.calculate_momentum_score(tech_data)
        assert score is not None
        # 50 + 50*2.5 = 175 -> clamped to 100
        assert score == 100.0


# =====================================================================
# calculate_technical_score
# =====================================================================


class TestCalculateTechnicalScore:
    def test_rsi_neutral(self, calc):
        tech_data = {"rsi": 50}
        score, meta = calc.calculate_technical_score(tech_data)
        assert score is not None
        assert abs(score - 50.0) < 1

    def test_rsi_overbought(self, calc):
        tech_data = {"rsi": 70}
        score, meta = calc.calculate_technical_score(tech_data)
        assert score is not None
        assert score == 100.0

    def test_rsi_oversold(self, calc):
        tech_data = {"rsi": 30}
        score, meta = calc.calculate_technical_score(tech_data)
        assert score is not None
        assert abs(score - 0.0) < 1

    def test_macd_positive(self, calc):
        tech_data = {"macd_histogram": 2.0}
        score, meta = calc.calculate_technical_score(tech_data)
        assert score is not None
        # 50 + min(50, max(-50, 2.0 * 10.0)) = 50 + 20 = 70
        assert abs(score - 70.0) < 0.01

    def test_macd_negative(self, calc):
        tech_data = {"macd_histogram": -3.0}
        score, meta = calc.calculate_technical_score(tech_data)
        assert score is not None
        # 50 + min(50, max(-50, -3.0 * 10.0)) = 50 + (-30) = 20
        assert abs(score - 20.0) < 0.01

    def test_trend_above_sma(self, calc):
        tech_data = {"current_price": 110, "sma_50": 100}
        score, meta = calc.calculate_technical_score(tech_data)
        assert score is not None
        # trend = (110/100 - 1) * 100 = 10%
        # trend_score = max(0, min(100, 50 + 10 * 5)) = 100
        assert score == 100.0

    def test_trend_below_sma(self, calc):
        tech_data = {"current_price": 90, "sma_50": 100}
        score, meta = calc.calculate_technical_score(tech_data)
        assert score is not None
        # trend = (90/100 - 1) * 100 = -10%
        # trend_score = max(0, min(100, 50 + (-10) * 5)) = 0
        assert score == 0.0

    def test_empty_data(self, calc):
        score, meta = calc.calculate_technical_score({})
        assert score is None

    def test_none_data(self, calc):
        score, meta = calc.calculate_technical_score(None)
        assert score is None

    def test_combined_indicators(self, calc):
        tech_data = {
            "rsi": 60,
            "macd_histogram": 1.0,
            "current_price": 105,
            "sma_50": 100,
        }
        score, meta = calc.calculate_technical_score(tech_data)
        assert score is not None
        assert 0 <= score <= 100
        assert "rsi" in meta
        assert "macd_histogram" in meta


# =====================================================================
# calculate_news_sentiment_score
# =====================================================================


class TestCalculateNewsSentimentScore:
    def test_positive_sentiment(self, calc):
        news_data = {
            "article_count": 20,
            "avg_sentiment": 75.0,
            "positive_count": 15,
            "negative_count": 2,
            "neutral_count": 3,
            "recent_article_count": 5,
            "recent_avg_sentiment": 80.0,
        }
        score, meta = calc.calculate_news_sentiment_score(news_data)
        assert score is not None
        assert score > 50
        assert meta["scoring_method"] == "recency_weighted"

    def test_no_recent_articles(self, calc):
        news_data = {
            "article_count": 10,
            "avg_sentiment": 60.0,
            "positive_count": 6,
            "negative_count": 2,
            "neutral_count": 2,
        }
        score, meta = calc.calculate_news_sentiment_score(news_data)
        assert score is not None
        assert meta["scoring_method"] == "overall_only"

    def test_negative_sentiment(self, calc):
        news_data = {
            "article_count": 10,
            "avg_sentiment": 20.0,
            "positive_count": 1,
            "negative_count": 8,
            "neutral_count": 1,
        }
        score, meta = calc.calculate_news_sentiment_score(news_data)
        assert score is not None
        assert score < 50

    def test_none_data(self, calc):
        score, meta = calc.calculate_news_sentiment_score(None)
        assert score is None

    def test_zero_articles(self, calc):
        news_data = {"article_count": 0}
        score, meta = calc.calculate_news_sentiment_score(news_data)
        assert score is None


# =====================================================================
# calculate_analyst_consensus_score
# =====================================================================


class TestCalculateAnalystConsensusScore:
    def test_all_buys(self, calc):
        analyst_data = {
            "total_analysts": 10,
            "buy_count": 10,
            "hold_count": 0,
            "sell_count": 0,
            "avg_price_target": 200.0,
        }
        valuation_data = {"stock_price": 150.0}
        score, meta = calc.calculate_analyst_consensus_score(
            analyst_data, valuation_data
        )
        assert score is not None
        assert score > 80  # Strong buy + upside

    def test_all_sells(self, calc):
        analyst_data = {
            "total_analysts": 10,
            "buy_count": 0,
            "hold_count": 0,
            "sell_count": 10,
            "avg_price_target": 100.0,
        }
        valuation_data = {"stock_price": 150.0}
        score, meta = calc.calculate_analyst_consensus_score(
            analyst_data, valuation_data
        )
        assert score is not None
        assert score < 30  # Sell + downside

    def test_mixed_no_price_target(self, calc):
        analyst_data = {
            "total_analysts": 10,
            "buy_count": 5,
            "hold_count": 5,
            "sell_count": 0,
            "avg_price_target": None,
        }
        score, meta = calc.calculate_analyst_consensus_score(
            analyst_data, None
        )
        assert score is not None
        # (5*100 + 5*50) / 10 = 75
        assert abs(score - 75.0) < 0.01

    def test_none_data(self, calc):
        score, meta = calc.calculate_analyst_consensus_score(None)
        assert score is None

    def test_zero_analysts(self, calc):
        analyst_data = {"total_analysts": 0}
        score, meta = calc.calculate_analyst_consensus_score(analyst_data)
        assert score is None

    def test_price_target_upside(self, calc):
        analyst_data = {
            "total_analysts": 5,
            "buy_count": 5,
            "hold_count": 0,
            "sell_count": 0,
            "avg_price_target": 195.0,
        }
        valuation_data = {"stock_price": 150.0}
        score, meta = calc.calculate_analyst_consensus_score(
            analyst_data, valuation_data
        )
        assert score is not None
        # Upside = (195/150 - 1) * 100 = 30%
        assert "price_target_upside_pct" in meta
        assert abs(meta["price_target_upside_pct"] - 30.0) < 0.01


# =====================================================================
# calculate_insider_activity_score
# =====================================================================


class TestCalculateInsiderActivityScore:
    def test_heavy_buying(self, calc):
        insider_data = {
            "net_buying_90d": 50000,
            "net_buying_value_90d": 1_000_000.0,
            "total_transactions": 5,
        }
        score, meta = calc.calculate_insider_activity_score(insider_data)
        assert score is not None
        # 50 + (1_000_000 / 1_000_000) * 50 = 100
        assert abs(score - 100.0) < 0.01

    def test_heavy_selling(self, calc):
        insider_data = {
            "net_buying_90d": -50000,
            "net_buying_value_90d": -1_000_000.0,
            "total_transactions": 5,
        }
        score, meta = calc.calculate_insider_activity_score(insider_data)
        assert score is not None
        # 50 + (-1_000_000 / 1_000_000) * 50 = 0
        assert abs(score - 0.0) < 0.01

    def test_neutral(self, calc):
        insider_data = {
            "net_buying_90d": 0,
            "net_buying_value_90d": 0,
            "total_transactions": 0,
        }
        score, meta = calc.calculate_insider_activity_score(insider_data)
        assert score is not None
        assert abs(score - 50.0) < 0.01

    def test_none_data(self, calc):
        score, meta = calc.calculate_insider_activity_score(None)
        assert score is None

    def test_fallback_to_shares_when_no_value(self, calc):
        insider_data = {
            "net_buying_90d": 1000,
            "net_buying_value_90d": 0,  # No dollar value
            "total_transactions": 2,
        }
        score, meta = calc.calculate_insider_activity_score(insider_data)
        assert score is not None
        # 50 + (1000 / 2000) = 50 + 0.5 = 50.5
        assert abs(score - 50.5) < 0.01


# =====================================================================
# calculate_institutional_score
# =====================================================================


class TestCalculateInstitutionalScore:
    def test_strong_institutional(self, calc):
        inst_data = {
            "num_institutions": 100,
            "total_shares": 50_000_000,
            "prev_shares": 45_000_000,
            "shares_outstanding": 100_000_000,
        }
        score, meta = calc.calculate_institutional_score(inst_data)
        assert score is not None
        assert score > 50

    def test_single_institution(self, calc):
        inst_data = {
            "num_institutions": 1,
            "total_shares": 1000,
            "prev_shares": None,
            "shares_outstanding": 1_000_000,
        }
        score, meta = calc.calculate_institutional_score(inst_data)
        assert score is not None
        # log2(1) = 0 -> 0 * 10 = 0 breadth score
        assert score < 50

    def test_no_institutions(self, calc):
        inst_data = {"num_institutions": 0}
        score, meta = calc.calculate_institutional_score(inst_data)
        assert score is None

    def test_none_data(self, calc):
        score, meta = calc.calculate_institutional_score(None)
        assert score is None

    def test_increasing_holdings(self, calc):
        inst_data = {
            "num_institutions": 50,
            "total_shares": 60_000_000,
            "prev_shares": 50_000_000,
            "shares_outstanding": 100_000_000,
        }
        score, meta = calc.calculate_institutional_score(inst_data)
        assert score is not None
        # Change = +20% -> change_score = 50 + 20*5 = 150 -> clamped to 100
        assert "holdings_change_pct" in meta
        assert meta["holdings_change_pct"] == 20.0

    def test_decreasing_holdings(self, calc):
        inst_data = {
            "num_institutions": 50,
            "total_shares": 40_000_000,
            "prev_shares": 50_000_000,
            "shares_outstanding": 100_000_000,
        }
        score, meta = calc.calculate_institutional_score(inst_data)
        assert score is not None
        assert "holdings_change_pct" in meta
        assert meta["holdings_change_pct"] == -20.0


# =====================================================================
# calculate_intrinsic_value_score
# =====================================================================


class TestCalculateIntrinsicValueScore:
    def test_undervalued_dcf(self, calc):
        fund_metrics = {
            "dcf_fair_value": 200.0,
            "dcf_upside_percent": 33.33,
            "graham_number": None,
        }
        val_data = {"stock_price": 150.0}
        score, meta = calc.calculate_intrinsic_value_score(
            fund_metrics, val_data
        )
        assert score is not None
        assert score > 50  # Undervalued -> above 50

    def test_overvalued_dcf(self, calc):
        fund_metrics = {
            "dcf_fair_value": 100.0,
            "dcf_upside_percent": -33.33,
            "graham_number": None,
        }
        val_data = {"stock_price": 150.0}
        score, meta = calc.calculate_intrinsic_value_score(
            fund_metrics, val_data
        )
        assert score is not None
        assert score < 50  # Overvalued -> below 50

    def test_at_fair_value(self, calc):
        fund_metrics = {
            "dcf_fair_value": 150.0,
            "dcf_upside_percent": 0.0,
            "graham_number": None,
        }
        val_data = {"stock_price": 150.0}
        score, meta = calc.calculate_intrinsic_value_score(
            fund_metrics, val_data
        )
        assert score is not None
        assert abs(score - 50.0) < 1

    def test_combined_dcf_and_graham(self, calc):
        fund_metrics = {
            "dcf_fair_value": 200.0,
            "dcf_upside_percent": 33.33,
            "graham_number": 180.0,  # Also undervalued
        }
        val_data = {"stock_price": 150.0}
        score, meta = calc.calculate_intrinsic_value_score(
            fund_metrics, val_data
        )
        assert score is not None
        assert score > 50
        assert "graham_number" in meta

    def test_extreme_upside_clamped(self, calc):
        fund_metrics = {
            "dcf_fair_value": 1000.0,
            "dcf_upside_percent": 500.0,  # Extreme outlier
            "graham_number": None,
        }
        val_data = {"stock_price": 150.0}
        score, meta = calc.calculate_intrinsic_value_score(
            fund_metrics, val_data
        )
        assert score is not None
        # Clamped to 50% upside -> score = 100
        assert abs(score - 100.0) < 0.01

    def test_none_metrics(self, calc):
        score, meta = calc.calculate_intrinsic_value_score(None, None)
        assert score is None

    def test_no_price(self, calc):
        fund_metrics = {
            "dcf_fair_value": 200.0,
            "dcf_upside_percent": 33.33,
            "graham_number": None,
        }
        val_data = {"stock_price": 0.0}
        score, meta = calc.calculate_intrinsic_value_score(
            fund_metrics, val_data
        )
        assert score is None


# =====================================================================
# Async scoring methods with legacy fallback paths
# =====================================================================


class TestCalculateValueScoreLegacy:
    @pytest.mark.asyncio
    async def test_low_pe(self, calc):
        # P/E=10 is below benchmark of 15 -> should score high
        val_data = {"pe_ratio": 10.0, "pb_ratio": None, "ps_ratio": None}
        score, meta = await calc.calculate_value_score(val_data, None)
        assert score is not None
        # 100 - (10-15)*2 = 100+10 = 110 -> clamped to 100
        assert score == 100.0

    @pytest.mark.asyncio
    async def test_high_pe(self, calc):
        # P/E=50 is well above benchmark
        val_data = {"pe_ratio": 50.0, "pb_ratio": None, "ps_ratio": None}
        score, meta = await calc.calculate_value_score(val_data, None)
        assert score is not None
        # 100 - (50-15)*2 = 100 - 70 = 30
        assert abs(score - 30.0) < 0.01

    @pytest.mark.asyncio
    async def test_none_valuation(self, calc):
        score, meta = await calc.calculate_value_score(None, None)
        assert score is None

    @pytest.mark.asyncio
    async def test_all_none_ratios(self, calc):
        val_data = {"pe_ratio": None, "pb_ratio": None, "ps_ratio": None}
        score, meta = await calc.calculate_value_score(val_data, None)
        assert score is None

    @pytest.mark.asyncio
    async def test_multiple_ratios_averaged(self, calc):
        val_data = {
            "pe_ratio": 15.0,
            "pb_ratio": 2.0,
            "ps_ratio": 2.0,
        }
        score, meta = await calc.calculate_value_score(val_data, None)
        assert score is not None
        # All at benchmark -> all scores = 100
        # pe: 100 - (15-15)*2 = 100
        # pb: 100 - (2-2)*20 = 100
        # ps: 100 - (2-2)*20 = 100
        assert abs(score - 100.0) < 0.01


class TestCalculateGrowthScoreLegacy:
    @pytest.mark.asyncio
    async def test_positive_growth(self, calc):
        fund_metrics = {
            "revenue_growth_yoy": 20.0,
            "eps_growth_yoy": 15.0,
            "fcf_growth_yoy": None,
        }
        score, meta = await calc.calculate_growth_score(
            fund_metrics, None, None
        )
        assert score is not None
        # rev: 50 + 20*2.5 = 100, eps: 50 + 15*2.5 = 87.5
        # avg = 93.75
        assert score > 80

    @pytest.mark.asyncio
    async def test_negative_growth(self, calc):
        fund_metrics = {
            "revenue_growth_yoy": -10.0,
            "eps_growth_yoy": -15.0,
            "fcf_growth_yoy": None,
        }
        score, meta = await calc.calculate_growth_score(
            fund_metrics, None, None
        )
        assert score is not None
        assert score < 50

    @pytest.mark.asyncio
    async def test_no_growth_data(self, calc):
        score, meta = await calc.calculate_growth_score(None, None, None)
        assert score is None


class TestCalculateProfitabilityScoreLegacy:
    @pytest.mark.asyncio
    async def test_high_profitability(self, calc):
        fund_metrics = {
            "net_margin": 20.0,
            "gross_margin": None,
            "operating_margin": None,
            "roe": 25.0,
            "roa": 10.0,
        }
        score, meta = await calc.calculate_profitability_score(
            fund_metrics, None, None
        )
        assert score is not None
        assert score > 50

    @pytest.mark.asyncio
    async def test_no_profitability_data(self, calc):
        score, meta = await calc.calculate_profitability_score(
            None, None, None
        )
        assert score is None


class TestCalculateFinancialHealthScoreLegacy:
    @pytest.mark.asyncio
    async def test_low_debt(self, calc):
        fund_metrics = {
            "debt_to_equity": 0.5,
            "current_ratio": 2.0,
            "quick_ratio": 1.5,
        }
        score, meta = await calc.calculate_financial_health_score(
            fund_metrics, None, None
        )
        assert score is not None
        # D/E: 100 - 0.5*50 = 75
        # CR: 100 - |2.0-2.0|*40 = 100
        # QR: 100 - |1.5-1.5|*40 = 100
        # avg = 91.67
        assert score > 80

    @pytest.mark.asyncio
    async def test_high_debt(self, calc):
        fund_metrics = {
            "debt_to_equity": 3.0,
            "current_ratio": 0.5,
            "quick_ratio": 0.3,
        }
        score, meta = await calc.calculate_financial_health_score(
            fund_metrics, None, None
        )
        assert score is not None
        assert score < 50

    @pytest.mark.asyncio
    async def test_no_data(self, calc):
        score, meta = await calc.calculate_financial_health_score(
            None, None, None
        )
        assert score is None
