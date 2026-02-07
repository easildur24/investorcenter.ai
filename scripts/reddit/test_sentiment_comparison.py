#!/usr/bin/env python3
"""Compare FinBERT vs Claude Haiku on Reddit sentiment analysis.

Tests both models against curated Reddit posts covering edge cases:
sarcasm, irony, WSB culture, multi-ticker, pump-and-dump, etc.

Usage:
    pip install anthropic transformers torch
    export ANTHROPIC_API_KEY=your_key
    python scripts/reddit/test_sentiment_comparison.py
"""

import json
import os
import sys
import time
from dataclasses import dataclass
from typing import List, Optional

# ──────────────────────────────────────────────────────────────────────
# Test Posts - curated to expose sentiment analysis weaknesses
# ──────────────────────────────────────────────────────────────────────

@dataclass
class TestPost:
    """A curated Reddit post with expected sentiment."""
    id: str
    subreddit: str
    title: str
    body: str
    upvotes: int
    expected_tickers: List[str]
    expected_sentiment: dict  # {ticker: "bullish"/"bearish"/"neutral"}
    category: str  # What this test case covers
    notes: str  # Why this is a hard case


TEST_POSTS = [
    # ── EASY CASES (both models should get right) ────────────────────
    TestPost(
        id="easy_bull_1",
        subreddit="stocks",
        title="NVDA earnings destroyed estimates, raising my price target to $200",
        body="Nvidia just reported incredible earnings. Revenue up 120% YoY, data center business is on fire. I'm loading up on calls expiring next month. This company is executing flawlessly.",
        upvotes=1250,
        expected_tickers=["NVDA"],
        expected_sentiment={"NVDA": "bullish"},
        category="straightforward_bullish",
        notes="Clear bullish language, easy for any model",
    ),
    TestPost(
        id="easy_bear_1",
        subreddit="stocks",
        title="Intel is done. Sold all my shares today",
        body="Losing market share every quarter to AMD and NVDA. Terrible management, fabs behind TSMC by 2 generations. The dividend isn't worth the capital destruction. I'm out.",
        upvotes=890,
        expected_tickers=["INTC"],
        expected_sentiment={"INTC": "bearish"},
        category="straightforward_bearish",
        notes="Clear bearish language, easy for any model",
    ),

    # ── SARCASM / IRONY (FinBERT will likely fail) ───────────────────
    TestPost(
        id="sarcasm_1",
        subreddit="wallstreetbets",
        title="Yeah sure, buy TSLA at all time highs, what could possibly go wrong 🚀🚀🚀",
        body="Just like last time when everyone said it was going to $500 and it dropped 40%. But hey, this time is different right? Elon will save us all. Can't wait to see the loss porn next week.",
        upvotes=3400,
        expected_tickers=["TSLA"],
        expected_sentiment={"TSLA": "bearish"},
        category="sarcasm",
        notes="Rocket emojis + 'buy' language but clearly mocking bulls. FinBERT will likely read surface-level positive words.",
    ),
    TestPost(
        id="sarcasm_2",
        subreddit="wallstreetbets",
        title="Great job holding GME bags from $400 to $20, real diamond hands there 💎🙌",
        body="Nothing says smart investing like watching your portfolio drop 95% and calling it conviction. But sure, the squeeze is still coming any day now. Trust the DD from 3 years ago.",
        upvotes=5200,
        expected_tickers=["GME"],
        expected_sentiment={"GME": "bearish"},
        category="sarcasm",
        notes="Uses bullish emoji (diamond hands) sarcastically. Lexicon would score this bullish.",
    ),
    TestPost(
        id="sarcasm_3",
        subreddit="wallstreetbets",
        title="Incredible strategy: buy high, sell low. My SOFI journey",
        body="Bought at $24, averaged down at $15, averaged down again at $8, sold everything at $5. Truly galaxy brain moves. Thanks for the financial education r/wallstreetbets.",
        upvotes=7800,
        expected_tickers=["SOFI"],
        expected_sentiment={"SOFI": "bearish"},
        category="sarcasm",
        notes="Self-deprecating humor. 'Incredible' and 'galaxy brain' are ironic.",
    ),

    # ── WSB CULTURE (context-dependent) ──────────────────────────────
    TestPost(
        id="wsb_culture_1",
        subreddit="wallstreetbets",
        title="Lost $50k on AAPL puts. GUH. Here's my loss porn 📉",
        body="Bought weekly puts before earnings, Tim Apple decided to beat estimates by a mile. Account down 80% in a day. At least I have content for you degenerates. Positions: AAPL 170p 01/19.",
        upvotes=12000,
        expected_tickers=["AAPL"],
        expected_sentiment={"AAPL": "neutral"},
        category="loss_porn",
        notes="The poster lost money being BEARISH on AAPL (puts). AAPL beat earnings = stock is doing well. This is neutral/bullish on AAPL, not bearish.",
    ),
    TestPost(
        id="wsb_culture_2",
        subreddit="wallstreetbets",
        title="My wife's boyfriend drives a Tesla and I can't afford rent. Calls on divorce lawyers 😂",
        body="Seriously though my TSLA calls printed this week. Up 300% on the position. Sometimes the universe gives back after taking everything.",
        upvotes=4500,
        expected_tickers=["TSLA"],
        expected_sentiment={"TSLA": "bullish"},
        category="wsb_humor",
        notes="Title is pure WSB humor/meme. Body reveals actual bullish position. Need to read both.",
    ),

    # ── MULTI-TICKER / MIXED SENTIMENT ───────────────────────────────
    TestPost(
        id="multi_1",
        subreddit="stocks",
        title="Selling all my INTC to buy NVDA. The AI trade is clear.",
        body="Intel is hemorrhaging market share and their foundry play is years away from paying off. Meanwhile Nvidia literally can't make enough H100s to meet demand. Easy swap.",
        upvotes=670,
        expected_tickers=["INTC", "NVDA"],
        expected_sentiment={"INTC": "bearish", "NVDA": "bullish"},
        category="multi_ticker_mixed",
        notes="Two tickers with opposite sentiment. Models need per-ticker analysis.",
    ),
    TestPost(
        id="multi_2",
        subreddit="investing",
        title="AMD vs NVDA - which is the better AI play?",
        body="Both are great companies but I think AMD is undervalued relative to NVDA. AMD's MI300 is gaining traction and at half the P/E ratio of Nvidia. I'm going 60% AMD, 40% NVDA.",
        upvotes=420,
        expected_tickers=["AMD", "NVDA"],
        expected_sentiment={"AMD": "bullish", "NVDA": "bullish"},
        category="multi_ticker_nuanced",
        notes="Both bullish but AMD more so. Subtle comparative analysis.",
    ),

    # ── SUBTLE / COMPLEX ─────────────────────────────────────────────
    TestPost(
        id="subtle_1",
        subreddit="stocks",
        title="The DD on PLTR is solid but the dilution concerns me",
        body="Palantir's government contracts are sticky and commercial growth is accelerating. But Karp keeps diluting shareholders with SBC. I'm holding my shares but not adding here. Waiting for a pullback to $15 before I increase my position.",
        upvotes=340,
        expected_tickers=["PLTR"],
        expected_sentiment={"PLTR": "neutral"},
        category="nuanced",
        notes="Mixed signals: positive on business, negative on dilution, holding but not buying more. Should be neutral.",
    ),
    TestPost(
        id="subtle_2",
        subreddit="stocks",
        title="Everyone says buy AMD but the chart looks terrible",
        body="I know the fundamentals are solid and MI300 is a game changer, but technically we just broke below the 200 DMA with increasing volume. I want to own this name but I'll wait for $120 support to hold before entering. Not bearish long term, just cautious short term.",
        upvotes=280,
        expected_tickers=["AMD"],
        expected_sentiment={"AMD": "neutral"},
        category="nuanced",
        notes="Says 'buy' and positive fundamentals but is actually cautious/waiting. Not bullish despite positive words.",
    ),

    # ── PUMP AND DUMP / SPAM ─────────────────────────────────────────
    TestPost(
        id="spam_1",
        subreddit="wallstreetbets",
        title="🚨🚨 $BBBY about to EXPLODE 🚨🚨 massive short squeeze incoming, get in NOW!!!",
        body="HUGE short interest at 45%!!! This is the next GME!!! 🚀🚀🚀🚀🚀 Don't miss out, once this squeezes it's going to $100+ easy. Load up on calls NOW before it's too late. Not financial advice but seriously BUY BUY BUY!!!",
        upvotes=150,
        expected_tickers=["BBBY"],
        expected_sentiment={"BBBY": "bullish"},
        category="pump_and_dump",
        notes="Technically bullish sentiment but extremely spammy. Should have high spam_score.",
    ),

    # ── NOT FINANCE RELATED ──────────────────────────────────────────
    TestPost(
        id="non_finance_1",
        subreddit="wallstreetbets",
        title="I work at McDonald's and my manager yells at me about burgers, not stonks",
        body="Day 47 of checking my portfolio during my shift. The fry oil is the only thing going up in my life. At least I get free nuggets. Puts on my career.",
        upvotes=9500,
        expected_tickers=[],
        expected_sentiment={},
        category="non_finance",
        notes="No real ticker discussion. Pure WSB lifestyle humor.",
    ),
    TestPost(
        id="non_finance_2",
        subreddit="wallstreetbets",
        title="Cathie Wood just doubled down on Tesla again",
        body="She keeps buying the dip on TSLA but ARK has been underperforming the S&P for 3 years straight. At some point you have to question the thesis. Is this conviction or stubbornness?",
        upvotes=2100,
        expected_tickers=["TSLA"],
        expected_sentiment={"TSLA": "bearish"},
        category="indirect_bearish",
        notes="Doesn't say 'TSLA is bad' directly but questions the bull thesis and highlights underperformance. Bearish undertone.",
    ),

    # ── EARNINGS REACTION ────────────────────────────────────────────
    TestPost(
        id="earnings_1",
        subreddit="stocks",
        title="META beat earnings but guided lower. Stock dropping after hours",
        body="Revenue beat by 3% and EPS crushed estimates, but forward guidance was below consensus citing increased AI infrastructure spending. Stock down 8% AH. I think this is an overreaction and I'm buying the dip tomorrow.",
        upvotes=1800,
        expected_tickers=["META"],
        expected_sentiment={"META": "bullish"},
        category="contrarian",
        notes="Stock is dropping but poster is bullish (buying the dip). Surface reading might say bearish because of 'dropping' and 'down 8%'.",
    ),
]


