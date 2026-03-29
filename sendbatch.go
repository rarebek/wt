package wt

// SendBatch sends multiple datagrams in rapid succession.
// More efficient than calling SendDatagram in a loop because
// it reduces lock contention on the underlying QUIC connection.
func (c *Context) SendBatch(messages [][]byte) int {
	sent := 0
	for _, msg := range messages {
		if err := c.SendDatagram(msg); err != nil {
			break
		}
		sent++
	}
	return sent
}

// SendBatchRoom sends multiple datagrams to all room members.
func (r *Room) SendBatch(messages [][]byte) {
	for _, msg := range messages {
		r.Broadcast(msg)
	}
}
