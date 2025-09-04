package alphavantage

import (
	"errors"
	"sync"
	"time"
)

// CircuitState represents the state of the circuit breaker
type CircuitState int

const (
	// StateClosed allows requests to pass through
	StateClosed CircuitState = iota
	// StateOpen blocks all requests
	StateOpen
	// StateHalfOpen allows a limited number of requests for testing
	StateHalfOpen
)

// CircuitBreaker protects against cascading failures
type CircuitBreaker struct {
	mu              sync.RWMutex
	state           CircuitState
	failureCount    int
	successCount    int
	lastFailureTime time.Time
	
	// Configuration
	maxFailures      int
	resetTimeout     time.Duration
	halfOpenRequests int
	
	logger Logger
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(maxFailures int, resetTimeout time.Duration, logger Logger) *CircuitBreaker {
	if logger == nil {
		logger = &defaultLogger{}
	}
	return &CircuitBreaker{
		state:            StateClosed,
		maxFailures:      maxFailures,
		resetTimeout:     resetTimeout,
		halfOpenRequests: 1,
		logger:           logger,
	}
}

// Allow checks if a request is allowed to proceed
func (cb *CircuitBreaker) Allow() error {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	
	now := time.Now()
	
	switch cb.state {
	case StateClosed:
		return nil
		
	case StateOpen:
		// Check if we should transition to half-open
		if now.Sub(cb.lastFailureTime) > cb.resetTimeout {
			cb.state = StateHalfOpen
			cb.successCount = 0
			cb.failureCount = 0
			cb.logger.Info("circuit breaker transitioning to half-open")
			return nil
		}
		return errors.New("circuit breaker is open")
		
	case StateHalfOpen:
		if cb.successCount+cb.failureCount < cb.halfOpenRequests {
			return nil
		}
		return errors.New("circuit breaker is half-open, limited requests only")
		
	default:
		return nil
	}
}

// RecordSuccess records a successful request
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	
	switch cb.state {
	case StateHalfOpen:
		cb.successCount++
		if cb.successCount >= cb.halfOpenRequests {
			cb.state = StateClosed
			cb.failureCount = 0
			cb.logger.Info("circuit breaker closed after successful recovery")
		}
		
	case StateClosed:
		// Reset failure count on success
		cb.failureCount = 0
	}
}

// RecordFailure records a failed request
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	
	cb.failureCount++
	cb.lastFailureTime = time.Now()
	
	switch cb.state {
	case StateClosed:
		if cb.failureCount >= cb.maxFailures {
			cb.state = StateOpen
			cb.logger.Warn("circuit breaker opened", "failures", cb.failureCount)
		}
		
	case StateHalfOpen:
		cb.state = StateOpen
		cb.logger.Warn("circuit breaker reopened from half-open state")
	}
}

// GetState returns the current state of the circuit breaker
func (cb *CircuitBreaker) GetState() CircuitState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// Reset manually resets the circuit breaker
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	
	cb.state = StateClosed
	cb.failureCount = 0
	cb.successCount = 0
	cb.logger.Info("circuit breaker manually reset")
}