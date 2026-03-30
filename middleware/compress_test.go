package middleware

import (
	"bytes"
	"testing"
)

func TestGzipCompressor(t *testing.T) {
	c := NewGzipCompressor()

	if c.Name() != "gzip" {
		t.Errorf("expected name 'gzip', got %q", c.Name())
	}

	original := []byte("hello world this is a test message for compression")

	compressed, err := c.Compress(original)
	if err != nil {
		t.Fatalf("compress error: %v", err)
	}

	decompressed, err := c.Decompress(compressed)
	if err != nil {
		t.Fatalf("decompress error: %v", err)
	}

	if !bytes.Equal(original, decompressed) {
		t.Errorf("decompressed data doesn't match original")
	}
}

func TestDeflateCompressor(t *testing.T) {
	c := NewDeflateCompressor()

	if c.Name() != "deflate" {
		t.Errorf("expected name 'deflate', got %q", c.Name())
	}

	original := []byte("this is another test message that should be compressed and decompressed correctly")

	compressed, err := c.Compress(original)
	if err != nil {
		t.Fatalf("compress error: %v", err)
	}

	decompressed, err := c.Decompress(compressed)
	if err != nil {
		t.Fatalf("decompress error: %v", err)
	}

	if !bytes.Equal(original, decompressed) {
		t.Errorf("decompressed data doesn't match original")
	}
}

func TestGzipCompressionRatio(t *testing.T) {
	c := NewGzipCompressor()

	// Highly compressible data
	original := bytes.Repeat([]byte("AAAA"), 1000)

	compressed, err := c.Compress(original)
	if err != nil {
		t.Fatalf("compress error: %v", err)
	}

	ratio := float64(len(compressed)) / float64(len(original))
	if ratio > 0.1 {
		t.Errorf("compression ratio too high for repetitive data: %.2f (expected < 0.1)", ratio)
	}
}

func TestGzipEmptyData(t *testing.T) {
	c := NewGzipCompressor()

	compressed, err := c.Compress([]byte{})
	if err != nil {
		t.Fatalf("compress error: %v", err)
	}

	decompressed, err := c.Decompress(compressed)
	if err != nil {
		t.Fatalf("decompress error: %v", err)
	}

	if len(decompressed) != 0 {
		t.Errorf("expected empty decompressed data, got %d bytes", len(decompressed))
	}
}

func BenchmarkGzipCompress(b *testing.B) {
	c := NewGzipCompressor()
	data := bytes.Repeat([]byte("hello world "), 100)

	b.ResetTimer()
	b.SetBytes(int64(len(data)))
	for b.Loop() {
		c.Compress(data)
	}
}

func BenchmarkGzipDecompress(b *testing.B) {
	c := NewGzipCompressor()
	data := bytes.Repeat([]byte("hello world "), 100)
	compressed, _ := c.Compress(data)

	b.ResetTimer()
	b.SetBytes(int64(len(compressed)))
	for b.Loop() {
		c.Decompress(compressed)
	}
}
