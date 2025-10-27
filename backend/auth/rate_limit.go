package auth

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// Simple in-memory rate limiter (use Redis in production for distributed systems)
type rateLimiter struct {
	mu       sync.RWMutex
	attempts map[string][]time.Time
	max      int
	window   time.Duration
}

var loginLimiter = &rateLimiter{
	attempts: make(map[string][]time.Time),
	max:      5,                // Max 5 attempts
	window:   15 * time.Minute, // Per 15 minutes
}

// RateLimitMiddleware limits requests by IP address
func RateLimitMiddleware(limiter *rateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()

		if !limiter.Allow(ip) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "Too many requests. Please try again later.",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// Allow checks if request from IP is allowed
func (rl *rateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-rl.window)

	// Get attempts for this key
	attempts, exists := rl.attempts[key]
	if !exists {
		rl.attempts[key] = []time.Time{now}
		return true
	}

	// Remove expired attempts
	validAttempts := []time.Time{}
	for _, t := range attempts {
		if t.After(cutoff) {
			validAttempts = append(validAttempts, t)
		}
	}

	// Check if under limit
	if len(validAttempts) >= rl.max {
		rl.attempts[key] = validAttempts
		return false
	}

	// Add new attempt
	validAttempts = append(validAttempts, now)
	rl.attempts[key] = validAttempts
	return true
}

// Cleanup removes old entries (call periodically)
func (rl *rateLimiter) Cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-rl.window)

	for key, attempts := range rl.attempts {
		validAttempts := []time.Time{}
		for _, t := range attempts {
			if t.After(cutoff) {
				validAttempts = append(validAttempts, t)
			}
		}

		if len(validAttempts) == 0 {
			delete(rl.attempts, key)
		} else {
			rl.attempts[key] = validAttempts
		}
	}
}

// StartRateLimiterCleanup starts periodic cleanup
func StartRateLimiterCleanup(limiter *rateLimiter) {
	ticker := time.NewTicker(5 * time.Minute)
	go func() {
		for range ticker.C {
			limiter.Cleanup()
		}
	}()
}

// GetLoginLimiter returns the login limiter instance
func GetLoginLimiter() *rateLimiter {
	return loginLimiter
}
