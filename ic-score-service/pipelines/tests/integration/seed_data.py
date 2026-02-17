"""Factory functions for seeding test data.

Each function inserts realistic data into the test database.
Tests call only the seed functions they need for isolation.
"""

import math
import random
from datetime import date, timedelta

from sqlalchemy import text

# Representative companies spanning sectors
COMPANIES = [
    ("AAPL", "Apple Inc.", "Technology", "Consumer Electronics"),
    ("JNJ", "Johnson & Johnson", "Healthcare", "Pharmaceuticals"),
    ("JPM", "JPMorgan Chase", "Financial Services", "Banks"),
    ("AMZN", "Amazon.com", "Consumer Cyclical", "Internet Retail"),
    ("XOM", "Exxon Mobil", "Energy", "Oil & Gas Integrated"),
    ("CAT", "Caterpillar", "Industrials", "Farm & Heavy Equipment"),
    ("GOOGL", "Alphabet", "Communication Services", "Internet"),
    ("AMT", "American Tower", "Real Estate", "REIT"),
    ("NEE", "NextEra Energy", "Utilities", "Utilities"),
    ("LIN", "Linde", "Basic Materials", "Specialty Chemicals"),
]


async def seed_companies(session, tickers=None):
    """Insert companies. If tickers is None, insert all 10."""
    targets = COMPANIES
    if tickers:
        targets = [c for c in COMPANIES if c[0] in tickers]
        # Allow custom tickers not in the default list
        for t in tickers:
            if not any(c[0] == t for c in targets):
                targets.append(
                    (t, f"{t} Corp", "Technology", "Software")
                )

    for ticker, name, sector, industry in targets:
        await session.execute(
            text(
                "INSERT INTO companies"
                " (ticker, name, sector, industry)"
                " VALUES (:t, :n, :s, :i)"
                " ON CONFLICT (ticker) DO NOTHING"
            ),
            {"t": ticker, "n": name, "s": sector, "i": industry},
        )
    await session.commit()


async def seed_financials(session, ticker, quarters=8):
    """Insert annual and quarterly financial records.

    Creates 2 annual (10-K) filings and ``quarters`` quarterly
    (10-Q) filings with realistic revenue, net_income, EPS, and
    balance sheet items.

    Quarterly data is CUMULATIVE YTD (matching real SEC filings).
    """
    today = date.today()
    base_revenue = 90_000_000_000  # $90B annual
    base_net_income = 20_000_000_000  # $20B annual
    shares = 15_500_000_000

    # --- Annual 10-K filings ---
    for year_offset in [1, 2]:
        fy = today.year - year_offset
        period_end = date(fy, 12, 31)
        filing_date = date(fy + 1, 2, 15)

        revenue = int(base_revenue * (1.05 ** (2 - year_offset)))
        net_income = int(base_net_income * (1.05 ** (2 - year_offset)))
        eps_basic = round(net_income / shares, 4)
        eps_diluted = round(eps_basic * 0.98, 4)

        await session.execute(
            text(
                "INSERT INTO financials"
                " (ticker, period_end_date, filing_date,"
                "  fiscal_year, fiscal_quarter, statement_type,"
                "  revenue, cost_of_revenue, gross_profit,"
                "  operating_income, net_income,"
                "  eps_basic, eps_diluted,"
                "  shares_outstanding,"
                "  total_assets, total_liabilities,"
                "  shareholders_equity,"
                "  short_term_debt, long_term_debt,"
                "  cash_and_equivalents,"
                "  operating_cash_flow, capex,"
                "  free_cash_flow)"
                " VALUES"
                " (:ticker, :period_end, :filing_date,"
                "  :fy, NULL, '10-K',"
                "  :revenue, :cogs, :gp,"
                "  :oi, :ni,"
                "  :eps_b, :eps_d,"
                "  :shares,"
                "  :ta, :tl,"
                "  :eq,"
                "  :std, :ltd,"
                "  :cash,"
                "  :ocf, :capex,"
                "  :fcf)"
                " ON CONFLICT DO NOTHING"
            ),
            {
                "ticker": ticker,
                "period_end": period_end,
                "filing_date": filing_date,
                "fy": fy,
                "revenue": revenue,
                "cogs": int(revenue * 0.58),
                "gp": int(revenue * 0.42),
                "oi": int(revenue * 0.28),
                "ni": net_income,
                "eps_b": eps_basic,
                "eps_d": eps_diluted,
                "shares": shares,
                "ta": int(revenue * 3.5),
                "tl": int(revenue * 2.5),
                "eq": int(revenue * 1.0),
                "std": int(revenue * 0.05),
                "ltd": int(revenue * 0.8),
                "cash": int(revenue * 0.3),
                "ocf": int(net_income * 1.3),
                "capex": int(-net_income * 0.4),
                "fcf": int(net_income * 0.9),
            },
        )

    # --- Quarterly 10-Q filings (cumulative YTD) ---
    for q_idx in range(quarters):
        # Work backwards from most recent quarter
        quarter_num = 3 - (q_idx % 4)  # Q3, Q2, Q1, Q3, Q2, Q1...
        if quarter_num <= 0:
            quarter_num += 4
        year_back = q_idx // 4
        fy = today.year - year_back

        # Q4 is never a 10-Q (it's in the 10-K)
        if quarter_num == 4:
            quarter_num = 3

        period_end = date(fy, quarter_num * 3, 28)
        filing_date = period_end + timedelta(days=45)

        # Cumulative YTD values
        cum_fraction = quarter_num / 4.0
        growth = 1.05 ** (2 - year_back)
        revenue = int(base_revenue * cum_fraction * growth)
        net_income = int(base_net_income * cum_fraction * growth)
        eps_basic = round(net_income / shares, 4)
        eps_diluted = round(eps_basic * 0.98, 4)

        await session.execute(
            text(
                "INSERT INTO financials"
                " (ticker, period_end_date, filing_date,"
                "  fiscal_year, fiscal_quarter, statement_type,"
                "  revenue, cost_of_revenue, gross_profit,"
                "  operating_income, net_income,"
                "  eps_basic, eps_diluted,"
                "  shares_outstanding,"
                "  total_assets, total_liabilities,"
                "  shareholders_equity,"
                "  short_term_debt, long_term_debt,"
                "  cash_and_equivalents,"
                "  operating_cash_flow, capex,"
                "  free_cash_flow)"
                " VALUES"
                " (:ticker, :period_end, :filing_date,"
                "  :fy, :fq, '10-Q',"
                "  :revenue, :cogs, :gp,"
                "  :oi, :ni,"
                "  :eps_b, :eps_d,"
                "  :shares,"
                "  :ta, :tl,"
                "  :eq,"
                "  :std, :ltd,"
                "  :cash,"
                "  :ocf, :capex,"
                "  :fcf)"
                " ON CONFLICT DO NOTHING"
            ),
            {
                "ticker": ticker,
                "period_end": period_end,
                "filing_date": filing_date,
                "fy": fy,
                "fq": quarter_num,
                "revenue": revenue,
                "cogs": int(revenue * 0.58),
                "gp": int(revenue * 0.42),
                "oi": int(revenue * 0.28),
                "ni": net_income,
                "eps_b": eps_basic,
                "eps_d": eps_diluted,
                "shares": shares,
                "ta": int(base_revenue * 3.5 * growth),
                "tl": int(base_revenue * 2.5 * growth),
                "eq": int(base_revenue * 1.0 * growth),
                "std": int(base_revenue * 0.05 * growth),
                "ltd": int(base_revenue * 0.8 * growth),
                "cash": int(base_revenue * 0.3 * growth),
                "ocf": int(net_income * 1.3),
                "capex": int(-net_income * 0.4),
                "fcf": int(net_income * 0.9),
            },
        )
    await session.commit()


