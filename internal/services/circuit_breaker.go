package services

import (
	"array-assessment/internal/models"
	"errors"
	"sync"
	"time"
)

var (
	ErrCircuitBreakerOpen = errors.New("circuit breaker is open")
)

type CircuitBreakerConfig struct {
	MaxFailures     int
	ResetTimeout    time.Duration
	HalfOpenMaxSucc int
}

func DefaultCircuitBreakerConfig() CircuitBreakerConfig {
	return CircuitBreakerConfig{
		MaxFailures:     5,
		ResetTimeout:    30 * time.Second,
		HalfOpenMaxSucc: 3,
	}
}

const (
	StateClosed models.CircuitBreakerState = iota
	StateOpen
	StateHalfOpen
)

type CircuitBreaker struct {
	mu                sync.RWMutex
	config            CircuitBreakerConfig
	state             models.CircuitBreakerState
	failures          int
	halfOpenSuccesses int
	lastFailureTime   time.Time
}

func NewCircuitBreaker(config CircuitBreakerConfig) CircuitBreakerInterface {
	return &CircuitBreaker{
		config:            config,
		state:             StateClosed,
		failures:          0,
		halfOpenSuccesses: 0,
	}
}

func (cb *CircuitBreaker) IsOpen() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.state == StateOpen && cb.shouldTransitionToHalfOpen() {
		cb.state = StateHalfOpen
		cb.halfOpenSuccesses = 0
		return false
	}

	return cb.state == StateOpen
}

func (cb *CircuitBreaker) shouldTransitionToHalfOpen() bool {
	return time.Since(cb.lastFailureTime) > cb.config.ResetTimeout
}

func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.state == StateHalfOpen {
		cb.halfOpenSuccesses++
		if cb.halfOpenSuccesses >= cb.config.HalfOpenMaxSucc {
			cb.transitionToClosed()
		}
	} else if cb.state == StateClosed {
		cb.failures = 0
	}
}

func (cb *CircuitBreaker) transitionToClosed() {
	cb.state = StateClosed
	cb.failures = 0
	cb.halfOpenSuccesses = 0
}

func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.lastFailureTime = time.Now()

	if cb.state == StateHalfOpen {
		cb.transitionToOpen()
	} else if cb.state == StateClosed {
		cb.failures++
		if cb.failures >= cb.config.MaxFailures {
			cb.transitionToOpen()
		}
	}
}

func (cb *CircuitBreaker) transitionToOpen() {
	cb.state = StateOpen
	cb.halfOpenSuccesses = 0
}

func (cb *CircuitBreaker) GetState() models.CircuitBreakerState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.state = StateClosed
	cb.failures = 0
	cb.halfOpenSuccesses = 0
}

func (cb *CircuitBreaker) GetFailureCount() int {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.failures
}
