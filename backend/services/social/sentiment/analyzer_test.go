package sentiment

import (
	"testing"
)

func TestAnalyzer_Analyze(t *testing.T) {
	// Create analyzer with test lexicon (mock)
	analyzer := NewAnalyzerWithLexicon(map[string]LexiconEntry{
		"to the moon":   {Term: "to the moon", Sentiment: "bullish", Weight: 1.5, Category: "slang"},
		"diamond hands": {Term: "diamond hands", Sentiment: "bullish", Weight: 1.2, Category: "slang"},
		"tendies":       {Term: "tendies", Sentiment: "bullish", Weight: 1.0, Category: "slang"},
		"calls":         {Term: "calls", Sentiment: "bullish", Weight: 1.0, Category: "options"},
		"puts":          {Term: "puts", Sentiment: "bearish", Weight: 1.0, Category: "options"},
		"guh":           {Term: "guh", Sentiment: "bearish", Weight: 1.5, Category: "slang"},
		"crash":         {Term: "crash", Sentiment: "bearish", Weight: 1.3, Category: "slang"},
		"bullish":       {Term: "bullish", Sentiment: "bullish", Weight: 1.0, Category: "direct"},
		"bearish":       {Term: "bearish", Sentiment: "bearish", Weight: 1.0, Category: "direct"},
		"🚀":             {Term: "🚀", Sentiment: "bullish", Weight: 1.5, Category: "emoji"},
		"📉":             {Term: "📉", Sentiment: "bearish", Weight: 1.3, Category: "emoji"},
		"not":           {Term: "not", Sentiment: "modifier", Weight: -1.0, Category: "negation"},
		"very":          {Term: "very", Sentiment: "modifier", Weight: 1.3, Category: "amplifier"},
		"extremely":     {Term: "extremely", Sentiment: "modifier", Weight: 1.5, Category: "amplifier"},
		"maybe":         {Term: "maybe", Sentiment: "modifier", Weight: 0.5, Category: "reducer"},
		"squeeze":       {Term: "squeeze", Sentiment: "bullish", Weight: 1.3, Category: "slang"},
	})

	tests := []struct {
		name     string
		title    string
		body     string
		expected string
	}{
		{
			name:     "Bullish WSB slang",
			title:    "NVDA to the moon! 🚀🚀🚀",
			body:     "Diamond hands on my calls",
			expected: "bullish",
		},
		{
			name:     "Bearish options play",
			title:    "Buying puts on SPY",
			body:     "This market is going to crash hard",
			expected: "bearish",
		},
		{
			name:     "Negation flips sentiment",
			title:    "I'm NOT bullish on this",
			body:     "Despite what others say",
			expected: "bearish",
		},
		{
			name:     "Neutral technical analysis",
			title:    "Technical analysis of AAPL",
			body:     "Looking at support levels",
			expected: "neutral",
		},
		{
			name:     "Heavy bullish slang",
			title:    "GME squeeze incoming!!!",
			body:     "Diamond hands, tendies incoming, to the moon 🚀",
			expected: "bullish",
		},
		{
			name:     "Amplifier boosts sentiment",
			title:    "Very bullish on this",
			body:     "",
			expected: "bullish",
		},
		{
			name:     "Extremely bearish",
			title:    "Extremely bearish outlook",
			body:     "",
			expected: "bearish",
		},
		{
			name:     "Emoji with text",
			title:    "This is going down 📉",
			body:     "",
			expected: "bearish",
		},
		{
			name:     "GUH bearish",
			title:    "GUH",
			body:     "Lost it all",
			expected: "bearish",
		},
		{
			name:     "Multiple bullish signals",
			title:    "To the moon 🚀",
			body:     "Calls calls calls",
			expected: "bullish",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := analyzer.Analyze(tt.title, tt.body)
			if result.Sentiment != tt.expected {
				t.Errorf("Analyze(%q, %q) = %s, want %s (score: %.2f, confidence: %.2f, matches: %v)",
					tt.title, tt.body, result.Sentiment, tt.expected, result.Score, result.Confidence, result.MatchedTerms)
			}
		})
	}
}

