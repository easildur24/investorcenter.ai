"""Ticker extraction from Reddit post content."""

import logging
import re
from typing import Dict, List, Set

from .models import TickerMention

logger = logging.getLogger(__name__)

# Common false positives - words that look like tickers but aren't
FALSE_POSITIVES = {
    # Single letters and common abbreviations
    "I", "A", "B", "C", "D", "E", "F", "G", "H", "J", "K", "L", "M", "N", "O", "P",
    "Q", "R", "S", "T", "U", "V", "W", "X", "Y", "Z",
    "IT", "AT", "ON", "IS", "BE", "DO", "GO", "SO", "TO", "UP",
    "OR", "IN", "IF", "AN", "AS", "BY", "OF", "NO", "AM", "PM", "OK",
    # Business terms
    "CEO", "CFO", "CTO", "COO", "DD", "IPO", "EPS", "PE", "ETF", "SEC", "FED",
    "GDP", "CPI", "ROI", "ROE", "PNL", "YOY", "QOQ", "MOM", "COM", "INC", "LLC",
    # Geographic
    "USA", "USD", "UK", "EU", "US", "CA", "NYC", "LA", "SF",
    # Reddit/Internet slang
    "IMO", "TBH", "LOL", "EDIT", "UPDATE", "TL", "DR", "TLDR", "AMA", "ELI5",
    "NSFW", "OP", "WSB", "TLDR", "LMAO", "ROFL", "BTW", "FYI", "IIRC", "TIL",
    # Common words that match ticker patterns
    "FOR", "THE", "AND", "ARE", "BUT", "NOT", "YOU", "ALL", "CAN", "HER",
    "WAS", "ONE", "OUR", "OUT", "DAY", "HAD", "HAS", "HIS", "HOW", "ITS",
    "MAY", "NEW", "NOW", "OLD", "SEE", "WAY", "WHO", "OIL", "ANY", "BOT",
    "GET", "LET", "RUN", "SET", "TOP", "LOW", "HIGH", "JUST", "WELL",
    "BEST", "NEXT", "LAST", "MOST", "MUCH", "VERY", "EVEN", "ONLY",
    "SOME", "THEN", "THAN", "WHEN", "WHAT", "THIS", "THAT", "WITH",
    "FROM", "BEEN", "HAVE", "WERE", "WILL", "YOUR", "THEM", "THEY",
    "BEEN", "CALL", "COME", "MADE", "FIND", "LONG", "DOWN", "SHOULD",
    "COULD", "WOULD", "THESE", "OTHER", "AFTER", "FIRST", "THINK",
    "ALSO", "BACK", "GOOD", "KNOW", "TAKE", "WANT", "YEAR", "WEEK",
    # More common words that are also tickers
    "ACT", "AGO", "AMID", "AMP", "AWAY", "BEAT", "BOLD", "BOND", "BOOM",
    "CASH", "COST", "CUT", "DEC", "DIP", "DON", "DUG", "EASY", "EAT", "EDIT",
    "ELSE", "EVER", "EYE", "FAST", "FEEL", "FIVE", "FLEX", "FUND", "GAME",
    "GAP", "GIFT", "GLAD", "GOLD", "GOOD", "GRAB", "GROW", "GT", "HALF",
    "HE", "HEAR", "HELP", "HERE", "HIDE", "HIT", "HOPE", "HUGE", "IDEA",
    "IR", "JOBS", "JUMP", "KEEP", "KEY", "KIND", "LAKE", "LAND", "LATE",
    "LEND", "LESS", "LIFE", "LIKE", "LINE", "LIVE", "LOOK", "LUCK", "MAIN",
    "MANY", "MIND", "MORE", "MOVE", "MUST", "NEED", "NICE", "NOTA", "ONCE",
    "OPEN", "OVER", "PACK", "PAID", "PEAK", "PLAY", "POST", "PUMP", "PURE",
    "PUSH", "PUTS", "RACE", "RATE", "READ", "REAL", "REST", "RIDE", "RISE",
    "ROAD", "ROCK", "ROOM", "SAFE", "SAME", "SAVE", "SEEM", "SELF", "SEND",
    "SHIP", "SHOW", "SIDE", "SIZE", "SLOW", "SOLO", "SOON", "SPOT", "STAY",
    "STEP", "STOP", "SURE", "TAKE", "TALK", "TELL", "TEST", "TIME", "TIPS",
    "TOLD", "TOOK", "TOWN", "TRIP", "TRUE", "TURN", "TYPE", "USED", "VERY",
    "VIEW", "WAIT", "WALK", "WANT", "WARM", "WEEK", "WENT", "WISE", "WISH",
    "WORD", "WORK", "WRAP", "YEAR", "IPOS", "OKAY", "ENDS", "FACT", "FREE",
    "HITS", "HUGE", "ZERO", "NEAR", "PAST", "POOR", "RELY", "WEAR", "ZERO",
    "MATH", "MID", "NOTE", "ODDS", "PATH", "PAY", "NAIL", "LINK", "OWN", "OWNS",
    "BIG", "GOT", "SAY", "WIN", "LOSE", "GAVE", "PUTS", "MEN", "BET", "PICK",
    # Additional false positives from real data analysis
    "TECH", "SH", "TWO", "WWW", "USE", "WAR", "TAX", "IRON", "PRE", "EDGE",
    "TASK", "HARD", "MEME", "MOAT", "LAB", "INFO", "UNIT", "DOW", "RENT", "TUG",
    "AREA", "AREN", "BASE", "CAR", "CARS", "COLD", "COMP", "CUZ", "DE", "EARN",
    "ET", "FOUR", "FUN", "AD", "APPS", "AR", "BETA", "BOX", "AKA", "ASIA",
    "EVER", "PLUS", "GOES", "GAVE", "GIVES", "GOT", "HITS", "HURT", "LEAD",
    "LOWS", "OKAY", "OUTS", "OWNS", "RELY", "SEES", "SIDE", "WAYS", "ZERO",
    # Trading terms (not tickers)
    "BUY", "SELL", "HOLD", "CALL", "PUT", "YOLO", "MOON", "BEAR", "BULL",
    "GAIN", "LOSS", "LONG", "SHORT", "LEAP", "LEAPS", "ITM", "OTM", "ATM",
    # Sentiment/WSB terms that overlap with our lexicon
    "ATH", "ATL", "FUD", "FOMO", "HODL", "DCA", "LFG",
    # Time-related
    "EST", "PST", "UTC", "EOD", "EOW", "EOM", "JAN", "FEB", "MAR", "APR",
    "JUN", "JUL", "AUG", "SEP", "OCT", "NOV", "DEC", "MON", "TUE", "WED",
    "THU", "FRI", "SAT", "SUN",
    # Other common false positives
    "AI", "API", "AWS", "GCP", "SQL", "PHP", "CSS", "HTML", "JSON", "XML",
    "RH", "TD", "WK", "YTD", "QE", "SPAC", "NFT", "DAO", "DEX", "CEX",
    "IV", "DTE", "OI", "SI", "PT", "TP", "SL", "BE", "PDF", "APP", "ADS",
}


