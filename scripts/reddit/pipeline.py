"""Unified Reddit Sentiment Pipeline.

Combines collection, AI processing, and aggregation into a single pipeline:

  Phase 1: COLLECT   - Fetch posts from Arctic Shift API -> reddit_posts_raw
  Phase 2: PROCESS   - Gemini LLM -> reddit_post_tickers
  Phase 3: AGGREGATE - Roll up into ticker_sentiment_snapshots + history

Runs as a single K8s CronJob (hourly), replacing the previous two-job
(collector + ai-processor) architecture.
"""

import argparse
import logging
import os
import sys

import psycopg2

from .aggregator import SentimentAggregator
from .ai_processor import RedditAIProcessor
from .database import Database
from .fetcher import RedditFetcher

logger = logging.getLogger(__name__)

# Default subreddits (same as existing collector CronJob)
DEFAULT_SUBREDDITS = [
    # Core trading/investing
    "wallstreetbets",
    "stocks",
    "options",
    "investing",
    "Daytrading",
    "StockMarket",
    "SecurityAnalysis",
    "ValueInvesting",
    # Additional trading
    "pennystocks",
    "thetagang",
    "smallstreetbets",
    "SPACs",
    "weedstocks",
    "RobinHood",
    "Superstonk",
    "ETFs",
    # Stock-specific
    "NVDA_Stock",
    "AMD_Stock",
    "intel",
    # Long-term investing
    "dividends",
    "Bogleheads",
    "financialindependence",
    "fatFIRE",
    "personalfinance",
    # Finance/Economics
    "FluentInFinance",
    "FinancialPlanning",
    "economy",
    "economics",
    # Crypto
    "CryptoCurrency",
    "Bitcoin",
]


def run_collect(db, fetcher, subreddits, limit, sort, min_score,
                max_age):
    """Phase 1: Fetch posts from Reddit via Arctic Shift API.

    Args:
        db: Database instance
        fetcher: RedditFetcher instance
        subreddits: List of subreddit names
        limit: Max posts per subreddit
        sort: Sort order (new, hot, top)
        min_score: Minimum post score
        max_age: Maximum post age in days

    Returns:
        Total number of posts collected
    """
    logger.info("=" * 50)
    logger.info("Phase 1: COLLECT")
    logger.info(f"  Subreddits: {len(subreddits)}")
    logger.info(f"  Limit per subreddit: {limit}")
    logger.info(f"  Sort: {sort}, Min score: {min_score}")
    logger.info(f"  Max age: {max_age} days")
    logger.info("=" * 50)

    total_collected = 0

    for subreddit in subreddits:
        try:
            posts = fetcher.fetch_subreddit(
                subreddit=subreddit,
                sort=sort,
                limit=limit,
                min_score=min_score,
                max_age_days=max_age,
            )

            if posts:
                inserted = db.bulk_upsert_raw_posts(posts)
                total_collected += inserted
                logger.info(
                    f"  r/{subreddit}: {len(posts)} fetched, "
                    f"{inserted} upserted"
                )
            else:
                logger.info(f"  r/{subreddit}: no posts found")

        except Exception as e:
            logger.error(f"  r/{subreddit}: failed - {e}")
            continue

    logger.info(f"Collection complete: {total_collected} posts upserted")
    return total_collected


def run_process(conn, batch_size, min_upvotes, min_comments,
                model, max_posts, process_all):
    """Phase 2: Extract tickers + sentiment via Gemini LLM.

    Args:
        conn: psycopg2 connection (shared)
        batch_size: Posts per LLM call
        min_upvotes: Minimum upvotes to process
        min_comments: Minimum comments to process
        model: Gemini model name
        max_posts: Maximum posts to process
        process_all: If True, process all unprocessed posts

    Returns:
        Number of posts processed
    """
    logger.info("=" * 50)
    logger.info("Phase 2: PROCESS (AI extraction)")
    logger.info(f"  Model: {model}")
    logger.info(f"  Batch size: {batch_size}")
    logger.info(f"  Min engagement: {min_upvotes} upvotes OR "
                f"{min_comments} comments")
    logger.info("=" * 50)

    processor = RedditAIProcessor(
        batch_size=batch_size,
        min_upvotes=min_upvotes,
        min_comments=min_comments,
        model=model,
        conn=conn,
    )

    try:
        processor.connect()
        processor.run(
            max_posts=max_posts,
            process_all=process_all,
        )
        return processor.stats["posts_processed"]
    finally:
        processor.close()


def run_aggregate(conn, time_ranges):
    """Phase 3: Aggregate per-post data into per-ticker snapshots.

    Args:
        conn: psycopg2 connection (shared)
        time_ranges: List of time range labels

    Returns:
        Number of snapshots upserted
    """
    logger.info("=" * 50)
    logger.info("Phase 3: AGGREGATE")
    logger.info(f"  Time ranges: {time_ranges}")
    logger.info("=" * 50)

    aggregator = SentimentAggregator(conn)
    aggregator.run(time_ranges=time_ranges)
    return aggregator.stats["snapshots_upserted"]


