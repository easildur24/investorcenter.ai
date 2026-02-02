"""
Portfolio construction utilities for IC Score backtesting.

This module provides tools for building decile portfolios and
tracking portfolio composition over time.
"""

from dataclasses import dataclass, field
from datetime import date
from enum import Enum
from typing import Dict, List, Optional, Tuple
import logging

logger = logging.getLogger(__name__)


class WeightingScheme(Enum):
    """Portfolio weighting schemes."""
    EQUAL = "equal"
    MARKET_CAP = "market_cap"
    SCORE_WEIGHTED = "score_weighted"
    INVERSE_VOLATILITY = "inverse_volatility"


@dataclass
class Holding:
    """Individual portfolio holding."""
    ticker: str
    weight: float
    score: Optional[float] = None
    market_cap: Optional[float] = None
    sector: Optional[str] = None


@dataclass
class Portfolio:
    """Portfolio state at a point in time."""
    as_of_date: date
    holdings: List[Holding]
    total_value: float = 1.0

    @property
    def tickers(self) -> List[str]:
        return [h.ticker for h in self.holdings]

    @property
    def num_holdings(self) -> int:
        return len(self.holdings)

    @property
    def average_score(self) -> Optional[float]:
        scores = [h.score for h in self.holdings if h.score is not None]
        return sum(scores) / len(scores) if scores else None

    def get_weight(self, ticker: str) -> float:
        for h in self.holdings:
            if h.ticker == ticker:
                return h.weight
        return 0.0

    def sector_weights(self) -> Dict[str, float]:
        """Get portfolio weights by sector."""
        weights: Dict[str, float] = {}
        for h in self.holdings:
            sector = h.sector or "Unknown"
            weights[sector] = weights.get(sector, 0) + h.weight
        return weights


@dataclass
class PortfolioTransition:
    """Track changes between portfolio periods."""
    from_date: date
    to_date: date
    buys: List[str]
    sells: List[str]
    holds: List[str]
    turnover: float


