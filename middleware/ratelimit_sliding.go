package middleware

import (
	"sync"
	"time"

	"github.com/rarebek/wt"
)

// SlidingWindowRateLimit provides a sliding window rate limiter per IP.
// More accurate than fixed windows — prevents burst-at-boundary attacks.
type SlidingWindowRateLimit struct {
	mu      sync.Mutex
	windows map[string]*slidingWindow
	limit   int
	window  time.Duration
}

type slidingWindow struct {
	prevCount int
	currCount int
	currStart time.Time
}

// NewSlidingWindowRateLimit creates a sliding window limiter.
// limit: max requests per window. window: time window duration.
func NewSlidingWindowRateLimit(limit int, window time.Duration) *SlidingWindowRateLimit {
	return &SlidingWindowRateLimit{
		windows: make(map[string]*slidingWindow),
		limit:   limit,
		window:  window,
	}
}

// Middleware returns wt middleware.
func (s *SlidingWindowRateLimit) Middleware() wt.MiddlewareFunc {
	return func(c *wt.Context, next wt.HandlerFunc) {
		ip := c.RemoteAddr().String()
		if !s.allow(ip) {
			_ = c.CloseWithError(429, "rate limit exceeded")
			return
		}
		next(c)
	}
}

func (s *SlidingWindowRateLimit) allow(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	w, ok := s.windows[key]
	if !ok {
		s.windows[key] = &slidingWindow{currStart: now, currCount: 1}
		return true
	}

	elapsed := now.Sub(w.currStart)
	if elapsed >= s.window {
		// Move to next window
		w.prevCount = w.currCount
		w.currCount = 0
		w.currStart = now
	}

	// Weighted estimate: previous window's count * remaining fraction + current count
	fraction := 1.0 - (float64(elapsed) / float64(s.window))
	if fraction < 0 {
		fraction = 0
	}
	estimate := float64(w.prevCount)*fraction + float64(w.currCount)

	if int(estimate) >= s.limit {
		return false
	}

	w.currCount++
	return true
}
