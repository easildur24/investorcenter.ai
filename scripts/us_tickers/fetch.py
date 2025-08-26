"""Core functionality for fetching and processing US exchange tickers."""

import logging
from typing import Optional, Tuple

import pandas as pd
import requests
from urllib3.util.retry import Retry

from .cache import SimpleCache
from .config import config
from .validation import validate_exchange_config

logger = logging.getLogger(__name__)

# Exchange codes mapping
EXCHANGE_CODES = {
    "Q": "Nasdaq",
    "N": "NYSE",
    "A": "NYSE American",
    "P": "NYSE Arca",
    "Z": "Cboe",
}

# Column mappings for different file formats
NASDAQ_COLUMNS = [
    "Symbol",
    "Security Name",
    "Market Category",
    "Test Issue",
    "Financial Status",
    "Round Lot Size",
    "ETF",
    "NextShares",
]

OTHER_COLUMNS = [
    "ACT Symbol",
    "Security Name",
    "Exchange",
    "CQS Symbol",
    "ETF",
    "Round Lot Size",
    "Test Issue",
    "NASDAQ Symbol",
    "NextShares",
]


def _create_session_with_retries() -> requests.Session:
    """Create a requests session with retry configuration."""
    session = requests.Session()

    # Configure retry strategy
    retry_strategy = Retry(
        total=config.max_retries,
        backoff_factor=config.backoff_factor,
        status_forcelist=[429, 500, 502, 503, 504],
        allowed_methods=["GET"],
    )

    # Configure adapter with retry strategy
    adapter = requests.adapters.HTTPAdapter(max_retries=retry_strategy)
    session.mount("http://", adapter)
    session.mount("https://", adapter)

    # Set timeout
    session.timeout = config.timeout_seconds

    return session


def _download_file(
    url: str,
    session: Optional[requests.Session] = None
) -> str:
    """Download file content from URL with retries."""
    if session is None:
        session = _create_session_with_retries()

    logger.info(f"Downloading {url}")
    try:
        response = session.get(url)
        response.raise_for_status()
        content = response.text
        msg = f"Successfully downloaded {url} ({len(content)} characters)"
        logger.info(msg)
        return content
    except requests.exceptions.RequestException as e:
        logger.error(f"Failed to download {url}: {e}")
        raise


def _clean_dataframe(df: pd.DataFrame) -> pd.DataFrame:
    """Clean and normalize DataFrame data."""
    if df.empty:
        return df

    # Clean and normalize string columns
    for col in df.columns:
        if df[col].dtype == "object":
            df[col] = df[col].str.strip().str.upper().fillna("")

    return df


def _parse_nasdaq_listed(content: str) -> pd.DataFrame:
    """Parse nasdaqlisted.txt content into a DataFrame."""
    logger.info("Parsing nasdaqlisted.txt")

    # Split content into lines and filter out empty lines
    lines = [line.strip() for line in content.split("\n") if line.strip()]

    # Remove footer row
    if lines and lines[-1].startswith("File Creation Time"):
        lines = lines[:-1]
        logger.debug("Removed footer row from nasdaqlisted.txt")

    if not lines:
        logger.warning("No data found in nasdaqlisted.txt after filtering")
        return pd.DataFrame()

    # Parse the data (skip header row)
    data = []
    for i, line in enumerate(lines):
        if "|" in line:
            fields = line.split("|")
            if len(fields) >= len(NASDAQ_COLUMNS):
                # Skip header row (first line)
                if i > 0:
                    data.append(fields[: len(NASDAQ_COLUMNS)])

    if not data:
        logger.warning("No valid data rows found in nasdaqlisted.txt")
        return pd.DataFrame()

    # Create DataFrame
    df = pd.DataFrame(data, columns=NASDAQ_COLUMNS)

    # Standardize columns
    df = df.rename(columns={"Symbol": "Ticker"})
    df["Exchange"] = "Q"  # Nasdaq exchange code

    # Clean and normalize data
    df = _clean_dataframe(df)

    logger.info(f"Parsed {len(df)} rows from nasdaqlisted.txt")
    return df


def _parse_other_listed(content: str) -> pd.DataFrame:
    """Parse otherlisted.txt content into a DataFrame."""
    logger.info("Parsing otherlisted.txt")

    # Split content into lines and filter out empty lines
    lines = [line.strip() for line in content.split("\n") if line.strip()]

    # Remove footer row
    if lines and lines[-1].startswith("File Creation Time"):
        lines = lines[:-1]
        logger.debug("Removed footer row from otherlisted.txt")

    if not lines:
        logger.warning("No data found in otherlisted.txt after filtering")
        return pd.DataFrame()

    # Parse the data (skip header row)
    data = []
    for i, line in enumerate(lines):
        if "|" in line:
            fields = line.split("|")
            if len(fields) >= len(OTHER_COLUMNS):
                # Skip header row (first line)
                if i > 0:
                    data.append(fields[: len(OTHER_COLUMNS)])

    if not data:
        logger.warning("No valid data rows found in otherlisted.txt")
        return pd.DataFrame()

    # Create DataFrame
    df = pd.DataFrame(data, columns=OTHER_COLUMNS)

    # Standardize columns
    df = df.rename(columns={"ACT Symbol": "Ticker"})

    # Clean and normalize data
    df = _clean_dataframe(df)

    logger.info(f"Parsed {len(df)} rows from otherlisted.txt")
    return df


