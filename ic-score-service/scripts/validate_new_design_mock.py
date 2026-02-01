#!/usr/bin/env python3
"""
IC Score Validation Script: Current vs Proposed Design (Mock Data)

Uses realistic sample data to demonstrate the difference between
current and proposed IC Score methodologies.
"""

import numpy as np
from typing import Dict, Tuple

# Realistic sample data for 10 diverse stocks (based on typical market data)
SAMPLE_STOCKS = [
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

# Sector median values for relative comparison
SECTOR_MEDIANS = {
    "Technology": {"pe": 28, "pb": 8, "ps": 6, "net_margin": 18, "roe": 25, "growth": 15},
    "Financial Services": {"pe": 12, "pb": 1.5, "ps": 3, "net_margin": 25, "roe": 12, "growth": 8},
    "Healthcare": {"pe": 22, "pb": 5, "ps": 3, "net_margin": 12, "roe": 18, "growth": 8},
    "Consumer Cyclical": {"pe": 25, "pb": 6, "ps": 2, "net_margin": 8, "roe": 18, "growth": 10},
    "Consumer Defensive": {"pe": 22, "pb": 5, "ps": 1, "net_margin": 5, "roe": 20, "growth": 5},
    "Energy": {"pe": 12, "pb": 1.8, "ps": 1, "net_margin": 8, "roe": 15, "growth": 0},
    "Industrials": {"pe": 18, "pb": 5, "ps": 2, "net_margin": 10, "roe": 25, "growth": 6},
}

# Current methodology weights
CURRENT_WEIGHTS = {
    'value': 0.12,
    'growth': 0.15,
    'profitability': 0.12,
    'financial_health': 0.10,
    'momentum': 0.08,
    'analyst_consensus': 0.10,
    'insider_activity': 0.08,
    'institutional': 0.10,
    'news_sentiment': 0.07,
    'technical': 0.08,
}

# Proposed methodology weights
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

# Lifecycle weight adjustments
LIFECYCLE_ADJUSTMENTS = {
    'hypergrowth': {'growth': 1.4, 'profitability': 0.6, 'value': 0.5, 'intrinsic_value': 0.5},
    'growth': {'growth': 1.2, 'profitability': 0.9, 'value': 0.8},
    'mature': {'profitability': 1.1, 'value': 1.1, 'financial_health': 1.1},
    'value': {'value': 1.3, 'profitability': 1.1, 'growth': 0.7, 'intrinsic_value': 1.2},
    'turnaround': {'financial_health': 1.3, 'momentum': 1.2, 'value': 1.2},
}


def classify_lifecycle(data: Dict) -> str:
    """Classify company into lifecycle stage."""
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
    """Calculate approximate sector percentile."""
    ratio = value / sector_median if sector_median > 0 else 1

    if lower_is_better:
        # Lower ratio = higher percentile (e.g., P/E)
        if ratio <= 0.5:
            pct = 95
        elif ratio <= 0.75:
            pct = 75
        elif ratio <= 1.0:
            pct = 50 + (1.0 - ratio) * 100
        elif ratio <= 1.5:
            pct = 50 - (ratio - 1.0) * 60
        else:
            pct = max(5, 20 - (ratio - 1.5) * 30)
    else:
        # Higher ratio = higher percentile (e.g., margin)
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


def calculate_current_score(data: Dict) -> Tuple[float, Dict[str, float]]:
    """Calculate IC Score using CURRENT methodology (absolute benchmarks)."""
    scores = {}

    # Value Score (absolute benchmarks: P/E=15, P/B=2, P/S=2)
    pe = data.get('pe_ratio', 15)
    pb = data.get('pb_ratio', 2)
    ps = data.get('ps_ratio', 2)
    pe_score = max(0, min(100, 100 - (pe - 15) * 2.0))
    pb_score = max(0, min(100, 100 - (pb - 2) * 20))
    ps_score = max(0, min(100, 100 - (ps - 2) * 20))
    scores['value'] = np.mean([pe_score, pb_score, ps_score])

    # Growth Score
    rev_growth = data.get('revenue_growth', 0)
    eps_growth = data.get('eps_growth', 0)
    rev_score = max(0, min(100, 50 + rev_growth * 2.5))
    eps_score = max(0, min(100, 50 + eps_growth * 2.5))
    scores['growth'] = np.mean([rev_score, eps_score])

    # Profitability Score
    margin = data.get('net_margin', 0)
    roe = data.get('roe', 0)
    roa = data.get('roa', 0)
    margin_score = max(0, min(100, margin * 5))
    roe_score = max(0, min(100, roe * 5))
    roa_score = max(0, min(100, roa * 10))
    scores['profitability'] = np.mean([margin_score, roe_score, roa_score])

    # Financial Health Score
    de = data.get('debt_to_equity', 0)
    cr = data.get('current_ratio', 2)
    de_score = max(0, min(100, 100 - de * 50))
    cr_score = max(0, min(100, 100 - abs(cr - 2.0) * 40))
    scores['financial_health'] = np.mean([de_score, cr_score])

    # Momentum Score
    returns = [data.get(f'return_{p}', 0) for p in ['1m', '3m', '6m', '12m']]
    momentum_scores = [max(0, min(100, 50 + r * 2.5)) for r in returns]
    scores['momentum'] = np.mean(momentum_scores)

    # Technical Score (RSI-based)
    rsi = data.get('rsi', 50)
    if rsi < 50:
        rsi_score = max(0, (rsi - 30) * 2.5)
    else:
        rsi_score = min(100, 50 + (rsi - 50) * 2.5)
    scores['technical'] = rsi_score

    # Analyst Consensus
    total = data.get('analyst_buy', 0) + data.get('analyst_hold', 0) + data.get('analyst_sell', 0)
    if total > 0:
        scores['analyst_consensus'] = (data['analyst_buy'] * 100 + data['analyst_hold'] * 50) / total

    # Insider Activity
    net = data.get('insider_net_buying', 0)
    scores['insider_activity'] = max(0, min(100, 50 + net / 2000))

    # Institutional
    num_inst = data.get('num_institutions', 0)
    scores['institutional'] = min(100, (num_inst / 100) * 100)

    # News Sentiment
    scores['news_sentiment'] = max(0, min(100, data.get('news_sentiment', 50)))

    # Weighted average
    total_weight = sum(CURRENT_WEIGHTS.values())
    overall = sum(scores.get(f, 50) * CURRENT_WEIGHTS[f] for f in CURRENT_WEIGHTS) / total_weight

    return overall, scores


def calculate_proposed_score(data: Dict) -> Tuple[float, Dict[str, float], str]:
    """Calculate IC Score using PROPOSED methodology (sector-relative, lifecycle-aware)."""
    scores = {}
    sector = data.get('sector', 'Technology')
    medians = SECTOR_MEDIANS.get(sector, SECTOR_MEDIANS['Technology'])

    # Classify lifecycle
    lifecycle = classify_lifecycle(data)

    # VALUE SCORE (Sector-relative)
    pe_pct = sector_percentile(data['pe_ratio'], medians['pe'], lower_is_better=True)
    pb_pct = sector_percentile(data['pb_ratio'], medians['pb'], lower_is_better=True)
    ps_pct = sector_percentile(data['ps_ratio'], medians['ps'], lower_is_better=True)
    scores['value'] = pe_pct * 0.4 + pb_pct * 0.3 + ps_pct * 0.3

    # INTRINSIC VALUE (Price vs Target)
    if data.get('price') and data.get('avg_price_target'):
        upside = (data['avg_price_target'] - data['price']) / data['price'] * 100
        scores['intrinsic_value'] = max(0, min(100, 50 + upside * 2))

    # GROWTH SCORE (Sector-adjusted expectations)
    sector_growth_expectation = medians.get('growth', 10)
    rev_growth = data.get('revenue_growth', 0)
    eps_growth = data.get('eps_growth', 0)
    adjusted_rev = rev_growth - sector_growth_expectation
    rev_score = max(0, min(100, 50 + adjusted_rev * 2.0))
    eps_score = max(0, min(100, 50 + eps_growth * 0.5))  # Cap EPS impact
    scores['growth'] = rev_score * 0.6 + eps_score * 0.4

    # PROFITABILITY SCORE (Sector-relative)
    margin_pct = sector_percentile(data['net_margin'], medians['net_margin'])
    roe_pct = sector_percentile(data['roe'], medians['roe'])
    gross_margin_score = min(100, data.get('gross_margin', 30) * 1.5)
    scores['profitability'] = margin_pct * 0.35 + roe_pct * 0.35 + gross_margin_score * 0.30

    # FINANCIAL HEALTH (Similar to current, works across sectors)
    de = data.get('debt_to_equity', 0)
    cr = data.get('current_ratio', 2)
    de_score = max(0, min(100, 100 - de * 40))
    # Optimal range for current ratio
    if cr < 1:
        cr_score = cr * 60
    elif cr <= 2:
        cr_score = 60 + (cr - 1) * 40
    else:
        cr_score = max(50, 100 - (cr - 2) * 15)
    scores['financial_health'] = de_score * 0.5 + cr_score * 0.5

    # SMART MONEY (Combined: Analyst 40% + Insider 35% + Institutional 25%)
    total_analysts = data.get('analyst_buy', 0) + data.get('analyst_hold', 0) + data.get('analyst_sell', 0)
    if total_analysts > 0:
        analyst_score = (data['analyst_buy'] * 100 + data['analyst_hold'] * 50) / total_analysts
    else:
        analyst_score = 50

    net = data.get('insider_net_buying', 0)
    insider_score = max(0, min(100, 50 + net / 1000))

    num_inst = data.get('num_institutions', 0)
    inst_score = min(100, (num_inst / 40) * 100)

    scores['smart_money'] = analyst_score * 0.4 + insider_score * 0.35 + inst_score * 0.25

    # MOMENTUM (Weighted by recency)
    returns = {
        'return_1m': (data.get('return_1m', 0), 0.2),
        'return_3m': (data.get('return_3m', 0), 0.3),
        'return_6m': (data.get('return_6m', 0), 0.3),
        'return_12m': (data.get('return_12m', 0), 0.2),
    }
    mom_score = 0
    for key, (ret, weight) in returns.items():
        mom_score += max(0, min(100, 50 + ret * 2.0)) * weight
    scores['momentum'] = mom_score

    # TECHNICAL (RSI interpretation)
    rsi = data.get('rsi', 50)
    if rsi <= 30:
        tech_score = 75 + (30 - rsi)  # Oversold = bullish
    elif rsi >= 70:
        tech_score = 25 - (rsi - 70)  # Overbought = bearish
    else:
        tech_score = 50  # Neutral
    scores['technical'] = max(0, min(100, tech_score))

    # Apply lifecycle weight adjustments
    adjusted_weights = PROPOSED_WEIGHTS.copy()
    if lifecycle in LIFECYCLE_ADJUSTMENTS:
        for factor, mult in LIFECYCLE_ADJUSTMENTS[lifecycle].items():
            if factor in adjusted_weights:
                adjusted_weights[factor] *= mult

    # Normalize
    total = sum(adjusted_weights.values())
    adjusted_weights = {k: v / total for k, v in adjusted_weights.items()}

    # Calculate weighted average
    overall = sum(scores.get(f, 50) * adjusted_weights.get(f, 0) for f in adjusted_weights)

    return overall, scores, lifecycle


def get_rating(score: float) -> str:
    """Convert score to rating."""
    if score >= 80:
        return "Strong Buy"
    elif score >= 65:
        return "Buy"
    elif score >= 50:
        return "Hold"
    elif score >= 35:
        return "Sell"
    else:
        return "Strong Sell"


def main():
    print("=" * 110)
    print("IC SCORE VALIDATION: Current vs Proposed Design")
    print("=" * 110)
    print()
    print("Key Differences:")
    print("  CURRENT: Uses absolute benchmarks (P/E=15 is 'fair', regardless of sector)")
    print("  PROPOSED: Uses sector-relative percentiles + lifecycle-aware weight adjustment")
    print()

    results = []

    for stock in SAMPLE_STOCKS:
        current_score, current_factors = calculate_current_score(stock)
        proposed_score, proposed_factors, lifecycle = calculate_proposed_score(stock)

        results.append({
            'ticker': stock['ticker'],
            'name': stock['name'],
            'sector': stock['sector'],
            'type': stock['type'],
            'lifecycle': lifecycle,
            'current_score': current_score,
            'current_rating': get_rating(current_score),
            'proposed_score': proposed_score,
            'proposed_rating': get_rating(proposed_score),
            'delta': proposed_score - current_score,
            'current_factors': current_factors,
            'proposed_factors': proposed_factors,
            'pe_ratio': stock['pe_ratio'],
            'revenue_growth': stock['revenue_growth'],
        })

    # Print comparison table
    print("=" * 110)
    print("COMPARISON RESULTS")
    print("=" * 110)
    print()
    print(f"{'Ticker':<7} {'Name':<22} {'Sector':<18} {'Lifecycle':<12} {'Current':>10} {'Proposed':>10} {'Delta':>8} {'Rating Change':<20}")
    print("-" * 110)

    for r in results:
        rating_change = ""
        if r['current_rating'] != r['proposed_rating']:
            rating_change = f"{r['current_rating']} → {r['proposed_rating']}"

        print(f"{r['ticker']:<7} {r['name']:<22} {r['sector']:<18} {r['lifecycle']:<12} "
              f"{r['current_score']:>6.1f} ({r['current_rating'][:4]}) "
              f"{r['proposed_score']:>6.1f} ({r['proposed_rating'][:4]}) "
              f"{r['delta']:>+6.1f}   {rating_change}")

    # Print detailed analysis for interesting cases
    print()
    print("=" * 110)
    print("DETAILED ANALYSIS OF KEY DIFFERENCES")
    print("=" * 110)

    # NVDA - Hypergrowth example
    nvda = next(r for r in results if r['ticker'] == 'NVDA')
    print(f"\n🔍 NVDA (Hypergrowth Tech) - Score changed by {nvda['delta']:+.1f}")
    print("-" * 80)
    print("   CURRENT METHOD PROBLEM:")
    print(f"     - P/E of 65 vs benchmark 15 → Value score penalized heavily")
    print(f"     - Current Value Score: {nvda['current_factors']['value']:.1f}")
    print("   PROPOSED METHOD FIX:")
    print(f"     - P/E of 65 compared to Tech sector median (28) → still expensive but contextual")
    print(f"     - Lifecycle 'hypergrowth' reduces Value weight from 15% to ~7%")
    print(f"     - Growth weight increased from 15% to ~21%")
    print(f"     - Proposed Value Score: {nvda['proposed_factors']['value']:.1f}")

    # JPM - Value example
    jpm = next(r for r in results if r['ticker'] == 'JPM')
    print(f"\n🔍 JPM (Value Financials) - Score changed by {jpm['delta']:+.1f}")
    print("-" * 80)
    print("   CURRENT METHOD PROBLEM:")
    print(f"     - P/E of 11.2 vs benchmark 15 → gets slight value bonus")
    print(f"     - But P/B of 1.65 vs benchmark 2 → another bonus")
    print(f"     - Current Value Score: {jpm['current_factors']['value']:.1f}")
    print("   PROPOSED METHOD FIX:")
    print(f"     - P/E of 11.2 vs Financial Services median (12) → actually slightly cheap")
    print(f"     - P/B of 1.65 vs sector median (1.5) → slightly expensive for financials")
    print(f"     - Lifecycle 'value' boosts Value factor weight by 30%")
    print(f"     - Proposed Value Score: {jpm['proposed_factors']['value']:.1f}")

    # XOM - Turnaround example
    xom = next(r for r in results if r['ticker'] == 'XOM')
    print(f"\n🔍 XOM (Energy Turnaround) - Score changed by {xom['delta']:+.1f}")
    print("-" * 80)
    print("   CURRENT METHOD PROBLEM:")
    print(f"     - Negative revenue growth (-2.5%) → Growth score severely penalized")
    print(f"     - Current Growth Score: {xom['current_factors']['growth']:.1f}")
    print("   PROPOSED METHOD FIX:")
    print(f"     - Compared to Energy sector expectation (0% growth) → not as bad")
    print(f"     - Lifecycle 'turnaround' reduces Growth weight, increases Value & Health")
    print(f"     - Low P/E (13.5) vs Energy median (12) is properly contextualized")
    print(f"     - Proposed Growth Score: {xom['proposed_factors']['growth']:.1f}")

    # Summary statistics
    print()
    print("=" * 110)
    print("SUMMARY STATISTICS")
    print("=" * 110)
    print()

    avg_current = np.mean([r['current_score'] for r in results])
    avg_proposed = np.mean([r['proposed_score'] for r in results])
    avg_delta = np.mean([r['delta'] for r in results])
    std_delta = np.std([r['delta'] for r in results])

    print(f"Average Current Score:  {avg_current:.1f}")
    print(f"Average Proposed Score: {avg_proposed:.1f}")
    print(f"Average Delta:          {avg_delta:+.1f}")
    print(f"Delta Std Dev:          {std_delta:.1f}")
    print()

    rating_changes = sum(1 for r in results if r['current_rating'] != r['proposed_rating'])
    print(f"Rating Changes: {rating_changes} out of {len(results)} stocks ({rating_changes/len(results)*100:.0f}%)")
    print()

    # Show which changed
    if rating_changes > 0:
        print("Stocks with Rating Changes:")
        for r in results:
            if r['current_rating'] != r['proposed_rating']:
                print(f"  {r['ticker']}: {r['current_rating']} → {r['proposed_rating']} (Δ{r['delta']:+.1f})")

    # Lifecycle distribution
    print()
    print("Lifecycle Classification Distribution:")
    from collections import Counter
    lifecycle_counts = Counter(r['lifecycle'] for r in results)
    for lc, count in lifecycle_counts.most_common():
        print(f"  {lc}: {count}")

    print()
    print("=" * 110)
    print("FACTOR COMPARISON TABLE")
    print("=" * 110)
    print()
    print(f"{'Ticker':<7} | {'Value':^11} | {'Growth':^11} | {'Profit':^11} | {'Health':^11} | {'Momentum':^11} | {'Smart$':^11}")
    print(f"{'':7} | {'Cur':>5} {'New':>5} | {'Cur':>5} {'New':>5} | {'Cur':>5} {'New':>5} | {'Cur':>5} {'New':>5} | {'Cur':>5} {'New':>5} | {'Cur':>5} {'New':>5}")
    print("-" * 95)

    for r in results:
        cf = r['current_factors']
        pf = r['proposed_factors']
        print(f"{r['ticker']:<7} | {cf.get('value',0):>5.0f} {pf.get('value',0):>5.0f} | "
              f"{cf.get('growth',0):>5.0f} {pf.get('growth',0):>5.0f} | "
              f"{cf.get('profitability',0):>5.0f} {pf.get('profitability',0):>5.0f} | "
              f"{cf.get('financial_health',0):>5.0f} {pf.get('financial_health',0):>5.0f} | "
              f"{cf.get('momentum',0):>5.0f} {pf.get('momentum',0):>5.0f} | "
              f"{(cf.get('analyst_consensus',0)+cf.get('insider_activity',0)+cf.get('institutional',0))/3:>5.0f} {pf.get('smart_money',0):>5.0f}")


if __name__ == '__main__':
    main()
