"""Integration tests for IC Score Calculator.

Tests the IC Score calculation pipeline end-to-end against a
real PostgreSQL database with all upstream data seeded.

The IC Score Calculator reads from:
  - fundamental_metrics_extended (growth, profitability, health)
  - valuation_ratios (P/E, P/B, P/S, market cap)
  - technical_indicators (momentum, technical signals)
  - financials (fallback financial data)
  - risk_metrics (beta for intrinsic value)
  - insider_trades, news_articles, analyst_ratings,
    institutional_holdings (optional signal factors)

It writes to: ic_scores
"""

import os
from datetime import date, datetime, timedelta, timezone

import pytest
import pytest_asyncio
from unittest.mock import patch

from sqlalchemy import text

pytestmark = [
    pytest.mark.skipif(
        os.getenv("INTEGRATION_TEST_DB") != "true",
        reason="INTEGRATION_TEST_DB not set",
    ),
    pytest.mark.asyncio,
]

from pipelines.tests.integration.seed_data import (
    seed_companies,
    seed_financials,
    seed_risk_metrics,
    seed_stock_prices,
    seed_treasury_rates,
    seed_ttm_financials,
)

# Valid IC Score ratings
VALID_RATINGS = [
    "Strong Buy",
    "Buy",
    "Hold",
    "Underperform",
    "Sell",
]


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------


async def _seed_fundamental_metrics(session, ticker):
    """Insert a realistic fundamental_metrics_extended row.

    Provides growth, profitability, and financial health data
    that the IC Score calculator reads as pre-calculated metrics.
    """
    today = date.today()
    await session.execute(
        text(
            "INSERT INTO fundamental_metrics_extended"
            " (ticker, calculation_date,"
            "  revenue_growth_yoy, eps_growth_yoy,"
            "  fcf_growth_yoy,"
            "  gross_margin, operating_margin, net_margin,"
            "  roe, roa, roic,"
            "  debt_to_equity, current_ratio, quick_ratio,"
            "  ev_to_ebitda, ev_to_revenue,"
            "  dcf_fair_value, dcf_upside_percent,"
            "  graham_number)"
            " VALUES (:t, :d,"
            "  :rev_g, :eps_g,"
            "  :fcf_g,"
            "  :gm, :om, :nm,"
            "  :roe, :roa, :roic,"
            "  :dte, :cr, :qr,"
            "  :ev_ebitda, :ev_rev,"
            "  :dcf, :dcf_up,"
            "  :graham)"
            " ON CONFLICT (ticker, calculation_date)"
            " DO NOTHING"
        ),
        {
            "t": ticker,
            "d": today,
            "rev_g": 12.5,
            "eps_g": 15.3,
            "fcf_g": 10.0,
            "gm": 42.0,
            "om": 28.0,
            "nm": 22.0,
            "roe": 23.0,
            "roa": 8.5,
            "roic": 18.0,
            "dte": 0.85,
            "cr": 1.8,
            "qr": 1.5,
            "ev_ebitda": 18.5,
            "ev_rev": 6.0,
            "dcf": 200.0,
            "dcf_up": 8.1,
            "graham": 165.0,
        },
    )
    await session.commit()


async def _seed_valuation_ratios(session, ticker):
    """Insert a valuation_ratios row so IC Score can read
    P/E, P/B, P/S, price, and market cap."""
    today = date.today()
    await session.execute(
        text(
            "INSERT INTO valuation_ratios"
            " (ticker, calculation_date, stock_price,"
            "  ttm_pe_ratio, ttm_pb_ratio, ttm_ps_ratio,"
            "  ttm_market_cap)"
            " VALUES (:t, :d, :price,"
            "  :pe, :pb, :ps,"
            "  :mcap)"
            " ON CONFLICT (ticker, calculation_date)"
            " DO NOTHING"
        ),
        {
            "t": ticker,
            "d": today,
            "price": 185.00,
            "pe": 28.5,
            "pb": 45.0,
            "ps": 7.5,
            "mcap": 2_850_000_000_000,
        },
    )
    await session.commit()


async def _seed_technical_indicators(session, ticker):
    """Insert technical indicator rows so IC Score can
    calculate momentum and technical scores.

    Seeds recent data (within 14-day window the calculator
    uses) with standard indicator names.
    """
    now = datetime.now(timezone.utc).replace(tzinfo=None)
    indicators = {
        "rsi_14": 55.0,
        "macd": 1.2,
        "macd_signal": 0.8,
        "macd_histogram": 0.4,
        "sma_50": 178.0,
        "sma_200": 165.0,
        "ema_12": 182.0,
        "ema_26": 179.0,
        "atr_14": 3.5,
        "1m_return": 5.2,
        "3m_return": 12.0,
        "6m_return": 18.5,
        "12m_return": 25.0,
    }

    for name, value in indicators.items():
        await session.execute(
            text(
                "INSERT INTO technical_indicators"
                " (time, ticker, indicator_name, value)"
                " VALUES (:t, :ticker, :name, :val)"
                " ON CONFLICT DO NOTHING"
            ),
            {
                "t": now - timedelta(hours=1),
                "ticker": ticker,
                "name": name,
                "val": value,
            },
        )
    await session.commit()