# ──────────────────────────────────────────────────────────────────────
# FinBERT Analyzer (adapted from news_sentiment_ingestion.py)
# ──────────────────────────────────────────────────────────────────────

class FinBERTAnalyzer:
    """FinBERT sentiment analyzer for financial text."""

    def __init__(self):
        self.tokenizer = None
        self.model = None
        self._initialized = False
        self.available = False

    def _initialize(self):
        if self._initialized:  # Only try once
            return
        self._initialized = True  # Mark as tried regardless of outcome
        try:
            from transformers import AutoTokenizer, AutoModelForSequenceClassification
            print("  Loading FinBERT model (first run downloads ~400MB)...")
            self.tokenizer = AutoTokenizer.from_pretrained("ProsusAI/finbert")
            self.model = AutoModelForSequenceClassification.from_pretrained("ProsusAI/finbert")
            self.model.eval()
            self.available = True
            print("  FinBERT ready.\n")
        except ImportError:
            print("  WARNING: transformers/torch not installed - FinBERT tests will be skipped")
            print("  Install: pip install transformers torch\n")
        except Exception as e:
            print(f"  WARNING: FinBERT model failed to load: {e}")
            print("  FinBERT tests will be skipped.\n")

    def analyze(self, title: str, body: str) -> dict:
        """Analyze sentiment. Returns dict with score, label, confidence."""
        if not self._initialized:
            self._initialize()
        if not self.available:
            return {"skipped": True}
        import torch

        text = f"{title} {body[:400]}"  # FinBERT max 512 tokens

        inputs = self.tokenizer(
            text, return_tensors="pt", truncation=True, max_length=512, padding=True
        )
        with torch.no_grad():
            outputs = self.model(**inputs)
            probs = torch.nn.functional.softmax(outputs.logits, dim=-1)

        labels = ["positive", "negative", "neutral"]
        pred_idx = probs.argmax().item()
        confidence = probs[0][pred_idx].item()

        # Map FinBERT labels to our labels
        label_map = {"positive": "bullish", "negative": "bearish", "neutral": "neutral"}

        return {
            "label": label_map[labels[pred_idx]],
            "confidence": round(confidence, 3),
            "raw_probs": {
                "bullish": round(probs[0][0].item(), 3),
                "bearish": round(probs[0][1].item(), 3),
                "neutral": round(probs[0][2].item(), 3),
            },
        }


