#!/usr/bin/env python3
"""Quick script to check AAON's revenue in ttm_financials."""

import os
import psycopg2
from psycopg2.extras import RealDictCursor

# Connect to database
conn = psycopg2.connect(
    host=os.getenv('DB_HOST', 'postgres-simple-service'),
    port=int(os.getenv('DB_PORT', '5432')),
    user=os.getenv('DB_USER'),
    password=os.getenv('DB_PASSWORD'),
    database=os.getenv('DB_NAME', 'investorcenter_db'),
    sslmode=os.getenv('DB_SSLMODE', 'disable')
)

cursor = conn.cursor(cursor_factory=RealDictCursor)

# Query TTM financials for AAON
cursor.execute("""
    SELECT ticker, revenue, eps_basic, eps_diluted,
           net_income, gross_profit,
           ttm_period_start, ttm_period_end,
           calculation_date, created_at
    FROM ttm_financials
    WHERE ticker = 'AAON'
    ORDER BY calculation_date DESC
    LIMIT 1
""")

result = cursor.fetchone()

if result:
    print("\n" + "="*80)
    print("AAON TTM FINANCIALS")
    print("="*80)
    for key, value in result.items():
        print(f"{key:20s}: {value}")
    print("="*80)

    # Verify the fix worked
    if result['revenue'] and result['revenue'] > 0:
        print(f"\n✓ SUCCESS: AAON now has revenue data: ${result['revenue']:,.0f}")
        if result['eps_diluted']:
            print(f"✓ SUCCESS: EPS diluted: ${result['eps_diluted']:.2f}")
        if result['gross_profit']:
            gross_margin = (result['gross_profit'] / result['revenue']) * 100
            print(f"✓ SUCCESS: Gross profit: ${result['gross_profit']:,.0f} (Margin: {gross_margin:.1f}%)")
        if result['net_income']:
            net_margin = (result['net_income'] / result['revenue']) * 100
            print(f"✓ SUCCESS: Net income: ${result['net_income']:,.0f} (Margin: {net_margin:.1f}%)")
        print(f"\nTTM Period: {result['ttm_period_start']} to {result['ttm_period_end']}")
        print("\nNote: PS ratio is calculated in valuation_ratios table using current market price")
    else:
        print("\n✗ ERROR: AAON still has zero or null revenue!")
else:
    print("ERROR: No TTM financials found for AAON")

# Also check if financials table has data
print("\n" + "="*80)
print("CHECKING FINANCIALS TABLE (QUARTERLY DATA)")
print("="*80)
cursor.execute("""
    SELECT COUNT(*) as count, MIN(period_end_date) as earliest, MAX(period_end_date) as latest
    FROM financials
    WHERE ticker = 'AAON'
""")
fin_result = cursor.fetchone()
print(f"Quarterly records for AAON: {fin_result['count']}")
print(f"Date range: {fin_result['earliest']} to {fin_result['latest']}")

cursor.close()
conn.close()
