"""AI-powered ticker extraction from Reddit posts using Gemini.

This processor fetches unprocessed posts from reddit_posts_raw and uses
Google's Gemini to extract tickers, map company names, determine sentiment,
and assess post quality for investment analysis.

Cost optimizations:
1. Batch processing (20 posts per API call)
2. Engagement threshold (skip low-engagement posts)
3. Company name cache (skip LLM for known mappings)

Gemini 2.5 Flash-Lite pricing (as of Feb 2026, verify at
https://ai.google.dev/pricing): ~$0.10/1M input, ~$0.40/1M output tokens
Estimated cost: ~$1.70/month for 30,000 posts
"""

import json
import logging
import os
import re
import time
from dataclasses import dataclass
from datetime import datetime
from typing import Dict, List, Optional, Tuple

import google.generativeai as genai
import psycopg2
from psycopg2.extras import execute_batch

logger = logging.getLogger(__name__)


# Extraction prompt template for batch processing
EXTRACTION_PROMPT = """Analyze these {n} Reddit posts and extract stock/ETF ticker mentions.

{posts_json}

For each post, return a JSON object with:
- post_id: The ID of the post
- tickers: Array of ticker objects, each with:
  - symbol: Uppercase ticker symbol (e.g., "AAPL", "TSLA")
  - sentiment: "bullish", "bearish", or "neutral"
  - confidence: 0.0 to 1.0
  - is_primary: true if this is the main topic
  - mention_type: "ticker", "company_name", or "abbreviation"
- is_finance_related: true if the post is about investing/trading
- spam_score: 0.0 to 1.0 (high for pump-and-dump or promotional posts)
- quality_score: 0.0 to 1.0 (investment analysis quality of the post)

Rules:
1. Map company names to tickers (e.g., "Oracle" -> "ORCL", "Tesla" -> "TSLA")
2. Handle common abbreviations (e.g., "Nvidia" -> "NVDA")
3. Sentiment is PER-TICKER - one post can be bullish on X, bearish on Y
4. Only include US-listed stocks and ETFs
5. Ignore crypto unless it's a crypto ETF (like BITO, GBTC)
6. Mark is_primary=true for the main ticker being discussed
7. If no valid tickers, return empty tickers array
8. Set spam_score > 0.5 for pump-and-dump or promotional posts
9. quality_score reflects overall post quality for investment sentiment analysis.
   High (0.7-1.0): original analysis, DD posts, earnings discussion with data,
   specific catalysts, well-reasoned bull/bear thesis.
   Medium (0.3-0.7): news sharing, basic price commentary, general market
   discussion with some substance.
   Low (0.0-0.3): memes, one-liners, reposts, vague hype, "to the moon"
   type posts, off-topic, emotional rants without substance.

Return ONLY valid JSON array, no other text:
[
  {{"post_id": "abc123", "tickers": [...], "is_finance_related": true, "spam_score": 0.1, "quality_score": 0.75}},
  ...
]"""


@dataclass
class TickerExtraction:
    """Represents an extracted ticker from a post."""
    post_id: int
    ticker: str
    sentiment: str
    confidence: float
    is_primary: bool
    mention_type: str


@dataclass
class ProcessingResult:
    """Results from processing a batch of posts."""
    extractions: List[TickerExtraction]
    posts_processed: int
    posts_skipped: int
    cache_hits: int
    llm_calls: int


