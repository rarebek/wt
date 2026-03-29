package wt

import (
	"fmt"
	"sync"
	"time"
)

// MaxDatagramSize is the recommended maximum datagram payload size.
// QUIC datagrams are limited by the path MTU minus QUIC overhead.
// Typical safe size is ~1200 bytes (minimum QUIC MTU of 1280 minus headers).
// Larger datagrams may be fragmented or dropped.
const MaxDatagramSize = 1200

// ValidateDatagramSize checks if a datagram payload is within safe limits.
// Returns an error if the payload is too large.
func ValidateDatagramSize(data []byte) error {
	if len(data) > MaxDatagramSize {
		return fmt.Errorf("wt: datagram payload %d bytes exceeds safe limit of %d bytes",
			len(data), MaxDatagramSize)
	}
	return nil
}

// SendDatagramSafe sends a datagram with size validation.
// Returns an error if the payload exceeds MaxDatagramSize.
func (c *Context) SendDatagramSafe(data []byte) error {
	if err := ValidateDatagramSize(data); err != nil {
		return err
	}
	return c.SendDatagram(data)
}

// Throttle limits the rate of datagram sends per session.
// Implements a simple token bucket: N tokens per second, burst B.
type Throttle struct {
	mu       sync.Mutex
	rate     float64 // tokens per second
	burst    float64 // max tokens
	tokens   float64
	lastTime time.Time
}

// NewThrottle creates a rate throttle.
// rate: messages per second allowed. burst: max messages allowed in a burst.
func NewThrottle(rate float64, burst int) *Throttle {
	return &Throttle{
		rate:     rate,
		burst:    float64(burst),
		tokens:   float64(burst),
		lastTime: time.Now(),
	}
}

// Allow checks if a message can be sent. Returns true and consumes a token if allowed.
func (t *Throttle) Allow() bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(t.lastTime).Seconds()
	t.lastTime = now

	t.tokens += elapsed * t.rate
	if t.tokens > t.burst {
		t.tokens = t.burst
	}

	if t.tokens < 1 {
		return false
	}
	t.tokens--
	return true
}

// Wait blocks until a token is available or context is cancelled.
func (t *Throttle) Wait() {
	for !t.Allow() {
		time.Sleep(time.Millisecond)
	}
}

// ThrottledSend sends a datagram only if the throttle allows it.
// Returns false if throttled (message dropped).
func (c *Context) ThrottledSend(t *Throttle, data []byte) bool {
	if !t.Allow() {
		return false
	}
	return c.SendDatagram(data) == nil
}

// Ticker sends periodic datagrams to a session at a fixed interval.
// Useful for game state updates, heartbeats, or periodic data pushes.
//
// Usage:
//
//	server.Handle("/game", func(c *wt.Context) {
//	    ticker := wt.NewTicker(c, 50*time.Millisecond, func() []byte {
//	        return getGameState()
//	    })
//	    defer ticker.Stop()
//	    // ... handle other streams
//	})
type Ticker struct {
	ctx    *Context
	ticker *time.Ticker
	done   chan struct{}
}

// NewTicker starts sending datagrams at the given interval.
// The getData function is called each tick to produce the datagram payload.
// Return nil to skip a tick (no datagram sent).
func NewTicker(c *Context, interval time.Duration, getData func() []byte) *Ticker {
	t := &Ticker{
		ctx:    c,
		ticker: time.NewTicker(interval),
		done:   make(chan struct{}),
	}

	go func() {
		for {
			select {
			case <-t.ticker.C:
				data := getData()
				if data != nil {
					_ = c.SendDatagram(data)
				}
			case <-t.done:
				return
			case <-c.Context().Done():
				return
			}
		}
	}()

	return t
}

// Stop stops the ticker.
func (t *Ticker) Stop() {
	if t.ticker != nil {
		t.ticker.Stop()
	}
	select {
	case <-t.done:
	default:
		close(t.done)
	}
}

// KeepAlive sends periodic datagrams to keep the QUIC connection alive.
// This prevents NAT mappings from expiring (typically 20-30 seconds for UDP).
//
// Usage:
//
//	server.Handle("/app", func(c *wt.Context) {
//	    stop := wt.KeepAlive(c, 15*time.Second)
//	    defer stop()
//	    // ... handle session
//	})
//
// Returns a stop function that cancels the keep-alive.
func KeepAlive(c *Context, interval time.Duration) func() {
	if interval <= 0 {
		interval = 15 * time.Second
	}

	done := make(chan struct{})
	ping := []byte{0x00} // minimal keep-alive packet

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				_ = c.SendDatagram(ping)
			case <-done:
				return
			case <-c.Context().Done():
				return
			}
		}
	}()

	return func() {
		select {
		case <-done:
		default:
			close(done)
		}
	}
}

