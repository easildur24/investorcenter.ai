#!/usr/bin/env python3
"""
Backfill Screener Data - Populates missing sector, industry, dividend yield, and revenue growth data.

This script addresses three data gaps for the stock screener:
1. Sector/Industry - Maps SIC codes to GICS sectors
2. Dividend Yield - Fetches dividend data from Polygon API
3. Revenue Growth - Triggers recalculation of fundamental metrics

Usage:
    python backfill_screener_data.py --sectors        # Backfill sector/industry from SIC codes
    python backfill_screener_data.py --dividends      # Fetch dividend data from Polygon
    python backfill_screener_data.py --all            # Run all backfills
"""

import argparse
import asyncio
import logging
import os
import sys
import time
from datetime import datetime, timedelta
from typing import Dict, List, Optional, Tuple

import psycopg2
import requests
from psycopg2.extras import RealDictCursor, execute_batch

# Setup logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)


# =============================================================================
# SIC TO GICS SECTOR MAPPING
# =============================================================================

# Standard Industrial Classification (SIC) to GICS Sector mapping
# SIC codes are 4-digit, we match by first 2 digits for broad categories
SIC_TO_SECTOR = {
    # Agriculture, Forestry, Fishing (01-09) -> Basic Materials
    "01": ("Basic Materials", "Agricultural Products"),
    "02": ("Basic Materials", "Forest Products"),
    "07": ("Basic Materials", "Agricultural Services"),
    "08": ("Basic Materials", "Forestry"),
    "09": ("Basic Materials", "Fishing"),

    # Mining (10-14) -> Energy / Basic Materials
    "10": ("Basic Materials", "Metal Mining"),
    "12": ("Energy", "Coal Mining"),
    "13": ("Energy", "Oil & Gas Extraction"),
    "14": ("Basic Materials", "Mining & Quarrying"),

    # Construction (15-17) -> Industrials
    "15": ("Industrials", "Construction - General"),
    "16": ("Industrials", "Construction - Heavy"),
    "17": ("Industrials", "Construction - Special Trade"),

    # Manufacturing - Food, Tobacco, Textiles (20-23) -> Consumer Defensive/Cyclical
    "20": ("Consumer Defensive", "Food Products"),
    "21": ("Consumer Defensive", "Tobacco"),
    "22": ("Consumer Cyclical", "Textile Mills"),
    "23": ("Consumer Cyclical", "Apparel Manufacturing"),

    # Manufacturing - Wood, Paper, Printing (24-27) -> Basic Materials / Communication Services
    "24": ("Basic Materials", "Lumber & Wood Products"),
    "25": ("Consumer Cyclical", "Furniture & Fixtures"),
    "26": ("Basic Materials", "Paper & Allied Products"),
    "27": ("Communication Services", "Printing & Publishing"),

    # Manufacturing - Chemicals, Petroleum, Rubber (28-30) -> Basic Materials / Healthcare
    "28": ("Healthcare", "Chemicals & Pharmaceuticals"),
    "29": ("Energy", "Petroleum Refining"),
    "30": ("Basic Materials", "Rubber & Plastics"),

    # Manufacturing - Stone, Metal, Machinery (31-35) -> Industrials / Basic Materials
    "31": ("Consumer Cyclical", "Leather Products"),
    "32": ("Basic Materials", "Stone, Clay, Glass Products"),
    "33": ("Basic Materials", "Primary Metal Industries"),
    "34": ("Industrials", "Fabricated Metal Products"),
    "35": ("Industrials", "Industrial Machinery"),

    # Manufacturing - Electronics, Transportation Equipment (36-39) -> Technology / Industrials
    "36": ("Technology", "Electronic Equipment"),
    "37": ("Consumer Cyclical", "Transportation Equipment"),
    "38": ("Healthcare", "Measuring & Medical Instruments"),
    "39": ("Consumer Cyclical", "Miscellaneous Manufacturing"),

    # Transportation & Utilities (40-49) -> Industrials / Utilities
    "40": ("Industrials", "Railroad Transportation"),
    "41": ("Industrials", "Local & Suburban Transit"),
    "42": ("Industrials", "Motor Freight Transportation"),
    "43": ("Industrials", "Postal Service"),
    "44": ("Industrials", "Water Transportation"),
    "45": ("Industrials", "Air Transportation"),
    "46": ("Industrials", "Pipelines"),
    "47": ("Industrials", "Transportation Services"),
    "48": ("Communication Services", "Communications"),
    "49": ("Utilities", "Electric, Gas & Sanitary Services"),

    # Wholesale Trade (50-51) -> Consumer Cyclical
    "50": ("Consumer Cyclical", "Wholesale Trade - Durable Goods"),
    "51": ("Consumer Defensive", "Wholesale Trade - Nondurable Goods"),

    # Retail Trade (52-59) -> Consumer Cyclical / Consumer Defensive
    "52": ("Consumer Cyclical", "Building Materials Retail"),
    "53": ("Consumer Cyclical", "General Merchandise Stores"),
    "54": ("Consumer Defensive", "Food Stores"),
    "55": ("Consumer Cyclical", "Auto Dealers & Gas Stations"),
    "56": ("Consumer Cyclical", "Apparel & Accessory Stores"),
    "57": ("Consumer Cyclical", "Home Furniture Stores"),
    "58": ("Consumer Cyclical", "Eating & Drinking Places"),
    "59": ("Consumer Cyclical", "Miscellaneous Retail"),

    # Finance, Insurance, Real Estate (60-67) -> Financial Services / Real Estate
    "60": ("Financial Services", "Depository Institutions"),
    "61": ("Financial Services", "Non-Depository Credit"),
    "62": ("Financial Services", "Security & Commodity Brokers"),
    "63": ("Financial Services", "Insurance Carriers"),
    "64": ("Financial Services", "Insurance Agents"),
    "65": ("Real Estate", "Real Estate"),
    "67": ("Financial Services", "Holding & Investment Offices"),

    # Services (70-89) -> Various
    "70": ("Consumer Cyclical", "Hotels & Lodging"),
    "72": ("Consumer Cyclical", "Personal Services"),
    "73": ("Technology", "Business Services"),
    "75": ("Consumer Cyclical", "Auto Repair & Services"),
    "76": ("Consumer Cyclical", "Miscellaneous Repair Services"),
    "78": ("Communication Services", "Motion Pictures"),
    "79": ("Communication Services", "Amusement & Recreation"),
    "80": ("Healthcare", "Health Services"),
    "81": ("Industrials", "Legal Services"),
    "82": ("Consumer Cyclical", "Educational Services"),
    "83": ("Industrials", "Social Services"),
    "84": ("Industrials", "Museums & Botanical Gardens"),
    "86": ("Industrials", "Membership Organizations"),
    "87": ("Technology", "Engineering & Management Services"),
    "89": ("Industrials", "Miscellaneous Services"),

    # Public Administration (90-99) -> Industrials
    "91": ("Industrials", "Executive & Legislative"),
    "92": ("Industrials", "Justice & Public Order"),
    "93": ("Industrials", "Public Finance"),
    "94": ("Industrials", "Administration of Human Resources"),
    "95": ("Industrials", "Environmental Administration"),
    "96": ("Industrials", "Economic Programs Administration"),
    "97": ("Industrials", "National Security"),
    "99": ("Industrials", "Nonclassifiable Establishments"),
}

