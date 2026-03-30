package middleware

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/rarebek/wt"
)

// Metrics tracks server-level metrics for monitoring.
type Metrics struct {
	ActiveSessions  atomic.Int64
	TotalSessions   atomic.Int64
	TotalDatagrams  atomic.Int64
	SessionDurations sync.Map // sessionID -> start time
}

// NewMetrics creates a new Metrics instance.
func NewMetrics() *Metrics {
	return &Metrics{}
}

// Snapshot returns a point-in-time copy of the metrics.
type MetricsSnapshot struct {
	ActiveSessions int64
	TotalSessions  int64
	TotalDatagrams int64
}

// Snapshot returns current metrics values.
func (m *Metrics) Snapshot() MetricsSnapshot {
	return MetricsSnapshot{
		ActiveSessions: m.ActiveSessions.Load(),
		TotalSessions:  m.TotalSessions.Load(),
		TotalDatagrams: m.TotalDatagrams.Load(),
	}
}

// Middleware returns a middleware that tracks session metrics.
func (m *Metrics) Middleware() wt.MiddlewareFunc {
	return func(c *wt.Context, next wt.HandlerFunc) {
		m.ActiveSessions.Add(1)
		m.TotalSessions.Add(1)
		m.SessionDurations.Store(c.ID(), time.Now())

		defer func() {
			m.ActiveSessions.Add(-1)
			m.SessionDurations.Delete(c.ID())
		}()

		next(c)
	}
}

// SessionDuration returns how long a session has been active.
func (m *Metrics) SessionDuration(sessionID string) time.Duration {
	v, ok := m.SessionDurations.Load(sessionID)
	if !ok {
		return 0
	}
	return time.Since(v.(time.Time))
}
