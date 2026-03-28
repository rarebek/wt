package wt

// BroadcastStream sends a message to all room members via reliable streams.
// Unlike Broadcast (datagrams, unreliable), this guarantees delivery.
// Each member gets a new stream with the message.
func (r *Room) BroadcastStream(data []byte) {
	r.ForEach(func(c *Context) {
		go func() {
			stream, err := c.OpenStream()
			if err != nil {
				return
			}
			_ = stream.WriteMessage(data)
			stream.Close()
		}()
	})
}

// BroadcastStreamExcept sends a reliable message to all except the given session.
func (r *Room) BroadcastStreamExcept(data []byte, excludeID string) {
	r.ForEach(func(c *Context) {
		if c.ID() == excludeID {
			return
		}
		go func() {
			stream, err := c.OpenStream()
			if err != nil {
				return
			}
			_ = stream.WriteMessage(data)
			stream.Close()
		}()
	})
}
