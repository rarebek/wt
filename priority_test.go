package wt

import "testing"

func TestPriorityValues(t *testing.T) {
	if PriorityBackground != 0 {
		t.Error("PriorityBackground should be 0")
	}
	if PriorityNormal != 3 {
		t.Error("PriorityNormal should be 3")
	}
	if PriorityCritical != 7 {
		t.Error("PriorityCritical should be 7")
	}
	if PriorityHigh <= PriorityNormal {
		t.Error("PriorityHigh should be greater than PriorityNormal")
	}
}

func TestDefaultStreamConfig(t *testing.T) {
	cfg := DefaultStreamConfig()
	if cfg.Priority != PriorityNormal {
		t.Errorf("expected PriorityNormal, got %d", cfg.Priority)
	}
	if cfg.TypeID != 0 {
		t.Errorf("expected TypeID 0, got %d", cfg.TypeID)
	}
}

func TestVersionString(t *testing.T) {
	if Version == "" {
		t.Error("Version should not be empty")
	}
}
