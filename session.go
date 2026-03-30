package wt

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"iter"
	"log/slog"
	"sync"
	"time"

	"github.com/rarebek/wt/codec"
)

// SessionStore tracks active sessions and provides lookup/broadcast capabilities.
type SessionStore struct {
	mu       sync.RWMutex
	sessions map[string]*Context
}

// NewSessionStore creates a new SessionStore.
func NewSessionStore() *SessionStore {
	return &SessionStore{
		sessions: make(map[string]*Context),
	}
}

// Add registers a session.
func (ss *SessionStore) Add(ctx *Context) {
	ss.mu.Lock()
	ss.sessions[ctx.ID()] = ctx
	ss.mu.Unlock()
}

// Remove unregisters a session.
func (ss *SessionStore) Remove(id string) {
	ss.mu.Lock()
	delete(ss.sessions, id)
	ss.mu.Unlock()
}

// Get returns a session by ID.
func (ss *SessionStore) Get(id string) (*Context, bool) {
	ss.mu.RLock()
	ctx, ok := ss.sessions[id]
	ss.mu.RUnlock()
	return ctx, ok
}

// Count returns the number of active sessions.
func (ss *SessionStore) Count() int {
	ss.mu.RLock()
	n := len(ss.sessions)
	ss.mu.RUnlock()
	return n
}

// Each iterates over all active sessions.
// The callback should not block for long.
func (ss *SessionStore) Each(fn func(*Context)) {
	ss.mu.RLock()
	defer ss.mu.RUnlock()
	for _, ctx := range ss.sessions {
		fn(ctx)
	}
}

// FindByValue returns all sessions where the given key matches the given value.
// Useful for finding all sessions for a specific user, role, etc.
func (ss *SessionStore) FindByValue(key string, value any) []*Context {
	ss.mu.RLock()
	defer ss.mu.RUnlock()
	var matches []*Context
	for _, ctx := range ss.sessions {
		v, ok := ctx.Get(key)
		if ok && v == value {
			matches = append(matches, ctx)
		}
	}
	return matches
}

// IDs returns all active session IDs.
func (ss *SessionStore) IDs() []string {
	ss.mu.RLock()
	defer ss.mu.RUnlock()
	ids := make([]string, 0, len(ss.sessions))
	for id := range ss.sessions {
		ids = append(ids, id)
	}
	return ids
}

// Broadcast sends a datagram to all active sessions.
func (ss *SessionStore) Broadcast(data []byte) {
	ss.mu.RLock()
	defer ss.mu.RUnlock()
	for _, ctx := range ss.sessions {
		_ = ctx.SendDatagram(data)
	}
}

// CloseAll closes all active sessions.
func (ss *SessionStore) CloseAll() {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	for _, ctx := range ss.sessions {
		_ = ctx.Close()
	}
}

// Room represents a named group of sessions for pub/sub messaging.
type Room struct {
	mu      sync.RWMutex
	name    string
	members map[string]*Context
	onJoin  func(*Context)
	onLeave func(*Context)
}

// RoomManager manages named rooms.
type RoomManager struct {
	mu    sync.RWMutex
	rooms map[string]*Room
}

// NewRoomManager creates a new RoomManager.
func NewRoomManager() *RoomManager {
	return &RoomManager{
		rooms: make(map[string]*Room),
	}
}

// GetOrCreate returns a room by name, creating it if it doesn't exist.
func (rm *RoomManager) GetOrCreate(name string) *Room {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if room, ok := rm.rooms[name]; ok {
		return room
	}
	room := &Room{
		name:    name,
		members: make(map[string]*Context),
	}
	rm.rooms[name] = room
	return room
}

// Get returns a room by name if it exists.
func (rm *RoomManager) Get(name string) (*Room, bool) {
	rm.mu.RLock()
	r, ok := rm.rooms[name]
	rm.mu.RUnlock()
	return r, ok
}

// Remove deletes a room.
func (rm *RoomManager) Remove(name string) {
	rm.mu.Lock()
	delete(rm.rooms, name)
	rm.mu.Unlock()
}

// Rooms returns all room names.
func (rm *RoomManager) Rooms() []string {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	names := make([]string, 0, len(rm.rooms))
	for name := range rm.rooms {
		names = append(names, name)
	}
	return names
}

