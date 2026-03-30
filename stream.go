package wt

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"time"

	"github.com/quic-go/webtransport-go"
	"github.com/rarebek/wt/codec"
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

// RPCRequest represents a JSON-RPC-like request over a stream.
type RPCRequest struct {
	ID     uint64          `json:"id"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params,omitempty"`
}

// RPCResponse represents a JSON-RPC-like response.
type RPCResponse struct {
	ID     uint64          `json:"id"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  *RPCError       `json:"error,omitempty"`
}

// RPCError represents an error in the RPC response.
type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *RPCError) Error() string {
	return fmt.Sprintf("rpc error %d: %s", e.Code, e.Message)
}

// RPCHandler handles a single RPC method.
type RPCHandler func(params json.RawMessage) (any, error)

// RPCServer handles JSON-RPC requests over a WebTransport stream.
type RPCServer struct {
	mu       sync.RWMutex
	handlers map[string]RPCHandler
}

// NewRPCServer creates a new RPC server.
func NewRPCServer() *RPCServer {
	return &RPCServer{
		handlers: make(map[string]RPCHandler),
	}
}

// Register adds a handler for the given method name.
func (rpc *RPCServer) Register(method string, handler RPCHandler) {
	rpc.mu.Lock()
	rpc.handlers[method] = handler
	rpc.mu.Unlock()
}

// Serve handles RPC requests on the given stream.
// Each request-response pair uses the stream's message framing.
// Call this from a StreamHandler or HandleStream.
func (rpc *RPCServer) Serve(s *Stream) {
	defer s.Close()

	for {
		data, err := s.ReadMessage()
		if err != nil {
			return
		}

		var req RPCRequest
		if err := json.Unmarshal(data, &req); err != nil {
			resp := RPCResponse{
				Error: &RPCError{Code: -32700, Message: "parse error"},
			}
			respData, _ := json.Marshal(resp)
			s.WriteMessage(respData)
			continue
		}

		rpc.mu.RLock()
		handler, ok := rpc.handlers[req.Method]
		rpc.mu.RUnlock()

		var resp RPCResponse
		resp.ID = req.ID

		if !ok {
			resp.Error = &RPCError{Code: -32601, Message: fmt.Sprintf("method not found: %s", req.Method)}
		} else {
			result, err := handler(req.Params)
			if err != nil {
				resp.Error = &RPCError{Code: -32000, Message: err.Error()}
			} else {
				resultData, _ := json.Marshal(result)
				resp.Result = resultData
			}
		}

		respData, _ := json.Marshal(resp)
		if err := s.WriteMessage(respData); err != nil {
			return
		}
	}
}

// RPCClient sends JSON-RPC requests over a WebTransport stream.
type RPCClient struct {
	stream *Stream
	nextID atomic.Uint64
}

// NewRPCClient wraps a stream for RPC calls.
func NewRPCClient(s *Stream) *RPCClient {
	return &RPCClient{stream: s}
}

// Call sends an RPC request and waits for the response.
func (c *RPCClient) Call(method string, params any) (json.RawMessage, error) {
	id := c.nextID.Add(1)

	paramsData, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}

	req := RPCRequest{
		ID:     id,
		Method: method,
		Params: paramsData,
	}

	reqData, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	if err := c.stream.WriteMessage(reqData); err != nil {
		return nil, err
	}

	respData, err := c.stream.ReadMessage()
	if err != nil {
		return nil, err
	}

	var resp RPCResponse
	if err := json.Unmarshal(respData, &resp); err != nil {
		return nil, err
	}

	if resp.Error != nil {
		return nil, resp.Error
	}

	return resp.Result, nil
}

// CallTyped sends an RPC request and unmarshals the result into the given type.
func CallTyped[T any](c *RPCClient, method string, params any) (T, error) {
	var zero T
	result, err := c.Call(method, params)
	if err != nil {
		return zero, err
	}

	var v T
	if err := json.Unmarshal(result, &v); err != nil {
		return zero, err
	}
	return v, nil
}

// Close closes the underlying stream.
func (c *RPCClient) Close() error {
	return c.stream.Close()
}

// StreamMux multiplexes different stream types within a single session.
// Each stream's first 2 bytes identify its type, and the mux routes it
// to the appropriate handler.
//
// This solves the problem of "I have one WebTransport session but I need
// different handlers for chat streams vs game state streams vs file uploads."
//
// Usage:
//
//	mux := wt.NewStreamMux()
//	mux.Handle(1, handleChat)      // type=1 → chat handler
//	mux.Handle(2, handleGameInput) // type=2 → game input handler
//	mux.Handle(3, handleFileUpload) // type=3 → file upload handler
//
//	server.Handle("/app", func(c *wt.Context) {
//	    mux.Serve(c) // auto-routes streams by type
//	})
type StreamMux struct {
	mu       sync.RWMutex
	handlers map[uint16]StreamHandler
	fallback StreamHandler
}

// StreamHandler handles a single stream.
type StreamHandler func(s *Stream, c *Context)

// NewStreamMux creates a new stream multiplexer.
func NewStreamMux() *StreamMux {
	return &StreamMux{
		handlers: make(map[uint16]StreamHandler),
	}
}

// Handle registers a handler for the given stream type ID.
func (m *StreamMux) Handle(typeID uint16, handler StreamHandler) {
	m.mu.Lock()
	m.handlers[typeID] = handler
	m.mu.Unlock()
}

// Fallback sets a handler for unrecognized stream types.
func (m *StreamMux) Fallback(handler StreamHandler) {
	m.mu.Lock()
	m.fallback = handler
	m.mu.Unlock()
}

// Serve accepts streams from the context and routes them by type.
// Blocks until the session is closed.
func (m *StreamMux) Serve(c *Context) {
	for {
		stream, err := c.AcceptStream()
		if err != nil {
			return
		}
		go m.route(stream, c)
	}
}

func (m *StreamMux) route(s *Stream, c *Context) {
	// Read the 2-byte type header
	header := make([]byte, 2)
	if _, err := io.ReadFull(s, header); err != nil {
		s.Close()
		return
	}
	typeID := binary.BigEndian.Uint16(header)

	m.mu.RLock()
	handler, ok := m.handlers[typeID]
	fallback := m.fallback
	m.mu.RUnlock()

	if ok {
		handler(s, c)
	} else if fallback != nil {
		fallback(s, c)
	} else {
		s.CancelWrite(uint32(CodeProtocolError))
		s.Close()
	}
}

// OpenTypedStream opens a stream with the given type header.
// The remote end's StreamMux will route it to the matching handler.
func OpenTypedStream(c *Context, typeID uint16) (*Stream, error) {
	stream, err := c.OpenStream()
	if err != nil {
		return nil, fmt.Errorf("wt: open typed stream: %w", err)
	}

	header := make([]byte, 2)
	binary.BigEndian.PutUint16(header, typeID)
	if _, err := stream.Write(header); err != nil {
		stream.Close()
		return nil, fmt.Errorf("wt: write stream type header: %w", err)
	}

	return stream, nil
}

// TypedStream provides type-safe read/write over a Stream using a codec.
type TypedStream[R any, W any] struct {
	stream *Stream
	codec  codec.Codec
}

// NewTypedStream wraps a Stream with typed encoding/decoding.
func NewTypedStream[R any, W any](s *Stream, c codec.Codec) *TypedStream[R, W] {
	return &TypedStream[R, W]{stream: s, codec: c}
}

// Read reads and decodes a message from the stream.
func (ts *TypedStream[R, W]) Read() (R, error) {
	var zero R
	data, err := ts.stream.ReadMessage()
	if err != nil {
		return zero, err
	}
	var v R
	if err := ts.codec.Unmarshal(data, &v); err != nil {
		return zero, err
	}
	return v, nil
}

// Write encodes and writes a message to the stream.
func (ts *TypedStream[R, W]) Write(v W) error {
	data, err := ts.codec.Marshal(v)
	if err != nil {
		return err
	}
	return ts.stream.WriteMessage(data)
}

// Stream returns the underlying Stream.
func (ts *TypedStream[R, W]) Stream() *Stream {
	return ts.stream
}

// Close closes the underlying stream.
func (ts *TypedStream[R, W]) Close() error {
	return ts.stream.Close()
}

// TypedDatagram provides type-safe datagram read/write on a Context.
type TypedDatagram[T any] struct {
	ctx   *Context
	codec codec.Codec
}

// NewTypedDatagram wraps a Context with typed datagram encoding/decoding.
func NewTypedDatagram[T any](ctx *Context, c codec.Codec) *TypedDatagram[T] {
	return &TypedDatagram[T]{ctx: ctx, codec: c}
}

// Send encodes and sends a datagram.
func (td *TypedDatagram[T]) Send(v T) error {
	data, err := td.codec.Marshal(v)
	if err != nil {
		return err
	}
	return td.ctx.SendDatagram(data)
}

// Receive receives and decodes a datagram.
func (td *TypedDatagram[T]) Receive() (T, error) {
	var zero T
	data, err := td.ctx.ReceiveDatagram()
	if err != nil {
		return zero, err
	}
	var v T
	if err := td.codec.Unmarshal(data, &v); err != nil {
		return zero, err
	}
	return v, nil
}

// TypedRoom provides type-safe broadcast over a Room using a codec.
type TypedRoom[T any] struct {
	room  *Room
	codec codec.Codec
}

// NewTypedRoom wraps a Room with typed broadcast support.
func NewTypedRoom[T any](room *Room, c codec.Codec) *TypedRoom[T] {
	return &TypedRoom[T]{room: room, codec: c}
}

// Broadcast encodes and broadcasts a value to all room members via datagrams.
func (tr *TypedRoom[T]) Broadcast(v T) error {
	data, err := tr.codec.Marshal(v)
	if err != nil {
		return err
	}
	tr.room.Broadcast(data)
	return nil
}

// BroadcastExcept broadcasts to all members except the specified session.
func (tr *TypedRoom[T]) BroadcastExcept(v T, excludeID string) error {
	data, err := tr.codec.Marshal(v)
	if err != nil {
		return err
	}
	tr.room.BroadcastExcept(data, excludeID)
	return nil
}

// Room returns the underlying Room.
func (tr *TypedRoom[T]) Room() *Room {
	return tr.room
}
