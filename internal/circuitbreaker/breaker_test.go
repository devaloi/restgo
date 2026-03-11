package circuitbreaker

import (
	"errors"
	"testing"
	"time"
)

func TestBreakerClosedState(t *testing.T) {
	b := New(WithMaxFailures(3))

	// Success keeps breaker closed
	err := b.Execute(func() error { return nil })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if b.State() != "closed" {
		t.Errorf("state = %q, want closed", b.State())
	}
}

func TestBreakerOpensAfterFailures(t *testing.T) {
	b := New(WithMaxFailures(3), WithTimeout(time.Second))
	fail := errors.New("db error")

	for i := 0; i < 3; i++ {
		_ = b.Execute(func() error { return fail })
	}

	if b.State() != "open" {
		t.Errorf("state = %q, want open", b.State())
	}

	// Calls should be rejected immediately
	err := b.Execute(func() error { return nil })
	if !errors.Is(err, ErrCircuitOpen) {
		t.Errorf("err = %v, want ErrCircuitOpen", err)
	}
}

func TestBreakerHalfOpenRecovery(t *testing.T) {
	b := New(WithMaxFailures(2), WithTimeout(10*time.Millisecond), WithHalfOpenMax(1))
	fail := errors.New("db error")

	// Trip the breaker
	for i := 0; i < 2; i++ {
		_ = b.Execute(func() error { return fail })
	}
	if b.State() != "open" {
		t.Fatalf("state = %q, want open", b.State())
	}

	// Wait for timeout
	time.Sleep(20 * time.Millisecond)

	// Next call should go through (half-open) and succeed → close
	err := b.Execute(func() error { return nil })
	if err != nil {
		t.Fatalf("half-open call failed: %v", err)
	}
	if b.State() != "closed" {
		t.Errorf("state = %q, want closed", b.State())
	}
}

func TestBreakerHalfOpenFailure(t *testing.T) {
	b := New(WithMaxFailures(2), WithTimeout(10*time.Millisecond), WithHalfOpenMax(2))
	fail := errors.New("db error")

	// Trip the breaker
	for i := 0; i < 2; i++ {
		_ = b.Execute(func() error { return fail })
	}

	time.Sleep(20 * time.Millisecond)

	// Fail in half-open → re-open
	_ = b.Execute(func() error { return fail })
	if b.State() != "open" {
		t.Errorf("state = %q, want open after half-open failure", b.State())
	}
}

func TestBreakerResetsOnSuccess(t *testing.T) {
	b := New(WithMaxFailures(3))
	fail := errors.New("db error")

	// Two failures, then a success → resets count
	_ = b.Execute(func() error { return fail })
	_ = b.Execute(func() error { return fail })
	_ = b.Execute(func() error { return nil })

	if b.State() != "closed" {
		t.Errorf("state = %q, want closed", b.State())
	}

	// One more failure should NOT trip (count was reset)
	_ = b.Execute(func() error { return fail })
	if b.State() != "closed" {
		t.Errorf("state = %q, want closed after reset", b.State())
	}
}