// Join adds a session to the room.
func (r *Room) Join(ctx *Context) {
	r.mu.Lock()
	r.members[ctx.ID()] = ctx
	onJoin := r.onJoin
	r.mu.Unlock()

	if onJoin != nil {
		onJoin(ctx)
	}
}

// Leave removes a session from the room.
func (r *Room) Leave(ctx *Context) {
	r.mu.Lock()
	delete(r.members, ctx.ID())
	onLeave := r.onLeave
	r.mu.Unlock()

	if onLeave != nil {
		onLeave(ctx)
	}
}

// Members returns all sessions in the room.
func (r *Room) Members() []*Context {
	r.mu.RLock()
	defer r.mu.RUnlock()
	members := make([]*Context, 0, len(r.members))
	for _, ctx := range r.members {
		members = append(members, ctx)
	}
	return members
}

// ForEach iterates over all members without allocating a slice.
// The callback runs under a read lock — do not block for long.
func (r *Room) ForEach(fn func(*Context)) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, ctx := range r.members {
		fn(ctx)
	}
}

// Count returns the number of members in the room.
func (r *Room) Count() int {
	r.mu.RLock()
	n := len(r.members)
	r.mu.RUnlock()
	return n
}

// Name returns the room name.
func (r *Room) Name() string {
	return r.name
}

// Broadcast sends a datagram to all members in the room.
func (r *Room) Broadcast(data []byte) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, ctx := range r.members {
		_ = ctx.SendDatagram(data)
	}
}

// BroadcastExcept sends a datagram to all members except the specified session.
func (r *Room) BroadcastExcept(data []byte, excludeID string) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for id, ctx := range r.members {
		if id != excludeID {
			_ = ctx.SendDatagram(data)
		}
	}
}

// OnJoin sets a callback for when sessions join the room.
func (r *Room) OnJoin(fn func(*Context)) {
	r.mu.Lock()
	r.onJoin = fn
	r.mu.Unlock()
}

// OnLeave sets a callback for when sessions leave the room.
func (r *Room) OnLeave(fn func(*Context)) {
	r.mu.Lock()
	r.onLeave = fn
	r.mu.Unlock()
}

// Filter returns sessions matching a predicate.
func (ss *SessionStore) Filter(fn func(*Context) bool) []*Context {
	ss.mu.RLock()
	defer ss.mu.RUnlock()
	var result []*Context
	for _, ctx := range ss.sessions {
		if fn(ctx) {
			result = append(result, ctx)
		}
	}
	return result
}

// CountWhere returns the number of sessions matching a predicate.
func (ss *SessionStore) CountWhere(fn func(*Context) bool) int {
	ss.mu.RLock()
	defer ss.mu.RUnlock()
	n := 0
	for _, ctx := range ss.sessions {
		if fn(ctx) {
			n++
		}
	}
	return n
}

// Sessions returns an iterator over all active sessions.
// Use with Go 1.23+ range-over-func:
//
//	for ctx := range server.Sessions().All() {
//	    log.Println(ctx.ID())
//	}
func (ss *SessionStore) All() iter.Seq[*Context] {
	return func(yield func(*Context) bool) {
		ss.mu.RLock()
		defer ss.mu.RUnlock()
		for _, ctx := range ss.sessions {
			if !yield(ctx) {
				return
			}
		}
	}
}

// FilterMembers returns room members matching a predicate.
func (r *Room) FilterMembers(fn func(*Context) bool) []*Context {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []*Context
	for _, ctx := range r.members {
		if fn(ctx) {
			result = append(result, ctx)
		}
	}
	return result
}

// Has checks if a session is in the room.
func (r *Room) Has(sessionID string) bool {
	r.mu.RLock()
	_, ok := r.members[sessionID]
	r.mu.RUnlock()
	return ok
}

// All returns an iterator over all room names.
func (rm *RoomManager) All() iter.Seq2[string, *Room] {
	return func(yield func(string, *Room) bool) {
		rm.mu.RLock()
		defer rm.mu.RUnlock()
		for name, room := range rm.rooms {
			if !yield(name, room) {
				return
			}
		}
	}
}

// MembersIter returns an iterator over room members.
func (r *Room) MembersIter() iter.Seq[*Context] {
	return func(yield func(*Context) bool) {
		r.mu.RLock()
		defer r.mu.RUnlock()
		for _, ctx := range r.members {
			if !yield(ctx) {
				return
			}
		}
	}
}

