package middleware

import (
	"sync"
	"time"

	"github.com/rarebek/wt"
)

// RateLimit returns middleware that limits the number of concurrent sessions per IP.
func RateLimit(maxPerIP int) wt.MiddlewareFunc {
	var mu sync.Mutex
	counts := make(map[string]int)

	return func(c *wt.Context, next wt.HandlerFunc) {
		ip := c.RemoteAddr().String()

		mu.Lock()
		if counts[ip] >= maxPerIP {
			mu.Unlock()
			_ = c.CloseWithError(429, "too many connections")
			return
		}
		counts[ip]++
		mu.Unlock()

		defer func() {
			mu.Lock()
			counts[ip]--
			if counts[ip] <= 0 {
				delete(counts, ip)
			}
			mu.Unlock()
		}()

		next(c)
	}
}

// TokenBucket returns middleware that rate-limits datagrams/messages using a token bucket.
// This middleware stores a rate limiter in the context under the key "_ratelimiter".
func TokenBucket(rate float64, burst int) wt.MiddlewareFunc {
	return func(c *wt.Context, next wt.HandlerFunc) {
		limiter := &tokenBucket{
			rate:   rate,
			burst:  burst,
			tokens: float64(burst),
			last:   time.Now(),
		}
		c.Set("_ratelimiter", limiter)
		next(c)
	}
}

type tokenBucket struct {
	mu     sync.Mutex
	rate   float64
	burst  int
	tokens float64
	last   time.Time
}

// Allow checks if a token is available, consuming one if so.
func (tb *tokenBucket) Allow() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(tb.last).Seconds()
	tb.last = now

	tb.tokens += elapsed * tb.rate
	if tb.tokens > float64(tb.burst) {
		tb.tokens = float64(tb.burst)
	}

	if tb.tokens < 1 {
		return false
	}
	tb.tokens--
	return true
}

// GetRateLimiter retrieves the token bucket rate limiter from the context.
// Returns nil if TokenBucket middleware wasn't applied.
func GetRateLimiter(c *wt.Context) *tokenBucket {
	v, ok := c.Get("_ratelimiter")
	if !ok {
		return nil
	}
	tb, _ := v.(*tokenBucket)
	return tb
}
