package wt

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/quic-go/webtransport-go"
)

// Stream wraps a bidirectional WebTransport stream with framing and helpers.
type Stream struct {
	raw    *webtransport.Stream
	ctx    *Context
	hdrBuf [4]byte // reusable header buffer for Read/WriteMessage
}

func newStream(s *webtransport.Stream, ctx *Context) *Stream {
	return &Stream{raw: s, ctx: ctx}
}

// Raw returns the underlying webtransport.Stream.
func (s *Stream) Raw() *webtransport.Stream {
	return s.raw
}

// Context returns the session context this stream belongs to.
func (s *Stream) SessionContext() *Context {
	return s.ctx
}

// Read reads raw bytes from the stream.
func (s *Stream) Read(b []byte) (int, error) {
	return s.raw.Read(b)
}

// Write writes raw bytes to the stream.
func (s *Stream) Write(b []byte) (int, error) {
	return s.raw.Write(b)
}

// Close closes the stream.
func (s *Stream) Close() error {
	return s.raw.Close()
}

// SetDeadline sets read and write deadlines.
func (s *Stream) SetDeadline(t time.Time) error {
	return s.raw.SetDeadline(t)
}

// SetReadDeadline sets the read deadline.
func (s *Stream) SetReadDeadline(t time.Time) error {
	return s.raw.SetReadDeadline(t)
}

// SetWriteDeadline sets the write deadline.
func (s *Stream) SetWriteDeadline(t time.Time) error {
	return s.raw.SetWriteDeadline(t)
}

// WriteMessage writes a length-prefixed message to the stream.
// Format: [4 bytes big-endian length][payload]
// Uses embedded header buffer to avoid allocation.
func (s *Stream) WriteMessage(data []byte) error {
	if len(data) > MaxMessageSize {
		return fmt.Errorf("wt: message too large: %d > %d", len(data), MaxMessageSize)
	}
	binary.BigEndian.PutUint32(s.hdrBuf[:], uint32(len(data)))
	if _, err := s.raw.Write(s.hdrBuf[:]); err != nil {
		return err
	}
	_, err := s.raw.Write(data)
	return err
}

// ReadMessage reads a length-prefixed message from the stream.
func (s *Stream) ReadMessage() ([]byte, error) {
	if _, err := io.ReadFull(s.raw, s.hdrBuf[:]); err != nil {
		return nil, err
	}
	length := binary.BigEndian.Uint32(s.hdrBuf[:])
	if length > uint32(MaxMessageSize) {
		return nil, fmt.Errorf("wt: message too large: %d > %d", length, MaxMessageSize)
	}
	data := make([]byte, length)
	if _, err := io.ReadFull(s.raw, data); err != nil {
		return nil, err
	}
	return data, nil
}

// CancelRead cancels the read side of the stream.
func (s *Stream) CancelRead(code uint32) {
	s.raw.CancelRead(webtransport.StreamErrorCode(code))
}

// CancelWrite cancels the write side of the stream.
func (s *Stream) CancelWrite(code uint32) {
	s.raw.CancelWrite(webtransport.StreamErrorCode(code))
}

// MaxMessageSize is the maximum size of a length-prefixed message (16 MB).
const MaxMessageSize = 16 * 1024 * 1024

// SendStream wraps a unidirectional send stream.
type SendStream struct {
	raw    *webtransport.SendStream
	hdrBuf [4]byte
}

// Write writes bytes to the send stream.
func (s *SendStream) Write(b []byte) (int, error) {
	return s.raw.Write(b)
}

// WriteMessage writes a length-prefixed message.
func (s *SendStream) WriteMessage(data []byte) error {
	if len(data) > MaxMessageSize {
		return fmt.Errorf("wt: message too large: %d > %d", len(data), MaxMessageSize)
	}
	binary.BigEndian.PutUint32(s.hdrBuf[:], uint32(len(data)))
	if _, err := s.raw.Write(s.hdrBuf[:]); err != nil {
		return err
	}
	_, err := s.raw.Write(data)
	return err
}

// Close closes the send stream.
func (s *SendStream) Close() error {
	return s.raw.Close()
}

// SetWriteDeadline sets the write deadline.
func (s *SendStream) SetWriteDeadline(t time.Time) error {
	return s.raw.SetWriteDeadline(t)
}

