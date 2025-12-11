"""Lexicon-based sentiment analyzer for Reddit posts."""

import logging
import re
from typing import Dict, List

from .models import LexiconEntry, MatchedTerm, SentimentResult

logger = logging.getLogger(__name__)

# TODD: Improve the analyzer to use ML to analyze sentiment.
class SentimentAnalyzer:
    """Performs sentiment analysis using a lexicon of terms."""

    def __init__(self, lexicon: Dict[str, LexiconEntry] = None):
        """Initialize analyzer.

        Args:
            lexicon: Dict mapping lowercase terms to LexiconEntry objects
        """
        self.lexicon = lexicon or {}
        logger.info(f"SentimentAnalyzer initialized with {len(self.lexicon)} terms")

    @classmethod
    def from_database(cls, db_connection) -> "SentimentAnalyzer":
        """Create analyzer with lexicon loaded from database.

        Args:
            db_connection: Database connection object

        Returns:
            SentimentAnalyzer instance
        """
        lexicon = {}

        try:
            cursor = db_connection.cursor()
            cursor.execute("""
                SELECT term, sentiment, weight, COALESCE(category, '') as category
                FROM sentiment_lexicon
            """)

            for row in cursor.fetchall():
                term, sentiment, weight, category = row
                entry = LexiconEntry(
                    term=term,
                    sentiment=sentiment,
                    weight=float(weight),
                    category=category if category else None,
                )
                lexicon[term.lower()] = entry

            cursor.close()
            logger.info(f"Loaded {len(lexicon)} terms from sentiment_lexicon")

        except Exception as e:
            logger.error(f"Failed to load lexicon from database: {e}")

        return cls(lexicon)

    def analyze(self, title: str, body: str = "") -> SentimentResult:
        """Analyze sentiment of post title and body.

        Args:
            title: Post title
            body: Post body/selftext

        Returns:
            SentimentResult with sentiment, score, confidence, and matched terms
        """
        # Combine title (weighted higher by repeating) and body
        text = f"{title} {title} {body}".lower()

        # Clean text
        text = self._clean_text(text)

        # Tokenize
        words = text.split()

        matches: List[MatchedTerm] = []
        bullish_score = 0.0
        bearish_score = 0.0
        total_weight = 0.0

        # Modifier state tracking
        negation_active = False
        negation_countdown = 0
        amplifier_active = False
        amplifier_multiplier = 1.0
        amplifier_countdown = 0

        skip_until = 0  # For multi-word phrase handling

        i = 0
        while i < len(words):
            if i < skip_until:
                i += 1
                continue

            # Check for multi-word phrases first (up to 4 words)
            matched = False

            for phrase_len in range(4, 0, -1):
                if i + phrase_len > len(words):
                    continue

                phrase = " ".join(words[i : i + phrase_len])
                entry = self.lexicon.get(phrase)

                if not entry:
                    continue

                matched = True

                if entry.sentiment == "modifier":
                    # Handle negation (negative weight)
                    if entry.weight < 0:
                        negation_active = True
                        negation_countdown = 3  # Affects next 3 sentiment words
                    elif entry.category == "amplifier":
                        # Handle amplifiers (very, extremely, etc.)
                        amplifier_active = True
                        amplifier_multiplier = entry.weight
                        amplifier_countdown = 2
                    elif entry.category == "reducer":
                        # Handle reducers (maybe, might, etc.)
                        amplifier_active = True
                        amplifier_multiplier = entry.weight  # e.g., 0.5
                        amplifier_countdown = 2
                else:
                    # Sentiment term found
                    weight = entry.weight

                    # Apply amplifier/reducer
                    if amplifier_active:
                        weight *= amplifier_multiplier

                    # Apply negation (flips sentiment)
                    if negation_active:
                        weight *= -1

                    matches.append(
                        MatchedTerm(
                            term=phrase,
                            sentiment=entry.sentiment,
                            weight=weight,
                            position=i,
                        )
                    )

                    # Accumulate scores based on original sentiment direction
                    if entry.sentiment == "bullish":
                        if weight > 0:
                            bullish_score += weight
                        else:
                            bearish_score += abs(weight)
                    elif entry.sentiment == "bearish":
                        if weight > 0:
                            bearish_score += weight
                        else:
                            bullish_score += abs(weight)

                    total_weight += abs(weight)

                skip_until = i + phrase_len
                break

            # Decrement modifier countdowns
            if not matched:
                if negation_active:
                    negation_countdown -= 1
                    if negation_countdown <= 0:
                        negation_active = False

                if amplifier_active:
                    amplifier_countdown -= 1
                    if amplifier_countdown <= 0:
                        amplifier_active = False
                        amplifier_multiplier = 1.0

            i += 1

        # Calculate final score and sentiment
        score = 0.0
        sentiment = "neutral"
        confidence = 0.0

        if total_weight > 0:
            score = (bullish_score - bearish_score) / total_weight
            # Confidence based on signal density
            confidence = min(total_weight / len(words) * 10, 1.0) if words else 0.0

        # Determine sentiment label
        if score > 0.1:
            sentiment = "bullish"
        elif score < -0.1:
            sentiment = "bearish"
        else:
            sentiment = "neutral"

        # Boost confidence if score is extreme
        if abs(score) > 0.5:
            confidence = min(confidence * 1.2, 1.0)

        # Minimum confidence threshold
        if len(matches) == 0:
            confidence = 0.0
        elif len(matches) == 1:
            confidence = min(confidence, 0.5)

        return SentimentResult(
            sentiment=sentiment,
            score=round(score, 4),
            confidence=round(confidence, 4),
            matched_terms=matches,
        )

    def _clean_text(self, text: str) -> str:
        """Clean and normalize text for analysis.

        Args:
            text: Raw text

        Returns:
            Cleaned text
        """
        # Remove URLs
        text = re.sub(r"https?://\S+", "", text)

        # Remove Reddit markdown links [text](url)
        text = re.sub(r"\[([^\]]+)\]\([^)]+\)", r"\1", text)

        # Keep emojis but remove other special chars
        # Allow letters, numbers, spaces, and unicode chars > 127 (emojis)
        cleaned = []
        for char in text:
            if char.isalnum() or char.isspace() or ord(char) > 127:
                cleaned.append(char)
            else:
                cleaned.append(" ")

        # Normalize whitespace
        return " ".join("".join(cleaned).split())

    def refresh_lexicon(self, db_connection):
        """Reload lexicon from database.

        Args:
            db_connection: Database connection object
        """
        new_analyzer = SentimentAnalyzer.from_database(db_connection)
        self.lexicon = new_analyzer.lexicon
        logger.info(f"Lexicon refreshed with {len(self.lexicon)} terms")

    def get_lexicon_size(self) -> int:
        """Get number of terms in lexicon.

        Returns:
            Term count
        """
        return len(self.lexicon)

    def get_lexicon_stats(self) -> Dict[str, int]:
        """Get statistics about the lexicon.

        Returns:
            Dict with counts by sentiment type
        """
        stats = {"bullish": 0, "bearish": 0, "modifier": 0, "total": len(self.lexicon)}

        for entry in self.lexicon.values():
            if entry.sentiment in stats:
                stats[entry.sentiment] += 1

        return stats
