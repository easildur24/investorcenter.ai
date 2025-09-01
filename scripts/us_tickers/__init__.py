"""US Tickers - Download and merge Nasdaq + NYSE tickers."""

__version__ = "0.1.0"

from .cache import SimpleCache
from .config import config
from .fetch import get_exchange_listed_tickers
from .transform import transform_for_database
from .validation import ExchangeConfig, TickerData, validate_exchange_config

# Optional database imports - only available if psycopg2 is installed
try:
    from .database import import_stocks_to_database, test_database_connection

    _database_available = True
except ImportError:
    _database_available = False
    import_stocks_to_database = None  # type: ignore
    test_database_connection = None  # type: ignore

__all__ = [
    "get_exchange_listed_tickers",
    "config",
    "SimpleCache",
    "validate_exchange_config",
    "ExchangeConfig",
    "TickerData",
    "transform_for_database",
]

# Add database functions to __all__ only if available
if _database_available:
    __all__.extend(
        [
            "import_stocks_to_database",
            "test_database_connection",
        ]
    )