def _filter_and_clean_data(
    df: pd.DataFrame,
    exchanges: Tuple[str, ...],
    include_etfs: bool,
    include_test_issues: bool,
) -> pd.DataFrame:
    """Filter and clean the combined DataFrame."""
    initial_count = len(df)

    # Handle empty DataFrame
    if df.empty:
        logger.info("Empty DataFrame provided, returning empty result")
        return df

    # Filter by exchange
    if exchanges:
        df = df[df["Exchange"].isin(exchanges)]
        logger.info(f"Filtered to exchanges {exchanges}: {len(df)} rows")

    # Filter out ETFs if not requested
    if not include_etfs:
        df = df[df["ETF"] != "Y"]
        logger.info(f"Excluded ETFs: {len(df)} rows")

    # Filter out test issues if not requested
    if not include_test_issues:
        df = df[df["Test Issue"] != "Y"]
        logger.info(f"Excluded test issues: {len(df)} rows")

    # Remove duplicates by ticker and sort
    df = df.drop_duplicates(subset=["Ticker"]).sort_values("Ticker")

    logger.info(
        f"Final result: {len(df)} unique tickers "
        f"(from {initial_count} total rows)"
    )
    return df


def get_exchange_listed_tickers(
    exchanges: Optional[Tuple[str, ...]] = None,
    include_etfs: Optional[bool] = None,
    include_test_issues: Optional[bool] = None,
    cache_ttl_hours: Optional[int] = None,
    session: Optional[requests.Session] = None,
) -> Tuple[list[str], pd.DataFrame]:
    """
    Download and merge Nasdaq + NYSE tickers from Nasdaq Trader symbol directories.

    Args:
        exchanges: Tuple of exchange codes to include. Default: ("Q", "N")
                  Q = Nasdaq, N = NYSE, A = NYSE American, P = NYSE Arca, Z = Cboe
        include_etfs: Whether to include ETFs. Default: False
        include_test_issues: Whether to include test issues. Default: False
        cache_ttl_hours: Cache TTL in hours (not implemented in this version). Default: 24
        session: Optional requests.Session for custom HTTP configuration

    Returns:
        Tuple of (ticker_list, dataframe) where:
        - ticker_list: List of ticker symbols as strings
        - dataframe: Pandas DataFrame with full ticker information

    Raises:
        requests.exceptions.RequestException: If download fails after retries
        ValueError: If invalid exchange codes provided
        RuntimeError: If all data sources fail
    """
    # Use configuration defaults if not specified
    if exchanges is None:
        exchanges = tuple(config.default_exchanges)
    if include_etfs is None:
        include_etfs = config.default_include_etfs
    if include_test_issues is None:
        include_test_issues = config.default_include_test_issues
    if cache_ttl_hours is None:
        cache_ttl_hours = config.cache_ttl_hours

    # Validate exchange configuration
    try:
        exchange_config = validate_exchange_config(
            exchanges, include_etfs, include_test_issues
        )
        exchanges = exchange_config.exchanges
        include_etfs = exchange_config.include_etfs
        include_test_issues = exchange_config.include_test_issues
    except ValueError as e:
        raise ValueError(f"Invalid configuration: {e}")

    logger.info(f"Fetching tickers for exchanges: {exchanges}")
    msg = (
        f"Include ETFs: {include_etfs}, "
        f"Include test issues: {include_test_issues}"
    )
    logger.info(msg)

    # Check cache first if caching is enabled
    if cache_ttl_hours and cache_ttl_hours > 0:
        cache = SimpleCache(config.cache_dir)
        cache_key = (
            f"tickers_{','.join(exchanges)}_"
            f"{include_etfs}_{include_test_issues}"
        )

        cached_result = cache.get(cache_key, cache_ttl_hours)
        if cached_result is not None:
            logger.info("Using cached ticker data")
            ticker_list, result_df = cached_result
            return ticker_list, result_df

    # Download data with fallback - continue if one source fails
    nasdaq_df = pd.DataFrame()
    other_df = pd.DataFrame()

    try:
        nasdaq_content = _download_file(config.nasdaq_listed_url, session)
        nasdaq_df = _parse_nasdaq_listed(nasdaq_content)
    except Exception as e:
        logger.error(f"Failed to fetch Nasdaq data: {e}")
        # Continue with other exchanges if possible

    try:
        other_content = _download_file(config.other_listed_url, session)
        other_df = _parse_other_listed(other_content)
    except Exception as e:
        logger.error(f"Failed to fetch other exchanges data: {e}")

    # Check if we have any data
    if nasdaq_df.empty and other_df.empty:
        msg = (
            "Failed to fetch data from all sources. "
            "Please check your internet connection and try again."
        )
        raise RuntimeError(msg)

    # Combine dataframes (handle empty DataFrames gracefully)
    if nasdaq_df.empty:
        combined_df = other_df
        logger.info(f"Using only other exchanges data: {len(other_df)} rows")
    elif other_df.empty:
        combined_df = nasdaq_df
        logger.info(f"Using only Nasdaq data: {len(nasdaq_df)} rows")
    else:
        combined_df = pd.concat([nasdaq_df, other_df], ignore_index=True)
        logger.info(
            f"Combined {len(nasdaq_df)} Nasdaq + {len(other_df)} other = "
            f"{len(combined_df)} total rows"
        )

    # Filter and clean
    result_df = _filter_and_clean_data(
        combined_df, exchanges, include_etfs, include_test_issues
    )

    # Extract ticker list
    ticker_list = result_df["Ticker"].tolist()

    # Cache the result if caching is enabled
    if cache_ttl_hours and cache_ttl_hours > 0:
        try:
            cache = SimpleCache(config.cache_dir)
            cache_key = (
                f"tickers_{','.join(exchanges)}_"
                f"{include_etfs}_{include_test_issues}"
            )
            cache.set(cache_key, (ticker_list, result_df))
            logger.info("Cached ticker data for future use")
        except Exception as e:
            logger.warning(f"Failed to cache data: {e}")
            # Continue without caching - this is not critical

    return ticker_list, result_df
