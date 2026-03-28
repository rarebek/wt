package wt

import (
	"net"
	"sync"
	"time"
)

// MigrationEvent represents a connection migration (IP address change).
type MigrationEvent struct {
	SessionID  string
	OldAddr    net.Addr
	NewAddr    net.Addr
	MigratedAt time.Time
}

// MigrationWatcher monitors sessions for address changes.
// QUIC handles migration transparently, but this watcher lets
// you react to migrations (logging, analytics, security checks).
type MigrationWatcher struct {
	mu        sync.Mutex
	sessions  map[string]string // sessionID -> last known remote addr
	onMigrate func(MigrationEvent)
	interval  time.Duration
	done      chan struct{}
}

// NewMigrationWatcher creates a watcher that polls session addresses.
func NewMigrationWatcher(store *SessionStore, onMigrate func(MigrationEvent)) *MigrationWatcher {
	mw := &MigrationWatcher{
		sessions:  make(map[string]string),
		onMigrate: onMigrate,
		interval:  5 * time.Second,
		done:      make(chan struct{}),
	}
	go mw.watchLoop(store)
	return mw
}

func (mw *MigrationWatcher) watchLoop(store *SessionStore) {
	ticker := time.NewTicker(mw.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			store.Each(func(c *Context) {
				addr := c.RemoteAddr().String()
				mw.mu.Lock()
				old, exists := mw.sessions[c.ID()]
				if exists && old != addr {
					// Migration detected!
					event := MigrationEvent{
						SessionID:  c.ID(),
						OldAddr:    parseAddr(old),
						NewAddr:    c.RemoteAddr(),
						MigratedAt: time.Now(),
					}
					mw.mu.Unlock()
					if mw.onMigrate != nil {
						mw.onMigrate(event)
					}
					mw.mu.Lock()
				}
				mw.sessions[c.ID()] = addr
				mw.mu.Unlock()
			})
		case <-mw.done:
			return
		}
	}
}

// Stop stops the migration watcher.
func (mw *MigrationWatcher) Stop() {
	select {
	case <-mw.done:
	default:
		close(mw.done)
	}
}

type simpleAddr struct{ s string }

func (a simpleAddr) Network() string { return "udp" }
func (a simpleAddr) String() string  { return a.s }

func parseAddr(s string) net.Addr {
	return simpleAddr{s}
}
