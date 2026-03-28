package wt

import "time"

// Ticker sends periodic datagrams to a session at a fixed interval.
// Useful for game state updates, heartbeats, or periodic data pushes.
//
// Usage:
//
//	server.Handle("/game", func(c *wt.Context) {
//	    ticker := wt.NewTicker(c, 50*time.Millisecond, func() []byte {
//	        return getGameState()
//	    })
//	    defer ticker.Stop()
//	    // ... handle other streams
//	})
type Ticker struct {
	ctx    *Context
	ticker *time.Ticker
	done   chan struct{}
}

// NewTicker starts sending datagrams at the given interval.
// The getData function is called each tick to produce the datagram payload.
// Return nil to skip a tick (no datagram sent).
func NewTicker(c *Context, interval time.Duration, getData func() []byte) *Ticker {
	t := &Ticker{
		ctx:    c,
		ticker: time.NewTicker(interval),
		done:   make(chan struct{}),
	}

	go func() {
		for {
			select {
			case <-t.ticker.C:
				data := getData()
				if data != nil {
					_ = c.SendDatagram(data)
				}
			case <-t.done:
				return
			case <-c.Context().Done():
				return
			}
		}
	}()

	return t
}

// Stop stops the ticker.
func (t *Ticker) Stop() {
	t.ticker.Stop()
	select {
	case <-t.done:
	default:
		close(t.done)
	}
}
