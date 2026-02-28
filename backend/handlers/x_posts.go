package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

// XPost represents a single X/Twitter post for API response
type XPost struct {
	AuthorHandle   *string `json:"author_handle"`
	AuthorName     *string `json:"author_name"`
	AuthorVerified *bool   `json:"author_verified"`
	Content        string  `json:"content"`
	Timestamp      *string `json:"timestamp"`
	Likes          *int    `json:"likes"`
	Reposts        *int    `json:"reposts"`
	Replies        *int    `json:"replies"`
	Views          *int    `json:"views"`
	Bookmarks      *int    `json:"bookmarks"`
	PostURL        *string `json:"post_url"`
	HasMedia       *bool   `json:"has_media"`
	IsRepost       *bool   `json:"is_repost"`
	IsReply        *bool   `json:"is_reply"`
}

// XPostsResponse is the Redis-stored structure
type XPostsResponse struct {
	Ticker    string        `json:"ticker"`
	UpdatedAt string        `json:"updated_at"`
	Posts     []interface{} `json:"posts"`
}

// GetXPosts returns the latest X/Twitter posts for a ticker from Redis.
// GET /api/v1/tickers/:symbol/x-posts
func GetXPosts(c *gin.Context) {
	symbol := strings.ToUpper(c.Param("symbol"))
	if symbol == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Symbol is required"})
		return
	}

	ctx := context.Background()
	redisKey := fmt.Sprintf("x:posts:%s", symbol)

	data, err := redisClient.Get(ctx, redisKey).Result()
	if err == redis.Nil {
		// No posts cached — return empty
		c.JSON(http.StatusOK, gin.H{
			"ticker":     symbol,
			"posts":      []interface{}{},
			"updated_at": nil,
		})
		return
	}
	if err != nil {
		log.Printf("Redis error reading x:posts:%s: %v", symbol, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch posts"})
		return
	}

	// Parse and return the cached data directly
	var cached map[string]interface{}
	if err := json.Unmarshal([]byte(data), &cached); err != nil {
		log.Printf("Failed to parse cached X posts for %s: %v", symbol, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse cached posts"})
		return
	}

	c.JSON(http.StatusOK, cached)
}
