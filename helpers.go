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
