package middleware

import (
	"testing"
)

func TestMetricsSnapshot(t *testing.T) {
	m := NewMetrics()

	snap := m.Snapshot()
	if snap.ActiveSessions != 0 {
		t.Errorf("expected 0 active sessions, got %d", snap.ActiveSessions)
	}
	if snap.TotalSessions != 0 {
		t.Errorf("expected 0 total sessions, got %d", snap.TotalSessions)
	}

	m.ActiveSessions.Add(5)
	m.TotalSessions.Add(10)

	snap = m.Snapshot()
	if snap.ActiveSessions != 5 {
		t.Errorf("expected 5 active sessions, got %d", snap.ActiveSessions)
	}
	if snap.TotalSessions != 10 {
		t.Errorf("expected 10 total sessions, got %d", snap.TotalSessions)
	}
}

func TestMetricsSessionDuration(t *testing.T) {
	m := NewMetrics()

	// Non-existent session
	d := m.SessionDuration("nonexistent")
	if d != 0 {
		t.Errorf("expected 0 duration for nonexistent session, got %v", d)
	}
}
