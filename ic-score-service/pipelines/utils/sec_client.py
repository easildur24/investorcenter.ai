"""SEC EDGAR API client for fetching financial data.

This module provides a client for accessing SEC EDGAR Company Facts API
to retrieve financial statement data in XBRL format.

Rate Limit: 10 requests/second as per SEC fair access policy.
User-Agent: Required by SEC (must include company name and email).
"""

import asyncio
import logging
import time
from datetime import datetime
from decimal import Decimal
from typing import Any, Dict, List, Optional
from urllib.parse import quote

import aiohttp
import requests
from requests.adapters import HTTPAdapter
from urllib3.util.retry import Retry

logger = logging.getLogger(__name__)


class SECClient:
    """Client for SEC EDGAR API with rate limiting and retry logic."""

    BASE_URL = "https://data.sec.gov"
    COMPANY_FACTS_URL = "{base}/api/xbrl/companyfacts/CIK{cik:010d}.json"

    # SEC requires User-Agent header with company info
    USER_AGENT = "InvestorCenter.ai admin@investorcenter.ai"

    # Rate limiting: 10 requests/second max
    REQUESTS_PER_SECOND = 10
    MIN_REQUEST_INTERVAL = 1.0 / REQUESTS_PER_SECOND

    # XBRL fact mappings (US-GAAP taxonomy)
    FACT_MAPPINGS = {
        # Income Statement
        'revenue': ['Revenues', 'RevenueFromContractWithCustomerExcludingAssessedTax',
                   'SalesRevenueNet', 'RevenueFromContractWithCustomer'],
        'cost_of_revenue': ['CostOfRevenue', 'CostOfGoodsAndServicesSold', 'CostOfGoodsSold'],
        'gross_profit': ['GrossProfit'],
        'operating_expenses': ['OperatingExpenses', 'CostsAndExpenses'],
        'operating_income': ['OperatingIncomeLoss', 'OperatingIncome'],
        'net_income': ['NetIncomeLoss', 'ProfitLoss', 'NetIncomeLossAvailableToCommonStockholdersBasic'],
        'eps_basic': ['EarningsPerShareBasic'],
        'eps_diluted': ['EarningsPerShareDiluted'],
        'shares_outstanding': ['CommonStockSharesOutstanding', 'WeightedAverageNumberOfSharesOutstandingBasic'],

        # Balance Sheet
        'total_assets': ['Assets'],
        'total_liabilities': ['Liabilities'],
        'shareholders_equity': ['StockholdersEquity', 'StockholdersEquityIncludingPortionAttributableToNoncontrollingInterest'],
        'cash_and_equivalents': ['CashAndCashEquivalentsAtCarryingValue', 'Cash'],
        'short_term_debt': ['ShortTermBorrowings', 'DebtCurrent'],
        'long_term_debt': ['LongTermDebt', 'LongTermDebtNoncurrent'],

        # Cash Flow
        'operating_cash_flow': ['NetCashProvidedByUsedInOperatingActivities'],
        'investing_cash_flow': ['NetCashProvidedByUsedInInvestingActivities'],
        'financing_cash_flow': ['NetCashProvidedByUsedInFinancingActivities'],
        'capex': ['PaymentsToAcquirePropertyPlantAndEquipment', 'CapitalExpenditures'],
    }

    def __init__(self, user_agent: Optional[str] = None):
        """Initialize SEC client.

        Args:
            user_agent: Custom User-Agent string. Defaults to InvestorCenter.ai.
        """
        self.user_agent = user_agent or self.USER_AGENT
        self.last_request_time = 0.0
        self.session = self._create_session()

    def _create_session(self) -> requests.Session:
        """Create requests session with retry logic."""
        session = requests.Session()

        # Configure retry strategy
        retry_strategy = Retry(
            total=3,
            backoff_factor=1,
            status_forcelist=[429, 500, 502, 503, 504],
            allowed_methods=["GET"]
        )

        adapter = HTTPAdapter(max_retries=retry_strategy)
        session.mount("http://", adapter)
        session.mount("https://", adapter)

        # Set headers
        session.headers.update({
            'User-Agent': self.user_agent,
            'Accept': 'application/json',
        })

        return session

    def _rate_limit(self):
        """Implement rate limiting to respect SEC's 10 req/sec limit."""
        current_time = time.time()
        elapsed = current_time - self.last_request_time

        if elapsed < self.MIN_REQUEST_INTERVAL:
            sleep_time = self.MIN_REQUEST_INTERVAL - elapsed
            time.sleep(sleep_time)

        self.last_request_time = time.time()

    def fetch_company_facts(self, cik: str) -> Optional[Dict[str, Any]]:
        """Fetch company facts from SEC EDGAR API.

        Args:
            cik: Company CIK number (can be string or int).

        Returns:
            Dict containing company facts, or None if request fails.
        """
        # Convert CIK to integer for formatting
        try:
            cik_int = int(str(cik).lstrip('0') or '0')
        except ValueError:
            logger.error(f"Invalid CIK format: {cik}")
            return None

        url = self.COMPANY_FACTS_URL.format(base=self.BASE_URL, cik=cik_int)

        self._rate_limit()

        try:
            response = self.session.get(url, timeout=30)
            response.raise_for_status()
            return response.json()
        except requests.exceptions.HTTPError as e:
            if e.response.status_code == 404:
                logger.warning(f"No company facts found for CIK {cik_int}")
            else:
                logger.error(f"HTTP error fetching CIK {cik_int}: {e}")
            return None
        except requests.exceptions.RequestException as e:
            logger.error(f"Request error fetching CIK {cik_int}: {e}")
            return None
        except Exception as e:
            logger.exception(f"Unexpected error fetching CIK {cik_int}: {e}")
            return None

    def parse_financial_data(self, company_facts: Dict[str, Any],
                           num_periods: int = 20) -> List[Dict[str, Any]]:
        """Parse company facts into structured financial data.

        Args:
            company_facts: Raw company facts from SEC API.
            num_periods: Number of most recent periods to return.

        Returns:
            List of financial data dictionaries, sorted by date (newest first).
        """
        if not company_facts or 'facts' not in company_facts:
            return []

        # Extract US-GAAP facts
        us_gaap = company_facts.get('facts', {}).get('us-gaap', {})
        if not us_gaap:
            logger.warning("No US-GAAP facts found")
            return []

        # Build period-based data structure
        periods_data = {}

        for field, fact_names in self.FACT_MAPPINGS.items():
            for fact_name in fact_names:
                if fact_name in us_gaap:
                    fact = us_gaap[fact_name]
                    units = fact.get('units', {})

                    # Try USD first, then shares
                    for unit_type in ['USD', 'shares', 'pure']:
                        if unit_type in units:
                            for datapoint in units[unit_type]:
                                # Only use filed forms (10-Q, 10-K)
                                form = datapoint.get('form')
                                if form not in ['10-Q', '10-K']:
                                    continue

                                period_end = datapoint.get('end')
                                filed = datapoint.get('filed')
                                fy = datapoint.get('fy')
                                fp = datapoint.get('fp')
                                value = datapoint.get('val')

                                if not all([period_end, filed, fy, value is not None]):
                                    continue

                                # Create unique period key
                                period_key = (period_end, fy, fp or 'FY')

                                if period_key not in periods_data:
                                    periods_data[period_key] = {
                                        'period_end_date': period_end,
                                        'filing_date': filed,
                                        'fiscal_year': fy,
                                        'fiscal_quarter': self._parse_fiscal_period(fp),
                                        'statement_type': form,
                                    }

                                # Store value (use first found value for each field)
                                if field not in periods_data[period_key]:
                                    periods_data[period_key][field] = value

                            break  # Found data for this fact, move to next field

        # Convert to list and sort by date
        financials = list(periods_data.values())
        financials.sort(key=lambda x: x['period_end_date'], reverse=True)

        # Calculate derived metrics
        for financial in financials[:num_periods]:
            self._calculate_metrics(financial)

        return financials[:num_periods]

    def _parse_fiscal_period(self, fp: Optional[str]) -> Optional[int]:
        """Parse fiscal period string to quarter number.

        Args:
            fp: Fiscal period (Q1, Q2, Q3, Q4, FY, etc.)

        Returns:
            Quarter number (1-4) or None for annual.
        """
        if not fp:
            return None

        fp = fp.upper()
        if fp == 'FY':
            return None
        elif fp in ['Q1', 'Q1I']:
            return 1
        elif fp in ['Q2', 'Q2I']:
            return 2
        elif fp in ['Q3', 'Q3I']:
            return 3
        elif fp in ['Q4', 'Q4I']:
            return 4

        return None

    def _calculate_metrics(self, financial: Dict[str, Any]) -> None:
        """Calculate financial ratios and metrics in-place.

        Args:
            financial: Dictionary of financial data to augment.
        """
        # Helper to safely get decimal value
        def get_decimal(key: str) -> Optional[Decimal]:
            val = financial.get(key)
            if val is None:
                return None
            try:
                return Decimal(str(val))
            except:
                return None

        revenue = get_decimal('revenue')
        cost_of_revenue = get_decimal('cost_of_revenue')
        gross_profit = get_decimal('gross_profit')
        operating_income = get_decimal('operating_income')
        net_income = get_decimal('net_income')
        total_assets = get_decimal('total_assets')
        total_liabilities = get_decimal('total_liabilities')
        shareholders_equity = get_decimal('shareholders_equity')
        short_term_debt = get_decimal('short_term_debt') or Decimal('0')
        long_term_debt = get_decimal('long_term_debt') or Decimal('0')
        operating_cash_flow = get_decimal('operating_cash_flow')
        capex = get_decimal('capex')

        # Calculate missing base metrics
        if gross_profit is None and revenue and cost_of_revenue:
            financial['gross_profit'] = float(revenue - cost_of_revenue)
            gross_profit = revenue - cost_of_revenue

        # Free cash flow
        if operating_cash_flow and capex:
            financial['free_cash_flow'] = float(operating_cash_flow - abs(capex))

        # Margins
        if revenue and revenue > 0:
            if gross_profit:
                financial['gross_margin'] = float((gross_profit / revenue) * 100)
            if operating_income:
                financial['operating_margin'] = float((operating_income / revenue) * 100)
            if net_income:
                financial['net_margin'] = float((net_income / revenue) * 100)

        # Leverage ratios
        if shareholders_equity and shareholders_equity > 0:
            total_debt = short_term_debt + long_term_debt
            financial['debt_to_equity'] = float(total_debt / shareholders_equity)

        # Profitability ratios
        if total_assets and total_assets > 0:
            if net_income:
                financial['roa'] = float((net_income / total_assets) * 100)

        if shareholders_equity and shareholders_equity > 0:
            if net_income:
                financial['roe'] = float((net_income / shareholders_equity) * 100)

        # Note: P/E, P/B, P/S ratios require current market price
        # These will be calculated later when we have price data

    def get_financials_for_ticker(self, cik: str, num_periods: int = 20) -> List[Dict[str, Any]]:
        """Fetch and parse financial data for a company.

        Args:
            cik: Company CIK number.
            num_periods: Number of periods to return.

        Returns:
            List of financial data dictionaries.
        """
        logger.info(f"Fetching company facts for CIK {cik}")

        company_facts = self.fetch_company_facts(cik)
        if not company_facts:
            return []

        financials = self.parse_financial_data(company_facts, num_periods)
        logger.info(f"Parsed {len(financials)} periods for CIK {cik}")

        return financials

    def close(self):
        """Close the HTTP session."""
        if self.session:
            self.session.close()


