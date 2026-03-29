package wt

import "encoding/json"

// Notification represents a structured notification message.
type Notification struct {
	Type    string `json:"type"`
	Title   string `json:"title,omitempty"`
	Body    string `json:"body,omitempty"`
	Data    any    `json:"data,omitempty"`
}

// Notify sends a notification datagram to the session.
func (c *Context) Notify(notif Notification) error {
	data, err := json.Marshal(notif)
	if err != nil {
		return err
	}
	return c.SendDatagram(data)
}

// NotifyAll sends a notification to all active sessions.
func (s *Server) NotifyAll(notif Notification) {
	data, _ := json.Marshal(notif)
	s.Broadcast(data)
}

// NotifyRoom sends a notification to all room members.
func NotifyRoom(r *Room, notif Notification) {
	data, _ := json.Marshal(notif)
	r.Broadcast(data)
}
