package wt

import (
	"testing"
)

func TestDefaultBatchEncode(t *testing.T) {
	batch := [][]byte{
		[]byte("hello"),
		[]byte("world"),
		[]byte("!"),
	}

	encoded := defaultBatchEncode(batch)

	// Decode and verify
	decoded := DecodeBatch(encoded)
	if len(decoded) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(decoded))
	}
	if string(decoded[0]) != "hello" {
		t.Errorf("msg[0] = %q, want 'hello'", decoded[0])
	}
	if string(decoded[1]) != "world" {
		t.Errorf("msg[1] = %q, want 'world'", decoded[1])
	}
	if string(decoded[2]) != "!" {
		t.Errorf("msg[2] = %q, want '!'", decoded[2])
	}
}

func TestDecodeBatchEmpty(t *testing.T) {
	msgs := DecodeBatch([]byte{})
	if len(msgs) != 0 {
		t.Errorf("expected 0 messages, got %d", len(msgs))
	}
}

func TestDecodeBatchSingle(t *testing.T) {
	batch := defaultBatchEncode([][]byte{[]byte("single")})
	msgs := DecodeBatch(batch)
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	if string(msgs[0]) != "single" {
		t.Errorf("expected 'single', got %q", msgs[0])
	}
}

func TestDecodeBatchTruncated(t *testing.T) {
	// Truncated data should not panic
	msgs := DecodeBatch([]byte{0x00, 0x05, 0x61, 0x62}) // claims length 5 but only 2 bytes
	if len(msgs) != 0 {
		t.Errorf("expected 0 messages from truncated data, got %d", len(msgs))
	}
}

func BenchmarkBatchEncode(b *testing.B) {
	batch := make([][]byte, 10)
	for i := range batch {
		batch[i] = make([]byte, 64)
	}

	b.ResetTimer()
	for b.Loop() {
		defaultBatchEncode(batch)
	}
}

func BenchmarkBatchDecode(b *testing.B) {
	batch := make([][]byte, 10)
	for i := range batch {
		batch[i] = make([]byte, 64)
	}
	encoded := defaultBatchEncode(batch)

	b.ResetTimer()
	for b.Loop() {
		DecodeBatch(encoded)
	}
}
