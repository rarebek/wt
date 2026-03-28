package wt

import "sync"

// EventType represents a session lifecycle event.
type EventType int

const (
	EventConnect    EventType = iota // Session opened
	EventDisconnect                  // Session closed
	EventJoinRoom                    // Session joined a room
	EventLeaveRoom                   // Session left a room
)

func (e EventType) String() string {
	switch e {
	case EventConnect:
		return "connect"
	case EventDisconnect:
		return "disconnect"
	case EventJoinRoom:
		return "join_room"
	case EventLeaveRoom:
		return "leave_room"
	default:
		return "unknown"
	}
}

// Event represents a session lifecycle event.
type Event struct {
	Type    EventType
	Session *Context
	Room    string // Only set for room events
}

// EventHandler is a callback for session events.
type EventHandler func(Event)

// EventBus provides a publish/subscribe system for session lifecycle events.
type EventBus struct {
	mu       sync.RWMutex
	handlers map[EventType][]EventHandler
}

// NewEventBus creates a new event bus.
func NewEventBus() *EventBus {
	return &EventBus{
		handlers: make(map[EventType][]EventHandler),
	}
}

// On registers a handler for the given event type.
func (eb *EventBus) On(eventType EventType, handler EventHandler) {
	eb.mu.Lock()
	eb.handlers[eventType] = append(eb.handlers[eventType], handler)
	eb.mu.Unlock()
}

// Emit publishes an event to all registered handlers.
// Handlers are called synchronously in registration order.
func (eb *EventBus) Emit(event Event) {
	eb.mu.RLock()
	handlers := eb.handlers[event.Type]
	eb.mu.RUnlock()

	for _, h := range handlers {
		h(event)
	}
}

// EmitAsync publishes an event to all registered handlers asynchronously.
func (eb *EventBus) EmitAsync(event Event) {
	eb.mu.RLock()
	handlers := eb.handlers[event.Type]
	eb.mu.RUnlock()

	for _, h := range handlers {
		go h(event)
	}
}
