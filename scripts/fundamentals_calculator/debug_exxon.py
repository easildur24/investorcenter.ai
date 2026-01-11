#!/usr/bin/env python3
"""
Debug Exxon Revenue TTM - Compare with YCharts
"""

import requests
from datetime import datetime

def debug_exxon():
    # Fetch Exxon data
    url = "https://data.sec.gov/api/xbrl/companyfacts/CIK0000034088.json"
    headers = {'User-Agent': 'InvestorCenter.ai financial-data-collector contact@investorcenter.ai'}
    
    response = requests.get(url, headers=headers)
    facts_data = response.json()
    
    # Get revenue data
    us_gaap = facts_data.get('facts', {}).get('us-gaap', {})
    
    revenue_concepts = ['Revenues', 'RevenueFromContractWithCustomerExcludingAssessedTax']
    
    print("🔍 EXXON REVENUE DATA ANALYSIS")
    print("=" * 60)
    
    for concept in revenue_concepts:
        if concept in us_gaap:
            print(f"\n📊 {concept}:")
            usd_data = us_gaap[concept].get('units', {}).get('USD', [])
            
            # Get 10-Qs
            quarterly_data = [item for item in usd_data 
                            if item.get('form') == '10-Q' 
                            and item.get('frame') and 'Q' in item.get('frame', '')
                            and item.get('end', '') >= '2022-01-01']
            
            # Get 10-Ks  
            annual_data = [item for item in usd_data if item.get('form') == '10-K']
            
            print(f"  Recent 10-Qs ({len(quarterly_data)} found):")
            sorted_10qs = sorted(quarterly_data, key=lambda x: x.get('end', ''), reverse=True)
            for i, q in enumerate(sorted_10qs[:8]):
                print(f"    {i+1}. {q.get('end')} ({q.get('frame')}): ${q.get('val', 0)/1000000:.1f}M")
            
            print(f"  Recent 10-Ks ({len(annual_data)} found):")
            sorted_10ks = sorted(annual_data, key=lambda x: x.get('end', ''), reverse=True)
            for i, k in enumerate(sorted_10ks[:3]):
                print(f"    {i+1}. {k.get('end')}: ${k.get('val', 0)/1000000:.1f}M")
    
    # Apply OUR algorithm
    print(f"\n🧮 OUR ALGORITHM BREAKDOWN:")
    print("=" * 40)
    
    # Get all revenue data (cross-concept)
    all_data = []
    for concept in revenue_concepts:
        if concept in us_gaap:
            usd_data = us_gaap[concept].get('units', {}).get('USD', [])
            for item in usd_data:
                item_copy = item.copy()
                item_copy['source_concept'] = concept
                all_data.append(item_copy)
    
    # Get 10-Qs (ONLY quarterly frames, no YTD)
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
    
    # Get latest 10-K
    all_10ks = [item for item in all_data if item.get('form') == '10-K']
    latest_10k = sorted(all_10ks, key=lambda x: x.get('end', ''), reverse=True)[0] if all_10ks else None
    
    print(f"Latest 3 10-Qs:")
    latest_3_10qs = sorted_10qs[:3]
    latest_3_sum = 0
    for i, q in enumerate(latest_3_10qs):
        val = q.get('val', 0)
        latest_3_sum += val
        print(f"  {i+1}. {q.get('end')} ({q.get('frame')}): ${val/1000000:.1f}M")
    print(f"  TOTAL: ${latest_3_sum/1000000:.1f}M")
    
    print(f"\nLatest 10-K:")
    annual_value = latest_10k.get('val', 0)
    print(f"  {latest_10k.get('end')}: ${annual_value/1000000:.1f}M")
    
    print(f"\n3 10-Qs BEFORE 10-K:")
    tenk_end_date = latest_10k.get('end')
    before_10k_10qs = [q for q in sorted_10qs if q.get('end') < tenk_end_date]
    before_10k_3 = sorted(before_10k_10qs, key=lambda x: x.get('end', ''), reverse=True)[:3]
    before_3_sum = 0
    for i, q in enumerate(before_10k_3):
        val = q.get('val', 0)
        before_3_sum += val
        print(f"  {i+1}. {q.get('end')} ({q.get('frame')}): ${val/1000000:.1f}M")
    print(f"  TOTAL: ${before_3_sum/1000000:.1f}M")
    
    ttm_result = latest_3_sum + annual_value - before_3_sum
    print(f"\n🎯 FINAL CALCULATION:")
    print(f"TTM = ${latest_3_sum/1000000:.1f}M + ${annual_value/1000000:.1f}M - ${before_3_sum/1000000:.1f}M")
    print(f"TTM = ${ttm_result/1000000:.1f}M = ${ttm_result/1000000000:.2f}B")
    
    print(f"\n📊 COMPARISON:")
    print(f"Our Result:    ${ttm_result/1000000000:.2f}B")
    print(f"YCharts:       $324.88B")
    print(f"Difference:    ${(ttm_result/1000000000 - 324.88):.2f}B")
    
    # Check if YCharts might be using simple 4-quarter sum
    print(f"\n🤔 ALTERNATIVE CALCULATIONS:")
    if len(sorted_10qs) >= 4:
        simple_4q_sum = sum(q.get('val', 0) for q in sorted_10qs[:4])
        print(f"Simple 4Q Sum: ${simple_4q_sum/1000000000:.2f}B")
    
    # Check most recent annual only
    print(f"Most Recent 10-K Only: ${annual_value/1000000000:.2f}B")

if __name__ == "__main__":
    debug_exxon()
