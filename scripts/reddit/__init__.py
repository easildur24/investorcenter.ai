# Reddit Sentiment Pipeline Package
"""
Collects posts from Reddit, extracts tickers via Gemini LLM,
and aggregates sentiment into per-ticker snapshots and time-series history.
"""

from .models import RedditPost, TickerMention, SentimentResult, LexiconEntry
from .fetcher import RedditFetcher
from .ai_processor import RedditAIProcessor
from .aggregator import SentimentAggregator

# Legacy imports (deprecated - replaced by LLM pipeline)
from .ticker_extractor import TickerExtractor
from .sentiment_analyzer import SentimentAnalyzer

__all__ = [
    "RedditPost",
    "TickerMention",
    "SentimentResult",
    "LexiconEntry",
    "RedditFetcher",
    "RedditAIProcessor",
    "SentimentAggregator",
    "TickerExtractor",
    "SentimentAnalyzer",
]
