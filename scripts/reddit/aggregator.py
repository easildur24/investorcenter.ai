"""Sentiment aggregation: per-post data -> per-ticker snapshots + history.

Reads from reddit_post_tickers + reddit_posts_raw, computes per-ticker
aggregated metrics, and writes to ticker_sentiment_snapshots (for the
trending page) and ticker_sentiment_history (for time-series charts).

The composite_score is primarily driven by LLM quality scores (40%),
supplemented by mention volume, engagement, and sentiment intensity.
"""

import json
import logging
import math
from datetime import datetime, timezone
from typing import Dict, List, Optional

logger = logging.getLogger(__name__)


# Mapping from time range label to number of days
TIME_RANGE_DAYS = {
    "1d": 1,
    "7d": 7,
    "14d": 14,
    "30d": 30,
}

# Default time range used for history point insertion
HISTORY_TIME_RANGE = "7d"


class SentimentAggregator:
    """Aggregates per-post sentiment data into per-ticker snapshots."""

    def __init__(self, conn, min_mentions: int = 2):
        """Initialize aggregator.

        Args:
            conn: psycopg2 connection (shared with pipeline).
                  Must have autocommit=False.
            min_mentions: Minimum mentions for a ticker to appear
                in snapshots (default: 2).
        """
        self.conn = conn
        self.min_mentions = min_mentions
        self.snapshot_time = datetime.now(timezone.utc)
        self.stats = {
            "time_ranges_processed": 0,
            "snapshots_upserted": 0,
            "history_points_inserted": 0,
        }

    def run(self, time_ranges: Optional[List[str]] = None):
        """Run aggregation for all time ranges.

        Args:
            time_ranges: List of time range labels (default: all)
        """
        time_ranges = time_ranges or list(TIME_RANGE_DAYS.keys())

        logger.info(
            f"Starting aggregation for {len(time_ranges)} "
            f"time ranges "
            f"(snapshot_time={self.snapshot_time.isoformat()})"
        )

        # Store aggregations keyed by time range for history
        all_snapshots: Dict[str, List[dict]] = {}

        for tr in time_ranges:
            if tr not in TIME_RANGE_DAYS:
                logger.warning(
                    f"Unknown time range: {tr}, skipping"
                )
                continue

            snapshots = self._aggregate_time_range(tr)
            self.stats["time_ranges_processed"] += 1
            all_snapshots[tr] = snapshots

        # Insert history points from HISTORY_TIME_RANGE
        history_snapshots = all_snapshots.get(HISTORY_TIME_RANGE)
        if history_snapshots:
            self._write_history_points(history_snapshots)
        elif HISTORY_TIME_RANGE not in time_ranges:
            logger.warning(
                f"'{HISTORY_TIME_RANGE}' not in requested "
                f"time_ranges {time_ranges}; no history "
                f"points written"
            )

        self._log_summary()

    def _aggregate_time_range(
        self, time_range: str
    ) -> List[dict]:
        """Aggregate metrics for a single time range.

        Args:
            time_range: Time range label (e.g., "7d")

        Returns:
            List of snapshot dicts (one per ticker)
        """
        days = TIME_RANGE_DAYS[time_range]
        logger.info(
            f"Aggregating time_range={time_range} "
            f"({days} days)"
        )

        # Step A: Query per-ticker aggregated metrics
        raw_aggs = self._query_aggregations(days)
        if not raw_aggs:
            logger.info(
                f"  No tickers found for {time_range}"
            )
            return []

        logger.info(
            f"  Found {len(raw_aggs)} tickers "
            f"with >= {self.min_mentions} mentions"
        )

        # Step B: Compute composite scores
        for agg in raw_aggs:
            agg["composite_score"] = (
                self._compute_composite_score(agg)
            )

        # Step C: Compute velocity metrics
        velocities = self._compute_velocities(time_range)

        # Step D: Compute ranks + previous ranks
        previous_ranks = self._get_previous_ranks(time_range)
        raw_aggs.sort(
            key=lambda x: x["composite_score"], reverse=True
        )
        for i, agg in enumerate(raw_aggs):
            agg["rank"] = i + 1
            prev = previous_ranks.get(agg["ticker"])
            agg["previous_rank"] = prev
            if prev is not None:
                agg["rank_change"] = prev - agg["rank"]
            else:
                agg["rank_change"] = None

        # Step E: Derive sentiment_label
        for agg in raw_aggs:
            score = agg["sentiment_score"]
            if score > 0.1:
                agg["sentiment_label"] = "bullish"
            elif score < -0.1:
                agg["sentiment_label"] = "bearish"
            else:
                agg["sentiment_label"] = "neutral"

        # Merge velocity data + compute sentiment_velocity_24h
        for agg in raw_aggs:
            ticker = agg["ticker"]
            vel = velocities.get(ticker, {})
            agg["mention_velocity_1h"] = vel.get(
                "mention_velocity_1h"
            )
            # sentiment_velocity_24h: diff between current
            # sentiment_score and the previous snapshot's value
            prev_sent = vel.get("prev_sentiment")
            if prev_sent is not None:
                agg["sentiment_velocity_24h"] = (
                    float(agg["sentiment_score"])
                    - float(prev_sent)
                )
            else:
                agg["sentiment_velocity_24h"] = None

        # Step F: Upsert into ticker_sentiment_snapshots
        self._upsert_snapshots(raw_aggs, time_range)

        return raw_aggs

    def _query_aggregations(self, days: int) -> List[dict]:
        """Query per-ticker aggregated metrics from post data.

        Args:
            days: Number of days to look back

        Returns:
            List of dicts with per-ticker metrics
        """
        query = """
            SELECT
                rpt.ticker,
                COUNT(DISTINCT rpt.post_id)
                    AS unique_posts,
                COUNT(*) AS mention_count,
                COALESCE(SUM(rpr.upvotes), 0)
                    AS total_upvotes,
                COALESCE(SUM(rpr.comment_count), 0)
                    AS total_comments,
                COUNT(*) FILTER (
                    WHERE rpt.sentiment = 'bullish'
                ) AS bullish_count,
                COUNT(*) FILTER (
                    WHERE rpt.sentiment = 'neutral'
                ) AS neutral_count,
                COUNT(*) FILTER (
                    WHERE rpt.sentiment = 'bearish'
                ) AS bearish_count,
                COALESCE(
                    SUM(
                        CASE rpt.sentiment
                            WHEN 'bullish'
                                THEN rpt.confidence
                            WHEN 'bearish'
                                THEN -rpt.confidence
                            ELSE 0
                        END
                    ) / NULLIF(SUM(rpt.confidence), 0),
                    0
                ) AS sentiment_score,
                -- quality_score NULL means "not assessed"
                -- and defaults to 0.5 via COALESCE below
                COALESCE(
                    AVG(rpr.quality_score) FILTER (
                        WHERE rpr.quality_score IS NOT NULL
                    ),
                    0.5
                ) AS avg_quality_score
            FROM reddit_post_tickers rpt
            JOIN reddit_posts_raw rpr
                ON rpt.post_id = rpr.id
            WHERE rpr.posted_at
                > NOW() - make_interval(days => %s)
              AND rpr.is_finance_related = true
              AND COALESCE(rpr.spam_score, 0) < 0.5
            GROUP BY rpt.ticker
            HAVING COUNT(*) >= %s
            ORDER BY COUNT(*) DESC
        """

        cursor = self.conn.cursor()
        cursor.execute(query, (days, self.min_mentions))
        columns = [desc[0] for desc in cursor.description]
        rows = cursor.fetchall()
        cursor.close()

        results = []
        for row in rows:
            agg = dict(zip(columns, row))
            # Compute percentages
            total = (
                agg["bullish_count"]
                + agg["neutral_count"]
                + agg["bearish_count"]
            )
            if total > 0:
                agg["bullish_pct"] = (
                    agg["bullish_count"] / total
                )
                agg["neutral_pct"] = (
                    agg["neutral_count"] / total
                )
                agg["bearish_pct"] = (
                    agg["bearish_count"] / total
                )
            else:
                agg["bullish_pct"] = 0.0
                agg["neutral_pct"] = 1.0
                agg["bearish_pct"] = 0.0

            results.append(agg)

        # Query subreddit distribution for each ticker
        self._add_subreddit_distributions(results, days)

        return results

    def _add_subreddit_distributions(
        self, aggregations: List[dict], days: int
    ):
        """Add subreddit distribution JSON to each ticker.

        Note: runs one query per time range. Could be folded
        into _query_aggregations for perf, but kept separate
        for clarity since this is a CronJob (not latency
        sensitive).

        Args:
            aggregations: List of per-ticker aggregation dicts
            days: Number of days to look back
        """
        if not aggregations:
            return

        tickers = [agg["ticker"] for agg in aggregations]

        query = """
            SELECT
                rpt.ticker,
                rpr.subreddit,
                COUNT(*) AS cnt
            FROM reddit_post_tickers rpt
            JOIN reddit_posts_raw rpr
                ON rpt.post_id = rpr.id
            WHERE rpr.posted_at
                > NOW() - make_interval(days => %s)
              AND rpr.is_finance_related = true
              AND COALESCE(rpr.spam_score, 0) < 0.5
              AND rpt.ticker = ANY(%s)
            GROUP BY rpt.ticker, rpr.subreddit
            ORDER BY rpt.ticker, cnt DESC
        """

        cursor = self.conn.cursor()
        cursor.execute(query, (days, tickers))

        # Build distribution map
        dist_map: Dict[str, dict] = {}
        for row in cursor.fetchall():
            ticker, subreddit, cnt = row
            if ticker not in dist_map:
                dist_map[ticker] = {}
            dist_map[ticker][subreddit] = cnt

        cursor.close()

        # Attach to aggregations
        for agg in aggregations:
            dist = dist_map.get(agg["ticker"], {})
            agg["subreddit_distribution"] = json.dumps(dist)

    def _compute_composite_score(self, agg: dict) -> float:
        """Compute composite score from aggregated metrics.

        The composite score is primarily driven by LLM quality
        (40%), supplemented by mention volume, engagement, and
        sentiment intensity.

        Args:
            agg: Per-ticker aggregation dict

        Returns:
            Composite score (0.0 to 1.0)
        """
        avg_quality = float(
            agg.get("avg_quality_score", 0.5)
        )

        # Log-normalized mentions (cap ~100)
        mentions = int(agg.get("mention_count", 0))
        log_mentions = min(
            math.log2(mentions + 1) / math.log2(100), 1.0
        )

        # Log-normalized engagement (cap ~10K)
        engagement = int(agg.get("total_upvotes", 0)) + int(
            agg.get("total_comments", 0)
        )
        log_engagement = min(
            math.log2(engagement + 1) / math.log2(10000),
            1.0,
        )

        # Sentiment intensity (absolute value)
        sentiment_intensity = abs(
            float(agg.get("sentiment_score", 0))
        )

        composite = (
            avg_quality * 0.4
            + log_mentions * 0.25
            + log_engagement * 0.20
            + sentiment_intensity * 0.15
        )

        return round(composite, 6)

    def _safe_select(self, query, params=None):
        """Execute a read-only SELECT inside a savepoint.

        On failure, rolls back only the savepoint (not the
        whole transaction), keeping the shared connection
        usable for subsequent operations.

        Args:
            query: SQL SELECT query
            params: Query parameters

        Returns:
            List of rows, or empty list on failure
        """
        cursor = self.conn.cursor()
        try:
            cursor.execute("SAVEPOINT safe_select")
            cursor.execute(query, params)
            rows = cursor.fetchall()
            cursor.execute("RELEASE SAVEPOINT safe_select")
            return rows
        except Exception as e:
            logger.warning(f"SELECT failed: {e}")
            cursor.execute(
                "ROLLBACK TO SAVEPOINT safe_select"
            )
            return []
        finally:
            cursor.close()

    def _compute_velocities(
        self, time_range: str
    ) -> Dict[str, dict]:
        """Compute velocity metrics for all tickers.

        Returns a dict mapping ticker -> {
            mention_velocity_1h: float or None,
            prev_sentiment: float or None,
        }

        Args:
            time_range: Time range label

        Returns:
            Dict mapping ticker -> velocity metrics
        """
        velocities: Dict[str, dict] = {}

        # Mention velocity: compare last 1h vs prior 1h
        mention_query = """
            WITH recent AS (
                SELECT rpt.ticker, COUNT(*) AS cnt
                FROM reddit_post_tickers rpt
                JOIN reddit_posts_raw rpr
                    ON rpt.post_id = rpr.id
                WHERE rpr.posted_at
                    > NOW() - INTERVAL '1 hour'
                  AND rpr.is_finance_related = true
                GROUP BY rpt.ticker
            ),
            prior AS (
                SELECT rpt.ticker, COUNT(*) AS cnt
                FROM reddit_post_tickers rpt
                JOIN reddit_posts_raw rpr
                    ON rpt.post_id = rpr.id
                WHERE rpr.posted_at
                    BETWEEN NOW() - INTERVAL '2 hours'
                        AND NOW() - INTERVAL '1 hour'
                  AND rpr.is_finance_related = true
                GROUP BY rpt.ticker
            )
            SELECT
                r.ticker,
                (r.cnt - COALESCE(p.cnt, 0))::float
                    / GREATEST(COALESCE(p.cnt, 1), 1)
                    AS mention_velocity_1h
            FROM recent r
            LEFT JOIN prior p ON r.ticker = p.ticker
        """

        rows = self._safe_select(mention_query)
        for ticker, vel in rows:
            velocities[ticker] = {
                "mention_velocity_1h": vel,
            }

        # Previous sentiment: for computing velocity_24h
        prev_query = """
            SELECT DISTINCT ON (ticker)
                ticker, sentiment_score
            FROM ticker_sentiment_snapshots
            WHERE time_range = %s
              AND snapshot_time < %s
            ORDER BY ticker, snapshot_time DESC
        """

        rows = self._safe_select(
            prev_query, (time_range, self.snapshot_time)
        )
        for ticker, prev_score in rows:
            if ticker not in velocities:
                velocities[ticker] = {}
            velocities[ticker]["prev_sentiment"] = prev_score

        return velocities

    def _get_previous_ranks(
        self, time_range: str
    ) -> Dict[str, int]:
        """Get the most recent rank for each ticker.

        Args:
            time_range: Time range label

        Returns:
            Dict mapping ticker -> previous rank
        """
        query = """
            SELECT DISTINCT ON (ticker) ticker, rank
            FROM ticker_sentiment_snapshots
            WHERE time_range = %s
              AND snapshot_time < %s
              AND rank IS NOT NULL
            ORDER BY ticker, snapshot_time DESC
        """

        rows = self._safe_select(
            query, (time_range, self.snapshot_time)
        )
        return {row[0]: row[1] for row in rows}

    def _upsert_snapshots(
        self, aggregations: List[dict], time_range: str
    ):
        """Upsert aggregated data into ticker_sentiment_snapshots.

        Args:
            aggregations: List of per-ticker aggregation dicts
            time_range: Time range label
        """
        if not aggregations:
            return

        query = """
            INSERT INTO ticker_sentiment_snapshots (
                ticker, snapshot_time, time_range,
                mention_count, total_upvotes,
                total_comments, unique_posts,
                bullish_count, neutral_count,
                bearish_count,
                bullish_pct, neutral_pct, bearish_pct,
                sentiment_score, sentiment_label,
                mention_velocity_1h,
                sentiment_velocity_24h,
                composite_score, subreddit_distribution,
                rank, previous_rank, rank_change
            ) VALUES (
                %s, %s, %s, %s, %s, %s, %s, %s, %s, %s,
                %s, %s, %s, %s, %s, %s, %s, %s, %s, %s,
                %s, %s
            )
            ON CONFLICT (ticker, snapshot_time, time_range)
            DO UPDATE SET
                mention_count = EXCLUDED.mention_count,
                total_upvotes = EXCLUDED.total_upvotes,
                total_comments = EXCLUDED.total_comments,
                unique_posts = EXCLUDED.unique_posts,
                bullish_count = EXCLUDED.bullish_count,
                neutral_count = EXCLUDED.neutral_count,
                bearish_count = EXCLUDED.bearish_count,
                bullish_pct = EXCLUDED.bullish_pct,
                neutral_pct = EXCLUDED.neutral_pct,
                bearish_pct = EXCLUDED.bearish_pct,
                sentiment_score = EXCLUDED.sentiment_score,
                sentiment_label = EXCLUDED.sentiment_label,
                mention_velocity_1h
                    = EXCLUDED.mention_velocity_1h,
                sentiment_velocity_24h
                    = EXCLUDED.sentiment_velocity_24h,
                composite_score = EXCLUDED.composite_score,
                subreddit_distribution
                    = EXCLUDED.subreddit_distribution,
                rank = EXCLUDED.rank,
                previous_rank = EXCLUDED.previous_rank,
                rank_change = EXCLUDED.rank_change
        """

        cursor = self.conn.cursor()
        try:
            for agg in aggregations:
                cursor.execute(
                    query,
                    (
                        agg["ticker"],
                        self.snapshot_time,
                        time_range,
                        agg["mention_count"],
                        agg["total_upvotes"],
                        agg["total_comments"],
                        agg["unique_posts"],
                        agg["bullish_count"],
                        agg["neutral_count"],
                        agg["bearish_count"],
                        round(agg["bullish_pct"], 6),
                        round(agg["neutral_pct"], 6),
                        round(agg["bearish_pct"], 6),
                        round(
                            float(agg["sentiment_score"]),
                            6,
                        ),
                        agg["sentiment_label"],
                        agg.get("mention_velocity_1h"),
                        agg.get("sentiment_velocity_24h"),
                        agg["composite_score"],
                        agg.get("subreddit_distribution"),
                        agg["rank"],
                        agg.get("previous_rank"),
                        agg.get("rank_change"),
                    ),
                )

            self.conn.commit()
            self.stats["snapshots_upserted"] += len(
                aggregations
            )
            logger.info(
                f"  Upserted {len(aggregations)} snapshots "
                f"for time_range={time_range}"
            )

        except Exception as e:
            logger.error(f"Failed to upsert snapshots: {e}")
            self.conn.rollback()
        finally:
            cursor.close()

    def _write_history_points(self, snapshots: List[dict]):
        """Insert time-series data points.

        Writes to ticker_sentiment_history from the
        HISTORY_TIME_RANGE aggregation. Only called once per
        run to avoid duplicate history entries across time
        ranges.

        Args:
            snapshots: List of snapshot dicts
        """
        if not snapshots:
            return

        query = """
            INSERT INTO ticker_sentiment_history
                (time, ticker, sentiment_score, bullish_pct,
                 bearish_pct, neutral_pct,
                 mention_count, composite_score)
            VALUES (%s, %s, %s, %s, %s, %s, %s, %s)
            ON CONFLICT (ticker, time) DO NOTHING
        """

        cursor = self.conn.cursor()
        try:
            inserted = 0

            for snap in snapshots:
                cursor.execute(
                    query,
                    (
                        self.snapshot_time,
                        snap["ticker"],
                        round(
                            float(snap["sentiment_score"]),
                            6,
                        ),
                        round(snap["bullish_pct"], 6),
                        round(snap["bearish_pct"], 6),
                        round(snap["neutral_pct"], 6),
                        snap["mention_count"],
                        snap["composite_score"],
                    ),
                )
                inserted += cursor.rowcount

            self.conn.commit()
            self.stats["history_points_inserted"] = inserted
            logger.info(
                f"Inserted {inserted} history points "
                f"(from {HISTORY_TIME_RANGE} aggregation)"
            )

        except Exception as e:
            logger.error(
                f"Failed to insert history points: {e}"
            )
            self.conn.rollback()
        finally:
            cursor.close()

    def _log_summary(self):
        """Log aggregation summary."""
        logger.info("=" * 50)
        logger.info("Aggregation Summary:")
        logger.info(
            f"  Time ranges processed: "
            f"{self.stats['time_ranges_processed']}"
        )
        logger.info(
            f"  Snapshots upserted:    "
            f"{self.stats['snapshots_upserted']}"
        )
        logger.info(
            f"  History points:        "
            f"{self.stats['history_points_inserted']}"
        )
        logger.info("=" * 50)