// CancelWrite cancels writing.
func (s *SendStream) CancelWrite(code uint32) {
	s.raw.CancelWrite(webtransport.StreamErrorCode(code))
}

// ReceiveStream wraps a unidirectional receive stream.
type ReceiveStream struct {
	raw    *webtransport.ReceiveStream
	hdrBuf [4]byte
}

// Read reads bytes from the receive stream.
func (s *ReceiveStream) Read(b []byte) (int, error) {
	return s.raw.Read(b)
}

// ReadMessage reads a length-prefixed message.
func (s *ReceiveStream) ReadMessage() ([]byte, error) {
	if _, err := io.ReadFull(s.raw, s.hdrBuf[:]); err != nil {
		return nil, err
	}
	length := binary.BigEndian.Uint32(s.hdrBuf[:])
	if length > uint32(MaxMessageSize) {
		return nil, fmt.Errorf("wt: message too large: %d > %d", length, MaxMessageSize)
	}
	data := make([]byte, length)
	if _, err := io.ReadFull(s.raw, data); err != nil {
		return nil, err
	}
	return data, nil
}

// SetReadDeadline sets the read deadline.
func (s *ReceiveStream) SetReadDeadline(t time.Time) error {
	return s.raw.SetReadDeadline(t)
}

// CancelRead cancels reading.
func (s *ReceiveStream) CancelRead(code uint32) {
	s.raw.CancelRead(webtransport.StreamErrorCode(code))
}

// ContextStream wraps a Stream with context-aware read/write operations.
// When the context is cancelled, all pending reads and writes are unblocked.
type ContextStream struct {
	*Stream
	ctx    context.Context
	cancel context.CancelFunc
}

// WithContext creates a ContextStream that respects the given context.
// When ctx is cancelled, the stream is closed automatically.
func (s *Stream) WithContext(ctx context.Context) *ContextStream {
	ctx, cancel := context.WithCancel(ctx)
	cs := &ContextStream{
		Stream: s,
		ctx:    ctx,
		cancel: cancel,
	}
	// Auto-close stream when context is cancelled
	go func() {
		<-ctx.Done()
		s.Close()
	}()
	return cs
}

// Context returns the stream's context.
func (cs *ContextStream) Context() context.Context {
	return cs.ctx
}

// ReadMessageContext reads a length-prefixed message with context support.
// Returns context.Canceled or context.DeadlineExceeded if the context is done.
func (cs *ContextStream) ReadMessageContext() ([]byte, error) {
	// Set deadline from context if available
	if deadline, ok := cs.ctx.Deadline(); ok {
		cs.Stream.SetReadDeadline(deadline)
	}

	msg, err := cs.Stream.ReadMessage()
	if err != nil {
		// Check if the error is due to context cancellation
		if cs.ctx.Err() != nil {
			return nil, cs.ctx.Err()
		}
		return nil, err
	}
	return msg, nil
}

// WriteMessageContext writes a length-prefixed message with context support.
func (cs *ContextStream) WriteMessageContext(data []byte) error {
	if deadline, ok := cs.ctx.Deadline(); ok {
		cs.Stream.SetWriteDeadline(deadline)
	}

	if err := cs.Stream.WriteMessage(data); err != nil {
		if cs.ctx.Err() != nil {
			return cs.ctx.Err()
		}
		return err
	}
	return nil
}

// Close cancels the context and closes the stream.
func (cs *ContextStream) Close() error {
	cs.cancel()
	return cs.Stream.Close()
}

// WithDeadline creates a stream that automatically closes at the given deadline.
func (s *Stream) WithDeadline(deadline time.Time) *ContextStream {
	ctx, cancel := context.WithDeadline(context.Background(), deadline)
	cs := &ContextStream{
		Stream: s,
		ctx:    ctx,
		cancel: cancel,
	}
	go func() {
		<-ctx.Done()
		s.Close()
	}()
	return cs
}

// WithTimeout creates a stream that automatically closes after the given duration.
func (s *Stream) WithTimeout(d time.Duration) *ContextStream {
	return s.WithDeadline(time.Now().Add(d))
}

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
	buf []byte
	n   int
	pos int
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

// StreamPool manages a pool of reusable outbound streams per session.
// Instead of opening a new stream for every message (expensive),
// grab one from the pool, use it, return it.
type StreamPool struct {
	ctx  *Context
	mu   sync.Mutex
	pool []*Stream
	max  int
}