# ──────────────────────────────────────────────────────────────────────
# Claude Haiku Analyzer
# ──────────────────────────────────────────────────────────────────────

HAIKU_PROMPT = """You are a financial sentiment analyst specializing in Reddit investing communities (r/wallstreetbets, r/stocks, r/investing, etc.).

Analyze this Reddit post and return a JSON object. Consider:
- Sarcasm and irony are extremely common on WSB. "What could go wrong" = bearish. Rocket emojis used mockingly = bearish.
- "Loss porn" is sharing losses for community bonding — it doesn't mean the poster is bearish on the stock.
- Diamond hands/holding used sarcastically (mocking bag holders) = bearish.
- Distinguish between a poster's VIEW on the stock vs what HAPPENED to the stock.
- A post saying "stock is dropping but I'm buying" = bullish.
- Pump-and-dump language: excessive emojis, urgency ("NOW", "before it's too late"), unrealistic targets.

Post from r/{subreddit}:
Title: {title}
Body: {body}

Return ONLY this JSON, no other text:
{{
  "is_finance_related": true/false,
  "tickers": [
    {{
      "symbol": "TICKER",
      "sentiment": "bullish" | "bearish" | "neutral",
      "confidence": 0.0-1.0,
      "reasoning": "brief explanation"
    }}
  ],
  "spam_score": 0.0-1.0
}}"""


