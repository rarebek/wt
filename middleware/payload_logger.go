package middleware

import (
	"log/slog"
	"sync/atomic"

	"github.com/rarebek/wt"
)

// PayloadStats tracks cumulative message/datagram sizes.
type PayloadStats struct {
	TotalMessages  atomic.Int64
	TotalBytes     atomic.Int64
	LargestMessage atomic.Int64
}

// NewPayloadStats creates a new payload tracker.
func NewPayloadStats() *PayloadStats {
	return &PayloadStats{}
}

// Record records a message.
func (ps *PayloadStats) Record(size int) {
	ps.TotalMessages.Add(1)
	ps.TotalBytes.Add(int64(size))
	for {
		old := ps.LargestMessage.Load()
		if int64(size) <= old || ps.LargestMessage.CompareAndSwap(old, int64(size)) {
			break
		}
	}
}

// Middleware logs payload sizes on each session.
func (ps *PayloadStats) Middleware(logger *slog.Logger) wt.MiddlewareFunc {
	if logger == nil {
		logger = slog.Default()
	}
	return func(c *wt.Context, next wt.HandlerFunc) {
		c.Set("_payload_stats", ps)
		next(c)
	}
}

// GetPayloadStats retrieves stats from context.
func GetPayloadStats(c *wt.Context) *PayloadStats {
	v, ok := c.Get("_payload_stats")
	if !ok {
		return nil
	}
	ps, _ := v.(*PayloadStats)
	return ps
}