// BroadcastStream sends a message to all room members via reliable streams.
// Unlike Broadcast (datagrams, unreliable), this guarantees delivery.
// Each member gets a new stream with the message.
func (r *Room) BroadcastStream(data []byte) {
	r.ForEach(func(c *Context) {
		go func() {
			stream, err := c.OpenStream()
			if err != nil {
				return
			}
			_ = stream.WriteMessage(data)
			stream.Close()
		}()
	})
}

// BroadcastStreamExcept sends a reliable message to all except the given session.
func (r *Room) BroadcastStreamExcept(data []byte, excludeID string) {
	r.ForEach(func(c *Context) {
		if c.ID() == excludeID {
			return
		}
		go func() {
			stream, err := c.OpenStream()
			if err != nil {
				return
			}
			_ = stream.WriteMessage(data)
			stream.Close()
		}()
	})
}

// SafeBroadcast sends a datagram to all room members, recovering from panics.
// Logs and skips any member that causes a panic (e.g., closed connection).
func (r *Room) SafeBroadcast(data []byte, logger *slog.Logger) {
	if logger == nil {
		logger = slog.Default()
	}
	r.ForEach(func(c *Context) {
		func() {
			defer func() {
				if rec := recover(); rec != nil {
					logger.Warn("broadcast panic recovered",
						"session", c.ID(),
						"room", r.Name(),
						"panic", rec,
					)
				}
			}()
			_ = c.SendDatagram(data)
		}()
	})
}

// SafeBroadcastExcept is SafeBroadcast but excludes the given session.
func (r *Room) SafeBroadcastExcept(data []byte, excludeID string, logger *slog.Logger) {
	if logger == nil {
		logger = slog.Default()
	}
	r.ForEach(func(c *Context) {
		if c.ID() == excludeID {
			return
		}
		func() {
			defer func() {
				if rec := recover(); rec != nil {
					logger.Warn("broadcast panic recovered",
						"session", c.ID(),
						"room", r.Name(),
						"panic", rec,
					)
				}
			}()
			_ = c.SendDatagram(data)
		}()
	})
}

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

// PresenceInfo represents a user's current state in a room.
type PresenceInfo struct {
	UserID    string         `json:"user_id"`
	SessionID string         `json:"session_id"`
	Status    string         `json:"status"` // "online", "idle", "away", "typing"
	Metadata  map[string]any `json:"metadata,omitempty"`
	UpdatedAt time.Time      `json:"updated_at"`
}

// PresenceTracker tracks user presence across rooms.
// Integrates with RoomManager to automatically track join/leave
// and supports custom status updates.
type PresenceTracker struct {
	mu       sync.RWMutex
	presence map[string]map[string]*PresenceInfo                // room -> sessionID -> info
	onChange func(room string, info PresenceInfo, event string) // "join", "leave", "update"
}

// NewPresenceTracker creates a new presence tracker.
func NewPresenceTracker() *PresenceTracker {
	return &PresenceTracker{
		presence: make(map[string]map[string]*PresenceInfo),
	}
}

// Join records a session joining a room.
func (pt *PresenceTracker) Join(room string, c *Context) {
	info := &PresenceInfo{
		SessionID: c.ID(),
		Status:    "online",
		UpdatedAt: time.Now(),
	}

	// Try to get user from context
	if user, ok := c.Get("user"); ok {
		if s, ok := user.(string); ok {
			info.UserID = s
		}
	}
	if info.UserID == "" {
		info.UserID = c.ID()
	}

	pt.mu.Lock()
	if pt.presence[room] == nil {
		pt.presence[room] = make(map[string]*PresenceInfo)
	}
	pt.presence[room][c.ID()] = info
	cb := pt.onChange
	pt.mu.Unlock()

	if cb != nil {
		cb(room, *info, "join")
	}
}

// Leave records a session leaving a room.
func (pt *PresenceTracker) Leave(room string, c *Context) {
	pt.mu.Lock()
	members, ok := pt.presence[room]
	var info *PresenceInfo
	if ok {
		info = members[c.ID()]
		delete(members, c.ID())
		if len(members) == 0 {
			delete(pt.presence, room)
		}
	}
	cb := pt.onChange
	pt.mu.Unlock()

	if cb != nil && info != nil {
		cb(room, *info, "leave")
	}
}

