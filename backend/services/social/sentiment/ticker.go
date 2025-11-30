package sentiment

import (
	"regexp"
	"strings"

	"investorcenter-api/database"
)

// TickerMention represents a ticker found in text
type TickerMention struct {
	Ticker    string
	Count     int
	InTitle   bool
	Positions []int
}

// TickerExtractor finds and validates ticker symbols
type TickerExtractor struct {
	validTickers   map[string]bool
	falsePositives map[string]bool
}

// Common words that look like tickers but aren't
var defaultFalsePositives = []string{
	// Single letters and common abbreviations
	"I", "A", "IT", "AT", "ON", "IS", "BE", "DO", "GO", "SO", "TO", "UP",
	// Business terms
	"CEO", "CFO", "CTO", "COO", "DD", "IPO", "EPS", "PE", "ETF", "SEC", "FED",
	// Geographic
	"USA", "USD", "UK", "EU",
	// Economic
	"GDP", "CPI",
	// Reddit/Internet slang
	"IMO", "TBH", "LOL", "EDIT", "UPDATE", "TL", "DR", "TLDR", "AMA", "ELI5", "NSFW", "OP", "WSB",
	// Common words
	"FOR", "THE", "AND", "ARE", "BUT", "NOT", "YOU", "ALL", "CAN", "HER",
	"WAS", "ONE", "OUR", "OUT", "DAY", "HAD", "HAS", "HIS", "HOW", "ITS",
	"MAY", "NEW", "NOW", "OLD", "SEE", "WAY", "WHO", "OIL", "ANY", "BOT",
	"BUY", "PUT", "GET", "LET", "RUN", "SET", "TOP", "LOW", "HIGH", "LONG",
	"HOLD", "SELL", "CALL", "YOLO", "MOON", "BEAR", "BULL", "GAIN", "LOSS",
	// Time-related
	"AM", "PM", "EST", "PST", "UTC",
	// Other common false positives
	"AI", "API", "ATH", "ATL", "AH", "PM", "EV", "FD", "IV", "OTM", "ITM",
	"RH", "TD", "WK", "YTD", "QE", "SPAC", "HODL", "FUD", "FOMO",
}

// NewTickerExtractor creates extractor with valid tickers from database
func NewTickerExtractor() (*TickerExtractor, error) {
	validTickers := make(map[string]bool)

	// Query tickers table - the source of truth
	rows, err := database.DB.Query(`
		SELECT symbol FROM tickers
		WHERE asset_type IN ('stock', 'etf', 'CS', 'ETF')
		  AND symbol ~ '^[A-Z]{1,5}$'
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var symbol string
		if err := rows.Scan(&symbol); err != nil {
			return nil, err
		}
		validTickers[strings.ToUpper(symbol)] = true
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	// Build false positives map
	falsePositives := make(map[string]bool)
	for _, fp := range defaultFalsePositives {
		falsePositives[fp] = true
	}

	return &TickerExtractor{
		validTickers:   validTickers,
		falsePositives: falsePositives,
	}, nil
}

// NewTickerExtractorWithTickers creates an extractor with provided tickers (for testing)
func NewTickerExtractorWithTickers(tickers []string, falsePositives []string) *TickerExtractor {
	validTickers := make(map[string]bool)
	for _, t := range tickers {
		validTickers[strings.ToUpper(t)] = true
	}

	fpMap := make(map[string]bool)
	for _, fp := range falsePositives {
		fpMap[fp] = true
	}

	return &TickerExtractor{
		validTickers:   validTickers,
		falsePositives: fpMap,
	}
}

// Extract finds all ticker mentions in title and body
func (t *TickerExtractor) Extract(title, body string) []TickerMention {
	mentions := make(map[string]*TickerMention)

	// Pattern: $TICKER (explicit, high confidence)
	dollarPattern := regexp.MustCompile(`\$([A-Z]{1,5})\b`)
	// Pattern: standalone TICKER (1-5 uppercase letters, word boundary)
	standalonePattern := regexp.MustCompile(`\b([A-Z]{1,5})\b`)

	// Extract from title (higher weight)
	t.extractFromText(title, dollarPattern, standalonePattern, mentions, true)

	// Extract from body
	t.extractFromText(body, dollarPattern, standalonePattern, mentions, false)

	// Convert map to slice and filter
	var result []TickerMention
	for _, m := range mentions {
		// Include if:
		// 1. Valid ticker in our database, OR
		// 2. Mentioned multiple times (likely intentional)
		if t.validTickers[m.Ticker] || m.Count >= 3 {
			result = append(result, *m)
		}
	}

	return result
}

func (t *TickerExtractor) extractFromText(text string, dollarPattern, standalonePattern *regexp.Regexp, mentions map[string]*TickerMention, isTitle bool) {
	// First find $TICKER patterns (high confidence, always include)
	dollarMatches := dollarPattern.FindAllStringSubmatchIndex(text, -1)
	for _, match := range dollarMatches {
		ticker := strings.ToUpper(text[match[2]:match[3]])
		if t.falsePositives[ticker] {
			continue
		}

		if m, ok := mentions[ticker]; ok {
			m.Count += 2 // $TICKER counts double
			if isTitle {
				m.InTitle = true
			}
			m.Positions = append(m.Positions, match[0])
		} else {
			mentions[ticker] = &TickerMention{
				Ticker:    ticker,
				Count:     2,
				InTitle:   isTitle,
				Positions: []int{match[0]},
			}
		}
	}

	// Then find standalone patterns (only if valid ticker in database)
	standaloneMatches := standalonePattern.FindAllStringSubmatchIndex(text, -1)
	for _, match := range standaloneMatches {
		ticker := text[match[2]:match[3]]
		if t.falsePositives[ticker] {
			continue
		}
		if !t.validTickers[ticker] {
			continue // Only accept valid tickers for standalone mentions
		}

		if m, ok := mentions[ticker]; ok {
			m.Count++
			if isTitle {
				m.InTitle = true
			}
			m.Positions = append(m.Positions, match[0])
		} else {
			mentions[ticker] = &TickerMention{
				Ticker:    ticker,
				Count:     1,
				InTitle:   isTitle,
				Positions: []int{match[0]},
			}
		}
	}
}

// GetPrimaryTicker determines the main ticker being discussed
func (t *TickerExtractor) GetPrimaryTicker(mentions []TickerMention) string {
	if len(mentions) == 0 {
		return ""
	}

	// If only one ticker, return it
	if len(mentions) == 1 {
		return mentions[0].Ticker
	}

	// Priority: in title > highest count
	var best *TickerMention
	for i := range mentions {
		m := &mentions[i]
		if best == nil {
			best = m
			continue
		}

		// Strong preference for tickers in title
		if m.InTitle && !best.InTitle {
			best = m
			continue
		}
		if !m.InTitle && best.InTitle {
			continue
		}

		// Then by mention count
		if m.Count > best.Count {
			best = m
		}
	}

	return best.Ticker
}

// IsValidTicker checks if a symbol is a known valid ticker
func (t *TickerExtractor) IsValidTicker(symbol string) bool {
	return t.validTickers[strings.ToUpper(symbol)]
}

// GetTickerCount returns number of valid tickers loaded
func (t *TickerExtractor) GetTickerCount() int {
	return len(t.validTickers)
}

// AddFalsePositive adds a term to the false positives list
func (t *TickerExtractor) AddFalsePositive(term string) {
	t.falsePositives[strings.ToUpper(term)] = true
}

// RemoveFalsePositive removes a term from the false positives list
func (t *TickerExtractor) RemoveFalsePositive(term string) {
	delete(t.falsePositives, strings.ToUpper(term))
}
