package wt

import "testing"

func TestRoomHas(t *testing.T) {
	rm := NewRoomManager()
	room := rm.GetOrCreate("test")
	ctx := &Context{id: "s1", store: make(map[string]any)}

	if room.Has("s1") {
		t.Error("should not have s1 before join")
	}

	room.Join(ctx)
	if !room.Has("s1") {
		t.Error("should have s1 after join")
	}

	room.Leave(ctx)
	if room.Has("s1") {
		t.Error("should not have s1 after leave")
	}
}

func TestRoomFilterMembers(t *testing.T) {
	rm := NewRoomManager()
	room := rm.GetOrCreate("test")

	for i := range 10 {
		ctx := &Context{
			id:    string(rune('a' + i)),
			store: map[string]any{"vip": i < 3},
		}
		room.Join(ctx)
	}

	vips := room.FilterMembers(func(c *Context) bool {
		v, _ := c.Get("vip")
		return v == true
	})

	if len(vips) != 3 {
		t.Errorf("expected 3 vips, got %d", len(vips))
	}
}