class DecilePortfolioBuilder:
    """
    Build decile portfolios from scored stocks.

    Supports various weighting schemes and provides
    utilities for tracking portfolio changes over time.
    """

    def __init__(
        self,
        weighting: WeightingScheme = WeightingScheme.EQUAL,
        min_holdings: int = 10,
        max_holdings: int = 100
    ):
        self.weighting = weighting
        self.min_holdings = min_holdings
        self.max_holdings = max_holdings

    def build_decile_portfolios(
        self,
        scores: Dict[str, float],
        market_caps: Optional[Dict[str, float]] = None,
        sectors: Optional[Dict[str, str]] = None,
        as_of_date: Optional[date] = None
    ) -> Dict[int, Portfolio]:
        """
        Build 10 portfolios from scored stocks.

        Args:
            scores: Dict of ticker -> IC Score
            market_caps: Optional dict of ticker -> market cap
            sectors: Optional dict of ticker -> sector
            as_of_date: Date for the portfolio

        Returns:
            Dict mapping decile (1-10) to Portfolio
        """
        if not scores:
            return {}

        # Sort by score descending
        sorted_stocks = sorted(
            scores.items(),
            key=lambda x: x[1],
            reverse=True
        )

        n = len(sorted_stocks)
        decile_size = max(1, n // 10)

        portfolios: Dict[int, Portfolio] = {}

        for decile in range(1, 11):
            start_idx = (decile - 1) * decile_size
            end_idx = start_idx + decile_size

            # Last decile gets remainder
            if decile == 10:
                end_idx = n

            decile_stocks = sorted_stocks[start_idx:end_idx]

            if not decile_stocks:
                continue

            # Build holdings
            holdings = self._build_holdings(
                decile_stocks,
                market_caps,
                sectors
            )

            # Apply weighting
            holdings = self._apply_weights(holdings, market_caps)

            portfolios[decile] = Portfolio(
                as_of_date=as_of_date or date.today(),
                holdings=holdings
            )

        return portfolios

    def build_long_short_portfolio(
        self,
        scores: Dict[str, float],
        long_decile: int = 1,
        short_decile: int = 10,
        market_caps: Optional[Dict[str, float]] = None,
        sectors: Optional[Dict[str, str]] = None,
        as_of_date: Optional[date] = None
    ) -> Tuple[Portfolio, Portfolio]:
        """
        Build a long-short portfolio.

        Returns:
            Tuple of (long_portfolio, short_portfolio)
        """
        deciles = self.build_decile_portfolios(
            scores, market_caps, sectors, as_of_date
        )

        long_port = deciles.get(long_decile, Portfolio(
            as_of_date=as_of_date or date.today(),
            holdings=[]
        ))

        short_port = deciles.get(short_decile, Portfolio(
            as_of_date=as_of_date or date.today(),
            holdings=[]
        ))

        return long_port, short_port

    def build_quintile_portfolios(
        self,
        scores: Dict[str, float],
        market_caps: Optional[Dict[str, float]] = None,
        sectors: Optional[Dict[str, str]] = None,
        as_of_date: Optional[date] = None
    ) -> Dict[int, Portfolio]:
        """Build 5 portfolios (quintiles) instead of 10."""
        if not scores:
            return {}

        sorted_stocks = sorted(
            scores.items(),
            key=lambda x: x[1],
            reverse=True
        )

        n = len(sorted_stocks)
        quintile_size = max(1, n // 5)

        portfolios: Dict[int, Portfolio] = {}

        for quintile in range(1, 6):
            start_idx = (quintile - 1) * quintile_size
            end_idx = start_idx + quintile_size

            if quintile == 5:
                end_idx = n

            quintile_stocks = sorted_stocks[start_idx:end_idx]

            if not quintile_stocks:
                continue

            holdings = self._build_holdings(
                quintile_stocks,
                market_caps,
                sectors
            )
            holdings = self._apply_weights(holdings, market_caps)

            portfolios[quintile] = Portfolio(
                as_of_date=as_of_date or date.today(),
                holdings=holdings
            )

        return portfolios

    def calculate_transition(
        self,
        from_portfolio: Portfolio,
        to_portfolio: Portfolio
    ) -> PortfolioTransition:
        """Calculate changes between two portfolios."""
        from_tickers = set(from_portfolio.tickers)
        to_tickers = set(to_portfolio.tickers)

        buys = list(to_tickers - from_tickers)
        sells = list(from_tickers - to_tickers)
        holds = list(from_tickers & to_tickers)

        # Turnover is the fraction of portfolio that changed
        total = len(from_tickers | to_tickers)
        changed = len(buys) + len(sells)
        turnover = changed / total if total > 0 else 0

        return PortfolioTransition(
            from_date=from_portfolio.as_of_date,
            to_date=to_portfolio.as_of_date,
            buys=buys,
            sells=sells,
            holds=holds,
            turnover=turnover
        )

    def _build_holdings(
        self,
        stocks: List[Tuple[str, float]],
        market_caps: Optional[Dict[str, float]],
        sectors: Optional[Dict[str, str]]
    ) -> List[Holding]:
        """Build holding objects from stock list."""
        holdings = []

        for ticker, score in stocks:
            holdings.append(Holding(
                ticker=ticker,
                weight=0.0,  # Will be set by _apply_weights
                score=score,
                market_cap=market_caps.get(ticker) if market_caps else None,
                sector=sectors.get(ticker) if sectors else None
            ))

        return holdings

    def _apply_weights(
        self,
        holdings: List[Holding],
        market_caps: Optional[Dict[str, float]]
    ) -> List[Holding]:
        """Apply weighting scheme to holdings."""
        if not holdings:
            return holdings

        if self.weighting == WeightingScheme.EQUAL:
            weight = 1.0 / len(holdings)
            for h in holdings:
                h.weight = weight

        elif self.weighting == WeightingScheme.MARKET_CAP:
            if not market_caps:
                # Fall back to equal weight
                weight = 1.0 / len(holdings)
                for h in holdings:
                    h.weight = weight
            else:
                total_cap = sum(
                    market_caps.get(h.ticker, 0) for h in holdings
                )
                if total_cap > 0:
                    for h in holdings:
                        cap = market_caps.get(h.ticker, 0)
                        h.weight = cap / total_cap
                else:
                    weight = 1.0 / len(holdings)
                    for h in holdings:
                        h.weight = weight

        elif self.weighting == WeightingScheme.SCORE_WEIGHTED:
            total_score = sum(h.score or 50 for h in holdings)
            if total_score > 0:
                for h in holdings:
                    h.weight = (h.score or 50) / total_score
            else:
                weight = 1.0 / len(holdings)
                for h in holdings:
                    h.weight = weight

        return holdings


class SectorNeutralBuilder:
    """
    Build sector-neutral decile portfolios.

    Ensures each decile has similar sector exposure
    to prevent sector bias.
    """

    def __init__(
        self,
        weighting: WeightingScheme = WeightingScheme.EQUAL
    ):
        self.weighting = weighting

    def build_sector_neutral_portfolios(
        self,
        scores: Dict[str, float],
        sectors: Dict[str, str],
        market_caps: Optional[Dict[str, float]] = None,
        as_of_date: Optional[date] = None
    ) -> Dict[int, Portfolio]:
        """
        Build decile portfolios that are sector-neutral.

        Each decile will have proportional representation
        from each sector based on universe weights.
        """
        # Group by sector
        sector_stocks: Dict[str, List[Tuple[str, float]]] = {}

        for ticker, score in scores.items():
            sector = sectors.get(ticker, "Unknown")
            if sector not in sector_stocks:
                sector_stocks[sector] = []
            sector_stocks[sector].append((ticker, score))

        # Sort each sector by score
        for sector in sector_stocks:
            sector_stocks[sector].sort(key=lambda x: x[1], reverse=True)

        # Calculate sector weights in universe
        total_stocks = len(scores)
        sector_weights = {
            s: len(stocks) / total_stocks
            for s, stocks in sector_stocks.items()
        }

        # Build deciles
        portfolios: Dict[int, Portfolio] = {
            i: Portfolio(
                as_of_date=as_of_date or date.today(),
                holdings=[]
            )
            for i in range(1, 11)
        }

        # Allocate from each sector to each decile
        for sector, stocks in sector_stocks.items():
            n_sector = len(stocks)
            decile_size = max(1, n_sector // 10)

            for decile in range(1, 11):
                start_idx = (decile - 1) * decile_size
                end_idx = start_idx + decile_size

                if decile == 10:
                    end_idx = n_sector

                for ticker, score in stocks[start_idx:end_idx]:
                    portfolios[decile].holdings.append(Holding(
                        ticker=ticker,
                        weight=0.0,
                        score=score,
                        sector=sector,
                        market_cap=market_caps.get(ticker) if market_caps else None
                    ))

        # Apply weights within each decile
        for decile, portfolio in portfolios.items():
            if portfolio.holdings:
                weight = 1.0 / len(portfolio.holdings)
                for h in portfolio.holdings:
                    h.weight = weight

        return portfolios


@dataclass
class FactorExposure:
    """Factor exposure of a portfolio."""
    beta: float = 1.0
    size: float = 0.0  # Positive = large cap
    value: float = 0.0  # Positive = value
    momentum: float = 0.0
    quality: float = 0.0
    volatility: float = 0.0


class FactorAnalyzer:
    """Analyze factor exposures of portfolios."""

    def calculate_exposures(
        self,
        portfolio: Portfolio,
        factor_scores: Dict[str, Dict[str, float]]
    ) -> FactorExposure:
        """
        Calculate portfolio's factor exposures.

        Args:
            portfolio: Portfolio to analyze
            factor_scores: Dict of factor_name -> {ticker: score}

        Returns:
            FactorExposure object
        """
        exposures = {
            'beta': 0.0,
            'size': 0.0,
            'value': 0.0,
            'momentum': 0.0,
            'quality': 0.0,
            'volatility': 0.0
        }

        for factor_name in exposures:
            if factor_name in factor_scores:
                scores = factor_scores[factor_name]
                weighted_score = 0.0

                for holding in portfolio.holdings:
                    if holding.ticker in scores:
                        weighted_score += (
                            holding.weight * scores[holding.ticker]
                        )

                exposures[factor_name] = weighted_score

        return FactorExposure(**exposures)

    def calculate_active_exposure(
        self,
        portfolio: Portfolio,
        benchmark: Portfolio,
        factor_scores: Dict[str, Dict[str, float]]
    ) -> FactorExposure:
        """Calculate active (relative to benchmark) factor exposures."""
        port_exp = self.calculate_exposures(portfolio, factor_scores)
        bench_exp = self.calculate_exposures(benchmark, factor_scores)

        return FactorExposure(
            beta=port_exp.beta - bench_exp.beta,
            size=port_exp.size - bench_exp.size,
            value=port_exp.value - bench_exp.value,
            momentum=port_exp.momentum - bench_exp.momentum,
            quality=port_exp.quality - bench_exp.quality,
            volatility=port_exp.volatility - bench_exp.volatility
        )
