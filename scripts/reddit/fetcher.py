"""Reddit data fetcher using Arctic Shift API (works from cloud environments)."""

import logging
import time
from datetime import datetime, timedelta
from typing import List, Optional

import requests

from .models import RedditPost

logger = logging.getLogger(__name__)


class RedditFetcher:
    """Fetches posts using Arctic Shift API (third-party Reddit data service).

    Arctic Shift provides Reddit data without IP blocking, unlike Reddit's
    direct API which blocks cloud provider IPs.

    Note: Arctic Shift archives posts at creation time, so score data reflects
    the score at that moment (typically low for new posts). For sentiment
    analysis, we analyze all recent posts rather than filtering by score.

    See: https://github.com/ArthurHeitmann/arctic_shift
    """

    # Arctic Shift API endpoint (primary)
    BASE_URL = "https://arctic-shift.photon-reddit.com/api/posts/search"
    # Fallback to Reddit direct API (works from non-cloud IPs)
    REDDIT_URL = "https://old.reddit.com/r/{subreddit}/{sort}.json"

    USER_AGENT = "InvestorCenter/1.0 (Stock Sentiment Analysis)"

    # Rate limiting: Arctic Shift is more generous but we still respect limits
    RATE_LIMIT_DELAY = 2.0  # seconds between requests
    MAX_RETRIES = 3
    RETRY_DELAY = 5  # seconds

    def __init__(self, rate_limit_delay: float = None):
        """Initialize fetcher.

        Args:
            rate_limit_delay: Seconds between requests (default: 2.0)
        """
        self.rate_limit_delay = rate_limit_delay or self.RATE_LIMIT_DELAY
        self.last_request_time = 0.0
        self.session = requests.Session()
        self.session.headers.update({
            "User-Agent": self.USER_AGENT,
            "Accept": "application/json",
        })

    def _wait_for_rate_limit(self):
        """Wait if needed to respect rate limits."""
        elapsed = time.time() - self.last_request_time
        if elapsed < self.rate_limit_delay:
            sleep_time = self.rate_limit_delay - elapsed
            logger.debug(f"Rate limiting: sleeping {sleep_time:.1f}s")
            time.sleep(sleep_time)
        self.last_request_time = time.time()

    def _fetch_json(self, url: str, params: dict = None) -> Optional[dict]:
        """Fetch JSON from URL with retry logic.

        Args:
            url: URL to fetch
            params: Query parameters

        Returns:
            JSON response or None on failure
        """
        self._wait_for_rate_limit()

        for attempt in range(self.MAX_RETRIES):
            try:
                response = self.session.get(url, params=params, timeout=30)

                # Handle rate limiting
                if response.status_code == 429:
                    retry_after = int(response.headers.get("Retry-After", 60))
                    logger.warning(f"Rate limited (429), waiting {retry_after}s")
                    time.sleep(retry_after)
                    continue

                response.raise_for_status()
                return response.json()

            except requests.exceptions.Timeout:
                logger.warning(f"Request timeout (attempt {attempt + 1}/{self.MAX_RETRIES})")
            except requests.exceptions.RequestException as e:
                logger.warning(f"Request error: {e} (attempt {attempt + 1}/{self.MAX_RETRIES})")
            except ValueError as e:
                logger.warning(f"JSON decode error: {e}")
                return None

            if attempt < self.MAX_RETRIES - 1:
                time.sleep(self.RETRY_DELAY * (attempt + 1))

        logger.error(f"Failed to fetch {url} after {self.MAX_RETRIES} attempts")
        return None

    def fetch_subreddit(
        self,
        subreddit: str,
        sort: str = "hot",
        limit: int = 100,
        time_filter: str = "day",
        min_score: int = 5,
        max_age_days: int = 7,
    ) -> List[RedditPost]:
        """Fetch posts from a subreddit using Arctic Shift API.

        Args:
            subreddit: Subreddit name (without r/)
            sort: Sort order - "hot", "new", "top" (Arctic Shift sorts by created_utc)
            limit: Max posts to fetch
            time_filter: Time filter - "hour", "day", "week", "month" (for score filtering)
            min_score: Minimum score (upvotes) to include
            max_age_days: Maximum post age in days

        Returns:
            List of RedditPost objects
        """
        # Calculate time window
        now = datetime.utcnow()
        after_time = now - timedelta(days=max_age_days)

        params = {
            "subreddit": subreddit,
            "limit": min(limit, 100),  # Arctic Shift max is 100
            "after": int(after_time.timestamp()),
        }

        logger.info(f"Fetching r/{subreddit} via Arctic Shift (limit={limit})")

        data = self._fetch_json(self.BASE_URL, params)
        if not data:
            return []

        posts = self._parse_posts(data, min_score, max_age_days)

        # Sort by score (descending) to simulate "hot" sorting
        if sort in ("hot", "top"):
            posts.sort(key=lambda p: p.score, reverse=True)
        elif sort == "new":
            posts.sort(key=lambda p: p.created_utc, reverse=True)

        posts = posts[:limit]
        logger.info(f"  Fetched {len(posts)} posts from r/{subreddit}")

        return posts

    def fetch_subreddit_paginated(
        self,
        subreddit: str,
        sort: str = "hot",
        max_posts: int = 500,
        time_filter: str = "day",
        min_score: int = 5,
        max_age_days: int = 7,
    ) -> List[RedditPost]:
        """Fetch multiple pages of posts from a subreddit.

        Args:
            subreddit: Subreddit name
            sort: Sort order
            max_posts: Maximum total posts to fetch
            time_filter: Time filter for sorting
            min_score: Minimum score to include
            max_age_days: Maximum post age in days

        Returns:
            List of RedditPost objects
        """
        all_posts = []
        before_time = None
        pages_fetched = 0
        max_pages = 5  # Limit pagination to avoid excessive requests

        now = datetime.utcnow()
        after_time = now - timedelta(days=max_age_days)

        while len(all_posts) < max_posts and pages_fetched < max_pages:
            params = {
                "subreddit": subreddit,
                "limit": 100,
                "after": int(after_time.timestamp()),
            }

            if before_time:
                params["before"] = before_time

            data = self._fetch_json(self.BASE_URL, params)
            if not data:
                break

            posts = self._parse_posts(data, min_score, max_age_days)
            if not posts:
                break

            all_posts.extend(posts)
            pages_fetched += 1

            # Get oldest post timestamp for pagination
            if posts:
                oldest_post = min(posts, key=lambda p: p.created_utc)
                before_time = oldest_post.created_utc
            else:
                break

            logger.debug(f"  Page {pages_fetched}: {len(posts)} posts (total: {len(all_posts)})")

        # Sort and limit
        if sort in ("hot", "top"):
            all_posts.sort(key=lambda p: p.score, reverse=True)
        elif sort == "new":
            all_posts.sort(key=lambda p: p.created_utc, reverse=True)

        logger.info(f"Fetched {len(all_posts)} total posts from r/{subreddit}")
        return all_posts[:max_posts]

    def _parse_posts(
        self, data: dict, min_score: int, max_age_days: int
    ) -> List[RedditPost]:
        """Parse Arctic Shift API response into RedditPost objects.

        Args:
            data: Arctic Shift API JSON response
            min_score: Minimum score to include
            max_age_days: Maximum post age in days

        Returns:
            List of filtered RedditPost objects
        """
        posts = []
        cutoff_time = datetime.utcnow() - timedelta(days=max_age_days)

        # Arctic Shift returns {"data": [post1, post2, ...]}
        post_list = data.get("data", [])

        for post_data in post_list:
            # Skip stickied posts (megathreads, rules, etc.)
            if post_data.get("stickied", False):
                continue

            # Skip NSFW posts
            if post_data.get("over_18", False):
                continue

            # Skip low-engagement posts
            score = post_data.get("score", post_data.get("ups", 0))
            if score < min_score:
                continue

            # Skip old posts
            created_utc = post_data.get("created_utc", 0)
            post_time = datetime.utcfromtimestamp(created_utc)
            if post_time < cutoff_time:
                continue

            # Skip removed/deleted posts
            selftext = post_data.get("selftext", "")
            if selftext in ("[removed]", "[deleted]"):
                selftext = ""

            try:
                post = RedditPost.from_json(post_data)
                posts.append(post)
            except Exception as e:
                logger.warning(f"Failed to parse post: {e}")
                continue

        return posts

    def fetch_multiple_subreddits(
        self,
        subreddits: List[str],
        sort: str = "hot",
        limit_per_subreddit: int = 100,
        **kwargs,
    ) -> List[RedditPost]:
        """Fetch posts from multiple subreddits.

        Args:
            subreddits: List of subreddit names
            sort: Sort order
            limit_per_subreddit: Max posts per subreddit
            **kwargs: Additional arguments for fetch_subreddit

        Returns:
            Combined list of posts from all subreddits
        """
        all_posts = []

        for subreddit in subreddits:
            try:
                posts = self.fetch_subreddit(
                    subreddit, sort=sort, limit=limit_per_subreddit, **kwargs
                )
                all_posts.extend(posts)
            except Exception as e:
                logger.error(f"Failed to fetch r/{subreddit}: {e}")
                continue

        return all_posts


