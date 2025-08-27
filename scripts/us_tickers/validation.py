"""Data validation for US Tickers package."""

from typing import Tuple

from pydantic import BaseModel, ConfigDict, field_validator


class ExchangeConfig(BaseModel):
    """Configuration for exchange filtering."""

    exchanges: Tuple[str, ...]
    include_etfs: bool
    include_test_issues: bool

    @field_validator("exchanges")
    @classmethod
    def validate_exchanges(cls, v: Tuple[str, ...]) -> Tuple[str, ...]:
        """Validate exchange codes."""
        valid_exchanges = {"Q", "N", "A", "P", "Z"}
        invalid_exchanges = [ex for ex in v if ex not in valid_exchanges]
        if invalid_exchanges:
            codes = list(valid_exchanges)
            raise ValueError(
                f"Invalid exchange codes: {invalid_exchanges}. "
                f"Valid codes: {codes}"
            )
        return v

    model_config = ConfigDict(frozen=True)


class TickerData(BaseModel):
    """Validation model for ticker data."""

    ticker: str
    exchange: str
    security_name: str
    etf: str
    test_issue: str

    @field_validator("ticker")
    @classmethod
    def validate_ticker(cls, v: str) -> str:
        """Validate ticker symbol."""
        if not v or not v.strip():
            raise ValueError("Ticker cannot be empty")
        if len(v) > 10:
            raise ValueError("Ticker too long (max 10 characters)")
        return v.strip().upper()

    @field_validator("exchange")
    @classmethod
    def validate_exchange(cls, v: str) -> str:
        """Validate exchange code."""
        valid_exchanges = {"Q", "N", "A", "P", "Z"}
        if v not in valid_exchanges:
            raise ValueError(f"Invalid exchange code: {v}")
        return v

    model_config = ConfigDict(
        extra="allow"  # Allow additional fields from the DataFrame
    )


def validate_exchange_config(
    exchanges: Tuple[str, ...], include_etfs: bool, include_test_issues: bool
) -> ExchangeConfig:
    """Validate exchange configuration."""
    return ExchangeConfig(
        exchanges=exchanges,
        include_etfs=include_etfs,
        include_test_issues=include_test_issues,
    )


def validate_ticker_data(data: dict) -> TickerData:
    """Validate individual ticker data."""
    return TickerData(**data)