# More specific SIC code mappings (full 4-digit codes for important sectors)
SPECIFIC_SIC_MAPPINGS = {
    # Technology - Software & Services
    "7370": ("Technology", "Software - Infrastructure"),
    "7371": ("Technology", "Software - Application"),
    "7372": ("Technology", "Software - Packaged"),
    "7373": ("Technology", "IT Services"),
    "7374": ("Technology", "Data Processing"),
    "7375": ("Technology", "Information Retrieval Services"),
    "7376": ("Technology", "Computer Facilities Management"),
    "7377": ("Technology", "Computer Rental & Leasing"),
    "7378": ("Technology", "Computer Maintenance & Repair"),
    "7379": ("Technology", "Computer Related Services"),

    # Technology - Hardware
    "3571": ("Technology", "Electronic Computers"),
    "3572": ("Technology", "Computer Storage Devices"),
    "3575": ("Technology", "Computer Terminals"),
    "3576": ("Technology", "Computer Communications Equipment"),
    "3577": ("Technology", "Computer Peripheral Equipment"),
    "3578": ("Technology", "Calculating & Accounting Machines"),

    # Technology - Semiconductors
    "3674": ("Technology", "Semiconductors & Related Devices"),

    # Healthcare - Pharma & Biotech
    "2833": ("Healthcare", "Medicinal Chemicals"),
    "2834": ("Healthcare", "Pharmaceutical Preparations"),
    "2835": ("Healthcare", "Diagnostic Substances"),
    "2836": ("Healthcare", "Biological Products"),
    "3826": ("Healthcare", "Laboratory Analytical Instruments"),
    "3841": ("Healthcare", "Surgical & Medical Instruments"),
    "3842": ("Healthcare", "Orthopedic & Prosthetic Appliances"),
    "3843": ("Healthcare", "Dental Equipment"),
    "3844": ("Healthcare", "X-Ray & Electromedical Equipment"),
    "3845": ("Healthcare", "Electromedical & Electrotherapeutic"),

    # Financial Services - Banks
    "6020": ("Financial Services", "Commercial Banking"),
    "6021": ("Financial Services", "National Commercial Banks"),
    "6022": ("Financial Services", "State Commercial Banks"),
    "6029": ("Financial Services", "Commercial Banks"),
    "6035": ("Financial Services", "Savings Institutions"),
    "6036": ("Financial Services", "Savings Institutions"),

    # Financial Services - Investment
    "6211": ("Financial Services", "Security Brokers & Dealers"),
    "6221": ("Financial Services", "Commodity Contracts"),
    "6282": ("Financial Services", "Investment Advice"),
    "6311": ("Financial Services", "Life Insurance"),
    "6321": ("Financial Services", "Accident & Health Insurance"),
    "6324": ("Financial Services", "Hospital & Medical Service Plans"),
    "6331": ("Financial Services", "Fire, Marine & Casualty Insurance"),
    "6351": ("Financial Services", "Surety Insurance"),
    "6361": ("Financial Services", "Title Insurance"),
    "6411": ("Financial Services", "Insurance Agents & Brokers"),
    "6719": ("Financial Services", "Holding Companies"),
    "6726": ("Financial Services", "Other Investment Offices"),
    "6798": ("Financial Services", "Real Estate Investment Trusts"),
    "6799": ("Financial Services", "Investors"),

    # Energy
    "1311": ("Energy", "Crude Petroleum & Natural Gas"),
    "1381": ("Energy", "Drilling Oil & Gas Wells"),
    "1382": ("Energy", "Oil & Gas Field Exploration"),
    "1389": ("Energy", "Oil & Gas Field Services"),
    "2911": ("Energy", "Petroleum Refining"),
    "4911": ("Utilities", "Electric Services"),
    "4922": ("Utilities", "Natural Gas Transmission"),
    "4923": ("Utilities", "Natural Gas Transmission & Distribution"),
    "4924": ("Utilities", "Natural Gas Distribution"),
    "4931": ("Utilities", "Electric & Other Services Combined"),
    "4932": ("Utilities", "Gas & Other Services Combined"),
    "4941": ("Utilities", "Water Supply"),

    # Consumer - Retail
    "5311": ("Consumer Cyclical", "Department Stores"),
    "5331": ("Consumer Cyclical", "Variety Stores"),
    "5399": ("Consumer Cyclical", "Miscellaneous General Merchandise"),
    "5411": ("Consumer Defensive", "Grocery Stores"),
    "5412": ("Consumer Defensive", "Convenience Stores"),
    "5812": ("Consumer Cyclical", "Eating Places"),
    "5912": ("Consumer Defensive", "Drug Stores"),
    "5941": ("Consumer Cyclical", "Sporting Goods Stores"),
    "5944": ("Consumer Cyclical", "Jewelry Stores"),
    "5945": ("Consumer Cyclical", "Hobby, Toy & Game Shops"),
    "5961": ("Consumer Cyclical", "Catalog & Mail-Order Houses"),

    # Consumer - Auto
    "3711": ("Consumer Cyclical", "Motor Vehicles & Car Bodies"),
    "3714": ("Consumer Cyclical", "Motor Vehicle Parts"),
    "5511": ("Consumer Cyclical", "Motor Vehicle Dealers"),
    "5521": ("Consumer Cyclical", "Motor Vehicle Dealers - Used"),
    "7510": ("Consumer Cyclical", "Auto Rental & Leasing"),

    # Communication Services
    "4812": ("Communication Services", "Radiotelephone Communications"),
    "4813": ("Communication Services", "Telephone Communications"),
    "4822": ("Communication Services", "Telegraph Communications"),
    "4832": ("Communication Services", "Radio Broadcasting"),
    "4833": ("Communication Services", "Television Broadcasting"),
    "4841": ("Communication Services", "Cable & Other Pay TV"),
    "7812": ("Communication Services", "Motion Picture Production"),
    "7819": ("Communication Services", "Motion Picture Services"),
    "7822": ("Communication Services", "Motion Picture Distribution"),
    "7832": ("Communication Services", "Motion Picture Theaters"),
    "7941": ("Communication Services", "Professional Sports Clubs"),
    "7996": ("Communication Services", "Amusement Parks"),
    "7997": ("Communication Services", "Membership Sports Clubs"),

    # Industrials - Aerospace & Defense
    "3720": ("Industrials", "Aircraft & Parts"),
    "3721": ("Industrials", "Aircraft"),
    "3724": ("Industrials", "Aircraft Engines"),
    "3728": ("Industrials", "Aircraft Parts"),
    "3760": ("Industrials", "Guided Missiles & Space Vehicles"),
    "3761": ("Industrials", "Guided Missiles & Space Vehicles"),
    "3764": ("Industrials", "Guided Missile & Space Parts"),
    "3769": ("Industrials", "Guided Missile Parts"),
    "3812": ("Industrials", "Search, Detection & Navigation"),

    # Real Estate
    "6500": ("Real Estate", "Real Estate"),
    "6510": ("Real Estate", "Real Estate Operators"),
    "6512": ("Real Estate", "Operators of Nonresidential Buildings"),
    "6513": ("Real Estate", "Operators of Apartment Buildings"),
    "6531": ("Real Estate", "Real Estate Agents & Managers"),
    "6532": ("Real Estate", "Real Estate Dealers"),
    "6552": ("Real Estate", "Land Subdividers & Developers"),
}


