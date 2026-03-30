package middleware

import "testing"

func TestBandwidthTracker(t *testing.T) {
	bt := &BandwidthTracker{}

	sent, received := bt.Stats()
	if sent != 0 || received != 0 {
		t.Error("initial stats should be 0")
	}

	bt.RecordSent(100)
	bt.RecordSent(200)
	bt.RecordReceived(50)

	sent, received = bt.Stats()
	if sent != 300 {
		t.Errorf("expected 300 sent, got %d", sent)
	}
	if received != 50 {
		t.Errorf("expected 50 received, got %d", received)
	}
}