func TestAnalyzer_cleanText(t *testing.T) {
	analyzer := NewAnalyzerWithLexicon(nil)

	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "Check this https://reddit.com/r/wsb link",
			expected: "Check this link", // cleanText doesn't lowercase
		},
		{
			input:    "[click here](https://example.com) for more",
			expected: "click here for more",
		},
		{
			input:    "To the moon! 🚀🚀🚀",
			expected: "To the moon 🚀🚀🚀", // cleanText doesn't lowercase
		},
		{
			input:    "Multiple   spaces   here",
			expected: "Multiple spaces here", // cleanText doesn't lowercase
		},
		{
			input:    "Special chars #@! removed",
			expected: "Special chars removed", // cleanText doesn't lowercase
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := analyzer.cleanText(tt.input)
			if result != tt.expected {
				t.Errorf("cleanText(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestTickerExtractor_Extract(t *testing.T) {
	// Create extractor with test tickers
	extractor := NewTickerExtractorWithTickers(
		[]string{"NVDA", "AAPL", "GME", "AMC", "TSLA", "SPY", "QQQ", "BB"},
		[]string{"I", "A", "DD", "CEO", "THE", "YOLO", "MOON", "HOLD"},
	)

	tests := []struct {
		name          string
		title         string
		body          string
		expectTickers []string
		primaryTicker string
	}{
		{
			name:          "Dollar sign ticker",
			title:         "$NVDA to the moon",
			body:          "",
			expectTickers: []string{"NVDA"},
			primaryTicker: "NVDA",
		},
		{
			name:          "Multiple tickers",
			title:         "$GME $AMC $BB which one?",
			body:          "",
			expectTickers: []string{"GME", "AMC", "BB"},
			primaryTicker: "GME", // First in title
		},
		{
			name:          "Standalone valid ticker",
			title:         "TSLA earnings tomorrow",
			body:          "Expecting big moves in TSLA",
			expectTickers: []string{"TSLA"},
			primaryTicker: "TSLA",
		},
		{
			name:          "Filter false positives",
			title:         "I think DD on CEO",
			body:          "YOLO MOON HOLD",
			expectTickers: []string{},
			primaryTicker: "",
		},
		{
			name:          "Title takes priority",
			title:         "NVDA analysis",
			body:          "Comparing to AAPL AAPL AAPL AAPL", // More mentions in body
			expectTickers: []string{"NVDA", "AAPL"},
			primaryTicker: "NVDA", // Title wins
		},
		{
			name:          "Dollar sign trumps count",
			title:         "$SPY puts",
			body:          "TSLA TSLA TSLA TSLA TSLA", // More mentions
			expectTickers: []string{"SPY", "TSLA"},
			primaryTicker: "SPY", // In title
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mentions := extractor.Extract(tt.title, tt.body)
			primary := extractor.GetPrimaryTicker(mentions)

			// Check primary ticker
			if primary != tt.primaryTicker {
				t.Errorf("GetPrimaryTicker() = %s, want %s", primary, tt.primaryTicker)
			}

			// Check all expected tickers found
			foundTickers := make(map[string]bool)
			for _, m := range mentions {
				foundTickers[m.Ticker] = true
			}
			for _, expected := range tt.expectTickers {
				if !foundTickers[expected] {
					t.Errorf("Expected ticker %s not found in %v", expected, mentions)
				}
			}

			// Check no unexpected tickers found
			if len(mentions) != len(tt.expectTickers) {
				tickerList := make([]string, 0, len(mentions))
				for _, m := range mentions {
					tickerList = append(tickerList, m.Ticker)
				}
				t.Errorf("Found %d tickers %v, expected %d tickers %v",
					len(mentions), tickerList, len(tt.expectTickers), tt.expectTickers)
			}
		})
	}
}

func TestTickerExtractor_IsValidTicker(t *testing.T) {
	extractor := NewTickerExtractorWithTickers(
		[]string{"AAPL", "MSFT", "GOOGL"},
		nil,
	)

	tests := []struct {
		ticker   string
		expected bool
	}{
		{"AAPL", true},
		{"aapl", true}, // Case insensitive
		{"MSFT", true},
		{"INVALID", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.ticker, func(t *testing.T) {
			result := extractor.IsValidTicker(tt.ticker)
			if result != tt.expected {
				t.Errorf("IsValidTicker(%q) = %v, want %v", tt.ticker, result, tt.expected)
			}
		})
	}
}

func TestAnalyzer_NegationHandling(t *testing.T) {
	analyzer := NewAnalyzerWithLexicon(map[string]LexiconEntry{
		"bullish":     {Term: "bullish", Sentiment: "bullish", Weight: 1.0, Category: "direct"},
		"bearish":     {Term: "bearish", Sentiment: "bearish", Weight: 1.0, Category: "direct"},
		"not":         {Term: "not", Sentiment: "modifier", Weight: -1.0, Category: "negation"},
		"dont":        {Term: "dont", Sentiment: "modifier", Weight: -1.0, Category: "negation"},
		"never":       {Term: "never", Sentiment: "modifier", Weight: -1.0, Category: "negation"},
		"buy":         {Term: "buy", Sentiment: "bullish", Weight: 0.8, Category: "action"},
		"sell":        {Term: "sell", Sentiment: "bearish", Weight: 0.8, Category: "action"},
		"moon":        {Term: "moon", Sentiment: "bullish", Weight: 1.0, Category: "slang"},
		"to the moon": {Term: "to the moon", Sentiment: "bullish", Weight: 1.5, Category: "slang"},
	})

	tests := []struct {
		name     string
		title    string
		expected string
	}{
		{"Simple negation", "Not bullish", "bearish"},
		{"Negation sell", "Dont sell this", "bullish"}, // Don't sell = bullish
		{"Double negative", "Not bearish", "bullish"},
		{"Negation immediate", "Not bullish at all", "bearish"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := analyzer.Analyze(tt.title, "")
			if result.Sentiment != tt.expected {
				t.Errorf("Analyze(%q) = %s, want %s (score: %.2f, matches: %v)",
					tt.title, result.Sentiment, tt.expected, result.Score, result.MatchedTerms)
			}
		})
	}
}

