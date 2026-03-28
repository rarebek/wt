package wt

import "iter"

// Streams returns an iterator over incoming streams.
// Use with Go 1.23+ range-over-func:
//
//	for stream := range wt.Streams(c) {
//	    go handleStream(stream)
//	}
func Streams(c *Context) iter.Seq[*Stream] {
	return func(yield func(*Stream) bool) {
		for {
			stream, err := c.AcceptStream()
			if err != nil {
				return
			}
			if !yield(stream) {
				return
			}
		}
	}
}

// Datagrams returns an iterator over incoming datagrams.
//
//	for data := range wt.Datagrams(c) {
//	    process(data)
//	}
func Datagrams(c *Context) iter.Seq[[]byte] {
	return func(yield func([]byte) bool) {
		for {
			data, err := c.ReceiveDatagram()
			if err != nil {
				return
			}
			if !yield(data) {
				return
			}
		}
	}
}

// Messages returns an iterator over length-prefixed messages from a stream.
//
//	for msg := range wt.Messages(stream) {
//	    process(msg)
//	}
func Messages(s *Stream) iter.Seq[[]byte] {
	return func(yield func([]byte) bool) {
		for {
			msg, err := s.ReadMessage()
			if err != nil {
				return
			}
			if !yield(msg) {
				return
			}
		}
	}
}