class RedditAIProcessor:
    """AI-powered processor for extracting tickers from Reddit posts."""

    # Gemini 2.5 Flash-Lite - best cost/quality for structured extraction
    DEFAULT_MODEL = "gemini-2.5-flash-lite"

    # Engagement thresholds
    MIN_UPVOTES = 5
    MIN_COMMENTS = 3

    # Batch size for LLM calls (20 posts * ~90 tokens avg output = ~1800 tokens)
    BATCH_SIZE = 20

    def __init__(
        self,
        db_host: str = None,
        db_port: int = None,
        db_user: str = None,
        db_password: str = None,
        db_name: str = None,
        google_api_key: str = None,
        model: str = None,
        batch_size: int = None,
        min_upvotes: int = None,
        min_comments: int = None,
        conn=None,
    ):
        """Initialize processor.

        Args:
            db_host: Database host
            db_port: Database port
            db_user: Database user
            db_password: Database password
            db_name: Database name
            google_api_key: Google API key for Gemini
            model: Gemini model to use
            batch_size: Posts per LLM call
            min_upvotes: Minimum upvotes to process
            min_comments: Minimum comments to process
            conn: Optional shared psycopg2 connection (from pipeline)
        """
        # Database config
        self.db_host = db_host or os.getenv("DB_HOST", "localhost")
        self.db_port = db_port or int(os.getenv("DB_PORT", "5432"))
        self.db_user = db_user or os.getenv("DB_USER", "investorcenter")
        self.db_password = db_password or os.getenv("DB_PASSWORD", "")
        self.db_name = db_name or os.getenv("DB_NAME", "investorcenter_db")

        # Shared connection (set before connect() is called)
        self._shared_conn = conn

        # Google API config
        api_key = google_api_key or os.getenv("GOOGLE_API_KEY")
        if not api_key:
            raise ValueError("GOOGLE_API_KEY environment variable required")

        genai.configure(api_key=api_key)
        self.model_name = model or self.DEFAULT_MODEL
        self.model = genai.GenerativeModel(self.model_name)

        # Processing config
        self.batch_size = batch_size or self.BATCH_SIZE
        self.min_upvotes = min_upvotes if min_upvotes is not None else self.MIN_UPVOTES
        self.min_comments = min_comments if min_comments is not None else self.MIN_COMMENTS

        # Database connection
        self.conn = None

        # Company name cache (loaded from DB)
        self.company_cache: Dict[str, str] = {}

        # Stats
        self.stats = {
            "posts_processed": 0,
            "posts_skipped": 0,
            "tickers_extracted": 0,
            "cache_hits": 0,
            "llm_calls": 0,
            "errors": 0,
        }

    def connect(self):
        """Establish database connection (or reuse shared connection)."""
        try:
            if self._shared_conn is not None:
                self.conn = self._shared_conn
                # Ensure transactional mode — our _save_extractions
                # relies on explicit commit/rollback.
                self.conn.autocommit = False
                logger.info("Using shared database connection")
            else:
                self.conn = psycopg2.connect(
                    host=self.db_host,
                    port=self.db_port,
                    user=self.db_user,
                    password=self.db_password,
                    database=self.db_name,
                )
                self.conn.autocommit = False
                logger.info(
                    f"Connected to database {self.db_name}@{self.db_host}"
                )

            # Load company name cache
            self._load_company_cache()

        except Exception as e:
            logger.error(f"Failed to connect to database: {e}")
            raise

    def close(self):
        """Close database connection (skips if using shared connection)."""
        if self.conn and self._shared_conn is None:
            self.conn.close()
            logger.info("Database connection closed")

    def _load_company_cache(self):
        """Load company name to ticker mappings from database."""
        try:
            cursor = self.conn.cursor()
            cursor.execute("""
                SELECT company_name, ticker
                FROM company_ticker_cache
                ORDER BY hit_count DESC
            """)

            for row in cursor.fetchall():
                self.company_cache[row[0].lower()] = row[1]

            cursor.close()
            logger.info(f"Loaded {len(self.company_cache)} company name mappings")

        except Exception as e:
            logger.warning(f"Failed to load company cache: {e}")

    def _update_company_cache(self, company_name: str, ticker: str, source: str = "llm"):
        """Update company name cache in database.

        Args:
            company_name: Company name (will be lowercased)
            ticker: Ticker symbol
            source: Source of mapping (llm, manual, sec)
        """
        try:
            cursor = self.conn.cursor()
            cursor.execute("""
                INSERT INTO company_ticker_cache (company_name, ticker, source, hit_count)
                VALUES (%s, %s, %s, 1)
                ON CONFLICT (company_name, ticker) DO UPDATE SET
                    hit_count = company_ticker_cache.hit_count + 1,
                    last_used_at = NOW()
            """, (company_name.lower(), ticker.upper(), source))
            self.conn.commit()
            cursor.close()

            # Update local cache
            self.company_cache[company_name.lower()] = ticker.upper()

        except Exception as e:
            logger.warning(f"Failed to update company cache: {e}")
            self.conn.rollback()

    def _check_company_cache(self, text: str) -> List[Tuple[str, str]]:
        """Check if text contains known company names.

        Args:
            text: Text to search

        Returns:
            List of (company_name, ticker) tuples found
        """
        found = []
        text_lower = text.lower()

        for company_name, ticker in self.company_cache.items():
            # Check for whole word match
            if re.search(rf'\b{re.escape(company_name)}\b', text_lower):
                found.append((company_name, ticker))
                self.stats["cache_hits"] += 1

        return found

    def get_unprocessed_posts(self, limit: int = 100) -> List[dict]:
        """Get unprocessed posts from database.

        Args:
            limit: Maximum posts to fetch

        Returns:
            List of post dicts
        """
        try:
            cursor = self.conn.cursor()
            cursor.execute("""
                SELECT id, external_id, subreddit, title, body, url,
                       upvotes, comment_count, flair, posted_at
                FROM reddit_posts_raw
                WHERE processed_at IS NULL
                  AND processing_skipped = FALSE
                ORDER BY fetched_at ASC
                LIMIT %s
            """, (limit,))

            posts = []
            for row in cursor.fetchall():
                posts.append({
                    "id": row[0],
                    "external_id": row[1],
                    "subreddit": row[2],
                    "title": row[3],
                    "body": row[4] or "",
                    "url": row[5],
                    "upvotes": row[6],
                    "comment_count": row[7],
                    "flair": row[8],
                    "posted_at": row[9],
                })

            cursor.close()
            return posts

        except Exception as e:
            logger.error(f"Failed to get unprocessed posts: {e}")
            return []

    def _should_process(self, post: dict) -> bool:
        """Check if post meets engagement threshold.

        Args:
            post: Post dict

        Returns:
            True if post should be processed
        """
        return (
            post["upvotes"] >= self.min_upvotes or
            post["comment_count"] >= self.min_comments
        )

    def _mark_skipped(self, post_ids: List[int]):
        """Mark posts as skipped due to low engagement.

        Args:
            post_ids: List of post IDs to mark
        """
        if not post_ids:
            return

        try:
            cursor = self.conn.cursor()
            cursor.execute("""
                UPDATE reddit_posts_raw
                SET processing_skipped = TRUE, processed_at = NOW()
                WHERE id = ANY(%s)
            """, (post_ids,))
            self.conn.commit()
            cursor.close()

        except Exception as e:
            logger.error(f"Failed to mark posts as skipped: {e}")
            self.conn.rollback()

    def _call_llm(self, posts: List[dict]) -> Optional[List[dict]]:
        """Call Gemini to extract tickers from posts.

        Args:
            posts: List of post dicts to process

        Returns:
            List of extraction results or None on error
        """
        if not posts:
            return []

        # Format posts for prompt
        posts_for_prompt = []
        for post in posts:
            # Truncate body for prompt
            body = post["body"][:1500] if post["body"] else ""
            posts_for_prompt.append({
                "post_id": post["external_id"],
                "subreddit": post["subreddit"],
                "title": post["title"],
                "body": body,
                "flair": post["flair"] or "",
            })

        prompt = EXTRACTION_PROMPT.format(
            n=len(posts),
            posts_json=json.dumps(posts_for_prompt, indent=2),
        )

        try:
            self.stats["llm_calls"] += 1

            # Call Gemini
            response = self.model.generate_content(
                prompt,
                generation_config=genai.types.GenerationConfig(
                    max_output_tokens=4000,
                    temperature=0.1,  # Low temperature for structured extraction
                ),
            )

            # Parse JSON response
            response_text = response.text.strip()

            # Try to extract JSON from response
            # Sometimes the model adds explanation text
            json_match = re.search(r'\[[\s\S]*\]', response_text)
            if json_match:
                response_text = json_match.group(0)

            return json.loads(response_text)

        except json.JSONDecodeError as e:
            logger.error(f"Failed to parse LLM response: {e}")
            logger.debug(f"Response was: {response_text[:500]}")
            self.stats["errors"] += 1
            return None

        except Exception as e:
            logger.error(f"LLM call failed: {e}")
            self.stats["errors"] += 1
            return None

    def _save_extractions(self, post_id: int, external_id: str, result: dict):
        """Save ticker extractions to database.

        Args:
            post_id: Database post ID
            external_id: Reddit post ID
            result: LLM extraction result
        """
        try:
            cursor = self.conn.cursor()

            # Save ticker extractions
            tickers = result.get("tickers", [])
            for ticker_data in tickers:
                # Skip if no valid ticker symbol
                symbol = ticker_data.get("symbol")
                if not symbol:
                    continue

                cursor.execute("""
                    INSERT INTO reddit_post_tickers (
                        post_id, ticker, sentiment, confidence,
                        is_primary, mention_type
                    ) VALUES (%s, %s, %s, %s, %s, %s)
                    ON CONFLICT (post_id, ticker) DO UPDATE SET
                        sentiment = EXCLUDED.sentiment,
                        confidence = EXCLUDED.confidence,
                        is_primary = EXCLUDED.is_primary,
                        extracted_at = NOW()
                """, (
                    post_id,
                    symbol.upper(),
                    ticker_data.get("sentiment", "neutral"),
                    ticker_data.get("confidence", 0.5),
                    ticker_data.get("is_primary", False),
                    ticker_data.get("mention_type", "ticker"),
                ))

                self.stats["tickers_extracted"] += 1

                # Update company cache if we learned a new mapping
                mention_type = ticker_data.get("mention_type", "")
                if mention_type == "company_name":
                    # We need to find the company name from the post
                    # For now, just track that we extracted it
                    pass

            # Update post as processed
            cursor.execute("""
                UPDATE reddit_posts_raw
                SET processed_at = NOW(),
                    llm_response = %s,
                    llm_model = %s,
                    is_finance_related = %s,
                    spam_score = %s,
                    quality_score = %s
                WHERE id = %s
            """, (
                json.dumps(result),
                self.model_name,
                result.get("is_finance_related", True),
                result.get("spam_score", 0.0),
                # NULL if LLM doesn't return quality_score;
                # aggregator defaults NULL to 0.5 in scoring
                result.get("quality_score"),
                post_id,
            ))

            self.conn.commit()
            cursor.close()
            self.stats["posts_processed"] += 1

        except Exception as e:
            logger.error(f"Failed to save extractions for post {external_id}: {e}")
            self.conn.rollback()
            self.stats["errors"] += 1

    def process_batch(self, limit: int = 100) -> ProcessingResult:
        """Process a batch of unprocessed posts.

        Args:
            limit: Maximum posts to process

        Returns:
            ProcessingResult with per-batch stats (not cumulative)
        """
        # Track starting stats to calculate per-batch deltas
        start_processed = self.stats["posts_processed"]
        start_skipped = self.stats["posts_skipped"]
        start_cache_hits = self.stats["cache_hits"]
        start_llm_calls = self.stats["llm_calls"]

        posts = self.get_unprocessed_posts(limit=limit)

        if not posts:
            logger.info("No unprocessed posts found")
            return ProcessingResult(
                extractions=[],
                posts_processed=0,
                posts_skipped=0,
                cache_hits=0,
                llm_calls=0,
            )

        logger.info(f"Found {len(posts)} unprocessed posts")

        # Filter by engagement
        posts_to_process = []
        posts_to_skip = []

        for post in posts:
            if self._should_process(post):
                posts_to_process.append(post)
            else:
                posts_to_skip.append(post["id"])

        # Mark low-engagement posts as skipped
        batch_skipped = len(posts_to_skip)
        if posts_to_skip:
            self._mark_skipped(posts_to_skip)
            self.stats["posts_skipped"] += batch_skipped
            logger.info(f"Skipped {batch_skipped} low-engagement posts")

        if not posts_to_process:
            logger.info("No posts met engagement threshold")
            return ProcessingResult(
                extractions=[],
                posts_processed=0,
                posts_skipped=batch_skipped,
                cache_hits=0,
                llm_calls=0,
            )

        logger.info(f"Processing {len(posts_to_process)} posts with engagement")

        # Process in batches
        all_extractions = []

        for i in range(0, len(posts_to_process), self.batch_size):
            batch = posts_to_process[i:i + self.batch_size]
            logger.info(f"Processing batch {i // self.batch_size + 1} ({len(batch)} posts)")

            # Call LLM
            results = self._call_llm(batch)

            if results is None:
                logger.error("LLM call failed, skipping batch")
                continue

            # Create mapping from external_id to database id
            id_mapping = {p["external_id"]: p["id"] for p in batch}

            # Save results
            for result in results:
                external_id = result.get("post_id")
                if external_id and external_id in id_mapping:
                    post_id = id_mapping[external_id]
                    self._save_extractions(post_id, external_id, result)

            # Rate limit between batches
            if i + self.batch_size < len(posts_to_process):
                time.sleep(1)  # Small delay between batches

        # Return per-batch stats (deltas from start)
        return ProcessingResult(
            extractions=all_extractions,
            posts_processed=self.stats["posts_processed"] - start_processed,
            posts_skipped=self.stats["posts_skipped"] - start_skipped,
            cache_hits=self.stats["cache_hits"] - start_cache_hits,
            llm_calls=self.stats["llm_calls"] - start_llm_calls,
        )

    def run(self, max_posts: int = 500, process_all: bool = False):
        """Run the processor.

        Args:
            max_posts: Maximum total posts to process (ignored if process_all=True)
            process_all: If True, process ALL unprocessed posts (no limit)
        """
        if process_all:
            logger.info("Starting AI processor (processing ALL unprocessed posts)...")
        else:
            logger.info(f"Starting AI processor (max {max_posts} posts)...")
        logger.info(f"  Model: {self.model_name}")
        logger.info(f"  Batch size: {self.batch_size}")
        logger.info(f"  Min engagement: {self.min_upvotes} upvotes OR {self.min_comments} comments")

        processed = 0
        batch_num = 0

        while process_all or processed < max_posts:
            batch_num += 1

            if process_all:
                batch_limit = 100  # Process 100 posts at a time when processing all
            else:
                remaining = max_posts - processed
                batch_limit = min(100, remaining)

            logger.info(f"\n--- Outer Batch {batch_num} (limit: {batch_limit}) ---")

            result = self.process_batch(limit=batch_limit)

            if result.posts_processed == 0 and result.posts_skipped == 0:
                logger.info("No more posts to process")
                break

            processed += result.posts_processed + result.posts_skipped
            logger.info(f"Progress: {processed} posts processed/skipped so far (cumulative: {self.stats['posts_processed']} processed, {self.stats['posts_skipped']} skipped)")

            # Rate limit between batches
            time.sleep(2)

        # Log summary
        self._log_summary()

    def _log_summary(self):
        """Log processing summary."""
        logger.info("=" * 50)
        logger.info("AI Processing Summary:")
        logger.info(f"  Posts processed:    {self.stats['posts_processed']}")
        logger.info(f"  Posts skipped:      {self.stats['posts_skipped']}")
        logger.info(f"  Tickers extracted:  {self.stats['tickers_extracted']}")
        logger.info(f"  LLM calls:          {self.stats['llm_calls']}")
        logger.info(f"  Cache hits:         {self.stats['cache_hits']}")
        logger.info(f"  Errors:             {self.stats['errors']}")
        logger.info("=" * 50)


