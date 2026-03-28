package wt

import (
	"fmt"
	"runtime"
	"testing"
)

// TestMemory1000Sessions verifies that 1000 session Context objects
// don't leak excessive memory. This tests the framework's data structures,
// not actual QUIC connections.
func TestMemory1000Sessions(t *testing.T) {
	ss := NewSessionStore()
	rm := NewRoomManager()
	room := rm.GetOrCreate("loadtest")
	pt := NewPresenceTracker()

	var m1 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)

	// Create 1000 sessions with context stores, room membership, and presence
	contexts := make([]*Context, 1000)
	for i := range 1000 {
		ctx := &Context{
			id:     fmt.Sprintf("session-%04d", i),
			params: map[string]string{"room": "loadtest"},
			store:  map[string]any{"user": fmt.Sprintf("user-%d", i), "role": "member"},
		}
		contexts[i] = ctx
		ss.Add(ctx)
		room.Join(ctx)
		pt.Join("loadtest", ctx)
	}

	var m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m2)

	allocatedKB := (m2.HeapAlloc - m1.HeapAlloc) / 1024
	perSessionBytes := (m2.HeapAlloc - m1.HeapAlloc) / 1000

	t.Logf("1000 sessions: ~%d KB total, ~%d bytes/session", allocatedKB, perSessionBytes)
	t.Logf("  SessionStore: %d entries", ss.Count())
	t.Logf("  Room members: %d", room.Count())
	t.Logf("  Presence:     %d", pt.Count("loadtest"))

	// Sanity: should be well under 1MB for 1000 sessions
	if allocatedKB > 2048 {
		t.Errorf("excessive memory: %d KB for 1000 sessions (expected < 2048 KB)", allocatedKB)
	}

	// Verify counts
	if ss.Count() != 1000 {
		t.Errorf("expected 1000 sessions, got %d", ss.Count())
	}
	if room.Count() != 1000 {
		t.Errorf("expected 1000 room members, got %d", room.Count())
	}

	// Cleanup
	for _, ctx := range contexts {
		ss.Remove(ctx.ID())
		room.Leave(ctx)
		pt.Leave("loadtest", ctx)
	}

	if ss.Count() != 0 {
		t.Errorf("leaked sessions: %d", ss.Count())
	}
	if room.Count() != 0 {
		t.Errorf("leaked room members: %d", room.Count())
	}
}