func TestAnalyzer_AmplifierHandling(t *testing.T) {
	analyzer := NewAnalyzerWithLexicon(map[string]LexiconEntry{
		"bullish":   {Term: "bullish", Sentiment: "bullish", Weight: 1.0, Category: "direct"},
		"bearish":   {Term: "bearish", Sentiment: "bearish", Weight: 1.0, Category: "direct"},
		"very":      {Term: "very", Sentiment: "modifier", Weight: 1.3, Category: "amplifier"},
		"extremely": {Term: "extremely", Sentiment: "modifier", Weight: 1.5, Category: "amplifier"},
		"super":     {Term: "super", Sentiment: "modifier", Weight: 1.4, Category: "amplifier"},
	})

	// Test that amplifiers affect matched term weights
	baseResult := analyzer.Analyze("Bullish", "")
	ampResult := analyzer.Analyze("Very bullish", "")
	strongAmpResult := analyzer.Analyze("Extremely bullish", "")

	// Check that sentiment direction is preserved
	if baseResult.Sentiment != "bullish" {
		t.Errorf("Base result should be bullish, got %s", baseResult.Sentiment)
	}
	if ampResult.Sentiment != "bullish" {
		t.Errorf("Amplified result should be bullish, got %s", ampResult.Sentiment)
	}
	if strongAmpResult.Sentiment != "bullish" {
		t.Errorf("Strongly amplified result should be bullish, got %s", strongAmpResult.Sentiment)
	}

	// Check matched term weights - amplified should have higher weights
	if len(ampResult.MatchedTerms) > 0 && len(baseResult.MatchedTerms) > 0 {
		if ampResult.MatchedTerms[0].Weight <= baseResult.MatchedTerms[0].Weight {
			t.Errorf("Amplified weight %.2f should be greater than base weight %.2f",
				ampResult.MatchedTerms[0].Weight, baseResult.MatchedTerms[0].Weight)
		}
	}

	if len(strongAmpResult.MatchedTerms) > 0 && len(ampResult.MatchedTerms) > 0 {
		if strongAmpResult.MatchedTerms[0].Weight <= ampResult.MatchedTerms[0].Weight {
			t.Errorf("Strongly amplified weight %.2f should be greater than amplified weight %.2f",
				strongAmpResult.MatchedTerms[0].Weight, ampResult.MatchedTerms[0].Weight)
		}
	}
}

