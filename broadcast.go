package wt

import "encoding/json"

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
