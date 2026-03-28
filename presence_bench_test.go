package wt

import (
	"fmt"
	"testing"
)

func BenchmarkPresenceJoin(b *testing.B) {
	pt := NewPresenceTracker()

	contexts := make([]*Context, b.N)
	for i := range contexts {
		contexts[i] = &Context{
			id:    fmt.Sprintf("s-%d", i),
			store: map[string]any{"user": fmt.Sprintf("user-%d", i)},
		}
	}

	b.ResetTimer()
	for i := range b.N {
		pt.Join("room", contexts[i])
	}
}

func BenchmarkPresenceGetPresence(b *testing.B) {
	pt := NewPresenceTracker()

	for i := range 100 {
		ctx := &Context{
			id:    fmt.Sprintf("s-%d", i),
			store: map[string]any{"user": fmt.Sprintf("user-%d", i)},
		}
		pt.Join("room", ctx)
	}

	b.ResetTimer()
	for b.Loop() {
		pt.GetPresence("room")
	}
}
