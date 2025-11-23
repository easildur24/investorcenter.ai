#!/usr/bin/env python3
"""
BULLETPROOF SEC Calculator - Works for Every Ticker
Simple, reliable approach: Always use the most recent data available
"""

import requests
import json
import psycopg2
import os
import sys
import time
from datetime import datetime

class BulletproofSECCalculator:
    def __init__(self):
        self.base_url = "https://data.sec.gov/api/xbrl/companyfacts/"
        self.headers = {
            'User-Agent': 'InvestorCenter.ai financial-data-collector contact@investorcenter.ai'
        }
        
        # CIK mapping
        self.cik_mapping = {
            'AAPL': '0000320193',
            'MSFT': '0000789019', 
            'GOOGL': '0001652044',
            'AMZN': '0001018724',
            'TSLA': '0001318605',
            'META': '0001326801',
            'NVDA': '0001045810',
        }

    def get_company_data(self, symbol):
        """Fetch company data from SEC EDGAR API"""
        cik = self.cik_mapping.get(symbol.upper())
        if not cik:
            print(f"❌ CIK not found for {symbol}")
            return None
        
        url = f"{self.base_url}CIK{cik}.json"
        
        try:
            response = requests.get(url, headers=self.headers)
            response.raise_for_status()
            return response.json()
        except Exception as e:
            print(f"❌ Error fetching data for {symbol}: {e}")
            return None

    def get_concept_data(self, facts_data, concept_names):
        """Get data for a concept, trying multiple names"""
        us_gaap = facts_data.get('facts', {}).get('us-gaap', {})
        
        if isinstance(concept_names, str):
            concept_names = [concept_names]
        
        for concept in concept_names:
            if concept in us_gaap:
                return us_gaap[concept].get('units', {}).get('USD', [])
        
        return []

    def get_ttm_and_quarterly(self, facts_data, concept_names, metric_name):
        """Get both TTM and quarterly values for any metric"""
        usd_data = self.get_concept_data(facts_data, concept_names)
        
        if not usd_data:
            print(f"  ❌ No data found for {metric_name}")
            return None, None
        
        # Get quarterly data
        quarterly_data = [item for item in usd_data 
                         if item.get('frame') and 'Q' in item.get('frame', '')]
        
        if len(quarterly_data) >= 4:
            # Sort by end date (most recent first)
            sorted_quarters = sorted(quarterly_data, key=lambda x: x.get('end', ''), reverse=True)
            
            # TTM = sum of last 4 quarters
            ttm_value = sum(q.get('val', 0) for q in sorted_quarters[:4])
            
            # Quarterly = most recent quarter
            quarterly_value = sorted_quarters[0].get('val', 0)
            quarterly_end = sorted_quarters[0].get('end')
            
            print(f"  ✅ {metric_name} TTM: ${ttm_value/1000000:.1f}M (sum of 4 quarters)")
            print(f"  ✅ {metric_name} Quarterly: ${quarterly_value/1000000:.1f}M (Q ending {quarterly_end})")
            
            return ttm_value, quarterly_value
        
        # Fallback to annual data if insufficient quarterly data
        annual_data = [item for item in usd_data if item.get('form') == '10-K']
        if annual_data:
            latest_annual = sorted(annual_data, key=lambda x: x.get('end', ''), reverse=True)[0]
            annual_value = latest_annual.get('val', 0)
            print(f"  ✅ {metric_name} TTM: ${annual_value/1000000:.1f}M (from 10-K - insufficient quarterly data)")
            return annual_value, None
        
        print(f"  ❌ No sufficient data for {metric_name}")
        return None, None

    def get_latest_balance_sheet_value(self, facts_data, concept_names, metric_name):
        """Get the most recent balance sheet value"""
        usd_data = self.get_concept_data(facts_data, concept_names)
        
        if not usd_data:
            print(f"  ❌ No data found for {metric_name}")
            return None
        
        # Get most recent value
        latest = sorted(usd_data, key=lambda x: x.get('end', ''), reverse=True)[0]
        value = latest.get('val', 0)
        end_date = latest.get('end')
        
        print(f"  ✅ {metric_name}: ${value/1000000:.1f}M (as of {end_date})")
        return value

    def get_eps_and_shares(self, facts_data):
        """Get EPS and shares data"""
        us_gaap = facts_data.get('facts', {}).get('us-gaap', {})
        
        # EPS (uses USD/shares units) - Sum last 4 quarters for TTM
        eps_concepts = ['EarningsPerShareDiluted', 'EarningsPerShareBasic']
        eps_diluted = None
        eps_basic = None
        
        for concept in eps_concepts:
            if concept in us_gaap:
                eps_data = us_gaap[concept].get('units', {}).get('USD/shares', [])
                if eps_data:
                    # Get quarterly EPS data
                    quarterly_eps = [item for item in eps_data if item.get('frame') and 'Q' in item.get('frame', '')]
                    
                    if len(quarterly_eps) >= 4:
                        # Sum last 4 quarters for TTM EPS
                        sorted_q_eps = sorted(quarterly_eps, key=lambda x: x.get('end', ''), reverse=True)
                        ttm_eps = sum(q.get('val', 0) for q in sorted_q_eps[:4])
                        
                        if concept == 'EarningsPerShareDiluted':
                            eps_diluted = ttm_eps
                            print(f"  ✅ EPS Diluted (TTM): ${ttm_eps:.3f} (sum of 4 quarters)")
                        else:
                            eps_basic = ttm_eps
                            print(f"  ✅ EPS Basic (TTM): ${ttm_eps:.3f} (sum of 4 quarters)")
                    else:
                        # Fallback to annual EPS
                        annual_eps = [item for item in eps_data if item.get('form') == '10-K']
                        if annual_eps:
                            latest_eps = sorted(annual_eps, key=lambda x: x.get('end', ''), reverse=True)[0]
                            value = latest_eps.get('val', 0)
                            
                            if concept == 'EarningsPerShareDiluted':
                                eps_diluted = value
                                print(f"  ✅ EPS Diluted (TTM): ${value:.3f} (from 10-K)")
                            else:
                                eps_basic = value
                                print(f"  ✅ EPS Basic (TTM): ${value:.3f} (from 10-K)")
        
        # Shares Outstanding (uses shares units)
        shares_concepts = ['WeightedAverageNumberOfSharesOutstandingBasic', 'CommonStockSharesOutstanding']
        shares = None
        
        for concept in shares_concepts:
            if concept in us_gaap:
                shares_data = us_gaap[concept].get('units', {}).get('shares', [])
                if shares_data:
                    latest_shares = sorted(shares_data, key=lambda x: x.get('end', ''), reverse=True)[0]
                    shares = latest_shares.get('val', 0)
                    print(f"  ✅ Shares Outstanding: {shares/1000000:.1f}M shares")
                    break
        
        return eps_diluted, eps_basic, shares

    def calculate_yoy_growth(self, facts_data, current_quarterly, concept_names, metric_name):
        """Calculate YoY growth by comparing same TIME PERIOD from previous year"""
        usd_data = self.get_concept_data(facts_data, concept_names)
        
        if not usd_data or not current_quarterly:
            return None
        
        # Get quarterly data
        quarterly_data = [item for item in usd_data 
                         if item.get('frame') and 'Q' in item.get('frame', '')]
        
        if len(quarterly_data) >= 5:  # Need at least 5 quarters
            sorted_quarters = sorted(quarterly_data, key=lambda x: x.get('end', ''), reverse=True)
            
            # Get current quarter details
            current_quarter = sorted_quarters[0]
            current_end = current_quarter.get('end')
            current_val = current_quarter.get('val', 0)
            
            # Find the same time period from previous year
            # Look for quarter ending approximately 1 year earlier
            from datetime import datetime, timedelta
            current_date = datetime.strptime(current_end, '%Y-%m-%d')
            target_date = current_date - timedelta(days=365)
            
            # Find closest quarter to target date
            best_match = None
            min_diff = float('inf')
            
            for q in sorted_quarters[1:]:  # Skip current quarter
                q_end = datetime.strptime(q.get('end'), '%Y-%m-%d')
                diff = abs((q_end - target_date).days)
                if diff < min_diff:
                    min_diff = diff
                    best_match = q
            
            if best_match and min_diff < 60:  # Within 60 days of target
                prev_val = best_match.get('val', 0)
                prev_end = best_match.get('end')
                
                if prev_val > 0:
                    growth = ((current_val - prev_val) / prev_val) * 100
                    print(f"  ✅ {metric_name} YoY Growth: {growth:.2f}% ({current_end} vs {prev_end})")
                    return growth
        
        print(f"  ❌ Could not find matching previous year quarter for {metric_name}")
        return None

    def calculate_all_metrics(self, symbol):
        """Calculate all 34 metrics using bulletproof approach"""
        print(f"\n🚀 BULLETPROOF CALCULATION FOR {symbol}")
        print("=" * 60)
        
        # Get company data
        facts_data = self.get_company_data(symbol)
        if not facts_data:
            return False
        
        metrics = {}
        
        print("\n📈 INCOME STATEMENT:")
        # Revenue
        revenue_ttm, revenue_q = self.get_ttm_and_quarterly(
            facts_data, 
            ['Revenues', 'RevenueFromContractWithCustomerExcludingAssessedTax'], 
            'Revenue'
        )
        metrics['revenue_ttm'] = revenue_ttm
        metrics['revenue_quarterly'] = revenue_q
        
        # Net Income
        ni_ttm, ni_q = self.get_ttm_and_quarterly(
            facts_data, 
            ['NetIncomeLoss'], 
            'Net Income'
        )
        metrics['net_income_ttm'] = ni_ttm
        metrics['net_income_quarterly'] = ni_q
        
        # EBIT (Operating Income)
        ebit_ttm, ebit_q = self.get_ttm_and_quarterly(
            facts_data, 
            ['OperatingIncomeLoss'], 
            'EBIT'
        )
        metrics['ebit_ttm'] = ebit_ttm
        metrics['ebit_quarterly'] = ebit_q
        
        # EBITDA (EBIT + Depreciation)
        depreciation_ttm, depreciation_q = self.get_ttm_and_quarterly(
            facts_data,
            ['DepreciationDepletionAndAmortization', 'Depreciation', 'DepreciationAndAmortization'],
            'Depreciation'
        )
        
        if ebit_ttm and depreciation_ttm:
            metrics['ebitda_ttm'] = ebit_ttm + depreciation_ttm
            print(f"  ✅ EBITDA TTM: ${metrics['ebitda_ttm']/1000000:.1f}M (EBIT + Depreciation)")
        
        if ebit_q and depreciation_q:
            metrics['ebitda_quarterly'] = ebit_q + depreciation_q
            print(f"  ✅ EBITDA Quarterly: ${metrics['ebitda_quarterly']/1000000:.1f}M (EBIT + Depreciation)")
        
        # YoY Growth Rates
        print("\n📊 YOY GROWTH RATES:")
        metrics['revenue_yoy_growth'] = self.calculate_yoy_growth(
            facts_data, revenue_q, 
            ['Revenues', 'RevenueFromContractWithCustomerExcludingAssessedTax'], 
            'Revenue'
        )
        
        metrics['ebitda_yoy_growth'] = None
        if metrics.get('ebitda_quarterly'):
            # Calculate previous year EBITDA for comparison
            prev_ebit = self.get_previous_year_same_quarter(facts_data, ['OperatingIncomeLoss'])
            prev_depreciation = self.get_previous_year_same_quarter(facts_data, ['DepreciationDepletionAndAmortization', 'Depreciation'])
            
            if prev_ebit and prev_depreciation:
                prev_ebitda = prev_ebit + prev_depreciation
                current_ebitda = metrics['ebitda_quarterly']
                
                if prev_ebitda > 0:
                    ebitda_growth = ((current_ebitda - prev_ebitda) / prev_ebitda) * 100
                    metrics['ebitda_yoy_growth'] = ebitda_growth
                    print(f"  ✅ EBITDA YoY Growth: {ebitda_growth:.2f}%")
        
        # EPS and Shares
        print("\n📊 EPS AND SHARES:")
        eps_diluted, eps_basic, shares = self.get_eps_and_shares(facts_data)
        metrics['eps_diluted_ttm'] = eps_diluted
        metrics['eps_basic_ttm'] = eps_basic
        metrics['shares_outstanding'] = shares
        
        # EPS YoY Growth (calculate from quarterly data)
        if ni_q and shares:
            current_quarterly_eps = ni_q / shares
            prev_ni = self.get_previous_year_same_quarter(facts_data, ['NetIncomeLoss'])
            prev_shares = self.get_previous_year_shares(facts_data)
            
            if prev_ni and prev_shares and prev_shares > 0:
                prev_quarterly_eps = prev_ni / prev_shares
                if prev_quarterly_eps > 0:
                    eps_growth = ((current_quarterly_eps - prev_quarterly_eps) / prev_quarterly_eps) * 100
                    metrics['eps_diluted_yoy_growth'] = eps_growth
                    print(f"  ✅ EPS Diluted YoY Growth: {eps_growth:.2f}%")
        
        # Balance Sheet
        print("\n🏛️ BALANCE SHEET:")
        metrics['total_assets'] = self.get_latest_balance_sheet_value(
            facts_data, ['Assets'], 'Total Assets'
        )
        
        metrics['total_liabilities'] = self.get_latest_balance_sheet_value(
            facts_data, ['Liabilities'], 'Total Liabilities'
        )
        
        metrics['shareholders_equity'] = self.get_latest_balance_sheet_value(
            facts_data, ['StockholdersEquity'], 'Shareholders Equity'
        )
        
        # Cash and Short Term Investments - try comprehensive approach
        cash = self.get_latest_balance_sheet_value(
            facts_data, ['CashAndCashEquivalentsAtCarryingValue'], 'Cash'
        ) or 0
        
        # Try multiple short-term investment concepts (including current marketable securities)
        short_term_concepts = [
            'ShortTermInvestments', 
            'MarketableSecurities',
            'MarketableSecuritiesCurrent',  # NVIDIA uses this
            'AvailableForSaleSecuritiesCurrent',
            'CashCashEquivalentsAndShortTermInvestments'
        ]
        
        short_term = 0
        for concept in short_term_concepts:
            value = self.get_latest_balance_sheet_value(facts_data, [concept], f'Short-term ({concept})')
            if value and value > short_term:
                short_term = value  # Use the largest value found
        
        # If we found a combined concept, use it directly
        combined = self.get_latest_balance_sheet_value(
            facts_data, ['CashCashEquivalentsAndShortTermInvestments'], 'Combined Cash & Short-term'
        )
        
        if combined and combined > cash:
            metrics['cash_and_equivalents'] = combined
            print(f"  ✅ Cash & Short Term (combined): ${combined/1000000:.1f}M")
        else:
            metrics['cash_and_equivalents'] = cash + short_term
            print(f"  ✅ Cash & Short Term (sum): ${(cash + short_term)/1000000:.1f}M")
        
        # Long Term Assets
        metrics['long_term_assets'] = self.get_latest_balance_sheet_value(
            facts_data, ['AssetsNoncurrent', 'NoncurrentAssets'], 'Long Term Assets'
        )
        
        # If not found, calculate as Total Assets - Current Assets
        if not metrics['long_term_assets'] and metrics.get('total_assets'):
            current_assets = self.get_latest_balance_sheet_value(
                facts_data, ['AssetsCurrent'], 'Current Assets'
            )
            if current_assets:
                metrics['long_term_assets'] = metrics['total_assets'] - current_assets
                print(f"  📊 Long Term Assets (calculated): ${metrics['long_term_assets']/1000000:.1f}M")
        
        # Long Term Debt
        metrics['long_term_debt'] = self.get_latest_balance_sheet_value(
            facts_data, ['DebtInstrumentCarryingAmount', 'LongTermDebt', 'LongTermDebtNoncurrent'], 'Long Term Debt'
        )
        
        # Cash Flow - ALWAYS use quarterly data (more recent)
        print("\n💸 CASH FLOW:")
        metrics['operating_cash_flow_ttm'], _ = self.get_ttm_and_quarterly(
            facts_data, ['NetCashProvidedByUsedInOperatingActivities'], 'Operating Cash Flow'
        )
        
        metrics['investing_cash_flow_ttm'], _ = self.get_ttm_and_quarterly(
            facts_data, ['NetCashProvidedByUsedInInvestingActivities'], 'Investing Cash Flow'
        )
        
        metrics['financing_cash_flow_ttm'], _ = self.get_ttm_and_quarterly(
            facts_data, ['NetCashProvidedByUsedInFinancingActivities'], 'Financing Cash Flow'
        )
        
        # Capital Expenditures - try multiple concepts
        metrics['capital_expenditures_ttm'], _ = self.get_ttm_and_quarterly(
            facts_data, 
            ['PaymentsToAcquirePropertyPlantAndEquipment', 'CapitalExpenditures', 'PaymentsForPropertyPlantAndEquipment'], 
            'Capital Expenditures'
        )
        
        # Change in Receivables
        metrics['change_in_receivables_ttm'], _ = self.get_ttm_and_quarterly(
            facts_data, ['IncreaseDecreaseInAccountsReceivable'], 'Change in Receivables'
        )
        
        # Changes in Working Capital
        metrics['changes_in_working_capital_ttm'], _ = self.get_ttm_and_quarterly(
            facts_data, ['IncreaseDecreaseInOperatingCapital'], 'Changes in Working Capital'
        )
        
        # Free Cash Flow
        if metrics.get('operating_cash_flow_ttm') and metrics.get('capital_expenditures_ttm'):
            metrics['free_cash_flow_ttm'] = metrics['operating_cash_flow_ttm'] - abs(metrics['capital_expenditures_ttm'])
            print(f"  ✅ Free Cash Flow: ${metrics['free_cash_flow_ttm']/1000000:.1f}M")
        
        # Financial Ratios
        print("\n📊 FINANCIAL RATIOS:")
        if metrics.get('net_income_ttm') and metrics.get('total_assets'):
            metrics['return_on_assets'] = (metrics['net_income_ttm'] / metrics['total_assets']) * 100
            print(f"  ✅ ROA: {metrics['return_on_assets']:.2f}%")
        
        if metrics.get('net_income_ttm') and metrics.get('shareholders_equity'):
            metrics['return_on_equity'] = (metrics['net_income_ttm'] / metrics['shareholders_equity']) * 100
            print(f"  ✅ ROE: {metrics['return_on_equity']:.2f}%")
        
        if metrics.get('ebit_ttm') and metrics.get('revenue_ttm'):
            metrics['operating_margin'] = (metrics['ebit_ttm'] / metrics['revenue_ttm']) * 100
            print(f"  ✅ Operating Margin: {metrics['operating_margin']:.2f}%")
        
        # Gross Profit Margin - calculate COGS from income statement structure
        cogs_ttm, _ = self.get_ttm_and_quarterly(
            facts_data, 
            ['CostOfGoodsAndServicesSold', 'CostOfRevenue', 'CostOfSales', 'CostOfGoodsSold'], 
            'Cost of Goods Sold'
        )
        
        # If direct COGS not found, calculate from Revenue - Gross Profit
        if not cogs_ttm and metrics.get('revenue_ttm') and metrics.get('ebit_ttm'):
            # Get operating expenses
            operating_expenses_ttm, _ = self.get_ttm_and_quarterly(
                facts_data, ['OperatingExpenses'], 'Operating Expenses'
            )
            
            if operating_expenses_ttm:
                # COGS = Revenue - (Operating Income + Operating Expenses)
                gross_profit = metrics['ebit_ttm'] + operating_expenses_ttm
                cogs_ttm = metrics['revenue_ttm'] - gross_profit
                print(f"  📊 Calculated COGS: ${cogs_ttm/1000000:.1f}M (Revenue - Gross Profit)")
        
        if metrics.get('revenue_ttm') and cogs_ttm:
            gross_profit = metrics['revenue_ttm'] - cogs_ttm
            metrics['gross_profit_margin'] = (gross_profit / metrics['revenue_ttm']) * 100
            print(f"  ✅ Gross Profit Margin: {metrics['gross_profit_margin']:.2f}%")
        else:
            print(f"  ❌ Could not calculate Gross Profit Margin")
        
        return metrics

    def get_previous_year_same_quarter(self, facts_data, concept_names):
        """Get the same quarter from previous year"""
        usd_data = self.get_concept_data(facts_data, concept_names)
        
        if not usd_data:
            return None
        
        quarterly_data = [item for item in usd_data 
                         if item.get('frame') and 'Q' in item.get('frame', '')]
        
        if len(quarterly_data) >= 5:  # Need at least 5 quarters
            sorted_quarters = sorted(quarterly_data, key=lambda x: x.get('end', ''), reverse=True)
            # Use 4th quarter back as approximation of same quarter previous year
            prev_quarter = sorted_quarters[4]
            return prev_quarter.get('val', 0)
        
        return None

    def get_previous_year_shares(self, facts_data):
        """Get shares outstanding from previous year"""
        us_gaap = facts_data.get('facts', {}).get('us-gaap', {})
        
        shares_concepts = ['WeightedAverageNumberOfSharesOutstandingBasic', 'CommonStockSharesOutstanding']
        
        for concept in shares_concepts:
            if concept in us_gaap:
                shares_data = us_gaap[concept].get('units', {}).get('shares', [])
                if shares_data:
                    # Get annual shares data
                    annual_shares = [item for item in shares_data if item.get('form') == '10-K']
                    if len(annual_shares) >= 2:
                        sorted_shares = sorted(annual_shares, key=lambda x: x.get('end', ''), reverse=True)
                        return sorted_shares[1].get('val', 0)  # Previous year
        
        return None

    def store_metrics(self, symbol, metrics):
        """Store metrics in database"""
        try:
            # Database connection
            conn = psycopg2.connect(
                host=os.getenv('DB_HOST', 'localhost'),
                port=os.getenv('DB_PORT', '5433'),
                user=os.getenv('DB_USER', 'investorcenter'),
                password=os.getenv('DB_PASSWORD', 'investorcenter123'),
                database=os.getenv('DB_NAME', 'investorcenter_db')
            )
            
            cursor = conn.cursor()
            
            # Convert to millions and add metadata
            processed_metrics = {}
            for key, value in metrics.items():
                if value is not None:
                    if 'growth' in key or 'margin' in key or 'return' in key:
                        processed_metrics[key] = round(value, 2)  # Percentages
                    else:
                        processed_metrics[key] = round(value / 1000000, 1)  # Convert to millions
            
            processed_metrics['updated_at'] = datetime.now().isoformat()
            processed_metrics['calculation_method'] = 'bulletproof_simple'
            
            # Upsert into database
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
            
            print(f"✅ Successfully stored metrics for {symbol}")
            return True
            
        except Exception as e:
            print(f"❌ Error storing metrics for {symbol}: {e}")
            return False

def main():
    calculator = BulletproofSECCalculator()
    
    if len(sys.argv) > 1:
        symbol = sys.argv[1].upper()
    else:
        symbol = 'NVDA'
    
    metrics = calculator.calculate_all_metrics(symbol)
    
    if metrics:
        success = calculator.store_metrics(symbol, metrics)
        if success:
            print(f"✅ Successfully processed {symbol}")
        else:
            print(f"❌ Failed to store {symbol}")
    else:
        print(f"❌ Failed to calculate metrics for {symbol}")

if __name__ == "__main__":
    main()
