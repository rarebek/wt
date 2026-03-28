package wt

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestRetrySuccess(t *testing.T) {
	cfg := DefaultRetryConfig()

	attempts := 0
	err := Retry(context.Background(), cfg, func() error {
		attempts++
		return nil
	})

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if attempts != 1 {
		t.Errorf("expected 1 attempt, got %d", attempts)
	}
}

func TestRetryEventualSuccess(t *testing.T) {
	cfg := RetryConfig{
		MaxAttempts: 5,
		InitDelay:   1 * time.Millisecond,
		MaxDelay:    10 * time.Millisecond,
	}

	attempts := 0
	err := Retry(context.Background(), cfg, func() error {
		attempts++
		if attempts < 3 {
			return fmt.Errorf("not yet")
		}
		return nil
	})

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
}

func TestRetryAllFail(t *testing.T) {
	cfg := RetryConfig{
		MaxAttempts: 3,
		InitDelay:   1 * time.Millisecond,
		MaxDelay:    10 * time.Millisecond,
	}

	err := Retry(context.Background(), cfg, func() error {
		return fmt.Errorf("always fails")
	})

	if err == nil {
		t.Error("expected error")
	}
}

func TestRetryContextCancelled(t *testing.T) {
	cfg := RetryConfig{
		MaxAttempts: 100,
		InitDelay:   100 * time.Millisecond,
		MaxDelay:    1 * time.Second,
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := Retry(ctx, cfg, func() error {
		return fmt.Errorf("fail")
	})

	if err != context.Canceled {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

func TestDefaultRetryConfig(t *testing.T) {
	cfg := DefaultRetryConfig()

	if cfg.MaxAttempts != 3 {
		t.Errorf("expected 3 max attempts, got %d", cfg.MaxAttempts)
	}
	if !cfg.Jitter {
		t.Error("expected jitter to be true")
	}
}
