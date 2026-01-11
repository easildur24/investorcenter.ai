#!/usr/bin/env python3
"""
Debug Exxon Revenue TTM - Check if YCharts uses Q3 2025 endpoint
"""

import requests

def debug_exxon_q3_endpoint():
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
    
    print("🔍 TESTING Q3 2025 ENDPOINT THEORY")
    print("=" * 50)
    
    # Method 1: Simple 4 quarters ending Q3 2025
    print("📊 METHOD 1: Simple 4Q sum ending Q3 2025")
    q3_2025_quarters = []
    for q in sorted_10qs:
        if q.get('end') <= '2025-09-30':
            q3_2025_quarters.append(q)
        if len(q3_2025_quarters) == 4:
            break
    
    simple_sum = 0
    for i, q in enumerate(q3_2025_quarters):
        val = q.get('val', 0)
        simple_sum += val
        print(f"  {i+1}. {q.get('end')} ({q.get('frame')}): ${val/1000000:.1f}M")
    
    print(f"  TOTAL: ${simple_sum/1000000000:.2f}B")
    
    # Method 2: Our algorithm but ending Q3 2025
    print(f"\n📊 METHOD 2: Our algorithm ending Q3 2025")
    
    # Find the annual report that would be used for Q3 2025 TTM
    all_10ks = [item for item in all_data if item.get('form') == '10-K']
    
    # For Q3 2025 TTM, we'd use 2024 annual report
    annual_2024 = None
    for k in all_10ks:
        if k.get('end') == '2024-12-31':
            annual_2024 = k
            break
    
    if annual_2024:
        # Latest 3 quarters ending Q3 2025: Q3, Q2, Q1 2025
        latest_3_q3_endpoint = []
        for q in sorted_10qs:
            if q.get('end') <= '2025-09-30' and q.get('end') >= '2025-01-01':
                latest_3_q3_endpoint.append(q)
            if len(latest_3_q3_endpoint) == 3:
                break
        
        latest_3_sum = sum(q.get('val', 0) for q in latest_3_q3_endpoint)
        
        # 3 quarters before 2024 annual: Q3, Q2, Q1 2024
        before_quarters = []
        for q in sorted_10qs:
            if q.get('end') < '2024-12-31' and q.get('end') >= '2024-01-01':
                before_quarters.append(q)
            if len(before_quarters) == 3:
                break
        
        before_3_sum = sum(q.get('val', 0) for q in before_quarters)
        annual_value = annual_2024.get('val', 0)
        
        ttm_q3_endpoint = latest_3_sum + annual_value - before_3_sum
        
        print(f"  Latest 3 (Q1-Q3 2025): ${latest_3_sum/1000000:.1f}M")
        print(f"  Annual 2024: ${annual_value/1000000:.1f}M") 
        print(f"  Before 3 (Q1-Q3 2024): ${before_3_sum/1000000:.1f}M")
        print(f"  TTM = ${latest_3_sum/1000000:.1f}M + ${annual_value/1000000:.1f}M - ${before_3_sum/1000000:.1f}M")
        print(f"  TTM = ${ttm_q3_endpoint/1000000000:.2f}B")
    
    print(f"\n🎯 COMPARISON WITH YCHARTS:")
    print(f"YCharts/StockAnalysis: $324.88-324.92B")
    print(f"Simple 4Q (Q3 endpoint): ${simple_sum/1000000000:.2f}B")
    if annual_2024:
        print(f"Our Algorithm (Q3 endpoint): ${ttm_q3_endpoint/1000000000:.2f}B")
    print(f"Our Algorithm (Latest): $333.36B")

if __name__ == "__main__":
    debug_exxon_q3_endpoint()