class HaikuAnalyzer:
    """Claude Haiku sentiment analyzer."""

    def __init__(self):
        self.available = False
        try:
            import anthropic
            api_key = os.environ.get("ANTHROPIC_API_KEY")
            if not api_key:
                print("  WARNING: ANTHROPIC_API_KEY not set - Haiku tests will be skipped")
                print("  Set it to enable: export ANTHROPIC_API_KEY=your_key\n")
                return
            self.client = anthropic.Anthropic(api_key=api_key)
            self.available = True
            print("  Claude Haiku ready.\n")
        except ImportError:
            print("  WARNING: anthropic SDK not installed - Haiku tests will be skipped")
            print("  Install it: pip install anthropic\n")

    def analyze(self, post: TestPost) -> dict:
        """Analyze a single post with Claude Haiku."""
        if not self.available:
            return {"skipped": True}

        prompt = HAIKU_PROMPT.format(
            subreddit=post.subreddit,
            title=post.title,
            body=post.body,
        )

        try:
            response = self.client.messages.create(
                model="claude-haiku-4-5-20251001",
                max_tokens=500,
                temperature=0.0,
                messages=[{"role": "user", "content": prompt}],
            )

            text = response.text.strip()
            # Extract JSON if wrapped in markdown code block
            if "```" in text:
                import re
                json_match = re.search(r'```(?:json)?\s*([\s\S]*?)```', text)
                if json_match:
                    text = json_match.group(1).strip()

            result = json.loads(text)

            # Calculate token usage
            input_tokens = response.usage.input_tokens
            output_tokens = response.usage.output_tokens

            return {
                "result": result,
                "input_tokens": input_tokens,
                "output_tokens": output_tokens,
            }

        except json.JSONDecodeError as e:
            return {"error": f"JSON parse error: {e}", "raw": text}
        except Exception as e:
            return {"error": str(e)}


# ──────────────────────────────────────────────────────────────────────
# Comparison Runner
# ──────────────────────────────────────────────────────────────────────

def sentiment_match(expected: str, actual: str) -> bool:
    """Check if sentiment labels match."""
    return expected.lower() == actual.lower()


def print_header(text: str):
    print(f"\n{'='*80}")
    print(f"  {text}")
    print(f"{'='*80}")


def print_result_row(label: str, value: str, match: Optional[bool] = None):
    icon = ""
    if match is True:
        icon = " ✅"
    elif match is False:
        icon = " ❌"
    print(f"  {label:<20} {value}{icon}")


