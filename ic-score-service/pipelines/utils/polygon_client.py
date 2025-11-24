"""Polygon.io API client for fetching market data.

This module provides a client for accessing Polygon.io API to retrieve
stock prices, news, and other market data.

Rate Limit: 5 requests/second for basic tier, adjust as needed.
"""

import logging
import os
import time
from datetime import datetime, timedelta
from typing import Any, Dict, List, Optional

import requests
from requests.adapters import HTTPAdapter
from urllib3.util.retry import Retry

logger = logging.getLogger(__name__)


class PolygonClient:
    """Client for Polygon.io API with rate limiting."""

    BASE_URL = "https://api.polygon.io"

    # Rate limiting (adjust based on your tier)
    REQUESTS_PER_SECOND = 5
    MIN_REQUEST_INTERVAL = 1.0 / REQUESTS_PER_SECOND

    def __init__(self, api_key: Optional[str] = None):
        """Initialize Polygon client.

        Args:
            api_key: Polygon API key. Defaults to POLYGON_API_KEY env var.
        """
        self.api_key = api_key or os.getenv('POLYGON_API_KEY')
        if not self.api_key:
            raise ValueError("POLYGON_API_KEY environment variable not set")

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

        return session

    def _rate_limit(self):
        """Implement rate limiting."""
        current_time = time.time()
        elapsed = current_time - self.last_request_time

        if elapsed < self.MIN_REQUEST_INTERVAL:
            sleep_time = self.MIN_REQUEST_INTERVAL - elapsed
            time.sleep(sleep_time)

        self.last_request_time = time.time()

    def _make_request(self, endpoint: str, params: Dict[str, Any] = None) -> Optional[Dict[str, Any]]:
        """Make API request with rate limiting.

        Args:
            endpoint: API endpoint path.
            params: Query parameters.

        Returns:
            JSON response or None if request fails.
        """
        url = f"{self.BASE_URL}{endpoint}"
        params = params or {}
        params['apiKey'] = self.api_key

        self._rate_limit()

        try:
            response = self.session.get(url, params=params, timeout=30)
            response.raise_for_status()
            return response.json()
        except requests.exceptions.HTTPError as e:
            if e.response.status_code == 404:
                logger.warning(f"Resource not found: {endpoint}")
            else:
                logger.error(f"HTTP error: {e}")
            return None
        except requests.exceptions.RequestException as e:
            logger.error(f"Request error: {e}")
            return None
        except Exception as e:
            logger.exception(f"Unexpected error: {e}")
            return None

    def get_aggregates(
        self,
        ticker: str,
        multiplier: int = 1,
        timespan: str = 'day',
        from_date: str = None,
        to_date: str = None,
        limit: int = 50000
    ) -> Optional[List[Dict[str, Any]]]:
        """Get aggregate bars for a ticker.

        Args:
            ticker: Stock ticker symbol.
            multiplier: Size of the timespan multiplier.
            timespan: Size of the time window (minute, hour, day, week, etc.).
            from_date: Start date (YYYY-MM-DD).
            to_date: End date (YYYY-MM-DD).
            limit: Limit the number of results.

        Returns:
            List of OHLCV bars or None if request fails.
        """
        if not from_date:
            # Default to 252 trading days (1 year)
            from_date = (datetime.now() - timedelta(days=365)).strftime('%Y-%m-%d')

        if not to_date:
            to_date = datetime.now().strftime('%Y-%m-%d')

        endpoint = f"/v2/aggs/ticker/{ticker}/range/{multiplier}/{timespan}/{from_date}/{to_date}"

        params = {
            'adjusted': 'true',
            'sort': 'asc',
            'limit': limit
        }

        response = self._make_request(endpoint, params)

        if not response or 'results' not in response:
            return None

        results = response['results']

        # Convert timestamps to dates
        for bar in results:
            if 't' in bar:
                bar['date'] = datetime.fromtimestamp(bar['t'] / 1000).date()

        return results

    def get_daily_prices(
        self,
        ticker: str,
        days: int = 252
    ) -> Optional[List[Dict[str, Any]]]:
        """Get daily OHLCV data for a ticker.

        Args:
            ticker: Stock ticker symbol.
            days: Number of days to retrieve (default: 252 trading days).

        Returns:
            List of daily price bars.
        """
        from_date = (datetime.now() - timedelta(days=days + 50)).strftime('%Y-%m-%d')
        to_date = datetime.now().strftime('%Y-%m-%d')

        return self.get_aggregates(
            ticker=ticker,
            multiplier=1,
            timespan='day',
            from_date=from_date,
            to_date=to_date
        )

    def get_news(
        self,
        ticker: Optional[str] = None,
        limit: int = 100,
        published_after: Optional[str] = None
    ) -> Optional[List[Dict[str, Any]]]:
        """Get news articles.

        Args:
            ticker: Stock ticker symbol (optional).
            limit: Maximum number of results.
            published_after: Filter for articles published after this date (YYYY-MM-DD).

        Returns:
            List of news articles.
        """
        endpoint = "/v2/reference/news"

        params = {
            'limit': limit,
            'order': 'desc',
            'sort': 'published_utc'
        }

        if ticker:
            params['ticker'] = ticker

        if published_after:
            params['published_utc.gte'] = published_after

        response = self._make_request(endpoint, params)

        if not response or 'results' not in response:
            return None

        return response['results']

    def get_ticker_details(self, ticker: str) -> Optional[Dict[str, Any]]:
        """Get ticker details and company information.

        Args:
            ticker: Stock ticker symbol.

        Returns:
            Ticker details dictionary.
        """
        endpoint = f"/v3/reference/tickers/{ticker}"

        response = self._make_request(endpoint)

        if not response or 'results' not in response:
            return None

        return response['results']

    def get_latest_price(self, ticker: str) -> Optional[Dict[str, Any]]:
        """Get the most recent closing price for a ticker.

        Args:
            ticker: Stock ticker symbol.

        Returns:
            Dictionary with latest price data (close, date) or None if not found.
        """
        # Get last 7 days of data to ensure we get latest
        bars = self.get_daily_prices(ticker, days=7)

        if not bars or len(bars) == 0:
            logger.warning(f"{ticker}: No price data available")
            return None

        # Return most recent bar
        latest_bar = bars[-1]

        return {
            'close': latest_bar.get('c'),  # Close price
            'open': latest_bar.get('o'),
            'high': latest_bar.get('h'),
            'low': latest_bar.get('l'),
            'volume': latest_bar.get('v'),
            'date': latest_bar.get('date'),
            'timestamp': latest_bar.get('t')
        }

    def close(self):
        """Close the HTTP session."""
        if self.session:
            self.session.close()
