#!/usr/bin/env python3
"""
Backfill sector and industry data from SEC EDGAR submissions API.

Fetches SIC codes from SEC EDGAR for tickers that have CIK but no
sector data, maps SIC codes to GICS sectors, and uses SEC's
sicDescription as the industry name.

No API key required — SEC EDGAR is free. Rate limited to 10 req/sec.

Usage:
    python backfill_sectors_from_sec.py              # Process all
    python backfill_sectors_from_sec.py --limit 100  # First 100
    python backfill_sectors_from_sec.py --dry-run    # Preview only
    python backfill_sectors_from_sec.py --ticker MSFT # Single ticker
"""

import argparse
import logging
import os
import sys
import time
from datetime import datetime
from typing import Dict, List, Optional, Tuple

import psycopg2
import requests
from requests.adapters import HTTPAdapter
from urllib3.util.retry import Retry

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s - %(levelname)s - %(message)s",
)
logger = logging.getLogger(__name__)

# ---------------------------------------------------------------------------
# SIC code -> GICS sector mappings (ported from scripts/backfill_sectors.go)
# ---------------------------------------------------------------------------

# 2-digit SIC prefix -> sector
SIC2_TO_SECTOR: Dict[str, str] = {
    # Agriculture, Forestry, Fishing (01-09)
    "01": "Consumer Defensive",
    "02": "Consumer Defensive",
    "07": "Consumer Defensive",
    "08": "Consumer Defensive",
    "09": "Consumer Defensive",
    # Mining (10-14)
    "10": "Energy",
    "12": "Energy",
    "13": "Energy",
    "14": "Basic Materials",
    # Construction (15-17)
    "15": "Industrials",
    "16": "Industrials",
    "17": "Industrials",
    # Manufacturing - Food, Tobacco, Textiles (20-23)
    "20": "Consumer Defensive",
    "21": "Consumer Defensive",
    "22": "Consumer Cyclical",
    "23": "Consumer Cyclical",
    # Manufacturing - Wood, Paper, Printing (24-27)
    "24": "Basic Materials",
    "25": "Consumer Cyclical",
    "26": "Basic Materials",
    "27": "Communication Services",
    # Manufacturing - Chemicals, Petroleum (28-29)
    "28": "Healthcare",
    "29": "Energy",
    # Manufacturing - Rubber, Plastics, Leather (30-31)
    "30": "Basic Materials",
    "31": "Consumer Cyclical",
    # Manufacturing - Stone, Clay, Glass, Metals (32-34)
    "32": "Basic Materials",
    "33": "Basic Materials",
    "34": "Industrials",
    # Manufacturing - Machinery, Electronics (35-36)
    "35": "Technology",
    "36": "Technology",
    # Manufacturing - Transportation Equipment (37)
    "37": "Consumer Cyclical",
    # Manufacturing - Instruments, Misc (38-39)
    "38": "Healthcare",
    "39": "Consumer Cyclical",
    # Transportation (40-47)
    "40": "Industrials",
    "41": "Industrials",
    "42": "Industrials",
    "43": "Industrials",
    "44": "Industrials",
    "45": "Industrials",
    "46": "Energy",
    "47": "Industrials",
    # Communications (48)
    "48": "Communication Services",
    # Utilities (49)
    "49": "Utilities",
    # Wholesale Trade (50-51)
    "50": "Consumer Cyclical",
    "51": "Consumer Cyclical",
    # Retail Trade (52-59)
    "52": "Consumer Cyclical",
    "53": "Consumer Cyclical",
    "54": "Consumer Defensive",
    "55": "Consumer Cyclical",
    "56": "Consumer Cyclical",
    "57": "Consumer Cyclical",
    "58": "Consumer Cyclical",
    "59": "Consumer Cyclical",
    # Finance, Insurance, Real Estate (60-67)
    "60": "Financial Services",
    "61": "Financial Services",
    "62": "Financial Services",
    "63": "Financial Services",
    "64": "Financial Services",
    "65": "Real Estate",
    "67": "Financial Services",
    # Services (70-89)
    "70": "Consumer Cyclical",
    "72": "Consumer Cyclical",
    "73": "Technology",
    "75": "Consumer Cyclical",
    "76": "Consumer Cyclical",
    "78": "Communication Services",
    "79": "Communication Services",
    "80": "Healthcare",
    "81": "Industrials",
    "82": "Consumer Cyclical",
    "83": "Consumer Defensive",
    "84": "Consumer Cyclical",
    "86": "Consumer Cyclical",
    "87": "Technology",
    "89": "Industrials",
    # Public Administration (91-99)
    "91": "Industrials",
    "92": "Industrials",
    "93": "Industrials",
    "94": "Industrials",
    "95": "Industrials",
    "96": "Industrials",
    "97": "Industrials",
    "99": "Industrials",
}