async def _seed_sector_percentiles(session, sector):
    """Populate sector_percentiles for all metrics the IC Score
    calculator looks up, then refresh the materialized view.

    Without this data the calculator gets 'No sector stats'
    for every metric and the scoring falls through to code
    paths that hit missing tables (eps_estimates, etc.).
    """
    today = date.today()
    metrics = [
        ("pe_ratio", 10, 15, 20, 25, 35, 45, 60, 28, 12, 50),
        ("pb_ratio", 1, 2, 3, 5, 8, 15, 40, 6, 5, 50),
        ("ps_ratio", 0.5, 1, 2, 4, 7, 12, 25, 5, 4, 50),
        (
            "revenue_growth_yoy",
            -10,
            0,
            3,
            8,
            15,
            25,
            60,
            10,
            12,
            50,
        ),
        (
            "eps_growth_yoy",
            -20,
            -5,
            2,
            10,
            20,
            35,
            80,
            12,
            18,
            50,
        ),
        (
            "net_margin",
            -5,
            2,
            5,
            10,
            18,
            25,
            40,
            12,
            8,
            50,
        ),
        (
            "gross_margin",
            10,
            20,
            30,
            40,
            55,
            65,
            80,
            42,
            15,
            50,
        ),
        (
            "operating_margin",
            -5,
            5,
            10,
            18,
            28,
            35,
            50,
            18,
            12,
            50,
        ),
        ("roe", -10, 2, 8, 15, 25, 35, 60, 18, 14, 50),
        ("roa", -5, 1, 3, 7, 12, 18, 30, 8, 6, 50),
        (
            "debt_to_equity",
            0,
            0.1,
            0.3,
            0.6,
            1.0,
            1.8,
            5.0,
            0.9,
            0.8,
            50,
        ),
        (
            "current_ratio",
            0.5,
            0.8,
            1.0,
            1.5,
            2.2,
            3.5,
            8.0,
            1.8,
            1.2,
            50,
        ),
        (
            "quick_ratio",
            0.3,
            0.5,
            0.8,
            1.2,
            1.8,
            3.0,
            6.0,
            1.4,
            1.0,
            50,
        ),
    ]
    for row in metrics:
        (
            name,
            mn,
            p10,
            p25,
            p50,
            p75,
            p90,
            mx,
            mean,
            std,
            cnt,
        ) = row
        await session.execute(
            text(
                "INSERT INTO sector_percentiles"
                " (sector, metric_name, calculated_at,"
                "  min_value, p10_value, p25_value,"
                "  p50_value, p75_value, p90_value,"
                "  max_value, mean_value, std_dev,"
                "  sample_count)"
                " VALUES (:sec, :met, :d,"
                "  :mn, :p10, :p25,"
                "  :p50, :p75, :p90,"
                "  :mx, :mean, :std,"
                "  :cnt)"
                " ON CONFLICT DO NOTHING"
            ),
            {
                "sec": sector,
                "met": name,
                "d": today,
                "mn": mn,
                "p10": p10,
                "p25": p25,
                "p50": p50,
                "p75": p75,
                "p90": p90,
                "mx": mx,
                "mean": mean,
                "std": std,
                "cnt": cnt,
            },
        )
    await session.commit()

    # Refresh materialized view so queries see the data
    await session.execute(
        text(
            "REFRESH MATERIALIZED VIEW"
            " mv_latest_sector_percentiles"
        )
    )
    await session.commit()


async def _seed_all_upstream(session, ticker, sector=None):
    """Seed all upstream data required for IC Score calculation.

    Calls seed_data helpers plus local helpers for tables
    that do not have dedicated seed functions.
    """
    await seed_companies(session, tickers=[ticker])
    await seed_financials(session, ticker, quarters=8)
    await seed_stock_prices(session, ticker, days=300)
    await seed_ttm_financials(session, ticker)
    await seed_risk_metrics(session, ticker)
    await seed_treasury_rates(session, days=30)
    await _seed_fundamental_metrics(session, ticker)
    await _seed_valuation_ratios(session, ticker)
    await _seed_technical_indicators(session, ticker)
    if sector:
        await _seed_sector_percentiles(session, sector)


# ---------------------------------------------------------------------------
# Fixtures
# ---------------------------------------------------------------------------


@pytest_asyncio.fixture
async def calculator(db):
    """Create an ICScoreCalculator with test database injected."""
    with patch(
        "pipelines.ic_score_calculator.get_database",
        return_value=db,
    ):
        from pipelines.ic_score_calculator import (
            ICScoreCalculator,
        )

        calc = ICScoreCalculator()
        yield calc


@pytest_asyncio.fixture
async def seeded_aapl(db):
    """Seed all upstream data for AAPL IC Score calculation."""
    async with db.session() as session:
        await _seed_all_upstream(
            session, "AAPL", sector="Technology"
        )


# ---------------------------------------------------------------------------
# Tests
# ---------------------------------------------------------------------------


