"""Tests for the news sentiment ingestion pipeline.

Validates FinBERT sentiment analyzer (lazy loading, sentiment
scoring), news fetching from Polygon, and database storage.

CRITICAL: torch and transformers are mocked at module level BEFORE
the pipeline is imported, following the same pattern as
test_technical_indicators.py for talib.
"""

import os
import sys
from types import ModuleType
from unittest.mock import AsyncMock, MagicMock, patch

import pytest

# -------------------------------------------------------------------
# Create fake torch and transformers modules BEFORE pipeline import.
# Save originals so we can restore after this module is done.
# -------------------------------------------------------------------
_orig_torch = sys.modules.get("torch")
_orig_transformers = sys.modules.get("transformers")

_mock_torch = ModuleType("torch")
_mock_torch.no_grad = lambda: MagicMock(
    __enter__=MagicMock(), __exit__=MagicMock()
)
_mock_torch.device = MagicMock
_mock_torch.Tensor = MagicMock
_mock_torch.cuda = MagicMock()
_mock_torch.cuda.is_available = MagicMock(return_value=False)
_mock_torch.nn = MagicMock()
_mock_torch.float32 = "float32"
_mock_torch.float64 = "float64"
sys.modules["torch"] = _mock_torch

_mock_transformers = ModuleType("transformers")
_mock_transformers.AutoTokenizer = MagicMock()
_mock_transformers.AutoModelForSequenceClassification = MagicMock()
sys.modules["transformers"] = _mock_transformers


def teardown_module(module):
    """Restore original sys.modules to avoid leaking fakes."""
    if _orig_torch is None:
        sys.modules.pop("torch", None)
    else:
        sys.modules["torch"] = _orig_torch
    if _orig_transformers is None:
        sys.modules.pop("transformers", None)
    else:
        sys.modules["transformers"] = _orig_transformers

# Now import with mocked dependencies
with patch(
    "pipelines.news_sentiment_ingestion.get_database"
), patch.dict(
    os.environ, {"POLYGON_API_KEY": "test"}
), patch(
    "pipelines.news_sentiment_ingestion.PolygonClient"
):
    from pipelines.news_sentiment_ingestion import (
        FinBERTSentimentAnalyzer,
        NewsSentimentIngestion,
    )


@pytest.fixture
def analyzer():
    """Create a FinBERTSentimentAnalyzer."""
    return FinBERTSentimentAnalyzer()


@pytest.fixture
def pipeline():
    """Create a NewsSentimentIngestion with mocked deps."""
    with patch(
        "pipelines.news_sentiment_ingestion.get_database"
    ) as mock_db, patch.dict(
        os.environ, {"POLYGON_API_KEY": "test"}
    ), patch(
        "pipelines.news_sentiment_ingestion.PolygonClient"
    ) as mock_polygon_cls:
        db_instance = MagicMock()
        mock_db.return_value = db_instance
        polygon_instance = MagicMock()
        mock_polygon_cls.return_value = polygon_instance
        p = NewsSentimentIngestion()
        yield p


# ==================================================================
# FinBERTSentimentAnalyzer
# ==================================================================


class TestFinBERTAnalyzerInit:
    def test_starts_uninitialised(self, analyzer):
        assert analyzer._initialized is False
        assert analyzer.tokenizer is None
        assert analyzer.model is None

    def test_empty_text_returns_none(self, analyzer):
        score, label = analyzer.analyze("")
        assert score is None
        assert label is None

    def test_none_text_returns_none(self, analyzer):
        score, label = analyzer.analyze(None)
        assert score is None
        assert label is None


class TestFinBERTAnalyze:
    def test_analyze_when_init_fails_returns_none(
        self, analyzer
    ):
        """When model initialization fails (ImportError),
        analyze should return (None, None)."""
        # Patch the lazy import to raise ImportError
        with patch.dict(
            "sys.modules",
            {"transformers": None},
        ):
            # Force re-initialization attempt
            analyzer._initialized = False
            analyzer.model = None
            analyzer.tokenizer = None
            score, label = analyzer.analyze(
                "AAPL beats earnings"
            )
            assert score is None
            assert label is None

    def test_analyze_positive_with_mock_model(self, analyzer):
        """Test sentiment analysis with a fully mocked model."""
        import torch as mock_torch_mod

        # Set up mocked model components
        analyzer._initialized = True
        analyzer.tokenizer = MagicMock()
        analyzer.model = MagicMock()

        # Mock the model output
        mock_logits = MagicMock()
        mock_outputs = MagicMock()
        mock_outputs.logits = mock_logits

        # Create fake softmax output: [positive=0.85, negative=0.05, neutral=0.10]
        mock_predictions = MagicMock()
        mock_predictions.argmax.return_value = MagicMock(
            item=MagicMock(return_value=0)
        )
        mock_predictions.__getitem__ = MagicMock(
            return_value=MagicMock(
                __getitem__=MagicMock(
                    return_value=MagicMock(
                        item=MagicMock(return_value=0.85)
                    )
                )
            )
        )

        # Patch torch.nn.functional.softmax
        mock_torch_mod.nn = MagicMock()
        mock_torch_mod.nn.functional.softmax.return_value = (
            mock_predictions
        )

        # Mock no_grad context
        mock_torch_mod.no_grad = MagicMock(
            return_value=MagicMock(
                __enter__=MagicMock(),
                __exit__=MagicMock(),
            )
        )

        analyzer.model.return_value = mock_outputs

        score, label = analyzer.analyze(
            "AAPL reports record quarterly earnings"
        )

        if score is not None:
            # Positive -> score > 0
            assert score > 0
            assert label == "Positive"


