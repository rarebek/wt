package wt

import (
	"encoding/json"
	"time"
)

// ConnInfo provides detailed connection information for a session.
type ConnInfo struct {
	SessionID   string    `json:"session_id"`
	RemoteAddr  string    `json:"remote_addr"`
	LocalAddr   string    `json:"local_addr"`
	Path        string    `json:"path"`
	Params      map[string]string `json:"params,omitempty"`
	ConnectedAt time.Time `json:"connected_at"`
	Transport   string    `json:"transport"` // "webtransport" or "websocket"
	UserAgent   string    `json:"user_agent,omitempty"`
	Origin      string    `json:"origin,omitempty"`
}

// Info returns connection information for the session.
func (c *Context) Info() ConnInfo {
	info := ConnInfo{
		SessionID:   c.ID(),
		RemoteAddr:  c.RemoteAddr().String(),
		LocalAddr:   c.LocalAddr().String(),
		Path:        c.Request().URL.Path,
		Params:      c.Params(),
		Transport:   "webtransport",
		UserAgent:   c.Request().UserAgent(),
		Origin:      c.Request().Header.Get("Origin"),
	}

	if t, ok := c.Get("_connected_at"); ok {
		info.ConnectedAt = t.(time.Time)
	}

	return info
}

// InfoJSON returns connection info as a JSON string.
func (c *Context) InfoJSON() string {
	data, _ := json.Marshal(c.Info())
	return string(data)
}