// UpdateStatus updates a session's presence status in a room.
func (pt *PresenceTracker) UpdateStatus(room, sessionID, status string) {
	pt.mu.Lock()
	members, ok := pt.presence[room]
	if !ok {
		pt.mu.Unlock()
		return
	}
	info, ok := members[sessionID]
	if !ok {
		pt.mu.Unlock()
		return
	}
	info.Status = status
	info.UpdatedAt = time.Now()
	infoCopy := *info
	cb := pt.onChange
	pt.mu.Unlock()

	if cb != nil {
		cb(room, infoCopy, "update")
	}
}

// SetMetadata sets custom metadata for a session's presence.
func (pt *PresenceTracker) SetMetadata(room, sessionID string, metadata map[string]any) {
	pt.mu.Lock()
	members, ok := pt.presence[room]
	if ok {
		if info, ok := members[sessionID]; ok {
			info.Metadata = metadata
			info.UpdatedAt = time.Now()
		}
	}
	pt.mu.Unlock()
}

// GetPresence returns all presence info for a room.
func (pt *PresenceTracker) GetPresence(room string) []PresenceInfo {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	members, ok := pt.presence[room]
	if !ok {
		return nil
	}

	result := make([]PresenceInfo, 0, len(members))
	for _, info := range members {
		result = append(result, *info)
	}
	return result
}

// GetPresenceJSON returns presence info as a JSON byte slice.
func (pt *PresenceTracker) GetPresenceJSON(room string) []byte {
	data, _ := json.Marshal(pt.GetPresence(room))
	return data
}

// Count returns the number of present sessions in a room.
func (pt *PresenceTracker) Count(room string) int {
	pt.mu.RLock()
	n := len(pt.presence[room])
	pt.mu.RUnlock()
	return n
}

// OnChange sets a callback for presence changes.
func (pt *PresenceTracker) OnChange(fn func(room string, info PresenceInfo, event string)) {
	pt.mu.Lock()
	pt.onChange = fn
	pt.mu.Unlock()
}

// Tags provides a thread-safe tagging system for sessions.
// Tags can be used to categorize sessions for targeted operations
// like "send to all sessions tagged 'premium'" or "disconnect all 'guest' sessions".
type Tags struct {
	mu   sync.RWMutex
	tags map[string]map[string]bool // tag -> set of session IDs
}

// NewTags creates a new tag registry.
func NewTags() *Tags {
	return &Tags{
		tags: make(map[string]map[string]bool),
	}
}

// Tag adds a tag to a session.
func (t *Tags) Tag(sessionID, tag string) {
	t.mu.Lock()
	if t.tags[tag] == nil {
		t.tags[tag] = make(map[string]bool)
	}
	t.tags[tag][sessionID] = true
	t.mu.Unlock()
}

// Untag removes a tag from a session.
func (t *Tags) Untag(sessionID, tag string) {
	t.mu.Lock()
	if m := t.tags[tag]; m != nil {
		delete(m, sessionID)
		if len(m) == 0 {
			delete(t.tags, tag)
		}
	}
	t.mu.Unlock()
}

// UntagAll removes all tags for a session.
func (t *Tags) UntagAll(sessionID string) {
	t.mu.Lock()
	for tag, m := range t.tags {
		delete(m, sessionID)
		if len(m) == 0 {
			delete(t.tags, tag)
		}
	}
	t.mu.Unlock()
}

// HasTag checks if a session has a specific tag.
func (t *Tags) HasTag(sessionID, tag string) bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.tags[tag][sessionID]
}

// SessionsWithTag returns all session IDs with the given tag.
func (t *Tags) SessionsWithTag(tag string) []string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	m := t.tags[tag]
	if m == nil {
		return nil
	}
	ids := make([]string, 0, len(m))
	for id := range m {
		ids = append(ids, id)
	}
	return ids
}

// TagsForSession returns all tags for a session.
func (t *Tags) TagsForSession(sessionID string) []string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	var tags []string
	for tag, m := range t.tags {
		if m[sessionID] {
			tags = append(tags, tag)
		}
	}
	return tags
}

// Count returns the number of sessions with the given tag.
func (t *Tags) Count(tag string) int {
	t.mu.RLock()
	n := len(t.tags[tag])
	t.mu.RUnlock()
	return n
}

