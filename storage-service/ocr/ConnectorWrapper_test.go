package ocr

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

// TestConnectorWrapper_RetrySucceedsAfterFailures verifies that Do retries
// the function and returns nil once it succeeds within the attempt budget.
func TestConnectorWrapper_RetrySucceedsAfterFailures(t *testing.T) {
	w := NewConnectorWrapper(100, 3)

	var calls atomic.Int32
	sentinel := errors.New("transient error")

	err := w.Do(context.Background(), func() error {
		if calls.Add(1) < 3 {
			return sentinel
		}
		return nil
	})

	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if calls.Load() != 3 {
		t.Fatalf("expected 3 calls, got %d", calls.Load())
	}
}

// TestConnectorWrapper_RetryExhausted verifies that Do returns an error
// after all attempts are used up.
func TestConnectorWrapper_RetryExhausted(t *testing.T) {
	w := NewConnectorWrapper(100, 3)

	sentinel := errors.New("permanent error")
	var calls atomic.Int32

	err := w.Do(context.Background(), func() error {
		calls.Add(1)
		return sentinel
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if calls.Load() != 3 {
		t.Fatalf("expected 3 calls, got %d", calls.Load())
	}
}

// TestConnectorWrapper_RateLimiter_BlocksAndRespectsContext verifies that once
// the burst is consumed, the next call blocks waiting for a token and unblocks
// when the context is cancelled, without calling fn at all.
func TestConnectorWrapper_RateLimiter_BlocksAndRespectsContext(t *testing.T) {
	// 0.001 req/sec → next token arrives in ~1000s; burst=1 is consumed by first call.
	w := NewConnectorWrapper(0.001, 3)

	if err := w.Do(context.Background(), func() error { return nil }); err != nil {
		t.Fatalf("first call failed: %v", err)
	}

	// Cancel via goroutine so the limiter actually blocks (no deadline to fast-fail on).
	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(50*time.Millisecond, cancel)

	var calls atomic.Int32
	err := w.Do(ctx, func() error {
		calls.Add(1)
		return nil
	})

	if err == nil {
		t.Fatal("expected context error, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected Canceled, got %v", err)
	}
	if calls.Load() != 0 {
		t.Fatalf("fn should not have been called, got %d calls", calls.Load())
	}
}

// TestConnectorWrapper_RateLimiter_CancelledContextNotRetried verifies that a
// context error from the rate limiter is treated as unrecoverable and not retried.
func TestConnectorWrapper_RateLimiter_CancelledContextNotRetried(t *testing.T) {
	w := NewConnectorWrapper(100, 5)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var calls atomic.Int32
	err := w.Do(ctx, func() error {
		calls.Add(1)
		return nil
	})

	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
	if calls.Load() != 0 {
		t.Fatalf("fn should not have been called, got %d calls", calls.Load())
	}
}
