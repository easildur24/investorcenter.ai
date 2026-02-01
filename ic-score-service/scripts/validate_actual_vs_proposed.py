#!/usr/bin/env python3
"""
IC Score Validation: Actual Current Scores vs Proposed Design

Uses ACTUAL current IC Scores from UI to compare against proposed methodology.
"""

import numpy as np
from typing import Dict, Tuple

# ACTUAL current scores from UI (January 2026)
ACTUAL_CURRENT_SCORES = {
    "AAPL": {"score": 42, "rating": "Underperform"},
    "NVDA": {"score": 61, "rating": "Hold"},
    "JPM": {"score": 45, "rating": "Underperform"},
    "BRK.B": {"score": 49, "rating": "Underperform"},
    "JNJ": {"score": 62, "rating": "Hold"},
    "UNH": {"score": None, "rating": "Unavailable"},
    "AMZN": {"score": 60, "rating": "Hold"},
    "WMT": {"score": 43, "rating": "Underperform"},
    "XOM": {"score": 49, "rating": "Underperform"},
    "CAT": {"score": 47, "rating": "Underperform"},
}

# Realistic financial data for these stocks
STOCK_DATA = [
    {
        "ticker": "AAPL", "name": "Apple", "sector": "Technology", "type": "Mature Tech",
        "pe_ratio": 28.5, "pb_ratio": 45.2, "ps_ratio": 7.8,
        "revenue_growth": 8.5, "eps_growth": 12.3,
        "net_margin": 25.3, "roe": 147.0, "roa": 28.5, "gross_margin": 45.0,
        "debt_to_equity": 1.87, "current_ratio": 0.99,
        "return_1m": 3.2, "return_3m": 8.5, "return_6m": 15.2, "return_12m": 28.4,
        "rsi": 58, "analyst_buy": 35, "analyst_hold": 10, "analyst_sell": 2,
        "insider_net_buying": -50000, "num_institutions": 4500, "news_sentiment": 72,
        "price": 185.0, "avg_price_target": 210.0,
    },
    {
        "ticker": "NVDA", "name": "NVIDIA", "sector": "Technology", "type": "Hypergrowth",
        "pe_ratio": 65.0, "pb_ratio": 52.0, "ps_ratio": 28.5,
        "revenue_growth": 122.0, "eps_growth": 580.0,
        "net_margin": 55.0, "roe": 91.0, "roa": 45.0, "gross_margin": 75.0,
        "debt_to_equity": 0.41, "current_ratio": 4.2,
        "return_1m": 12.5, "return_3m": 25.0, "return_6m": 45.0, "return_12m": 185.0,
        "rsi": 72, "analyst_buy": 52, "analyst_hold": 8, "analyst_sell": 1,
        "insider_net_buying": -200000, "num_institutions": 3800, "news_sentiment": 85,
        "price": 480.0, "avg_price_target": 550.0,
    },
    {
        "ticker": "JPM", "name": "JPMorgan", "sector": "Financial Services", "type": "Value",
        "pe_ratio": 11.2, "pb_ratio": 1.65, "ps_ratio": 3.2,
        "revenue_growth": 12.0, "eps_growth": 18.0,
        "net_margin": 32.0, "roe": 15.0, "roa": 1.2, "gross_margin": 65.0,
        "debt_to_equity": 1.2, "current_ratio": 0.9,
        "return_1m": 2.1, "return_3m": 5.5, "return_6m": 12.0, "return_12m": 35.0,
        "rsi": 55, "analyst_buy": 18, "analyst_hold": 8, "analyst_sell": 1,
        "insider_net_buying": 25000, "num_institutions": 3200, "news_sentiment": 65,
        "price": 195.0, "avg_price_target": 215.0,
    },
    {
        "ticker": "BRK.B", "name": "Berkshire Hathaway", "sector": "Financial Services", "type": "Value",
        "pe_ratio": 9.5, "pb_ratio": 1.45, "ps_ratio": 2.1,
        "revenue_growth": 5.0, "eps_growth": 8.0,
        "net_margin": 18.0, "roe": 12.0, "roa": 4.5, "gross_margin": 40.0,
        "debt_to_equity": 0.25, "current_ratio": 2.8,
        "return_1m": 1.5, "return_3m": 4.0, "return_6m": 8.0, "return_12m": 22.0,
        "rsi": 48, "analyst_buy": 5, "analyst_hold": 3, "analyst_sell": 0,
        "insider_net_buying": 0, "num_institutions": 2800, "news_sentiment": 60,
        "price": 410.0, "avg_price_target": 450.0,
    },
    {
        "ticker": "JNJ", "name": "Johnson & Johnson", "sector": "Healthcare", "type": "Defensive",
        "pe_ratio": 15.5, "pb_ratio": 5.8, "ps_ratio": 4.2,
        "revenue_growth": 3.5, "eps_growth": 5.2,
        "net_margin": 20.5, "roe": 22.0, "roa": 9.5, "gross_margin": 68.0,
        "debt_to_equity": 0.45, "current_ratio": 1.2,
        "return_1m": -1.2, "return_3m": 0.5, "return_6m": 2.0, "return_12m": 5.0,
        "rsi": 42, "analyst_buy": 12, "analyst_hold": 15, "analyst_sell": 3,
        "insider_net_buying": 5000, "num_institutions": 4200, "news_sentiment": 55,
        "price": 155.0, "avg_price_target": 175.0,
    },
    {
        "ticker": "UNH", "name": "UnitedHealth", "sector": "Healthcare", "type": "Growth",
        "pe_ratio": 19.5, "pb_ratio": 6.2, "ps_ratio": 1.4,
        "revenue_growth": 14.5, "eps_growth": 12.0,
        "net_margin": 6.5, "roe": 25.0, "roa": 8.0, "gross_margin": 24.0,
        "debt_to_equity": 0.72, "current_ratio": 0.85,
        "return_1m": 4.5, "return_3m": 8.0, "return_6m": 10.0, "return_12m": 18.0,
        "rsi": 62, "analyst_buy": 25, "analyst_hold": 5, "analyst_sell": 1,
        "insider_net_buying": -15000, "num_institutions": 3500, "news_sentiment": 68,
        "price": 520.0, "avg_price_target": 580.0,
    },
    {
        "ticker": "AMZN", "name": "Amazon", "sector": "Consumer Cyclical", "type": "Growth",
        "pe_ratio": 42.0, "pb_ratio": 8.5, "ps_ratio": 3.2,
        "revenue_growth": 12.5, "eps_growth": 95.0,
        "net_margin": 7.8, "roe": 20.0, "roa": 6.5, "gross_margin": 47.0,
        "debt_to_equity": 0.58, "current_ratio": 1.05,
        "return_1m": 5.2, "return_3m": 12.0, "return_6m": 22.0, "return_12m": 48.0,
        "rsi": 65, "analyst_buy": 58, "analyst_hold": 4, "analyst_sell": 0,
        "insider_net_buying": -180000, "num_institutions": 4800, "news_sentiment": 75,
        "price": 185.0, "avg_price_target": 220.0,
    },
    {
        "ticker": "WMT", "name": "Walmart", "sector": "Consumer Defensive", "type": "Value",
        "pe_ratio": 28.0, "pb_ratio": 6.8, "ps_ratio": 0.75,
        "revenue_growth": 5.5, "eps_growth": 8.2,
        "net_margin": 2.5, "roe": 20.0, "roa": 6.8, "gross_margin": 24.5,
        "debt_to_equity": 0.52, "current_ratio": 0.82,
        "return_1m": 2.8, "return_3m": 6.5, "return_6m": 12.0, "return_12m": 35.0,
        "rsi": 58, "analyst_buy": 32, "analyst_hold": 8, "analyst_sell": 1,
        "insider_net_buying": 0, "num_institutions": 3600, "news_sentiment": 62,
        "price": 165.0, "avg_price_target": 180.0,
    },
    {
        "ticker": "XOM", "name": "Exxon Mobil", "sector": "Energy", "type": "Value/Dividend",
        "pe_ratio": 13.5, "pb_ratio": 2.1, "ps_ratio": 1.2,
        "revenue_growth": -2.5, "eps_growth": -8.0,
        "net_margin": 10.5, "roe": 16.0, "roa": 8.5, "gross_margin": 32.0,
        "debt_to_equity": 0.18, "current_ratio": 1.45,
        "return_1m": -3.5, "return_3m": -5.0, "return_6m": -8.0, "return_12m": -2.0,
        "rsi": 38, "analyst_buy": 15, "analyst_hold": 12, "analyst_sell": 4,
        "insider_net_buying": 12000, "num_institutions": 3100, "news_sentiment": 48,
        "price": 108.0, "avg_price_target": 125.0,
    },
    {
        "ticker": "CAT", "name": "Caterpillar", "sector": "Industrials", "type": "Cyclical",
        "pe_ratio": 15.8, "pb_ratio": 8.5, "ps_ratio": 2.5,
        "revenue_growth": 8.0, "eps_growth": 15.0,
        "net_margin": 16.0, "roe": 55.0, "roa": 12.0, "gross_margin": 35.0,
        "debt_to_equity": 1.85, "current_ratio": 1.35,
        "return_1m": 4.0, "return_3m": 10.0, "return_6m": 18.0, "return_12m": 42.0,
        "rsi": 60, "analyst_buy": 18, "analyst_hold": 10, "analyst_sell": 2,
        "insider_net_buying": -8000, "num_institutions": 2900, "news_sentiment": 65,
        "price": 355.0, "avg_price_target": 390.0,
    },
]

