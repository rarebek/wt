package wt

import (
	"encoding/json"
	"sync"
	"time"
)

// PresenceInfo represents a user's current state in a room.
type PresenceInfo struct {
	UserID    string    `json:"user_id"`
	SessionID string    `json:"session_id"`
	Status    string    `json:"status"` // "online", "idle", "away", "typing"
	Metadata  map[string]any `json:"metadata,omitempty"`
	UpdatedAt time.Time `json:"updated_at"`
}

// PresenceTracker tracks user presence across rooms.
// Integrates with RoomManager to automatically track join/leave
// and supports custom status updates.
type PresenceTracker struct {
	mu       sync.RWMutex
	presence map[string]map[string]*PresenceInfo // room -> sessionID -> info
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
