package wt

// HandlerFunc is the function signature for WebTransport session handlers.
type HandlerFunc func(*Context)

// MiddlewareFunc is the function signature for middleware.
// Call next(c) to pass control to the next middleware or final handler.
type MiddlewareFunc func(c *Context, next HandlerFunc)

// buildChain wraps the handler with middleware in reverse order,
// so middleware[0] runs first and the handler runs last.
func buildChain(handler HandlerFunc, mw []MiddlewareFunc) HandlerFunc {
	if len(mw) == 0 {
		return handler
	}

	// Build from inside out
	h := handler
	for i := len(mw) - 1; i >= 0; i-- {
		m := mw[i]
		next := h
		h = func(c *Context) {
			m(c, next)
		}
	}
	return h
}
