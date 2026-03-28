package wt

// HandleStream is a convenience for handlers that process one stream at a time.
// It auto-accepts streams and calls the handler for each in a new goroutine.
//
// Usage:
//
//	server.Handle("/echo", wt.HandleStream(func(s *wt.Stream, c *wt.Context) {
//	    defer s.Close()
//	    msg, _ := s.ReadMessage()
//	    s.WriteMessage(msg)
//	}))
func HandleStream(fn StreamHandler) HandlerFunc {
	return func(c *Context) {
		for {
			stream, err := c.AcceptStream()
			if err != nil {
				return
			}
			go fn(stream, c)
		}
	}
}

// HandleDatagram is a convenience for handlers that only process datagrams.
// It loops receiving datagrams and calls the handler for each.
//
// Usage:
//
//	server.Handle("/ping", wt.HandleDatagram(func(data []byte, c *wt.Context) []byte {
//	    return data // echo
//	}))
func HandleDatagram(fn func(data []byte, c *Context) []byte) HandlerFunc {
	return func(c *Context) {
		for {
			data, err := c.ReceiveDatagram()
			if err != nil {
				return
			}
			reply := fn(data, c)
			if reply != nil {
				_ = c.SendDatagram(reply)
			}
		}
	}
}

// HandleBoth is a convenience for handlers that process both streams and datagrams.
// Datagrams are handled in a background goroutine, streams in the main loop.
//
// Usage:
//
//	server.Handle("/game", wt.HandleBoth(
//	    func(s *wt.Stream, c *wt.Context) {
//	        // handle game event stream
//	    },
//	    func(data []byte, c *wt.Context) []byte {
//	        // handle position datagram
//	        return nil // no reply
//	    },
//	))
func HandleBoth(streamFn StreamHandler, datagramFn func([]byte, *Context) []byte) HandlerFunc {
	return func(c *Context) {
		// Datagrams in background
		go func() {
			for {
				data, err := c.ReceiveDatagram()
				if err != nil {
					return
				}
				reply := datagramFn(data, c)
				if reply != nil {
					_ = c.SendDatagram(reply)
				}
			}
		}()

		// Streams in foreground
		for {
			stream, err := c.AcceptStream()
			if err != nil {
				return
			}
			go streamFn(stream, c)
		}
	}
}
