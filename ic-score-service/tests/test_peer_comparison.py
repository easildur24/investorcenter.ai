"""Unit tests for Peer Comparison service."""

import pytest
from unittest.mock import AsyncMock, MagicMock
from datetime import date

import sys
from pathlib import Path
sys.path.insert(0, str(Path(__file__).parent.parent))

from pipelines.utils.peer_comparison import (
    PeerComparisonService,
    PeerComparisonResult,
    PeerStock,
)


class TestPeerComparisonService:
    """Test cases for PeerComparisonService."""

    @pytest.fixture
    def mock_session(self):
        """Create a mock database session."""
        session = AsyncMock()
        return session

    @pytest.fixture
    def service(self, mock_session):
        """Create a service instance with mocked session."""
        return PeerComparisonService(mock_session)

    # ==================
    # Market Cap Filter Tests
    # ==================

    def test_market_cap_in_range(self, service):
        """Test market cap range calculation."""
        # For a $100B company, range should be $25B-$400B
        base_market_cap = 100_000_000_000

        min_cap = base_market_cap * service.MARKET_CAP_MIN_RATIO
        max_cap = base_market_cap * service.MARKET_CAP_MAX_RATIO

        assert min_cap == 25_000_000_000
        assert max_cap == 400_000_000_000

    # ==================
    # Similarity Score Tests
    # ==================

    def test_calculate_similarity_identical(self, service):
        """Test similarity calculation for identical metrics."""
        stock1 = {
            'market_cap': 100_000_000_000,
            'revenue_growth': 10.0,
            'net_margin': 20.0,
            'pe_ratio': 25.0,
            'beta': 1.2,
        }
        stock2 = stock1.copy()

        similarity = service._calculate_similarity(stock1, stock2)

        # Identical stocks should have perfect similarity
        assert similarity == 1.0

    def test_calculate_similarity_different(self, service):
        """Test similarity calculation for different metrics."""
        stock1 = {
            'market_cap': 100_000_000_000,
            'revenue_growth': 10.0,
            'net_margin': 20.0,
            'pe_ratio': 25.0,
            'beta': 1.2,
        }
        stock2 = {
            'market_cap': 50_000_000_000,  # Different
            'revenue_growth': 30.0,  # Different
            'net_margin': 5.0,  # Different
            'pe_ratio': 50.0,  # Different
            'beta': 2.0,  # Different
        }

        similarity = service._calculate_similarity(stock1, stock2)

        # Different stocks should have lower similarity
        assert similarity < 1.0
        assert similarity >= 0.0

    def test_calculate_similarity_partial_data(self, service):
        """Test similarity calculation with missing metrics."""
        stock1 = {
            'market_cap': 100_000_000_000,
            'revenue_growth': 10.0,
            'net_margin': None,  # Missing
            'pe_ratio': 25.0,
            'beta': None,  # Missing
        }
        stock2 = {
            'market_cap': 80_000_000_000,
            'revenue_growth': 12.0,
            'net_margin': 15.0,
            'pe_ratio': 28.0,
            'beta': 1.1,
        }

        similarity = service._calculate_similarity(stock1, stock2)

        # Should still compute similarity for available metrics
        assert 0 <= similarity <= 1.0

    # ==================
    # Get Peers Tests
    # ==================

    @pytest.mark.asyncio
    async def test_get_peers_returns_result(self, service, mock_session):
        """Test get_peers returns valid result."""
        # Mock stock data query
        stock_data = MagicMock()
        stock_data.fetchone.return_value = (
            'AAPL', 'Technology', 3_000_000_000_000, 10.0, 25.0, 30.0, 1.1
        )
        mock_session.execute = AsyncMock(return_value=stock_data)

        # Mock peer candidates query
        peer_data = MagicMock()
        peer_data.fetchall.return_value = [
            ('MSFT', 'Microsoft', 2_800_000_000_000, 12.0, 27.0, 35.0, 1.0, 85.0),
            ('GOOGL', 'Alphabet', 2_000_000_000_000, 15.0, 24.0, 28.0, 1.15, 78.0),
        ]

        # Set up mock to return different results for different queries
        async def mock_execute(query, params=None):
            if 'peer' in str(query).lower() or 'sector' in str(query).lower():
                return peer_data
            return stock_data

        mock_session.execute = AsyncMock(side_effect=mock_execute)

        service._get_stock_data = AsyncMock(return_value={
            'ticker': 'AAPL',
            'sector': 'Technology',
            'market_cap': 3_000_000_000_000,
            'revenue_growth': 10.0,
            'net_margin': 25.0,
            'pe_ratio': 30.0,
            'beta': 1.1,
        })

        service._get_peer_candidates = AsyncMock(return_value=[
            {
                'ticker': 'MSFT',
                'company_name': 'Microsoft',
                'market_cap': 2_800_000_000_000,
                'revenue_growth': 12.0,
                'net_margin': 27.0,
                'pe_ratio': 35.0,
                'beta': 1.0,
                'ic_score': 85.0,
            },
            {
                'ticker': 'GOOGL',
                'company_name': 'Alphabet',
                'market_cap': 2_000_000_000_000,
                'revenue_growth': 15.0,
                'net_margin': 24.0,
                'pe_ratio': 28.0,
                'beta': 1.15,
                'ic_score': 78.0,
            },
        ])

        service._get_sector_rank = AsyncMock(return_value=(5, 50))

        result = await service.get_peers('AAPL', limit=5)

        assert result is not None
        assert isinstance(result, PeerComparisonResult)

    @pytest.mark.asyncio
    async def test_get_peers_no_candidates(self, service):
        """Test get_peers when no peer candidates found."""
        service._get_stock_data = AsyncMock(return_value={
            'ticker': 'RARE',
            'sector': 'Unique Sector',
            'market_cap': 1_000_000,
        })
        service._get_peer_candidates = AsyncMock(return_value=[])

        result = await service.get_peers('RARE', limit=5)

        assert result is None or len(result.peers) == 0

    @pytest.mark.asyncio
    async def test_get_peers_missing_stock_data(self, service):
        """Test get_peers when stock data not found."""
        service._get_stock_data = AsyncMock(return_value=None)

        result = await service.get_peers('UNKNOWN', limit=5)

        assert result is None