# Sector medians for relative comparison
SECTOR_MEDIANS = {
    "Technology": {"pe": 28, "pb": 8, "ps": 6, "net_margin": 18, "roe": 25, "growth": 15},
    "Financial Services": {"pe": 12, "pb": 1.5, "ps": 3, "net_margin": 25, "roe": 12, "growth": 8},
    "Healthcare": {"pe": 22, "pb": 5, "ps": 3, "net_margin": 12, "roe": 18, "growth": 8},
    "Consumer Cyclical": {"pe": 25, "pb": 6, "ps": 2, "net_margin": 8, "roe": 18, "growth": 10},
    "Consumer Defensive": {"pe": 22, "pb": 5, "ps": 1, "net_margin": 5, "roe": 20, "growth": 5},
    "Energy": {"pe": 12, "pb": 1.8, "ps": 1, "net_margin": 8, "roe": 15, "growth": 0},
    "Industrials": {"pe": 18, "pb": 5, "ps": 2, "net_margin": 10, "roe": 25, "growth": 6},
}

# Proposed weights
PROPOSED_WEIGHTS = {
    'growth': 0.15,
    'profitability': 0.13,
    'financial_health': 0.12,
    'value': 0.15,
    'intrinsic_value': 0.15,
    'smart_money': 0.12,
    'momentum': 0.10,
    'technical': 0.08,
}

