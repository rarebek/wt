package middleware

import (
	"sync/atomic"

	"github.com/rarebek/wt"
)

// BandwidthTracker tracks bytes sent and received per session.
// Access the tracker via GetBandwidthTracker(c) within handlers.
type BandwidthTracker struct {
	BytesSent     atomic.Int64
	BytesReceived atomic.Int64
}

// Bandwidth returns middleware that installs a bandwidth tracker on each session.
func Bandwidth() wt.MiddlewareFunc {
	return func(c *wt.Context, next wt.HandlerFunc) {
		tracker := &BandwidthTracker{}
		c.Set("_bandwidth", tracker)
		next(c)
	}
}

// GetBandwidthTracker retrieves the bandwidth tracker from the context.
func GetBandwidthTracker(c *wt.Context) *BandwidthTracker {
	v, ok := c.Get("_bandwidth")
	if !ok {
		return nil
	}
	bt, _ := v.(*BandwidthTracker)
	return bt
}

// Stats returns current bandwidth stats.
func (bt *BandwidthTracker) Stats() (sent, received int64) {
	return bt.BytesSent.Load(), bt.BytesReceived.Load()
}

// RecordSent records bytes sent.
func (bt *BandwidthTracker) RecordSent(n int64) {
	bt.BytesSent.Add(n)
}

// RecordReceived records bytes received.
func (bt *BandwidthTracker) RecordReceived(n int64) {
	bt.BytesReceived.Add(n)
}
