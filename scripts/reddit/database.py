"""Database operations for Reddit sentiment collector."""

import logging
import os
from datetime import datetime
from typing import List, Optional, Set

import psycopg2
from psycopg2.extras import execute_batch

from .models import RedditPost, SentimentResult, SocialPostRecord

logger = logging.getLogger(__name__)


class Database:
    """Database operations for social posts and sentiment data."""

    def __init__(
        self,
        host: str = None,
        port: int = None,
        user: str = None,
        password: str = None,
        database: str = None,
    ):
        """Initialize database connection.

        Args:
            host: Database host (default: DB_HOST env var)
            port: Database port (default: DB_PORT env var or 5432)
            user: Database user (default: DB_USER env var)
            password: Database password (default: DB_PASSWORD env var)
            database: Database name (default: DB_NAME env var)
        """
        self.host = host or os.getenv("DB_HOST", "localhost")
        self.port = port or int(os.getenv("DB_PORT", "5432"))
        self.user = user or os.getenv("DB_USER", "investorcenter")
        self.password = password or os.getenv("DB_PASSWORD", "")
        self.database = database or os.getenv("DB_NAME", "investorcenter_db")

        self.conn = None

    def connect(self):
        """Establish database connection."""
        try:
            self.conn = psycopg2.connect(
                host=self.host,
                port=self.port,
                user=self.user,
                password=self.password,
                database=self.database,
            )
            self.conn.autocommit = False
            logger.info(f"Connected to database {self.database}@{self.host}")
        except Exception as e:
            logger.error(f"Failed to connect to database: {e}")
            raise

    def close(self):
        """Close database connection."""
        if self.conn:
            self.conn.close()
            logger.info("Database connection closed")

    def get_connection(self):
        """Get the raw connection object.

        Returns:
            psycopg2 connection
        """
        if not self.conn:
            self.connect()
        return self.conn

    def load_valid_tickers(self) -> Set[str]:
        """Load valid stock/ETF tickers from database.

        Returns:
            Set of valid ticker symbols
        """
        tickers = set()

        try:
            cursor = self.conn.cursor()
            cursor.execute("""
                SELECT symbol FROM tickers
                WHERE asset_type IN ('stock', 'etf', 'CS', 'ETF')
                  AND symbol ~ '^[A-Z]{1,5}$'
            """)

            for row in cursor.fetchall():
                tickers.add(row[0].upper())

            cursor.close()
            logger.info(f"Loaded {len(tickers)} valid tickers")

        except Exception as e:
            logger.error(f"Failed to load tickers: {e}")

        return tickers

    def load_lexicon(self) -> dict:
        """Load sentiment lexicon from database.

        Returns:
            Dict mapping lowercase terms to (sentiment, weight, category)
        """
        from .models import LexiconEntry

        lexicon = {}

        try:
            cursor = self.conn.cursor()
            cursor.execute("""
                SELECT term, sentiment, weight, COALESCE(category, '') as category
                FROM sentiment_lexicon
            """)

            for row in cursor.fetchall():
                term, sentiment, weight, category = row
                lexicon[term.lower()] = LexiconEntry(
                    term=term,
                    sentiment=sentiment,
                    weight=float(weight),
                    category=category if category else None,
                )

            cursor.close()
            logger.info(f"Loaded {len(lexicon)} lexicon terms")

        except Exception as e:
            logger.error(f"Failed to load lexicon: {e}")

        return lexicon

    def post_exists(self, external_post_id: str) -> bool:
        """Check if a post already exists in database.

        Args:
            external_post_id: Reddit post ID

        Returns:
            True if post exists
        """
        try:
            cursor = self.conn.cursor()
            cursor.execute(
                "SELECT 1 FROM social_posts WHERE external_post_id = %s",
                (external_post_id,),
            )
            exists = cursor.fetchone() is not None
            cursor.close()
            return exists
        except Exception as e:
            logger.error(f"Failed to check post existence: {e}")
            return False

    def upsert_post(self, record: SocialPostRecord) -> bool:
        """Insert or update a social post.

        Args:
            record: SocialPostRecord to insert/update

        Returns:
            True if successful
        """
        try:
            cursor = self.conn.cursor()
            cursor.execute(
                """
                INSERT INTO social_posts (
                    external_post_id, source, ticker, subreddit, title, body_preview,
                    url, upvotes, comment_count, award_count, sentiment,
                    sentiment_confidence, flair, posted_at, updated_at
                ) VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, NOW())
                ON CONFLICT (external_post_id) DO UPDATE SET
                    upvotes = EXCLUDED.upvotes,
                    comment_count = EXCLUDED.comment_count,
                    award_count = EXCLUDED.award_count,
                    sentiment = COALESCE(EXCLUDED.sentiment, social_posts.sentiment),
                    sentiment_confidence = COALESCE(EXCLUDED.sentiment_confidence, social_posts.sentiment_confidence),
                    updated_at = NOW()
                """,
                (
                    record.external_post_id,
                    record.source,
                    record.ticker,
                    record.subreddit,
                    record.title,
                    record.body_preview,
                    record.url,
                    record.upvotes,
                    record.comment_count,
                    record.award_count,
                    record.sentiment,
                    record.sentiment_confidence,
                    record.flair,
                    record.posted_at,
                ),
            )
            cursor.close()
            return True

        except Exception as e:
            logger.error(f"Failed to upsert post {record.external_post_id}: {e}")
            self.conn.rollback()
            return False

    def bulk_upsert_posts(self, records: List[SocialPostRecord]) -> int:
        """Bulk insert/update social posts.

        Args:
            records: List of SocialPostRecord objects

        Returns:
            Number of records processed
        """
        if not records:
            return 0

        try:
            cursor = self.conn.cursor()

            query = """
                INSERT INTO social_posts (
                    external_post_id, source, ticker, subreddit, title, body_preview,
                    url, upvotes, comment_count, award_count, sentiment,
                    sentiment_confidence, flair, posted_at, updated_at
                ) VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, NOW())
                ON CONFLICT (external_post_id) DO UPDATE SET
                    upvotes = EXCLUDED.upvotes,
                    comment_count = EXCLUDED.comment_count,
                    award_count = EXCLUDED.award_count,
                    updated_at = NOW()
            """

            data = [
                (
                    r.external_post_id,
                    r.source,
                    r.ticker,
                    r.subreddit,
                    r.title,
                    r.body_preview,
                    r.url,
                    r.upvotes,
                    r.comment_count,
                    r.award_count,
                    r.sentiment,
                    r.sentiment_confidence,
                    r.flair,
                    r.posted_at,
                )
                for r in records
            ]

            execute_batch(cursor, query, data, page_size=100)
            self.conn.commit()

            cursor.close()
            logger.info(f"Bulk upserted {len(records)} posts")
            return len(records)

        except Exception as e:
            logger.error(f"Failed to bulk upsert posts: {e}")
            self.conn.rollback()
            return 0

    def prune_old_posts(self, retention_days: int = 30) -> int:
        """Delete posts older than retention period.

        Args:
            retention_days: Number of days to keep

        Returns:
            Number of posts deleted
        """
        try:
            cursor = self.conn.cursor()
            cursor.execute(
                """
                DELETE FROM social_posts
                WHERE posted_at < NOW() - %s * INTERVAL '1 day'
                """,
                (retention_days,),
            )
            deleted = cursor.rowcount
            self.conn.commit()
            cursor.close()

            logger.info(f"Pruned {deleted} posts older than {retention_days} days")
            return deleted

        except Exception as e:
            logger.error(f"Failed to prune old posts: {e}")
            self.conn.rollback()
            return 0

    def bulk_upsert_raw_posts(self, posts: List["RedditPost"]) -> int:
        """Bulk insert raw posts to reddit_posts_raw table for V2 AI processing.

        This stores the raw post content without any ticker extraction.
        The AI processor will later extract tickers from these posts.

        Args:
            posts: List of RedditPost objects from Arctic Shift

        Returns:
            Number of records inserted/updated
        """
        if not posts:
            return 0

        try:
            cursor = self.conn.cursor()

            query = """
                INSERT INTO reddit_posts_raw (
                    external_id, subreddit, author, title, body, url,
                    upvotes, comment_count, award_count, flair, posted_at, fetched_at
                ) VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, NOW())
                ON CONFLICT (external_id) DO UPDATE SET
                    upvotes = EXCLUDED.upvotes,
                    comment_count = EXCLUDED.comment_count,
                    award_count = EXCLUDED.award_count
            """

            data = [
                (
                    post.id,  # external_id (Reddit's post ID)
                    post.subreddit,
                    post.author,
                    post.title,
                    post.body,  # Full body, not truncated
                    post.full_url,
                    post.score,
                    post.num_comments,
                    0,  # award_count (not available from Arctic Shift)
                    post.flair,
                    post.created_utc,
                )
                for post in posts
            ]

            execute_batch(cursor, query, data, page_size=100)
            self.conn.commit()

            cursor.close()
            logger.info(f"Bulk upserted {len(posts)} raw posts to reddit_posts_raw")
            return len(posts)

        except Exception as e:
            logger.error(f"Failed to bulk upsert raw posts: {e}")
            self.conn.rollback()
            return 0

    def get_raw_posts_stats(self) -> dict:
        """Get statistics about raw posts and AI processing status.

        Returns:
            Dict with counts for processed/unprocessed posts
        """
        stats = {}

        try:
            cursor = self.conn.cursor()

            # Total raw posts
            cursor.execute("SELECT COUNT(*) FROM reddit_posts_raw")
            stats["total_raw_posts"] = cursor.fetchone()[0]

            # Unprocessed posts (awaiting AI extraction)
            cursor.execute("""
                SELECT COUNT(*) FROM reddit_posts_raw
                WHERE processed_at IS NULL AND processing_skipped = FALSE
            """)
            stats["unprocessed_posts"] = cursor.fetchone()[0]

            # Processed posts
            cursor.execute("""
                SELECT COUNT(*) FROM reddit_posts_raw
                WHERE processed_at IS NOT NULL
            """)
            stats["processed_posts"] = cursor.fetchone()[0]

            # Skipped posts (low engagement)
            cursor.execute("""
                SELECT COUNT(*) FROM reddit_posts_raw
                WHERE processing_skipped = TRUE
            """)
            stats["skipped_posts"] = cursor.fetchone()[0]

            # Total ticker extractions
            cursor.execute("SELECT COUNT(*) FROM reddit_post_tickers")
            stats["total_ticker_extractions"] = cursor.fetchone()[0]

            # Unique tickers in V2
            cursor.execute("SELECT COUNT(DISTINCT ticker) FROM reddit_post_tickers")
            stats["unique_tickers_v2"] = cursor.fetchone()[0]

            cursor.close()

        except Exception as e:
            logger.error(f"Failed to get raw posts stats: {e}")

        return stats

    def prune_old_raw_posts(self, retention_days: int = 60) -> int:
        """Delete raw posts older than retention period.

        Args:
            retention_days: Number of days to keep

        Returns:
            Number of posts deleted
        """
        try:
            cursor = self.conn.cursor()
            cursor.execute(
                """
                DELETE FROM reddit_posts_raw
                WHERE posted_at < NOW() - %s * INTERVAL '1 day'
                """,
                (retention_days,),
            )
            deleted = cursor.rowcount
            self.conn.commit()
            cursor.close()

            logger.info(f"Pruned {deleted} raw posts older than {retention_days} days")
            return deleted

        except Exception as e:
            logger.error(f"Failed to prune old raw posts: {e}")
            self.conn.rollback()
            return 0

    def get_collection_stats(self) -> dict:
        """Get statistics about collected posts.

        Returns:
            Dict with post counts and other stats
        """
        stats = {}

        try:
            cursor = self.conn.cursor()

            # Total posts
            cursor.execute("SELECT COUNT(*) FROM social_posts")
            stats["total_posts"] = cursor.fetchone()[0]

            # Posts in last 24 hours
            cursor.execute(
                "SELECT COUNT(*) FROM social_posts WHERE created_at > NOW() - INTERVAL '24 hours'"
            )
            stats["posts_24h"] = cursor.fetchone()[0]

            # Unique tickers
            cursor.execute("SELECT COUNT(DISTINCT ticker) FROM social_posts")
            stats["unique_tickers"] = cursor.fetchone()[0]

            # Sentiment breakdown
            cursor.execute("""
                SELECT sentiment, COUNT(*)
                FROM social_posts
                WHERE posted_at > NOW() - INTERVAL '24 hours'
                GROUP BY sentiment
            """)
            sentiment_counts = {row[0] or "unknown": row[1] for row in cursor.fetchall()}
            stats["sentiment_24h"] = sentiment_counts

            # Top subreddits
            cursor.execute("""
                SELECT subreddit, COUNT(*) as cnt
                FROM social_posts
                WHERE posted_at > NOW() - INTERVAL '24 hours'
                GROUP BY subreddit
                ORDER BY cnt DESC
                LIMIT 5
            """)
            stats["top_subreddits"] = [
                {"subreddit": row[0], "count": row[1]} for row in cursor.fetchall()
            ]

            cursor.close()

        except Exception as e:
            logger.error(f"Failed to get collection stats: {e}")

        return stats


def create_post_record(
    post: RedditPost,
    ticker: str,
    sentiment_result: Optional[SentimentResult] = None,
) -> SocialPostRecord:
    """Create a SocialPostRecord from a RedditPost.

    Args:
        post: RedditPost object
        ticker: Primary ticker for this post
        sentiment_result: Optional sentiment analysis result

    Returns:
        SocialPostRecord ready for database insertion
    """
    # Truncate body to 500 chars for preview
    body_preview = post.body[:500] if post.body else None

    return SocialPostRecord(
        external_post_id=f"reddit_{post.id}",
        source="reddit",
        ticker=ticker,
        subreddit=post.subreddit,
        title=post.title,
        body_preview=body_preview,
        url=post.full_url,
        upvotes=post.score,
        comment_count=post.num_comments,
        award_count=0,  # Reddit JSON doesn't include award count directly
        sentiment=sentiment_result.sentiment if sentiment_result else None,
        sentiment_confidence=sentiment_result.confidence if sentiment_result else None,
        flair=post.flair,
        posted_at=post.created_utc,
    )
