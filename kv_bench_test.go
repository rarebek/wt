package wt

import "testing"

func BenchmarkKVSyncSet(b *testing.B) {
	kv := NewKVSync()
	b.ResetTimer()
	for i := range b.N {
		kv.Set("key", i)
	}
}

func BenchmarkKVSyncGet(b *testing.B) {
	kv := NewKVSync()
	kv.Set("key", 42)
	b.ResetTimer()
	for b.Loop() {
		var v int
		kv.Get("key", &v)
	}
}
