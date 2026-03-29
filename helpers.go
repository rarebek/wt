package wt

import "encoding/json"

// SendJSON sends a JSON-encoded value as a datagram.
func (c *Context) SendJSON(v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return c.SendDatagram(data)
}

// WriteJSON writes a JSON-encoded value as a stream message.
func (s *Stream) WriteJSON(v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return s.WriteMessage(data)
}

// ReadJSON reads a stream message and decodes it as JSON.
func (s *Stream) ReadJSON(v any) error {
	data, err := s.ReadMessage()
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

// BroadcastJSON encodes v as JSON and broadcasts to all active sessions.
func (s *Server) BroadcastJSON(v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	s.Broadcast(data)
	return nil
}

// BroadcastJSONExcept sends to all except one session.
func (s *Server) BroadcastJSONExcept(v any, excludeID string) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	s.BroadcastExcept(data, excludeID)
	return nil
}

// MulticastJSON sends JSON to sessions matching a filter.
func (s *Server) MulticastJSON(v any, filter func(*Context) bool) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	s.Multicast(data, filter)
	return nil
}

// BroadcastJSONRoom encodes and broadcasts to all room members.
func BroadcastJSONRoom(r *Room, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	r.Broadcast(data)
	return nil
}

// Multicast sends a datagram to sessions matching a filter function.
// More flexible than Broadcast — only sends to sessions that match.
//
// Usage:
//
//	// Send to all sessions with user role "admin"
//	server.Multicast(data, func(c *Context) bool {
//	    role, _ := c.Get("role")
//	    return role == "admin"
//	})
func (s *Server) Multicast(data []byte, filter func(*Context) bool) {
	s.sessions.Each(func(c *Context) {
		if filter(c) {
			_ = c.SendDatagram(data)
		}
	})
}

// MulticastStream sends a reliable message via streams to matching sessions.
func (s *Server) MulticastStream(data []byte, filter func(*Context) bool) {
	s.sessions.Each(func(c *Context) {
		if filter(c) {
			go func() {
				stream, err := c.OpenStream()
				if err != nil {
					return
				}
				_ = stream.WriteMessage(data)
				stream.Close()
			}()
		}
	})
}

// Notification represents a structured notification message.
type Notification struct {
	Type  string `json:"type"`
	Title string `json:"title,omitempty"`
	Body  string `json:"body,omitempty"`
	Data  any    `json:"data,omitempty"`
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
