package wt

// ShutdownHook is a function called during server shutdown.
type ShutdownHook func()

// OnShutdown registers a function to be called when the server shuts down.
// Hooks run in registration order before connections are drained.
func (s *Server) OnShutdown(fn ShutdownHook) {
	s.mu.Lock()
	s.shutdownHooks = append(s.shutdownHooks, fn)
	s.mu.Unlock()
}
