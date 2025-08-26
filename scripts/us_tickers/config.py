"""Configuration management for US Tickers package."""

import os
from dataclasses import dataclass
from typing import List, Optional


@dataclass
class Config:
    """Configuration for US Tickers package."""

    # Data source URLs
    nasdaq_base_url: str = "https://www.nasdaqtrader.com/dynamic/symdir/"
    nasdaq_listed_filename: str = "nasdaqlisted.txt"
    other_listed_filename: str = "otherlisted.txt"

    # Network settings
    timeout_seconds: int = 20
    max_retries: int = 3
    backoff_factor: float = 1.0

    # Cache settings
    cache_ttl_hours: int = 24
    cache_dir: str = ".cache"

    # Data processing
    default_exchanges: Optional[List[str]] = None
    default_include_etfs: bool = False
    default_include_test_issues: bool = False

    def __post_init__(self) -> None:
        """Set default values after initialization."""
        if self.default_exchanges is None:
            self.default_exchanges = ["Q", "N"]

    @property
    def nasdaq_listed_url(self) -> str:
        """Get full Nasdaq listed URL."""
        base = self.nasdaq_base_url.rstrip("/")
        return f"{base}/{self.nasdaq_listed_filename}"

    @property
    def other_listed_url(self) -> str:
        """Get full other listed URL."""
        base = self.nasdaq_base_url.rstrip("/")
        return f"{base}/{self.other_listed_filename}"

    @classmethod
    def from_env(cls) -> "Config":
        """Create configuration from environment variables."""
        return cls(
            nasdaq_base_url=os.getenv(
                "NASDAQ_BASE_URL",
                "https://www.nasdaqtrader.com/dynamic/symdir/",
            ),
            timeout_seconds=int(os.getenv("TIMEOUT_SECONDS", "20")),
            max_retries=int(os.getenv("MAX_RETRIES", "3")),
            backoff_factor=float(os.getenv("BACKOFF_FACTOR", "1.0")),
            cache_ttl_hours=int(os.getenv("CACHE_TTL_HOURS", "24")),
            cache_dir=os.getenv("CACHE_DIR", ".cache"),
            default_include_etfs=(
                os.getenv("DEFAULT_INCLUDE_ETFS", "false").lower() == "true"
            ),
            default_include_test_issues=(
                os.getenv("DEFAULT_INCLUDE_TEST_ISSUES", "false").lower()
                == "true"
            ),
        )


# Global configuration instance
config = Config.from_env()
