package wt

import "testing"

func TestDefaultKeepAliveInterval(t *testing.T) {
	if DefaultKeepAliveInterval.Seconds() != 15 {
		t.Errorf("expected 15s, got %v", DefaultKeepAliveInterval)
	}
}
