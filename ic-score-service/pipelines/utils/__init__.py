"""Utility modules for data ingestion pipelines."""

from .sector_percentile import (
    SectorPercentileCalculator,
    SectorPercentileAggregator,
    SectorStats,
)
from .lifecycle import (
    LifecycleClassifier,
    LifecycleStage,
    ClassificationResult,
    get_lifecycle_stage_description,
)
from .earnings_revisions import (
    EarningsRevisionsCalculator,
    EarningsRevisionsResult,
)
from .historical_valuation import (
    HistoricalValuationCalculator,
    HistoricalValuationResult,
)
from .dividend_quality import (
    DividendQualityCalculator,
    DividendQualityResult,
)
from .score_stabilizer import (
    ScoreStabilizer,
    StabilizationResult,
    ScoreEvent,
    EventType,
)
from .peer_comparison import (
    PeerComparisonService,
    PeerComparisonResult,
    PeerStock,
)
from .catalyst_detector import (
    CatalystService,
    Catalyst,
    CatalystDetector,
    EarningsDetector,
    AnalystRatingDetector,
    InsiderTradeDetector,
    TechnicalBreakoutDetector,
    DividendDateDetector,
    FiftyTwoWeekDetector,
)
from .score_explainer import (
    ScoreExplainer,
    ScoreChangeExplanation,
    ScoreChangeReason,
    GranularConfidence,
    FactorDataStatus,
)

__all__ = [
    # Sector percentiles
    'SectorPercentileCalculator',
    'SectorPercentileAggregator',
    'SectorStats',
    # Lifecycle classification
    'LifecycleClassifier',
    'LifecycleStage',
    'ClassificationResult',
    'get_lifecycle_stage_description',
    # Phase 2 factors
    'EarningsRevisionsCalculator',
    'EarningsRevisionsResult',
    'HistoricalValuationCalculator',
    'HistoricalValuationResult',
    'DividendQualityCalculator',
    'DividendQualityResult',
    # Phase 3: Score stability
    'ScoreStabilizer',
    'StabilizationResult',
    'ScoreEvent',
    'EventType',
    # Phase 3: Peer comparison
    'PeerComparisonService',
    'PeerComparisonResult',
    'PeerStock',
    # Phase 3: Catalysts
    'CatalystService',
    'Catalyst',
    'CatalystDetector',
    'EarningsDetector',
    'AnalystRatingDetector',
    'InsiderTradeDetector',
    'TechnicalBreakoutDetector',
    'DividendDateDetector',
    'FiftyTwoWeekDetector',
    # Phase 3: Explanations & confidence
    'ScoreExplainer',
    'ScoreChangeExplanation',
    'ScoreChangeReason',
    'GranularConfidence',
    'FactorDataStatus',
]