def run_comparison():
    """Run FinBERT vs Haiku on all test posts."""
    print_header("SENTIMENT ANALYSIS COMPARISON: FinBERT vs Claude Haiku")
    print(f"  Test posts: {len(TEST_POSTS)}")
    print(f"  Categories: {len(set(p.category for p in TEST_POSTS))}")

    # Initialize models
    print("\nInitializing models...")
    finbert = FinBERTAnalyzer()
    haiku = HaikuAnalyzer()

    # Track scores
    finbert_correct = 0
    haiku_correct = 0
    total_evaluated = 0
    total_input_tokens = 0
    total_output_tokens = 0

    results = []

    for i, post in enumerate(TEST_POSTS):
        print_header(f"[{i+1}/{len(TEST_POSTS)}] {post.category.upper()}: {post.id}")
        print(f"  Title: {post.title[:75]}{'...' if len(post.title) > 75 else ''}")
        print(f"  Body:  {post.body[:75]}{'...' if len(post.body) > 75 else ''}")
        print(f"  Expected: tickers={post.expected_tickers}, sentiment={post.expected_sentiment}")
        print(f"  Notes: {post.notes}")

        # ── FinBERT ──
        print(f"\n  --- FinBERT ---")
        finbert_result = finbert.analyze(post.title, post.body)
        finbert_label = None
        finbert_match = None

        if finbert_result.get("skipped"):
            print_result_row("Status:", "SKIPPED (model unavailable)")
        else:
            finbert_label = finbert_result["label"]
            finbert_conf = finbert_result["confidence"]
            finbert_probs = finbert_result["raw_probs"]

            # FinBERT gives one overall sentiment (no per-ticker)
            # Compare against primary ticker's expected sentiment
            if post.expected_tickers:
                primary_ticker = post.expected_tickers[0]
                expected = post.expected_sentiment.get(primary_ticker, "neutral")
                finbert_match = sentiment_match(expected, finbert_label)
                if finbert_match:
                    finbert_correct += 1

            print_result_row("Sentiment:", f"{finbert_label} (conf: {finbert_conf:.3f})", finbert_match)
            print_result_row("Probabilities:", f"bull={finbert_probs['bullish']:.3f}  bear={finbert_probs['bearish']:.3f}  neut={finbert_probs['neutral']:.3f}")

        # ── Claude Haiku ──
        print(f"\n  --- Claude Haiku ---")
        haiku_result = haiku.analyze(post)

        haiku_match = None
        if haiku_result.get("skipped"):
            print_result_row("Status:", "SKIPPED (no API key)")
        elif "error" in haiku_result:
            print_result_row("Error:", haiku_result["error"])
        else:
            result = haiku_result["result"]
            tickers = result.get("tickers", [])
            is_finance = result.get("is_finance_related", False)
            spam_score = result.get("spam_score", 0.0)
            input_tok = haiku_result["input_tokens"]
            output_tok = haiku_result["output_tokens"]
            total_input_tokens += input_tok
            total_output_tokens += output_tok

            print_result_row("Finance related:", str(is_finance))
            print_result_row("Spam score:", f"{spam_score:.2f}")
            print_result_row("Tokens:", f"in={input_tok}, out={output_tok}")

            # Check each ticker
            haiku_all_correct = True
            if not post.expected_tickers and not tickers:
                haiku_all_correct = True  # Correctly identified no tickers
            elif not post.expected_tickers and tickers:
                haiku_all_correct = False
            else:
                haiku_ticker_map = {t["symbol"]: t for t in tickers}
                for exp_ticker, exp_sentiment in post.expected_sentiment.items():
                    if exp_ticker in haiku_ticker_map:
                        t = haiku_ticker_map[exp_ticker]
                        match = sentiment_match(exp_sentiment, t["sentiment"])
                        if not match:
                            haiku_all_correct = False
                        print_result_row(
                            f"  {t['symbol']}:",
                            f"{t['sentiment']} (conf: {t['confidence']:.2f}) - {t.get('reasoning', 'n/a')}",
                            match,
                        )
                    else:
                        haiku_all_correct = False
                        print_result_row(f"  {exp_ticker}:", "NOT FOUND", False)

            haiku_match = haiku_all_correct
            if haiku_match:
                haiku_correct += 1

        if post.expected_tickers:
            total_evaluated += 1

        results.append({
            "post_id": post.id,
            "category": post.category,
            "finbert_label": finbert_label or "skipped",
            "finbert_correct": finbert_match,
            "haiku_correct": haiku_match,
        })

        # Small delay between Haiku calls
        time.sleep(0.5)

    # ── Summary ──────────────────────────────────────────────────────
    print_header("RESULTS SUMMARY")

    finbert_evaluated = sum(1 for r in results if r["finbert_correct"] is not None)
    haiku_evaluated = sum(1 for r in results if r["haiku_correct"] is not None)

    print(f"\n  Posts evaluated: {total_evaluated} (excluding non-finance posts)")
    print(f"\n  {'Model':<20} {'Correct':<10} {'Accuracy':<10}")
    print(f"  {'-'*40}")
    if finbert_evaluated > 0:
        print(f"  {'FinBERT':<20} {finbert_correct}/{finbert_evaluated:<9} {finbert_correct/max(finbert_evaluated,1)*100:.0f}%")
    else:
        print(f"  {'FinBERT':<20} {'SKIPPED':<10} {'n/a':<10}")
    if haiku_evaluated > 0:
        print(f"  {'Claude Haiku':<20} {haiku_correct}/{haiku_evaluated:<9} {haiku_correct/max(haiku_evaluated,1)*100:.0f}%")
    else:
        print(f"  {'Claude Haiku':<20} {'SKIPPED':<10} {'n/a':<10}")

    # Per-category breakdown
    print(f"\n  {'Category':<25} {'FinBERT':<10} {'Haiku':<10}")
    print(f"  {'-'*45}")
    categories = {}
    for r in results:
        cat = r["category"]
        if cat not in categories:
            categories[cat] = {"finbert": [], "haiku": []}
        if r["finbert_correct"] is not None:
            categories[cat]["finbert"].append(r["finbert_correct"])
        if r["haiku_correct"] is not None:
            categories[cat]["haiku"].append(r["haiku_correct"])

    for cat, scores in categories.items():
        fb = f"{sum(scores['finbert'])}/{len(scores['finbert'])}" if scores["finbert"] else "n/a"
        hk = f"{sum(scores['haiku'])}/{len(scores['haiku'])}" if scores["haiku"] else "n/a"
        print(f"  {cat:<25} {fb:<10} {hk:<10}")

    # Token cost estimate
    if total_input_tokens == 0 and total_output_tokens == 0:
        print(f"\n  Token Usage: N/A (Haiku was skipped)")
        print(f"\n  To run with Haiku: export ANTHROPIC_API_KEY=your_key")
        return

    print(f"\n  Token Usage (Claude Haiku):")
    print(f"    Input tokens:  {total_input_tokens:,}")
    print(f"    Output tokens: {total_output_tokens:,}")

    # Haiku 4.5 pricing: $1/M input, $5/M output
    input_cost = total_input_tokens / 1_000_000 * 1.0
    output_cost = total_output_tokens / 1_000_000 * 5.0
    total_cost = input_cost + output_cost
    print(f"    Cost for {len(TEST_POSTS)} posts: ${total_cost:.4f}")

    per_post_cost = total_cost / len(TEST_POSTS) if TEST_POSTS else 0
    daily_estimate = per_post_cost * 24000
    monthly_estimate = daily_estimate * 30
    print(f"    Per post:     ${per_post_cost:.6f}")
    print(f"    Estimated daily (24K posts):   ${daily_estimate:.2f}")
    print(f"    Estimated monthly (720K posts): ${monthly_estimate:.2f}")

    # Tiered estimate (only 25% go to Haiku)
    tiered_daily = per_post_cost * 24000 * 0.25
    tiered_monthly = tiered_daily * 30
    print(f"\n  Tiered Pipeline Estimate (25% to Haiku):")
    print(f"    Daily:   ${tiered_daily:.2f}")
    print(f"    Monthly: ${tiered_monthly:.2f}")


if __name__ == "__main__":
    run_comparison()
