package wt

import (
	"time"
)

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
