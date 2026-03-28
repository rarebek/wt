package wt

import (
	"context"
	"math/rand/v2"
	"time"
)

// RetryConfig configures retry behavior for stream operations.
type RetryConfig struct {
	MaxAttempts int
	InitDelay   time.Duration
	MaxDelay    time.Duration
	Jitter      bool // Add random jitter to delays
}

// DefaultRetryConfig returns sensible retry defaults.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts: 3,
		InitDelay:   100 * time.Millisecond,
		MaxDelay:    5 * time.Second,
		Jitter:      true,
	}
}

// Retry executes fn up to MaxAttempts times with exponential backoff.
// Returns the last error if all attempts fail.
func Retry(ctx context.Context, cfg RetryConfig, fn func() error) error {
	var lastErr error
	delay := cfg.InitDelay

	for attempt := range cfg.MaxAttempts {
		if err := fn(); err == nil {
			return nil
		} else {
			lastErr = err
		}

		if attempt == cfg.MaxAttempts-1 {
			break
		}

		// Calculate backoff with optional jitter
		wait := delay
		if cfg.Jitter {
			jitter := time.Duration(rand.Int64N(int64(delay) / 2))
			wait = delay + jitter
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(wait):
		}

		delay = min(delay*2, cfg.MaxDelay)
	}

	return lastErr
}

// RetryStream attempts to open a stream with retry.
func RetryStream(ctx context.Context, c *Context, cfg RetryConfig) (*Stream, error) {
	var stream *Stream
	err := Retry(ctx, cfg, func() error {
		s, err := c.OpenStream()
		if err != nil {
			return err
		}
		stream = s
		return nil
	})
	return stream, err
}
