"""Peer Comparison Service for IC Score v2.1 Phase 3.

This module identifies and compares similar stocks (peers) based on:
- Same sector
- Similar market cap (0.25x to 4x range)
- Similar growth profile
- Similar profitability

Key features:
- Multi-factor similarity scoring
- Caching for performance
- IC Score comparison for peers
"""

import logging
from dataclasses import dataclass
from datetime import date
from typing import Dict, List, Optional, Tuple, Any

from sqlalchemy import text

logger = logging.getLogger(__name__)


@dataclass
class PeerStock:
    """A peer stock with similarity metrics."""
    ticker: str
    company_name: Optional[str]
    sector: str
    market_cap: Optional[float]
    similarity_score: float
    similarity_factors: Dict[str, float]


@dataclass
class PeerComparison:
    """Comparison between a stock and its peer."""
    ticker: str
    company_name: Optional[str]
    ic_score: Optional[float]
    delta: float  # Difference from target stock's IC Score
    similarity_score: float


@dataclass
class PeerComparisonResult:
    """Result of peer comparison analysis."""
    ticker: str
    ic_score: float
    sector: str
    peers: List[PeerComparison]
    sector_rank: int
    sector_total: int
    sector_percentile: float


class PeerComparisonService:
    """Service for finding and comparing peer stocks.

    Peer selection algorithm:
    1. Filter to same sector
    2. Filter by market cap range (0.25x to 4x)
    3. Score by multi-factor similarity
    4. Return top N peers with IC Score comparison
    """

    # Market cap range for peer selection
    MARKET_CAP_MIN_RATIO = 0.25
    MARKET_CAP_MAX_RATIO = 4.0

    # Similarity weights
    SIMILARITY_WEIGHTS = {
        'market_cap': 0.30,
        'revenue_growth': 0.20,
        'net_margin': 0.20,
        'pe_ratio': 0.15,
        'beta': 0.15,
    }

    # Default number of peers
    DEFAULT_PEER_COUNT = 5

    def __init__(self, session):
        """Initialize service with database session.

        Args:
            session: SQLAlchemy async session for database queries.
        """
        self.session = session
        self._stock_cache: Dict[str, Dict] = {}

    async def get_peers(
        self,
        ticker: str,
        limit: int = DEFAULT_PEER_COUNT
    ) -> Optional[PeerComparisonResult]:
        """Get peer comparison for a stock.

        Args:
            ticker: Stock ticker symbol.
            limit: Maximum number of peers to return.

        Returns:
            PeerComparisonResult with peers and rankings, or None.
        """
        # Get target stock data
        stock = await self._get_stock_data(ticker)
        if not stock:
            logger.warning(f"Stock data not found for {ticker}")
            return None

        sector = stock.get('sector')
        if not sector:
            logger.warning(f"No sector information for {ticker}")
            return None

        market_cap = stock.get('market_cap')
        if not market_cap:
            logger.warning(f"No market cap for {ticker}")
            return None

        # Get candidate peers
        candidates = await self._get_peer_candidates(
            ticker, sector, market_cap
        )

        if not candidates:
            logger.warning(f"No peer candidates found for {ticker}")
            return None

        # Score candidates by similarity
        scored_peers = []
        for candidate in candidates:
            similarity = self._calculate_similarity(stock, candidate)
            scored_peers.append(PeerStock(
                ticker=candidate['ticker'],
                company_name=candidate.get('company_name'),
                sector=sector,
                market_cap=candidate.get('market_cap'),
                similarity_score=similarity['total'],
                similarity_factors=similarity['factors']
            ))

        # Sort by similarity and take top N
        scored_peers.sort(key=lambda x: x.similarity_score, reverse=True)
        top_peers = scored_peers[:limit]

        # Get IC Scores for peers
        stock_ic_score = await self._get_ic_score(ticker)
        peer_comparisons = []

        for peer in top_peers:
            peer_ic_score = await self._get_ic_score(peer.ticker)

            peer_comparisons.append(PeerComparison(
                ticker=peer.ticker,
                company_name=peer.company_name,
                ic_score=peer_ic_score,
                delta=round((peer_ic_score or 0) - (stock_ic_score or 0), 2) if peer_ic_score else 0,
                similarity_score=peer.similarity_score
            ))

        # Get sector ranking
        sector_rank, sector_total = await self._get_sector_rank(
            ticker, sector, stock_ic_score
        )

        sector_percentile = 0
        if sector_rank and sector_total and sector_total > 0:
            sector_percentile = round((sector_total - sector_rank + 1) / sector_total * 100, 1)

        return PeerComparisonResult(
            ticker=ticker,
            ic_score=stock_ic_score or 0,
            sector=sector,
            peers=peer_comparisons,
            sector_rank=sector_rank or 0,
            sector_total=sector_total or 0,
            sector_percentile=sector_percentile
        )

    async def _get_stock_data(self, ticker: str) -> Optional[Dict]:
        """Get stock data for similarity calculation."""
        if ticker in self._stock_cache:
            return self._stock_cache[ticker]

        try:
            query = text("""
                SELECT
                    t.symbol as ticker,
                    t.name,
                    t.sector,
                    t.market_cap,
                    fme.revenue_growth_yoy,
                    fme.net_margin,
                    fme.roe,
                    v.ttm_pe_ratio as pe_ratio
                FROM tickers t
                LEFT JOIN LATERAL (
                    SELECT revenue_growth_yoy, net_margin, roe
                    FROM fundamental_metrics_extended
                    WHERE ticker = t.symbol
                    ORDER BY calculation_date DESC
                    LIMIT 1
                ) fme ON true
                LEFT JOIN LATERAL (
                    SELECT ttm_pe_ratio
                    FROM valuation_ratios
                    WHERE ticker = t.symbol
                    ORDER BY calculation_date DESC
                    LIMIT 1
                ) v ON true
                WHERE t.symbol = :ticker
            """)

            result = await self.session.execute(query, {"ticker": ticker})
            row = result.fetchone()

            if not row:
                return None

            stock_data = {
                'ticker': row[0],
                'company_name': row[1],
                'sector': row[2],
                'market_cap': float(row[3]) if row[3] else None,
                'revenue_growth': float(row[4]) if row[4] else None,
                'net_margin': float(row[5]) if row[5] else None,
                'roe': float(row[6]) if row[6] else None,
                'pe_ratio': float(row[7]) if row[7] else None,
            }

            self._stock_cache[ticker] = stock_data
            return stock_data

        except Exception as e:
            logger.error(f"Error fetching stock data for {ticker}: {e}")
            return None

    async def _get_peer_candidates(
        self,
        ticker: str,
        sector: str,
        market_cap: float
    ) -> List[Dict]:
        """Get candidate peers from same sector with similar market cap."""
        try:
            min_cap = market_cap * self.MARKET_CAP_MIN_RATIO
            max_cap = market_cap * self.MARKET_CAP_MAX_RATIO

            query = text("""
                SELECT
                    t.symbol as ticker,
                    t.name,
                    t.sector,
                    t.market_cap,
                    fme.revenue_growth_yoy,
                    fme.net_margin,
                    fme.roe,
                    v.ttm_pe_ratio as pe_ratio
                FROM tickers t
                LEFT JOIN LATERAL (
                    SELECT revenue_growth_yoy, net_margin, roe
                    FROM fundamental_metrics_extended
                    WHERE ticker = t.symbol
                    ORDER BY calculation_date DESC
                    LIMIT 1
                ) fme ON true
                LEFT JOIN LATERAL (
                    SELECT ttm_pe_ratio
                    FROM valuation_ratios
                    WHERE ticker = t.symbol
                    ORDER BY calculation_date DESC
                    LIMIT 1
                ) v ON true
                WHERE t.sector = :sector
                  AND t.symbol != :ticker
                  AND t.market_cap BETWEEN :min_cap AND :max_cap
                  AND t.active = true
                LIMIT 50
            """)

            result = await self.session.execute(query, {
                "sector": sector,
                "ticker": ticker,
                "min_cap": min_cap,
                "max_cap": max_cap
            })

            candidates = []
            for row in result.fetchall():
                candidates.append({
                    'ticker': row[0],
                    'company_name': row[1],
                    'sector': row[2],
                    'market_cap': float(row[3]) if row[3] else None,
                    'revenue_growth': float(row[4]) if row[4] else None,
                    'net_margin': float(row[5]) if row[5] else None,
                    'roe': float(row[6]) if row[6] else None,
                    'pe_ratio': float(row[7]) if row[7] else None,
                })

            return candidates

        except Exception as e:
            logger.error(f"Error fetching peer candidates: {e}")
            return []

    def _calculate_similarity(
        self,
        stock: Dict,
        candidate: Dict
    ) -> Dict[str, Any]:
        """Calculate multi-factor similarity score.

        Returns a score between 0 and 1, where 1 is most similar.
        """
        factors = {}
        total_weight = 0
        weighted_score = 0

        # Market cap similarity (log scale)
        if stock.get('market_cap') and candidate.get('market_cap'):
            import math
            log_ratio = abs(
                math.log10(candidate['market_cap'] / stock['market_cap'])
            )
            # Perfect match = 1, 10x diff = 0
            cap_sim = max(0, 1 - log_ratio)
            factors['market_cap'] = round(cap_sim, 3)
            weight = self.SIMILARITY_WEIGHTS['market_cap']
            weighted_score += cap_sim * weight
            total_weight += weight

        # Revenue growth similarity
        if stock.get('revenue_growth') is not None and candidate.get('revenue_growth') is not None:
            growth_diff = abs(stock['revenue_growth'] - candidate['revenue_growth'])
            # Perfect match = 1, 50pp diff = 0
            growth_sim = max(0, 1 - growth_diff / 50)
            factors['revenue_growth'] = round(growth_sim, 3)
            weight = self.SIMILARITY_WEIGHTS['revenue_growth']
            weighted_score += growth_sim * weight
            total_weight += weight

        # Net margin similarity
        if stock.get('net_margin') is not None and candidate.get('net_margin') is not None:
            margin_diff = abs(stock['net_margin'] - candidate['net_margin'])
            # Perfect match = 1, 30pp diff = 0
            margin_sim = max(0, 1 - margin_diff / 30)
            factors['net_margin'] = round(margin_sim, 3)
            weight = self.SIMILARITY_WEIGHTS['net_margin']
            weighted_score += margin_sim * weight
            total_weight += weight

        # P/E ratio similarity
        if stock.get('pe_ratio') and candidate.get('pe_ratio'):
            if stock['pe_ratio'] > 0 and candidate['pe_ratio'] > 0:
                pe_ratio = max(stock['pe_ratio'], candidate['pe_ratio']) / min(stock['pe_ratio'], candidate['pe_ratio'])
                # Perfect match = 1, 3x diff = 0
                pe_sim = max(0, 1 - (pe_ratio - 1) / 2)
                factors['pe_ratio'] = round(pe_sim, 3)
                weight = self.SIMILARITY_WEIGHTS['pe_ratio']
                weighted_score += pe_sim * weight
                total_weight += weight

        # Calculate total similarity
        if total_weight > 0:
            total_similarity = weighted_score / total_weight
        else:
            total_similarity = 0.5  # Neutral if no data

        return {
            'total': round(total_similarity, 4),
            'factors': factors
        }

    async def _get_ic_score(self, ticker: str) -> Optional[float]:
        """Get latest IC Score for a ticker."""
        try:
            query = text("""
                SELECT overall_score
                FROM ic_scores
                WHERE ticker = :ticker
                ORDER BY date DESC
                LIMIT 1
            """)

            result = await self.session.execute(query, {"ticker": ticker})
            row = result.fetchone()

            if row and row[0]:
                return float(row[0])

            return None

        except Exception as e:
            logger.error(f"Error fetching IC Score for {ticker}: {e}")
            return None

    async def _get_sector_rank(
        self,
        ticker: str,
        sector: str,
        ic_score: Optional[float]
    ) -> Tuple[Optional[int], Optional[int]]:
        """Get stock's rank within its sector by IC Score."""
        if ic_score is None:
            return None, None

        try:
            query = text("""
                WITH sector_scores AS (
                    SELECT DISTINCT ON (t.symbol)
                        t.symbol,
                        ic.overall_score
                    FROM tickers t
                    JOIN ic_scores ic ON ic.ticker = t.symbol
                    WHERE t.sector = :sector
                      AND t.active = true
                    ORDER BY t.symbol, ic.date DESC
                )
                SELECT
                    (SELECT COUNT(*) FROM sector_scores WHERE overall_score > :score) + 1 as rank,
                    (SELECT COUNT(*) FROM sector_scores) as total
            """)

            result = await self.session.execute(
                query, {"sector": sector, "score": ic_score}
            )
            row = result.fetchone()

            if row:
                return row[0], row[1]

            return None, None

        except Exception as e:
            logger.error(f"Error getting sector rank: {e}")
            return None, None

    async def store_peers(
        self,
        ticker: str,
        peers: List[PeerStock]
    ) -> bool:
        """Store peer relationships in database."""
        try:
            for peer in peers:
                query = text("""
                    INSERT INTO stock_peers (
                        ticker, peer_ticker, similarity_score,
                        similarity_factors, calculated_at
                    ) VALUES (
                        :ticker, :peer_ticker, :similarity_score,
                        :similarity_factors, CURRENT_DATE
                    )
                    ON CONFLICT (ticker, peer_ticker, calculated_at)
                    DO UPDATE SET
                        similarity_score = EXCLUDED.similarity_score,
                        similarity_factors = EXCLUDED.similarity_factors
                """)

                await self.session.execute(query, {
                    "ticker": ticker,
                    "peer_ticker": peer.ticker,
                    "similarity_score": peer.similarity_score,
                    "similarity_factors": peer.similarity_factors
                })

            return True

        except Exception as e:
            logger.error(f"Error storing peers for {ticker}: {e}")
            return False
