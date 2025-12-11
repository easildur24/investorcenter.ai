package sentiment

import (
	"context"
	"fmt"
	"time"

	"investorcenter-api/database"
	"investorcenter-api/models"
	"investorcenter-api/services/social"
)

// Service combines analyzer and ticker extractor
type Service struct {
	analyzer  *Analyzer
	extractor *TickerExtractor
}

// NewService creates a new sentiment analysis service
func NewService() (*Service, error) {
	analyzer, err := NewAnalyzer()
	if err != nil {
		return nil, fmt.Errorf("failed to create analyzer: %w", err)
	}

	extractor, err := NewTickerExtractor()
	if err != nil {
		return nil, fmt.Errorf("failed to create ticker extractor: %w", err)
	}

	return &Service{
		analyzer:  analyzer,
		extractor: extractor,
	}, nil
}

// NewServiceWithComponents creates a service with provided components (for testing)
func NewServiceWithComponents(analyzer *Analyzer, extractor *TickerExtractor) *Service {
	return &Service{
		analyzer:  analyzer,
		extractor: extractor,
	}
}

// ProcessPost analyzes a raw post and returns processed post with sentiment
// Returns nil if no valid ticker is found
func (s *Service) ProcessPost(raw social.RawPost) *social.ProcessedPost {
	// Extract tickers
	mentions := s.extractor.Extract(raw.Title, raw.Body)
	primaryTicker := s.extractor.GetPrimaryTicker(mentions)

	if primaryTicker == "" {
		return nil // No valid ticker found, skip this post
	}

	// Analyze sentiment
	result := s.analyzer.Analyze(raw.Title, raw.Body)

	// Create body preview (first 500 chars)
	bodyPreview := raw.Body
	if len(bodyPreview) > 500 {
		bodyPreview = bodyPreview[:497] + "..."
	}

	return &social.ProcessedPost{
		RawPost:             raw,
		Ticker:              primaryTicker,
		BodyPreview:         bodyPreview,
		Sentiment:           result.Sentiment,
		SentimentConfidence: result.Confidence,
	}
}

// ProcessAndSavePost processes a post and saves it to the database
func (s *Service) ProcessAndSavePost(raw social.RawPost) (*models.SocialPost, error) {
	processed := s.ProcessPost(raw)
	if processed == nil {
		return nil, nil // No ticker found
	}

	// Convert to database model
	post := &models.SocialPost{
		ExternalPostID: processed.ExternalID,
		Source:         processed.Source,
		Ticker:         processed.Ticker,
		Subreddit:      processed.Community,
		Title:          processed.Title,
		URL:            processed.URL,
		Upvotes:        processed.Upvotes,
		CommentCount:   processed.CommentCount,
		AwardCount:     processed.AwardCount,
		PostedAt:       processed.PostedAt,
	}

	// Set nullable fields
	if processed.BodyPreview != "" {
		post.BodyPreview = &processed.BodyPreview
	}
	if processed.Sentiment != "" {
		post.Sentiment = &processed.Sentiment
	}
	if processed.SentimentConfidence > 0 {
		post.SentimentConfidence = &processed.SentimentConfidence
	}
	if processed.Flair != "" {
		post.Flair = &processed.Flair
	}

	// Save to database
	if err := database.UpsertSocialPost(post); err != nil {
		return nil, fmt.Errorf("failed to save post: %w", err)
	}

	return post, nil
}

// ProcessPosts processes multiple posts and returns processed posts with sentiment
// Skips posts without valid tickers
func (s *Service) ProcessPosts(rawPosts []social.RawPost) []social.ProcessedPost {
	var processed []social.ProcessedPost
	for _, raw := range rawPosts {
		if p := s.ProcessPost(raw); p != nil {
			processed = append(processed, *p)
		}
	}
	return processed
}

// ProcessAndSavePosts processes multiple posts and saves them to the database
// Returns the number of successfully saved posts
func (s *Service) ProcessAndSavePosts(rawPosts []social.RawPost) (int, error) {
	saved := 0
	for _, raw := range rawPosts {
		post, err := s.ProcessAndSavePost(raw)
		if err != nil {
			// Log error but continue processing other posts
			continue
		}
		if post != nil {
			saved++
		}
	}
	return saved, nil
}

