package wt

import "testing"

func TestFlowControlMonitor(t *testing.T) {
	fc := NewFlowControlMonitor()

	fc.StreamsOpened.Add(10)
	fc.StreamsClosed.Add(3)
	fc.DatagramsSent.Add(100)
	fc.DatagramsRecvd.Add(95)
	fc.BytesSent.Add(50000)
	fc.BytesReceived.Add(48000)

	stats := fc.Stats()

	if stats.StreamsActive != 7 {
		t.Errorf("expected 7 active streams, got %d", stats.StreamsActive)
	}
	if stats.DatagramsSent != 100 {
		t.Errorf("expected 100 sent, got %d", stats.DatagramsSent)
	}
	if stats.BytesSent != 50000 {
		t.Errorf("expected 50000 bytes sent, got %d", stats.BytesSent)
	}
}

func TestFlowControlMonitorEmpty(t *testing.T) {
	fc := NewFlowControlMonitor()
	stats := fc.Stats()

	if stats.StreamsActive != 0 || stats.BytesSent != 0 {
		t.Error("fresh monitor should have all zeros")
	}
}
