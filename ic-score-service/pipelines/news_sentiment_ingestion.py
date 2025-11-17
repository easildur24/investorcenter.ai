#!/usr/bin/env python3
"""News Sentiment Ingestion Pipeline.

This script fetches news articles from Polygon.io and stores them with
sentiment analysis in the news_articles table.

Usage:
    python news_sentiment_ingestion.py --hours 24        # Last 24 hours
    python news_sentiment_ingestion.py --ticker AAPL     # Single ticker
    python news_sentiment_ingestion.py --backfill 30     # Last 30 days
"""

import argparse
import asyncio
import logging
import sys
from datetime import datetime, timedelta
from pathlib import Path
from typing import List, Optional, Dict, Any

sys.path.insert(0, str(Path(__file__).parent.parent))

from sqlalchemy import text
from sqlalchemy.dialects.postgresql import insert as pg_insert
from tqdm import tqdm

from database.database import get_database
from models import NewsArticle
from pipelines.utils.polygon_client import PolygonClient

# Setup logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s',
    handlers=[
        logging.StreamHandler(),
        logging.FileHandler('/app/logs/news_sentiment_ingestion.log')
    ]
)
logger = logging.getLogger(__name__)


class FinBERTSentimentAnalyzer:
    """FinBERT sentiment analyzer for financial text."""

    def __init__(self):
        """Initialize FinBERT model (lazy loading)."""
        self.tokenizer = None
        self.model = None
        self._initialized = False

    def _initialize_model(self):
        """Initialize the model (lazy loading to avoid startup delays)."""
        if self._initialized:
            return

        try:
            from transformers import AutoTokenizer, AutoModelForSequenceClassification
            import torch

            logger.info("Loading FinBERT model...")
            self.tokenizer = AutoTokenizer.from_pretrained("ProsusAI/finbert")
            self.model = AutoModelForSequenceClassification.from_pretrained("ProsusAI/finbert")
            self.model.eval()
            self._initialized = True
            logger.info("FinBERT model loaded successfully")

        except ImportError:
            logger.warning("transformers or torch not available - sentiment analysis disabled")
            self._initialized = False
        except Exception as e:
            logger.error(f"Error loading FinBERT model: {e}")
            self._initialized = False

    def analyze(self, text: str) -> tuple[Optional[float], Optional[str]]:
        """Analyze sentiment of financial text using FinBERT.

        Args:
            text: Text to analyze (title + summary).

        Returns:
            tuple: (score from -100 to 100, label: positive/negative/neutral)
        """
        if not text:
            return None, None

        # Initialize model if needed
        if not self._initialized:
            self._initialize_model()

        if not self._initialized or not self.model:
            return None, None

        try:
            import torch

            # Truncate text to max length
            inputs = self.tokenizer(
                text,
                return_tensors="pt",
                truncation=True,
                max_length=512,
                padding=True
            )

            # Get predictions
            with torch.no_grad():
                outputs = self.model(**inputs)
                predictions = torch.nn.functional.softmax(outputs.logits, dim=-1)

            # FinBERT labels: positive, negative, neutral
            labels = ['positive', 'negative', 'neutral']
            predicted_class = predictions.argmax().item()
            confidence = predictions[0][predicted_class].item()

            # Map to -100 to 100 scale
            if labels[predicted_class] == 'positive':
                score = confidence * 100
            elif labels[predicted_class] == 'negative':
                score = -confidence * 100
            else:
                score = 0.0

            return score, labels[predicted_class].capitalize()

        except Exception as e:
            logger.error(f"Error in FinBERT sentiment analysis: {e}")
            return None, None


