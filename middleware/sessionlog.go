package middleware

import (
	"sync"
	"time"

	"github.com/rarebek/wt"
)

// SessionLogEntry records a single event in a session's lifetime.
type SessionLogEntry struct {
	Time    time.Time `json:"time"`
	Event   string    `json:"event"`
	Details string    `json:"details,omitempty"`
}

// SessionLog collects structured events for a session.
// Access via GetSessionLog(c) within handlers.
type SessionLog struct {
	mu      sync.Mutex
	entries []SessionLogEntry
}

// Add appends an event to the log.
func (sl *SessionLog) Add(event, details string) {
	sl.mu.Lock()
	sl.entries = append(sl.entries, SessionLogEntry{
		Time:    time.Now(),
		Event:   event,
		Details: details,
	})
	sl.mu.Unlock()
}

// Entries returns all log entries (copy).
func (sl *SessionLog) Entries() []SessionLogEntry {
	sl.mu.Lock()
	defer sl.mu.Unlock()
	out := make([]SessionLogEntry, len(sl.entries))
	copy(out, sl.entries)
	return out
}

// Len returns the number of entries.
func (sl *SessionLog) Len() int {
	sl.mu.Lock()
	n := len(sl.entries)
	sl.mu.Unlock()
	return n
}

// SessionLogger returns middleware that creates a per-session event log.
func SessionLogger() wt.MiddlewareFunc {
	return func(c *wt.Context, next wt.HandlerFunc) {
		log := &SessionLog{}
		log.Add("connect", c.RemoteAddr().String())
		c.Set("_session_log", log)
		next(c)
		log.Add("disconnect", "")
	}
}

// GetSessionLog retrieves the session log from context.
func GetSessionLog(c *wt.Context) *SessionLog {
	v, ok := c.Get("_session_log")
	if !ok {
		return nil
	}
	sl, _ := v.(*SessionLog)
	return sl
}