# 4-digit SIC code -> sector (more specific overrides)
SIC4_TO_SECTOR: Dict[str, str] = {
    # Technology
    "3571": "Technology",
    "3572": "Technology",
    "3575": "Technology",
    "3576": "Technology",
    "3577": "Technology",
    "3578": "Technology",
    "3579": "Technology",
    "3661": "Technology",
    "3663": "Technology",
    "3669": "Technology",
    "3674": "Technology",
    "3825": "Technology",
    "7370": "Technology",
    "7371": "Technology",
    "7372": "Technology",
    "7373": "Technology",
    "7374": "Technology",
    "7375": "Technology",
    "7376": "Technology",
    "7377": "Technology",
    "7378": "Technology",
    "7379": "Technology",
    # Healthcare
    "2834": "Healthcare",
    "2835": "Healthcare",
    "2836": "Healthcare",
    "3826": "Healthcare",
    "3841": "Healthcare",
    "3842": "Healthcare",
    "3843": "Healthcare",
    "3844": "Healthcare",
    "3845": "Healthcare",
    "8011": "Healthcare",
    "8021": "Healthcare",
    "8031": "Healthcare",
    "8041": "Healthcare",
    "8042": "Healthcare",
    "8049": "Healthcare",
    "8051": "Healthcare",
    "8052": "Healthcare",
    "8059": "Healthcare",
    "8062": "Healthcare",
    "8063": "Healthcare",
    "8069": "Healthcare",
    "8071": "Healthcare",
    "8072": "Healthcare",
    "8082": "Healthcare",
    "8092": "Healthcare",
    "8093": "Healthcare",
    "8099": "Healthcare",
    # Communication Services
    "4812": "Communication Services",
    "4813": "Communication Services",
    "4822": "Communication Services",
    "4832": "Communication Services",
    "4833": "Communication Services",
    "4841": "Communication Services",
    "7812": "Communication Services",
    "7819": "Communication Services",
    "7822": "Communication Services",
    "7829": "Communication Services",
    "7832": "Communication Services",
    "7841": "Communication Services",
    "7941": "Communication Services",
    # Consumer Cyclical (specific overrides)
    "5961": "Consumer Cyclical",
    "3711": "Consumer Cyclical",
    "3713": "Consumer Cyclical",
    "3714": "Consumer Cyclical",
    "3715": "Consumer Cyclical",
    "3716": "Consumer Cyclical",
    "5511": "Consumer Cyclical",
    "5521": "Consumer Cyclical",
    "5531": "Consumer Cyclical",
}


def get_sector_from_sic(sic_code: Optional[str]) -> str:
    """Map a SIC code to a GICS sector name.

    Tries 4-digit match first, then falls back to 2-digit prefix.
    Returns empty string if no mapping found.
    """
    if not sic_code:
        return ""
    sic_code = sic_code.strip()
    if sic_code in SIC4_TO_SECTOR:
        return SIC4_TO_SECTOR[sic_code]
    if len(sic_code) >= 2:
        prefix = sic_code[:2]
        if prefix in SIC2_TO_SECTOR:
            return SIC2_TO_SECTOR[prefix]
    return ""