// NewStreamPool creates a stream pool for the given session.
func NewStreamPool(c *Context, maxIdle int) *StreamPool {
	if maxIdle < 1 {
		maxIdle = 4
	}
	return &StreamPool{ctx: c, max: maxIdle}
}

// Get returns a stream from the pool or opens a new one.
func (sp *StreamPool) Get() (*Stream, error) {
	sp.mu.Lock()
	if len(sp.pool) > 0 {
		s := sp.pool[len(sp.pool)-1]
		sp.pool = sp.pool[:len(sp.pool)-1]
		sp.mu.Unlock()
		return s, nil
	}
	sp.mu.Unlock()
	return sp.ctx.OpenStream()
}

// Put returns a stream to the pool for reuse.
// If the pool is full, the stream is closed.
func (sp *StreamPool) Put(s *Stream) {
	sp.mu.Lock()
	if len(sp.pool) < sp.max {
		sp.pool = append(sp.pool, s)
		sp.mu.Unlock()
		return
	}
	sp.mu.Unlock()
	s.Close()
}

// Size returns the number of idle streams in the pool.
func (sp *StreamPool) Size() int {
	sp.mu.Lock()
	n := len(sp.pool)
	sp.mu.Unlock()
	return n
}

// Close closes all pooled streams.
func (sp *StreamPool) Close() {
	sp.mu.Lock()
	for _, s := range sp.pool {
		s.Close()
	}
	sp.pool = nil
	sp.mu.Unlock()
}

// Pipe bidirectionally copies data between two streams.
// Useful for proxying one WebTransport stream to another.
// Returns when either stream closes or encounters an error.
func Pipe(a, b *Stream) error {
	var wg sync.WaitGroup
	var firstErr error
	var errOnce sync.Once

	setErr := func(err error) {
		if err != nil && err != io.EOF {
			errOnce.Do(func() { firstErr = err })
		}
	}

	wg.Add(2)

	// a → b
	go func() {
		defer wg.Done()
		_, err := io.Copy(b, a)
		setErr(err)
		b.Close()
	}()

	// b → a
	go func() {
		defer wg.Done()
		_, err := io.Copy(a, b)
		setErr(err)
		a.Close()
	}()

	wg.Wait()
	return firstErr
}

// PipeRaw bidirectionally copies between an io.ReadWriteCloser and a Stream.
func PipeRaw(rw io.ReadWriteCloser, s *Stream) error {
	var wg sync.WaitGroup
	var firstErr error
	var errOnce sync.Once

	setErr := func(err error) {
		if err != nil && err != io.EOF {
			errOnce.Do(func() { firstErr = err })
		}
	}

	wg.Add(2)

	go func() {
		defer wg.Done()
		_, err := io.Copy(s, rw)
		setErr(err)
		s.Close()
	}()

	go func() {
		defer wg.Done()
		_, err := io.Copy(rw, s)
		setErr(err)
		rw.Close()
	}()

	wg.Wait()
	return firstErr
}

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

// CompressedStream wraps a Stream with per-message gzip compression.
// Messages are compressed before writing and decompressed after reading.
// Only useful for messages > ~100 bytes; smaller messages may increase in size.
type CompressedStream struct {
	*Stream
	writerPool sync.Pool
	threshold  int // minimum size to compress
}

// NewCompressedStream wraps a stream with gzip compression.
// Messages smaller than threshold bytes are sent uncompressed.
// A 1-byte header indicates compression: 0x00 = raw, 0x01 = gzip.
func NewCompressedStream(s *Stream, threshold int) *CompressedStream {
	if threshold <= 0 {
		threshold = 128
	}
	return &CompressedStream{
		Stream:    s,
		threshold: threshold,
		writerPool: sync.Pool{
			New: func() any {
				w, _ := gzip.NewWriterLevel(nil, gzip.BestSpeed)
				return w
			},
		},
	}
}

