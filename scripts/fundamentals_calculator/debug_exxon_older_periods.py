#!/usr/bin/env python3
"""
Debug Exxon - Test if YCharts is using older periods
"""

import requests

def debug_exxon_older_periods():
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
    
    print("🔍 TESTING OLDER PERIODS TO MATCH YCHARTS $324.88B")
    print("=" * 60)
    
    # Show all quarters for manual inspection
    print("📊 ALL AVAILABLE QUARTERS:")
    for i, q in enumerate(sorted_10qs[:12]):
        val = q.get('val', 0)
        print(f"  {i+1:2d}. {q.get('end')} ({q.get('frame')}): ${val/1000000:.1f}M")
    
    print(f"\n🧪 TESTING DIFFERENT 4-QUARTER COMBINATIONS:")
    
    # Test different starting points
    target = 324.88
    
    for start_idx in range(0, min(8, len(sorted_10qs)-3)):
        test_quarters = sorted_10qs[start_idx:start_idx+4]
        test_sum = sum(q.get('val', 0) for q in test_quarters)
        test_billions = test_sum / 1000000000
        
        print(f"\n  Quarters {start_idx+1}-{start_idx+4}:")
        for q in test_quarters:
            print(f"    {q.get('end')} ({q.get('frame')}): ${q.get('val', 0)/1000000:.1f}M")
        print(f"    Total: ${test_billions:.2f}B")
        
        if abs(test_billions - target) < 2.0:
            print(f"    🎯 CLOSE TO YCHARTS! Diff: ${abs(test_billions - target):.2f}B")
    
    # Test specific combinations that might make sense
    print(f"\n🎯 TESTING LOGICAL COMBINATIONS:")
    
    # Q4 2023 - Q3 2024 (full year ending Q3 2024)
    q2024_quarters = [q for q in sorted_10qs if '2024' in q.get('end', '') and q.get('end') <= '2024-09-30']
    
    if len(q2024_quarters) >= 3:
        # Get annual 2023 data
        all_10ks = [item for item in all_data if item.get('form') == '10-K']
        annual_2023 = None
        for k in all_10ks:
            if k.get('end') == '2023-12-31':
                annual_2023 = k
                break
        
        if annual_2023:
            # Get Q1-Q3 2023
            q2023_quarters = [q for q in sorted_10qs if '2023' in q.get('end', '') and q.get('end') < '2023-12-31']
            
            if len(q2023_quarters) >= 3:
                q1_q3_2024_sum = sum(q.get('val', 0) for q in q2024_quarters[:3])
                q1_q3_2023_sum = sum(q.get('val', 0) for q in q2023_quarters[:3])
                annual_2023_val = annual_2023.get('val', 0)
                calc_q4_2023 = annual_2023_val - q1_q3_2023_sum
                
                ttm_2024_q3_endpoint = q1_q3_2024_sum + calc_q4_2023
                
                print(f"\n  Q4 2023 - Q3 2024 TTM:")
                print(f"    Q1-Q3 2024: ${q1_q3_2024_sum/1000000:.1f}M")
                print(f"    Calculated Q4 2023: ${calc_q4_2023/1000000:.1f}M")
                print(f"    TTM: ${ttm_2024_q3_endpoint/1000000000:.2f}B")
                
                if abs(ttm_2024_q3_endpoint/1000000000 - target) < 2.0:
                    print(f"    🎯 POTENTIAL MATCH! Diff: ${abs(ttm_2024_q3_endpoint/1000000000 - target):.2f}B")

if __name__ == "__main__":
    debug_exxon_older_periods()