def map_sic_to_sector(sic_code: str) -> Tuple[Optional[str], Optional[str]]:
    """Map SIC code to GICS sector and industry.

    Args:
        sic_code: 4-digit SIC code string

    Returns:
        Tuple of (sector, industry), or (None, None) if no mapping found
    """
    if not sic_code:
        return None, None

    sic_code = str(sic_code).strip()

    # Try specific 4-digit mapping first
    if sic_code in SPECIFIC_SIC_MAPPINGS:
        return SPECIFIC_SIC_MAPPINGS[sic_code]

    # Fall back to 2-digit category mapping
    if len(sic_code) >= 2:
        prefix = sic_code[:2]
        if prefix in SIC_TO_SECTOR:
            return SIC_TO_SECTOR[prefix]

    return None, None


class ScreenerDataBackfill:
    """Backfill missing data for stock screener."""

    def __init__(self):
        self.db_config = {
            'host': os.environ.get('DB_HOST', 'localhost'),
            'port': int(os.environ.get('DB_PORT', 5432)),
            'database': os.environ.get('DB_NAME', 'investorcenter_db'),
            'user': os.environ.get('DB_USER'),
            'password': os.environ.get('DB_PASSWORD')
        }

        self.polygon_api_key = os.environ.get('POLYGON_API_KEY', 'Q9LhuSPrdj8Fqv9ejYqwXF6AKv7YAsWa')
        self.conn = None

        self.stats = {
            'sectors_updated': 0,
            'sectors_skipped': 0,
            'dividends_fetched': 0,
            'dividends_inserted': 0,
            'api_calls': 0,
            'errors': 0
        }

    def connect_db(self) -> bool:
        """Connect to PostgreSQL database."""
        try:
            self.conn = psycopg2.connect(**self.db_config)
            logger.info(f"Connected to database: {self.db_config['database']}")
            return True
        except Exception as e:
            logger.error(f"Database connection failed: {e}")
            return False

    # =========================================================================
    # SECTOR/INDUSTRY BACKFILL
    # =========================================================================

    def backfill_sectors(self) -> int:
        """Backfill sector and industry from SIC codes.

        Returns:
            Number of tickers updated
        """
        logger.info("=== Starting Sector/Industry Backfill ===")

        try:
            with self.conn.cursor(cursor_factory=RealDictCursor) as cursor:
                # Get tickers with SIC codes but missing sector
                cursor.execute("""
                    SELECT id, symbol, sic_code, sic_description, sector, industry
                    FROM tickers
                    WHERE sic_code IS NOT NULL
                      AND sic_code != ''
                      AND asset_type IN ('stock', 'CS')
                      AND (sector IS NULL OR sector = '' OR industry IS NULL OR industry = '')
                    ORDER BY market_cap DESC NULLS LAST
                """)

                tickers = cursor.fetchall()
                logger.info(f"Found {len(tickers)} tickers to update")

                updates = []
                for ticker in tickers:
                    sector, industry = map_sic_to_sector(ticker['sic_code'])

                    if sector:
                        updates.append((sector, industry or ticker.get('sic_description', ''), ticker['id']))
                        self.stats['sectors_updated'] += 1
                    else:
                        self.stats['sectors_skipped'] += 1

                # Batch update
                if updates:
                    execute_batch(cursor, """
                        UPDATE tickers
                        SET sector = %s, industry = %s, updated_at = CURRENT_TIMESTAMP
                        WHERE id = %s
                    """, updates, page_size=500)

                    self.conn.commit()
                    logger.info(f"Updated {len(updates)} tickers with sector/industry")

                return len(updates)

        except Exception as e:
            logger.error(f"Error backfilling sectors: {e}")
            self.conn.rollback()
            return 0

    # =========================================================================
    # DIVIDEND DATA FETCH
    # =========================================================================

    def ensure_dividends_table(self):
        """Create dividends table if it doesn't exist."""
        try:
            with self.conn.cursor() as cursor:
                cursor.execute("""
                    CREATE TABLE IF NOT EXISTS dividends (
                        id SERIAL PRIMARY KEY,
                        symbol VARCHAR(20) NOT NULL,
                        ex_date DATE NOT NULL,
                        pay_date DATE,
                        record_date DATE,
                        declaration_date DATE,
                        amount DECIMAL(12, 6) NOT NULL,
                        currency VARCHAR(10) DEFAULT 'USD',
                        frequency INTEGER,  -- Payments per year (1=annual, 4=quarterly, 12=monthly)
                        type VARCHAR(20) DEFAULT 'CD',  -- CD=Cash Dividend, SC=Special Cash, etc.
                        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                        updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                        UNIQUE(symbol, ex_date, type)
                    );

                    CREATE INDEX IF NOT EXISTS idx_dividends_symbol ON dividends(symbol);
                    CREATE INDEX IF NOT EXISTS idx_dividends_ex_date ON dividends(ex_date DESC);
                    CREATE INDEX IF NOT EXISTS idx_dividends_symbol_date ON dividends(symbol, ex_date DESC);
                """)
                self.conn.commit()
                logger.info("Dividends table ensured")
        except Exception as e:
            logger.error(f"Error creating dividends table: {e}")
            self.conn.rollback()

    def fetch_polygon_dividends(self, symbol: str) -> List[Dict]:
        """Fetch dividend history from Polygon API.

        Args:
            symbol: Stock ticker symbol

        Returns:
            List of dividend records
        """
        url = f"https://api.polygon.io/v3/reference/dividends"
        params = {
            'ticker': symbol,
            'limit': 100,  # Get last 100 dividends
            'apiKey': self.polygon_api_key
        }

        try:
            self.stats['api_calls'] += 1
            response = requests.get(url, params=params, timeout=10)
            response.raise_for_status()

            data = response.json()
            if data.get('status') == 'OK' and 'results' in data:
                return data['results']
            return []

        except requests.exceptions.RequestException as e:
            logger.debug(f"Failed to fetch dividends for {symbol}: {e}")
            self.stats['errors'] += 1
            return []

    def backfill_dividends(self, limit: Optional[int] = None) -> int:
        """Fetch and store dividend data from Polygon.

        Args:
            limit: Maximum number of tickers to process

        Returns:
            Number of dividend records inserted
        """
        logger.info("=== Starting Dividend Data Backfill ===")

        # Ensure table exists
        self.ensure_dividends_table()

        try:
            with self.conn.cursor(cursor_factory=RealDictCursor) as cursor:
                # Get active stock tickers
                query = """
                    SELECT DISTINCT t.symbol
                    FROM tickers t
                    LEFT JOIN dividends d ON t.symbol = d.symbol
                    WHERE t.asset_type IN ('stock', 'CS')
                      AND t.active = true
                      AND d.symbol IS NULL
                    ORDER BY t.market_cap DESC NULLS LAST
                """
                if limit:
                    query += f" LIMIT {limit}"

                cursor.execute(query)
                tickers = [row['symbol'] for row in cursor.fetchall()]
                logger.info(f"Found {len(tickers)} tickers to fetch dividends for")

                total_inserted = 0
                batch_dividends = []

                for i, symbol in enumerate(tickers):
                    if (i + 1) % 100 == 0:
                        logger.info(f"Progress: {i + 1}/{len(tickers)} - API calls: {self.stats['api_calls']}")

                    dividends = self.fetch_polygon_dividends(symbol)
                    self.stats['dividends_fetched'] += len(dividends)

                    for div in dividends:
                        batch_dividends.append((
                            symbol,
                            div.get('ex_dividend_date'),
                            div.get('pay_date'),
                            div.get('record_date'),
                            div.get('declaration_date'),
                            div.get('cash_amount', 0),
                            div.get('currency', 'USD'),
                            div.get('frequency'),
                            div.get('dividend_type', 'CD')
                        ))

                    # Batch insert every 1000 records
                    if len(batch_dividends) >= 1000:
                        total_inserted += self._insert_dividends_batch(cursor, batch_dividends)
                        batch_dividends = []

                    # Rate limiting - 5 requests per second
                    time.sleep(0.2)

                # Insert remaining
                if batch_dividends:
                    total_inserted += self._insert_dividends_batch(cursor, batch_dividends)

                self.stats['dividends_inserted'] = total_inserted
                logger.info(f"Inserted {total_inserted} dividend records")
                return total_inserted

        except Exception as e:
            logger.error(f"Error backfilling dividends: {e}")
            self.conn.rollback()
            return 0

    def _insert_dividends_batch(self, cursor, batch: List[tuple]) -> int:
        """Insert a batch of dividend records.

        Args:
            cursor: Database cursor
            batch: List of dividend tuples

        Returns:
            Number of records inserted
        """
        if not batch:
            return 0

        try:
            execute_batch(cursor, """
                INSERT INTO dividends (
                    symbol, ex_date, pay_date, record_date, declaration_date,
                    amount, currency, frequency, type
                ) VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s)
                ON CONFLICT (symbol, ex_date, type) DO UPDATE SET
                    pay_date = EXCLUDED.pay_date,
                    amount = EXCLUDED.amount,
                    updated_at = CURRENT_TIMESTAMP
            """, batch, page_size=500)

            self.conn.commit()
            return len(batch)

        except Exception as e:
            logger.error(f"Error inserting dividend batch: {e}")
            self.conn.rollback()
            return 0

    # =========================================================================
    # REVENUE GROWTH RECALCULATION
    # =========================================================================

    def recalculate_revenue_growth(self) -> int:
        """Recalculate revenue growth from financials data.

        This updates the fundamental_metrics_extended table with YoY revenue growth
        calculated directly from the financials table.

        Returns:
            Number of tickers updated
        """
        logger.info("=== Starting Revenue Growth Recalculation ===")

        try:
            with self.conn.cursor() as cursor:
                # Calculate YoY revenue growth from annual financials
                cursor.execute("""
                    WITH annual_revenue AS (
                        SELECT
                            ticker,
                            fiscal_year,
                            revenue,
                            LAG(revenue) OVER (PARTITION BY ticker ORDER BY fiscal_year) as prev_revenue
                        FROM financials
                        WHERE statement_type = '10-K'
                          AND fiscal_quarter IS NULL
                          AND revenue IS NOT NULL
                          AND revenue > 0
                    ),
                    growth_rates AS (
                        SELECT
                            ticker,
                            fiscal_year,
                            CASE
                                WHEN prev_revenue > 0
                                THEN ((revenue - prev_revenue) / prev_revenue * 100)
                                ELSE NULL
                            END as revenue_growth_yoy
                        FROM annual_revenue
                        WHERE fiscal_year = EXTRACT(YEAR FROM CURRENT_DATE) - 1
                    )
                    UPDATE fundamental_metrics_extended fme
                    SET
                        revenue_growth_yoy = gr.revenue_growth_yoy,
                        updated_at = CURRENT_TIMESTAMP
                    FROM growth_rates gr
                    WHERE fme.ticker = gr.ticker
                      AND (fme.revenue_growth_yoy IS NULL OR fme.revenue_growth_yoy != gr.revenue_growth_yoy)
                """)

                updated = cursor.rowcount
                self.conn.commit()

                logger.info(f"Updated {updated} tickers with revenue growth")
                return updated

        except Exception as e:
            logger.error(f"Error recalculating revenue growth: {e}")
            self.conn.rollback()
            return 0

    # =========================================================================
    # DIVIDEND YIELD CALCULATION
    # =========================================================================

    def calculate_dividend_yields(self) -> int:
        """Calculate dividend yield from dividends and price data.

        Returns:
            Number of tickers updated
        """
        logger.info("=== Starting Dividend Yield Calculation ===")

        try:
            with self.conn.cursor() as cursor:
                # Calculate TTM dividend yield
                cursor.execute("""
                    WITH ttm_dividends AS (
                        SELECT
                            symbol,
                            SUM(amount) as annual_dividend
                        FROM dividends
                        WHERE ex_date >= CURRENT_DATE - INTERVAL '1 year'
                          AND type = 'CD'
                        GROUP BY symbol
                    ),
                    current_prices AS (
                        SELECT DISTINCT ON (ticker)
                            ticker,
                            close as price
                        FROM stock_prices
                        ORDER BY ticker, time DESC
                    )
                    UPDATE fundamental_metrics_extended fme
                    SET
                        dividend_yield = CASE
                            WHEN cp.price > 0
                            THEN (td.annual_dividend / cp.price * 100)
                            ELSE NULL
                        END,
                        updated_at = CURRENT_TIMESTAMP
                    FROM ttm_dividends td
                    JOIN current_prices cp ON td.symbol = cp.ticker
                    WHERE fme.ticker = td.symbol
                """)

                updated = cursor.rowcount
                self.conn.commit()

                logger.info(f"Updated {updated} tickers with dividend yield")
                return updated

        except Exception as e:
            logger.error(f"Error calculating dividend yields: {e}")
            self.conn.rollback()
            return 0

    # =========================================================================
    # MARKET CAP BACKFILL
    # =========================================================================

    def backfill_market_cap(self, limit: Optional[int] = None) -> int:
        """Fetch and update market cap from Polygon ticker details API.

        Args:
            limit: Maximum number of tickers to process

        Returns:
            Number of tickers updated
        """
        logger.info("=== Starting Market Cap Backfill ===")

        try:
            with self.conn.cursor(cursor_factory=RealDictCursor) as cursor:
                # Get tickers missing market_cap
                # Exclude likely preferred shares (names with "Preferred", "Depositary", etc.)
                # and order by shorter symbols first (more likely to be common stock)
                query = """
                    SELECT symbol
                    FROM tickers
                    WHERE asset_type = 'CS'
                      AND active = true
                      AND (market_cap IS NULL OR market_cap = 0)
                      AND name NOT ILIKE '%Preferred%'
                      AND name NOT ILIKE '%Depositary%'
                      AND name NOT ILIKE '%Subordinated%'
                      AND name NOT ILIKE '%Notes due%'
                      AND name NOT ILIKE '%Warrant%'
                      AND name NOT ILIKE '%Unit%'
                      AND name NOT ILIKE '%Rights%'
                    ORDER BY LENGTH(symbol), symbol
                """
                if limit:
                    query += f" LIMIT {limit}"

                cursor.execute(query)
                tickers = [row['symbol'] for row in cursor.fetchall()]
                logger.info(f"Found {len(tickers)} tickers to update market cap")

                updated = 0
                batch_updates = []

                for i, symbol in enumerate(tickers):
                    if (i + 1) % 100 == 0:
                        logger.info(f"Progress: {i + 1}/{len(tickers)} - Updated: {updated}")

                    # Fetch from Polygon
                    url = f"https://api.polygon.io/v3/reference/tickers/{symbol}"
                    params = {'apiKey': self.polygon_api_key}

                    try:
                        self.stats['api_calls'] += 1
                        response = requests.get(url, params=params, timeout=10)

                        if response.status_code == 200:
                            data = response.json()
                            results = data.get('results', {})
                            market_cap = results.get('market_cap')

                            if market_cap and market_cap > 0:
                                batch_updates.append((market_cap, symbol))
                                updated += 1

                        # Rate limiting - 5 requests per second
                        time.sleep(0.2)

                    except requests.exceptions.RequestException as e:
                        logger.debug(f"Failed to fetch market cap for {symbol}: {e}")
                        self.stats['errors'] += 1

                    # Batch update every 100 records
                    if len(batch_updates) >= 100:
                        execute_batch(cursor, """
                            UPDATE tickers
                            SET market_cap = %s, updated_at = CURRENT_TIMESTAMP
                            WHERE symbol = %s
                        """, batch_updates, page_size=100)
                        self.conn.commit()
                        batch_updates = []

                # Final batch
                if batch_updates:
                    execute_batch(cursor, """
                        UPDATE tickers
                        SET market_cap = %s, updated_at = CURRENT_TIMESTAMP
                        WHERE symbol = %s
                    """, batch_updates, page_size=100)
                    self.conn.commit()

                logger.info(f"Updated {updated} tickers with market cap")
                self.stats['market_cap_updated'] = updated
                return updated

        except Exception as e:
            logger.error(f"Error backfilling market cap: {e}")
            self.conn.rollback()
            return 0

    # =========================================================================
    # MAIN EXECUTION
    # =========================================================================

    def run(self, backfill_sectors: bool = False, backfill_dividends: bool = False,
            dividend_limit: Optional[int] = None, all_backfills: bool = False,
            backfill_market_cap: bool = False, market_cap_limit: Optional[int] = None):
        """Run the backfill process.

        Args:
            backfill_sectors: Whether to backfill sector/industry data
            backfill_dividends: Whether to fetch dividend data
            dividend_limit: Limit for dividend fetch
            all_backfills: Run all backfills
            backfill_market_cap: Whether to fetch market cap data
            market_cap_limit: Limit for market cap fetch
        """
        start_time = datetime.now()
        logger.info("=" * 80)
        logger.info("Screener Data Backfill")
        logger.info("=" * 80)

        if not self.connect_db():
            sys.exit(1)

        try:
            if all_backfills or backfill_sectors:
                self.backfill_sectors()

            if all_backfills or backfill_market_cap:
                self.backfill_market_cap(limit=market_cap_limit)

            if all_backfills or backfill_dividends:
                self.backfill_dividends(limit=dividend_limit)
                self.calculate_dividend_yields()

            # Always recalculate revenue growth if we have new data
            if all_backfills:
                self.recalculate_revenue_growth()

            # Print summary
            duration = datetime.now() - start_time
            logger.info("=" * 80)
            logger.info("Backfill Complete")
            logger.info("=" * 80)
            logger.info(f"Duration: {duration}")
            logger.info(f"Sectors updated: {self.stats['sectors_updated']}")
            logger.info(f"Sectors skipped (no mapping): {self.stats['sectors_skipped']}")
            logger.info(f"Market cap updated: {self.stats.get('market_cap_updated', 0)}")
            logger.info(f"API calls made: {self.stats['api_calls']}")
            logger.info(f"Dividends fetched: {self.stats['dividends_fetched']}")
            logger.info(f"Dividends inserted: {self.stats['dividends_inserted']}")
            logger.info(f"Errors: {self.stats['errors']}")

        finally:
            if self.conn:
                self.conn.close()


