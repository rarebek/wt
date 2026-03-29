package wt

import "iter"

// Sessions returns an iterator over all active sessions.
// Use with Go 1.23+ range-over-func:
//
//	for ctx := range server.Sessions().All() {
//	    log.Println(ctx.ID())
//	}
func (ss *SessionStore) All() iter.Seq[*Context] {
	return func(yield func(*Context) bool) {
		ss.mu.RLock()
		defer ss.mu.RUnlock()
		for _, ctx := range ss.sessions {
			if !yield(ctx) {
				return
			}
		}
	}
}
