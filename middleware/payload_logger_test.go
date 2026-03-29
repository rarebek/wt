package middleware

import "testing"

func TestPayloadStats(t *testing.T) {
	ps := NewPayloadStats()
	ps.Record(100)
	ps.Record(200)
	ps.Record(50)

	if ps.TotalMessages.Load() != 3 {
		t.Errorf("expected 3 messages, got %d", ps.TotalMessages.Load())
	}
	if ps.TotalBytes.Load() != 350 {
		t.Errorf("expected 350 bytes, got %d", ps.TotalBytes.Load())
	}
	if ps.LargestMessage.Load() != 200 {
		t.Errorf("expected largest 200, got %d", ps.LargestMessage.Load())
	}
}
