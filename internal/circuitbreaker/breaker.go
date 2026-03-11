package circuitbreaker

import (
	"errors"
	"sync"
	"time"
)

// ErrCircuitOpen is returned when the circuit breaker is in the open state.
var ErrCircuitOpen = errors.New("circuit breaker is open")

type state int

const (
	stateClosed state = iota
	stateOpen
	stateHalfOpen
)

// Breaker implements a simple circuit breaker pattern.
type Breaker struct {
	mu           sync.Mutex
	state        state
	failCount    int
	successCount int
	lastFailure  time.Time

	maxFailures int
	timeout     time.Duration
	halfOpenMax int
}

// Option configures a Breaker.
type Option func(*Breaker)

// WithMaxFailures sets the number of consecutive failures before opening.
func WithMaxFailures(n int) Option {
	return func(b *Breaker) { b.maxFailures = n }
}

// WithTimeout sets how long the breaker stays open before moving to half-open.
func WithTimeout(d time.Duration) Option {
	return func(b *Breaker) { b.timeout = d }
}

// WithHalfOpenMax sets the number of successes needed to close from half-open.
func WithHalfOpenMax(n int) Option {
	return func(b *Breaker) { b.halfOpenMax = n }
}

// New creates a Breaker with the given options.
func New(opts ...Option) *Breaker {
	b := &Breaker{
		maxFailures: 5,
		timeout:     30 * time.Second,
		halfOpenMax: 2,
	}
	for _, o := range opts {
		o(b)
	}
	return b
}

// Execute runs fn through the circuit breaker. Returns ErrCircuitOpen if
// the circuit is open. Records success/failure to manage state transitions.
func (b *Breaker) Execute(fn func() error) error {
	b.mu.Lock()
	switch b.state {
	case stateOpen:
		if time.Since(b.lastFailure) > b.timeout {
			b.state = stateHalfOpen
			b.successCount = 0
		} else {
			b.mu.Unlock()
			return ErrCircuitOpen
		}
	}
	b.mu.Unlock()

	err := fn()

	b.mu.Lock()
	defer b.mu.Unlock()

	if err != nil {
		b.failCount++
		b.lastFailure = time.Now()
		if b.failCount >= b.maxFailures {
			b.state = stateOpen
		}
		return err
	}

	if b.state == stateHalfOpen {
		b.successCount++
		if b.successCount >= b.halfOpenMax {
			b.state = stateClosed
			b.failCount = 0
		}
	} else {
		b.failCount = 0
	}

	return nil
}

// State returns the current circuit state as a string.
func (b *Breaker) State() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	switch b.state {
	case stateOpen:
		return "open"
	case stateHalfOpen:
		return "half-open"
	default:
		return "closed"
	}
}
