"""Tests for lifecycle classification.

Comprehensive tests covering the company lifecycle classification,
weight adjustment functionality, _clamp_numeric utility, boundary
conditions, confidence scoring, and real-world company scenarios.
"""
import pytest
from decimal import Decimal
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


# =====================================================================
# TestLifecycleStage enum
# =====================================================================


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

    def test_stages_constructable_from_string(self):
        """Each stage should be constructable from its string value."""
        for stage in LifecycleStage:
            assert LifecycleStage(stage.value) == stage

    def test_invalid_stage_raises(self):
        """Invalid stage value should raise ValueError."""
        with pytest.raises(ValueError):
            LifecycleStage("nonexistent")


# =====================================================================
# TestLifecycleClassifier
# =====================================================================


class TestLifecycleClassifier:
    """Tests for LifecycleClassifier."""

    @pytest.fixture
    def classifier(self):
        """Create classifier instance without database."""
        return LifecycleClassifier(session=None)

    # =========================================================================
    # Base Weight Tests
    # =========================================================================

    def test_base_weights_sum_to_one(self, classifier):
        """Verify base weights sum to approximately 1.0."""
        total = sum(classifier.BASE_WEIGHTS.values())
        assert abs(total - 1.0) < 0.01

    def test_base_weights_all_positive(self, classifier):
        """All base weights should be positive."""
        for factor, weight in classifier.BASE_WEIGHTS.items():
            assert weight > 0, f"Weight for {factor} should be positive"

    def test_all_stages_have_weight_adjustments(self, classifier):
        """Verify all stages have weight adjustment configurations."""
        for stage in LifecycleStage:
            assert stage in classifier.WEIGHT_ADJUSTMENTS

    def test_base_weights_have_expected_factors(self, classifier):
        """Check that the expected factor names are present."""
        expected = {
            'profitability', 'financial_health', 'growth',
            'value', 'intrinsic_value', 'historical_value',
            'momentum', 'smart_money', 'earnings_revisions',
            'technical',
        }
        assert set(classifier.BASE_WEIGHTS.keys()) == expected

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
            'revenue_growth_yoy': 150.0,
            'net_margin': -10.0,
            'pe_ratio': None,
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
            'revenue_growth_yoy': 5.0,
            'net_margin': 15.0,
            'pe_ratio': 20,
        }
        result = classifier.classify(data)

        assert result.stage == LifecycleStage.MATURE
        assert result.confidence > 0.5

    def test_classify_mature_high_confidence(self, classifier):
        """Mature with typical metrics should have higher confidence."""
        data = {
            'revenue_growth_yoy': 8.0,  # 0 < 8 < 15
            'net_margin': 10.0,         # > 0
            'pe_ratio': 18,
        }
        result = classifier.classify(data)
        assert result.stage == LifecycleStage.MATURE
        assert result.confidence == 0.8

    def test_classify_value(self, classifier):
        """Test classification of value company."""
        data = {
            'revenue_growth_yoy': 3.0,
            'net_margin': 12.0,   # > 5%
            'pe_ratio': 8,        # < 12
        }
        result = classifier.classify(data)

        assert result.stage == LifecycleStage.VALUE
        assert result.confidence > 0.5

    def test_classify_turnaround(self, classifier):
        """Test classification of turnaround company."""
        data = {
            'revenue_growth_yoy': -15.0,  # < -5%
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
        assert result.stage == LifecycleStage.MATURE

    def test_classify_with_empty_dict(self, classifier):
        """Test classification handles empty dict gracefully."""
        result = classifier.classify({})
        assert result.stage == LifecycleStage.MATURE

    def test_classify_with_zero_values(self, classifier):
        """Zero values for all metrics."""
        data = {
            'revenue_growth_yoy': 0,
            'net_margin': 0,
            'pe_ratio': 0,
        }
        result = classifier.classify(data)
        # 0 revenue growth -> not hypergrowth/growth/turnaround
        # 0 pe_ratio -> treated as 20 (default), not value
        # Falls to MATURE
        assert result.stage == LifecycleStage.MATURE

    def test_classify_with_decimal_values(self, classifier):
        """Decimal values should be converted to float."""
        data = {
            'revenue_growth_yoy': Decimal('60.5'),
            'net_margin': Decimal('12.0'),
            'pe_ratio': Decimal('30'),
        }
        result = classifier.classify(data)
        assert result.stage == LifecycleStage.HYPERGROWTH

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
        adjusted = classifier.adjust_weights(
            classifier.BASE_WEIGHTS,
            LifecycleStage.VALUE
        )
        assert adjusted['value'] > classifier.BASE_WEIGHTS['value']
        assert adjusted['growth'] < classifier.BASE_WEIGHTS['growth']
        total = sum(adjusted.values())
        assert abs(total - 1.0) < 0.01

    def test_adjust_weights_turnaround(self, classifier):
        adjusted = classifier.adjust_weights(
            classifier.BASE_WEIGHTS,
            LifecycleStage.TURNAROUND
        )
        assert adjusted['financial_health'] > classifier.BASE_WEIGHTS['financial_health']
        total = sum(adjusted.values())
        assert abs(total - 1.0) < 0.01

    def test_adjust_weights_growth(self, classifier):
        adjusted = classifier.adjust_weights(
            classifier.BASE_WEIGHTS,
            LifecycleStage.GROWTH
        )
        assert adjusted['growth'] > classifier.BASE_WEIGHTS['growth']
        total = sum(adjusted.values())
        assert abs(total - 1.0) < 0.01

    def test_adjust_weights_mature(self, classifier):
        adjusted = classifier.adjust_weights(
            classifier.BASE_WEIGHTS,
            LifecycleStage.MATURE
        )
        assert adjusted['profitability'] > classifier.BASE_WEIGHTS['profitability']
        total = sum(adjusted.values())
        assert abs(total - 1.0) < 0.01

    def test_adjust_weights_normalization(self, classifier):
        """Test that adjusted weights are properly normalized."""
        for stage in LifecycleStage:
            adjusted = classifier.adjust_weights(
                classifier.BASE_WEIGHTS, stage
            )
            total = sum(adjusted.values())
            assert abs(total - 1.0) < 0.01, (
                f"Weights don't sum to 1.0 for {stage}"
            )

    def test_adjust_weights_all_positive(self, classifier):
        """All adjusted weights should remain positive."""
        for stage in LifecycleStage:
            adjusted = classifier.adjust_weights(
                classifier.BASE_WEIGHTS, stage
            )
            for factor, weight in adjusted.items():
                assert weight > 0, (
                    f"Weight for {factor} in {stage} should be positive"
                )

    def test_adjust_weights_preserves_factors(self, classifier):
        """All base factors should be present in adjusted weights."""
        for stage in LifecycleStage:
            adjusted = classifier.adjust_weights(
                classifier.BASE_WEIGHTS, stage
            )
            assert set(adjusted.keys()) == set(
                classifier.BASE_WEIGHTS.keys()
            )

    def test_adjust_weights_empty_base(self, classifier):
        """Empty base weights should return empty dict."""
        adjusted = classifier.adjust_weights(
            {}, LifecycleStage.MATURE
        )
        assert adjusted == {}

    # =========================================================================
    # Classification Result Tests
    # =========================================================================

    def test_classification_result_structure(self, classifier):
        data = {
            'revenue_growth_yoy': 30.0,
            'net_margin': 10.0,
            'pe_ratio': 25,
        }
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
        data = {
            'revenue_growth_yoy': 30.0,
            'net_margin': 10.0,
            'pe_ratio': 25,
        }
        result = classifier.classify(data)

        assert result.metrics_used['revenue_growth_yoy'] == 30.0
        assert result.metrics_used['net_margin'] == 10.0
        assert result.metrics_used['pe_ratio'] == 25

    def test_classification_adjusted_weights_sum_to_one(self, classifier):
        """Adjusted weights in result should sum to 1.0."""
        data = {'revenue_growth_yoy': 30.0, 'net_margin': 10.0}
        result = classifier.classify(data)
        total = sum(result.adjusted_weights.values())
        assert abs(total - 1.0) < 0.01

    # =========================================================================
    # _clamp_numeric Tests
    # =========================================================================

    def test_clamp_numeric_normal_value(self, classifier):
        assert classifier._clamp_numeric(50.0) == 50.0

    def test_clamp_numeric_none(self, classifier):
        assert classifier._clamp_numeric(None) is None

    def test_clamp_numeric_below_min(self, classifier):
        result = classifier._clamp_numeric(-100.0, min_val=-50, max_val=50)
        assert result == -50

    def test_clamp_numeric_above_max(self, classifier):
        result = classifier._clamp_numeric(100.0, min_val=-50, max_val=50)
        assert result == 50

    def test_clamp_numeric_within_range(self, classifier):
        result = classifier._clamp_numeric(25.0, min_val=-50, max_val=50)
        assert result == 25.0

    def test_clamp_numeric_at_min(self, classifier):
        result = classifier._clamp_numeric(-50.0, min_val=-50, max_val=50)
        assert result == -50.0

    def test_clamp_numeric_at_max(self, classifier):
        result = classifier._clamp_numeric(50.0, min_val=-50, max_val=50)
        assert result == 50.0

    def test_clamp_numeric_extreme_positive_outlier(self, classifier):
        """Values >= 1e10 should return None."""
        assert classifier._clamp_numeric(1e10) is None
        assert classifier._clamp_numeric(1e15) is None

    def test_clamp_numeric_extreme_negative_outlier(self, classifier):
        """Values <= -1e10 should return None."""
        assert classifier._clamp_numeric(-1e10) is None
        assert classifier._clamp_numeric(-1e15) is None

    def test_clamp_numeric_just_under_extreme(self, classifier):
        """Values just under 1e10 should be returned."""
        result = classifier._clamp_numeric(9_999_999_999.0)
        assert result is not None

    def test_clamp_numeric_string_value(self, classifier):
        """Non-numeric strings should return None."""
        assert classifier._clamp_numeric("not a number") is None

    def test_clamp_numeric_string_number(self, classifier):
        """Numeric strings should be converted."""
        result = classifier._clamp_numeric("42.5", min_val=0, max_val=100)
        assert result == 42.5

    def test_clamp_numeric_int(self, classifier):
        """Integer values should work."""
        result = classifier._clamp_numeric(42, min_val=0, max_val=100)
        assert result == 42.0

    def test_clamp_numeric_decimal(self, classifier):
        """Decimal values should be converted."""
        result = classifier._clamp_numeric(
            Decimal('42.5'), min_val=0, max_val=100
        )
        assert result == 42.5

    def test_clamp_numeric_zero(self, classifier):
        result = classifier._clamp_numeric(0)
        assert result == 0

    def test_clamp_numeric_default_bounds(self, classifier):
        """Default min/max of -999999/999999 should allow normal values."""
        result = classifier._clamp_numeric(500000)
        assert result == 500000
        result = classifier._clamp_numeric(-500000)
        assert result == -500000

    def test_clamp_numeric_exceeds_default_max(self, classifier):
        result = classifier._clamp_numeric(1_500_000)
        assert result == 999999

    def test_clamp_numeric_exceeds_default_min(self, classifier):
        result = classifier._clamp_numeric(-1_500_000)
        assert result == -999999


# =====================================================================
# TestLifecycleDescriptions
# =====================================================================


class TestLifecycleDescriptions:
    """Tests for lifecycle stage descriptions."""

    def test_all_stages_have_descriptions(self):
        """Verify all stages have descriptions."""
        for stage in LifecycleStage:
            description = get_lifecycle_stage_description(stage)
            assert description is not None
            assert len(description) > 20

    def test_hypergrowth_description_content(self):
        desc = get_lifecycle_stage_description(LifecycleStage.HYPERGROWTH)
        assert "growth" in desc.lower()
        assert "50%" in desc or ">50" in desc

    def test_growth_description_content(self):
        desc = get_lifecycle_stage_description(LifecycleStage.GROWTH)
        assert "growth" in desc.lower()

    def test_mature_description_content(self):
        desc = get_lifecycle_stage_description(LifecycleStage.MATURE)
        assert "mature" in desc.lower() or "stable" in desc.lower()

    def test_value_description_content(self):
        desc = get_lifecycle_stage_description(LifecycleStage.VALUE)
        assert "value" in desc.lower() or "valuation" in desc.lower()

    def test_turnaround_description_content(self):
        desc = get_lifecycle_stage_description(LifecycleStage.TURNAROUND)
        assert "turnaround" in desc.lower() or "declining" in desc.lower()


# =====================================================================
# TestEdgeCases
# =====================================================================


class TestEdgeCases:
    """Tests for edge cases and boundary conditions."""

    @pytest.fixture
    def classifier(self):
        return LifecycleClassifier(session=None)

    def test_boundary_hypergrowth(self, classifier):
        """Test classification at hypergrowth boundary (50%)."""
        # Just above boundary
        data = {
            'revenue_growth_yoy': 51.0,
            'net_margin': 5.0,
            'pe_ratio': 50,
        }
        result = classifier.classify(data)
        assert result.stage == LifecycleStage.HYPERGROWTH

        # Just below boundary
        data = {
            'revenue_growth_yoy': 49.0,
            'net_margin': 5.0,
            'pe_ratio': 50,
        }
        result = classifier.classify(data)
        assert result.stage == LifecycleStage.GROWTH

    def test_boundary_growth(self, classifier):
        """Test classification at growth boundary (20%)."""
        data = {
            'revenue_growth_yoy': 21.0,
            'net_margin': 5.0,
            'pe_ratio': 50,
        }
        result = classifier.classify(data)
        assert result.stage == LifecycleStage.GROWTH

        data = {
            'revenue_growth_yoy': 19.0,
            'net_margin': 5.0,
            'pe_ratio': 50,
        }
        result = classifier.classify(data)
        assert result.stage == LifecycleStage.MATURE

    def test_boundary_turnaround(self, classifier):
        """Test classification at turnaround boundary (-5%)."""
        data = {
            'revenue_growth_yoy': -6.0,
            'net_margin': 5.0,
            'pe_ratio': 15,
        }
        result = classifier.classify(data)
        assert result.stage == LifecycleStage.TURNAROUND

        data = {
            'revenue_growth_yoy': -4.0,
            'net_margin': 10.0,
            'pe_ratio': 10,
        }
        result = classifier.classify(data)
        assert result.stage == LifecycleStage.VALUE

    def test_boundary_value_pe_threshold(self, classifier):
        """Test value classification at P/E boundary (12)."""
        # P/E just below 12 with good margin -> VALUE
        data = {
            'revenue_growth_yoy': 3.0,
            'net_margin': 10.0,
            'pe_ratio': 11.9,
        }
        result = classifier.classify(data)
        assert result.stage == LifecycleStage.VALUE

        # P/E at exactly 12 -> not value (< 12 required)
        data = {
            'revenue_growth_yoy': 3.0,
            'net_margin': 10.0,
            'pe_ratio': 12.0,
        }
        result = classifier.classify(data)
        assert result.stage == LifecycleStage.MATURE

    def test_boundary_value_margin_threshold(self, classifier):
        """Test value classification at margin boundary (5%)."""
        # Low P/E but margin <= 5% -> not VALUE
        data = {
            'revenue_growth_yoy': 3.0,
            'net_margin': 5.0,
            'pe_ratio': 8,
        }
        result = classifier.classify(data)
        assert result.stage == LifecycleStage.MATURE

        # Margin just above 5% with low P/E -> VALUE
        data = {
            'revenue_growth_yoy': 3.0,
            'net_margin': 5.1,
            'pe_ratio': 8,
        }
        result = classifier.classify(data)
        assert result.stage == LifecycleStage.VALUE

    def test_conflicting_signals(self, classifier):
        """Low P/E but negative margins should not be VALUE."""
        data = {
            'revenue_growth_yoy': 5.0,
            'net_margin': -2.0,
            'pe_ratio': 8,
        }
        result = classifier.classify(data)
        assert result.stage == LifecycleStage.MATURE

    def test_extreme_values(self, classifier):
        """Test classification with extreme metric values."""
        data = {
            'revenue_growth_yoy': 500.0,
            'net_margin': 0,
            'pe_ratio': 500,
        }
        result = classifier.classify(data)
        assert result.stage == LifecycleStage.HYPERGROWTH
        assert result.confidence > 0.9

        data = {
            'revenue_growth_yoy': -80.0,
            'net_margin': -50,
            'pe_ratio': None,
        }
        result = classifier.classify(data)
        assert result.stage == LifecycleStage.TURNAROUND

    def test_confidence_always_bounded(self, classifier):
        """Confidence should always be between 0 and 1."""
        test_cases = [
            {'revenue_growth_yoy': 500.0},
            {'revenue_growth_yoy': -100.0},
            {'revenue_growth_yoy': 30.0},
            {'revenue_growth_yoy': 3.0, 'net_margin': 20.0, 'pe_ratio': 5},
            {},
        ]
        for data in test_cases:
            result = classifier.classify(data)
            assert 0 <= result.confidence <= 1.0, (
                f"Confidence {result.confidence} out of bounds for {data}"
            )

    def test_exactly_at_hypergrowth_boundary(self, classifier):
        """Exactly 50% growth -> not hypergrowth (requires >50)."""
        data = {'revenue_growth_yoy': 50.0}
        result = classifier.classify(data)
        assert result.stage == LifecycleStage.GROWTH

    def test_exactly_at_growth_boundary(self, classifier):
        """Exactly 20% growth -> not growth (requires >20)."""
        data = {'revenue_growth_yoy': 20.0}
        result = classifier.classify(data)
        assert result.stage == LifecycleStage.MATURE

    def test_exactly_at_turnaround_boundary(self, classifier):
        """Exactly -5% growth -> not turnaround (requires < -5)."""
        data = {'revenue_growth_yoy': -5.0}
        result = classifier.classify(data)
        # -5.0 is not < -5.0, so should not be turnaround
        assert result.stage != LifecycleStage.TURNAROUND


# =====================================================================
# TestRealWorldExamples
# =====================================================================


class TestRealWorldExamples:
    """Tests using real-world company examples."""

    @pytest.fixture
    def classifier(self):
        return LifecycleClassifier(session=None)

    def test_nvidia_like_company(self, classifier):
        data = {
            'revenue_growth_yoy': 122.0,
            'net_margin': 55.0,
            'pe_ratio': 60,
        }
        result = classifier.classify(data)
        assert result.stage == LifecycleStage.HYPERGROWTH

    def test_jpm_like_company(self, classifier):
        data = {
            'revenue_growth_yoy': 8.0,
            'net_margin': 25.0,
            'pe_ratio': 12,
        }
        result = classifier.classify(data)
        assert result.stage in [LifecycleStage.VALUE, LifecycleStage.MATURE]

    def test_walmart_like_company(self, classifier):
        data = {
            'revenue_growth_yoy': 5.0,
            'net_margin': 2.5,
            'pe_ratio': 28,
        }
        result = classifier.classify(data)
        assert result.stage == LifecycleStage.MATURE

    def test_startup_like_company(self, classifier):
        data = {
            'revenue_growth_yoy': 45.0,
            'net_margin': -15.0,
            'pe_ratio': None,
        }
        result = classifier.classify(data)
        assert result.stage == LifecycleStage.GROWTH

    def test_deep_value_stock(self, classifier):
        """Classic deep value: very low P/E, decent margins."""
        data = {
            'revenue_growth_yoy': 1.0,
            'net_margin': 15.0,
            'pe_ratio': 5,
        }
        result = classifier.classify(data)
        assert result.stage == LifecycleStage.VALUE
        assert result.confidence > 0.7

    def test_distressed_company(self, classifier):
        """Company with severe revenue decline."""
        data = {
            'revenue_growth_yoy': -50.0,
            'net_margin': -20.0,
            'pe_ratio': None,
        }
        result = classifier.classify(data)
        assert result.stage == LifecycleStage.TURNAROUND
        assert result.confidence > 0.8


if __name__ == "__main__":
    pytest.main([__file__, "-v"])