class TickerExtractor:
    """Extracts and validates stock ticker symbols from text."""

    # Pattern for explicit $TICKER mentions (highest confidence)
    DOLLAR_PATTERN = re.compile(r"\$([A-Z]{1,5})\b")

    # Pattern for standalone uppercase words (1-5 letters)
    STANDALONE_PATTERN = re.compile(r"\b([A-Z]{1,5})\b")

    def __init__(self, valid_tickers: Set[str] = None):
        """Initialize extractor.

        Args:
            valid_tickers: Set of valid ticker symbols. If None, all matches
                           that pass false positive filter are accepted.
        """
        self.valid_tickers = valid_tickers or set()
        self.false_positives = FALSE_POSITIVES.copy()

        logger.info(f"TickerExtractor initialized with {len(self.valid_tickers)} valid tickers")

    @classmethod
    def from_database(cls, db_connection) -> "TickerExtractor":
        """Create extractor with valid tickers loaded from database.

        Args:
            db_connection: Database connection object

        Returns:
            TickerExtractor instance
        """
        valid_tickers = set()

        try:
            cursor = db_connection.cursor()
            cursor.execute("""
                SELECT symbol FROM tickers
                WHERE asset_type IN ('stock', 'etf', 'CS', 'ETF')
                  AND symbol ~ '^[A-Z]{1,5}$'
            """)

            for row in cursor.fetchall():
                valid_tickers.add(row[0].upper())

            cursor.close()
            logger.info(f"Loaded {len(valid_tickers)} valid tickers from database")

        except Exception as e:
            logger.error(f"Failed to load tickers from database: {e}")

        return cls(valid_tickers)

    def extract(self, title: str, body: str = "") -> List[TickerMention]:
        """Extract ticker mentions from post title and body.

        Args:
            title: Post title
            body: Post body/selftext

        Returns:
            List of TickerMention objects, sorted by relevance
        """
        mentions: Dict[str, TickerMention] = {}

        # Extract from title (higher weight)
        self._extract_from_text(title, mentions, is_title=True)

        # Extract from body
        self._extract_from_text(body, mentions, is_title=False)

        # Filter and convert to list
        result = []
        for mention in mentions.values():
            # Include if:
            # 1. Valid ticker in our database, OR
            # 2. Mentioned with $ prefix (explicit), OR
            # 3. Mentioned multiple times (likely intentional)
            if (
                mention.ticker in self.valid_tickers
                or mention.count >= 2  # $TICKER counts as 2
                or mention.count >= 3
            ):
                result.append(mention)

        # Sort by: in_title > count > alphabetical
        result.sort(key=lambda m: (-m.in_title, -m.count, m.ticker))

        return result

    def _extract_from_text(
        self,
        text: str,
        mentions: Dict[str, TickerMention],
        is_title: bool,
    ):
        """Extract tickers from text and update mentions dict.

        Args:
            text: Text to extract from
            mentions: Dict to update with findings
            is_title: Whether this is the post title
        """
        if not text:
            return

        # First: Find $TICKER patterns (high confidence, always include)
        for match in self.DOLLAR_PATTERN.finditer(text):
            ticker = match.group(1).upper()

            if ticker in self.false_positives:
                continue

            if ticker in mentions:
                mentions[ticker].count += 2  # $TICKER counts double
                if is_title:
                    mentions[ticker].in_title = True
                mentions[ticker].positions.append(match.start())
            else:
                mentions[ticker] = TickerMention(
                    ticker=ticker,
                    count=2,  # $TICKER starts with count 2
                    in_title=is_title,
                    positions=[match.start()],
                )

        # Second: Find standalone TICKER patterns
        # Convert to uppercase for matching
        upper_text = text.upper()

        for match in self.STANDALONE_PATTERN.finditer(upper_text):
            ticker = match.group(1)

            # Skip false positives
            if ticker in self.false_positives:
                continue

            # For standalone mentions, only accept if in valid tickers list
            if self.valid_tickers and ticker not in self.valid_tickers:
                continue

            # Skip if we already found this via $TICKER
            if ticker in mentions and mentions[ticker].count >= 2:
                mentions[ticker].count += 1
                if is_title:
                    mentions[ticker].in_title = True
                mentions[ticker].positions.append(match.start())
            elif ticker not in mentions:
                mentions[ticker] = TickerMention(
                    ticker=ticker,
                    count=1,
                    in_title=is_title,
                    positions=[match.start()],
                )
            else:
                mentions[ticker].count += 1
                if is_title:
                    mentions[ticker].in_title = True
                mentions[ticker].positions.append(match.start())

    def get_primary_ticker(self, mentions: List[TickerMention]) -> str:
        """Determine the main ticker being discussed in a post.

        Priority: in_title > highest count > earliest position

        Args:
            mentions: List of ticker mentions

        Returns:
            Primary ticker symbol, or empty string if none
        """
        if not mentions:
            return ""

        if len(mentions) == 1:
            return mentions[0].ticker

        # Find best match
        best = mentions[0]

        for mention in mentions[1:]:
            # Strong preference for tickers in title
            if mention.in_title and not best.in_title:
                best = mention
                continue
            if not mention.in_title and best.in_title:
                continue

            # Then by mention count
            if mention.count > best.count:
                best = mention
                continue
            if mention.count < best.count:
                continue

            # Tiebreaker: earliest position
            if mention.positions and best.positions:
                if mention.positions[0] < best.positions[0]:
                    best = mention

        return best.ticker

    def is_valid_ticker(self, symbol: str) -> bool:
        """Check if a symbol is a known valid ticker.

        Args:
            symbol: Ticker symbol to check

        Returns:
            True if valid ticker
        """
        return symbol.upper() in self.valid_tickers

    def add_false_positive(self, term: str):
        """Add a term to the false positives list.

        Args:
            term: Term to add
        """
        self.false_positives.add(term.upper())

    def remove_false_positive(self, term: str):
        """Remove a term from the false positives list.

        Args:
            term: Term to remove
        """
        self.false_positives.discard(term.upper())

    def get_ticker_count(self) -> int:
        """Get number of valid tickers loaded.

        Returns:
            Count of valid tickers
        """
        return len(self.valid_tickers)
