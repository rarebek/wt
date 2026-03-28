package wt

import (
	"bytes"
	"compress/gzip"
	"io"
	"sync"
)

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
