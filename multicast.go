package wt

// Multicast sends a datagram to sessions matching a filter function.
// More flexible than Broadcast — only sends to sessions that match.
//
// Usage:
//
//	// Send to all sessions with user role "admin"
//	server.Multicast(data, func(c *Context) bool {
//	    role, _ := c.Get("role")
//	    return role == "admin"
//	})
func (s *Server) Multicast(data []byte, filter func(*Context) bool) {
	s.sessions.Each(func(c *Context) {
		if filter(c) {
			_ = c.SendDatagram(data)
		}
	})
}

// MulticastStream sends a reliable message via streams to matching sessions.
func (s *Server) MulticastStream(data []byte, filter func(*Context) bool) {
	s.sessions.Each(func(c *Context) {
		if filter(c) {
			go func() {
				stream, err := c.OpenStream()
				if err != nil {
					return
				}
				_ = stream.WriteMessage(data)
				stream.Close()
			}()
		}
	})
}