def main():
    """Main entry point."""
    import argparse

    parser = argparse.ArgumentParser(
        description="Process Reddit posts with AI to extract tickers and sentiment"
    )
    parser.add_argument(
        "--max-posts",
        type=int,
        default=500,
        help="Maximum posts to process (default: 500, ignored with --process-all)",
    )
    parser.add_argument(
        "--process-all",
        action="store_true",
        help="Process ALL unprocessed posts (ignores --max-posts)",
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
        default=5,
        help="Minimum upvotes to process (default: 5)",
    )
    parser.add_argument(
        "--min-comments",
        type=int,
        default=3,
        help="Minimum comments to process (default: 3)",
    )
    parser.add_argument(
        "--model",
        default="gemini-2.5-flash-lite",
        help="Gemini model to use (default: gemini-2.5-flash-lite)",
    )
    parser.add_argument(
        "-v", "--verbose",
        action="store_true",
        help="Enable verbose logging",
    )

    args = parser.parse_args()

    # Configure logging
    logging.basicConfig(
        level=logging.DEBUG if args.verbose else logging.INFO,
        format="%(asctime)s - %(name)s - %(levelname)s - %(message)s",
    )

    # Create and run processor
    processor = RedditAIProcessor(
        batch_size=args.batch_size,
        min_upvotes=args.min_upvotes,
        min_comments=args.min_comments,
        model=args.model,
    )

    try:
        processor.connect()
        processor.run(max_posts=args.max_posts, process_all=args.process_all)
    except KeyboardInterrupt:
        logger.info("Interrupted by user")
    except Exception as e:
        logger.error(f"Fatal error: {e}")
        raise
    finally:
        processor.close()


if __name__ == "__main__":
    main()
