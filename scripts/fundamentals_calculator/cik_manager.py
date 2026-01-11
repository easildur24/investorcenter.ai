#!/usr/bin/env python3
"""
CIK Manager - Handles CIK lookup from database and external sources
Provides CIK numbers needed for SEC EDGAR API calls
"""

import os
import psycopg2
import requests
from typing import Optional, Dict, List

class CIKManager:
    def __init__(self):
        # Hardcoded CIKs for major companies (backup)
        self.hardcoded_ciks = {
            # Mega Cap Tech
            'AAPL': '0000320193',    # Apple Inc.
            'MSFT': '0000789019',    # Microsoft Corporation
            'GOOGL': '0001652044',   # Alphabet Inc. (Class A)
            'GOOG': '0001652044',    # Alphabet Inc. (Class C) - same CIK
            'AMZN': '0001018724',    # Amazon.com Inc.
            'TSLA': '0001318605',    # Tesla Inc.
            'META': '0001326801',    # Meta Platforms Inc.
            'NVDA': '0001045810',    # NVIDIA Corporation
            
            # Large Cap Tech
            'NFLX': '0001065280',    # Netflix Inc.
            'ADBE': '0000796343',    # Adobe Inc.
            'CRM': '0001108524',     # Salesforce Inc.
            'ORCL': '0000777676',    # Oracle Corporation
            'INTC': '0000050863',    # Intel Corporation
            'AMD': '0000002488',     # Advanced Micro Devices Inc.
            'CSCO': '0000858877',    # Cisco Systems Inc.
            'IBM': '0000051143',     # International Business Machines
            
            # Healthcare & Biotech
            'HIMS': '0001773751',    # Hims & Hers Health Inc.
            'UNH': '0000731766',     # UnitedHealth Group Inc.
            'JNJ': '0000200406',     # Johnson & Johnson
            'PFE': '0000078003',     # Pfizer Inc.
            'ABBV': '0001551152',    # AbbVie Inc.
            'MRK': '0000310158',     # Merck & Co Inc.
            'TMO': '0000097745',     # Thermo Fisher Scientific Inc.
            
            # Financial Services
            'BRK-A': '0001067983',   # Berkshire Hathaway Inc. (Class A)
            'BRK-B': '0001067983',   # Berkshire Hathaway Inc. (Class B)
            'JPM': '0000019617',     # JPMorgan Chase & Co.
            'BAC': '0000070858',     # Bank of America Corp.
            'WFC': '0000072971',     # Wells Fargo & Company
            'GS': '0000886982',      # Goldman Sachs Group Inc.
            'MS': '0000895421',      # Morgan Stanley
            'V': '0001403161',       # Visa Inc.
            'MA': '0001141391',      # Mastercard Incorporated
            'PYPL': '0001633917',    # PayPal Holdings Inc.
            
            # Consumer & Retail
            'HD': '0000354950',      # The Home Depot Inc.
            'WMT': '0000104169',     # Walmart Inc.
            'COST': '0000909832',    # Costco Wholesale Corp.
            'TGT': '0000027419',     # Target Corporation
            'LOW': '0000060667',     # Lowe's Companies Inc.
            'SBUX': '0000829224',    # Starbucks Corporation
            'MCD': '0000063908',     # McDonald's Corporation
            'NKE': '0000320187',     # Nike Inc.
            'LULU': '0001397187',    # Lululemon Athletica Inc.
            
            # Consumer Goods
            'PG': '0000080424',      # Procter & Gamble Company
            'KO': '0000021344',      # The Coca-Cola Company
            'PEP': '0000077476',     # PepsiCo Inc.
            'UL': '0000025537',      # Unilever PLC
            'CL': '0000021665',      # Colgate-Palmolive Company
            
            # Industrial & Manufacturing
            'BA': '0000012927',      # Boeing Company
            'CAT': '0000018230',     # Caterpillar Inc.
            'GE': '0000040545',      # General Electric Company
            'MMM': '0000066740',     # 3M Company
            'HON': '0000773840',     # Honeywell International Inc.
            
            # Energy & Utilities
            'XOM': '0000034088',     # Exxon Mobil Corporation
            'CVX': '0000093410',     # Chevron Corporation
            'COP': '0001163165',     # ConocoPhillips
            'SLB': '0000087347',     # Schlumberger Limited
            'NEE': '0000753308',     # NextEra Energy Inc.
            
            # Telecom & Media
            'T': '0000732717',       # AT&T Inc.
            'VZ': '0000732712',      # Verizon Communications Inc.
            'CMCSA': '0001166691',   # Comcast Corporation
            'DIS': '0001001039',     # Walt Disney Company
            'NFLX': '0001065280',    # Netflix Inc. (duplicate but important)
            
            # Real Estate & REITs
            'AMT': '0001053507',     # American Tower Corporation
            'PLD': '0001045609',     # Prologis Inc.
            'CCI': '0001051470',     # Crown Castle International Corp.
            
            # Materials & Chemicals
            'LIN': '0001707925',     # Linde plc
            'APD': '0000002969',     # Air Products and Chemicals Inc.
            'DD': '0001666700',      # DuPont de Nemours Inc.
        }
        
        # Database connection info
        self.db_config = {
            'host': os.getenv('DB_HOST', 'localhost'),
            'port': os.getenv('DB_PORT', '5432'),
            'user': os.getenv('DB_USER', 'investorcenter'),
            'password': os.getenv('DB_PASSWORD', 'investorcenter123'),
            'database': os.getenv('DB_NAME', 'investorcenter_db')
        }
        
        self.polygon_api_key = os.getenv('POLYGON_API_KEY')

    def get_cik_from_database(self, ticker: str) -> Optional[str]:
        """Get CIK from our database"""
        try:
            conn = psycopg2.connect(**self.db_config)
            cursor = conn.cursor()
            
            cursor.execute(
                "SELECT cik FROM tickers WHERE symbol = %s AND cik IS NOT NULL",
                (ticker.upper(),)
            )
            
            result = cursor.fetchone()
            cursor.close()
            conn.close()
            
            if result:
                cik = result[0]
                print(f"✅ Found CIK in database for {ticker}: {cik}")
                return cik
            else:
                print(f"⚠️ No CIK in database for {ticker}")
                return None
                
        except Exception as e:
            print(f"⚠️ Database error for {ticker}: {e}")
            return None

    def get_cik_from_polygon(self, ticker: str) -> Optional[str]:
        """Get CIK from Polygon API"""
        if not self.polygon_api_key:
            print(f"⚠️ No Polygon API key for {ticker}")
            return None
        
        try:
            url = f"https://api.polygon.io/v3/reference/tickers/{ticker.upper()}"
            params = {'apikey': self.polygon_api_key}
            
            response = requests.get(url, params=params)
            response.raise_for_status()
            
            data = response.json()
            if data.get('status') == 'OK' and 'results' in data:
                cik = data['results'].get('cik')
                if cik:
                    # Format CIK with leading zeros (10 digits)
                    formatted_cik = f"{int(cik):010d}"
                    print(f"✅ Found CIK from Polygon for {ticker}: {formatted_cik}")
                    return formatted_cik
            
            print(f"⚠️ No CIK from Polygon for {ticker}")
            return None
            
        except Exception as e:
            print(f"⚠️ Polygon API error for {ticker}: {e}")
            return None

    def get_cik_hardcoded(self, ticker: str) -> Optional[str]:
        """Get CIK from hardcoded mapping (backup)"""
        cik = self.hardcoded_ciks.get(ticker.upper())
        if cik:
            print(f"✅ Found hardcoded CIK for {ticker}: {cik}")
            return cik
        else:
            print(f"⚠️ No hardcoded CIK for {ticker}")
            return None

    def get_cik(self, ticker: str) -> Optional[str]:
        """
        Get CIK for ticker using multiple sources (waterfall approach):
        1. Database (fastest)
        2. Polygon API (comprehensive)
        3. Hardcoded mapping (backup)
        """
        ticker = ticker.upper()
        
        print(f"🔍 Looking up CIK for {ticker}...")
        
        # Try database first
        cik = self.get_cik_from_database(ticker)
        if cik:
            return cik
        
        # Try Polygon API
        cik = self.get_cik_from_polygon(ticker)
        if cik:
            # TODO: Save to database for future use
            return cik
        
        # Try hardcoded mapping
        cik = self.get_cik_hardcoded(ticker)
        if cik:
            return cik
        
        print(f"❌ Could not find CIK for {ticker}")
        return None

    def get_supported_tickers(self) -> List[str]:
        """Get list of tickers we can calculate (have CIKs for)"""
        try:
            conn = psycopg2.connect(**self.db_config)
            cursor = conn.cursor()
            
            cursor.execute(
                "SELECT symbol FROM tickers WHERE cik IS NOT NULL ORDER BY symbol"
            )
            
            results = cursor.fetchall()
            cursor.close()
            conn.close()
            
            db_tickers = [row[0] for row in results]
            hardcoded_tickers = list(self.hardcoded_ciks.keys())
            
            # Combine and deduplicate
            all_tickers = list(set(db_tickers + hardcoded_tickers))
            
            print(f"✅ Found {len(all_tickers)} supported tickers")
            return sorted(all_tickers)
            
        except Exception as e:
            print(f"⚠️ Database error getting supported tickers: {e}")
            return sorted(self.hardcoded_ciks.keys())

# Convenience functions
def get_cik(ticker: str) -> Optional[str]:
    """Quick function to get CIK for a ticker"""
    manager = CIKManager()
    return manager.get_cik(ticker)

def get_supported_tickers() -> List[str]:
    """Quick function to get all supported tickers"""
    manager = CIKManager()
    return manager.get_supported_tickers()