class RSSFetcher:
    """Fetches posts from Reddit RSS feeds (lighter weight, for quick updates).

    Note: RSS feeds may also be blocked from cloud IPs. Use RedditFetcher
    with Arctic Shift for reliable cloud access.
    """

    RSS_URL = "https://www.reddit.com/r/{subreddit}/new/.rss"
    USER_AGENT = "InvestorCenter/1.0 (Stock Sentiment Analysis)"

    def __init__(self):
        self.session = requests.Session()
        self.session.headers.update({"User-Agent": self.USER_AGENT})

    def fetch_new_posts(self, subreddit: str, limit: int = 25) -> List[dict]:
        """Fetch new posts from RSS feed.

        Note: RSS provides limited data (title, link, time) compared to JSON.

        Args:
            subreddit: Subreddit name
            limit: Max posts (RSS max is 25)

        Returns:
            List of basic post data dicts
        """
        url = self.RSS_URL.format(subreddit=subreddit)

        try:
            response = self.session.get(url, timeout=10)
            response.raise_for_status()

            # Parse RSS (simple extraction without feedparser dependency)
            # For production, consider using feedparser library
            return self._parse_rss(response.text, limit)

        except Exception as e:
            logger.error(f"RSS fetch failed for r/{subreddit}: {e}")
            return []

    def _parse_rss(self, content: str, limit: int) -> List[dict]:
        """Parse RSS content (basic implementation).

        For production use, consider using feedparser library.
        """
        import re

        posts = []

        # Extract entry blocks
        entries = re.findall(r"<entry>(.*?)</entry>", content, re.DOTALL)

        for entry in entries[:limit]:
            try:
                # Extract title
                title_match = re.search(r"<title>(.+?)</title>", entry)
                title = title_match.group(1) if title_match else ""

                # Extract link
                link_match = re.search(r'<link href="([^"]+)"', entry)
                link = link_match.group(1) if link_match else ""

                # Extract ID from link
                id_match = re.search(r"/comments/([a-z0-9]+)/", link)
                post_id = id_match.group(1) if id_match else ""

                if title and post_id:
                    posts.append(
                        {
                            "id": post_id,
                            "title": title,
                            "url": link,
                        }
                    )
            except Exception:
                continue

        return posts
