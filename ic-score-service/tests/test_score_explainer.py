"""Unit tests for Score Explainer service."""

import pytest
from unittest.mock import AsyncMock, MagicMock
from datetime import date

import sys
from pathlib import Path
sys.path.insert(0, str(Path(__file__).parent.parent))

from pipelines.utils.score_explainer import (
    ScoreExplainer,
    ScoreChangeExplanation,
    ScoreChangeReason,
    GranularConfidence,
    FactorDataStatus,
)


class TestScoreExplainer:
    """Test cases for ScoreExplainer."""

    @pytest.fixture
    def mock_session(self):
        """Create a mock database session."""
        session = AsyncMock()
        return session

    @pytest.fixture
    def explainer(self, mock_session):
        """Create an explainer instance with mocked session."""
        return ScoreExplainer(mock_session)

    # ==================
    # Explanation Generation Tests
    # ==================

    @pytest.mark.asyncio
    async def test_explain_change_no_previous(self, explainer):
        """Test explanation when no previous score exists."""
        explainer._get_previous_scores = AsyncMock(return_value=None)
        explainer.get_granular_confidence = AsyncMock(return_value=GranularConfidence(
            level='High',
            percentage=90.0,
            factors={},
            warnings=[]
        ))

        current_scores = {
            'overall_score': 75.0,
            'value_score': 80.0,
            'growth_score': 70.0,
            'profitability_score': 75.0,
        }

        result = await explainer.explain_change('AAPL', current_scores)

        assert result is not None
        assert isinstance(result, ScoreChangeExplanation)
        assert result.current_score == 75.0
        assert result.delta == 0  # No previous score

    @pytest.mark.asyncio
    async def test_explain_change_with_previous(self, explainer):
        """Test explanation with previous score comparison."""
        previous_scores = {
            'overall_score': 70.0,
            'value_score': 75.0,
            'growth_score': 65.0,
            'profitability_score': 70.0,
        }
        explainer._get_previous_scores = AsyncMock(return_value=previous_scores)
        explainer.get_granular_confidence = AsyncMock(return_value=GranularConfidence(
            level='High',
            percentage=90.0,
            factors={},
            warnings=[]
        ))

        current_scores = {
            'overall_score': 78.0,
            'value_score': 85.0,  # +10 change
            'growth_score': 72.0,  # +7 change
            'profitability_score': 75.0,  # +5 change
        }

        result = await explainer.explain_change('AAPL', current_scores)

        assert result.delta == 8.0
        assert len(result.reasons) > 0

    @pytest.mark.asyncio
    async def test_explain_change_filters_insignificant(self, explainer):
        """Test that small changes are filtered out."""
        previous_scores = {
            'overall_score': 75.0,
            'value_score': 75.0,
            'growth_score': 75.0,
            'profitability_score': 75.0,
        }
        explainer._get_previous_scores = AsyncMock(return_value=previous_scores)
        explainer.get_granular_confidence = AsyncMock(return_value=GranularConfidence(
            level='High',
            percentage=90.0,
            factors={},
            warnings=[]
        ))

        current_scores = {
            'overall_score': 76.0,
            'value_score': 76.0,  # +1 (insignificant)
            'growth_score': 74.0,  # -1 (insignificant)
            'profitability_score': 77.0,  # +2 (insignificant)
        }

        result = await explainer.explain_change('AAPL', current_scores)

        # Changes < 3 points should not appear in reasons
        assert len(result.reasons) == 0

    # ==================
    # Explanation Text Tests
    # ==================

    def test_get_explanation_positive(self, explainer):
        """Test positive explanation generation."""
        explanation = explainer._get_explanation('value', 10.0)
        assert 'undervalued' in explanation.lower() or 'improved' in explanation.lower()

    def test_get_explanation_negative(self, explainer):
        """Test negative explanation generation."""
        explanation = explainer._get_explanation('value', -10.0)
        assert 'overvalued' in explanation.lower() or 'declined' in explanation.lower()

    def test_get_explanation_unknown_factor(self, explainer):
        """Test explanation for unknown factor."""
        explanation = explainer._get_explanation('unknown_factor', 5.0)
        assert 'improved' in explanation.lower()

    # ==================
    # Summary Generation Tests
    # ==================

    def test_generate_summary_no_change(self, explainer):
        """Test summary for unchanged score."""
        summary = explainer._generate_summary('AAPL', 0.2, [])
        assert 'unchanged' in summary.lower()

    def test_generate_summary_improved(self, explainer):
        """Test summary for improved score."""
        reason = ScoreChangeReason(
            factor='value',
            previous_score=70.0,
            current_score=80.0,
            delta=10.0,
            weight=0.12,
            contribution=1.2,
            explanation='Value improved'
        )
        summary = explainer._generate_summary('AAPL', 5.0, [reason])
        assert 'improved' in summary.lower()

    def test_generate_summary_declined(self, explainer):
        """Test summary for declined score."""
        reason = ScoreChangeReason(
            factor='growth',
            previous_score=80.0,
            current_score=70.0,
            delta=-10.0,
            weight=0.13,
            contribution=-1.3,
            explanation='Growth declined'
        )
        summary = explainer._generate_summary('AAPL', -5.0, [reason])
        assert 'declined' in summary.lower()

    def test_generate_summary_significant_change(self, explainer):
        """Test summary for significant change."""
        reason = ScoreChangeReason(
            factor='momentum',
            previous_score=50.0,
            current_score=80.0,
            delta=30.0,
            weight=0.10,
            contribution=3.0,
            explanation='Momentum strengthened'
        )
        summary = explainer._generate_summary('AAPL', 10.0, [reason])
        assert 'significantly' in summary.lower()


