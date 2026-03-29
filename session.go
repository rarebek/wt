package wt

import (
	"iter"
	"log/slog"
	"sync"
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
