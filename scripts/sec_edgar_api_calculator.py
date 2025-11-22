#!/usr/bin/env python3
"""
SEC EDGAR API Financial Calculator
Uses the official SEC EDGAR API to get accurate financial data
"""

import requests
import json
import psycopg2
from datetime import datetime
import sys
import os
import time

class SECEdgarCalculator:
    def __init__(self):
        self.base_url = "https://data.sec.gov/api/xbrl"
        self.headers = {
            'User-Agent': 'InvestorCenter.ai financial-data-collector contact@investorcenter.ai'
        }
        
        # Database connection
        self.db_conn = psycopg2.connect(
            host=os.getenv('DB_HOST', 'localhost'),
            port=os.getenv('DB_PORT', '5432'),
            user=os.getenv('DB_USER', 'investorcenter'),
            password=os.getenv('DB_PASSWORD', ''),
            database=os.getenv('DB_NAME', 'investorcenter_db')
        )
    
    def get_company_cik(self, symbol):
        """Get CIK for a symbol"""
        # Apple's CIK is 0000320193
        cik_mapping = {
            'AAPL': '0000320193',
            'MSFT': '0000789019',
            'GOOGL': '0001652044',
            'AMZN': '0001018724',
            'TSLA': '0001318605',
        }
        return cik_mapping.get(symbol.upper())
    
    def fetch_company_facts(self, symbol):
        """Fetch all company facts from SEC EDGAR API"""
        cik = self.get_company_cik(symbol)
        if not cik:
            raise ValueError(f"CIK not found for symbol {symbol}")
        
        url = f"https://data.sec.gov/api/xbrl/companyfacts/CIK{cik}.json"
        print(f"🔍 Fetching SEC data from: {url}")
        
        response = requests.get(url, headers=self.headers)
        response.raise_for_status()
        
        return response.json()
    
    def extract_latest_value(self, facts_data, concept, form_type="10-K"):
        """Extract the latest value for a financial concept"""
        try:
            us_gaap = facts_data.get('facts', {}).get('us-gaap', {})
            concept_data = us_gaap.get(concept, {})
            
            if not concept_data:
                return None
            
            units = concept_data.get('units', {})
            usd_data = units.get('USD', [])
            
            if not usd_data:
                return None
            
            # Filter for the form type and get most recent
            filtered_data = [item for item in usd_data if item.get('form') == form_type]
            if not filtered_data:
                # Fallback to any form type
                filtered_data = usd_data
            
            # Sort by filing date and get most recent
            sorted_data = sorted(filtered_data, key=lambda x: x.get('filed', ''), reverse=True)
            
            if sorted_data:
                return sorted_data[0].get('val')
            
            return None
        except Exception as e:
            print(f"⚠️ Error extracting {concept}: {e}")
            return None
    
    def extract_latest_value_with_units(self, facts_data, concept, unit_type, form_type="10-Q"):
        """Extract the latest value for a concept with specific units (like 'shares')"""
        try:
            us_gaap = facts_data.get('facts', {}).get('us-gaap', {})
            concept_data = us_gaap.get(concept, {})
            
            if not concept_data:
                return None
            
            units = concept_data.get('units', {})
            unit_data = units.get(unit_type, [])
            
            if not unit_data:
                return None
            
            # Filter for recent data and form type
            filtered_data = [item for item in unit_data 
                           if item.get('form') == form_type 
                           and item.get('end', '') >= '2024-01-01']
            
            if not filtered_data:
                # Fallback to any form type
                filtered_data = [item for item in unit_data if item.get('end', '') >= '2024-01-01']
            
            # Sort by end date and get most recent
            sorted_data = sorted(filtered_data, key=lambda x: x.get('end', ''), reverse=True)
            
            if sorted_data:
                return sorted_data[0].get('val')
            
            return None
        except Exception as e:
            print(f"⚠️ Error extracting {concept} with units {unit_type}: {e}")
            return None
    
    def extract_ttm_value(self, facts_data, concept):
        """Extract TTM (trailing twelve months) value by summing last 4 quarters"""
        try:
            us_gaap = facts_data.get('facts', {}).get('us-gaap', {})
            concept_data = us_gaap.get(concept, {})
            
            if not concept_data:
                return None
            
            units = concept_data.get('units', {})
            usd_data = units.get('USD', [])
            
            if not usd_data:
                return None
            
            # Get quarterly data (look for frame patterns like CY2025Q1, CY2025Q2, etc.)
            quarterly_data = [item for item in usd_data 
                            if item.get('frame') and 'Q' in item.get('frame', '') 
                            and item.get('end', '') >= '2024-01-01']
            
            # Sort by end date (most recent first)
            sorted_quarterly = sorted(quarterly_data, key=lambda x: x.get('end', ''), reverse=True)
            
            print(f"  🔍 Found {len(sorted_quarterly)} quarterly data points for {concept}")
            for i, item in enumerate(sorted_quarterly[:4]):
                print(f"    Q{i+1}: ${item.get('val', 0)/1000000:.1f}M ({item.get('start')} to {item.get('end')})")
            
            # Sum the last 4 quarters for TTM
            if len(sorted_quarterly) >= 4:
                ttm_value = sum(item.get('val', 0) for item in sorted_quarterly[:4])
                print(f"  ✅ TTM {concept}: ${ttm_value/1000000:.1f}M")
                return ttm_value
            
            # If not enough quarterly data, try to get the most recent annual data
            annual_data = [item for item in usd_data 
                          if item.get('form') == '10-K' 
                          and item.get('end', '') >= '2023-01-01']
            
            if annual_data:
                sorted_annual = sorted(annual_data, key=lambda x: x.get('end', ''), reverse=True)
                annual_value = sorted_annual[0].get('val')
                print(f"  ✅ Annual {concept}: ${annual_value/1000000:.1f}M (from {sorted_annual[0].get('end')})")
                return annual_value
            
            return None
        except Exception as e:
            print(f"⚠️ Error extracting TTM {concept}: {e}")
            return None
    
    def calculate_comprehensive_metrics(self, symbol):
        """Calculate comprehensive financial metrics using SEC EDGAR API"""
        print(f"🔍 Calculating metrics for {symbol} using SEC EDGAR API")
        
        # Fetch company facts
        facts_data = self.fetch_company_facts(symbol)
        
        print(f"✅ Successfully fetched SEC data for {symbol}")
        
        # Extract key financial metrics using proper XBRL concepts
        metrics = {
            'symbol': symbol,
            'updated_at': datetime.now().isoformat(),
        }
        
        # Income Statement - TTM values
        print("📊 Extracting Income Statement data...")
        metrics['revenue_ttm'] = self.extract_ttm_value(facts_data, 'Revenues') or self.extract_ttm_value(facts_data, 'RevenueFromContractWithCustomerExcludingAssessedTax')
        metrics['net_income_ttm'] = self.extract_ttm_value(facts_data, 'NetIncomeLoss')
        metrics['ebit_ttm'] = self.extract_ttm_value(facts_data, 'OperatingIncomeLoss')
        
        # Try to calculate EBITDA (EBIT + Depreciation)
        depreciation = self.extract_ttm_value(facts_data, 'DepreciationDepletionAndAmortization')
        if metrics['ebit_ttm'] and depreciation:
            metrics['ebitda_ttm'] = metrics['ebit_ttm'] + depreciation
        
        print(f"  💰 Revenue TTM: ${metrics['revenue_ttm']/1000000:.1f}M" if metrics['revenue_ttm'] else "  ❌ Revenue TTM: Not found")
        print(f"  💰 Net Income TTM: ${metrics['net_income_ttm']/1000000:.1f}M" if metrics['net_income_ttm'] else "  ❌ Net Income TTM: Not found")
        print(f"  💰 EBIT TTM: ${metrics['ebit_ttm']/1000000:.1f}M" if metrics['ebit_ttm'] else "  ❌ EBIT TTM: Not found")
        
        # Balance Sheet - Latest quarterly values
        print("🏛️ Extracting Balance Sheet data...")
        metrics['total_assets'] = self.extract_latest_value(facts_data, 'Assets', '10-Q') or self.extract_latest_value(facts_data, 'Assets', '10-K')
        metrics['total_liabilities'] = self.extract_latest_value(facts_data, 'Liabilities', '10-Q') or self.extract_latest_value(facts_data, 'Liabilities', '10-K')
        metrics['shareholders_equity'] = self.extract_latest_value(facts_data, 'StockholdersEquity', '10-Q') or self.extract_latest_value(facts_data, 'StockholdersEquity', '10-K')
        
        # Cash and cash equivalents
        cash_only = self.extract_latest_value(facts_data, 'CashAndCashEquivalentsAtCarryingValue', '10-Q') or self.extract_latest_value(facts_data, 'CashAndCashEquivalentsAtCarryingValue', '10-K')
        marketable_securities = self.extract_latest_value(facts_data, 'MarketableSecuritiesCurrent', '10-Q') or self.extract_latest_value(facts_data, 'MarketableSecuritiesCurrent', '10-K')
        
        # Total cash + short-term investments
        if cash_only and marketable_securities:
            metrics['cash_short_term_investments'] = cash_only + marketable_securities
            print(f"  💵 Cash + Short-term Investments: ${metrics['cash_short_term_investments']/1000000:.1f}M (Cash: ${cash_only/1000000:.1f}M + Securities: ${marketable_securities/1000000:.1f}M)")
        elif cash_only:
            metrics['cash_short_term_investments'] = cash_only
            print(f"  💵 Cash: ${metrics['cash_short_term_investments']/1000000:.1f}M")
        
        # Current assets and current liabilities for ratios
        metrics['current_assets'] = self.extract_latest_value(facts_data, 'AssetsCurrent', '10-Q') or self.extract_latest_value(facts_data, 'AssetsCurrent', '10-K')
        metrics['current_liabilities'] = self.extract_latest_value(facts_data, 'LiabilitiesCurrent', '10-Q') or self.extract_latest_value(facts_data, 'LiabilitiesCurrent', '10-K')
        
        # Book value = shareholders equity
        metrics['book_value'] = metrics['shareholders_equity']
        metrics['ending_cash'] = metrics['cash_short_term_investments']
        
        print(f"  🏛️ Total Assets: ${metrics['total_assets']/1000000:.1f}M" if metrics['total_assets'] else "  ❌ Total Assets: Not found")
        print(f"  🏛️ Shareholders Equity: ${metrics['shareholders_equity']/1000000:.1f}M" if metrics['shareholders_equity'] else "  ❌ Shareholders Equity: Not found")
        print(f"  🏛️ Current Assets: ${metrics['current_assets']/1000000:.1f}M" if metrics['current_assets'] else "  ❌ Current Assets: Not found")
        print(f"  🏛️ Current Liabilities: ${metrics['current_liabilities']/1000000:.1f}M" if metrics['current_liabilities'] else "  ❌ Current Liabilities: Not found")
        
        # Cash Flow - TTM values
        print("💸 Extracting Cash Flow data...")
        metrics['cash_from_operations'] = self.extract_ttm_value(facts_data, 'NetCashProvidedByUsedInOperatingActivities')
        metrics['cash_from_investing'] = self.extract_ttm_value(facts_data, 'NetCashProvidedByUsedInInvestingActivities')
        metrics['cash_from_financing'] = self.extract_ttm_value(facts_data, 'NetCashProvidedByUsedInFinancingActivities')
        metrics['capital_expenditures'] = self.extract_ttm_value(facts_data, 'PaymentsToAcquirePropertyPlantAndEquipment')
        
        # Calculate Free Cash Flow
        if metrics['cash_from_operations'] and metrics['capital_expenditures']:
            metrics['free_cash_flow'] = metrics['cash_from_operations'] - abs(metrics['capital_expenditures'])
        
        print(f"  💸 Operating Cash Flow: ${metrics['cash_from_operations']/1000000:.1f}M" if metrics['cash_from_operations'] else "  ❌ Operating Cash Flow: Not found")
        print(f"  💸 Free Cash Flow: ${metrics['free_cash_flow']/1000000:.1f}M" if metrics['free_cash_flow'] else "  ❌ Free Cash Flow: Not calculated")
        
        # Calculate key ratios
        print("📊 Calculating financial ratios...")
        if metrics['net_income_ttm'] and metrics['total_assets']:
            metrics['return_on_assets'] = (metrics['net_income_ttm'] / metrics['total_assets']) * 100
            print(f"  📊 ROA: {metrics['return_on_assets']:.1f}%")
        
        if metrics['net_income_ttm'] and metrics['shareholders_equity']:
            metrics['return_on_equity'] = (metrics['net_income_ttm'] / metrics['shareholders_equity']) * 100
            print(f"  📊 ROE: {metrics['return_on_equity']:.1f}%")
        
        if metrics['ebit_ttm'] and metrics['revenue_ttm']:
            metrics['operating_margin'] = (metrics['ebit_ttm'] / metrics['revenue_ttm']) * 100
            print(f"  📊 Operating Margin: {metrics['operating_margin']:.1f}%")
        
        # Current ratio
        if metrics.get('current_assets') and metrics.get('current_liabilities'):
            metrics['current_ratio'] = metrics['current_assets'] / metrics['current_liabilities']
            print(f"  📊 Current Ratio: {metrics['current_ratio']:.2f}")
        
        # Debt-to-equity ratio
        if metrics.get('total_long_term_debt') and metrics.get('shareholders_equity'):
            metrics['debt_to_equity_ratio'] = metrics['total_long_term_debt'] / metrics['shareholders_equity']
            print(f"  📊 Debt-to-Equity: {metrics['debt_to_equity_ratio']:.2f}")
        
        # EPS and Shares
        print("📈 Extracting EPS and Share data...")
        
        # For EPS, get the latest annual data (TTM EPS)
        metrics['eps_diluted'] = self.extract_latest_value(facts_data, 'EarningsPerShareDiluted', '10-K')
        metrics['eps_basic'] = self.extract_latest_value(facts_data, 'EarningsPerShareBasic', '10-K')
        
        # For shares outstanding, get the latest quarterly data
        try:
            us_gaap = facts_data.get('facts', {}).get('us-gaap', {})
            shares_concept = us_gaap.get('CommonStockSharesOutstanding', {})
            if shares_concept:
                shares_units = shares_concept.get('units', {}).get('shares', [])
                recent_shares = [item for item in shares_units if item.get('end', '') >= '2024-01-01']
                if recent_shares:
                    latest_shares = sorted(recent_shares, key=lambda x: x.get('end', ''), reverse=True)[0]
                    metrics['shares_outstanding'] = latest_shares.get('val')
                    print(f"  📈 Shares Outstanding: {metrics['shares_outstanding']/1000000:.1f}M shares (from {latest_shares.get('end')})")
        except Exception as e:
            print(f"  ⚠️ Error extracting shares: {e}")
        
        print(f"  📈 EPS Diluted (Annual): ${metrics['eps_diluted']:.2f}" if metrics['eps_diluted'] else "  ❌ EPS Diluted: Not found")
        print(f"  📈 EPS Basic (Annual): ${metrics['eps_basic']:.2f}" if metrics['eps_basic'] else "  ❌ EPS Basic: Not found")
        print(f"  📈 Shares Outstanding: {metrics['shares_outstanding']/1000000:.1f}M shares" if metrics['shares_outstanding'] else "  ❌ Shares Outstanding: Not found")
        
        # Extract additional metrics
        print("🔍 Extracting additional metrics...")
        
        # Long-term debt
        metrics['total_long_term_debt'] = self.extract_latest_value(facts_data, 'LongTermDebt', '10-Q') or self.extract_latest_value(facts_data, 'LongTermDebt', '10-K')
        print(f"  🏛️ Long-term Debt: ${metrics['total_long_term_debt']/1000000:.1f}M" if metrics['total_long_term_debt'] else "  ❌ Long-term Debt: Not found")
        
        # Cost of goods sold for gross margin calculation
        cost_of_sales = self.extract_ttm_value(facts_data, 'CostOfGoodsAndServicesSold') or self.extract_ttm_value(facts_data, 'CostOfRevenue')
        if cost_of_sales and metrics['revenue_ttm']:
            gross_profit = metrics['revenue_ttm'] - cost_of_sales
            metrics['gross_profit_margin'] = (gross_profit / metrics['revenue_ttm']) * 100
            print(f"  📊 Gross Profit Margin: {metrics['gross_profit_margin']:.1f}%")
        
        # Employee count (from 10-K)
        metrics['total_employees'] = self.extract_latest_value(facts_data, 'Employees', '10-K')
        if metrics['total_employees']:
            print(f"  👥 Total Employees: {metrics['total_employees']:,.0f}")
            
            # Calculate per-employee metrics
            if metrics['revenue_ttm']:
                metrics['revenue_per_employee'] = (metrics['revenue_ttm'] * 1000000) / metrics['total_employees']  # Convert back to actual dollars
                print(f"  👥 Revenue per Employee: ${metrics['revenue_per_employee']:,.0f}")
            
            if metrics['net_income_ttm']:
                metrics['net_income_per_employee'] = (metrics['net_income_ttm'] * 1000000) / metrics['total_employees']
                print(f"  👥 Net Income per Employee: ${metrics['net_income_per_employee']:,.0f}")
        else:
            print(f"  ❌ Employee count: Not found")
        
        # Convert to millions for storage (to match our database format)
        for key in ['revenue_ttm', 'net_income_ttm', 'ebit_ttm', 'ebitda_ttm', 'total_assets', 
                   'total_liabilities', 'shareholders_equity', 'cash_short_term_investments',
                   'book_value', 'ending_cash', 'cash_from_operations', 'cash_from_investing',
                   'cash_from_financing', 'capital_expenditures', 'free_cash_flow']:
            if metrics.get(key):
                metrics[key] = metrics[key] / 1000000  # Convert to millions
        
        # Convert shares to millions
        if metrics.get('shares_outstanding'):
            metrics['shares_outstanding'] = metrics['shares_outstanding'] / 1000000
        
        # Set market data placeholders
        market_fields = ['market_cap', 'pe_ratio', 'price_to_book', 'dividend_yield',
                        'one_month_returns', 'three_month_returns', 'six_month_returns',
                        'year_to_date_returns', 'one_year_returns', 'three_year_returns',
                        'five_year_returns', 'fifty_two_week_high', 'fifty_two_week_low',
                        'alpha_5y', 'beta_5y']
        
        for field in market_fields:
            metrics[field] = 'needs data'
        
        return metrics
    
    def store_metrics(self, metrics):
        """Store calculated metrics in the database"""
        print(f"💾 Storing SEC EDGAR metrics for {metrics['symbol']}")
        
        cursor = self.db_conn.cursor()
        
        # Convert metrics to JSON
        metrics_json = json.dumps(metrics, default=str)
        
        # Upsert query
        query = """
            INSERT INTO fundamental_metrics (symbol, metrics_data, updated_at) 
            VALUES (%s, %s, %s)
            ON CONFLICT (symbol) 
            DO UPDATE SET 
                metrics_data = EXCLUDED.metrics_data,
                updated_at = EXCLUDED.updated_at
        """
        
        cursor.execute(query, (
            metrics['symbol'],
            metrics_json,
            datetime.now()
        ))
        
        self.db_conn.commit()
        cursor.close()
        
        print(f"✅ Successfully stored SEC EDGAR metrics for {metrics['symbol']}")
    
    def calculate_and_store(self, symbol):
        """Calculate and store comprehensive metrics for a symbol"""
        try:
            print(f"{'='*60}")
            print(f"SEC EDGAR API CALCULATION FOR {symbol}")
            print(f"{'='*60}")
            
            metrics = self.calculate_comprehensive_metrics(symbol)
            
            if metrics:
                self.store_metrics(metrics)
                
                # Test retrieval
                print(f"\n🔍 Testing data retrieval...")
                cursor = self.db_conn.cursor()
                cursor.execute("""
                    SELECT 
                        metrics_data->>'revenue_ttm', 
                        metrics_data->>'net_income_ttm', 
                        metrics_data->>'total_assets',
                        metrics_data->>'return_on_equity'
                    FROM fundamental_metrics 
                    WHERE symbol = %s
                """, (symbol,))
                result = cursor.fetchone()
                if result:
                    print(f"📊 Stored: Revenue TTM: ${result[0]}M, Net Income TTM: ${result[1]}M")
                    print(f"📊 Stored: Total Assets: ${result[2]}M, ROE: {result[3]}%")
                cursor.close()
                
                return True
            else:
                print(f"❌ No metrics calculated for {symbol}")
                return False
        except Exception as e:
            print(f"❌ Error calculating metrics for {symbol}: {e}")
            import traceback
            traceback.print_exc()
            return False

def main():
    calculator = SECEdgarCalculator()
    
    symbols = ['AAPL']  # Start with AAPL
    
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
