package wt

import (
	"testing"

	"github.com/rarebek/wt/codec"
)

func TestNewTypedRoom(t *testing.T) {
	rm := NewRoomManager()
	room := rm.GetOrCreate("test")

	type ChatMsg struct {
		User string `json:"user"`
		Text string `json:"text"`
	}

	tr := NewTypedRoom[ChatMsg](room, codec.JSON{})

	if tr.Room() != room {
		t.Error("expected same room reference")
	}
}

func TestTypedRoomBroadcastEncodes(t *testing.T) {
	rm := NewRoomManager()
	room := rm.GetOrCreate("test")

	type Msg struct {
		Value int `json:"value"`
	}

	tr := NewTypedRoom[Msg](room, codec.JSON{})

	// Broadcast with no members should not error
	err := tr.Broadcast(Msg{Value: 42})
	if err != nil {
		t.Errorf("broadcast with no members should not error: %v", err)
	}

	err = tr.BroadcastExcept(Msg{Value: 99}, "nobody")
	if err != nil {
		t.Errorf("broadcast except with no members should not error: %v", err)
	}
}