LIFECYCLE_ADJUSTMENTS = {
    'hypergrowth': {'growth': 1.4, 'profitability': 0.6, 'value': 0.5, 'intrinsic_value': 0.5},
    'growth': {'growth': 1.2, 'profitability': 0.9, 'value': 0.8},
    'mature': {'profitability': 1.1, 'value': 1.1, 'financial_health': 1.1},
    'value': {'value': 1.3, 'profitability': 1.1, 'growth': 0.7, 'intrinsic_value': 1.2},
    'turnaround': {'financial_health': 1.3, 'momentum': 1.2, 'value': 1.2},
}


def classify_lifecycle(data: Dict) -> str:
    revenue_growth = data.get('revenue_growth', 0) or 0
    net_margin = data.get('net_margin', 0) or 0
    pe_ratio = data.get('pe_ratio', 20) or 20

    if revenue_growth > 50:
        return 'hypergrowth'
    elif revenue_growth > 15:
        return 'growth'
    elif revenue_growth < 0:
        return 'turnaround'
    elif pe_ratio < 15 and net_margin > 5:
        return 'value'
    else:
        return 'mature'


def sector_percentile(value: float, sector_median: float, lower_is_better: bool = False) -> float:
    ratio = value / sector_median if sector_median > 0 else 1
    if lower_is_better:
        if ratio <= 0.5:
            pct = 95
        elif ratio <= 0.75:
            pct = 75 + (0.75 - ratio) * 80
        elif ratio <= 1.0:
            pct = 50 + (1.0 - ratio) * 100
        elif ratio <= 1.5:
            pct = 50 - (ratio - 1.0) * 60
        else:
            pct = max(5, 20 - (ratio - 1.5) * 30)
    else:
        if ratio >= 2.0:
            pct = 95
        elif ratio >= 1.5:
            pct = 75 + (ratio - 1.5) * 40
        elif ratio >= 1.0:
            pct = 50 + (ratio - 1.0) * 50
        elif ratio >= 0.5:
            pct = 25 + (ratio - 0.5) * 50
        else:
            pct = max(5, ratio * 50)
    return max(0, min(100, pct))


