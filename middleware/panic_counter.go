package middleware

import (
	"log/slog"
	"sync/atomic"

	"github.com/rarebek/wt"
)

// PanicCounter recovers from panics AND counts them.
// Unlike Recover, this tracks the total panic count for alerting.
type PanicCounter struct {
	Total atomic.Int64
}

// NewPanicCounter creates a panic counter.
func NewPanicCounter() *PanicCounter {
	return &PanicCounter{}
}

// Middleware returns wt middleware that recovers and counts panics.
func (pc *PanicCounter) Middleware(logger *slog.Logger) wt.MiddlewareFunc {
	if logger == nil {
		logger = slog.Default()
	}
	return func(c *wt.Context, next wt.HandlerFunc) {
		defer func() {
			if r := recover(); r != nil {
				pc.Total.Add(1)
				logger.Error("panic recovered",
					"session", c.ID(),
					"panic", r,
					"total_panics", pc.Total.Load(),
				)
				_ = c.CloseWithError(500, "internal error")
			}
		}()
		next(c)
	}
}

// Count returns total panics recovered.
func (pc *PanicCounter) Count() int64 {
	return pc.Total.Load()
}
