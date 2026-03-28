package wt

// StreamOptions configures stream behavior.
type StreamOptions struct {
	// ReadBufferSize sets the size of the read buffer (default: 4096).
	ReadBufferSize int
	// WriteBufferSize sets the size of the write buffer (default: 4096).
	WriteBufferSize int
	// MaxMessageSize overrides the default maximum message size for this stream.
	MaxMessageSize int
}

// DefaultStreamOptions returns default stream configuration.
func DefaultStreamOptions() StreamOptions {
	return StreamOptions{
		ReadBufferSize:  4096,
		WriteBufferSize: 4096,
		MaxMessageSize:  MaxMessageSize,
	}
}

// BufferedReader wraps a Stream with a larger read buffer.
type BufferedReader struct {
	*Stream
	buf  []byte
	n    int
	pos  int
}

// NewBufferedReader creates a stream reader with a custom buffer size.
func NewBufferedReader(s *Stream, bufSize int) *BufferedReader {
	if bufSize < 1024 {
		bufSize = 4096
	}
	return &BufferedReader{
		Stream: s,
		buf:    make([]byte, bufSize),
	}
}

// ReadBuffered reads into the internal buffer and returns available bytes.
// More efficient than multiple small Read calls.
func (br *BufferedReader) ReadBuffered() ([]byte, error) {
	n, err := br.Stream.Read(br.buf)
	if err != nil {
		return nil, err
	}
	return br.buf[:n], nil
}