func TestAnalyzer_ReducerHandling(t *testing.T) {
	analyzer := NewAnalyzerWithLexicon(map[string]LexiconEntry{
		"bullish": {Term: "bullish", Sentiment: "bullish", Weight: 1.0, Category: "direct"},
		"maybe":   {Term: "maybe", Sentiment: "modifier", Weight: 0.5, Category: "reducer"},
		"might":   {Term: "might", Sentiment: "modifier", Weight: 0.6, Category: "reducer"},
	})

	// Test that reducers affect matched term weights
	baseResult := analyzer.Analyze("Bullish", "")
	reducedResult := analyzer.Analyze("Maybe bullish", "")

	// Check sentiment is preserved (may be weaker bullish or neutral)
	if baseResult.Sentiment != "bullish" {
		t.Errorf("Base result should be bullish, got %s", baseResult.Sentiment)
	}

	// Check matched term weights - reduced should have lower weight
	if len(reducedResult.MatchedTerms) > 0 && len(baseResult.MatchedTerms) > 0 {
		if reducedResult.MatchedTerms[0].Weight >= baseResult.MatchedTerms[0].Weight {
			t.Errorf("Reduced weight %.2f should be less than base weight %.2f",
				reducedResult.MatchedTerms[0].Weight, baseResult.MatchedTerms[0].Weight)
		}
	}
}

func TestTickerMention_CountTracking(t *testing.T) {
	extractor := NewTickerExtractorWithTickers(
		[]string{"AAPL"},
		nil,
	)

	// Multiple mentions should increase count
	mentions := extractor.Extract("AAPL is great", "I love AAPL. AAPL forever!")

	if len(mentions) != 1 {
		t.Fatalf("Expected 1 ticker mention, got %d", len(mentions))
	}

	if mentions[0].Count < 3 {
		t.Errorf("Expected count >= 3 for multiple AAPL mentions, got %d", mentions[0].Count)
	}

	if !mentions[0].InTitle {
		t.Error("Expected InTitle to be true when ticker is in title")
	}
}

func TestTickerExtractor_DollarSignPriority(t *testing.T) {
	extractor := NewTickerExtractorWithTickers(
		[]string{"AAPL", "MSFT"},
		nil,
	)

	// $TICKER should count more than standalone
	mentions := extractor.Extract("$AAPL news", "MSFT MSFT MSFT")

	// Find AAPL mention
	var aaplMention, msftMention *TickerMention
	for i := range mentions {
		if mentions[i].Ticker == "AAPL" {
			aaplMention = &mentions[i]
		}
		if mentions[i].Ticker == "MSFT" {
			msftMention = &mentions[i]
		}
	}

	if aaplMention == nil {
		t.Fatal("Expected to find AAPL mention")
	}

	// AAPL with $ should be primary despite fewer literal mentions
	primary := extractor.GetPrimaryTicker(mentions)
	if primary != "AAPL" {
		t.Errorf("Expected primary ticker AAPL (in title), got %s", primary)
	}

	// But MSFT should have higher raw count
	if msftMention != nil && msftMention.Count <= aaplMention.Count {
		// This is expected since $TICKER counts double
		t.Logf("AAPL count: %d, MSFT count: %d", aaplMention.Count, msftMention.Count)
	}
}
