package wt

import (
	"time"
)

// KeepAlive sends periodic datagrams to keep the QUIC connection alive.
// This prevents NAT mappings from expiring (typically 20-30 seconds for UDP).
//
// Usage:
//
//	server.Handle("/app", func(c *wt.Context) {
//	    stop := wt.KeepAlive(c, 15*time.Second)
//	    defer stop()
//	    // ... handle session
//	})
//
// Returns a stop function that cancels the keep-alive.
func KeepAlive(c *Context, interval time.Duration) func() {
	if interval <= 0 {
		interval = 15 * time.Second
	}

	done := make(chan struct{})
	ping := []byte{0x00} // minimal keep-alive packet

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				_ = c.SendDatagram(ping)
			case <-done:
				return
			case <-c.Context().Done():
				return
			}
		}
	}()

	return func() {
		select {
		case <-done:
		default:
			close(done)
		}
	}
}

// DefaultKeepAliveInterval is 15 seconds, chosen to be well under
// typical UDP NAT timeout of 20-30 seconds.
const DefaultKeepAliveInterval = 15 * time.Second