// DefaultKeepAliveInterval is 15 seconds, chosen to be well under
// typical UDP NAT timeout of 20-30 seconds.
const DefaultKeepAliveInterval = 15 * time.Second

// DatagramBatcher collects datagrams and sends them in batches.
// This reduces the number of individual QUIC packets, improving throughput
// at the cost of slightly increased latency.
//
// Useful for high-frequency updates (game state, sensor readings)
// where batching many small messages into fewer packets improves efficiency.
type DatagramBatcher struct {
	ctx      *Context
	mu       sync.Mutex
	buf      [][]byte
	maxSize  int
	interval time.Duration
	done     chan struct{}
	encode   func(batch [][]byte) []byte
}

// BatcherOption configures the DatagramBatcher.
type BatcherOption func(*DatagramBatcher)

// WithBatchSize sets the maximum number of datagrams per batch (default: 10).
func WithBatchSize(n int) BatcherOption {
	return func(b *DatagramBatcher) { b.maxSize = n }
}

// WithBatchInterval sets the maximum time to wait before flushing (default: 16ms ≈ 60Hz).
func WithBatchInterval(d time.Duration) BatcherOption {
	return func(b *DatagramBatcher) { b.interval = d }
}

// WithBatchEncoder sets a custom function to encode a batch into a single datagram.
// Default: concatenates with 2-byte length prefix per item.
func WithBatchEncoder(fn func(batch [][]byte) []byte) BatcherOption {
	return func(b *DatagramBatcher) { b.encode = fn }
}

// NewDatagramBatcher creates a batcher that flushes either when maxSize
// is reached or interval elapses, whichever comes first.
func NewDatagramBatcher(c *Context, opts ...BatcherOption) *DatagramBatcher {
	b := &DatagramBatcher{
		ctx:      c,
		maxSize:  10,
		interval: 16 * time.Millisecond,
		done:     make(chan struct{}),
		encode:   defaultBatchEncode,
	}
	for _, opt := range opts {
		opt(b)
	}
	b.buf = make([][]byte, 0, b.maxSize)
	go b.flushLoop()
	return b
}

// Add adds a datagram to the current batch.
// If the batch is full, it's flushed immediately.
func (b *DatagramBatcher) Add(data []byte) {
	b.mu.Lock()
	b.buf = append(b.buf, data)
	if len(b.buf) >= b.maxSize {
		batch := b.buf
		b.buf = make([][]byte, 0, b.maxSize)
		b.mu.Unlock()
		b.send(batch)
		return
	}
	b.mu.Unlock()
}

// Flush sends any buffered datagrams immediately.
func (b *DatagramBatcher) Flush() {
	b.mu.Lock()
	if len(b.buf) == 0 {
		b.mu.Unlock()
		return
	}
	batch := b.buf
	b.buf = make([][]byte, 0, b.maxSize)
	b.mu.Unlock()
	b.send(batch)
}

// Close stops the batcher and flushes remaining data.
func (b *DatagramBatcher) Close() {
	close(b.done)
	b.Flush()
}

func (b *DatagramBatcher) flushLoop() {
	ticker := time.NewTicker(b.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			b.Flush()
		case <-b.done:
			return
		}
	}
}

func (b *DatagramBatcher) send(batch [][]byte) {
	if len(batch) == 0 {
		return
	}
	encoded := b.encode(batch)
	_ = b.ctx.SendDatagram(encoded)
}

// defaultBatchEncode concatenates messages with 2-byte big-endian length prefixes.
// Format: [len1 uint16][data1][len2 uint16][data2]...
func defaultBatchEncode(batch [][]byte) []byte {
	totalLen := 0
	for _, msg := range batch {
		totalLen += 2 + len(msg)
	}

	out := make([]byte, 0, totalLen)
	for _, msg := range batch {
		out = append(out, byte(len(msg)>>8), byte(len(msg)))
		out = append(out, msg...)
	}
	return out
}

// DecodeBatch decodes a batch-encoded datagram into individual messages.
// Pre-allocates the result slice based on estimated message count.
func DecodeBatch(data []byte) [][]byte {
	if len(data) < 2 {
		return nil
	}
	// Estimate: at least 1 message per 8 bytes (2 header + min 6 payload)
	estimated := len(data) / 8
	if estimated < 1 {
		estimated = 1
	}
	msgs := make([][]byte, 0, estimated)
	for len(data) >= 2 {
		msgLen := int(data[0])<<8 | int(data[1])
		data = data[2:]
		if msgLen > len(data) {
			break
		}
		msgs = append(msgs, data[:msgLen])
		data = data[msgLen:]
	}
	return msgs
}
