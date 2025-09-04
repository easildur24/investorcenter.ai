package alphavantage

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// MetricsCollector defines the interface for collecting rate limiter metrics
type MetricsCollector interface {
	RecordRequest()
	RecordRateLimitHit(limitType string)
	RecordWaitTime(duration time.Duration)
}

// NopMetricsCollector is a no-op implementation of MetricsCollector
type NopMetricsCollector struct{}

func (n *NopMetricsCollector) RecordRequest()                      {}
func (n *NopMetricsCollector) RecordRateLimitHit(limitType string) {}
func (n *NopMetricsCollector) RecordWaitTime(duration time.Duration) {}

// RateLimiter implements rate limiting for API requests
type RateLimiter struct {
	mu              sync.Mutex
	requestsPerMin  int
	requestsPerDay  int
	minuteCounter   int
	dayCounter      int
	lastMinuteReset time.Time
	lastDayReset    time.Time
	logger          Logger
	metrics         MetricsCollector
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(perMinute, perDay int) *RateLimiter {
	return NewRateLimiterWithOptions(perMinute, perDay, nil, nil)
}

// NewRateLimiterWithOptions creates a new rate limiter with custom options
func NewRateLimiterWithOptions(perMinute, perDay int, logger Logger, metrics MetricsCollector) *RateLimiter {
	now := time.Now()
	if logger == nil {
		logger = &defaultLogger{}
	}
	if metrics == nil {
		metrics = &NopMetricsCollector{}
	}
	return &RateLimiter{
		requestsPerMin:  perMinute,
		requestsPerDay:  perDay,
		minuteCounter:   0,
		dayCounter:      0,
		lastMinuteReset: now,
		lastDayReset:    now.Truncate(24 * time.Hour),
		logger:          logger,
		metrics:         metrics,
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
			r.metrics.RecordRequest()
			r.logger.Debug("rate limit check passed", 
				"minute_used", r.minuteCounter, "minute_limit", r.requestsPerMin,
				"day_used", r.dayCounter, "day_limit", r.requestsPerDay)
			return nil
		}

		// Calculate wait time
		var waitDuration time.Duration

		if r.minuteCounter >= r.requestsPerMin {
			// Wait until next minute
			waitDuration = time.Minute - now.Sub(r.lastMinuteReset)
			r.metrics.RecordRateLimitHit("minute")
			r.logger.Info("minute rate limit reached", "wait_duration", waitDuration)
		} else if r.dayCounter >= r.requestsPerDay {
			// Wait until next day
			waitDuration = 24*time.Hour - now.Sub(r.lastDayReset)
			r.metrics.RecordRateLimitHit("day")
			r.logger.Warn("daily rate limit exceeded", "wait_duration", waitDuration)
			return fmt.Errorf("daily rate limit exceeded, retry after %v", waitDuration)
		}

		// Wait with context
		timer := time.NewTimer(waitDuration)
		r.metrics.RecordWaitTime(waitDuration)
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

// GetRemainingQuota returns the remaining requests allowed without blocking
func (r *RateLimiter) GetRemainingQuota() (minuteRemaining, dayRemaining int) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Reset counters if needed
	now := time.Now()
	if now.Sub(r.lastMinuteReset) >= time.Minute {
		r.minuteCounter = 0
		r.lastMinuteReset = now
	}
	if now.Sub(r.lastDayReset) >= 24*time.Hour {
		r.dayCounter = 0
		r.lastDayReset = now.Truncate(24 * time.Hour)
	}

	minuteRemaining = r.requestsPerMin - r.minuteCounter
	dayRemaining = r.requestsPerDay - r.dayCounter
	
	if minuteRemaining < 0 {
		minuteRemaining = 0
	}
	if dayRemaining < 0 {
		dayRemaining = 0
	}
	
	return minuteRemaining, dayRemaining
}

// CanMakeRequest checks if a request can be made without blocking
func (r *RateLimiter) CanMakeRequest() bool {
	minuteRemaining, dayRemaining := r.GetRemainingQuota()
	return minuteRemaining > 0 && dayRemaining > 0
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
