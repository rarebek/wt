package wt

import (
	"testing"
	"time"
)

func BenchmarkResumeStoreSave(b *testing.B) {
	rs := NewResumeStore(5 * time.Minute)
	ctx := &Context{store: map[string]any{"user": "alice"}}
	b.ResetTimer()
	for range b.N {
		rs.Save(ctx)
	}
}

func BenchmarkResumeStoreRestore(b *testing.B) {
	rs := NewResumeStore(5 * time.Minute)
	ctx := &Context{store: map[string]any{"user": "alice"}}
	tokens := make([]ResumeToken, b.N)
	for i := range b.N {
		tokens[i] = rs.Save(ctx)
	}
	b.ResetTimer()
	for i := range b.N {
		newCtx := &Context{store: make(map[string]any)}
		rs.Restore(newCtx, tokens[i])
	}
}
