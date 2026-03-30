package middleware

import "testing"

func TestConcurrencyStats(t *testing.T) {
	cs := NewConcurrencyStats()

	cs.ActiveSessions.Add(5)
	cs.TotalAccepted.Add(10)
	cs.PeakSessions.Store(8)

	snap := cs.Snapshot()
	if snap.Active != 5 {
		t.Errorf("expected 5 active, got %d", snap.Active)
	}
	if snap.Peak != 8 {
		t.Errorf("expected 8 peak, got %d", snap.Peak)
	}
	if snap.Accepted != 10 {
		t.Errorf("expected 10 accepted, got %d", snap.Accepted)
	}
}