// WriteMessage compresses and writes a message.
// Small messages are sent raw with a 0x00 prefix.
// Large messages are gzip-compressed with a 0x01 prefix.
func (cs *CompressedStream) WriteMessage(data []byte) error {
	if len(data) < cs.threshold {
		// Send raw with 0x00 prefix
		msg := make([]byte, 1+len(data))
		msg[0] = 0x00
		copy(msg[1:], data)
		return cs.Stream.WriteMessage(msg)
	}

	// Compress
	var buf bytes.Buffer
	buf.WriteByte(0x01) // compressed flag

	w := cs.writerPool.Get().(*gzip.Writer)
	w.Reset(&buf)
	if _, err := w.Write(data); err != nil {
		cs.writerPool.Put(w)
		return err
	}
	if err := w.Close(); err != nil {
		cs.writerPool.Put(w)
		return err
	}
	cs.writerPool.Put(w)

	return cs.Stream.WriteMessage(buf.Bytes())
}

// ReadMessage reads and decompresses a message.
func (cs *CompressedStream) ReadMessage() ([]byte, error) {
	msg, err := cs.Stream.ReadMessage()
	if err != nil {
		return nil, err
	}

	if len(msg) == 0 {
		return nil, nil
	}

	switch msg[0] {
	case 0x00:
		// Raw
		return msg[1:], nil
	case 0x01:
		// Gzip compressed
		r, err := gzip.NewReader(bytes.NewReader(msg[1:]))
		if err != nil {
			return nil, err
		}
		defer r.Close()
		return io.ReadAll(r)
	default:
		// Unknown — treat as raw (backwards compat)
		return msg, nil
	}
}

// CompressionStats tracks compression effectiveness.
type CompressionStats struct {
	RawBytes        int64
	CompressedBytes int64
	MessagesRaw     int64
	MessagesGzip    int64
}

// Ratio returns the compression ratio (0.0 = perfect, 1.0 = no compression).
func (cs CompressionStats) Ratio() float64 {
	if cs.RawBytes == 0 {
		return 0
	}
	return float64(cs.CompressedBytes) / float64(cs.RawBytes)
}

// BackpressureWriter wraps stream writes with buffering and backpressure tracking.
// Use this when you need to detect slow consumers without blocking the sender goroutine.
type BackpressureWriter struct {
	stream *Stream
	mu     sync.Mutex
	queue  chan []byte
	done   chan struct{}
	closed bool

	// Stats
	dropped uint64
	sent    uint64
}

// NewBackpressureWriter creates a writer with the given buffer size.
// Messages are dropped (not queued indefinitely) when the buffer is full,
// preventing slow consumers from causing memory leaks.
func NewBackpressureWriter(s *Stream, bufferSize int) *BackpressureWriter {
	if bufferSize < 1 {
		bufferSize = 16
	}
	bw := &BackpressureWriter{
		stream: s,
		queue:  make(chan []byte, bufferSize),
		done:   make(chan struct{}),
	}
	go bw.writeLoop()
	return bw
}

func (bw *BackpressureWriter) writeLoop() {
	for {
		select {
		case msg, ok := <-bw.queue:
			if !ok {
				return
			}
			_ = bw.stream.WriteMessage(msg)
			bw.mu.Lock()
			bw.sent++
			bw.mu.Unlock()
		case <-bw.done:
			return
		}
	}
}

// Send attempts to queue a message for sending.
// Returns true if queued, false if the buffer is full (message dropped).
func (bw *BackpressureWriter) Send(msg []byte) bool {
	select {
	case bw.queue <- msg:
		return true
	default:
		bw.mu.Lock()
		bw.dropped++
		bw.mu.Unlock()
		return false
	}
}

// Stats returns the number of messages sent and dropped.
func (bw *BackpressureWriter) Stats() (sent, dropped uint64) {
	bw.mu.Lock()
	defer bw.mu.Unlock()
	return bw.sent, bw.dropped
}

// Close stops the writer and drains remaining messages.
func (bw *BackpressureWriter) Close() {
	bw.mu.Lock()
	if bw.closed {
		bw.mu.Unlock()
		return
	}
	bw.closed = true
	bw.mu.Unlock()
	close(bw.done)
	close(bw.queue)
}

// IsFull returns true if the send buffer is at capacity.
func (bw *BackpressureWriter) IsFull() bool {
	return len(bw.queue) == cap(bw.queue)
}

// BufferUsage returns the current buffer utilization as a fraction (0.0 to 1.0).
func (bw *BackpressureWriter) BufferUsage() float64 {
	return float64(len(bw.queue)) / float64(cap(bw.queue))
}