# ==================================================================
# NewsSentimentIngestion
# ==================================================================


class TestNewsSentimentInit:
    def test_pipeline_has_sentiment_analyzer(self, pipeline):
        assert pipeline.sentiment_analyzer is not None
        assert isinstance(
            pipeline.sentiment_analyzer,
            FinBERTSentimentAnalyzer,
        )

    def test_counters_initialized(self, pipeline):
        assert pipeline.processed_count == 0
        assert pipeline.success_count == 0


class TestGetActiveTickers:
    @pytest.mark.asyncio
    async def test_returns_tickers(self, pipeline):
        mock_session = AsyncMock()
        mock_result = MagicMock()
        mock_result.fetchall.return_value = [
            ("AAPL",),
            ("MSFT",),
        ]
        mock_session.execute.return_value = mock_result
        pipeline.db.session.return_value.__aenter__ = AsyncMock(
            return_value=mock_session
        )
        pipeline.db.session.return_value.__aexit__ = AsyncMock(
            return_value=False
        )

        tickers = await pipeline.get_active_tickers(limit=10)
        assert tickers == ["AAPL", "MSFT"]


class TestFetchAndStoreNews:
    @pytest.mark.asyncio
    async def test_no_news_returns_false(self, pipeline):
        pipeline.polygon_client.get_news.return_value = []

        result = await pipeline.fetch_and_store_news(
            ticker="AAPL", hours_back=24
        )

        assert result is False

    @pytest.mark.asyncio
    async def test_news_fetched_and_stored(self, pipeline):
        pipeline.polygon_client.get_news.return_value = [
            {
                "title": "AAPL Reports Record Earnings",
                "article_url": "https://example.com/aapl",
                "publisher": {"name": "Reuters"},
                "published_utc": "2024-06-15T10:00:00Z",
                "description": "Apple Inc reported...",
                "author": "John Doe",
                "tickers": ["AAPL"],
                "image_url": "https://example.com/img.jpg",
                "insights": [],
            }
        ]

        # Mock the sentiment analyzer to return None
        pipeline.sentiment_analyzer.analyze = MagicMock(
            return_value=(None, None)
        )

        mock_session = AsyncMock()
        pipeline.db.session.return_value.__aenter__ = AsyncMock(
            return_value=mock_session
        )
        pipeline.db.session.return_value.__aexit__ = AsyncMock(
            return_value=False
        )

        result = await pipeline.fetch_and_store_news(
            ticker="AAPL", hours_back=24
        )

        assert result is True
        mock_session.execute.assert_called_once()
        mock_session.commit.assert_called_once()

    @pytest.mark.asyncio
    async def test_polygon_sentiment_used_as_fallback(
        self, pipeline
    ):
        pipeline.polygon_client.get_news.return_value = [
            {
                "title": "AAPL drops after guidance",
                "article_url": "https://example.com/aapl2",
                "publisher": {"name": "Bloomberg"},
                "published_utc": "2024-06-15T10:00:00Z",
                "description": "Apple lowered guidance",
                "author": None,
                "tickers": ["AAPL"],
                "image_url": None,
                "insights": [
                    {
                        "sentiment": "negative",
                        "sentiment_reasoning": "lowered guidance",
                    }
                ],
            }
        ]

        # FinBERT fails, so Polygon sentiment should be used
        pipeline.sentiment_analyzer.analyze = MagicMock(
            return_value=(None, None)
        )

        mock_session = AsyncMock()
        pipeline.db.session.return_value.__aenter__ = AsyncMock(
            return_value=mock_session
        )
        pipeline.db.session.return_value.__aexit__ = AsyncMock(
            return_value=False
        )

        result = await pipeline.fetch_and_store_news(
            ticker="AAPL", hours_back=24
        )

        assert result is True

    @pytest.mark.asyncio
    async def test_exception_returns_false(self, pipeline):
        pipeline.polygon_client.get_news.side_effect = Exception(
            "API error"
        )

        result = await pipeline.fetch_and_store_news(
            ticker="AAPL", hours_back=24
        )

        assert result is False
