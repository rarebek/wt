package wt

// StreamInterceptor intercepts stream message reads and writes.
// Useful for logging, metrics, validation, or transformation of messages.
type StreamInterceptor struct {
	onRead  func(data []byte) ([]byte, error)
	onWrite func(data []byte) ([]byte, error)
}

// InterceptorOption configures a StreamInterceptor.
type InterceptorOption func(*StreamInterceptor)

// OnRead sets a function that processes messages after reading from the stream.
// The function can transform or validate the message.
func OnRead(fn func(data []byte) ([]byte, error)) InterceptorOption {
	return func(si *StreamInterceptor) { si.onRead = fn }
}

// OnWrite sets a function that processes messages before writing to the stream.
func OnWrite(fn func(data []byte) ([]byte, error)) InterceptorOption {
	return func(si *StreamInterceptor) { si.onWrite = fn }
}

// InterceptedStream wraps a Stream with read/write interceptors.
type InterceptedStream struct {
	*Stream
	interceptor *StreamInterceptor
}

// Intercept wraps a stream with the given interceptors.
//
// Usage:
//
//	stream := wt.Intercept(rawStream,
//	    wt.OnRead(func(data []byte) ([]byte, error) {
//	        log.Printf("received %d bytes", len(data))
//	        return data, nil
//	    }),
//	    wt.OnWrite(func(data []byte) ([]byte, error) {
//	        log.Printf("sending %d bytes", len(data))
//	        return data, nil
//	    }),
//	)
func Intercept(s *Stream, opts ...InterceptorOption) *InterceptedStream {
	si := &StreamInterceptor{}
	for _, opt := range opts {
		opt(si)
	}
	return &InterceptedStream{
		Stream:      s,
		interceptor: si,
	}
}

// ReadMessage reads a message through the read interceptor.
func (is *InterceptedStream) ReadMessage() ([]byte, error) {
	data, err := is.Stream.ReadMessage()
	if err != nil {
		return nil, err
	}
	if is.interceptor.onRead != nil {
		return is.interceptor.onRead(data)
	}
	return data, nil
}

// WriteMessage writes a message through the write interceptor.
func (is *InterceptedStream) WriteMessage(data []byte) error {
	if is.interceptor.onWrite != nil {
		var err error
		data, err = is.interceptor.onWrite(data)
		if err != nil {
			return err
		}
	}
	return is.Stream.WriteMessage(data)
}
