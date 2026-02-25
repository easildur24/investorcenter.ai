"""Database operations for Reddit sentiment collector."""

import logging
import os
from typing import List, Set

import psycopg2
from psycopg2.extras import execute_batch

from .models import RedditPost

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

