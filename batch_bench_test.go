package wt

import "testing"

func BenchmarkBatchEncode8(b *testing.B) {
	batch := makeBatch(8, 32)
	b.ResetTimer()
	for b.Loop() { defaultBatchEncode(batch) }
}

func BenchmarkBatchEncode16(b *testing.B) {
	batch := makeBatch(16, 32)
	b.ResetTimer()
	for b.Loop() { defaultBatchEncode(batch) }
}

func BenchmarkBatchEncode32(b *testing.B) {
	batch := makeBatch(32, 32)
	b.ResetTimer()
	for b.Loop() { defaultBatchEncode(batch) }
}

func BenchmarkBatchEncode64(b *testing.B) {
	batch := makeBatch(64, 32)
	b.ResetTimer()
	for b.Loop() { defaultBatchEncode(batch) }
}

func BenchmarkBatchDecode8(b *testing.B) {
	encoded := defaultBatchEncode(makeBatch(8, 32))
	b.ResetTimer()
	for b.Loop() { DecodeBatch(encoded) }
}

func BenchmarkBatchDecode16(b *testing.B) {
	encoded := defaultBatchEncode(makeBatch(16, 32))
	b.ResetTimer()
	for b.Loop() { DecodeBatch(encoded) }
}

func BenchmarkBatchDecode32(b *testing.B) {
	encoded := defaultBatchEncode(makeBatch(32, 32))
	b.ResetTimer()
	for b.Loop() { DecodeBatch(encoded) }
}

func BenchmarkBatchDecode64(b *testing.B) {
	encoded := defaultBatchEncode(makeBatch(64, 32))
	b.ResetTimer()
	for b.Loop() { DecodeBatch(encoded) }
}

func makeBatch(count, msgSize int) [][]byte {
	batch := make([][]byte, count)
	for i := range batch {
		batch[i] = make([]byte, msgSize)
		for j := range batch[i] {
			batch[i][j] = byte(j)
		}
	}
	return batch
}