// AnalyzeText performs sentiment analysis on arbitrary text (for testing/API)
func (s *Service) AnalyzeText(title, body string) Result {
	return s.analyzer.Analyze(title, body)
}

// ExtractTickers extracts ticker symbols from text (for testing/API)
func (s *Service) ExtractTickers(title, body string) []TickerMention {
	return s.extractor.Extract(title, body)
}

// UpdateHeatmapSentiment aggregates post sentiment into reddit_heatmap_daily
func (s *Service) UpdateHeatmapSentiment(ctx context.Context, tickerSymbol string, date time.Time) error {
	query := `
		WITH post_counts AS (
			SELECT
				COUNT(*) FILTER (WHERE sentiment = 'bullish') as bullish,
				COUNT(*) FILTER (WHERE sentiment = 'bearish') as bearish,
				COUNT(*) FILTER (WHERE sentiment = 'neutral' OR sentiment IS NULL) as neutral,
				COUNT(*) as total
			FROM social_posts
			WHERE ticker = $1
			  AND DATE(posted_at) = $2
		)
		UPDATE reddit_heatmap_daily
		SET
			bullish_count = pc.bullish,
			bearish_count = pc.bearish,
			neutral_count = pc.neutral,
			sentiment_score = CASE
				WHEN pc.total > 0 THEN
					((pc.bullish::DECIMAL - pc.bearish::DECIMAL) / pc.total) * 100
				ELSE 0
			END
		FROM post_counts pc
		WHERE reddit_heatmap_daily.ticker_symbol = $1
		  AND reddit_heatmap_daily.date = $2
	`

	_, err := database.DB.ExecContext(ctx, query, tickerSymbol, date.Format("2006-01-02"))
	return err
}

// UpdateAllHeatmapSentiment updates sentiment for all tickers on a given date
func (s *Service) UpdateAllHeatmapSentiment(ctx context.Context, date time.Time) error {
	// Get all unique tickers from posts on this date
	query := `
		SELECT DISTINCT ticker FROM social_posts
		WHERE DATE(posted_at) = $1
	`

	rows, err := database.DB.QueryContext(ctx, query, date.Format("2006-01-02"))
	if err != nil {
		return fmt.Errorf("failed to get tickers: %w", err)
	}
	defer rows.Close()

	var tickers []string
	for rows.Next() {
		var ticker string
		if err := rows.Scan(&ticker); err != nil {
			return fmt.Errorf("failed to scan ticker: %w", err)
		}
		tickers = append(tickers, ticker)
	}

	// Update each ticker
	for _, ticker := range tickers {
		if err := s.UpdateHeatmapSentiment(ctx, ticker, date); err != nil {
			// Log error but continue with other tickers
			continue
		}
	}

	return nil
}

// GetAnalyzerStats returns statistics about the loaded lexicon
func (s *Service) GetAnalyzerStats() map[string]int {
	stats := make(map[string]int)
	stats["lexicon_terms"] = s.analyzer.GetLexiconSize()
	stats["valid_tickers"] = s.extractor.GetTickerCount()
	return stats
}

// Refresh reloads both lexicon and ticker list from database
func (s *Service) Refresh() error {
	if err := s.analyzer.RefreshLexicon(); err != nil {
		return fmt.Errorf("failed to refresh lexicon: %w", err)
	}

	newExtractor, err := NewTickerExtractor()
	if err != nil {
		return fmt.Errorf("failed to refresh ticker extractor: %w", err)
	}
	s.extractor = newExtractor

	return nil
}

// IsValidTicker checks if a symbol is a known valid ticker
func (s *Service) IsValidTicker(symbol string) bool {
	return s.extractor.IsValidTicker(symbol)
}

// GetPrimaryTicker determines the main ticker being discussed in text
func (s *Service) GetPrimaryTicker(title, body string) string {
	mentions := s.extractor.Extract(title, body)
	return s.extractor.GetPrimaryTicker(mentions)
}
