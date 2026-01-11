#!/usr/bin/env python3
"""
Debug Exxon - Test alternative TTM calculations to match YCharts
"""

import requests

def debug_exxon_alternative():
    # Fetch Exxon data
    url = "https://data.sec.gov/api/xbrl/companyfacts/CIK0000034088.json"
    headers = {'User-Agent': 'InvestorCenter.ai financial-data-collector contact@investorcenter.ai'}
    
    response = requests.get(url, headers=headers)
    facts_data = response.json()
    
    us_gaap = facts_data.get('facts', {}).get('us-gaap', {})
    
    print("🔍 TESTING ALTERNATIVE EXXON TTM CALCULATIONS")
    print("=" * 60)
    
    # Test different revenue concepts individually
    revenue_concepts = ['Revenues', 'RevenueFromContractWithCustomerExcludingAssessedTax']
    
    for concept in revenue_concepts:
        if concept in us_gaap:
            print(f"\n📊 USING ONLY {concept}:")
            usd_data = us_gaap[concept].get('units', {}).get('USD', [])
            
            # Get 10-Qs
            quarterly_data = [item for item in usd_data 
                            if item.get('form') == '10-Q' 
                            and item.get('frame') and 'Q' in item.get('frame', '')
                            and item.get('end', '') >= '2020-01-01']
            
            sorted_10qs = sorted(quarterly_data, key=lambda x: x.get('end', ''), reverse=True)
            
            if len(sorted_10qs) >= 4:
                # Simple 4Q sum
                recent_4q = sorted_10qs[:4]
                simple_sum = sum(q.get('val', 0) for q in recent_4q)
                
                print(f"  Recent 4 quarters:")
                for q in recent_4q:
                    print(f"    {q.get('end')} ({q.get('frame')}): ${q.get('val', 0)/1000000:.1f}M")
                print(f"  Simple 4Q Sum: ${simple_sum/1000000000:.2f}B")
                
                # Check if this matches YCharts
                if abs(simple_sum/1000000000 - 324.90) < 1.0:
                    print(f"  🎯 CLOSE MATCH TO YCHARTS!")
    
    # Test using Q4 2024 through Q3 2025 (different endpoint)
    print(f"\n📊 TESTING Q4 2024 - Q3 2025 PERIOD:")
    
    all_data = []
    for concept in revenue_concepts:
        if concept in us_gaap:
            usd_data = us_gaap[concept].get('units', {}).get('USD', [])
            for item in usd_data:
                item_copy = item.copy()
                item_copy['source_concept'] = concept
                all_data.append(item_copy)
    
    # Get all quarters
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
    
    # Find Q1, Q2, Q3 2025 + calculated Q4 2024
    q2025_data = [q for q in sorted_10qs if '2025' in q.get('end', '') and q.get('end') <= '2025-09-30']
    q2024_data = [q for q in sorted_10qs if '2024' in q.get('end', '') and q.get('end') < '2024-12-31']
    
    annual_2024 = None
    for k in all_10ks:
        if k.get('end') == '2024-12-31':
            annual_2024 = k
            break
    
    if len(q2025_data) >= 3 and len(q2024_data) >= 3 and annual_2024:
        q2025_sum = sum(q.get('val', 0) for q in q2025_data[:3])
        q2024_sum = sum(q.get('val', 0) for q in q2024_data[:3])
        annual_val = annual_2024.get('val', 0)
        calc_q4_2024 = annual_val - q2024_sum
        
        # Alternative: Maybe they use Q4 2024 - Q3 2025
        alt_ttm = calc_q4_2024 + q2025_sum
        print(f"  Q1+Q2+Q3 2025: ${q2025_sum/1000000:.1f}M")
        print(f"  Calculated Q4 2024: ${calc_q4_2024/1000000:.1f}M")
        print(f"  TTM (Q4 2024 - Q3 2025): ${alt_ttm/1000000000:.2f}B")
        
        # Maybe they exclude Q4 and use different quarters?
        # Test Q1 2024 - Q4 2024 (using calculated Q4)
        q1_q4_2024 = q2024_sum + calc_q4_2024
        print(f"  TTM (Q1-Q4 2024): ${q1_q4_2024/1000000000:.2f}B")
        
        # Test different combinations
        print(f"\n🧪 TESTING DIFFERENT QUARTER COMBINATIONS:")
        
        # Maybe they use Q2 2024 - Q1 2025?
        if len(sorted_10qs) >= 4:
            test_quarters = sorted_10qs[2:6]  # Skip most recent 2, take next 4
            test_sum = sum(q.get('val', 0) for q in test_quarters)
            print(f"  Quarters 3-6 from latest: ${test_sum/1000000000:.2f}B")
            
            if abs(test_sum/1000000000 - 324.90) < 1.0:
                print(f"    🎯 POTENTIAL MATCH!")
                for q in test_quarters:
                    print(f"      {q.get('end')} ({q.get('frame')}): ${q.get('val', 0)/1000000:.1f}M")

if __name__ == "__main__":
    debug_exxon_alternative()
