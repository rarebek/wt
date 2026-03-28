package wt

import (
	"fmt"
	"testing"
)

func TestSessionStore(t *testing.T) {
	ss := NewSessionStore()

	if ss.Count() != 0 {
		t.Errorf("expected 0 sessions, got %d", ss.Count())
	}

	// We can't create real Context objects without a webtransport.Session,
	// but we can test the store operations with the map directly.
	// For unit testing, we'll test the data structure logic.
}

func TestRoomManager(t *testing.T) {
	rm := NewRoomManager()

	room := rm.GetOrCreate("lobby")
	if room.Name() != "lobby" {
		t.Errorf("expected room name 'lobby', got %q", room.Name())
	}

	if room.Count() != 0 {
		t.Errorf("expected 0 members, got %d", room.Count())
	}

	// GetOrCreate returns same room
	room2 := rm.GetOrCreate("lobby")
	if room2 != room {
		t.Error("expected same room instance")
	}

	// Different room
	room3 := rm.GetOrCreate("game")
	if room3 == room {
		t.Error("expected different room instance")
	}

	rooms := rm.Rooms()
	if len(rooms) != 2 {
		t.Errorf("expected 2 rooms, got %d", len(rooms))
	}

	rm.Remove("game")
	rooms = rm.Rooms()
	if len(rooms) != 1 {
		t.Errorf("expected 1 room after removal, got %d", len(rooms))
	}
}

func TestSessionStoreFindByValue(t *testing.T) {
	ss := NewSessionStore()

	// Add contexts with different user values
	for i, user := range []string{"alice", "bob", "alice", "charlie", "alice"} {
		ctx := &Context{
			id:    fmt.Sprintf("session-%d", i),
			store: map[string]any{"user": user},
		}
		ss.Add(ctx)
	}

	// Find alice's sessions
	aliceSessions := ss.FindByValue("user", "alice")
	if len(aliceSessions) != 3 {
		t.Errorf("expected 3 alice sessions, got %d", len(aliceSessions))
	}

	bobSessions := ss.FindByValue("user", "bob")
	if len(bobSessions) != 1 {
		t.Errorf("expected 1 bob session, got %d", len(bobSessions))
	}

	noneSessions := ss.FindByValue("user", "nobody")
	if len(noneSessions) != 0 {
		t.Errorf("expected 0 sessions for nobody, got %d", len(noneSessions))
	}
}

func TestSessionStoreIDs(t *testing.T) {
	ss := NewSessionStore()

	for i := range 3 {
		ctx := &Context{
			id:    fmt.Sprintf("s-%d", i),
			store: make(map[string]any),
		}
		ss.Add(ctx)
	}

	ids := ss.IDs()
	if len(ids) != 3 {
		t.Errorf("expected 3 IDs, got %d", len(ids))
	}
}

func TestRoomManagerGet(t *testing.T) {
	rm := NewRoomManager()

	_, ok := rm.Get("nonexistent")
	if ok {
		t.Error("expected false for nonexistent room")
	}

	rm.GetOrCreate("test")
	room, ok := rm.Get("test")
	if !ok {
		t.Error("expected true for existing room")
	}
	if room.Name() != "test" {
		t.Errorf("expected room name 'test', got %q", room.Name())
	}
}