// AllTags returns all known tags.
func (t *Tags) AllTags() []string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	tags := make([]string, 0, len(t.tags))
	for tag := range t.tags {
		tags = append(tags, tag)
	}
	return tags
}

// RoomMessage represents a message stored in room history.
type RoomMessage struct {
	SenderID  string    `json:"sender_id"`
	Data      []byte    `json:"data"`
	Timestamp time.Time `json:"timestamp"`
}

// RoomWithHistory wraps a Room with message history support.
// New members can receive recent messages when they join.
type RoomWithHistory struct {
	*Room
	history *RingBuffer[RoomMessage]
}

// NewRoomWithHistory creates a room wrapper with history of the given capacity.
func NewRoomWithHistory(room *Room, historySize int) *RoomWithHistory {
	return &RoomWithHistory{
		Room:    room,
		history: NewRingBuffer[RoomMessage](historySize),
	}
}

// BroadcastAndRecord sends data to all room members AND records it in history.
func (r *RoomWithHistory) BroadcastAndRecord(senderID string, data []byte) {
	r.history.Push(RoomMessage{
		SenderID:  senderID,
		Data:      data,
		Timestamp: time.Now(),
	})
	r.Room.Broadcast(data)
}

// BroadcastExceptAndRecord sends to all except sender AND records in history.
func (r *RoomWithHistory) BroadcastExceptAndRecord(senderID string, data []byte) {
	r.history.Push(RoomMessage{
		SenderID:  senderID,
		Data:      data,
		Timestamp: time.Now(),
	})
	r.Room.BroadcastExcept(data, senderID)
}

// History returns recent messages (oldest first).
func (r *RoomWithHistory) History() []RoomMessage {
	return r.history.Items()
}

// HistorySize returns the number of messages in history.
func (r *RoomWithHistory) HistorySize() int {
	return r.history.Len()
}

// ReplayHistory sends all history messages to a specific session via datagrams.
// Useful for sending catch-up data when a new member joins.
func (r *RoomWithHistory) ReplayHistory(c *Context) {
	for _, msg := range r.history.Items() {
		_ = c.SendDatagram(msg.Data)
	}
}

// ReplayHistorySince sends messages newer than the given timestamp.
func (r *RoomWithHistory) ReplayHistorySince(c *Context, since time.Time) {
	for _, msg := range r.history.Items() {
		if msg.Timestamp.After(since) {
			_ = c.SendDatagram(msg.Data)
		}
	}
}

// ClearHistory removes all stored messages.
func (r *RoomWithHistory) ClearHistory() {
	r.history.Clear()
}

// RingBuffer is a fixed-size, lock-free ring buffer for messages.
// When full, the oldest message is overwritten.
// Useful for storing recent messages (chat history, event log).
type RingBuffer[T any] struct {
	mu   sync.RWMutex
	data []T
	head int
	size int
	cap  int
}

// NewRingBuffer creates a ring buffer with the given capacity.
func NewRingBuffer[T any](capacity int) *RingBuffer[T] {
	return &RingBuffer[T]{
		data: make([]T, capacity),
		cap:  capacity,
	}
}

// Push adds an item to the buffer. Overwrites oldest if full.
func (rb *RingBuffer[T]) Push(item T) {
	rb.mu.Lock()
	rb.data[rb.head] = item
	rb.head = (rb.head + 1) % rb.cap
	if rb.size < rb.cap {
		rb.size++
	}
	rb.mu.Unlock()
}

// Items returns all items in order (oldest first).
func (rb *RingBuffer[T]) Items() []T {
	rb.mu.RLock()
	defer rb.mu.RUnlock()

	if rb.size == 0 {
		return nil
	}

	result := make([]T, rb.size)
	if rb.size < rb.cap {
		copy(result, rb.data[:rb.size])
	} else {
		// Buffer is full, head points to oldest
		n := copy(result, rb.data[rb.head:])
		copy(result[n:], rb.data[:rb.head])
	}
	return result
}

// Last returns the most recently added item.
func (rb *RingBuffer[T]) Last() (T, bool) {
	rb.mu.RLock()
	defer rb.mu.RUnlock()

	var zero T
	if rb.size == 0 {
		return zero, false
	}

	idx := (rb.head - 1 + rb.cap) % rb.cap
	return rb.data[idx], true
}

