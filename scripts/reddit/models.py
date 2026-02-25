"""Data models for Reddit sentiment collector."""

from dataclasses import dataclass, field
from datetime import datetime
from typing import List, Optional


@dataclass
class RedditPost:
    """Represents a Reddit post."""

    id: str  # Reddit's post ID (e.g., "abc123")
    title: str
    body: str  # selftext
    author: str
    subreddit: str
    score: int  # upvotes - downvotes
    num_comments: int
    created_utc: datetime
    permalink: str
    url: str
    flair: Optional[str] = None
    is_self: bool = True
    over_18: bool = False
    stickied: bool = False

    @property
    def full_url(self) -> str:
        """Get full Reddit URL."""
        return f"https://reddit.com{self.permalink}"

    @classmethod
    def from_json(cls, data: dict) -> "RedditPost":
        """Create RedditPost from Reddit API JSON response."""
        return cls(
            id=data.get("id", ""),
            title=data.get("title", ""),
            body=data.get("selftext", ""),
            author=data.get("author", "[deleted]"),
            subreddit=data.get("subreddit", ""),
            score=data.get("score", 0),
            num_comments=data.get("num_comments", 0),
            created_utc=datetime.utcfromtimestamp(data.get("created_utc", 0)),
            permalink=data.get("permalink", ""),
            url=data.get("url", ""),
            flair=data.get("link_flair_text"),
            is_self=data.get("is_self", True),
            over_18=data.get("over_18", False),
            stickied=data.get("stickied", False),
        )


@dataclass
class TickerMention:
    """Represents a ticker symbol found in text."""

    ticker: str
    count: int = 1
    in_title: bool = False
    positions: List[int] = field(default_factory=list)

    def __hash__(self):
        return hash(self.ticker)

    def __eq__(self, other):
        if isinstance(other, TickerMention):
            return self.ticker == other.ticker
        return False


@dataclass
class LexiconEntry:
    """Represents a term from the sentiment lexicon."""

    term: str
    sentiment: str  # "bullish", "bearish", "modifier"
    weight: float
    category: Optional[str] = None  # "slang", "emoji", "negation", etc.


@dataclass
class MatchedTerm:
    """Represents a term found during sentiment analysis."""

    term: str
    sentiment: str
    weight: float
    position: int


@dataclass
class SentimentResult:
    """Result of sentiment analysis."""

    sentiment: str  # "bullish", "bearish", "neutral"
    score: float  # -1.0 to 1.0
    confidence: float  # 0.0 to 1.0
    matched_terms: List[MatchedTerm] = field(default_factory=list)

    @property
    def is_bullish(self) -> bool:
        return self.sentiment == "bullish"

    @property
    def is_bearish(self) -> bool:
        return self.sentiment == "bearish"