def calculate_proposed_score(data: Dict) -> Tuple[float, Dict[str, float], str]:
    """Calculate IC Score using PROPOSED methodology."""
    scores = {}
    sector = data.get('sector', 'Technology')
    medians = SECTOR_MEDIANS.get(sector, SECTOR_MEDIANS['Technology'])
    lifecycle = classify_lifecycle(data)

    # VALUE (Sector-relative)
    pe_pct = sector_percentile(data['pe_ratio'], medians['pe'], lower_is_better=True)
    pb_pct = sector_percentile(data['pb_ratio'], medians['pb'], lower_is_better=True)
    ps_pct = sector_percentile(data['ps_ratio'], medians['ps'], lower_is_better=True)
    scores['value'] = pe_pct * 0.4 + pb_pct * 0.3 + ps_pct * 0.3

    # INTRINSIC VALUE
    if data.get('price') and data.get('avg_price_target'):
        upside = (data['avg_price_target'] - data['price']) / data['price'] * 100
        scores['intrinsic_value'] = max(0, min(100, 50 + upside * 2))

    # GROWTH (Sector-adjusted)
    sector_growth_expectation = medians.get('growth', 10)
    rev_growth = data.get('revenue_growth', 0)
    eps_growth = data.get('eps_growth', 0)
    adjusted_rev = rev_growth - sector_growth_expectation
    rev_score = max(0, min(100, 50 + adjusted_rev * 2.0))
    eps_score = max(0, min(100, 50 + min(eps_growth, 100) * 0.4))
    scores['growth'] = rev_score * 0.6 + eps_score * 0.4

    # PROFITABILITY (Sector-relative)
    margin_pct = sector_percentile(data['net_margin'], medians['net_margin'])
    roe_pct = sector_percentile(min(data['roe'], 100), medians['roe'])  # Cap extreme ROE
    gross_margin_score = min(100, data.get('gross_margin', 30) * 1.5)
    scores['profitability'] = margin_pct * 0.35 + roe_pct * 0.35 + gross_margin_score * 0.30

    # FINANCIAL HEALTH
    de = data.get('debt_to_equity', 0)
    cr = data.get('current_ratio', 2)
    de_score = max(0, min(100, 100 - de * 40))
    if cr < 1:
        cr_score = cr * 60
    elif cr <= 2:
        cr_score = 60 + (cr - 1) * 40
    else:
        cr_score = max(50, 100 - (cr - 2) * 15)
    scores['financial_health'] = de_score * 0.5 + cr_score * 0.5

    # SMART MONEY
    total_analysts = data.get('analyst_buy', 0) + data.get('analyst_hold', 0) + data.get('analyst_sell', 0)
    analyst_score = (data['analyst_buy'] * 100 + data['analyst_hold'] * 50) / total_analysts if total_analysts > 0 else 50
    net = data.get('insider_net_buying', 0)
    insider_score = max(0, min(100, 50 + net / 1000))
    num_inst = data.get('num_institutions', 0)
    inst_score = min(100, (num_inst / 40) * 100)
    scores['smart_money'] = analyst_score * 0.4 + insider_score * 0.35 + inst_score * 0.25

    # MOMENTUM
    returns = {
        'return_1m': (data.get('return_1m', 0), 0.2),
        'return_3m': (data.get('return_3m', 0), 0.3),
        'return_6m': (data.get('return_6m', 0), 0.3),
        'return_12m': (data.get('return_12m', 0), 0.2),
    }
    mom_score = sum(max(0, min(100, 50 + ret * 2.0)) * weight for ret, weight in returns.values())
    scores['momentum'] = mom_score

    # TECHNICAL
    rsi = data.get('rsi', 50)
    if rsi <= 30:
        tech_score = 75 + (30 - rsi)
    elif rsi >= 70:
        tech_score = 25 - (rsi - 70)
    else:
        tech_score = 50
    scores['technical'] = max(0, min(100, tech_score))

    # Apply lifecycle adjustments
    adjusted_weights = PROPOSED_WEIGHTS.copy()
    if lifecycle in LIFECYCLE_ADJUSTMENTS:
        for factor, mult in LIFECYCLE_ADJUSTMENTS[lifecycle].items():
            if factor in adjusted_weights:
                adjusted_weights[factor] *= mult
    total = sum(adjusted_weights.values())
    adjusted_weights = {k: v / total for k, v in adjusted_weights.items()}

    overall = sum(scores.get(f, 50) * adjusted_weights.get(f, 0) for f in adjusted_weights)
    return overall, scores, lifecycle