// Len returns the number of items in the buffer.
func (rb *RingBuffer[T]) Len() int {
	rb.mu.RLock()
	n := rb.size
	rb.mu.RUnlock()
	return n
}

// Cap returns the buffer capacity.
func (rb *RingBuffer[T]) Cap() int {
	return rb.cap
}

// Clear empties the buffer.
func (rb *RingBuffer[T]) Clear() {
	rb.mu.Lock()
	rb.head = 0
	rb.size = 0
	rb.mu.Unlock()
}

// PubSub provides topic-based publish/subscribe messaging.
// Sessions subscribe to topics and receive datagrams published to those topics.
// More granular than rooms — a session can subscribe to any combination of topics.
type PubSub struct {
	mu          sync.RWMutex
	subscribers map[string]map[string]*Context // topic -> sessionID -> context
}

// NewPubSub creates a new pub/sub hub.
func NewPubSub() *PubSub {
	return &PubSub{
		subscribers: make(map[string]map[string]*Context),
	}
}

// Subscribe adds a session to a topic.
func (ps *PubSub) Subscribe(topic string, c *Context) {
	ps.mu.Lock()
	if ps.subscribers[topic] == nil {
		ps.subscribers[topic] = make(map[string]*Context)
	}
	ps.subscribers[topic][c.ID()] = c
	ps.mu.Unlock()
}

// Unsubscribe removes a session from a topic.
func (ps *PubSub) Unsubscribe(topic string, c *Context) {
	ps.mu.Lock()
	if m := ps.subscribers[topic]; m != nil {
		delete(m, c.ID())
		if len(m) == 0 {
			delete(ps.subscribers, topic)
		}
	}
	ps.mu.Unlock()
}

// UnsubscribeAll removes a session from all topics.
func (ps *PubSub) UnsubscribeAll(c *Context) {
	ps.mu.Lock()
	for topic, subs := range ps.subscribers {
		delete(subs, c.ID())
		if len(subs) == 0 {
			delete(ps.subscribers, topic)
		}
	}
	ps.mu.Unlock()
}

// Publish sends a datagram to all subscribers of a topic.
func (ps *PubSub) Publish(topic string, data []byte) {
	ps.mu.RLock()
	subs := ps.subscribers[topic]
	ps.mu.RUnlock()

	for _, c := range subs {
		_ = c.SendDatagram(data)
	}
}

// PublishExcept sends to all subscribers except the given session.
func (ps *PubSub) PublishExcept(topic string, data []byte, excludeID string) {
	ps.mu.RLock()
	subs := ps.subscribers[topic]
	ps.mu.RUnlock()

	for id, c := range subs {
		if id != excludeID {
			_ = c.SendDatagram(data)
		}
	}
}

// SubscriberCount returns the number of subscribers for a topic.
func (ps *PubSub) SubscriberCount(topic string) int {
	ps.mu.RLock()
	n := len(ps.subscribers[topic])
	ps.mu.RUnlock()
	return n
}

// Topics returns all active topics.
func (ps *PubSub) Topics() []string {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	topics := make([]string, 0, len(ps.subscribers))
	for t := range ps.subscribers {
		topics = append(topics, t)
	}
	return topics
}

// TopicsForSession returns all topics a session is subscribed to.
func (ps *PubSub) TopicsForSession(sessionID string) []string {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	var topics []string
	for topic, subs := range ps.subscribers {
		if _, ok := subs[sessionID]; ok {
			topics = append(topics, topic)
		}
	}
	return topics
}

// PersistentPubSub extends PubSub with per-topic message history.
// When a client subscribes, they can replay messages they missed.
type PersistentPubSub struct {
	*PubSub
	mu      sync.RWMutex
	history map[string]*RingBuffer[[]byte] // topic -> message history
	maxHist int
}

// NewPersistentPubSub creates a pub/sub with per-topic history.
func NewPersistentPubSub(historyPerTopic int) *PersistentPubSub {
	return &PersistentPubSub{
		PubSub:  NewPubSub(),
		history: make(map[string]*RingBuffer[[]byte]),
		maxHist: historyPerTopic,
	}
}

