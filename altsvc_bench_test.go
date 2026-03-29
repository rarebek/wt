package wt

import "testing"

func BenchmarkAltSvcHeader(b *testing.B) {
	for b.Loop() {
		AltSvcHeader(4433)
	}
}

func BenchmarkJoinPath(b *testing.B) {
	for b.Loop() {
		JoinPath("/api", "/v1", "/users")
	}
}

func BenchmarkHash(b *testing.B) {
	data := []byte("benchmark hash input")
	b.ResetTimer()
	for b.Loop() {
		Hash(data)
	}
}
