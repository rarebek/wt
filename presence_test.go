package wt

import (
	"testing"
)

func TestPresenceTrackerJoinLeave(t *testing.T) {
	pt := NewPresenceTracker()

	ctx := &Context{
		id:    "session-1",
		store: map[string]any{"user": "alice"},
	}

	pt.Join("room1", ctx)

	presence := pt.GetPresence("room1")
	if len(presence) != 1 {
		t.Fatalf("expected 1 present, got %d", len(presence))
	}
	if presence[0].UserID != "alice" {
		t.Errorf("expected user 'alice', got %q", presence[0].UserID)
	}
	if presence[0].Status != "online" {
		t.Errorf("expected status 'online', got %q", presence[0].Status)
	}

	pt.Leave("room1", ctx)

	presence = pt.GetPresence("room1")
	if len(presence) != 0 {
		t.Errorf("expected 0 present after leave, got %d", len(presence))
	}
}

func TestPresenceTrackerCount(t *testing.T) {
	pt := NewPresenceTracker()

	for i := range 5 {
		ctx := &Context{
			id:    string(rune('a' + i)),
			store: make(map[string]any),
		}
		pt.Join("room", ctx)
	}

	if pt.Count("room") != 5 {
		t.Errorf("expected 5, got %d", pt.Count("room"))
	}
	if pt.Count("empty") != 0 {
		t.Errorf("expected 0 for empty room, got %d", pt.Count("empty"))
	}
}

func TestPresenceTrackerUpdateStatus(t *testing.T) {
	pt := NewPresenceTracker()

	ctx := &Context{
		id:    "s1",
		store: make(map[string]any),
	}
	pt.Join("room", ctx)
	pt.UpdateStatus("room", "s1", "typing")

	presence := pt.GetPresence("room")
	if presence[0].Status != "typing" {
		t.Errorf("expected 'typing', got %q", presence[0].Status)
	}
}

func TestPresenceTrackerSetMetadata(t *testing.T) {
	pt := NewPresenceTracker()

	ctx := &Context{
		id:    "s1",
		store: make(map[string]any),
	}
	pt.Join("room", ctx)
	pt.SetMetadata("room", "s1", map[string]any{"cursor": 42})

	presence := pt.GetPresence("room")
	if presence[0].Metadata["cursor"] != 42 {
		t.Errorf("expected cursor=42, got %v", presence[0].Metadata["cursor"])
	}
}

func TestPresenceTrackerOnChange(t *testing.T) {
	pt := NewPresenceTracker()

	var events []string
	pt.OnChange(func(room string, info PresenceInfo, event string) {
		events = append(events, event)
	})

	ctx := &Context{
		id:    "s1",
		store: make(map[string]any),
	}
	pt.Join("room", ctx)
	pt.UpdateStatus("room", "s1", "idle")
	pt.Leave("room", ctx)

	if len(events) != 3 {
		t.Fatalf("expected 3 events, got %d: %v", len(events), events)
	}
	if events[0] != "join" || events[1] != "update" || events[2] != "leave" {
		t.Errorf("wrong events: %v", events)
	}
}

func TestPresenceTrackerJSON(t *testing.T) {
	pt := NewPresenceTracker()

	ctx := &Context{
		id:    "s1",
		store: map[string]any{"user": "bob"},
	}
	pt.Join("room", ctx)

	data := pt.GetPresenceJSON("room")
	if len(data) == 0 {
		t.Error("expected non-empty JSON")
	}
}
