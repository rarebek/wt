package wt

import "testing"

func TestRoomWithHistoryBroadcastRecords(t *testing.T) {
	rm := NewRoomManager()
	room := rm.GetOrCreate("chat")
	rwh := NewRoomWithHistory(room, 50)

	rwh.BroadcastAndRecord("user1", []byte("hello"))
	rwh.BroadcastAndRecord("user2", []byte("world"))

	if rwh.HistorySize() != 2 {
		t.Errorf("expected 2 messages, got %d", rwh.HistorySize())
	}

	history := rwh.History()
	if string(history[0].Data) != "hello" {
		t.Errorf("expected 'hello', got %q", history[0].Data)
	}
	if history[0].SenderID != "user1" {
		t.Errorf("expected sender 'user1', got %q", history[0].SenderID)
	}
	if string(history[1].Data) != "world" {
		t.Errorf("expected 'world', got %q", history[1].Data)
	}
}

func TestRoomWithHistoryOverflow(t *testing.T) {
	rm := NewRoomManager()
	room := rm.GetOrCreate("test")
	rwh := NewRoomWithHistory(room, 3)

	for i := range 5 {
		rwh.BroadcastAndRecord("user", []byte{byte(i)})
	}

	if rwh.HistorySize() != 3 {
		t.Errorf("expected 3 (capacity), got %d", rwh.HistorySize())
	}

	history := rwh.History()
	// Should contain 2, 3, 4 (oldest overwritten)
	if history[0].Data[0] != 2 || history[1].Data[0] != 3 || history[2].Data[0] != 4 {
		t.Errorf("expected [2,3,4], got [%d,%d,%d]",
			history[0].Data[0], history[1].Data[0], history[2].Data[0])
	}
}

func TestRoomWithHistoryClear(t *testing.T) {
	rm := NewRoomManager()
	room := rm.GetOrCreate("test")
	rwh := NewRoomWithHistory(room, 10)

	rwh.BroadcastAndRecord("u", []byte("a"))
	rwh.ClearHistory()

	if rwh.HistorySize() != 0 {
		t.Errorf("expected 0 after clear, got %d", rwh.HistorySize())
	}
}