def get_rating(score: float) -> str:
    if score >= 80:
        return "Strong Buy"
    elif score >= 65:
        return "Buy"
    elif score >= 50:
        return "Hold"
    elif score >= 35:
        return "Underperform"
    else:
        return "Sell"


def main():
    print("=" * 120)
    print("IC SCORE VALIDATION: Actual Current Scores vs Proposed Design")
    print("=" * 120)
    print()
    print("Using ACTUAL current IC Scores from UI (not mock calculations)")
    print()

    results = []

    for stock in STOCK_DATA:
        ticker = stock['ticker']
        actual = ACTUAL_CURRENT_SCORES.get(ticker, {})
        current_score = actual.get('score')
        current_rating = actual.get('rating', 'N/A')

        if current_score is None:
            continue

        proposed_score, proposed_factors, lifecycle = calculate_proposed_score(stock)

        results.append({
            'ticker': ticker,
            'name': stock['name'],
            'sector': stock['sector'],
            'lifecycle': lifecycle,
            'current_score': current_score,
            'current_rating': current_rating,
            'proposed_score': proposed_score,
            'proposed_rating': get_rating(proposed_score),
            'delta': proposed_score - current_score,
            'proposed_factors': proposed_factors,
        })

    # Print comparison table
    print("=" * 120)
    print("COMPARISON: ACTUAL CURRENT vs PROPOSED")
    print("=" * 120)
    print()
    print(f"{'Ticker':<7} {'Name':<22} {'Sector':<18} {'Lifecycle':<12} {'CURRENT':>12} {'PROPOSED':>12} {'Delta':>8} {'Rating Change':<25}")
    print("-" * 120)

    for r in results:
        rating_change = ""
        if r['current_rating'] != r['proposed_rating']:
            rating_change = f"{r['current_rating']} → {r['proposed_rating']}"

        print(f"{r['ticker']:<7} {r['name']:<22} {r['sector']:<18} {r['lifecycle']:<12} "
              f"{r['current_score']:>5.0f} ({r['current_rating'][:5]:>5}) "
              f"{r['proposed_score']:>5.0f} ({r['proposed_rating'][:5]:>5}) "
              f"{r['delta']:>+6.0f}   {rating_change}")

    # Summary
    print()
    print("=" * 120)
    print("SUMMARY")
    print("=" * 120)
    print()

    avg_current = np.mean([r['current_score'] for r in results])
    avg_proposed = np.mean([r['proposed_score'] for r in results])
    avg_delta = np.mean([r['delta'] for r in results])

    print(f"Average CURRENT Score:  {avg_current:.1f}")
    print(f"Average PROPOSED Score: {avg_proposed:.1f}")
    print(f"Average Delta:          {avg_delta:+.1f}")
    print()

    # Rating distribution
    print("Rating Distribution:")
    print()
    print(f"{'Rating':<15} {'Current':<12} {'Proposed':<12}")
    print("-" * 40)
    for rating in ["Strong Buy", "Buy", "Hold", "Underperform", "Sell"]:
        current_count = sum(1 for r in results if r['current_rating'] == rating)
        proposed_count = sum(1 for r in results if r['proposed_rating'] == rating)
        print(f"{rating:<15} {current_count:<12} {proposed_count:<12}")

    # Rating changes
    print()
    upgrades = [(r['ticker'], r['current_rating'], r['proposed_rating']) for r in results if r['delta'] > 5]
    downgrades = [(r['ticker'], r['current_rating'], r['proposed_rating']) for r in results if r['delta'] < -5]

    if upgrades:
        print(f"Upgrades ({len(upgrades)}):")
        for ticker, old, new in upgrades:
            delta = next(r['delta'] for r in results if r['ticker'] == ticker)
            print(f"  {ticker}: {old} → {new} (+{delta:.0f})")

    if downgrades:
        print(f"\nDowngrades ({len(downgrades)}):")
        for ticker, old, new in downgrades:
            delta = next(r['delta'] for r in results if r['ticker'] == ticker)
            print(f"  {ticker}: {old} → {new} ({delta:.0f})")

    # Key insights
    print()
    print("=" * 120)
    print("KEY INSIGHTS")
    print("=" * 120)
    print()
    print("PROBLEM WITH CURRENT SCORES:")
    print("  - 6 of 9 stocks rated 'Underperform' (AAPL, JPM, BRK.B, WMT, XOM, CAT)")
    print("  - These are all high-quality, blue-chip companies")
    print("  - Current methodology appears too harsh/pessimistic")
    print()
    print("PROPOSED METHODOLOGY IMPROVEMENTS:")
    print("  - Sector-relative scoring: AAPL P/E=28 is fair for Tech (median=28)")
    print("  - Lifecycle awareness: NVDA's high P/E is acceptable for hypergrowth")
    print("  - Intrinsic value factor: Uses analyst price targets for fair value")
    print("  - Smart Money combination: Analyst + Insider + Institutional signals")
    print()

    # Factor breakdown for interesting cases
    print("=" * 120)
    print("PROPOSED FACTOR SCORES")
    print("=" * 120)
    print()
    print(f"{'Ticker':<7} {'Value':>8} {'Intrins':>8} {'Growth':>8} {'Profit':>8} {'Health':>8} {'Smart$':>8} {'Moment':>8} {'Tech':>8}")
    print("-" * 80)
    for r in results:
        pf = r['proposed_factors']
        print(f"{r['ticker']:<7} {pf.get('value',0):>8.0f} {pf.get('intrinsic_value',0):>8.0f} "
              f"{pf.get('growth',0):>8.0f} {pf.get('profitability',0):>8.0f} "
              f"{pf.get('financial_health',0):>8.0f} {pf.get('smart_money',0):>8.0f} "
              f"{pf.get('momentum',0):>8.0f} {pf.get('technical',0):>8.0f}")


if __name__ == '__main__':
    main()
