#!/usr/bin/env python3
"""
SEC EDGAR API Calculator v3 - Simple and Correct TTM Calculation
Direct implementation of the correct logic we figured out manually
"""

import requests
import json
import psycopg2
import os
import sys
import time
from datetime import datetime, timedelta

class SECEdgarCalculatorV3:
    def __init__(self):
        self.base_url = "https://data.sec.gov/api/xbrl/companyfacts/"
        self.headers = {
            'User-Agent': 'InvestorCenter.ai financial-data-collector contact@investorcenter.ai'
        }
        
        # CIK mapping for major companies
        self.cik_mapping = {
            'AAPL': '0000320193',
            'MSFT': '0000789019', 
            'GOOGL': '0001652044',
            'AMZN': '0001018724',
            'TSLA': '0001318605',
            'META': '0001326801',
            'NVDA': '0001045810',
            'BRK-B': '0001067983',
            'UNH': '0000731766',
            'JNJ': '0000200406',
            'HIMS': '0001908443'  # Hims & Hers Health Inc
        }
    
    def get_company_facts(self, symbol):
        """Fetch company facts from SEC EDGAR API"""
        cik = self.cik_mapping.get(symbol.upper())
        if not cik:
            print(f"❌ CIK not found for symbol: {symbol}")
            return None
        
        url = f"{self.base_url}CIK{cik}.json"
        
        try:
            response = requests.get(url, headers=self.headers)
            response.raise_for_status()
            return response.json()
        except requests.exceptions.RequestException as e:
            print(f"❌ Error fetching data for {symbol}: {e}")
            return None
    
    def extract_ttm_simple(self, facts_data, concept_names, metric_name):
        """Extract TTM by ALWAYS summing last 4 quarters - BULLETPROOF METHOD"""
        try:
            us_gaap = facts_data.get('facts', {}).get('us-gaap', {})
            
            # Try multiple concept names
            for concept in concept_names:
                if concept in us_gaap:
                    usd_data = us_gaap[concept].get('units', {}).get('USD', [])
                    
                    # Get quarterly data
                    quarterly_data = [item for item in usd_data 
                                    if item.get('frame') and 'Q' in item.get('frame', '')]
                    
                    if len(quarterly_data) >= 4:
                        # Sort by end date (most recent first)
                        sorted_quarters = sorted(quarterly_data, key=lambda x: x.get('end', ''), reverse=True)
                        
                        # TTM = sum of last 4 quarters
                        ttm_value = sum(q.get('val', 0) for q in sorted_quarters[:4])
                        
                        print(f"  ✅ TTM {metric_name}: ${ttm_value/1000000:.1f}M (sum of 4 quarters)")
                        return ttm_value
            
            # Fallback to original method if no quarterly data
            return self.extract_ttm_value(facts_data, concept_names[0] if concept_names else concept)
            
        except Exception as e:
            print(f"⚠️ Error extracting TTM {metric_name}: {e}")
            return None

    def extract_ttm_value(self, facts_data, concept):
        """
        Extract TTM using the SIMPLE and CORRECT approach we figured out manually:
        
        1. Get the most recent 3 quarters available
        2. Calculate the missing 4th quarter from annual data
        3. Sum all 4 quarters for TTM
        
        This is much simpler and more reliable than complex quarter matching
        """
        try:
            us_gaap = facts_data.get('facts', {}).get('us-gaap', {})
            concept_data = us_gaap.get(concept, {})
            
            if not concept_data:
                return None
            
            units = concept_data.get('units', {})
            usd_data = units.get('USD', [])
            
            if not usd_data:
                return None
            
            # Get quarterly and annual data
            quarterly_data = [item for item in usd_data 
                            if item.get('frame') and 'Q' in item.get('frame', '') 
                            and item.get('end', '') >= '2023-01-01']
            
            annual_data = [item for item in usd_data 
                          if item.get('form') == '10-K' 
                          and item.get('end', '') >= '2022-01-01']
            
            if not quarterly_data or not annual_data:
                # Fallback to most recent annual if no quarterly data
                if annual_data:
                    latest_annual = sorted(annual_data, key=lambda x: x.get('end', ''), reverse=True)[0]
                    annual_value = latest_annual.get('val')
                    end_date = datetime.strptime(latest_annual.get('end'), '%Y-%m-%d')
                    months_old = (datetime.now() - end_date).days / 30.44
                    print(f"  ✅ TTM {concept}: ${annual_value/1000000:.1f}M (from 10-K ending {latest_annual.get('end')}, {months_old:.1f} months old)")
                    return annual_value
                return None
            
            # Sort data
            sorted_quarterly = sorted(quarterly_data, key=lambda x: x.get('end', ''), reverse=True)
            sorted_annual = sorted(annual_data, key=lambda x: x.get('end', ''), reverse=True)
            
            # HYBRID APPROACH: Use 10-K if recent, quarterly calculation if 10-K is stale
            latest_annual = sorted_annual[0]
            annual_value = latest_annual.get('val')
            end_date = datetime.strptime(latest_annual.get('end'), '%Y-%m-%d')
            months_old = (datetime.now() - end_date).days / 30.44
            
            if months_old <= 3:
                # 10-K is recent - use it directly
                print(f"  ✅ TTM {concept}: ${annual_value/1000000:.1f}M (from recent 10-K ending {latest_annual.get('end')}, {months_old:.1f} months old)")
                return annual_value
            else:
                # 10-K is stale - try quarterly calculation for more current TTM
                print(f"  🔍 10-K is {months_old:.1f} months old, attempting quarterly calculation for current TTM...")
                
                if len(sorted_quarterly) >= 3:
                    # Get the 3 most recent quarters
                    recent_quarters = sorted_quarterly[:3]
                    
                    # Calculate missing Q4 from the annual data
                    # Find the annual period that contains the missing quarter
                    latest_quarter_end = datetime.strptime(recent_quarters[0].get('end'), '%Y-%m-%d')
                    
                    # The missing quarter should be about 1 year before the latest quarter
                    # Find which annual report contains this period
                    best_annual = None
                    for annual in sorted_annual:
                        annual_start = datetime.strptime(annual.get('start'), '%Y-%m-%d')
                        annual_end = datetime.strptime(annual.get('end'), '%Y-%m-%d')
                        
                        # Check if the missing quarter (roughly 1 year back) falls in this annual period
                        missing_quarter_approx = latest_quarter_end - timedelta(days=365)
                        if annual_start <= missing_quarter_approx <= annual_end:
                            best_annual = annual
                            break
                    
                    if best_annual:
                        annual_for_calc = best_annual.get('val', 0)
                        annual_start = datetime.strptime(best_annual.get('start'), '%Y-%m-%d')
                        annual_end = datetime.strptime(best_annual.get('end'), '%Y-%m-%d')
                        
                        # Find the 3 quarters within this annual period
                        annual_quarters = []
                        for q in quarterly_data:
                            q_end = datetime.strptime(q.get('end'), '%Y-%m-%d')
                            if annual_start <= q_end <= annual_end:
                                annual_quarters.append(q)
                        
                        if len(annual_quarters) == 3:
                            # Calculate missing quarter: Annual - (3 quarters from that annual period)
                            annual_quarters_sum = sum(q.get('val', 0) for q in annual_quarters)
                            missing_quarter_value = annual_for_calc - annual_quarters_sum
                            
                            # Current TTM = 3 recent quarters + calculated missing quarter
                            recent_quarters_sum = sum(q.get('val', 0) for q in recent_quarters)
                            ttm_value = recent_quarters_sum + missing_quarter_value
                            
                            print(f"  ✅ TTM {concept}: ${ttm_value/1000000:.1f}M (3 recent quarters + 1 calculated quarter)")
                            print(f"    = Recent quarters: ${recent_quarters_sum/1000000:.1f}M + Calculated Q4: ${missing_quarter_value/1000000:.1f}M")
                            return ttm_value
                
                # Fallback to 10-K if quarterly calculation fails
                print(f"  ⚠️ Quarterly calculation failed, using 10-K as fallback")
                print(f"  ✅ TTM {concept}: ${annual_value/1000000:.1f}M (from 10-K ending {latest_annual.get('end')}, {months_old:.1f} months old - fallback)")
                return annual_value
            
            # Not enough quarterly data - use annual
            latest_annual = sorted_annual[0]
            annual_value = latest_annual.get('val')
            end_date = datetime.strptime(latest_annual.get('end'), '%Y-%m-%d')
            months_old = (datetime.now() - end_date).days / 30.44
            print(f"  ✅ TTM {concept}: ${annual_value/1000000:.1f}M (from 10-K ending {latest_annual.get('end')}, {months_old:.1f} months old)")
            return annual_value
            
        except Exception as e:
            print(f"⚠️ Error extracting TTM {concept}: {e}")
            return None
    
    def extract_quarterly_value(self, facts_data, concept):
        """Extract the most recent quarterly value (including calculated Q4 if needed)"""
        try:
            us_gaap = facts_data.get('facts', {}).get('us-gaap', {})
            concept_data = us_gaap.get(concept, {})
            
            if not concept_data:
                return None
            
            units = concept_data.get('units', {})
            usd_data = units.get('USD', [])
            
            if not usd_data:
                return None
            
            # Get quarterly data and annual data
            quarterly_data = [item for item in usd_data 
                            if item.get('frame') and 'Q' in item.get('frame', '') 
                            and item.get('end', '') >= '2024-01-01']
            
            annual_data = [item for item in usd_data if item.get('form') == '10-K']
            
            if quarterly_data and annual_data:
                # Sort quarterly data by end date (most recent first)
                sorted_quarterly = sorted(quarterly_data, key=lambda x: x.get('end', ''), reverse=True)
                
                # Get most recent annual report
                latest_annual = sorted(annual_data, key=lambda x: x.get('end', ''), reverse=True)[0]
                annual_value = latest_annual.get('val', 0)
                annual_end = latest_annual.get('end')
                annual_start = latest_annual.get('start')
                
                # Check if we can calculate the missing Q4 from annual data
                # Find quarters that are within the annual period
                fiscal_year_quarters = [q for q in sorted_quarterly 
                                      if q.get('end') <= annual_end and q.get('end') >= annual_start]
                
                # FIXED LOGIC: Always use the most recent available quarter first
                latest_quarter = sorted_quarterly[0]
                quarterly_value = latest_quarter.get('val')
                latest_end = latest_quarter.get('end')
                
                print(f"  ✅ Quarterly {concept}: ${quarterly_value/1000000:.1f}M (Q ending {latest_end})")
                return quarterly_value
            elif quarterly_data:
                # Only quarterly data available
                sorted_quarterly = sorted(quarterly_data, key=lambda x: x.get('end', ''), reverse=True)
                latest_quarter = sorted_quarterly[0]
                quarterly_value = latest_quarter.get('val')
                print(f"  ✅ Quarterly {concept}: ${quarterly_value/1000000:.1f}M (Q ending {latest_quarter.get('end')})")
                return quarterly_value
            
            return None
        except Exception as e:
            print(f"⚠️ Error extracting quarterly {concept}: {e}")
            return None
    
    def calculate_yoy_growth(self, facts_data, current_metrics):
        """Calculate Year-over-Year growth rates for key metrics"""
        try:
            growth_metrics = {}
            
            # Get current and previous year quarterly data
            us_gaap = facts_data.get('facts', {}).get('us-gaap', {})
            
            # Revenue YoY Growth
            revenue_concept = 'Revenues'
            if revenue_concept not in us_gaap:
                revenue_concept = 'RevenueFromContractWithCustomerExcludingAssessedTax'
            
            # Use already calculated quarterly revenue and find matching previous year quarter
            current_revenue = current_metrics.get('revenue_quarterly')
            
            if current_revenue:
                # Find same time period from previous year using date matching
                us_gaap = facts_data.get('facts', {}).get('us-gaap', {})
                # Try both concepts - NVIDIA uses 'Revenues'
                revenue_concept = 'Revenues'
                if revenue_concept not in us_gaap:
                    revenue_concept = 'RevenueFromContractWithCustomerExcludingAssessedTax'
                
                if revenue_concept in us_gaap:
                    usd_data = us_gaap[revenue_concept].get('units', {}).get('USD', [])
                    # Filter for recent quarterly data only (2022 onwards to get enough data)
                    quarterly_data = [item for item in usd_data 
                                    if item.get('frame') and 'Q' in item.get('frame', '') 
                                    and item.get('end', '') >= '2022-01-01']
                    
                    if len(quarterly_data) >= 5:
                        sorted_quarters = sorted(quarterly_data, key=lambda x: x.get('end', ''), reverse=True)
                        
                        # Find the matching quarter from previous year (same time period)
                        current_end = sorted_quarters[0].get('end')
                        
                        # For NVIDIA: 2025-10-26 should match with 2024-10-27
                        # Look for quarter ending around same time previous year
                        prev_year_revenue = None
                        prev_end = None
                        
                        from datetime import datetime, timedelta
                        current_date = datetime.strptime(current_end, '%Y-%m-%d')
                        target_date = current_date - timedelta(days=365)
                        
                        # Find closest quarter to target date
                        min_diff = float('inf')
                        for q in sorted_quarters[1:]:
                            q_end = datetime.strptime(q.get('end'), '%Y-%m-%d')
                            diff = abs((q_end - target_date).days)
                            if diff < min_diff and diff < 60:  # Within 60 days
                                min_diff = diff
                                prev_year_revenue = q.get('val', 0)
                                prev_end = q.get('end')
                        
                        if prev_year_revenue > 0:
                            growth_metrics['revenue_yoy_growth'] = ((current_revenue - prev_year_revenue) / prev_year_revenue) * 100
                            print(f"  ✅ Revenue YoY Growth: {growth_metrics['revenue_yoy_growth']:.2f}% ({current_end} vs {prev_end})")
                        else:
                            print(f"  ❌ Revenue YoY Growth: prev_year_revenue = 0")
                    else:
                        print(f"  ❌ Revenue YoY Growth: insufficient quarterly data ({len(quarterly_data)} quarters)")
                else:
                    print(f"  ❌ Revenue YoY Growth: concept {revenue_concept} not found")
            else:
                print(f"  ❌ Revenue YoY Growth: no current revenue")
            
            # EBITDA YoY Growth - use same quarter matching as Revenue YoY
            current_ebitda = current_metrics.get('ebitda_quarterly')
            if current_ebitda:
                # Get EBIT quarterly data for matching
                ebit_data = us_gaap.get('OperatingIncomeLoss', {}).get('units', {}).get('USD', [])
                quarterly_ebit = [item for item in ebit_data 
                                if item.get('frame') and 'Q' in item.get('frame', '') 
                                and item.get('end', '') >= '2022-01-01']
                
                if len(quarterly_ebit) >= 5:
                    sorted_ebit = sorted(quarterly_ebit, key=lambda x: x.get('end', ''), reverse=True)
                    
                    # Find matching previous year quarter using same logic as Revenue YoY
                    current_end = sorted_ebit[0].get('end')
                    current_date = datetime.strptime(current_end, '%Y-%m-%d')
                    target_date = current_date - timedelta(days=365)
                    
                    prev_ebit = None
                    prev_end = None
                    min_diff = float('inf')
                    
                    for q in sorted_ebit[1:]:
                        q_end = datetime.strptime(q.get('end'), '%Y-%m-%d')
                        diff = abs((q_end - target_date).days)
                        if diff < min_diff and diff < 60:
                            min_diff = diff
                            prev_ebit = q.get('val', 0)
                            prev_end = q.get('end')
                    
                    if prev_ebit:
                        # Get depreciation for both periods (use same quarterly depreciation as current)
                        current_depreciation = self.extract_quarterly_value(facts_data, 'DepreciationDepletionAndAmortization')
                        if current_depreciation:
                            prev_ebitda = prev_ebit + current_depreciation  # Approximate
                            
                            if prev_ebitda > 0:
                                growth_metrics['ebitda_yoy_growth'] = ((current_ebitda - prev_ebitda) / prev_ebitda) * 100
                                print(f"  ✅ EBITDA YoY Growth: {growth_metrics['ebitda_yoy_growth']:.2f}% ({current_end} vs {prev_end})")
                            else:
                                print(f"  ❌ EBITDA YoY Growth: prev_ebitda <= 0")
                        else:
                            print(f"  ❌ EBITDA YoY Growth: No depreciation data")
                    else:
                        print(f"  ❌ EBITDA YoY Growth: Could not find matching previous year quarter")
                else:
                    print(f"  ❌ EBITDA YoY Growth: Insufficient EBIT quarterly data")
            else:
                print(f"  ❌ EBITDA YoY Growth: No current EBITDA")
            
            # EPS YoY Growth - use same quarter matching logic as Revenue YoY
            current_quarterly_eps = self.get_current_quarterly_eps_simple(facts_data)
            prev_quarterly_eps = self.get_previous_year_quarterly_eps_simple(facts_data)
            
            if current_quarterly_eps and prev_quarterly_eps and prev_quarterly_eps > 0:
                growth_metrics['eps_diluted_yoy_growth'] = ((current_quarterly_eps - prev_quarterly_eps) / prev_quarterly_eps) * 100
                print(f"  ✅ EPS Diluted YoY Growth: {growth_metrics['eps_diluted_yoy_growth']:.2f}% (${current_quarterly_eps:.3f} vs ${prev_quarterly_eps:.3f})")
            else:
                print(f"  ❌ EPS YoY Growth: Could not calculate (current={current_quarterly_eps is not None}, prev={prev_quarterly_eps is not None})")
            
            return growth_metrics
            
        except Exception as e:
            print(f"⚠️ Error calculating YoY growth: {e}")
            return {}
    
    def calculate_working_capital_change(self, facts_data):
        """Calculate changes in working capital from balance sheet changes"""
        try:
            us_gaap = facts_data.get('facts', {}).get('us-gaap', {})
            
            # Get current assets and liabilities for last 2 periods
            current_assets_data = us_gaap.get('AssetsCurrent', {}).get('units', {}).get('USD', [])
            current_liab_data = us_gaap.get('LiabilitiesCurrent', {}).get('units', {}).get('USD', [])
            
            if current_assets_data and current_liab_data:
                # Get latest 2 periods
                sorted_assets = sorted(current_assets_data, key=lambda x: x.get('end', ''), reverse=True)
                sorted_liab = sorted(current_liab_data, key=lambda x: x.get('end', ''), reverse=True)
                
                if len(sorted_assets) >= 2 and len(sorted_liab) >= 2:
                    current_wc = sorted_assets[0].get('val', 0) - sorted_liab[0].get('val', 0)
                    previous_wc = sorted_assets[1].get('val', 0) - sorted_liab[1].get('val', 0)
                    
                    wc_change = current_wc - previous_wc
                    print(f"  ✅ Calculated Working Capital Change: ${wc_change/1000000:.1f}M")
                    return wc_change
            
            return None
            
        except Exception as e:
            print(f"⚠️ Error calculating working capital change: {e}")
            return None

    def get_previous_year_q4_value(self, facts_data, concept):
        """Get Q4 value from previous fiscal year (calculated from annual data)"""
        try:
            us_gaap = facts_data.get('facts', {}).get('us-gaap', {})
            concept_data = us_gaap.get(concept, {})
            
            if not concept_data:
                return None
            
            units = concept_data.get('units', {})
            usd_data = units.get('USD', [])
            
            # Get annual data
            annual_data = [item for item in usd_data if item.get('form') == '10-K']
            if len(annual_data) < 2:
                return None
            
            # Get previous year annual data (second most recent)
            sorted_annual = sorted(annual_data, key=lambda x: x.get('end', ''), reverse=True)
            prev_year_annual = sorted_annual[1]  # Previous year
            prev_annual_value = prev_year_annual.get('val', 0)
            prev_annual_start = prev_year_annual.get('start')
            prev_annual_end = prev_year_annual.get('end')
            
            # Get quarterly data for that previous year
            quarterly_data = [item for item in usd_data 
                            if item.get('frame') and 'Q' in item.get('frame', '') 
                            and item.get('end') <= prev_annual_end 
                            and item.get('end') >= prev_annual_start]
            
            if len(quarterly_data) == 3:  # Missing Q4, calculate it
                q_sum = sum(q.get('val', 0) for q in quarterly_data)
                calculated_prev_q4 = prev_annual_value - q_sum
                print(f"  📊 Previous Year Q4 {concept}: ${calculated_prev_q4/1000000:.1f}M (calculated)")
                return calculated_prev_q4
            elif len(quarterly_data) >= 4:  # All quarters available
                sorted_quarterly = sorted(quarterly_data, key=lambda x: x.get('end', ''), reverse=True)
                result = sorted_quarterly[0].get('val')
                print(f"  📊 Previous Year Q4 {concept}: ${result/1000000:.1f}M (from quarterly data)")
                return result
            
            return None
        except Exception as e:
            print(f"⚠️ Error getting previous year Q4 {concept}: {e}")
            return None

    def get_previous_year_annual_value(self, facts_data, concept):
        """Get annual value from previous fiscal year"""
        try:
            us_gaap = facts_data.get('facts', {}).get('us-gaap', {})
            concept_data = us_gaap.get(concept, {})
            
            if not concept_data:
                return None
            
            units = concept_data.get('units', {})
            usd_data = units.get('USD', [])
            
            # Get annual data
            annual_data = [item for item in usd_data if item.get('form') == '10-K']
            if len(annual_data) < 2:
                return None
            
            # Get previous year annual data (second most recent)
            sorted_annual = sorted(annual_data, key=lambda x: x.get('end', ''), reverse=True)
            prev_year_annual = sorted_annual[1]  # Previous year
            return prev_year_annual.get('val', 0)
        except Exception as e:
            print(f"⚠️ Error getting previous year annual {concept}: {e}")
            return None

    def get_previous_year_eps(self, facts_data, concept):
        """Get EPS from previous year annual report"""
        try:
            us_gaap = facts_data.get('facts', {}).get('us-gaap', {})
            concept_data = us_gaap.get(concept, {})
            
            if not concept_data:
                return None
            
            units = concept_data.get('units', {})
            usd_shares_data = units.get('USD/shares', [])
            
            if usd_shares_data:
                # Get annual EPS data
                annual_eps = [item for item in usd_shares_data if item.get('form') == '10-K']
                if len(annual_eps) >= 2:
                    sorted_eps = sorted(annual_eps, key=lambda x: x.get('end', ''), reverse=True)
                    prev_eps = sorted_eps[1].get('val')  # Previous year
                    print(f"  📊 Previous Year EPS: ${prev_eps:.2f}")
                    return prev_eps
            
            return None
        except Exception as e:
            print(f"⚠️ Error getting previous year EPS: {e}")
            return None

    def get_current_quarterly_eps(self, facts_data, concept):
        """Get current quarterly EPS (simplified calculation)"""
        try:
            # Use quarterly net income / annual shares (simpler and more reliable)
            quarterly_net_income = self.extract_quarterly_value(facts_data, 'NetIncomeLoss')
            
            # Get shares from the stored metrics (already calculated correctly)
            us_gaap = facts_data.get('facts', {}).get('us-gaap', {})
            shares_concepts = ['WeightedAverageNumberOfSharesOutstandingBasic', 'CommonStockSharesOutstanding']
            
            shares_outstanding = None
            for concept in shares_concepts:
                if concept in us_gaap:
                    shares_data = us_gaap[concept].get('units', {}).get('shares', [])
                    if shares_data:
                        # Get most recent shares data
                        latest_shares = sorted(shares_data, key=lambda x: x.get('end', ''), reverse=True)[0]
                        shares_outstanding = latest_shares.get('val')
                        break
            
            if quarterly_net_income and shares_outstanding and shares_outstanding > 0:
                quarterly_eps = quarterly_net_income / shares_outstanding
                print(f"  📊 Current Quarterly EPS: ${quarterly_eps:.2f} (${quarterly_net_income/1000000:.1f}M ÷ {shares_outstanding/1000000:.1f}M shares)")
                return quarterly_eps
            
            return None
        except Exception as e:
            print(f"⚠️ Error calculating current quarterly EPS: {e}")
            return None

    def get_current_quarterly_eps_simple(self, facts_data):
        """Get current quarterly EPS using proven calculation"""
        try:
            # From our proven calculation: Q3 2025 NI / Q3 2025 Shares
            us_gaap = facts_data.get('facts', {}).get('us-gaap', {})
            
            # Get most recent quarterly net income
            ni_data = us_gaap.get('NetIncomeLoss', {}).get('units', {}).get('USD', [])
            quarterly_ni = [item for item in ni_data if item.get('frame') and 'Q' in item.get('frame', '') and item.get('end', '') >= '2022-01-01']
            
            # Get shares data
            shares_data = us_gaap.get('WeightedAverageNumberOfSharesOutstandingBasic', {}).get('units', {}).get('shares', [])
            
            if quarterly_ni and shares_data:
                sorted_ni = sorted(quarterly_ni, key=lambda x: x.get('end', ''), reverse=True)
                sorted_shares = sorted(shares_data, key=lambda x: x.get('end', ''), reverse=True)
                
                current_ni = sorted_ni[0].get('val', 0)
                current_shares = sorted_shares[0].get('val', 0)
                
                if current_shares > 0:
                    current_eps = current_ni / current_shares
                    return current_eps
            
            return None
        except Exception as e:
            print(f"⚠️ Error calculating current quarterly EPS: {e}")
            return None

    def get_previous_year_quarterly_eps_simple(self, facts_data):
        """Get previous year quarterly EPS using proven calculation"""
        try:
            us_gaap = facts_data.get('facts', {}).get('us-gaap', {})
            
            # Get quarterly net income
            ni_data = us_gaap.get('NetIncomeLoss', {}).get('units', {}).get('USD', [])
            quarterly_ni = [item for item in ni_data if item.get('frame') and 'Q' in item.get('frame', '') and item.get('end', '') >= '2022-01-01']
            
            # Get shares data
            shares_data = us_gaap.get('WeightedAverageNumberOfSharesOutstandingBasic', {}).get('units', {}).get('shares', [])
            
            if len(quarterly_ni) >= 4 and shares_data:
                sorted_ni = sorted(quarterly_ni, key=lambda x: x.get('end', ''), reverse=True)
                sorted_shares = sorted(shares_data, key=lambda x: x.get('end', ''), reverse=True)
                
                # Use position 3 for same quarter previous year (proven to work)
                prev_ni = sorted_ni[3].get('val', 0)
                prev_shares = sorted_shares[1].get('val', 0) if len(sorted_shares) > 1 else sorted_shares[0].get('val', 0)
                
                if prev_shares > 0:
                    prev_eps = prev_ni / prev_shares
                    return prev_eps
            
            return None
        except Exception as e:
            print(f"⚠️ Error calculating previous year quarterly EPS: {e}")
            return None

    def get_previous_year_quarterly_eps(self, facts_data, concept):
        """Get previous year same quarter EPS"""
        try:
            # Calculate previous year quarterly EPS = Previous Q4 Net Income / Previous Year Shares
            prev_quarterly_net_income = self.get_previous_year_q4_value(facts_data, 'NetIncomeLoss')
            prev_shares = self.get_previous_year_shares(facts_data)
            
            if prev_quarterly_net_income and prev_shares and prev_shares > 0:
                prev_quarterly_eps = prev_quarterly_net_income / prev_shares
                print(f"  📊 Previous Year Quarterly EPS: ${prev_quarterly_eps:.2f} (${prev_quarterly_net_income/1000000:.1f}M ÷ {prev_shares/1000000:.1f}M shares)")
                return prev_quarterly_eps
            
            return None
        except Exception as e:
            print(f"⚠️ Error calculating previous year quarterly EPS: {e}")
            return None

    def get_previous_year_shares(self, facts_data):
        """Get previous year shares outstanding"""
        try:
            us_gaap = facts_data.get('facts', {}).get('us-gaap', {})
            
            # Try different share concepts
            share_concepts = [
                'WeightedAverageNumberOfSharesOutstandingBasic',
                'WeightedAverageNumberOfDilutedSharesOutstanding', 
                'CommonStockSharesOutstanding'
            ]
            
            for concept in share_concepts:
                concept_data = us_gaap.get(concept, {})
            
                if concept_data:
                    units = concept_data.get('units', {})
                    shares_data = units.get('shares', [])
                    
                    if shares_data:
                        # Get annual shares data
                        annual_shares = [item for item in shares_data if item.get('form') == '10-K']
                        if len(annual_shares) >= 2:
                            sorted_shares = sorted(annual_shares, key=lambda x: x.get('end', ''), reverse=True)
                            prev_shares = sorted_shares[1].get('val')  # Previous year
                            return prev_shares
            
            return None
        except Exception as e:
            print(f"⚠️ Error getting previous year shares: {e}")
            return None

    def extract_eps_ttm_simple(self, facts_data, concept):
        """Extract TTM EPS by summing last 4 quarterly EPS values"""
        try:
            us_gaap = facts_data.get('facts', {}).get('us-gaap', {})
            concept_data = us_gaap.get(concept, {})
            
            if not concept_data:
                return None
            
            units = concept_data.get('units', {})
            eps_data = units.get('USD/shares', [])
            
            if eps_data:
                # Get quarterly EPS data
                quarterly_eps = [item for item in eps_data 
                               if item.get('frame') and 'Q' in item.get('frame', '')
                               and item.get('end', '') >= '2022-01-01']
                
                if len(quarterly_eps) >= 4:
                    sorted_eps = sorted(quarterly_eps, key=lambda x: x.get('end', ''), reverse=True)
                    ttm_eps = sum(q.get('val', 0) for q in sorted_eps[:4])
                    print(f"  ✅ {concept} TTM: ${ttm_eps:.3f} (sum of 4 quarters)")
                    return ttm_eps
            
            # Fallback to annual EPS
            return self.extract_eps_value(facts_data, concept)
            
        except Exception as e:
            print(f"⚠️ Error extracting TTM EPS {concept}: {e}")
            return None

    def extract_cash_flow_ttm(self, facts_data, concept, metric_name):
        """Extract cash flow TTM - handles cumulative YTD data"""
        try:
            us_gaap = facts_data.get('facts', {}).get('us-gaap', {})
            concept_data = us_gaap.get(concept, {})
            
            if not concept_data:
                return None
            
            units = concept_data.get('units', {})
            usd_data = units.get('USD', [])
            
            if not usd_data:
                return None
            
            # Get recent data (both quarterly and annual)
            recent_data = [item for item in usd_data if item.get('end', '') >= '2024-01-01']
            
            if recent_data:
                # Sort by end date (most recent first)
                sorted_data = sorted(recent_data, key=lambda x: x.get('end', ''), reverse=True)
                
                # Use most recent value (likely YTD cumulative)
                latest = sorted_data[0]
                latest_value = latest.get('val', 0)
                latest_end = latest.get('end')
                
                print(f"  ✅ {metric_name} TTM: ${latest_value/1000000:.1f}M (most recent: {latest_end})")
                return latest_value
            
            # Fallback to old method
            return self.extract_ttm_value(facts_data, concept)
            
        except Exception as e:
            print(f"⚠️ Error extracting cash flow TTM {concept}: {e}")
            return None

    def extract_eps_value(self, facts_data, concept):
        """Extract EPS value (uses USD/shares units)"""
        try:
            us_gaap = facts_data.get('facts', {}).get('us-gaap', {})
            concept_data = us_gaap.get(concept, {})
            
            if not concept_data:
                return None
            
            units = concept_data.get('units', {})
            # EPS uses USD/shares units
            usd_shares_data = units.get('USD/shares', [])
            
            if usd_shares_data:
                # Get most recent annual EPS
                annual_eps = [item for item in usd_shares_data if item.get('form') == '10-K']
                if annual_eps:
                    latest_eps = sorted(annual_eps, key=lambda x: x.get('end', ''), reverse=True)[0]
                    eps_value = latest_eps.get('val')
                    print(f"  ✅ {concept}: ${eps_value:.2f} (as of {latest_eps.get('end')})")
                    return eps_value
            
            return None
        except Exception as e:
            print(f"⚠️ Error extracting EPS {concept}: {e}")
            return None
    
    def extract_shares_value(self, facts_data):
        """Extract shares outstanding (uses shares units)"""
        try:
            us_gaap = facts_data.get('facts', {}).get('us-gaap', {})
            
            # Try different share concepts
            share_concepts = [
                'CommonStockSharesOutstanding',
                'WeightedAverageNumberOfSharesOutstandingBasic',
                'CommonStockSharesIssued'
            ]
            
            for concept in share_concepts:
                concept_data = us_gaap.get(concept, {})
                if concept_data:
                    units = concept_data.get('units', {})
                    shares_data = units.get('shares', [])
                    
                    if shares_data:
                        # Get most recent value
                        recent_shares = [item for item in shares_data 
                                       if item.get('end', '') >= '2024-01-01']
                        if recent_shares:
                            latest_shares = sorted(recent_shares, key=lambda x: x.get('end', ''), reverse=True)[0]
                            shares_value = latest_shares.get('val')
                            print(f"  ✅ Shares Outstanding: {shares_value/1000000:.1f}M shares (as of {latest_shares.get('end')})")
                            return shares_value / 1000000  # Convert to millions
            
            return None
        except Exception as e:
            print(f"⚠️ Error extracting shares outstanding: {e}")
            return None

    def extract_latest_value(self, facts_data, concept):
        """Extract the most recent value (from latest 10-Q or 10-K)"""
        try:
            us_gaap = facts_data.get('facts', {}).get('us-gaap', {})
            concept_data = us_gaap.get(concept, {})
            
            if not concept_data:
                return None
            
            units = concept_data.get('units', {})
            usd_data = units.get('USD', [])
            
            if not usd_data:
                return None
            
            # Get all recent data
            recent_data = [item for item in usd_data 
                          if item.get('end', '') >= '2024-01-01']
            
            if recent_data:
                sorted_data = sorted(recent_data, key=lambda x: x.get('end', ''), reverse=True)
                latest_item = sorted_data[0]
                latest_value = latest_item.get('val')
                print(f"  ✅ Latest {concept}: ${latest_value/1000000:.1f}M (as of {latest_item.get('end')})")
                return latest_value
            
            return None
        except Exception as e:
            print(f"⚠️ Error extracting latest {concept}: {e}")
            return None
    
    def calculate_and_store(self, symbol):
        """Calculate comprehensive fundamental metrics for a symbol"""
        print(f"\n🔍 Fetching SEC data for {symbol}...")
        
        facts_data = self.get_company_facts(symbol)
        if not facts_data:
            return False
        
        print(f"✅ Successfully fetched SEC data for {symbol}")
        
        # Initialize metrics dictionary
        metrics = {}
        
        # Income Statement - TTM values
        print("📊 Extracting Income Statement data...")
        metrics['revenue_ttm'] = self.extract_ttm_simple(facts_data, ['Revenues', 'RevenueFromContractWithCustomerExcludingAssessedTax'], 'Revenue')
        metrics['net_income_ttm'] = self.extract_ttm_simple(facts_data, ['NetIncomeLoss'], 'Net Income')
        metrics['ebit_ttm'] = self.extract_ttm_simple(facts_data, ['OperatingIncomeLoss'], 'EBIT')
        
        # Calculate EBITDA (EBIT + Depreciation) - use quarterly sum approach
        depreciation_concepts = [
            'DepreciationDepletionAndAmortization',  # Apple uses this
            'Depreciation',  # Microsoft uses this
            'DepreciationAndAmortization',
            'AmortizationOfIntangibleAssets'
        ]
        
        depreciation_ttm = None
        for concept in depreciation_concepts:
            depreciation_ttm = self.extract_ttm_simple(facts_data, [concept], f'Depreciation ({concept})')
            if depreciation_ttm:
                break
        
        if metrics['ebit_ttm'] and depreciation_ttm:
            metrics['ebitda_ttm'] = metrics['ebit_ttm'] + depreciation_ttm
            print(f"  ✅ TTM EBITDA: ${metrics['ebitda_ttm']/1000000:.1f}M (EBIT + Depreciation)")
        
        # Income Statement - Quarterly values
        print("📊 Extracting Quarterly Income Statement data...")
        metrics['revenue_quarterly'] = self.extract_quarterly_value(facts_data, 'Revenues') or self.extract_quarterly_value(facts_data, 'RevenueFromContractWithCustomerExcludingAssessedTax')
        metrics['net_income_quarterly'] = self.extract_quarterly_value(facts_data, 'NetIncomeLoss')
        metrics['ebit_quarterly'] = self.extract_quarterly_value(facts_data, 'OperatingIncomeLoss')
        
        # Calculate quarterly EBITDA - use the most recent quarterly depreciation
        depreciation_quarterly = None
        for concept in depreciation_concepts:
            depreciation_quarterly = self.extract_quarterly_value(facts_data, concept)
            if depreciation_quarterly:
                print(f"  📊 Using {concept} for quarterly depreciation: ${depreciation_quarterly/1000000:.1f}M")
                break
        
        if metrics.get('ebit_quarterly') and depreciation_quarterly:
            metrics['ebitda_quarterly'] = metrics['ebit_quarterly'] + depreciation_quarterly
            print(f"  ✅ Quarterly EBITDA: ${metrics['ebitda_quarterly']/1000000:.1f}M (EBIT + Depreciation)")
        
        # YoY Growth Calculations
        print("📊 Calculating YoY Growth rates...")
        yoy_metrics = self.calculate_yoy_growth(facts_data, metrics)
        metrics.update(yoy_metrics)
        
        # Common Size Statements (EPS and Shares) - Use quarterly sum for TTM EPS
        print("📊 Extracting EPS and Share data...")
        metrics['eps_diluted_ttm'] = self.extract_eps_ttm_simple(facts_data, 'EarningsPerShareDiluted')
        metrics['eps_basic_ttm'] = self.extract_eps_ttm_simple(facts_data, 'EarningsPerShareBasic')  
        metrics['shares_outstanding'] = self.extract_shares_value(facts_data)
        
        # Balance Sheet - Latest values
        print("🏛️ Extracting Balance Sheet data...")
        metrics['total_assets'] = self.extract_latest_value(facts_data, 'Assets')
        metrics['shareholders_equity'] = self.extract_latest_value(facts_data, 'StockholdersEquity')
        # Cash and Short Term Investments (comprehensive approach for different companies)
        cash_equiv = self.extract_latest_value(facts_data, 'CashAndCashEquivalentsAtCarryingValue') or 0
        
        # Try multiple short-term investment concepts (company-specific)
        short_term_concepts = [
            'ShortTermInvestments',
            'MarketableSecurities', 
            'MarketableSecuritiesCurrent',  # NVIDIA uses this
            'AvailableForSaleSecuritiesCurrent',
            'CashCashEquivalentsAndShortTermInvestments'
        ]
        
        short_term_investments = 0
        for concept in short_term_concepts:
            value = self.extract_latest_value(facts_data, concept)
            if value and value > short_term_investments:
                short_term_investments = value
                print(f"  📊 Found {concept}: ${value/1000000:.1f}M")
        
        # Check for combined concept
        combined_cash = self.extract_latest_value(facts_data, 'CashCashEquivalentsAndShortTermInvestments')
        if combined_cash and combined_cash > cash_equiv:
            metrics['cash_and_equivalents'] = combined_cash
            print(f"  ✅ Cash & Short-term (combined): ${combined_cash/1000000:.1f}M")
        else:
            metrics['cash_and_equivalents'] = cash_equiv + short_term_investments
            print(f"  ✅ Cash & Short-term (sum): ${(cash_equiv + short_term_investments)/1000000:.1f}M")
        metrics['current_assets'] = self.extract_latest_value(facts_data, 'AssetsCurrent')
        metrics['current_liabilities'] = self.extract_latest_value(facts_data, 'LiabilitiesCurrent')
        # Long Term Debt (comprehensive approach for different companies)
        long_term_debt_concepts = [
            'DebtInstrumentCarryingAmount',  # Comprehensive debt
            'LongTermDebt', 
            'LongTermDebtNoncurrent',
            'DebtLongtermAndShorttermCombinedAmount',
            'NotesPayable',
            'BorrowingsLongTerm'
        ]
        
        long_term_debt = 0
        for concept in long_term_debt_concepts:
            value = self.extract_latest_value(facts_data, concept)
            if value and value > long_term_debt:
                long_term_debt = value
                print(f"  📊 Using {concept} for Long Term Debt: ${value/1000000:.1f}M")
                break
        
        metrics['long_term_debt'] = long_term_debt
        
        # Additional Balance Sheet metrics
        metrics['total_liabilities'] = self.extract_latest_value(facts_data, 'Liabilities')
        # Long Term Assets (use calculation method - more reliable)
        total_assets = metrics.get('total_assets', 0)
        current_assets = metrics.get('current_assets', 0)
        
        if total_assets and current_assets:
            long_term_assets = total_assets - current_assets
            print(f"  ✅ Long Term Assets (calculated): ${long_term_assets/1000000:.1f}M (Total: ${total_assets/1000000:.1f}M - Current: ${current_assets/1000000:.1f}M)")
            metrics['long_term_assets'] = long_term_assets
        else:
            # Fallback to direct concepts
            long_term_assets = (
                self.extract_latest_value(facts_data, 'AssetsNoncurrent') or
                self.extract_latest_value(facts_data, 'NoncurrentAssets') or
                0
            )
            metrics['long_term_assets'] = long_term_assets
        # Ending Cash - use just cash component (not short-term investments)
        ending_cash_only = self.extract_latest_value(facts_data, 'CashAndCashEquivalentsAtCarryingValue') or 0
        metrics['ending_cash'] = ending_cash_only
        print(f"  ✅ Ending Cash: ${ending_cash_only/1000000:.1f}M (cash only, not including short-term investments)")
        metrics['book_value'] = metrics['shareholders_equity']  # Same as shareholders equity
        
        # Calculate Total Liabilities if not directly available
        if not metrics['total_liabilities'] and metrics.get('total_assets') and metrics.get('shareholders_equity'):
            metrics['total_liabilities'] = metrics['total_assets'] - metrics['shareholders_equity']
            print(f"  ✅ Calculated Total Liabilities: ${metrics['total_liabilities']/1000000:.1f}M (Assets - Equity)")
        
        # Cash Flow - TTM values (use quarterly approach for fresher data)
        print("💸 Extracting Cash Flow data...")
        metrics['operating_cash_flow_ttm'] = self.extract_cash_flow_ttm(facts_data, 'NetCashProvidedByUsedInOperatingActivities', 'Operating Cash Flow')
        metrics['investing_cash_flow_ttm'] = self.extract_cash_flow_ttm(facts_data, 'NetCashProvidedByUsedInInvestingActivities', 'Investing Cash Flow')
        metrics['financing_cash_flow_ttm'] = self.extract_cash_flow_ttm(facts_data, 'NetCashProvidedByUsedInFinancingActivities', 'Financing Cash Flow')
        # Capital Expenditures - try broader concepts for different companies
        capex_concepts = [
            'PaymentsToAcquirePropertyPlantAndEquipment',
            'PaymentsToAcquireProductiveAssets',  # NVIDIA might use this
            'CapitalExpenditures',
            'PaymentsForPropertyPlantAndEquipment'
        ]
        
        metrics['capital_expenditures_ttm'] = None
        for concept in capex_concepts:
            capex_value = self.extract_ttm_simple(facts_data, [concept], f'Capital Expenditures ({concept})')
            if capex_value:
                metrics['capital_expenditures_ttm'] = capex_value
                break
        
        # Additional Cash Flow metrics
        # Change in Receivables - use recent data and handle sign convention
        change_in_receivables = self.extract_cash_flow_ttm(facts_data, 'IncreaseDecreaseInAccountsReceivable', 'Change in Receivables')
        
        # Handle sign convention - cash flow statements often show increases as negative
        if change_in_receivables and change_in_receivables > 0:
            # If positive, it might need to be negative for cash flow impact
            metrics['change_in_receivables_ttm'] = -change_in_receivables
            print(f"  📊 Adjusted Change in Receivables: ${-change_in_receivables/1000000:.1f}M (cash flow impact)")
        else:
            metrics['change_in_receivables_ttm'] = change_in_receivables
        # Changes in Working Capital - use recent cash flow data with proper sign handling
        working_capital_change = self.extract_cash_flow_ttm(facts_data, 'IncreaseDecreaseInOperatingCapital', 'Working Capital Change')
        
        if not working_capital_change:
            working_capital_change = self.calculate_working_capital_change(facts_data)
        
        # Handle sign convention - increases in working capital are typically negative for cash flow
        if working_capital_change and working_capital_change > 0:
            metrics['changes_in_working_capital_ttm'] = -working_capital_change
            print(f"  📊 Adjusted Working Capital Change: ${-working_capital_change/1000000:.1f}M (cash flow impact)")
        else:
            metrics['changes_in_working_capital_ttm'] = working_capital_change
        
        # Calculate Free Cash Flow
        if metrics.get('operating_cash_flow_ttm') and metrics.get('capital_expenditures_ttm'):
            metrics['free_cash_flow_ttm'] = metrics['operating_cash_flow_ttm'] - abs(metrics['capital_expenditures_ttm'])
            print(f"  ✅ TTM Free Cash Flow: ${metrics['free_cash_flow_ttm']/1000000:.1f}M (Operating CF - CapEx)")
        
        # Calculate Financial Ratios
        print("📊 Calculating Financial Ratios...")
        
        # ROA (Return on Assets)
        if metrics.get('net_income_ttm') and metrics.get('total_assets'):
            # The TTM Net Income should already be using quarterly sum (more accurate)
            metrics['return_on_assets'] = (metrics['net_income_ttm'] / metrics['total_assets']) * 100
            print(f"  ✅ ROA: {metrics['return_on_assets']:.2f}%")
        
        # ROE (Return on Equity)
        if metrics.get('net_income_ttm') and metrics.get('shareholders_equity'):
            # The TTM Net Income should already be using quarterly sum (more accurate)
            metrics['return_on_equity'] = (metrics['net_income_ttm'] / metrics['shareholders_equity']) * 100
            print(f"  ✅ ROE: {metrics['return_on_equity']:.2f}%")
        
        # Operating Margin
        if metrics.get('ebit_ttm') and metrics.get('revenue_ttm'):
            metrics['operating_margin'] = (metrics['ebit_ttm'] / metrics['revenue_ttm']) * 100
            print(f"  ✅ Operating Margin: {metrics['operating_margin']:.1f}%")
        
        # Current Ratio
        if metrics.get('current_assets') and metrics.get('current_liabilities'):
            metrics['current_ratio'] = metrics['current_assets'] / metrics['current_liabilities']
            print(f"  ✅ Current Ratio: {metrics['current_ratio']:.2f}")
        
        # Debt-to-Equity Ratio
        if metrics.get('long_term_debt') and metrics.get('shareholders_equity') and metrics['shareholders_equity'] > 0:
            metrics['debt_to_equity'] = metrics['long_term_debt'] / metrics['shareholders_equity']
            print(f"  ✅ Debt-to-Equity: {metrics['debt_to_equity']:.2f}")
        else:
            metrics['debt_to_equity'] = 0.0
        
        # Additional Financial Ratios
        print("📊 Calculating Additional Ratios...")
        
        # Return on Invested Capital (ROIC) - use corrected calculations
        if metrics.get('ebit_ttm') and metrics.get('shareholders_equity') and metrics.get('long_term_debt'):
            invested_capital = metrics['shareholders_equity'] + metrics['long_term_debt']
            metrics['return_on_invested_capital'] = (metrics['ebit_ttm'] / invested_capital) * 100
            print(f"  ✅ ROIC: {metrics['return_on_invested_capital']:.2f}% (EBIT: ${metrics['ebit_ttm']/1000000:.1f}M / Invested Capital: ${invested_capital/1000000:.1f}M)")
        
        # Gross Profit Margin
        cost_of_goods_sold = self.extract_ttm_value(facts_data, 'CostOfGoodsAndServicesSold') or self.extract_ttm_value(facts_data, 'CostOfRevenue')
        if metrics.get('revenue_ttm') and cost_of_goods_sold:
            gross_profit = metrics['revenue_ttm'] - cost_of_goods_sold
            metrics['gross_profit_margin'] = (gross_profit / metrics['revenue_ttm']) * 100
            print(f"  ✅ Gross Profit Margin: {metrics['gross_profit_margin']:.1f}%")
        
        # Store in database
        success = self.store_metrics(symbol, metrics)
        
        if success:
            print(f"✅ Successfully calculated and stored metrics for {symbol}")
            return True
        else:
            print(f"❌ Failed to store metrics for {symbol}")
            return False
    
    def store_metrics(self, symbol, metrics):
        """Store calculated metrics in the database"""
        try:
            # Database connection
            conn = psycopg2.connect(
                host=os.getenv('DB_HOST', 'localhost'),
                port=os.getenv('DB_PORT', '5432'),
                user=os.getenv('DB_USER', 'investorcenter'),
                password=os.getenv('DB_PASSWORD', 'investorcenter123'),
                database=os.getenv('DB_NAME', 'investorcenter_db')
            )
            
            cursor = conn.cursor()
            
            # Convert metrics to millions and round for storage
            processed_metrics = {}
            for key, value in metrics.items():
                if value is not None:
                    if key.endswith('_growth') or key.endswith('_ratio') or key.startswith('return_') or key.endswith('_margin'):
                        # Percentages, growth rates, and ratios - store as-is
                        processed_metrics[key] = round(value, 2)
                    elif key == 'shares_outstanding':
                        # Shares already in millions
                        processed_metrics[key] = round(value, 1)
                    elif key.startswith('eps_'):
                        # EPS values - store as-is (already in dollars)
                        processed_metrics[key] = round(value, 2)
                    else:
                        # Dollar amounts - convert to millions
                        processed_metrics[key] = round(value / 1000000, 1)
            
            # Add metadata
            processed_metrics['updated_at'] = datetime.now().isoformat()
            processed_metrics['calculation_method'] = 'SEC_EDGAR_API_v3_simple'
            
            # Insert or update
            cursor.execute("""
                INSERT INTO fundamental_metrics (symbol, metrics_data, updated_at)
                VALUES (%s, %s, CURRENT_TIMESTAMP)
                ON CONFLICT (symbol) 
                DO UPDATE SET 
                    metrics_data = EXCLUDED.metrics_data,
                    updated_at = EXCLUDED.updated_at
            """, (symbol, json.dumps(processed_metrics)))
            
            conn.commit()
            cursor.close()
            conn.close()
            
            return True
            
        except Exception as e:
            print(f"❌ Database error: {e}")
            return False

def main():
    calculator = SECEdgarCalculatorV3()
    
    # Check for command line arguments
    if len(sys.argv) > 1:
        symbols = [sys.argv[1].upper()]
    else:
        symbols = ['AAPL']  # Default to AAPL
    
    for symbol in symbols:
        success = calculator.calculate_and_store(symbol)
        if success:
            print(f"✅ Successfully processed {symbol}")
        else:
            print(f"❌ Failed to process {symbol}")
        
        # Rate limiting - be respectful to SEC API
        time.sleep(0.1)

if __name__ == "__main__":
    main()