class TestGranularConfidence:
    """Test GranularConfidence calculation."""

    @pytest.fixture
    def explainer(self):
        return ScoreExplainer(AsyncMock())

    @pytest.mark.asyncio
    async def test_high_confidence(self, explainer):
        """Test high confidence when most factors available."""
        explainer._get_factor_freshness = AsyncMock(return_value={
            'status': 'fresh',
            'days': 1,
            'count': 100,
            'warning': None
        })

        # All factors have scores
        current_scores = {
            'value_score': 75.0,
            'growth_score': 70.0,
            'profitability_score': 80.0,
            'financial_health_score': 85.0,
            'momentum_score': 65.0,
            'technical_score': 60.0,
            'analyst_consensus_score': 75.0,
            'insider_activity_score': 70.0,
            'institutional_score': 72.0,
            'news_sentiment_score': 68.0,
            'earnings_revisions_score': 78.0,
            'historical_value_score': 74.0,
            'dividend_quality_score': 80.0,
        }

        result = await explainer.get_granular_confidence('AAPL', current_scores)

        assert result.level == 'High'
        assert result.percentage >= 90.0

    @pytest.mark.asyncio
    async def test_medium_confidence(self, explainer):
        """Test medium confidence when some factors missing."""
        explainer._get_factor_freshness = AsyncMock(return_value={
            'status': 'fresh',
            'days': 1,
            'count': 50,
            'warning': None
        })

        # Only some factors have scores
        current_scores = {
            'value_score': 75.0,
            'growth_score': 70.0,
            'profitability_score': 80.0,
            'financial_health_score': 85.0,
            'momentum_score': 65.0,
            'technical_score': 60.0,
            # Missing: analyst_consensus, insider_activity, institutional
            # news_sentiment, earnings_revisions, historical_value, dividend_quality
        }

        result = await explainer.get_granular_confidence('AAPL', current_scores)

        # 6/13 factors = ~46%, but it depends on exact threshold
        assert result.level in ['Low', 'Medium']

    @pytest.mark.asyncio
    async def test_stale_data_warning(self, explainer):
        """Test warning for stale data."""
        explainer._get_factor_freshness = AsyncMock(return_value={
            'status': 'stale',
            'days': 45,
            'count': 10,
            'warning': 'Data is 45 days old'
        })

        current_scores = {
            'value_score': 75.0,
            'growth_score': None,
        }

        result = await explainer.get_granular_confidence('AAPL', current_scores)

        assert len(result.warnings) > 0


