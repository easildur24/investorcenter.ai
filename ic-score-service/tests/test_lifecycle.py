"""Tests for lifecycle classification.

These tests verify the company lifecycle classification
and weight adjustment functionality introduced in IC Score v2.1.
"""
import pytest
from unittest.mock import AsyncMock

# Add parent directory to path for imports
import sys
from pathlib import Path
sys.path.insert(0, str(Path(__file__).parent.parent))

from pipelines.utils.lifecycle import (
    LifecycleClassifier,
    LifecycleStage,
    ClassificationResult,
    get_lifecycle_stage_description,
)


class TestLifecycleStage:
    """Tests for LifecycleStage enum."""

    def test_all_stages_defined(self):
        """Verify all expected lifecycle stages exist."""
        assert LifecycleStage.HYPERGROWTH.value == "hypergrowth"
        assert LifecycleStage.GROWTH.value == "growth"
        assert LifecycleStage.MATURE.value == "mature"
        assert LifecycleStage.VALUE.value == "value"
        assert LifecycleStage.TURNAROUND.value == "turnaround"

    def test_stage_count(self):
        """Verify correct number of stages."""
        assert len(LifecycleStage) == 5


class TestLifecycleClassifier:
    """Tests for LifecycleClassifier."""

    @pytest.fixture
    def classifier(self):
        """Create classifier instance without database."""
        return LifecycleClassifier(session=None)

    def test_base_weights_sum_to_one(self, classifier):
        """Verify base weights sum to approximately 1.0."""
        total = sum(classifier.BASE_WEIGHTS.values())
        assert abs(total - 1.0) < 0.01

    def test_all_stages_have_weight_adjustments(self, classifier):
        """Verify all stages have weight adjustment configurations."""
        for stage in LifecycleStage:
            assert stage in classifier.WEIGHT_ADJUSTMENTS

    # =========================================================================
    # Classification Tests
    # =========================================================================

    def test_classify_hypergrowth(self, classifier):
        """Test classification of hypergrowth company."""
        data = {
            'revenue_growth_yoy': 75.0,  # >50% growth
            'net_margin': 5.0,
            'pe_ratio': 100,
        }
        result = classifier.classify(data)

        assert result.stage == LifecycleStage.HYPERGROWTH
        assert result.confidence > 0.5

    def test_classify_hypergrowth_extreme(self, classifier):
        """Test classification of extreme hypergrowth company."""
        data = {
            'revenue_growth_yoy': 150.0,  # Very high growth
            'net_margin': -10.0,  # Unprofitable (common for hypergrowth)
            'pe_ratio': None,  # Often N/A for unprofitable
        }
        result = classifier.classify(data)

        assert result.stage == LifecycleStage.HYPERGROWTH
        assert result.confidence > 0.8

    def test_classify_growth(self, classifier):
        """Test classification of growth company."""
        data = {
            'revenue_growth_yoy': 35.0,  # 20-50% growth
            'net_margin': 10.0,
            'pe_ratio': 40,
        }
        result = classifier.classify(data)

        assert result.stage == LifecycleStage.GROWTH
        assert result.confidence > 0.5

    def test_classify_mature(self, classifier):
        """Test classification of mature company."""
        data = {
            'revenue_growth_yoy': 5.0,  # Low but positive growth
            'net_margin': 15.0,  # Good profitability
            'pe_ratio': 20,
        }
        result = classifier.classify(data)

        assert result.stage == LifecycleStage.MATURE
        assert result.confidence > 0.5

    def test_classify_value(self, classifier):
        """Test classification of value company."""
        data = {
            'revenue_growth_yoy': 3.0,  # Low growth
            'net_margin': 12.0,  # Good margin > 5%
            'pe_ratio': 8,  # Low P/E < 12
        }
        result = classifier.classify(data)

        assert result.stage == LifecycleStage.VALUE
        assert result.confidence > 0.5

    def test_classify_turnaround(self, classifier):
        """Test classification of turnaround company."""
        data = {
            'revenue_growth_yoy': -15.0,  # Declining revenue < -5%
            'net_margin': 2.0,
            'pe_ratio': 15,
        }
        result = classifier.classify(data)

        assert result.stage == LifecycleStage.TURNAROUND
        assert result.confidence > 0.5

    def test_classify_with_none_values(self, classifier):
        """Test classification handles None values gracefully."""
        data = {
            'revenue_growth_yoy': None,
            'net_margin': None,
            'pe_ratio': None,
        }
        result = classifier.classify(data)

        # Should default to MATURE with None values
        assert result.stage == LifecycleStage.MATURE

    def test_classify_with_empty_dict(self, classifier):
        """Test classification handles empty dict gracefully."""
        result = classifier.classify({})

        # Should default to MATURE
        assert result.stage == LifecycleStage.MATURE

    # =========================================================================
    # Weight Adjustment Tests
    # =========================================================================

    def test_adjust_weights_hypergrowth(self, classifier):
        """Test weight adjustments for hypergrowth stage."""
        adjusted = classifier.adjust_weights(
            classifier.BASE_WEIGHTS,
            LifecycleStage.HYPERGROWTH
        )

        # Growth should be emphasized
        assert adjusted['growth'] > classifier.BASE_WEIGHTS['growth']

        # Value should be de-emphasized
        assert adjusted['value'] < classifier.BASE_WEIGHTS['value']

        # Weights should still sum to 1.0
        total = sum(adjusted.values())
        assert abs(total - 1.0) < 0.01

    def test_adjust_weights_value(self, classifier):
        """Test weight adjustments for value stage."""
        adjusted = classifier.adjust_weights(
            classifier.BASE_WEIGHTS,
            LifecycleStage.VALUE
        )

        # Value should be emphasized
        assert adjusted['value'] > classifier.BASE_WEIGHTS['value']

        # Growth should be de-emphasized
        assert adjusted['growth'] < classifier.BASE_WEIGHTS['growth']

        # Weights should still sum to 1.0
        total = sum(adjusted.values())
        assert abs(total - 1.0) < 0.01

    def test_adjust_weights_turnaround(self, classifier):
        """Test weight adjustments for turnaround stage."""
        adjusted = classifier.adjust_weights(
            classifier.BASE_WEIGHTS,
            LifecycleStage.TURNAROUND
        )

        # Financial health should be emphasized (survival focus)
        assert adjusted['financial_health'] > classifier.BASE_WEIGHTS['financial_health']

        # Weights should still sum to 1.0
        total = sum(adjusted.values())
        assert abs(total - 1.0) < 0.01

    def test_adjust_weights_normalization(self, classifier):
        """Test that adjusted weights are properly normalized."""
        for stage in LifecycleStage:
            adjusted = classifier.adjust_weights(classifier.BASE_WEIGHTS, stage)
            total = sum(adjusted.values())
            assert abs(total - 1.0) < 0.01, f"Weights don't sum to 1.0 for {stage}"

    # =========================================================================
    # Classification Result Tests
    # =========================================================================

    def test_classification_result_structure(self, classifier):
        """Test ClassificationResult contains expected fields."""
        data = {'revenue_growth_yoy': 30.0, 'net_margin': 10.0, 'pe_ratio': 25}
        result = classifier.classify(data)

        assert hasattr(result, 'stage')
        assert hasattr(result, 'confidence')
        assert hasattr(result, 'metrics_used')
        assert hasattr(result, 'adjusted_weights')

        assert isinstance(result.stage, LifecycleStage)
        assert 0 <= result.confidence <= 1
        assert isinstance(result.metrics_used, dict)
        assert isinstance(result.adjusted_weights, dict)

    def test_classification_result_metrics_recorded(self, classifier):
        """Test that input metrics are recorded in result."""
        data = {'revenue_growth_yoy': 30.0, 'net_margin': 10.0, 'pe_ratio': 25}
        result = classifier.classify(data)

        assert result.metrics_used['revenue_growth_yoy'] == 30.0
        assert result.metrics_used['net_margin'] == 10.0
        assert result.metrics_used['pe_ratio'] == 25


