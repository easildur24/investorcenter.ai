#!/usr/bin/env python3
"""Reddit sentiment collector - fetches posts, extracts tickers, analyzes sentiment.

Usage:
    python reddit_sentiment_collector.py                    # Run with defaults
    python reddit_sentiment_collector.py --subreddits wallstreetbets stocks
    python reddit_sentiment_collector.py --dry-run          # Don't save to database
    python reddit_sentiment_collector.py --limit 50         # Posts per subreddit
"""

import argparse
import logging
import sys
from typing import List

from reddit import (
    RedditFetcher,
    SentimentAnalyzer,
    TickerExtractor,
)
from reddit.database import Database, create_post_record
from reddit.models import RedditPost, SocialPostRecord

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s - %(name)s - %(levelname)s - %(message)s",
    handlers=[logging.StreamHandler(sys.stdout)],
)
logger = logging.getLogger(__name__)

# Default subreddits to monitor (ordered by activity/relevance)
DEFAULT_SUBREDDITS = [
    "wallstreetbets",      # Most active, meme stocks, YOLO trades
    "stocks",              # General stock discussion
    "options",             # Options trading
    "investing",           # Long-term investing
    "Daytrading",          # Day trading strategies
    "StockMarket",         # General market discussion
    "SecurityAnalysis",    # Value investing, deep dives
    "ValueInvesting",      # Value investing strategies
]