def format_industry(sic_description: Optional[str]) -> str:
    """Convert SEC sicDescription to a title-cased industry name.

    SEC descriptions are ALL CAPS and often prefixed with a category
    (e.g. "SERVICES-COMPUTER PROGRAMMING, DATA PROCESSING").
    We strip the prefix and title-case the remainder.
    """
    if not sic_description:
        return ""
    desc = sic_description.strip()
    if not desc:
        return ""

    # Strip category prefix before the first hyphen
    if "-" in desc:
        desc = desc.split("-", 1)[1].strip()

    # Title-case
    industry = desc.title()

    # Lowercase small words (except at start of string)
    small_words = {
        "And", "Or", "Of", "The", "In", "For",
        "By", "To", "On", "At", "An", "A",
    }
    words = industry.split()
    for i, word in enumerate(words):
        if i > 0 and word in small_words:
            words[i] = word.lower()
    industry = " ".join(words)

    # Fix common abbreviations that title() mangles
    industry = industry.replace("'S", "'s")

    return industry


# ---------------------------------------------------------------------------
# Main backfill class
# ---------------------------------------------------------------------------


class SECSectorBackfiller:
    """Fetch SIC codes from SEC EDGAR and backfill sector/industry."""

    SEC_BASE_URL = "https://data.sec.gov"
    USER_AGENT = "InvestorCenter.ai admin@investorcenter.ai"
    MIN_REQUEST_INTERVAL = 0.1  # 10 requests/second

    def __init__(
        self,
        dry_run: bool = False,
        batch_size: int = 100,
        skip_refresh: bool = False,
    ):
        self.dry_run = dry_run
        self.batch_size = batch_size
        self.skip_refresh = skip_refresh

        self.db_config = {
            "host": os.environ.get("DB_HOST", "localhost"),
            "port": int(os.environ.get("DB_PORT", 5432)),
            "dbname": os.environ.get("DB_NAME", "investorcenter_db"),
            "user": os.environ.get("DB_USER"),
            "password": os.environ.get("DB_PASSWORD"),
        }

        self.conn: Optional[psycopg2.extensions.connection] = None
        self.session: Optional[requests.Session] = None
        self.last_request_time = 0.0

        self.stats = {
            "tickers_processed": 0,
            "sectors_updated": 0,
            "api_calls": 0,
            "errors": 0,
            "no_sic_in_sec": 0,
            "no_sector_mapping": 0,
            "sec_not_found": 0,
        }

    def connect_db(self) -> bool:
        """Connect to PostgreSQL."""
        try:
            self.conn = psycopg2.connect(**self.db_config)
            logger.info("Connected to database: %s", self.db_config["dbname"])
            return True
        except Exception as e:
            logger.error("Database connection failed: %s", e)
            return False

    def create_session(self) -> None:
        """Create requests session with retry adapter."""
        self.session = requests.Session()
        self.session.headers["User-Agent"] = self.USER_AGENT
        self.session.headers["Accept"] = "application/json"

        retry = Retry(
            total=3,
            backoff_factor=1,
            status_forcelist=[429, 500, 502, 503, 504],
        )
        adapter = HTTPAdapter(max_retries=retry)
        self.session.mount("https://", adapter)

    def rate_limit(self) -> None:
        """Enforce SEC fair-access rate limit."""
        now = time.time()
        elapsed = now - self.last_request_time
        if elapsed < self.MIN_REQUEST_INTERVAL:
            time.sleep(self.MIN_REQUEST_INTERVAL - elapsed)
        self.last_request_time = time.time()

    def fetch_submissions(self, cik: str) -> Optional[Dict]:
        """Fetch company metadata from SEC EDGAR submissions API.

        Returns dict with 'sic' and 'sicDescription' keys, or None.
        """
        assert self.session is not None
        try:
            cik_int = int(cik)
        except (ValueError, TypeError):
            return None

        url = f"{self.SEC_BASE_URL}/submissions/CIK{cik_int:010d}.json"

        try:
            self.rate_limit()
            self.stats["api_calls"] += 1
            resp = self.session.get(url, timeout=10)

            if resp.status_code == 404:
                self.stats["sec_not_found"] += 1
                return None

            resp.raise_for_status()
            return resp.json()
        except requests.exceptions.RequestException as e:
            logger.debug("Failed to fetch CIK %s: %s", cik, e)
            self.stats["errors"] += 1
            return None

    def get_tickers_to_process(
        self,
        limit: Optional[int] = None,
        ticker: Optional[str] = None,
    ) -> List[Tuple[str, str]]:
        """Get tickers with CIK but no sector data.

        Returns list of (symbol, cik) tuples.
        """
        assert self.conn is not None
        if ticker:
            with self.conn.cursor() as cur:
                cur.execute(
                    "SELECT symbol, cik FROM tickers "
                    "WHERE symbol = %s AND cik IS NOT NULL",
                    (ticker,),
                )
                row = cur.fetchone()
                return [(row[0], row[1])] if row else []

        query = """
            SELECT symbol, cik FROM tickers
            WHERE asset_type = 'CS'
              AND active = true
              AND cik IS NOT NULL AND cik != ''
              AND (sector IS NULL OR sector = '')
            ORDER BY market_cap DESC NULLS LAST
        """
        if limit:
            query += f" LIMIT {limit}"

        with self.conn.cursor() as cur:
            cur.execute(query)
            return cur.fetchall()

    def batch_update_tickers(
        self, updates: List[Tuple[str, str, str, str, str]]
    ) -> None:
        """Write a batch of updates to the database.

        Each tuple: (sector, industry, sic_code, sic_description, symbol)
        """
        if not updates or self.dry_run:
            return
        assert self.conn is not None

        with self.conn.cursor() as cur:
            cur.executemany(
                """
                UPDATE tickers
                SET sector = %s,
                    industry = %s,
                    sic_code = COALESCE(sic_code, %s),
                    sic_description = COALESCE(sic_description, %s),
                    updated_at = CURRENT_TIMESTAMP
                WHERE symbol = %s
                  AND (sector IS NULL OR sector = '')
                """,
                updates,
            )
        self.conn.commit()

    def refresh_screener_data(self) -> None:
        """Refresh the screener_data materialized view."""
        if self.skip_refresh or self.dry_run:
            logger.info("Skipping materialized view refresh")
            return
        assert self.conn is not None

        logger.info("Refreshing screener_data materialized view...")
        with self.conn.cursor() as cur:
            cur.execute(
                "REFRESH MATERIALIZED VIEW CONCURRENTLY screener_data"
            )
        self.conn.commit()
        logger.info("Materialized view refreshed successfully")

    def run(
        self,
        limit: Optional[int] = None,
        ticker: Optional[str] = None,
    ) -> None:
        """Run the sector backfill process."""
        start_time = datetime.now()
        logger.info("=" * 60)
        logger.info("SEC EDGAR Sector Backfiller")
        if self.dry_run:
            logger.info("DRY RUN — no database changes will be made")
        logger.info("=" * 60)

        if not self.connect_db():
            sys.exit(1)
        self.create_session()

        try:
            tickers = self.get_tickers_to_process(
                limit=limit, ticker=ticker
            )
            logger.info("Found %d tickers to process", len(tickers))

            pending_updates: List[Tuple[str, str, str, str, str]] = []

            for i, (symbol, cik) in enumerate(tickers):
                self.stats["tickers_processed"] += 1

                data = self.fetch_submissions(cik)
                if data is None:
                    continue

                sic_code = data.get("sic", "")
                sic_desc = data.get("sicDescription", "")

                if not sic_code:
                    self.stats["no_sic_in_sec"] += 1
                    logger.debug(
                        "[%d/%d] %s: no SIC in SEC response",
                        i + 1, len(tickers), symbol,
                    )
                    continue

                sector = get_sector_from_sic(sic_code)
                if not sector:
                    self.stats["no_sector_mapping"] += 1
                    logger.warning(
                        "[%d/%d] %s: unmapped SIC %s (%s)",
                        i + 1, len(tickers), symbol, sic_code, sic_desc,
                    )
                    continue

                industry = format_industry(sic_desc)
                self.stats["sectors_updated"] += 1

                if self.dry_run:
                    logger.info(
                        "[%d/%d] %s: SIC %s -> %s / %s",
                        i + 1, len(tickers), symbol,
                        sic_code, sector, industry,
                    )
                else:
                    pending_updates.append(
                        (sector, industry, sic_code, sic_desc, symbol)
                    )

                # Commit batch
                if len(pending_updates) >= self.batch_size:
                    self.batch_update_tickers(pending_updates)
                    logger.info(
                        "Committed batch (%d updates)", len(pending_updates)
                    )
                    pending_updates = []

                # Progress log
                if (i + 1) % 200 == 0:
                    logger.info(
                        "Progress: %d/%d — updated: %d, "
                        "no SIC: %d, unmapped: %d, not found: %d",
                        i + 1,
                        len(tickers),
                        self.stats["sectors_updated"],
                        self.stats["no_sic_in_sec"],
                        self.stats["no_sector_mapping"],
                        self.stats["sec_not_found"],
                    )

            # Commit remaining
            if pending_updates:
                self.batch_update_tickers(pending_updates)
                logger.info(
                    "Committed final batch (%d updates)",
                    len(pending_updates),
                )

            # Refresh materialized view
            if self.stats["sectors_updated"] > 0:
                self.refresh_screener_data()

            # Summary
            duration = datetime.now() - start_time
            logger.info("=" * 60)
            logger.info("Backfill Complete")
            logger.info("=" * 60)
            logger.info("Duration:           %s", duration)
            logger.info(
                "Tickers processed:  %d", self.stats["tickers_processed"]
            )
            logger.info("API calls:          %d", self.stats["api_calls"])
            logger.info(
                "Sectors updated:    %d", self.stats["sectors_updated"]
            )
            logger.info(
                "No SIC in SEC:      %d", self.stats["no_sic_in_sec"]
            )
            logger.info(
                "No sector mapping:  %d", self.stats["no_sector_mapping"]
            )
            logger.info(
                "SEC 404 (CIK gone): %d", self.stats["sec_not_found"]
            )
            logger.info("Errors:             %d", self.stats["errors"])

            # Coverage report
            if not self.dry_run and self.conn:
                with self.conn.cursor() as cur:
                    cur.execute(
                        "SELECT COUNT(*) FROM tickers "
                        "WHERE asset_type = 'CS' AND active = true "
                        "AND (sector IS NULL OR sector = '')"
                    )
                    remaining = cur.fetchone()[0]
                    cur.execute(
                        "SELECT COUNT(*) FROM tickers "
                        "WHERE asset_type = 'CS' AND active = true"
                    )
                    total = cur.fetchone()[0]
                logger.info(
                    "Coverage: %d/%d tickers now have sector data "
                    "(%d still missing)",
                    total - remaining,
                    total,
                    remaining,
                )

        finally:
            if self.conn:
                self.conn.close()


def main() -> None:
    parser = argparse.ArgumentParser(
        description="Backfill sector/industry from SEC EDGAR"
    )
    parser.add_argument(
        "--limit", type=int, help="Max tickers to process"
    )
    parser.add_argument(
        "--ticker", type=str, help="Process a single ticker"
    )
    parser.add_argument(
        "--dry-run",
        action="store_true",
        help="Preview changes without writing to DB",
    )
    parser.add_argument(
        "--skip-refresh",
        action="store_true",
        help="Skip materialized view refresh",
    )
    parser.add_argument(
        "--batch-size",
        type=int,
        default=100,
        help="DB commit batch size (default: 100)",
    )
    args = parser.parse_args()

    backfiller = SECSectorBackfiller(
        dry_run=args.dry_run,
        batch_size=args.batch_size,
        skip_refresh=args.skip_refresh,
    )
    backfiller.run(limit=args.limit, ticker=args.ticker)


if __name__ == "__main__":
    main()