class TestLifecycleDescriptions:
    """Tests for lifecycle stage descriptions."""

    def test_all_stages_have_descriptions(self):
        """Verify all stages have descriptions."""
        for stage in LifecycleStage:
            description = get_lifecycle_stage_description(stage)
            assert description is not None
            assert len(description) > 20  # Non-trivial description

    def test_hypergrowth_description_content(self):
        """Test hypergrowth description mentions key concepts."""
        desc = get_lifecycle_stage_description(LifecycleStage.HYPERGROWTH)
        assert "growth" in desc.lower()
        assert "50%" in desc or ">50" in desc

    def test_value_description_content(self):
        """Test value description mentions key concepts."""
        desc = get_lifecycle_stage_description(LifecycleStage.VALUE)
        assert "value" in desc.lower() or "valuation" in desc.lower()


class TestEdgeCases:
    """Tests for edge cases and boundary conditions."""

    @pytest.fixture
    def classifier(self):
        return LifecycleClassifier(session=None)

    def test_boundary_hypergrowth(self, classifier):
        """Test classification at hypergrowth boundary (50%)."""
        # Just above boundary
        data = {'revenue_growth_yoy': 51.0, 'net_margin': 5.0, 'pe_ratio': 50}
        result = classifier.classify(data)
        assert result.stage == LifecycleStage.HYPERGROWTH

        # Just below boundary
        data = {'revenue_growth_yoy': 49.0, 'net_margin': 5.0, 'pe_ratio': 50}
        result = classifier.classify(data)
        assert result.stage == LifecycleStage.GROWTH

    def test_boundary_growth(self, classifier):
        """Test classification at growth boundary (20%)."""
        # Just above boundary
        data = {'revenue_growth_yoy': 21.0, 'net_margin': 5.0, 'pe_ratio': 50}
        result = classifier.classify(data)
        assert result.stage == LifecycleStage.GROWTH

        # Just below boundary
        data = {'revenue_growth_yoy': 19.0, 'net_margin': 5.0, 'pe_ratio': 50}
        result = classifier.classify(data)
        assert result.stage == LifecycleStage.MATURE  # Falls through to mature

    def test_boundary_turnaround(self, classifier):
        """Test classification at turnaround boundary (-5%)."""
        # Just below boundary (declining)
        data = {'revenue_growth_yoy': -6.0, 'net_margin': 5.0, 'pe_ratio': 15}
        result = classifier.classify(data)
        assert result.stage == LifecycleStage.TURNAROUND

        # Just above boundary (not quite turnaround)
        data = {'revenue_growth_yoy': -4.0, 'net_margin': 10.0, 'pe_ratio': 10}
        result = classifier.classify(data)
        assert result.stage == LifecycleStage.VALUE  # Low P/E with margins

    def test_conflicting_signals(self, classifier):
        """Test classification with conflicting signals."""
        # Low P/E (value signal) but negative margins (not classic value)
        data = {'revenue_growth_yoy': 5.0, 'net_margin': -2.0, 'pe_ratio': 8}
        result = classifier.classify(data)
        # Should fall through to mature since margin < 5%
        assert result.stage == LifecycleStage.MATURE

    def test_extreme_values(self, classifier):
        """Test classification with extreme metric values."""
        # Extremely high growth
        data = {'revenue_growth_yoy': 500.0, 'net_margin': 0, 'pe_ratio': 500}
        result = classifier.classify(data)
        assert result.stage == LifecycleStage.HYPERGROWTH
        assert result.confidence > 0.9

        # Extremely negative growth
        data = {'revenue_growth_yoy': -80.0, 'net_margin': -50, 'pe_ratio': None}
        result = classifier.classify(data)
        assert result.stage == LifecycleStage.TURNAROUND


