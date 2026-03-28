package wt

import (
	"sync"
	"time"
)

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
