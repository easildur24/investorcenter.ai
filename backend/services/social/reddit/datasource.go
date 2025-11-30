package reddit

import (
	"context"
	"time"

	"investorcenter-api/services/social"
)

// DataSource implements social.SocialDataSource for Reddit
type DataSource struct {
	client  *Client
	enabled bool
}

// NewDataSource creates a Reddit data source
func NewDataSource(config Config) *DataSource {
	client := NewClient(config)
	return &DataSource{
		client:  client,
		enabled: client.IsConfigured(),
	}
}

// GetName returns the data source name
func (d *DataSource) GetName() string {
	return "reddit"
}

// IsEnabled returns true if the data source is configured and enabled
func (d *DataSource) IsEnabled() bool {
	return d.enabled
}

// GetRateLimits returns the rate limiting configuration
func (d *DataSource) GetRateLimits() social.RateLimitConfig {
	return social.RateLimitConfig{
		RequestsPerMinute: 90, // Conservative limit
		BurstLimit:        10,
	}
}

// FetchPosts implements SocialDataSource interface
func (d *DataSource) FetchPosts(ctx context.Context, opts social.FetchOptions) ([]social.RawPost, error) {
	if !d.enabled {
		return nil, nil
	}

	var rawPosts []RawRedditPost
	var err error

	subreddit := opts.Subreddit
	if subreddit == "" {
		subreddit = "wallstreetbets"
	}

	limit := opts.Limit
	if limit == 0 {
		limit = 100
	}

	if opts.Ticker != "" {
		// Search for specific ticker with $ prefix
		rawPosts, err = d.client.SearchPosts(ctx, "$"+opts.Ticker, subreddit, "new", limit)
	} else {
		// Get hot posts from subreddit
		rawPosts, err = d.client.FetchSubredditPosts(ctx, subreddit, "hot", limit)
	}

	if err != nil {
		return nil, err
	}

	// Convert to platform-agnostic format
	posts := make([]social.RawPost, 0, len(rawPosts))
	for _, rp := range rawPosts {
		postedAt := time.Unix(int64(rp.CreatedUTC), 0)

		// Filter by time if specified
		if !opts.Since.IsZero() && postedAt.Before(opts.Since) {
			continue
		}

		// Skip NSFW posts
		if rp.Over18 {
			continue
		}

		posts = append(posts, social.RawPost{
			ExternalID:   "t3_" + rp.ID, // Reddit fullname format
			Source:       "reddit",
			Community:    rp.Subreddit,
			Title:        rp.Title,
			Body:         rp.Selftext,
			URL:          "https://reddit.com" + rp.Permalink,
			Upvotes:      rp.Score,
			CommentCount: rp.NumComments,
			AwardCount:   rp.TotalAwards,
			Flair:        rp.Flair,
			PostedAt:     postedAt,
		})
	}

	return posts, nil
}

// SetEnabled allows enabling/disabling the data source
func (d *DataSource) SetEnabled(enabled bool) {
	d.enabled = enabled && d.client.IsConfigured()
}
