package wt

// Chain composes multiple handlers into a single handler.
// Each handler runs sequentially. Useful for setup+teardown patterns.
func Chain(handlers ...HandlerFunc) HandlerFunc {
	return func(c *Context) {
		for _, h := range handlers {
			h(c)
		}
	}
}

// FirstMatch tries handlers in order, stopping at the first one that
// doesn't close the session. Useful for fallback patterns.
func FirstMatch(handlers ...HandlerFunc) HandlerFunc {
	return func(c *Context) {
		for _, h := range handlers {
			h(c)
			if c.Context().Err() != nil {
				return // session closed by this handler
			}
		}
	}
}
