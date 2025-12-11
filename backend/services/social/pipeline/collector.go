package pipeline

import (
	"context"
	"fmt"
	"log"
	"time"

	"investorcenter-api/database"
	"investorcenter-api/models"
	"investorcenter-api/services/social"
	"investorcenter-api/services/social/sentiment"
)

// Collector fetches and processes social media posts
type Collector struct {
	dataSources   []social.SocialDataSource
	sentimentSvc  *sentiment.Service
	subreddits    []string
	minEngagement int
}

// Config for the collector
type Config struct {
	Subreddits    []string
	MinEngagement int // Minimum upvotes to store
}

// NewCollector creates a new post collector
func NewCollector(sources []social.SocialDataSource, cfg Config) (*Collector, error) {
	sentimentSvc, err := sentiment.NewService()
	if err != nil {
		return nil, fmt.Errorf("failed to create sentiment service: %w", err)
	}

	if len(cfg.Subreddits) == 0 {
		cfg.Subreddits = []string{"wallstreetbets", "stocks", "options", "investing", "Daytrading"}
	}
	if cfg.MinEngagement == 0 {
		cfg.MinEngagement = 10
	}

	return &Collector{
		dataSources:   sources,
		sentimentSvc:  sentimentSvc,
		subreddits:    cfg.Subreddits,
		minEngagement: cfg.MinEngagement,
	}, nil
}

// Run executes one collection cycle (called by CronJob)
func (c *Collector) Run(ctx context.Context) error {
	log.Println("Starting post collection cycle")
	startTime := time.Now()

	var totalProcessed, totalStored, totalUpdated int

	for _, source := range c.dataSources {
		if !source.IsEnabled() {
			log.Printf("Skipping disabled source: %s", source.GetName())
			continue
		}

		for _, subreddit := range c.subreddits {
			processed, stored, updated, err := c.processSubreddit(ctx, source, subreddit)
			if err != nil {
				log.Printf("Error processing %s/%s: %v", source.GetName(), subreddit, err)
				continue
			}

			totalProcessed += processed
			totalStored += stored
			totalUpdated += updated
		}
	}

	// Prune old posts
	pruned, err := database.PruneOldPosts(30) // 30 day retention
	if err != nil {
		log.Printf("Error pruning old posts: %v", err)
	} else if pruned > 0 {
		log.Printf("Pruned %d old posts", pruned)
	}

	log.Printf("Collection complete: processed=%d, stored=%d, updated=%d, duration=%v",
		totalProcessed, totalStored, totalUpdated, time.Since(startTime))

	return nil
}

func (c *Collector) processSubreddit(ctx context.Context, source social.SocialDataSource, subreddit string) (processed, stored, updated int, err error) {
	log.Printf("Fetching posts from %s/%s", source.GetName(), subreddit)

	posts, err := source.FetchPosts(ctx, social.FetchOptions{
		Subreddit: subreddit,
		Limit:     100,
		Since:     time.Now().Add(-24 * time.Hour),
	})
	if err != nil {
		return 0, 0, 0, err
	}

	log.Printf("Fetched %d posts from %s/%s", len(posts), source.GetName(), subreddit)

	for _, raw := range posts {
		processed++

		// Skip low-engagement posts
		if raw.Upvotes < c.minEngagement {
			continue
		}

		// Check if post already exists
		existingCount, _ := database.GetPostCountByExternalID(raw.ExternalID)
		if existingCount > 0 {
			// Update engagement metrics
			if err := c.updatePostMetrics(raw); err != nil {
				log.Printf("Error updating post %s: %v", raw.ExternalID, err)
			} else {
				updated++
			}
			continue
		}

		// Process new post with sentiment analysis
		processedPost := c.sentimentSvc.ProcessPost(raw)
		if processedPost == nil {
			// No valid ticker found
			continue
		}

		// Store the post
		if err := c.storePost(processedPost); err != nil {
			log.Printf("Error storing post %s: %v", raw.ExternalID, err)
			continue
		}
		stored++
	}

	return processed, stored, updated, nil
}

func (c *Collector) storePost(processed *social.ProcessedPost) error {
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

	// Use existing database function from Phase 1
	return database.UpsertSocialPost(post)
}

func (c *Collector) updatePostMetrics(raw social.RawPost) error {
	query := `
		UPDATE social_posts
		SET upvotes = $1, comment_count = $2, award_count = $3, updated_at = NOW()
		WHERE external_post_id = $4
	`
	_, err := database.DB.Exec(query, raw.Upvotes, raw.CommentCount, raw.AwardCount, raw.ExternalID)
	return err
}

// GetStats returns statistics about the sentiment service
func (c *Collector) GetStats() map[string]int {
	return c.sentimentSvc.GetAnalyzerStats()
}

// RunForTicker runs collection for a specific ticker across all subreddits
func (c *Collector) RunForTicker(ctx context.Context, ticker string) error {
	log.Printf("Collecting posts for ticker: %s", ticker)

	var totalProcessed, totalStored int

	for _, source := range c.dataSources {
		if !source.IsEnabled() {
			continue
		}

		for _, subreddit := range c.subreddits {
			posts, err := source.FetchPosts(ctx, social.FetchOptions{
				Subreddit: subreddit,
				Ticker:    ticker,
				Limit:     50,
				Since:     time.Now().Add(-7 * 24 * time.Hour), // Last 7 days for ticker search
			})
			if err != nil {
				log.Printf("Error fetching %s posts for %s: %v", subreddit, ticker, err)
				continue
			}

			for _, raw := range posts {
				totalProcessed++

				if raw.Upvotes < c.minEngagement {
					continue
				}

				existingCount, _ := database.GetPostCountByExternalID(raw.ExternalID)
				if existingCount > 0 {
					continue
				}

				processedPost := c.sentimentSvc.ProcessPost(raw)
				if processedPost == nil {
					continue
				}

				if err := c.storePost(processedPost); err != nil {
					continue
				}
				totalStored++
			}
		}
	}

	log.Printf("Ticker collection complete for %s: processed=%d, stored=%d", ticker, totalProcessed, totalStored)
	return nil
}