// PublishPersistent publishes to subscribers AND stores in history.
func (pps *PersistentPubSub) PublishPersistent(topic string, data []byte) {
	pps.mu.Lock()
	if pps.history[topic] == nil {
		pps.history[topic] = NewRingBuffer[[]byte](pps.maxHist)
	}
	pps.history[topic].Push(data)
	pps.mu.Unlock()

	pps.PubSub.Publish(topic, data)
}

// Replay sends all stored history for a topic to the given session.
func (pps *PersistentPubSub) Replay(topic string, c *Context) {
	pps.mu.RLock()
	rb, ok := pps.history[topic]
	pps.mu.RUnlock()

	if !ok {
		return
	}

	for _, msg := range rb.Items() {
		_ = c.SendDatagram(msg)
	}
}

// HistoryLen returns the number of stored messages for a topic.
func (pps *PersistentPubSub) HistoryLen(topic string) int {
	pps.mu.RLock()
	defer pps.mu.RUnlock()
	rb, ok := pps.history[topic]
	if !ok {
		return 0
	}
	return rb.Len()
}

// ClearHistory removes all stored messages for a topic.
func (pps *PersistentPubSub) ClearHistory(topic string) {
	pps.mu.Lock()
	delete(pps.history, topic)
	pps.mu.Unlock()
}

// TypedPubSub provides type-safe publish/subscribe with automatic encoding.
type TypedPubSub[T any] struct {
	ps    *PubSub
	codec codec.Codec
}

// NewTypedPubSub wraps a PubSub with typed messages.
func NewTypedPubSub[T any](ps *PubSub, c codec.Codec) *TypedPubSub[T] {
	return &TypedPubSub[T]{ps: ps, codec: c}
}

// Publish encodes and publishes a message to all subscribers of a topic.
func (tp *TypedPubSub[T]) Publish(topic string, msg T) error {
	data, err := tp.codec.Marshal(msg)
	if err != nil {
		return err
	}
	tp.ps.Publish(topic, data)
	return nil
}

// PublishExcept encodes and publishes to all except one session.
func (tp *TypedPubSub[T]) PublishExcept(topic string, msg T, excludeID string) error {
	data, err := tp.codec.Marshal(msg)
	if err != nil {
		return err
	}
	tp.ps.PublishExcept(topic, data, excludeID)
	return nil
}

// Subscribe delegates to the underlying PubSub.
func (tp *TypedPubSub[T]) Subscribe(topic string, c *Context) {
	tp.ps.Subscribe(topic, c)
}

// Unsubscribe delegates to the underlying PubSub.
func (tp *TypedPubSub[T]) Unsubscribe(topic string, c *Context) {
	tp.ps.Unsubscribe(topic, c)
}

// UnsubscribeAll delegates to the underlying PubSub.
func (tp *TypedPubSub[T]) UnsubscribeAll(c *Context) {
	tp.ps.UnsubscribeAll(c)
}

// TopicsIter returns an iterator over pub/sub topics and subscriber counts.
func (ps *PubSub) TopicsIter() iter.Seq2[string, int] {
	return func(yield func(string, int) bool) {
		ps.mu.RLock()
		defer ps.mu.RUnlock()
		for topic, subs := range ps.subscribers {
			if !yield(topic, len(subs)) {
				return
			}
		}
	}
}

// TagsIter returns an iterator over all tags and their session counts.
func (t *Tags) TagsIter() iter.Seq2[string, int] {
	return func(yield func(string, int) bool) {
		t.mu.RLock()
		defer t.mu.RUnlock()
		for tag, sessions := range t.tags {
			if !yield(tag, len(sessions)) {
				return
			}
		}
	}
}

// KVSync provides a synchronized key-value store that can be shared
// between server and client via a stream.
// Useful for game state sync, config sync, or shared document state.
type KVSync struct {
	mu       sync.RWMutex
	data     map[string]json.RawMessage
	onChange func(key string, value json.RawMessage)
}

// NewKVSync creates a new synchronized key-value store.
func NewKVSync() *KVSync {
	return &KVSync{
		data: make(map[string]json.RawMessage),
	}
}

// Set sets a key-value pair and notifies the onChange callback.
func (kv *KVSync) Set(key string, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	kv.mu.Lock()
	kv.data[key] = data
	cb := kv.onChange
	kv.mu.Unlock()

	if cb != nil {
		cb(key, data)
	}
	return nil
}

