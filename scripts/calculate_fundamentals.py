#!/usr/bin/env python3
"""
Offline SEC Filing Parser for Fundamental Metrics
Calculates and stores fundamental metrics in the database
"""

import boto3
import re
import json
import psycopg2
from datetime import datetime
from decimal import Decimal
import sys
import os

class SECFundamentalsCalculator:
    def __init__(self):
        self.s3_client = boto3.client('s3')
        self.bucket_name = 'investorcenter-sec-filings'
        
        # Database connection
        self.db_conn = psycopg2.connect(
            host=os.getenv('DB_HOST', 'localhost'),
            port=os.getenv('DB_PORT', '5432'),
            user=os.getenv('DB_USER', 'investorcenter'),
            password=os.getenv('DB_PASSWORD', ''),
            database=os.getenv('DB_NAME', 'investorcenter_db')
        )
        
    def get_latest_filings(self, symbol):
        """Get the latest 10-K and 4 most recent 10-Qs"""
        filings = []
        
        # Get latest 10-K
        try:
            ten_k = self.get_latest_10k(symbol)
            if ten_k:
                filings.append(ten_k)
        except Exception as e:
            print(f"Warning: Could not get 10-K for {symbol}: {e}")
        
        # Get latest 4 10-Qs
        try:
            ten_qs = self.get_latest_10qs(symbol, 4)
            filings.extend(ten_qs)
        except Exception as e:
            print(f"Warning: Could not get 10-Qs for {symbol}: {e}")
            
        return filings
    
    def get_latest_10k(self, symbol):
        """Get the most recent 10-K filing"""
        prefix = f'filings/{symbol}/10-K/'
        
        # List years
        response = self.s3_client.list_objects_v2(
            Bucket=self.bucket_name,
            Prefix=prefix,
            Delimiter='/'
        )
        
        if 'CommonPrefixes' not in response:
            return None
            
        years = sorted([p['Prefix'].split('/')[-2] for p in response['CommonPrefixes']])
        if not years:
            return None
            
        # Get files from the latest year
        latest_year = years[-1]
        files_response = self.s3_client.list_objects_v2(
            Bucket=self.bucket_name,
            Prefix=f'{prefix}{latest_year}/'
        )
        
        if 'Contents' not in files_response:
            return None
            
        # Get the most recent file
        files = sorted(files_response['Contents'], key=lambda x: x['LastModified'])
        if not files:
            return None
            
        latest_file = files[-1]
        
        # Download the file
        content = self.download_file(latest_file['Key'])
        
        return {
            'type': '10-K',
            'date': latest_file['LastModified'],
            'content': content,
            'key': latest_file['Key']
        }
    
    def get_latest_10qs(self, symbol, count=4):
        """Get the most recent N 10-Q filings"""
        prefix = f'filings/{symbol}/10-Q/'
        all_files = []
        
        # List years
        response = self.s3_client.list_objects_v2(
            Bucket=self.bucket_name,
            Prefix=prefix,
            Delimiter='/'
        )
        
        if 'CommonPrefixes' not in response:
            return []
            
        years = sorted([p['Prefix'].split('/')[-2] for p in response['CommonPrefixes']])
        
        # Collect files from all years
        for year in years:
            files_response = self.s3_client.list_objects_v2(
                Bucket=self.bucket_name,
                Prefix=f'{prefix}{year}/'
            )
            
            if 'Contents' in files_response:
                for file_obj in files_response['Contents']:
                    if file_obj['Key'].endswith('.html'):
                        all_files.append(file_obj)
        
        # Sort by date and take the most recent N
        all_files = sorted(all_files, key=lambda x: x['LastModified'], reverse=True)
        recent_files = all_files[:count]
        
        filings = []
        for file_obj in recent_files:
            content = self.download_file(file_obj['Key'])
            filings.append({
                'type': '10-Q',
                'date': file_obj['LastModified'],
                'content': content,
                'key': file_obj['Key']
            })
            
        return filings
    
    def download_file(self, key):
        """Download file from S3"""
        response = self.s3_client.get_object(Bucket=self.bucket_name, Key=key)
        return response['Body'].read().decode('utf-8')
    
    def extract_financial_data(self, content):
        """Extract comprehensive financial data from SEC filing HTML"""
        data = {}
        
        # Apple's SEC filings structure - extract from financial statement tables
        # All numbers are in millions in the filings
        
        # 1. REVENUE (Products + Services)
        if 'Products' in content and 'Services' in content:
            # Extract Products nine-month revenue (3rd column in the table)
            products_match = re.search(r'Products.*?\$.*?\d{1,3},\d{3}&#160;.*?\$.*?\d{1,3},\d{3}&#160;.*?\$.*?(\d{1,3},\d{3})&#160;', content)
            services_match = re.search(r'Services.*?(\d{1,3},\d{3})&#160;.*?(\d{1,3},\d{3})&#160;.*?(\d{1,3},\d{3})&#160;.*?(\d{1,3},\d{3})&#160;', content)
            
            if products_match and services_match:
                products_9m = int(products_match.group(1).replace(',', ''))
                services_9m = int(services_match.group(3).replace(',', ''))
                total_9m = products_9m + services_9m
                ttm_estimate = int(total_9m * 4 / 3)  # Rough TTM estimation
                data['revenue'] = Decimal(str(ttm_estimate))
                data['revenue_9m'] = Decimal(str(total_9m))
                print(f"✅ Revenue: Products ${products_9m}M + Services ${services_9m}M = TTM ${ttm_estimate}M")
        
        # 2. NET INCOME - Look for net income in the income statement
        net_income_patterns = [
            r'Net income.*?\$.*?(\d{1,3},\d{3})&#160;.*?\$.*?(\d{1,3},\d{3})&#160;.*?\$.*?(\d{1,3},\d{3})&#160;.*?\$.*?(\d{1,3},\d{3})&#160;',
            r'Net earnings.*?\$.*?(\d{1,3},\d{3})&#160;.*?\$.*?(\d{1,3},\d{3})&#160;.*?\$.*?(\d{1,3},\d{3})&#160;.*?\$.*?(\d{1,3},\d{3})&#160;',
        ]
        
        for pattern in net_income_patterns:
            match = re.search(pattern, content, re.IGNORECASE | re.DOTALL)
            if match:
                ni_9m = int(match.group(3).replace(',', ''))  # 9-month column
                ni_ttm = int(ni_9m * 4 / 3)
                data['net_income'] = Decimal(str(ni_ttm))
                print(f"✅ Net Income TTM: ${ni_ttm}M")
                break
        
        # 3. OPERATING INCOME - Look for operating income
        operating_income_patterns = [
            r'Operating income.*?\$.*?(\d{1,3},\d{3})&#160;.*?\$.*?(\d{1,3},\d{3})&#160;.*?\$.*?(\d{1,3},\d{3})&#160;.*?\$.*?(\d{1,3},\d{3})&#160;',
        ]
        
        for pattern in operating_income_patterns:
            match = re.search(pattern, content, re.IGNORECASE | re.DOTALL)
            if match:
                oi_9m = int(match.group(3).replace(',', ''))
                oi_ttm = int(oi_9m * 4 / 3)
                data['operating_income'] = Decimal(str(oi_ttm))
                print(f"✅ Operating Income TTM: ${oi_ttm}M")
                break
        
        # 4. R&D EXPENSES
        rd_match = re.search(r'Research and development.*?\$.*?(\d{1,3},\d{3})&#160;.*?\$.*?(\d{1,3},\d{3})&#160;.*?\$.*?(\d{1,3},\d{3})&#160;.*?\$.*?(\d{1,3},\d{3})&#160;', content)
        if rd_match:
            rd_9m = int(rd_match.group(3).replace(',', ''))
            rd_ttm = int(rd_9m * 4 / 3)
            data['research_development'] = Decimal(str(rd_ttm))
            print(f"✅ R&D TTM: ${rd_ttm}M")
        
        # 5. BALANCE SHEET DATA - Look for balance sheet items
        # Total assets
        assets_patterns = [
            r'Total assets.*?\$.*?(\d{1,3},\d{3})&#160;',
            r'TOTAL ASSETS.*?\$.*?(\d{1,3},\d{3})&#160;',
        ]
        
        for pattern in assets_patterns:
            match = re.search(pattern, content, re.IGNORECASE)
            if match:
                assets = int(match.group(1).replace(',', ''))
                data['total_assets'] = Decimal(str(assets))
                print(f"✅ Total Assets: ${assets}M")
                break
        
        # Shareholders' equity
        equity_patterns = [
            r'Total shareholders.? equity.*?\$.*?(\d{1,3},\d{3})&#160;',
            r'Total stockholders.? equity.*?\$.*?(\d{1,3},\d{3})&#160;',
            r'Shareholders.? equity.*?\$.*?(\d{1,3},\d{3})&#160;',
        ]
        
        for pattern in equity_patterns:
            match = re.search(pattern, content, re.IGNORECASE)
            if match:
                equity = int(match.group(1).replace(',', ''))
                data['shareholders_equity'] = Decimal(str(equity))
                print(f"✅ Shareholders Equity: ${equity}M")
                break
        
        # Cash and cash equivalents
        cash_patterns = [
            r'Cash and cash equivalents.*?\$.*?(\d{1,3},\d{3})&#160;',
            r'Cash and marketable securities.*?\$.*?(\d{1,3},\d{3})&#160;',
        ]
        
        for pattern in cash_patterns:
            match = re.search(pattern, content, re.IGNORECASE)
            if match:
                cash = int(match.group(1).replace(',', ''))
                data['cash_and_equivalents'] = Decimal(str(cash))
                print(f"✅ Cash & Equivalents: ${cash}M")
                break
        
        # 6. CASH FLOW DATA
        # Operating cash flow
        ocf_patterns = [
            r'Cash generated by operating activities.*?\$.*?(\d{1,3},\d{3})&#160;.*?\$.*?(\d{1,3},\d{3})&#160;.*?\$.*?(\d{1,3},\d{3})&#160;.*?\$.*?(\d{1,3},\d{3})&#160;',
            r'Net cash provided by operating activities.*?\$.*?(\d{1,3},\d{3})&#160;.*?\$.*?(\d{1,3},\d{3})&#160;.*?\$.*?(\d{1,3},\d{3})&#160;.*?\$.*?(\d{1,3},\d{3})&#160;',
        ]
        
        for pattern in ocf_patterns:
            match = re.search(pattern, content, re.IGNORECASE | re.DOTALL)
            if match:
                ocf_9m = int(match.group(3).replace(',', ''))
                ocf_ttm = int(ocf_9m * 4 / 3)
                data['operating_cash_flow'] = Decimal(str(ocf_ttm))
                print(f"✅ Operating Cash Flow TTM: ${ocf_ttm}M")
                break
        
        # Capital expenditures (usually negative)
        capex_patterns = [
            r'Payments for acquisition of property, plant and equipment.*?\$.*?\((\d{1,3},\d{3})\)&#160;.*?\$.*?\((\d{1,3},\d{3})\)&#160;.*?\$.*?\((\d{1,3},\d{3})\)&#160;.*?\$.*?\((\d{1,3},\d{3})\)&#160;',
            r'Capital expenditures.*?\$.*?\((\d{1,3},\d{3})\)&#160;.*?\$.*?\((\d{1,3},\d{3})\)&#160;.*?\$.*?\((\d{1,3},\d{3})\)&#160;.*?\$.*?\((\d{1,3},\d{3})\)&#160;',
        ]
        
        for pattern in capex_patterns:
            match = re.search(pattern, content, re.IGNORECASE | re.DOTALL)
            if match:
                capex_9m = int(match.group(3).replace(',', ''))
                capex_ttm = int(capex_9m * 4 / 3)
                data['capital_expenditures'] = Decimal(str(-capex_ttm))  # Negative because it's an expense
                print(f"✅ Capital Expenditures TTM: -${capex_ttm}M")
                break
        
        # Free cash flow calculation
        if 'operating_cash_flow' in data and 'capital_expenditures' in data:
            fcf = data['operating_cash_flow'] + data['capital_expenditures']  # capex is already negative
            data['free_cash_flow'] = fcf
            print(f"✅ Free Cash Flow TTM: ${fcf}M")
        
        return data
    
    def calculate_metrics(self, symbol):
        """Calculate all fundamental metrics for a symbol"""
        print(f"🔍 Calculating fundamental metrics for {symbol}")
        
        # Get filings
        filings = self.get_latest_filings(symbol)
        if not filings:
            print(f"❌ No filings found for {symbol}")
            return None
            
        print(f"📄 Found {len(filings)} filings for {symbol}")
        
        # Extract data from all filings
        quarterly_data = []
        annual_data = {}
        
        for filing in filings:
            print(f"📊 Processing {filing['type']} from {filing['date']}")
            
            extracted = self.extract_financial_data(filing['content'])
            extracted['filing_date'] = filing['date']
            extracted['filing_type'] = filing['type']
            
            if filing['type'] == '10-Q':
                quarterly_data.append(extracted)
            else:
                annual_data = extracted
        
        # Sort quarterly data by date (most recent first)
        quarterly_data.sort(key=lambda x: x['filing_date'], reverse=True)
        
        # Calculate metrics
        metrics = {
            'symbol': symbol,
            'updated_at': datetime.now().isoformat(),
            'revenue_ttm': None,
            'net_income_ttm': None,
            'ebit_ttm': None,
            'ebitda_ttm': None,
            'revenue_quarterly': None,
            'net_income_quarterly': None,
            'ebit_quarterly': None,
            'ebitda_quarterly': None,
            'revenue_qoq_growth': None,
            'eps_qoq_growth': None,
            'ebitda_qoq_growth': None,
            'total_assets': None,
            'total_liabilities': None,
            'shareholders_equity': None,
            'cash_short_term_investments': None,
            'total_long_term_assets': None,
            'total_long_term_debt': None,
            'book_value': None,
            'cash_from_operations': None,
            'cash_from_investing': None,
            'cash_from_financing': None,
            'change_in_receivables': None,
            'changes_in_working_capital': None,
            'capital_expenditures': None,
            'ending_cash': None,
            'free_cash_flow': None,
            'return_on_assets': None,
            'return_on_equity': None,
            'return_on_invested_capital': None,
            'operating_margin': None,
            'gross_profit_margin': None,
            'eps_diluted': None,
            'eps_basic': None,
            'shares_outstanding': None,
            'total_employees': None,
            'revenue_per_employee': None,
            'net_income_per_employee': None,
            # Market data fields
            'market_cap': 'needs data',
            'pe_ratio': 'needs data',
            'price_to_book': 'needs data',
            'dividend_yield': 'needs data',
            'one_month_returns': 'needs data',
            'three_month_returns': 'needs data',
            'six_month_returns': 'needs data',
            'year_to_date_returns': 'needs data',
            'one_year_returns': 'needs data',
            'three_year_returns': 'needs data',
            'five_year_returns': 'needs data',
            'fifty_two_week_high': 'needs data',
            'fifty_two_week_low': 'needs data',
            'alpha_5y': 'needs data',
            'beta_5y': 'needs data',
        }
        
        # Calculate comprehensive metrics from quarterly data
        if quarterly_data:
            latest_quarter = quarterly_data[0]
            
            # Revenue
            if latest_quarter.get('revenue'):
                revenue_ttm = float(latest_quarter['revenue'])
                metrics['revenue_ttm'] = revenue_ttm
                print(f"💰 Revenue TTM: ${revenue_ttm}M")
                
            if latest_quarter.get('revenue_9m'):
                quarterly_revenue = float(latest_quarter['revenue_9m']) / 3
                metrics['revenue_quarterly'] = quarterly_revenue
                print(f"📊 Revenue Quarterly: ${quarterly_revenue}M")
            
            # Net Income
            if latest_quarter.get('net_income'):
                net_income_ttm = float(latest_quarter['net_income'])
                metrics['net_income_ttm'] = net_income_ttm
                print(f"💰 Net Income TTM: ${net_income_ttm}M")
            
            # Operating Income (EBIT)
            if latest_quarter.get('operating_income'):
                ebit_ttm = float(latest_quarter['operating_income'])
                metrics['ebit_ttm'] = ebit_ttm
                print(f"💰 EBIT TTM: ${ebit_ttm}M")
            
            # Balance Sheet
            if latest_quarter.get('total_assets'):
                metrics['total_assets'] = float(latest_quarter['total_assets'])
                print(f"🏛️ Total Assets: ${metrics['total_assets']}M")
            
            if latest_quarter.get('shareholders_equity'):
                equity = float(latest_quarter['shareholders_equity'])
                metrics['shareholders_equity'] = equity
                metrics['book_value'] = equity  # Book value = shareholders' equity
                print(f"🏛️ Shareholders Equity: ${equity}M")
            
            if latest_quarter.get('cash_and_equivalents'):
                cash = float(latest_quarter['cash_and_equivalents'])
                metrics['cash_short_term_investments'] = cash
                metrics['ending_cash'] = cash
                print(f"💵 Cash & Equivalents: ${cash}M")
            
            # Cash Flow
            if latest_quarter.get('operating_cash_flow'):
                ocf = float(latest_quarter['operating_cash_flow'])
                metrics['cash_from_operations'] = ocf
                print(f"💸 Operating Cash Flow TTM: ${ocf}M")
            
            if latest_quarter.get('capital_expenditures'):
                capex = float(latest_quarter['capital_expenditures'])
                metrics['capital_expenditures'] = capex
                print(f"🏗️ Capital Expenditures TTM: ${capex}M")
            
            if latest_quarter.get('free_cash_flow'):
                fcf = float(latest_quarter['free_cash_flow'])
                metrics['free_cash_flow'] = fcf
                print(f"💎 Free Cash Flow TTM: ${fcf}M")
            
            # Calculate ratios
            if metrics.get('net_income_ttm') and metrics.get('total_assets'):
                roa = (metrics['net_income_ttm'] / metrics['total_assets']) * 100
                metrics['return_on_assets'] = roa
                print(f"📊 ROA: {roa:.1f}%")
            
            if metrics.get('net_income_ttm') and metrics.get('shareholders_equity'):
                roe = (metrics['net_income_ttm'] / metrics['shareholders_equity']) * 100
                metrics['return_on_equity'] = roe
                print(f"📊 ROE: {roe:.1f}%")
        
        # Calculate growth rates if we have multiple quarters
        if len(quarterly_data) >= 2:
            current_rev = quarterly_data[0].get('revenue')
            prev_rev = quarterly_data[1].get('revenue')
            if current_rev and prev_rev and prev_rev > 0:
                growth = ((float(current_rev) - float(prev_rev)) / float(prev_rev)) * 100
                metrics['revenue_qoq_growth'] = growth
                print(f"📈 Revenue QoQ Growth: {growth:.1f}%")
        
        return metrics
    
    def store_metrics(self, metrics):
        """Store calculated metrics in the database"""
        print(f"💾 Storing metrics for {metrics['symbol']}")
        
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
        
        print(f"✅ Successfully stored metrics for {metrics['symbol']}")
    
    def calculate_and_store(self, symbol):
        """Calculate and store fundamental metrics for a symbol"""
        try:
            metrics = self.calculate_metrics(symbol)
            if metrics:
                self.store_metrics(metrics)
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
    calculator = SECFundamentalsCalculator()
    
    symbols = ['AAPL']  # Start with AAPL
    
    for symbol in symbols:
        print(f"\n{'='*50}")
        print(f"Processing {symbol}")
        print(f"{'='*50}")
        
        success = calculator.calculate_and_store(symbol)
        if success:
            print(f"✅ Successfully processed {symbol}")
        else:
            print(f"❌ Failed to process {symbol}")

if __name__ == "__main__":
    main()
