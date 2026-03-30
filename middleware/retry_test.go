package middleware

import "testing"

func TestAutoReconnectStats(t *testing.T) {
	ars := NewAutoReconnectStats()

	ars.Connections.Add(10)
	ars.Disconnections.Add(3)
	ars.Reconnections.Add(2)

	conns, disconns, reconns := ars.Stats()
	if conns != 10 || disconns != 3 || reconns != 2 {
		t.Errorf("expected 10/3/2, got %d/%d/%d", conns, disconns, reconns)
	}
}