async def seed_stock_prices(session, ticker, days=300):
    """Insert synthetic daily stock price data.

    Generates a random-walk price series starting at $150.
    Needed for: risk_metrics, technical_indicators.
    """
    random.seed(hash(ticker) % 2**32)
    price = 150.0
    today = date.today()

    for i in range(days, 0, -1):
        day = today - timedelta(days=i)
        # Skip weekends
        if day.weekday() >= 5:
            continue

        # Random walk with slight upward drift
        change = random.gauss(0.0005, 0.015)
        price *= 1 + change
        price = max(price, 1.0)

        high = price * (1 + abs(random.gauss(0, 0.01)))
        low = price * (1 - abs(random.gauss(0, 0.01)))
        volume = random.randint(10_000_000, 100_000_000)

        await session.execute(
            text(
                "INSERT INTO stock_prices"
                " (time, ticker, open, high, low, close, volume)"
                " VALUES (:t, :ticker, :o, :h, :l, :c, :v)"
                " ON CONFLICT DO NOTHING"
            ),
            {
                "t": day,
                "ticker": ticker,
                "o": round(price * (1 + random.gauss(0, 0.005)), 2),
                "h": round(high, 2),
                "l": round(low, 2),
                "c": round(price, 2),
                "v": volume,
            },
        )
    await session.commit()


async def seed_benchmark_returns(session, days=300):
    """Insert SPY benchmark return data.

    Needed for: risk_metrics (beta, alpha calculation).
    """
    random.seed(42)
    price = 450.0
    today = date.today()

    for i in range(days, 0, -1):
        day = today - timedelta(days=i)
        if day.weekday() >= 5:
            continue

        daily_return = random.gauss(0.0004, 0.01)
        price *= 1 + daily_return

        await session.execute(
            text(
                "INSERT INTO benchmark_returns"
                " (time, symbol, close, daily_return)"
                " VALUES (:t, 'SPY', :c, :r)"
                " ON CONFLICT DO NOTHING"
            ),
            {
                "t": day,
                "c": round(price, 2),
                "r": round(daily_return, 6),
            },
        )
    await session.commit()


