package wt

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestEventBusOn(t *testing.T) {
	bus := NewEventBus()

	called := false
	bus.On(EventConnect, func(e Event) {
		called = true
	})

	bus.Emit(Event{Type: EventConnect})

	if !called {
		t.Error("handler was not called")
	}
}

func TestEventBusMultipleHandlers(t *testing.T) {
	bus := NewEventBus()

	var count int
	bus.On(EventConnect, func(e Event) { count++ })
	bus.On(EventConnect, func(e Event) { count++ })
	bus.On(EventConnect, func(e Event) { count++ })

	bus.Emit(Event{Type: EventConnect})

	if count != 3 {
		t.Errorf("expected 3 handler calls, got %d", count)
	}
}

func TestEventBusDifferentTypes(t *testing.T) {
	bus := NewEventBus()

	connectCalled := false
	disconnectCalled := false

	bus.On(EventConnect, func(e Event) { connectCalled = true })
	bus.On(EventDisconnect, func(e Event) { disconnectCalled = true })

	bus.Emit(Event{Type: EventConnect})

	if !connectCalled {
		t.Error("connect handler not called")
	}
	if disconnectCalled {
		t.Error("disconnect handler should not be called")
	}
}

func TestEventBusAsync(t *testing.T) {
	bus := NewEventBus()

	var called atomic.Bool
	bus.On(EventConnect, func(e Event) {
		called.Store(true)
	})

	bus.EmitAsync(Event{Type: EventConnect})

	time.Sleep(50 * time.Millisecond)
	if !called.Load() {
		t.Error("async handler was not called")
	}
}

func TestEventTypeString(t *testing.T) {
	tests := []struct {
		et   EventType
		want string
	}{
		{EventConnect, "connect"},
		{EventDisconnect, "disconnect"},
		{EventJoinRoom, "join_room"},
		{EventLeaveRoom, "leave_room"},
		{EventType(99), "unknown"},
	}

	for _, tt := range tests {
		if got := tt.et.String(); got != tt.want {
			t.Errorf("EventType(%d).String() = %q, want %q", tt.et, got, tt.want)
		}
	}
}

func TestEventBusRoomEvent(t *testing.T) {
	bus := NewEventBus()

	var receivedRoom string
	bus.On(EventJoinRoom, func(e Event) {
		receivedRoom = e.Room
	})

	bus.Emit(Event{Type: EventJoinRoom, Room: "lobby"})

	if receivedRoom != "lobby" {
		t.Errorf("expected room 'lobby', got %q", receivedRoom)
	}
}