# Async version for concurrent requests
class AsyncSECClient:
    """Async client for SEC EDGAR API with rate limiting."""

    BASE_URL = "https://data.sec.gov"
    COMPANY_FACTS_URL = "{base}/api/xbrl/companyfacts/CIK{cik:010d}.json"
    USER_AGENT = "InvestorCenter.ai admin@investorcenter.ai"
    REQUESTS_PER_SECOND = 10
    MIN_REQUEST_INTERVAL = 1.0 / REQUESTS_PER_SECOND

    def __init__(self, user_agent: Optional[str] = None):
        """Initialize async SEC client."""
        self.user_agent = user_agent or self.USER_AGENT
        self.last_request_time = 0.0
        self.semaphore = asyncio.Semaphore(self.REQUESTS_PER_SECOND)
        self.sync_client = SECClient(user_agent)

    async def _rate_limit(self):
        """Async rate limiting."""
        async with self.semaphore:
            current_time = time.time()
            elapsed = current_time - self.last_request_time

            if elapsed < self.MIN_REQUEST_INTERVAL:
                sleep_time = self.MIN_REQUEST_INTERVAL - elapsed
                await asyncio.sleep(sleep_time)

            self.last_request_time = time.time()

    async def fetch_company_facts(self, cik: str) -> Optional[Dict[str, Any]]:
        """Async fetch company facts."""
        try:
            cik_int = int(str(cik).lstrip('0') or '0')
        except ValueError:
            logger.error(f"Invalid CIK format: {cik}")
            return None

        url = self.COMPANY_FACTS_URL.format(base=self.BASE_URL, cik=cik_int)

        await self._rate_limit()

        try:
            async with aiohttp.ClientSession() as session:
                headers = {
                    'User-Agent': self.user_agent,
                    'Accept': 'application/json',
                }
                async with session.get(url, headers=headers, timeout=30) as response:
                    if response.status == 404:
                        logger.warning(f"No company facts found for CIK {cik_int}")
                        return None
                    response.raise_for_status()
                    return await response.json()
        except Exception as e:
            logger.error(f"Error fetching CIK {cik_int}: {e}")
            return None

    async def get_financials_for_ticker(self, cik: str, num_periods: int = 20) -> List[Dict[str, Any]]:
        """Async fetch and parse financial data."""
        company_facts = await self.fetch_company_facts(cik)
        if not company_facts:
            return []

        # Use sync parsing (CPU-bound)
        financials = self.sync_client.parse_financial_data(company_facts, num_periods)
        return financials