def main():
    """Main entry point for the unified pipeline."""
    parser = argparse.ArgumentParser(
        description=(
            "Unified Reddit Sentiment Pipeline: "
            "collect + process + aggregate"
        ),
    )

    # Phase control
    parser.add_argument(
        "--skip-collect",
        action="store_true",
        help="Skip Phase 1 (collection from Reddit)",
    )
    parser.add_argument(
        "--skip-process",
        action="store_true",
        help="Skip Phase 2 (AI processing with Gemini)",
    )
    parser.add_argument(
        "--skip-aggregate",
        action="store_true",
        help="Skip Phase 3 (aggregation into snapshots)",
    )

    # Collector args
    parser.add_argument(
        "--subreddits",
        nargs="+",
        default=DEFAULT_SUBREDDITS,
        help="Subreddits to collect from",
    )
    parser.add_argument(
        "--limit",
        type=int,
        default=1000,
        help="Max posts per subreddit (default: 1000)",
    )
    parser.add_argument(
        "--sort",
        default="new",
        choices=["new", "hot", "top"],
        help="Sort order (default: new)",
    )
    parser.add_argument(
        "--min-score",
        type=int,
        default=1,
        help="Minimum post score (default: 1)",
    )
    parser.add_argument(
        "--max-age",
        type=int,
        default=14,
        help="Maximum post age in days (default: 14)",
    )

    # Processor args
    parser.add_argument(
        "--process-all",
        action="store_true",
        default=True,
        help="Process all unprocessed posts (default: True)",
    )
    parser.add_argument(
        "--batch-size",
        type=int,
        default=20,
        help="Posts per LLM call (default: 20)",
    )
    parser.add_argument(
        "--min-upvotes",
        type=int,
        default=1,
        help="Minimum upvotes to process (default: 1)",
    )
    parser.add_argument(
        "--min-comments",
        type=int,
        default=1,
        help="Minimum comments to process (default: 1)",
    )
    parser.add_argument(
        "--model",
        default="gemini-2.5-flash-lite",
        help="Gemini model (default: gemini-2.5-flash-lite)",
    )
    parser.add_argument(
        "--max-posts",
        type=int,
        default=500,
        help=(
            "Max posts to process per run "
            "(default: 500, ignored with --process-all)"
        ),
    )

    # Aggregator args
    parser.add_argument(
        "--time-ranges",
        nargs="+",
        default=["1d", "7d", "14d", "30d"],
        help="Time ranges for snapshots (default: 1d 7d 14d 30d)",
    )

    # Housekeeping
    parser.add_argument(
        "--prune",
        type=int,
        default=60,
        help="Prune raw posts older than N days (default: 60)",
    )

    # General
    parser.add_argument(
        "-v",
        "--verbose",
        action="store_true",
        help="Enable verbose logging",
    )

    args = parser.parse_args()

    # Configure logging
    logging.basicConfig(
        level=logging.DEBUG if args.verbose else logging.INFO,
        format="%(asctime)s - %(name)s - %(levelname)s - %(message)s",
    )

    logger.info("=" * 60)
    logger.info("Reddit Sentiment Pipeline - Starting")
    logger.info("=" * 60)

    # Connect to database (shared connection for all phases)
    db = Database()

    try:
        db.connect()
        conn = db.get_connection()

        # Phase 1: Collect
        if not args.skip_collect:
            fetcher = RedditFetcher()
            run_collect(
                db=db,
                fetcher=fetcher,
                subreddits=args.subreddits,
                limit=args.limit,
                sort=args.sort,
                min_score=args.min_score,
                max_age=args.max_age,
            )
        else:
            logger.info("Skipping Phase 1 (collect)")

        # Phase 2: Process
        if not args.skip_process:
            run_process(
                conn=conn,
                batch_size=args.batch_size,
                min_upvotes=args.min_upvotes,
                min_comments=args.min_comments,
                model=args.model,
                max_posts=args.max_posts,
                process_all=args.process_all,
            )
        else:
            logger.info("Skipping Phase 2 (process)")

        # Phase 3: Aggregate
        if not args.skip_aggregate:
            run_aggregate(
                conn=conn,
                time_ranges=args.time_ranges,
            )
        else:
            logger.info("Skipping Phase 3 (aggregate)")

        # Housekeeping: prune old raw posts
        if args.prune > 0:
            pruned = db.prune_old_raw_posts(
                retention_days=args.prune
            )
            logger.info(
                f"Housekeeping: pruned {pruned} raw posts "
                f"older than {args.prune} days"
            )

        logger.info("=" * 60)
        logger.info("Reddit Sentiment Pipeline - Complete")
        logger.info("=" * 60)

    except KeyboardInterrupt:
        logger.info("Interrupted by user")
        sys.exit(1)
    except Exception as e:
        logger.error(f"Pipeline failed: {e}", exc_info=True)
        sys.exit(1)
    finally:
        db.close()


if __name__ == "__main__":
    main()
