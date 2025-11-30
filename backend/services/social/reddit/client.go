package reddit

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	AuthURL    = "https://www.reddit.com/api/v1/access_token"
	BaseAPIURL = "https://oauth.reddit.com"
	UserAgent  = "investorCenter.ai/1.0 (contact: support@investorcenter.ai)"
)

// Config holds Reddit API credentials
type Config struct {
	ClientID     string
	ClientSecret string
}

// Client handles Reddit API communication
type Client struct {
	config      Config
	httpClient  *http.Client
	accessToken string
	tokenExpiry time.Time
	mu          sync.RWMutex

	// Rate limiting (Reddit allows 100 req/min for OAuth)
	lastRequest time.Time
	minInterval time.Duration
}

// NewClient creates a new Reddit API client
func NewClient(config Config) *Client {
	return &Client{
		config:      config,
		httpClient:  &http.Client{Timeout: 30 * time.Second},
		minInterval: 650 * time.Millisecond, // ~92 req/min to stay safe
	}
}

// Authenticate gets an access token using client credentials
func (c *Client) Authenticate(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if token is still valid (with 1 minute buffer)
	if c.accessToken != "" && time.Now().Before(c.tokenExpiry.Add(-1*time.Minute)) {
		return nil
	}

	data := url.Values{}
	data.Set("grant_type", "client_credentials")

	req, err := http.NewRequestWithContext(ctx, "POST", AuthURL, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("creating auth request: %w", err)
	}

	req.SetBasicAuth(c.config.ClientID, c.config.ClientSecret)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", UserAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("executing auth request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("auth failed: %d - %s", resp.StatusCode, string(body))
	}

	var authResp struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
		TokenType   string `json:"token_type"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return fmt.Errorf("decoding auth response: %w", err)
	}

	c.accessToken = authResp.AccessToken
	c.tokenExpiry = time.Now().Add(time.Duration(authResp.ExpiresIn) * time.Second)

	return nil
}

// RawRedditPost represents the Reddit API response structure
// NOTE: "author" field intentionally omitted for privacy
type RawRedditPost struct {
	ID          string  `json:"id"`
	Title       string  `json:"title"`
	Selftext    string  `json:"selftext"`
	Subreddit   string  `json:"subreddit"`
	Score       int     `json:"score"`
	NumComments int     `json:"num_comments"`
	TotalAwards int     `json:"total_awards_received"`
	URL         string  `json:"url"`
	Permalink   string  `json:"permalink"`
	Flair       string  `json:"link_flair_text"`
	CreatedUTC  float64 `json:"created_utc"`
	IsSelf      bool    `json:"is_self"`
	Over18      bool    `json:"over_18"`
	Stickied    bool    `json:"stickied"`
}

// FetchSubredditPosts gets posts from a subreddit
// sort options: "hot", "new", "top", "rising"
func (c *Client) FetchSubredditPosts(ctx context.Context, subreddit, sort string, limit int) ([]RawRedditPost, error) {
	if err := c.Authenticate(ctx); err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}

	c.rateLimit()

	if limit > 100 {
		limit = 100 // Reddit max
	}

	endpoint := fmt.Sprintf("%s/r/%s/%s.json?limit=%d&raw_json=1",
		BaseAPIURL, subreddit, sort, limit)

	return c.fetchListing(ctx, endpoint)
}

// SearchPosts searches for posts containing a query (e.g., ticker symbol)
func (c *Client) SearchPosts(ctx context.Context, query, subreddit string, sort string, limit int) ([]RawRedditPost, error) {
	if err := c.Authenticate(ctx); err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}

	c.rateLimit()

	if limit > 100 {
		limit = 100
	}

	endpoint := fmt.Sprintf("%s/r/%s/search.json?q=%s&restrict_sr=on&sort=%s&limit=%d&raw_json=1",
		BaseAPIURL, subreddit, url.QueryEscape(query), sort, limit)

	return c.fetchListing(ctx, endpoint)
}

func (c *Client) fetchListing(ctx context.Context, endpoint string) ([]RawRedditPost, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	c.mu.RLock()
	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	c.mu.RUnlock()
	req.Header.Set("User-Agent", UserAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, fmt.Errorf("rate limited by Reddit")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("request failed: %d - %s", resp.StatusCode, string(body))
	}

	var listing struct {
		Data struct {
			Children []struct {
				Kind string        `json:"kind"`
				Data RawRedditPost `json:"data"`
			} `json:"children"`
			After string `json:"after"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&listing); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	posts := make([]RawRedditPost, 0, len(listing.Data.Children))
	for _, child := range listing.Data.Children {
		// Skip non-post items and stickied posts
		if child.Kind != "t3" || child.Data.Stickied {
			continue
		}
		posts = append(posts, child.Data)
	}

	return posts, nil
}

func (c *Client) rateLimit() {
	c.mu.Lock()
	defer c.mu.Unlock()

	elapsed := time.Since(c.lastRequest)
	if elapsed < c.minInterval {
		time.Sleep(c.minInterval - elapsed)
	}
	c.lastRequest = time.Now()
}

// IsConfigured returns true if credentials are set
func (c *Client) IsConfigured() bool {
	return c.config.ClientID != "" && c.config.ClientSecret != ""
}