class NewsSentimentIngestion:
    """Ingestion pipeline for news with sentiment analysis."""

    def __init__(self):
        """Initialize the ingestion pipeline."""
        self.polygon_client = PolygonClient()
        self.db = get_database()
        self.sentiment_analyzer = FinBERTSentimentAnalyzer()

        self.processed_count = 0
        self.success_count = 0

    async def get_active_tickers(self, limit: Optional[int] = None) -> List[str]:
        """Get list of active tickers from database."""
        async with self.db.session() as session:
            query = text("""
                SELECT ticker
                FROM stocks
                WHERE ticker NOT LIKE '%-%'
                  AND is_active = true
                ORDER BY ticker
                LIMIT :limit
            """)
            result = await session.execute(query, {"limit": limit or 500})
            return [row[0] for row in result.fetchall()]

    async def fetch_and_store_news(
        self,
        ticker: Optional[str] = None,
        hours_back: int = 24
    ) -> bool:
        """Fetch news and store in database.

        Args:
            ticker: Stock ticker symbol (optional).
            hours_back: How many hours back to fetch news.

        Returns:
            True if successful.
        """
        try:
            # Calculate date range
            published_after = (datetime.now() - timedelta(hours=hours_back)).strftime('%Y-%m-%d')

            # Fetch news from Polygon
            news_articles = self.polygon_client.get_news(
                ticker=ticker,
                limit=100,
                published_after=published_after
            )

            if not news_articles:
                logger.warning(f"No news found for {ticker or 'all tickers'}")
                return False

            # Prepare records
            records = []
            for article in news_articles:
                # Parse published date (convert to naive datetime for PostgreSQL TIMESTAMP WITHOUT TIME ZONE)
                published_at = datetime.fromisoformat(article['published_utc'].replace('Z', '+00:00')).replace(tzinfo=None)

                record = {
                    'title': article.get('title', '')[:500],
                    'url': article.get('article_url', ''),
                    'source': article.get('publisher', {}).get('name', 'Unknown'),
                    'published_at': published_at,
                    'summary': article.get('description', ''),
                    'author': article.get('author'),
                    'tickers': article.get('tickers', []),
                    'sentiment_score': None,  # Polygon provides sentiment in insights
                    'sentiment_label': None,
                    'image_url': article.get('image_url'),
                }

                # Extract sentiment if available from Polygon
                insights = article.get('insights', [])
                if insights:
                    for insight in insights:
                        if insight.get('sentiment'):
                            record['sentiment_label'] = insight['sentiment'].capitalize()
                            # Map to numeric score
                            if insight['sentiment'] == 'positive':
                                record['sentiment_score'] = 75.0
                            elif insight['sentiment'] == 'negative':
                                record['sentiment_score'] = -75.0
                            else:
                                record['sentiment_score'] = 0.0
                            break

                # If sentiment not provided by Polygon, use FinBERT
                if record['sentiment_score'] is None:
                    text = f"{record['title']} {record.get('summary', '')}"
                    score, label = self.sentiment_analyzer.analyze(text)
                    if score is not None:
                        record['sentiment_score'] = score
                        record['sentiment_label'] = label

                records.append(record)

            # Store in database
            async with self.db.session() as session:
                stmt = pg_insert(NewsArticle).values(records)
                stmt = stmt.on_conflict_do_nothing(index_elements=['url'])

                await session.execute(stmt)
                await session.commit()

            logger.info(f"Stored {len(records)} news articles")
            return True

        except Exception as e:
            logger.error(f"Error fetching/storing news: {e}", exc_info=True)
            return False

    async def run(
        self,
        hours: int = 24,
        ticker: Optional[str] = None,
        backfill_days: int = 0
    ):
        """Run the ingestion pipeline."""
        start_time = datetime.now()
        logger.info("=" * 80)
        logger.info("News Sentiment Ingestion Pipeline")
        logger.info("=" * 80)

        if backfill_days > 0:
            hours = backfill_days * 24

        if ticker:
            # Single ticker
            await self.fetch_and_store_news(ticker=ticker, hours_back=hours)
        else:
            # Fetch for top tickers
            tickers = await self.get_active_tickers(limit=100)
            logger.info(f"Fetching news for {len(tickers)} tickers...")

            for ticker in tqdm(tickers, desc="Processing tickers"):
                await self.fetch_and_store_news(ticker=ticker, hours_back=hours)
                await asyncio.sleep(0.2)  # Rate limiting

        duration = datetime.now() - start_time
        logger.info(f"Completed in {duration}")

        self.polygon_client.close()


def main():
    """Main entry point for CLI."""
    parser = argparse.ArgumentParser(description='Ingest news with sentiment analysis')

    parser.add_argument('--hours', type=int, default=24, help='Hours back to fetch')
    parser.add_argument('--ticker', type=str, help='Single ticker symbol')
    parser.add_argument('--backfill', type=int, default=0, help='Backfill N days')

    args = parser.parse_args()

    pipeline = NewsSentimentIngestion()
    asyncio.run(pipeline.run(
        hours=args.hours,
        ticker=args.ticker,
        backfill_days=args.backfill
    ))


if __name__ == '__main__':
    main()
