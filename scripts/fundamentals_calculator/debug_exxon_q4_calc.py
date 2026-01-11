#!/usr/bin/env python3
"""
Debug Exxon - Check if YCharts calculates Q4 2024 differently
"""

import requests

def debug_exxon_q4_calc():
    # Fetch Exxon data
    url = "https://data.sec.gov/api/xbrl/companyfacts/CIK0000034088.json"
    headers = {'User-Agent': 'InvestorCenter.ai financial-data-collector contact@investorcenter.ai'}
    
    response = requests.get(url, headers=headers)
    facts_data = response.json()
    
    us_gaap = facts_data.get('facts', {}).get('us-gaap', {})
    
    # Get all revenue data
    all_data = []
    revenue_concepts = ['Revenues', 'RevenueFromContractWithCustomerExcludingAssessedTax']
    
    for concept in revenue_concepts:
        if concept in us_gaap:
            usd_data = us_gaap[concept].get('units', {}).get('USD', [])
            for item in usd_data:
                item_copy = item.copy()
                item_copy['source_concept'] = concept
                all_data.append(item_copy)
    
    # Get 10-Qs
    all_10qs = [item for item in all_data 
               if item.get('form') == '10-Q' 
               and item.get('frame') and 'Q' in item.get('frame', '')
               and item.get('end', '') >= '2020-01-01']
    
    # Remove duplicates
    unique_10qs = {}
    for item in all_10qs:
        key = f"{item.get('end')}_{item.get('frame')}"
        if key not in unique_10qs:
            unique_10qs[key] = item
    
    sorted_10qs = sorted(unique_10qs.values(), key=lambda x: x.get('end', ''), reverse=True)
    
    # Get annual data
    all_10ks = [item for item in all_data if item.get('form') == '10-K']
    annual_2024 = None
    for k in all_10ks:
        if k.get('end') == '2024-12-31':
            annual_2024 = k
            break
    
    print("🔍 TESTING Q4 2024 CALCULATION METHODS")
    print("=" * 50)
    
    # Get Q1, Q2, Q3 2024
    q2024_quarters = []
    for q in sorted_10qs:
        if '2024' in q.get('end', '') and q.get('end') < '2024-12-31':
            q2024_quarters.append(q)
    
    q2024_quarters = sorted(q2024_quarters, key=lambda x: x.get('end', ''))
    
    print("📊 2024 QUARTERLY DATA:")
    q1_q2_q3_2024_sum = 0
    for q in q2024_quarters:
        val = q.get('val', 0)
        q1_q2_q3_2024_sum += val
        print(f"  {q.get('end')} ({q.get('frame')}): ${val/1000000:.1f}M")
    
    print(f"  Q1+Q2+Q3 2024 Total: ${q1_q2_q3_2024_sum/1000000:.1f}M")
    
    if annual_2024:
        annual_val = annual_2024.get('val', 0)
        calculated_q4_2024 = annual_val - q1_q2_q3_2024_sum
        print(f"  Annual 2024: ${annual_val/1000000:.1f}M")
        print(f"  Calculated Q4 2024: ${calculated_q4_2024/1000000:.1f}M")
        
        # Now test TTM ending Q3 2025 using calculated Q4
        print(f"\n📊 TTM ENDING Q3 2025 WITH CALCULATED Q4:")
        
        # Get Q1, Q2, Q3 2025
        q2025_quarters = []
        for q in sorted_10qs:
            if '2025' in q.get('end', '') and q.get('end') <= '2025-09-30':
                q2025_quarters.append(q)
        
        q2025_quarters = sorted(q2025_quarters, key=lambda x: x.get('end', ''))
        
        q1_q2_q3_2025_sum = 0
        for q in q2025_quarters:
            val = q.get('val', 0)
            q1_q2_q3_2025_sum += val
            print(f"  {q.get('end')} ({q.get('frame')}): ${val/1000000:.1f}M")
        
        print(f"  Q1+Q2+Q3 2025 Total: ${q1_q2_q3_2025_sum/1000000:.1f}M")
        print(f"  Calculated Q4 2024: ${calculated_q4_2024/1000000:.1f}M")
        
        ttm_with_calc_q4 = q1_q2_q3_2025_sum + calculated_q4_2024
        print(f"  TTM = ${q1_q2_q3_2025_sum/1000000:.1f}M + ${calculated_q4_2024/1000000:.1f}M")
        print(f"  TTM = ${ttm_with_calc_q4/1000000000:.2f}B")
        
        print(f"\n🎯 FINAL COMPARISON:")
        print(f"YCharts/StockAnalysis: $324.88-324.92B")
        print(f"TTM with Calc Q4 2024: ${ttm_with_calc_q4/1000000000:.2f}B")
        print(f"Difference: ${abs(ttm_with_calc_q4/1000000000 - 324.90):.2f}B")

if __name__ == "__main__":
    debug_exxon_q4_calc()