async def test_ic_score_end_to_end(
    db, seeded_aapl, calculator
):
    """Seed all upstream data for AAPL, run IC Score
    calculator, and verify at least 1 row in ic_scores with
    overall_score between 1 and 100."""
    stocks = [{"ticker": "AAPL", "sector": "Technology"}]
    await calculator.process_stocks(stocks, show_progress=False)

    async with db.session() as session:
        result = await session.execute(
            text(
                "SELECT overall_score, rating,"
                " data_completeness"
                " FROM ic_scores"
                " WHERE ticker = 'AAPL'"
                " ORDER BY date DESC LIMIT 1"
            )
        )
        row = result.fetchone()

    assert row is not None, (
        "ic_scores should have a row for AAPL after "
        "running the calculator"
    )
    overall_score = float(row[0])
    assert 1.0 <= overall_score <= 100.0, (
        f"overall_score={overall_score} should be "
        f"between 1 and 100"
    )


async def test_rating_is_valid(
    db, seeded_aapl, calculator
):
    """Verify the rating is one of the defined rating
    categories."""
    stocks = [{"ticker": "AAPL", "sector": "Technology"}]
    await calculator.process_stocks(stocks, show_progress=False)

    async with db.session() as session:
        result = await session.execute(
            text(
                "SELECT rating FROM ic_scores"
                " WHERE ticker = 'AAPL'"
                " ORDER BY date DESC LIMIT 1"
            )
        )
        row = result.fetchone()

    assert row is not None, (
        "ic_scores should have a row for AAPL"
    )
    rating = row[0]
    assert rating in VALID_RATINGS, (
        f"rating='{rating}' should be one of {VALID_RATINGS}"
    )


async def test_score_components_present(
    db, seeded_aapl, calculator
):
    """Verify the ic_scores row has factor scores (value,
    growth, profitability) all between 0 and 100."""
    stocks = [{"ticker": "AAPL", "sector": "Technology"}]
    await calculator.process_stocks(stocks, show_progress=False)

    async with db.session() as session:
        result = await session.execute(
            text(
                "SELECT value_score, growth_score,"
                " profitability_score,"
                " financial_health_score"
                " FROM ic_scores"
                " WHERE ticker = 'AAPL'"
                " ORDER BY date DESC LIMIT 1"
            )
        )
        row = result.fetchone()

    assert row is not None, (
        "ic_scores should have a row for AAPL"
    )

    score_names = [
        "value_score",
        "growth_score",
        "profitability_score",
        "financial_health_score",
    ]
    for i, name in enumerate(score_names):
        score = row[i]
        assert score is not None, (
            f"{name} should not be NULL when upstream "
            f"data is seeded"
        )
        score_float = float(score)
        assert 0.0 <= score_float <= 100.0, (
            f"{name}={score_float} should be between "
            f"0 and 100"
        )


async def test_skips_without_upstream_data(
    db, calculator
):
    """Seed a company WITHOUT any upstream metric data and
    verify no IC score is generated (insufficient factors)."""
    async with db.session() as session:
        await seed_companies(session, tickers=["ZZZZ"])

    stocks = [{"ticker": "ZZZZ", "sector": "Technology"}]
    await calculator.process_stocks(stocks, show_progress=False)

    async with db.session() as session:
        result = await session.execute(
            text(
                "SELECT COUNT(*) FROM ic_scores"
                " WHERE ticker = 'ZZZZ'"
            )
        )
        count = result.scalar()

    assert count == 0, (
        "No IC score should be generated for a ticker "
        "without upstream data"
    )


async def test_multi_ticker_ic_scores(db, calculator):
    """Seed 3 tickers with full upstream data, run the
    calculator, and verify each ticker has an IC score."""
    stocks = [
        {"ticker": "AAPL", "sector": "Technology"},
        {"ticker": "JNJ", "sector": "Healthcare"},
        {"ticker": "JPM", "sector": "Financial Services"},
    ]

    async with db.session() as session:
        for s in stocks:
            await _seed_all_upstream(
                session, s["ticker"], sector=s["sector"]
            )
    await calculator.process_stocks(stocks, show_progress=False)

    async with db.session() as session:
        result = await session.execute(
            text(
                "SELECT ticker, overall_score, rating"
                " FROM ic_scores"
                " ORDER BY ticker"
            )
        )
        rows = result.fetchall()

    scored_tickers = {row[0] for row in rows}
    expected = [s["ticker"] for s in stocks]
    for ticker in expected:
        assert ticker in scored_tickers, (
            f"{ticker} should have an IC score, "
            f"got scores for: {scored_tickers}"
        )

    # Verify all scores are valid
    for row in rows:
        ticker, score, rating = row[0], float(row[1]), row[2]
        assert 1.0 <= score <= 100.0, (
            f"{ticker}: overall_score={score} should be "
            f"between 1 and 100"
        )
        assert rating in VALID_RATINGS, (
            f"{ticker}: rating='{rating}' should be one "
            f"of {VALID_RATINGS}"
        )