class TestRealWorldExamples:
    """Tests using real-world company examples."""

    @pytest.fixture
    def classifier(self):
        return LifecycleClassifier(session=None)

    def test_nvidia_like_company(self, classifier):
        """Test classification for NVIDIA-like hypergrowth company."""
        data = {
            'revenue_growth_yoy': 122.0,  # Massive AI-driven growth
            'net_margin': 55.0,  # High margins
            'pe_ratio': 60,  # Premium valuation
        }
        result = classifier.classify(data)
        assert result.stage == LifecycleStage.HYPERGROWTH

    def test_jpm_like_company(self, classifier):
        """Test classification for JPM-like mature bank."""
        data = {
            'revenue_growth_yoy': 8.0,
            'net_margin': 25.0,
            'pe_ratio': 12,
        }
        result = classifier.classify(data)
        # Could be VALUE (low P/E) or MATURE depending on margin threshold
        assert result.stage in [LifecycleStage.VALUE, LifecycleStage.MATURE]

    def test_walmart_like_company(self, classifier):
        """Test classification for Walmart-like mature retailer."""
        data = {
            'revenue_growth_yoy': 5.0,
            'net_margin': 2.5,  # Low retail margins
            'pe_ratio': 28,
        }
        result = classifier.classify(data)
        assert result.stage == LifecycleStage.MATURE

    def test_startup_like_company(self, classifier):
        """Test classification for unprofitable growth startup."""
        data = {
            'revenue_growth_yoy': 45.0,
            'net_margin': -15.0,  # Still losing money
            'pe_ratio': None,  # N/A due to losses
        }
        result = classifier.classify(data)
        assert result.stage == LifecycleStage.GROWTH


if __name__ == "__main__":
    pytest.main([__file__, "-v"])