def main():
    """Main entry point."""
    parser = argparse.ArgumentParser(
        description='Backfill missing screener data (sectors, dividends, revenue growth)',
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  # Backfill sectors from SIC codes
  python backfill_screener_data.py --sectors

  # Fetch dividend data (limited to 100 tickers for testing)
  python backfill_screener_data.py --dividends --limit 100

  # Run all backfills
  python backfill_screener_data.py --all
        """
    )

    parser.add_argument('--sectors', action='store_true',
                        help='Backfill sector/industry from SIC codes')
    parser.add_argument('--market-cap', action='store_true',
                        help='Fetch market cap from Polygon API')
    parser.add_argument('--dividends', action='store_true',
                        help='Fetch dividend data from Polygon API')
    parser.add_argument('--limit', type=int, default=None,
                        help='Limit number of tickers for API fetches')
    parser.add_argument('--all', action='store_true',
                        help='Run all backfills')

    args = parser.parse_args()

    if not any([args.sectors, args.market_cap, args.dividends, args.all]):
        parser.print_help()
        sys.exit(1)

    backfill = ScreenerDataBackfill()
    backfill.run(
        backfill_sectors=args.sectors,
        backfill_market_cap=args.market_cap,
        backfill_dividends=args.dividends,
        dividend_limit=args.limit,
        market_cap_limit=args.limit,
        all_backfills=args.all
    )


if __name__ == '__main__':
    main()
