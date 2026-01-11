#!/usr/bin/env python3
"""
Comprehensive SEC Filing Financial Analyzer

This script analyzes SEC filing documents to extract comprehensive financial metrics
including income statement, balance sheet, cash flow, and key financial ratios.

Usage:
    python comprehensive_financial_analyzer.py [ticker] [documents_directory]
"""

import os
import re
import sys
from datetime import datetime
from typing import List, Dict, Optional, Tuple
import argparse


class ComprehensiveFinancialAnalyzer:
    def __init__(self, ticker: str, documents_dir: str):
        self.ticker = ticker.upper()
        self.documents_dir = documents_dir
        self.filing_data = []
        self.current_price = 55.50  # Default for HIMS - should be updated via API
        
    def extract_filing_data(self, filepath: str) -> Dict:
        """Extract comprehensive financial data from a single SEC filing."""
        try:
            with open(filepath, 'r', encoding='utf-8') as f:
                content = f.read()
        except Exception as e:
            print(f"Error reading {filepath}: {e}")
            return {}
        
        filename = os.path.basename(filepath)
        date_match = re.search(r'(\d{8})', filename)
        filing_date = date_match.group(1) if date_match else "unknown"
        
        quarter_info = self._extract_quarter_info(content, filing_date)
        
        # Extract all financial metrics
        financial_data = {
            'filename': filename,
            'filing_date': filing_date,
            'quarter': quarter_info['quarter'],
            'year': quarter_info['year'],
            
            # Income Statement
            'revenue_ttm': self._extract_value(content, 'RevenueFromContractWithCustomerExcludingAssessedTax'),
            'revenue_quarterly': self._extract_value(content, 'RevenueFromContractWithCustomerExcludingAssessedTax'),
            'cost_of_revenue': self._extract_value(content, 'CostOfRevenue'),
            'gross_profit': self._extract_value(content, 'GrossProfit'),
            'operating_expenses': self._extract_value(content, 'OperatingExpenses'),
            'marketing_expense': self._extract_value(content, 'MarketingExpense'),
            'general_admin_expense': self._extract_value(content, 'GeneralAndAdministrativeExpense'),
            'operating_income': self._extract_value(content, 'OperatingIncomeLoss'),
            'net_income_ttm': self._extract_value(content, 'NetIncomeLoss'),
            'net_income_quarterly': self._extract_value(content, 'NetIncomeLoss'),
            'ebit_ttm': self._extract_value(content, 'OperatingIncomeLoss'),
            'ebitda_ttm': self._calculate_ebitda(content),
            'ebitda_quarterly': self._calculate_ebitda(content),
            
            # Balance Sheet
            'total_assets': self._extract_value(content, 'Assets'),
            'total_liabilities': self._extract_value(content, 'Liabilities'),
            'stockholders_equity': self._extract_value(content, 'StockholdersEquity'),
            'cash_and_equivalents': self._extract_value(content, 'CashAndCashEquivalentsAtCarryingValue'),
            'short_term_investments': self._extract_value(content, 'ShortTermInvestments'),
            'inventory': self._extract_value(content, 'InventoryNet'),
            'total_long_term_assets': self._extract_long_term_assets(content),
            'total_long_term_debt': self._extract_value(content, 'ConvertibleDebtNoncurrent'),
            'book_value': self._extract_value(content, 'StockholdersEquity'),
            
            # Cash Flow
            'cash_from_operations_ttm': self._extract_value(content, 'NetCashProvidedByUsedInOperatingActivities'),
            'cash_from_investing_ttm': self._extract_value(content, 'NetCashProvidedByUsedInInvestingActivities'),
            'cash_from_financing_ttm': self._extract_value(content, 'NetCashProvidedByUsedInFinancingActivities'),
            'capital_expenditures_ttm': self._extract_value(content, 'PaymentsToAcquireProductiveAssets'),
            'free_cash_flow': self._calculate_free_cash_flow(content),
            
            # Share data
            'shares_outstanding': self._extract_value(content, 'SharesOutstanding'),
            'weighted_avg_shares_basic': self._extract_value(content, 'WeightedAverageNumberOfSharesOutstandingBasic'),
            'weighted_avg_shares_diluted': self._extract_value(content, 'WeightedAverageNumberOfDilutedSharesOutstanding'),
            'eps_basic_ttm': self._extract_value(content, 'EarningsPerShareBasic'),
            'eps_diluted_ttm': self._extract_value(content, 'EarningsPerShareDiluted'),
            'eps_basic_quarterly': self._extract_value(content, 'EarningsPerShareBasic'),
            'eps_diluted_quarterly': self._extract_value(content, 'EarningsPerShareDiluted'),
        }
        
        return financial_data
    
    def _extract_quarter_info(self, content: str, filing_date: str) -> Dict:
        """Extract quarter and year information."""
        year_match = re.search(r'dei:DocumentFiscalYearFocus[^>]*>(\d{4})<', content)
        period_match = re.search(r'dei:DocumentFiscalPeriodFocus[^>]*>(Q\d)<', content)
        
        if year_match and period_match:
            return {
                'year': int(year_match.group(1)),
                'quarter': period_match.group(1)
            }
        
        if len(filing_date) == 8:
            year = int(filing_date[:4])
            month = int(filing_date[4:6])
            quarter = f"Q{((month - 1) // 3) + 1}"
            return {'year': year, 'quarter': quarter}
        
        return {'year': None, 'quarter': None}
    
    def _extract_value(self, content: str, gaap_element: str) -> Optional[float]:
        """Extract a specific US-GAAP value from XBRL content."""
        pattern = f'us-gaap:{gaap_element}[^>]*>([^<]+)<'
        matches = re.findall(pattern, content)
        
        if matches:
            # Take the first non-empty match and clean it
            for match in matches:
                cleaned = self._clean_financial_value(match)
                if cleaned is not None:
                    return cleaned
        return None
    
    def _clean_financial_value(self, value: str) -> Optional[float]:
        """Clean and convert financial values from XBRL format."""
        if not value or value == '&#8212;':  # Em dash for zero/not applicable
            return None
            
        # Remove commas and handle parentheses (negative values)
        cleaned = value.replace(',', '').strip()
        
        # Handle negative values in parentheses
        if cleaned.startswith('(') and cleaned.endswith(')'):
            cleaned = '-' + cleaned[1:-1]
        
        try:
            return float(cleaned)
        except ValueError:
            return None
    
    def _calculate_ebitda(self, content: str) -> Optional[float]:
        """Calculate EBITDA from available components."""
        operating_income = self._extract_value(content, 'OperatingIncomeLoss')
        depreciation = self._extract_value(content, 'DepreciationDepletionAndAmortization')
        
        if operating_income is not None:
            ebitda = operating_income
            if depreciation:
                ebitda += depreciation
            return ebitda
        return None
    
    def _calculate_free_cash_flow(self, content: str) -> Optional[float]:
        """Calculate free cash flow."""
        operating_cash_flow = self._extract_value(content, 'NetCashProvidedByUsedInOperatingActivities')
        capex = self._extract_value(content, 'PaymentsToAcquireProductiveAssets')
        
        if operating_cash_flow is not None:
            fcf = operating_cash_flow
            if capex:
                fcf -= capex
            return fcf
        return operating_cash_flow
    
    def _extract_long_term_assets(self, content: str) -> Optional[float]:
        """Calculate total long-term assets."""
        total_assets = self._extract_value(content, 'Assets')
        current_assets = self._extract_value(content, 'AssetsCurrent')
        
        if total_assets and current_assets:
            return total_assets - current_assets
        return None
    
    def analyze_all_filings(self) -> List[Dict]:
        """Analyze all SEC filing documents."""
        if not os.path.exists(self.documents_dir):
            raise FileNotFoundError(f"Documents directory not found: {self.documents_dir}")
        
        filing_files = [f for f in os.listdir(self.documents_dir) 
                       if f.endswith('.htm') and self.ticker.lower() in f.lower()]
        
        if not filing_files:
            raise ValueError(f"No SEC filing documents found for {self.ticker}")
        
        print(f"Found {len(filing_files)} filing documents for {self.ticker}")
        
        for filename in sorted(filing_files):
            filepath = os.path.join(self.documents_dir, filename)
            filing_data = self.extract_filing_data(filepath)
            if filing_data:
                self.filing_data.append(filing_data)
        
        return self.filing_data
    
    def calculate_ttm_metrics(self) -> Dict:
        """Calculate trailing twelve months metrics."""
        if not self.filing_data:
            raise ValueError("No filing data available")
        
        # Sort by year and quarter
        sorted_filings = sorted(self.filing_data, 
                              key=lambda x: (x['year'] or 0, x['quarter'] or 'Q0'), 
                              reverse=True)
        
        # Take last 4 quarters for TTM calculations
        ttm_filings = sorted_filings[:4]
        
        ttm_metrics = {
            'revenue_ttm': sum(f['revenue_quarterly'] or 0 for f in ttm_filings),
            'net_income_ttm': sum(f['net_income_quarterly'] or 0 for f in ttm_filings),
            'cash_from_operations_ttm': sum(f['cash_from_operations_ttm'] or 0 for f in ttm_filings),
            'eps_basic_ttm': sum(f['eps_basic_quarterly'] or 0 for f in ttm_filings),
            'eps_diluted_ttm': sum(f['eps_diluted_quarterly'] or 0 for f in ttm_filings),
        }
        
        # Get most recent quarter data for balance sheet items
        latest_filing = sorted_filings[0]
        
        return {
            **ttm_metrics,
            'latest_filing': latest_filing,
            'quarters_analyzed': ttm_filings
        }
    
    def calculate_financial_ratios(self) -> Dict:
        """Calculate comprehensive financial ratios and metrics."""
        ttm_data = self.calculate_ttm_metrics()
        latest = ttm_data['latest_filing']
        
        # Get shares outstanding (in millions)
        shares_outstanding = (latest['shares_outstanding'] or 0) / 1_000_000
        
        # Market metrics
        market_cap = self.current_price * shares_outstanding if shares_outstanding else None
        
        # Valuation ratios
        pe_ratio = self.current_price / ttm_data['eps_diluted_ttm'] if ttm_data['eps_diluted_ttm'] else None
        price_to_book = self.current_price / (latest['book_value'] / shares_outstanding) if latest['book_value'] and shares_outstanding else None
        
        # Profitability ratios
        gross_margin = (latest['gross_profit'] / latest['revenue_quarterly'] * 100) if latest['gross_profit'] and latest['revenue_quarterly'] else None
        operating_margin = (ttm_data['net_income_ttm'] / ttm_data['revenue_ttm'] * 100) if ttm_data['net_income_ttm'] and ttm_data['revenue_ttm'] else None
        
        # Efficiency ratios
        roa = (ttm_data['net_income_ttm'] / latest['total_assets'] * 100) if ttm_data['net_income_ttm'] and latest['total_assets'] else None
        roe = (ttm_data['net_income_ttm'] / latest['stockholders_equity'] * 100) if ttm_data['net_income_ttm'] and latest['stockholders_equity'] else None
        
        # Growth calculations (YoY if we have enough data)
        revenue_growth = self._calculate_growth_rate('revenue_quarterly')
        eps_growth = self._calculate_growth_rate('eps_diluted_quarterly')
        ebitda_growth = self._calculate_growth_rate('ebitda_quarterly')
        
        return {
            'market_cap': market_cap,
            'enterprise_value': market_cap + (latest['total_long_term_debt'] or 0) - (latest['cash_and_equivalents'] or 0) if market_cap else None,
            'price': self.current_price,
            'pe_ratio': pe_ratio,
            'price_to_book': price_to_book,
            'gross_margin': gross_margin,
            'operating_margin': operating_margin,
            'roa': roa,
            'roe': roe,
            'revenue_growth_yoy': revenue_growth,
            'eps_growth_yoy': eps_growth,
            'ebitda_growth_yoy': ebitda_growth,
            'shares_outstanding': shares_outstanding,
            'free_cash_flow': latest['free_cash_flow'],
        }
    
    def _calculate_growth_rate(self, metric: str) -> Optional[float]:
        """Calculate year-over-year growth rate for a metric."""
        if len(self.filing_data) < 4:
            return None
        
        sorted_filings = sorted(self.filing_data, 
                              key=lambda x: (x['year'] or 0, x['quarter'] or 'Q0'), 
                              reverse=True)
        
        current_quarter = sorted_filings[0][metric]
        year_ago_quarter = sorted_filings[3][metric] if len(sorted_filings) > 3 else None
        
        if current_quarter and year_ago_quarter and year_ago_quarter != 0:
            return ((current_quarter - year_ago_quarter) / abs(year_ago_quarter)) * 100
        
        return None
    
    def print_comprehensive_analysis(self):
        """Print comprehensive financial analysis."""
        ttm_data = self.calculate_ttm_metrics()
        ratios = self.calculate_financial_ratios()
        latest = ttm_data['latest_filing']
        
        print("\n" + "="*80)
        print(f"COMPREHENSIVE FINANCIAL ANALYSIS FOR {self.ticker}")
        print("="*80)
        
        # Income Statement (TTM)
        print(f"\n📊 INCOME STATEMENT (TTM)")
        print("-" * 40)
        print(f"Revenue (TTM):              ${ttm_data['revenue_ttm']:,.0f}K")
        print(f"Net Income (TTM):           ${ttm_data['net_income_ttm']:,.0f}K")
        print(f"EBIT (TTM):                 ${latest['ebit_ttm'] or 0:,.0f}K")
        print(f"EBITDA (TTM):               ${latest['ebitda_ttm'] or 0:,.0f}K")
        
        # Quarterly metrics
        print(f"\n📈 QUARTERLY METRICS")
        print("-" * 40)
        print(f"Revenue (Quarterly):        ${latest['revenue_quarterly'] or 0:,.0f}K")
        print(f"Net Income (Quarterly):     ${latest['net_income_quarterly'] or 0:,.0f}K")
        print(f"EBITDA (Quarterly):         ${latest['ebitda_quarterly'] or 0:,.0f}K")
        print(f"Revenue QoQ Growth:         {ratios['revenue_growth_yoy'] or 0:.1f}%")
        print(f"EPS Diluted (Quarterly):    ${latest['eps_diluted_quarterly'] or 0:.2f}")
        print(f"EPS QoQ Growth:             {ratios['eps_growth_yoy'] or 0:.1f}%")
        print(f"EBITDA QoQ Growth:          {ratios['ebitda_growth_yoy'] or 0:.1f}%")
        
        # Balance Sheet
        print(f"\n🏦 BALANCE SHEET (Quarterly)")
        print("-" * 40)
        print(f"Total Assets (Quarterly):   ${latest['total_assets'] or 0:,.0f}K")
        print(f"Total Liabilities (Quarterly): ${latest['total_liabilities'] or 0:,.0f}K")
        print(f"Shareholders Equity (Quarterly): ${latest['stockholders_equity'] or 0:,.0f}K")
        print(f"Cash and Short Term Investments: ${(latest['cash_and_equivalents'] or 0) + (latest['short_term_investments'] or 0):,.0f}K")
        print(f"Total Long Term Assets:     ${latest['total_long_term_assets'] or 0:,.0f}K")
        print(f"Total Long Term Debt:       ${latest['total_long_term_debt'] or 0:,.0f}K")
        print(f"Book Value (Quarterly):     ${latest['book_value'] or 0:,.0f}K")
        
        # Cash Flow
        print(f"\n💰 CASH FLOW (TTM)")
        print("-" * 40)
        print(f"Cash from Operations (TTM): ${ttm_data['cash_from_operations_ttm']:,.0f}K")
        print(f"Cash from Investing (TTM):  ${latest['cash_from_investing_ttm'] or 0:,.0f}K")
        print(f"Cash from Financing (TTM):  ${latest['cash_from_financing_ttm'] or 0:,.0f}K")
        print(f"Capital Expenditures (TTM): ${latest['capital_expenditures_ttm'] or 0:,.0f}K")
        print(f"Free Cash Flow:             ${ratios['free_cash_flow'] or 0:,.0f}K")
        
        # Common Size Statements
        print(f"\n📋 COMMON SIZE STATEMENTS")
        print("-" * 40)
        print(f"EPS Diluted (TTM):          ${ttm_data['eps_diluted_ttm']:.2f}")
        print(f"EPS Basic (TTM):            ${ttm_data['eps_basic_ttm']:.2f}")
        print(f"Shares Outstanding:         {ratios['shares_outstanding']:,.0f}M")
        
        # Performance, Risk and Estimates
        print(f"\n📈 PERFORMANCE & VALUATION")
        print("-" * 40)
        print(f"Current Price:              ${ratios['price']:.2f}")
        print(f"Market Cap:                 ${ratios['market_cap'] or 0:,.0f}M")
        print(f"Enterprise Value:           ${ratios['enterprise_value'] or 0:,.0f}M")
        print(f"PE Ratio:                   {ratios['pe_ratio']:.2f}")
        print(f"Price to Book:              {ratios['price_to_book'] or 0:.2f}")
        
        # Profitability
        print(f"\n💼 PROFITABILITY")
        print("-" * 40)
        print(f"Operating Margin (TTM):     {ratios['operating_margin'] or 0:.1f}%")
        print(f"Gross Profit Margin:        {ratios['gross_margin'] or 0:.1f}%")
        print(f"Return on Assets:           {ratios['roa'] or 0:.1f}%")
        print(f"Return on Equity:           {ratios['roe'] or 0:.1f}%")
        
        # Analysis date
        print(f"\nAnalysis Date: {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}")
        
        # Quarterly breakdown
        print(f"\n📅 QUARTERLY EPS BREAKDOWN")
        print("-" * 40)
        for filing in ttm_data['quarters_analyzed']:
            print(f"{filing['quarter']} {filing['year']}: ${filing['eps_diluted_quarterly'] or 0:.2f}")


def main():
    parser = argparse.ArgumentParser(description='Comprehensive financial analysis from SEC filings')
    parser.add_argument('ticker', nargs='?', default='HIMS', help='Stock ticker symbol (default: HIMS)')
    parser.add_argument('documents_dir', nargs='?', default='documents', help='Directory containing SEC filing documents (default: documents)')
    
    args = parser.parse_args()
    
    try:
        analyzer = ComprehensiveFinancialAnalyzer(args.ticker, args.documents_dir)
        analyzer.analyze_all_filings()
        analyzer.print_comprehensive_analysis()
        
    except Exception as e:
        print(f"Error: {e}")
        sys.exit(1)


if __name__ == "__main__":
    if len(sys.argv) == 1:
        print("Running comprehensive financial analysis for HIMS...")
        analyzer = ComprehensiveFinancialAnalyzer("HIMS", "documents")
        try:
            analyzer.analyze_all_filings()
            analyzer.print_comprehensive_analysis()
        except Exception as e:
            print(f"Error: {e}")
            print("\nUsage: python comprehensive_financial_analyzer.py [ticker] [documents_directory]")
    else:
        main()


