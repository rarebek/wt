package middleware

import "testing"

func TestSessionLog(t *testing.T) {
	sl := &SessionLog{}
	sl.Add("connect", "127.0.0.1")
	sl.Add("auth", "user=alice")
	sl.Add("join_room", "lobby")

	if sl.Len() != 3 {
		t.Errorf("expected 3 entries, got %d", sl.Len())
	}

	entries := sl.Entries()
	if entries[0].Event != "connect" {
		t.Error("first entry should be connect")
	}
	if entries[1].Details != "user=alice" {
		t.Errorf("expected details 'user=alice', got %q", entries[1].Details)
	}
}