class TestPeerComparisonEdgeCases:
    """Test edge cases and boundary conditions."""

    @pytest.fixture
    def service(self):
        return PeerComparisonService(AsyncMock())

    def test_similarity_weights_sum_to_one(self, service):
        """Test that similarity weights sum to 1.0."""
        total_weight = sum(service.SIMILARITY_WEIGHTS.values())
        assert abs(total_weight - 1.0) < 0.01  # Allow small floating point error

    def test_market_cap_ratio_valid(self, service):
        """Test market cap ratio boundaries are valid."""
        assert service.MARKET_CAP_MIN_RATIO > 0
        assert service.MARKET_CAP_MIN_RATIO < 1.0
        assert service.MARKET_CAP_MAX_RATIO > 1.0

    def test_similarity_with_zero_values(self, service):
        """Test similarity calculation handles zero values."""
        stock1 = {
            'market_cap': 100_000_000_000,
            'revenue_growth': 0.0,
            'net_margin': 0.0,
            'pe_ratio': 0.0,
            'beta': 0.0,
        }
        stock2 = {
            'market_cap': 100_000_000_000,
            'revenue_growth': 10.0,
            'net_margin': 20.0,
            'pe_ratio': 25.0,
            'beta': 1.0,
        }

        # Should not raise exception
        similarity = service._calculate_similarity(stock1, stock2)
        assert 0 <= similarity <= 1.0

    def test_similarity_with_negative_growth(self, service):
        """Test similarity calculation handles negative growth."""
        stock1 = {
            'market_cap': 100_000_000_000,
            'revenue_growth': -10.0,  # Negative growth
            'net_margin': 15.0,
            'pe_ratio': 20.0,
            'beta': 1.2,
        }
        stock2 = {
            'market_cap': 100_000_000_000,
            'revenue_growth': -15.0,  # Also negative
            'net_margin': 12.0,
            'pe_ratio': 18.0,
            'beta': 1.3,
        }

        similarity = service._calculate_similarity(stock1, stock2)
        assert 0 <= similarity <= 1.0
