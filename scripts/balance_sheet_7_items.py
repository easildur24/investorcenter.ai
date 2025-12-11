#!/usr/bin/env python3
"""
7 Balance Sheet Items - Using YOUR ALGORITHM APPROACH
Balance sheet items use the most recent quarterly data (latest filing)
"""

import requests
import json
import sys

class BalanceSheet7Items:
    def __init__(self):
        self.base_url = "https://data.sec.gov/api/xbrl/companyfacts/"
        self.headers = {
            'User-Agent': 'InvestorCenter.ai financial-data-collector contact@investorcenter.ai'
        }
        
        self.cik_mapping = {
            'AAPL': '0000320193',
            'MSFT': '0000789019', 
            'GOOGL': '0001652044',
            'AMZN': '0001018724',
            'TSLA': '0001318605',
            'META': '0001326801',
            'NVDA': '0001045810',
            'HIMS': '0001773751',
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

    def get_latest_balance_sheet_value(self, facts_data, concepts, metric_name):
        """Get most recent balance sheet value from latest filing"""
        us_gaap = facts_data.get('facts', {}).get('us-gaap', {})
        
        # Get ALL data from all concepts
        all_data = []
        for concept in concepts:
            if concept in us_gaap:
                usd_data = us_gaap[concept].get('units', {}).get('USD', [])
                for item in usd_data:
                    item_copy = item.copy()
                    item_copy['source_concept'] = concept
                    all_data.append(item_copy)
        
        if not all_data:
            print(f"  ❌ No data found for {metric_name}")
            return None
        
        # Get most recent value (from any filing type)
        latest_item = sorted(all_data, key=lambda x: x.get('end', ''), reverse=True)[0]
        value = latest_item.get('val', 0)
        end_date = latest_item.get('end')
        form = latest_item.get('form', '')
        source = latest_item.get('source_concept', 'Unknown')
        
        print(f"  ✅ {metric_name}: ${value/1000000:.1f}M (as of {end_date} - {form})")
        return value

    def calculate_balance_sheet_7(self, symbol):
        """Calculate 7 balance sheet items"""
        print(f"\n🎯 {symbol} - 7 BALANCE SHEET ITEMS")
        print("=" * 50)
        
        facts_data = self.get_company_data(symbol)
        if not facts_data:
            return None
        
        results = {}
        
        # 1. Total Assets
        print("📊 1. TOTAL ASSETS:")
        results['total_assets'] = self.get_latest_balance_sheet_value(
            facts_data, ['Assets'], 'Total Assets'
        )
        
        # 2. Total Liabilities  
        print("\n📊 2. TOTAL LIABILITIES:")
        results['total_liabilities'] = self.get_latest_balance_sheet_value(
            facts_data, ['Liabilities'], 'Total Liabilities'
        )
        
        # 3. Shareholders Equity
        print("\n📊 3. SHAREHOLDERS EQUITY:")
        results['shareholders_equity'] = self.get_latest_balance_sheet_value(
            facts_data, ['StockholdersEquity'], 'Shareholders Equity'
        )
        
        # 4. Cash and Short Term Investments
        print("\n📊 4. CASH AND SHORT TERM INVESTMENTS:")
        # Get cash
        cash = self.get_latest_balance_sheet_value(
            facts_data, ['CashAndCashEquivalentsAtCarryingValue'], 'Cash'
        ) or 0
        
        # Get short-term investments
        short_term = self.get_latest_balance_sheet_value(
            facts_data, ['ShortTermInvestments', 'MarketableSecurities', 'MarketableSecuritiesCurrent'], 'Short-term Investments'
        ) or 0
        
        results['cash_and_short_term'] = cash + short_term
        print(f"  ✅ Cash & Short-term Total: ${(cash + short_term)/1000000:.1f}M")
        
        # 5. Total Long Term Assets
        print("\n📊 5. TOTAL LONG TERM ASSETS:")
        # Try direct concept first
        long_term_assets = self.get_latest_balance_sheet_value(
            facts_data, ['AssetsNoncurrent', 'NoncurrentAssets'], 'Long Term Assets (direct)'
        )
        
        # If not found, calculate as Total Assets - Current Assets
        if not long_term_assets and results.get('total_assets'):
            current_assets = self.get_latest_balance_sheet_value(
                facts_data, ['AssetsCurrent'], 'Current Assets'
            )
            if current_assets:
                long_term_assets = results['total_assets'] - current_assets
                print(f"  ✅ Long Term Assets (calculated): ${long_term_assets/1000000:.1f}M (Total - Current)")
        
        results['long_term_assets'] = long_term_assets
        
        # 6. Total Long Term Debt
        print("\n📊 6. TOTAL LONG TERM DEBT:")
        results['long_term_debt'] = self.get_latest_balance_sheet_value(
            facts_data, 
            ['DebtInstrumentCarryingAmount', 'LongTermDebt', 'LongTermDebtNoncurrent'], 
            'Long Term Debt'
        )
        
        # 7. Book Value (same as Shareholders Equity)
        print("\n📊 7. BOOK VALUE:")
        results['book_value'] = results.get('shareholders_equity')
        if results['book_value']:
            print(f"  ✅ Book Value: ${results['book_value']/1000000:.1f}M (same as Shareholders Equity)")
        
        return results

def main():
    calculator = BalanceSheet7Items()
    
    symbols = ['TSLA', 'HIMS', 'GOOGL'] if len(sys.argv) == 1 else [sys.argv[1].upper()]
    
    for symbol in symbols:
        results = calculator.calculate_balance_sheet_7(symbol)
        
        if results:
            print(f"\n🎯 {symbol} BALANCE SHEET SUMMARY:")
            print("-" * 40)
            balance_sheet_items = [
                'total_assets', 'total_liabilities', 'shareholders_equity',
                'cash_and_short_term', 'long_term_assets', 'long_term_debt', 'book_value'
            ]
            
            for item in balance_sheet_items:
                if results.get(item):
                    value = results[item] / 1000000000 if results[item] > 1000000000 else results[item] / 1000000
                    unit = 'B' if results[item] > 1000000000 else 'M'
                    print(f"{item.replace('_', ' ').title():<25}: ${value:.2f}{unit}")
                else:
                    print(f"{item.replace('_', ' ').title():<25}: NOT FOUND")
        
        print()

if __name__ == "__main__":
    main()