class RedditSentimentCollector:
    """Main collector that orchestrates fetching, extraction, and analysis."""

    def __init__(
        self,
        subreddits: List[str] = None,
        db_host: str = None,
        db_port: int = None,
        db_user: str = None,
        db_password: str = None,
        db_name: str = None,
        dry_run: bool = False,
    ):
        """Initialize collector.

        Args:
            subreddits: List of subreddits to monitor
            db_host: Database host
            db_port: Database port
            db_user: Database user
            db_password: Database password
            db_name: Database name
            dry_run: If True, don't save to database
        """
        self.subreddits = subreddits or DEFAULT_SUBREDDITS
        self.dry_run = dry_run

        # Initialize components
        self.fetcher = RedditFetcher()
        self.db = Database(
            host=db_host,
            port=db_port,
            user=db_user,
            password=db_password,
            database=db_name,
        )

        # These will be initialized after DB connection
        self.ticker_extractor = None
        self.sentiment_analyzer = None

        # Stats tracking
        self.stats = {
            "posts_fetched": 0,
            "posts_with_tickers": 0,
            "posts_saved": 0,
            "posts_skipped": 0,
            "tickers_found": set(),
        }

    def initialize(self):
        """Initialize database connection and load data from DB."""
        logger.info("Initializing collector...")

        # Connect to database
        self.db.connect()

        # Load ticker extractor with valid tickers from DB
        self.ticker_extractor = TickerExtractor.from_database(self.db.get_connection())

        # Load sentiment analyzer with lexicon from DB
        self.sentiment_analyzer = SentimentAnalyzer.from_database(self.db.get_connection())

        logger.info(
            f"Initialized: {self.ticker_extractor.get_ticker_count()} tickers, "
            f"{self.sentiment_analyzer.get_lexicon_size()} lexicon terms"
        )

    def process_post(self, post: RedditPost) -> List[SocialPostRecord]:
        """Process a single post - extract tickers and analyze sentiment.

        Args:
            post: RedditPost to process

        Returns:
            List of SocialPostRecord (one per ticker found)
        """
        records = []

        # Extract tickers from post
        mentions = self.ticker_extractor.extract(post.title, post.body)

        if not mentions:
            return records

        # Analyze sentiment
        sentiment_result = self.sentiment_analyzer.analyze(post.title, post.body)

        # Get primary ticker
        primary_ticker = self.ticker_extractor.get_primary_ticker(mentions)

        # Create record for primary ticker
        record = create_post_record(post, primary_ticker, sentiment_result)
        records.append(record)

        # Track stats
        self.stats["posts_with_tickers"] += 1
        for mention in mentions:
            self.stats["tickers_found"].add(mention.ticker)

        # Log interesting findings
        if sentiment_result.confidence > 0.5:
            logger.debug(
                f"  [{post.subreddit}] {primary_ticker}: {sentiment_result.sentiment} "
                f"(score={sentiment_result.score:.2f}, conf={sentiment_result.confidence:.2f})"
            )

        return records

    def run(
        self,
        limit_per_subreddit: int = 100,
        sort: str = "hot",
        time_filter: str = "day",
        min_score: int = 5,
        max_age_days: int = 7,
    ):
        """Run the collector.

        Args:
            limit_per_subreddit: Max posts to fetch per subreddit
            sort: Sort order (hot, new, top, rising)
            time_filter: Time filter for top sort (hour, day, week, month)
            min_score: Minimum score (upvotes) to include
            max_age_days: Maximum post age in days
        """
        logger.info(f"Starting collection from {len(self.subreddits)} subreddits...")
        logger.info(f"  Sort: {sort}, Time filter: {time_filter}")
        logger.info(f"  Min score: {min_score}, Max age: {max_age_days} days")
        logger.info(f"  Dry run: {self.dry_run}")

        all_records: List[SocialPostRecord] = []
        all_raw_posts: List[RedditPost] = []  # For V2 raw storage

        for subreddit in self.subreddits:
            logger.info(f"Fetching r/{subreddit}...")

            try:
                posts = self.fetcher.fetch_subreddit(
                    subreddit=subreddit,
                    sort=sort,
                    limit=limit_per_subreddit,
                    time_filter=time_filter,
                    min_score=min_score,
                    max_age_days=max_age_days,
                )

                self.stats["posts_fetched"] += len(posts)
                logger.info(f"  Fetched {len(posts)} posts from r/{subreddit}")

                # Collect raw posts for V2 storage (all posts, before filtering)
                all_raw_posts.extend(posts)

                # Process each post (V1 extraction)
                for post in posts:
                    # Check if we already have this post
                    external_id = f"reddit_{post.id}"
                    if not self.dry_run and self.db.post_exists(external_id):
                        self.stats["posts_skipped"] += 1
                        continue

                    records = self.process_post(post)
                    all_records.extend(records)

            except Exception as e:
                logger.error(f"Error processing r/{subreddit}: {e}")
                continue

        # Save raw posts to reddit_posts_raw for V2 AI processing
        if not self.dry_run and all_raw_posts:
            logger.info(f"Saving {len(all_raw_posts)} raw posts for V2 AI processing...")
            try:
                raw_saved = self.db.bulk_upsert_raw_posts(all_raw_posts)
                self.stats["raw_posts_saved"] = raw_saved
            except Exception as e:
                logger.warning(f"Failed to save raw posts (V2 table may not exist): {e}")
                self.stats["raw_posts_saved"] = 0

        # Save to social_posts (V1 with extraction)
        if not self.dry_run and all_records:
            logger.info(f"Saving {len(all_records)} records to database...")
            saved = self.db.bulk_upsert_posts(all_records)
            self.stats["posts_saved"] = saved
        elif self.dry_run:
            logger.info(f"Dry run - would save {len(all_records)} records")
            self.stats["posts_saved"] = 0

        # Log summary
        self._log_summary()

    def _log_summary(self):
        """Log collection summary."""
        logger.info("=" * 50)
        logger.info("Collection Summary:")
        logger.info(f"  Posts fetched:      {self.stats['posts_fetched']}")
        logger.info(f"  Posts with tickers: {self.stats['posts_with_tickers']}")
        logger.info(f"  Posts skipped:      {self.stats['posts_skipped']}")
        logger.info(f"  Posts saved (V1):   {self.stats['posts_saved']}")
        logger.info(f"  Raw posts (V2):     {self.stats.get('raw_posts_saved', 0)}")
        logger.info(f"  Unique tickers:     {len(self.stats['tickers_found'])}")

        if self.stats["tickers_found"]:
            top_tickers = sorted(self.stats["tickers_found"])[:20]
            logger.info(f"  Sample tickers:     {', '.join(top_tickers)}")

        logger.info("=" * 50)

    def prune_old_posts(self, retention_days: int = 30):
        """Prune posts older than retention period.

        Args:
            retention_days: Number of days to keep
        """
        if self.dry_run:
            logger.info(f"Dry run - would prune posts older than {retention_days} days")
            return

        deleted = self.db.prune_old_posts(retention_days)
        logger.info(f"Pruned {deleted} posts older than {retention_days} days")

    def get_stats(self) -> dict:
        """Get collection statistics from database.

        Returns:
            Dict with statistics
        """
        return self.db.get_collection_stats()

    def close(self):
        """Close database connection."""
        self.db.close()


