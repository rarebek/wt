package wt

import "testing"

func TestTickerStop(t *testing.T) {
	// Verify Stop is idempotent and doesn't panic
	tk := &Ticker{
		ticker: nil,
		done:   make(chan struct{}),
	}
	// Can't create full Ticker without Context, but test Stop logic
	close(tk.done)
	// Double stop should not panic
	tk.Stop()
}

func TestDefaultKeepAliveValue(t *testing.T) {
	if DefaultKeepAliveInterval.Seconds() != 15 {
		t.Errorf("expected 15s, got %v", DefaultKeepAliveInterval)
	}
}
