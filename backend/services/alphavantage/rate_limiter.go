package alphavantage

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// RateLimiter implements rate limiting for API requests
type RateLimiter struct {
	mu              sync.Mutex
	requestsPerMin  int
	requestsPerDay  int
	minuteCounter   int
	dayCounter      int
	lastMinuteReset time.Time
	lastDayReset    time.Time
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(perMinute, perDay int) *RateLimiter {
	now := time.Now()
	return &RateLimiter{
		requestsPerMin:  perMinute,
		requestsPerDay:  perDay,
		minuteCounter:   0,
		dayCounter:      0,
		lastMinuteReset: now,
		lastDayReset:    now.Truncate(24 * time.Hour),
	}
}

// Wait blocks until a request can be made without exceeding rate limits
func (r *RateLimiter) Wait(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		now := time.Now()

		// Reset minute counter if needed
		if now.Sub(r.lastMinuteReset) >= time.Minute {
			r.minuteCounter = 0
			r.lastMinuteReset = now
		}

		// Reset day counter if needed
		if now.Sub(r.lastDayReset) >= 24*time.Hour {
			r.dayCounter = 0
			r.lastDayReset = now.Truncate(24 * time.Hour)
		}

		// Check if we can make a request
		if r.minuteCounter < r.requestsPerMin && r.dayCounter < r.requestsPerDay {
			r.minuteCounter++
			r.dayCounter++
			return nil
		}

		// Calculate wait time
		var waitDuration time.Duration

		if r.minuteCounter >= r.requestsPerMin {
			// Wait until next minute
			waitDuration = time.Minute - now.Sub(r.lastMinuteReset)
		} else if r.dayCounter >= r.requestsPerDay {
			// Wait until next day
			waitDuration = 24*time.Hour - now.Sub(r.lastDayReset)
			return fmt.Errorf("daily rate limit exceeded, retry after %v", waitDuration)
		}

		// Wait with context
		timer := time.NewTimer(waitDuration)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
			// Continue to next iteration
		}
	}
}

// GetStatus returns current rate limit status
func (r *RateLimiter) GetStatus() (minuteUsed, dayUsed, minuteLimit, dayLimit int) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Reset counters if needed
	now := time.Now()
	if now.Sub(r.lastMinuteReset) >= time.Minute {
		r.minuteCounter = 0
	}
	if now.Sub(r.lastDayReset) >= 24*time.Hour {
		r.dayCounter = 0
	}

	return r.minuteCounter, r.dayCounter, r.requestsPerMin, r.requestsPerDay
}

// Reset manually resets the rate limiter counters
func (r *RateLimiter) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	r.minuteCounter = 0
	r.dayCounter = 0
	r.lastMinuteReset = now
	r.lastDayReset = now.Truncate(24 * time.Hour)
}