// Get retrieves a value by key and unmarshals it into v.
func (kv *KVSync) Get(key string, v any) error {
	kv.mu.RLock()
	data, ok := kv.data[key]
	kv.mu.RUnlock()

	if !ok {
		return nil
	}
	return json.Unmarshal(data, v)
}

// GetRaw retrieves the raw JSON for a key.
func (kv *KVSync) GetRaw(key string) (json.RawMessage, bool) {
	kv.mu.RLock()
	data, ok := kv.data[key]
	kv.mu.RUnlock()
	return data, ok
}

// Delete removes a key.
func (kv *KVSync) Delete(key string) {
	kv.mu.Lock()
	delete(kv.data, key)
	kv.mu.Unlock()
}

// Keys returns all keys.
func (kv *KVSync) Keys() []string {
	kv.mu.RLock()
	defer kv.mu.RUnlock()
	keys := make([]string, 0, len(kv.data))
	for k := range kv.data {
		keys = append(keys, k)
	}
	return keys
}

// Snapshot returns a copy of all key-value pairs as raw JSON.
func (kv *KVSync) Snapshot() map[string]json.RawMessage {
	kv.mu.RLock()
	defer kv.mu.RUnlock()
	snap := make(map[string]json.RawMessage, len(kv.data))
	for k, v := range kv.data {
		snap[k] = v
	}
	return snap
}

// OnChange sets a callback that fires when a key is updated.
func (kv *KVSync) OnChange(fn func(key string, value json.RawMessage)) {
	kv.mu.Lock()
	kv.onChange = fn
	kv.mu.Unlock()
}

// Len returns the number of keys.
func (kv *KVSync) Len() int {
	kv.mu.RLock()
	n := len(kv.data)
	kv.mu.RUnlock()
	return n
}

// ResumeToken is an opaque token that clients can use to restore
// session state after a reconnect.
type ResumeToken string

// ResumeStore persists session state for reconnection.
// When a client disconnects and reconnects with a resume token,
// the framework can restore their context store values (user info, etc.)
// without requiring re-authentication.
type ResumeStore struct {
	mu     sync.Mutex
	states map[ResumeToken]*resumeState
	ttl    time.Duration
}

type resumeState struct {
	store   map[string]any
	expires time.Time
}

// NewResumeStore creates a store with the given TTL for saved states.
// States are automatically expired after TTL.
func NewResumeStore(ttl time.Duration) *ResumeStore {
	rs := &ResumeStore{
		states: make(map[ResumeToken]*resumeState),
		ttl:    ttl,
	}
	go rs.cleanup()
	return rs
}

// Save stores the session's context values and returns a resume token.
// The client should store this token and present it on reconnection.
func (rs *ResumeStore) Save(c *Context) ResumeToken {
	token := generateToken()

	c.mu.RLock()
	storeCopy := make(map[string]any, len(c.store))
	for k, v := range c.store {
		storeCopy[k] = v
	}
	c.mu.RUnlock()

	rs.mu.Lock()
	rs.states[token] = &resumeState{
		store:   storeCopy,
		expires: time.Now().Add(rs.ttl),
	}
	rs.mu.Unlock()

	return token
}

// Restore applies saved state to a new session context.
// Returns true if the token was valid and state was restored.
func (rs *ResumeStore) Restore(c *Context, token ResumeToken) bool {
	rs.mu.Lock()
	state, ok := rs.states[token]
	if ok {
		delete(rs.states, token) // single-use token
	}
	rs.mu.Unlock()

	if !ok || time.Now().After(state.expires) {
		return false
	}

	for k, v := range state.store {
		c.Set(k, v)
	}
	return true
}

// Count returns the number of stored resume states.
func (rs *ResumeStore) Count() int {
	rs.mu.Lock()
	n := len(rs.states)
	rs.mu.Unlock()
	return n
}

func (rs *ResumeStore) cleanup() {
	ticker := time.NewTicker(rs.ttl / 2)
	defer ticker.Stop()
	for range ticker.C {
		rs.mu.Lock()
		now := time.Now()
		for token, state := range rs.states {
			if now.After(state.expires) {
				delete(rs.states, token)
			}
		}
		rs.mu.Unlock()
	}
}

func generateToken() ResumeToken {
	b := make([]byte, 32)
	rand.Read(b)
	return ResumeToken(hex.EncodeToString(b))
}
