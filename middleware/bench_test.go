package middleware

import (
	"bytes"
	"context"
	"testing"
)

func BenchmarkGzipCompress1KB(b *testing.B) {
	c := NewGzipCompressor()
	data := bytes.Repeat([]byte("benchmark data for compression "), 33) // ~1KB

	b.ResetTimer()
	b.SetBytes(int64(len(data)))
	for b.Loop() {
		c.Compress(data)
	}
}

func BenchmarkGzipDecompress1KB(b *testing.B) {
	c := NewGzipCompressor()
	data := bytes.Repeat([]byte("benchmark data for compression "), 33)
	compressed, _ := c.Compress(data)

	b.ResetTimer()
	b.SetBytes(int64(len(compressed)))
	for b.Loop() {
		c.Decompress(compressed)
	}
}

func BenchmarkDeflateCompress1KB(b *testing.B) {
	c := NewDeflateCompressor()
	data := bytes.Repeat([]byte("benchmark data for compression "), 33)

	b.ResetTimer()
	b.SetBytes(int64(len(data)))
	for b.Loop() {
		c.Compress(data)
	}
}

func BenchmarkNoopTracer(b *testing.B) {
	tracer := NoopTracer{}
	ctx := context.Background()

	b.ResetTimer()
	for b.Loop() {
		_, span := tracer.StartSpan(ctx, "test")
		span.SetAttribute("key", "value")
		span.End()
	}
}

func BenchmarkGenerateRequestID(b *testing.B) {
	for b.Loop() {
		generateRequestID()
	}
}