def main():
    """Main entry point."""
    parser = argparse.ArgumentParser(
        description="Collect Reddit posts, extract tickers, and analyze sentiment"
    )
    parser.add_argument(
        "--subreddits",
        nargs="+",
        default=DEFAULT_SUBREDDITS,
        help=f"Subreddits to monitor (default: {', '.join(DEFAULT_SUBREDDITS)})",
    )
    parser.add_argument(
        "--limit",
        type=int,
        default=100,
        help="Max posts per subreddit (default: 100)",
    )
    parser.add_argument(
        "--sort",
        choices=["hot", "new", "top", "rising"],
        default="hot",
        help="Sort order (default: hot)",
    )
    parser.add_argument(
        "--time-filter",
        choices=["hour", "day", "week", "month", "year", "all"],
        default="day",
        help="Time filter for 'top' sort (default: day)",
    )
    parser.add_argument(
        "--min-score",
        type=int,
        default=5,
        help="Minimum post score to include (default: 5)",
    )
    parser.add_argument(
        "--max-age",
        type=int,
        default=7,
        help="Maximum post age in days (default: 7)",
    )
    parser.add_argument(
        "--dry-run",
        action="store_true",
        help="Don't save to database, just log what would be saved",
    )
    parser.add_argument(
        "--prune",
        type=int,
        metavar="DAYS",
        help="Prune posts older than N days (run after collection)",
    )
    parser.add_argument(
        "--stats",
        action="store_true",
        help="Show collection statistics and exit",
    )
    parser.add_argument(
        "-v", "--verbose",
        action="store_true",
        help="Enable verbose logging",
    )

    args = parser.parse_args()

    # Set log level
    if args.verbose:
        logging.getLogger().setLevel(logging.DEBUG)

    # Create collector
    collector = RedditSentimentCollector(
        subreddits=args.subreddits,
        dry_run=args.dry_run,
    )

    try:
        # Initialize
        collector.initialize()

        # Just show stats?
        if args.stats:
            stats = collector.get_stats()
            print("\nCollection Statistics:")
            print(f"  Total posts:      {stats.get('total_posts', 0)}")
            print(f"  Posts (24h):      {stats.get('posts_24h', 0)}")
            print(f"  Unique tickers:   {stats.get('unique_tickers', 0)}")

            sentiment = stats.get("sentiment_24h", {})
            if sentiment:
                print("\n  Sentiment (24h):")
                for s, count in sorted(sentiment.items()):
                    print(f"    {s}: {count}")

            top_subs = stats.get("top_subreddits", [])
            if top_subs:
                print("\n  Top subreddits (24h):")
                for sub in top_subs:
                    print(f"    r/{sub['subreddit']}: {sub['count']} posts")

            return

        # Run collection
        collector.run(
            limit_per_subreddit=args.limit,
            sort=args.sort,
            time_filter=args.time_filter,
            min_score=args.min_score,
            max_age_days=args.max_age,
        )

        # Prune old posts if requested
        if args.prune:
            collector.prune_old_posts(args.prune)

    except KeyboardInterrupt:
        logger.info("Interrupted by user")
        sys.exit(1)
    except Exception as e:
        logger.error(f"Fatal error: {e}")
        sys.exit(1)
    finally:
        collector.close()


if __name__ == "__main__":
    main()
