#!/usr/bin/env python3
"""
Diagnostic script to analyze XBRL revenue fact names used by companies.

This script fetches SEC data for a sample of companies missing revenue
and identifies what XBRL fact names they use for revenue.
"""

import logging
import os
import sys
from collections import Counter
from typing import Dict, List, Optional

import psycopg2
from psycopg2.extras import RealDictCursor

# Import SEC client directly
import sys
sys.path.insert(0, '/app/pipelines/utils')
from sec_client import SECClient

logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)


def get_db_connection():
    """Connect to PostgreSQL database."""
    return psycopg2.connect(
        host=os.getenv('DB_HOST', 'postgres-simple-service'),
        port=int(os.getenv('DB_PORT', '5432')),
        user=os.getenv('DB_USER', 'postgres'),
        password=os.getenv('DB_PASSWORD'),
        database=os.getenv('DB_NAME', 'investorcenter_db'),
        sslmode=os.getenv('DB_SSLMODE', 'disable')
    )


def get_stocks_missing_revenue(limit: int = 50) -> List[Dict]:
    """Get sample of stocks missing revenue in TTM financials."""
    conn = get_db_connection()
    cursor = conn.cursor(cursor_factory=RealDictCursor)

    # Get regular stocks (not warrants/units) missing revenue
    cursor.execute("""
        SELECT s.ticker, s.cik, s.name
        FROM stocks s
        JOIN ttm_financials t ON s.ticker = t.ticker
        WHERE t.revenue IS NULL
          AND s.cik IS NOT NULL
          AND s.ticker NOT LIKE '%%.%%'
          AND s.ticker NOT LIKE '%%/W%%'
          AND s.ticker NOT LIKE '%%/U%%'
          AND s.ticker NOT LIKE '%%/R%%'
        ORDER BY RANDOM()
        LIMIT %s
    """, (limit,))

    stocks = cursor.fetchall()
    cursor.close()
    conn.close()

    return stocks


def find_revenue_facts_for_cik(sec_client: SECClient, cik: str) -> Optional[List[str]]:
    """Find all XBRL fact names containing 'Revenue' for a given CIK."""
    try:
        company_facts = sec_client.fetch_company_facts(cik)

        if not company_facts or 'facts' not in company_facts:
            return None

        us_gaap = company_facts.get('facts', {}).get('us-gaap', {})
        if not us_gaap:
            return None

        # Find all facts containing 'Revenue' or 'Sales'
        revenue_facts = [
            name for name in us_gaap.keys()
            if 'Revenue' in name or 'Sales' in name
        ]

        return revenue_facts

    except Exception as e:
        logger.error(f"Error fetching facts for CIK {cik}: {e}")
        return None


def main():
    """Main diagnostic function."""
    logger.info("Starting revenue fact diagnosis...")

    # Get sample of stocks missing revenue
    logger.info("Fetching stocks missing revenue from database...")
    stocks = get_stocks_missing_revenue(limit=50)
    logger.info(f"Found {len(stocks)} stocks missing revenue")

    if not stocks:
        logger.info("No stocks missing revenue found")
        return

    # Current fact mappings
    current_mappings = [
        'Revenues',
        'RevenueFromContractWithCustomerExcludingAssessedTax',
        'SalesRevenueNet',
        'RevenueFromContractWithCustomer'
    ]

    logger.info(f"\nCurrent revenue fact mappings: {current_mappings}\n")

    # Analyze each stock
    sec_client = SECClient()
    all_revenue_facts = Counter()
    missing_count = 0
    analyzed_count = 0

    for stock in stocks:
        ticker = stock['ticker']
        cik = stock['cik']
        name = stock['name']

        logger.info(f"Analyzing {ticker} ({name}, CIK: {cik})...")

        revenue_facts = find_revenue_facts_for_cik(sec_client, cik)

        if revenue_facts is None:
            logger.warning(f"  ✗ No SEC data found for {ticker}")
            missing_count += 1
            continue

        if not revenue_facts:
            logger.warning(f"  ✗ No revenue facts found for {ticker}")
            missing_count += 1
            continue

        analyzed_count += 1

        # Count this company's revenue facts
        for fact in revenue_facts:
            all_revenue_facts[fact] += 1

        # Check if any of our mapped facts exist
        found_mapped = [f for f in revenue_facts if f in current_mappings]
        found_unmapped = [f for f in revenue_facts if f not in current_mappings]

        if found_mapped:
            logger.info(f"  ✓ Found mapped facts: {found_mapped}")

        if found_unmapped:
            logger.warning(f"  ! Found unmapped facts: {found_unmapped}")

    logger.info(f"\n{'='*80}")
    logger.info("SUMMARY")
    logger.info(f"{'='*80}")
    logger.info(f"Total stocks analyzed: {analyzed_count}")
    logger.info(f"Stocks with no SEC data: {missing_count}")
    logger.info(f"\nMost common revenue fact names found:")
    logger.info(f"{'-'*80}")

    for fact, count in all_revenue_facts.most_common(30):
        in_current = "✓" if fact in current_mappings else " "
        logger.info(f"  [{in_current}] {fact}: {count} companies")

    # Find facts not in current mappings
    unmapped_facts = [f for f in all_revenue_facts if f not in current_mappings]

    if unmapped_facts:
        logger.info(f"\n{'='*80}")
        logger.info("RECOMMENDED ADDITIONS TO FACT_MAPPINGS")
        logger.info(f"{'='*80}")
        logger.info("Consider adding these fact names to revenue mapping:")

        for fact in sorted(unmapped_facts, key=lambda x: all_revenue_facts[x], reverse=True)[:10]:
            count = all_revenue_facts[fact]
            logger.info(f"  - '{fact}'  ({count} companies)")

    sec_client.close()


if __name__ == '__main__':
    try:
        main()
    except Exception as e:
        logger.exception(f"Fatal error: {e}")
        sys.exit(1)