class TestFactorDataStatus:
    """Test FactorDataStatus creation."""

    def test_available_fresh_data(self):
        """Test status for fresh available data."""
        status = FactorDataStatus(
            available=True,
            freshness='fresh',
            freshness_days=1,
            count=100,
            warning=None,
            reason=None
        )

        assert status.available is True
        assert status.freshness == 'fresh'
        assert status.warning is None

    def test_missing_data(self):
        """Test status for missing data."""
        status = FactorDataStatus(
            available=False,
            freshness='missing',
            freshness_days=None,
            count=0,
            warning=None,
            reason='No analyst coverage'
        )

        assert status.available is False
        assert status.reason == 'No analyst coverage'


class TestScoreExplainerEdgeCases:
    """Test edge cases and boundary conditions."""

    @pytest.fixture
    def explainer(self):
        return ScoreExplainer(AsyncMock())

    def test_missing_reason_for_unknown_factor(self, explainer):
        """Test missing reason for unknown factor."""
        reason = explainer._get_missing_reason('unknown_factor')
        assert reason == 'Data not available'

    def test_missing_reason_for_known_factors(self, explainer):
        """Test missing reasons for known factors."""
        # These are the factors with custom reasons in _get_missing_reason
        known_factors = [
            'value', 'growth', 'profitability', 'financial_health',
            'momentum', 'technical', 'analyst_consensus', 'insider_activity',
            'institutional', 'news_sentiment',
        ]

        for factor in known_factors:
            reason = explainer._get_missing_reason(factor)
            assert reason != 'Data not available', (
                f"Factor '{factor}' should have a custom reason"
            )
            assert len(reason) > 0

        # These factors don't have custom reasons and use the default
        default_factors = [
            'earnings_revisions', 'historical_value', 'dividend_quality'
        ]
        for factor in default_factors:
            reason = explainer._get_missing_reason(factor)
            assert reason == 'Data not available'

    @pytest.mark.asyncio
    async def test_reasons_sorted_by_contribution(self, explainer):
        """Test that reasons are sorted by absolute contribution."""
        previous_scores = {
            'overall_score': 50.0,
            'value_score': 50.0,
            'growth_score': 50.0,
            'momentum_score': 50.0,
        }
        explainer._get_previous_scores = AsyncMock(return_value=previous_scores)
        explainer.get_granular_confidence = AsyncMock(return_value=GranularConfidence(
            level='High',
            percentage=90.0,
            factors={},
            warnings=[]
        ))

        current_scores = {
            'overall_score': 70.0,
            'value_score': 55.0,    # +5 (small)
            'growth_score': 70.0,   # +20 (large)
            'momentum_score': 60.0,  # +10 (medium)
        }

        result = await explainer.explain_change('AAPL', current_scores)

        # Growth has largest change, should be first if reported
        if len(result.reasons) >= 2:
            assert abs(result.reasons[0].contribution) >= abs(result.reasons[1].contribution)

    @pytest.mark.asyncio
    async def test_top_5_reasons_limit(self, explainer):
        """Test that only top 5 reasons are returned."""
        previous_scores = {'overall_score': 50.0}
        for factor in explainer.FACTOR_WEIGHTS.keys():
            previous_scores[f'{factor}_score'] = 50.0

        explainer._get_previous_scores = AsyncMock(return_value=previous_scores)
        explainer.get_granular_confidence = AsyncMock(return_value=GranularConfidence(
            level='High',
            percentage=90.0,
            factors={},
            warnings=[]
        ))

        current_scores = {'overall_score': 80.0}
        for i, factor in enumerate(explainer.FACTOR_WEIGHTS.keys()):
            # Give each factor a significant change
            current_scores[f'{factor}_score'] = 50.0 + (i + 1) * 5

        result = await explainer.explain_change('AAPL', current_scores)

        assert len(result.reasons) <= 5
