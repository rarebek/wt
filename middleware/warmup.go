package middleware

import (
	"sync/atomic"

	"github.com/rarebek/wt"
)

// Warmup tracks the server warmup period. During warmup, you might want
// to limit traffic or skip non-essential processing.
type Warmup struct {
	warmedUp atomic.Bool
	count    atomic.Int64
	target   int64
}

// NewWarmup creates a warmup tracker. The server is considered warmed up
// after `target` sessions have connected (JIT compilation, caches filled, etc).
func NewWarmup(target int) *Warmup {
	return &Warmup{target: int64(target)}
}

// Middleware tracks connections toward warmup.
func (w *Warmup) Middleware() wt.MiddlewareFunc {
	return func(c *wt.Context, next wt.HandlerFunc) {
		n := w.count.Add(1)
		if n >= w.target {
			w.warmedUp.Store(true)
		}
		c.Set("_warmed_up", w.warmedUp.Load())
		next(c)
	}
}

// IsWarmedUp returns whether the server has completed warmup.
func (w *Warmup) IsWarmedUp() bool {
	return w.warmedUp.Load()
}