async def seed_treasury_rates(session, days=300):
    """Insert treasury rate data.

    Needed for: fair_value (risk-free rate), risk_metrics.
    """
    today = date.today()

    for i in range(days, 0, -1):
        day = today - timedelta(days=i)
        if day.weekday() >= 5:
            continue

        # Slowly varying rates
        base = 4.5 + 0.3 * math.sin(i / 60.0)

        await session.execute(
            text(
                "INSERT INTO treasury_rates"
                " (date, rate_1m, rate_3m, rate_6m,"
                "  rate_1y, rate_2y, rate_5y, rate_10y)"
                " VALUES (:d, :r1m, :r3m, :r6m,"
                "  :r1y, :r2y, :r5y, :r10y)"
                " ON CONFLICT DO NOTHING"
            ),
            {
                "d": day,
                "r1m": round(base - 0.5, 2),
                "r3m": round(base - 0.3, 2),
                "r6m": round(base - 0.1, 2),
                "r1y": round(base, 2),
                "r2y": round(base + 0.1, 2),
                "r5y": round(base + 0.2, 2),
                "r10y": round(base + 0.3, 2),
            },
        )
    await session.commit()


async def seed_ttm_financials(session, ticker):
    """Insert pre-calculated TTM financials.

    Needed when testing pipelines that depend on TTM data
    (fundamental_metrics, fair_value) without running the
    TTM calculator first.
    """
    today = date.today()
    await session.execute(
        text(
            "INSERT INTO ttm_financials"
            " (ticker, calculation_date,"
            "  ttm_period_start, ttm_period_end,"
            "  revenue, cost_of_revenue, gross_profit,"
            "  operating_income, net_income,"
            "  eps_basic, eps_diluted,"
            "  shares_outstanding,"
            "  total_assets, total_liabilities,"
            "  shareholders_equity,"
            "  short_term_debt, long_term_debt,"
            "  cash_and_equivalents,"
            "  operating_cash_flow, capex,"
            "  free_cash_flow,"
            "  quarters_included)"
            " VALUES"
            " (:ticker, :calc_date,"
            "  :start, :end,"
            "  :rev, :cogs, :gp,"
            "  :oi, :ni,"
            "  :eps_b, :eps_d,"
            "  :shares,"
            "  :ta, :tl,"
            "  :eq,"
            "  :std, :ltd,"
            "  :cash,"
            "  :ocf, :capex,"
            "  :fcf,"
            "  :quarters)"
            " ON CONFLICT (ticker, calculation_date) DO NOTHING"
        ),
        {
            "ticker": ticker,
            "calc_date": today,
            "start": today - timedelta(days=365),
            "end": today - timedelta(days=1),
            "rev": 95_000_000_000,
            "cogs": 55_000_000_000,
            "gp": 40_000_000_000,
            "oi": 27_000_000_000,
            "ni": 21_000_000_000,
            "eps_b": 1.3548,
            "eps_d": 1.3277,
            "shares": 15_500_000_000,
            "ta": 330_000_000_000,
            "tl": 240_000_000_000,
            "eq": 90_000_000_000,
            "std": 5_000_000_000,
            "ltd": 75_000_000_000,
            "cash": 28_000_000_000,
            "ocf": 27_000_000_000,
            "capex": -8_000_000_000,
            "fcf": 19_000_000_000,
            "quarters": '["Q1","Q2","Q3","Q4"]',
        },
    )
    await session.commit()


async def seed_risk_metrics(session, ticker):
    """Insert pre-calculated risk metrics.

    Needed when testing pipelines that depend on risk data
    (fair_value needs beta for WACC).
    """
    today = date.today()
    await session.execute(
        text(
            "INSERT INTO risk_metrics"
            " (time, ticker, period,"
            "  beta, alpha, sharpe_ratio, sortino_ratio,"
            "  max_drawdown, volatility,"
            "  var_95, cvar_95,"
            "  tracking_error, information_ratio,"
            "  downside_deviation)"
            " VALUES"
            " (:t, :ticker, '1Y',"
            "  :beta, :alpha, :sharpe, :sortino,"
            "  :mdd, :vol,"
            "  :var95, :cvar95,"
            "  :te, :ir,"
            "  :dd)"
            " ON CONFLICT DO NOTHING"
        ),
        {
            "t": today,
            "ticker": ticker,
            "beta": 1.15,
            "alpha": 0.02,
            "sharpe": 1.5,
            "sortino": 2.1,
            "mdd": -0.18,
            "vol": 0.22,
            "var95": -0.028,
            "cvar95": -0.038,
            "te": 0.05,
            "ir": 0.8,
            "dd": 0.12,
        },
    )
    await session.commit()


async def seed_all(session, tickers=None):
    """Seed all data for full pipeline chain tests.

    Calls all seed functions for a comprehensive test dataset.
    """
    target_tickers = tickers or [c[0] for c in COMPANIES[:5]]
    await seed_companies(session, target_tickers)
    await seed_treasury_rates(session, days=300)
    await seed_benchmark_returns(session, days=300)

    for ticker in target_tickers:
        await seed_financials(session, ticker, quarters=8)
        await seed_stock_prices(session, ticker, days=300)
        await seed_ttm_financials(session, ticker)
        await seed_risk_metrics(session, ticker)
