package sentiment

import (
	"regexp"
	"strings"
	"unicode"

	"investorcenter-api/database"
)

// LexiconEntry represents a term from the sentiment_lexicon table
type LexiconEntry struct {
	Term      string
	Sentiment string  // "bullish", "bearish", "modifier"
	Weight    float64
	Category  string  // "slang", "options", "emoji", "negation", "amplifier", "reducer"
}

// MatchedTerm represents a term found during analysis
type MatchedTerm struct {
	Term      string
	Sentiment string
	Weight    float64
	Position  int
}

// Result contains the sentiment analysis output
type Result struct {
	Sentiment    string        // "bullish", "bearish", "neutral"
	Score        float64       // -1.0 to 1.0
	Confidence   float64       // 0.0 to 1.0
	MatchedTerms []MatchedTerm
}

// Analyzer performs sentiment analysis using the lexicon
type Analyzer struct {
	lexicon map[string]LexiconEntry
}

// NewAnalyzer loads lexicon from database using existing DB singleton
func NewAnalyzer() (*Analyzer, error) {
	lexicon := make(map[string]LexiconEntry)

	rows, err := database.DB.Query(`
		SELECT term, sentiment, weight, COALESCE(category, '') as category
		FROM sentiment_lexicon
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var entry LexiconEntry
		if err := rows.Scan(&entry.Term, &entry.Sentiment, &entry.Weight, &entry.Category); err != nil {
			return nil, err
		}
		// Store lowercase for matching
		lexicon[strings.ToLower(entry.Term)] = entry
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return &Analyzer{lexicon: lexicon}, nil
}

// NewAnalyzerWithLexicon creates an analyzer with a provided lexicon (for testing)
func NewAnalyzerWithLexicon(lexicon map[string]LexiconEntry) *Analyzer {
	return &Analyzer{lexicon: lexicon}
}

// Analyze performs sentiment analysis on title and body text
func (a *Analyzer) Analyze(title, body string) Result {
	// Combine title (weighted higher by repeating) and body
	text := strings.ToLower(title + " " + title + " " + body)

	// Clean text: remove URLs, special chars, normalize whitespace
	text = a.cleanText(text)

	// Tokenize
	words := strings.Fields(text)

	var matches []MatchedTerm
	var bullishScore, bearishScore float64
	var totalWeight float64

	// Modifier state tracking
	negationActive := false
	negationCountdown := 0
	amplifierActive := false
	amplifierMultiplier := 1.0
	amplifierCountdown := 0

	skipUntil := 0 // For multi-word phrase handling

	for i := 0; i < len(words); i++ {
		if i < skipUntil {
			continue
		}

		// Check for multi-word phrases first (up to 4 words)
		matched := false
		for phraseLen := 4; phraseLen >= 1; phraseLen-- {
			if i+phraseLen > len(words) {
				continue
			}
			phrase := strings.Join(words[i:i+phraseLen], " ")
			entry, ok := a.lexicon[phrase]
			if !ok {
				continue
			}

			matched = true

			if entry.Sentiment == "modifier" {
				// Handle negation (negative weight)
				if entry.Weight < 0 {
					negationActive = true
					negationCountdown = 3 // Affects next 3 sentiment words
				} else if entry.Category == "amplifier" {
					// Handle amplifiers (very, extremely, etc.)
					amplifierActive = true
					amplifierMultiplier = entry.Weight
					amplifierCountdown = 2
				} else if entry.Category == "reducer" {
					// Handle reducers (maybe, might, etc.)
					amplifierActive = true
					amplifierMultiplier = entry.Weight // e.g., 0.5
					amplifierCountdown = 2
				}
			} else {
				// Sentiment term found
				weight := entry.Weight

				// Apply amplifier/reducer
				if amplifierActive {
					weight *= amplifierMultiplier
				}

				// Apply negation (flips sentiment)
				if negationActive {
					weight *= -1
				}

				matches = append(matches, MatchedTerm{
					Term:      phrase,
					Sentiment: entry.Sentiment,
					Weight:    weight,
					Position:  i,
				})

				// Accumulate scores based on original sentiment direction
				if entry.Sentiment == "bullish" {
					if weight > 0 {
						bullishScore += weight
					} else {
						bearishScore += absFloat(weight)
					}
				} else if entry.Sentiment == "bearish" {
					if weight > 0 {
						bearishScore += weight
					} else {
						bullishScore += absFloat(weight)
					}
				}
				totalWeight += absFloat(weight)
			}

			skipUntil = i + phraseLen
			break
		}

		// Decrement modifier countdowns
		if !matched {
			if negationActive {
				negationCountdown--
				if negationCountdown <= 0 {
					negationActive = false
				}
			}
			if amplifierActive {
				amplifierCountdown--
				if amplifierCountdown <= 0 {
					amplifierActive = false
					amplifierMultiplier = 1.0
				}
			}
		}
	}

	// Calculate final score and sentiment
	var score float64
	var sentiment string
	var confidence float64

	if totalWeight > 0 {
		score = (bullishScore - bearishScore) / totalWeight
		// Confidence based on signal density
		confidence = minFloat(totalWeight/float64(len(words))*10, 1.0)
	}

	// Determine sentiment label
	if score > 0.1 {
		sentiment = "bullish"
	} else if score < -0.1 {
		sentiment = "bearish"
	} else {
		sentiment = "neutral"
	}

	// Boost confidence if score is extreme
	if absFloat(score) > 0.5 {
		confidence = minFloat(confidence*1.2, 1.0)
	}

	// Minimum confidence threshold
	if len(matches) == 0 {
		confidence = 0
	} else if len(matches) == 1 {
		confidence = minFloat(confidence, 0.5)
	}

	return Result{
		Sentiment:    sentiment,
		Score:        score,
		Confidence:   confidence,
		MatchedTerms: matches,
	}
}

// cleanText removes URLs, handles Reddit markdown, normalizes text
func (a *Analyzer) cleanText(text string) string {
	// Remove URLs
	urlRegex := regexp.MustCompile(`https?://\S+`)
	text = urlRegex.ReplaceAllString(text, "")

	// Remove Reddit markdown links [text](url)
	linkRegex := regexp.MustCompile(`\[([^\]]+)\]\([^)]+\)`)
	text = linkRegex.ReplaceAllString(text, "$1")

	// Keep emojis but remove other special chars
	var cleaned strings.Builder
	for _, r := range text {
		if unicode.IsLetter(r) || unicode.IsNumber(r) || unicode.IsSpace(r) || r > 127 {
			cleaned.WriteRune(r)
		} else {
			cleaned.WriteRune(' ')
		}
	}

	// Normalize whitespace
	return strings.Join(strings.Fields(cleaned.String()), " ")
}

// RefreshLexicon reloads the lexicon from database (for hot updates)
func (a *Analyzer) RefreshLexicon() error {
	newAnalyzer, err := NewAnalyzer()
	if err != nil {
		return err
	}
	a.lexicon = newAnalyzer.lexicon
	return nil
}

// GetLexiconSize returns the number of terms in the lexicon
func (a *Analyzer) GetLexiconSize() int {
	return len(a.lexicon)
}

func absFloat(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
