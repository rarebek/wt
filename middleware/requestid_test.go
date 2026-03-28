package middleware

import (
	"testing"
)

func TestGenerateRequestID(t *testing.T) {
	id1 := generateRequestID()
	id2 := generateRequestID()

	if id1 == "" {
		t.Error("expected non-empty ID")
	}
	if len(id1) != 32 { // 16 bytes = 32 hex chars
		t.Errorf("expected 32 char ID, got %d", len(id1))
	}
	if id1 == id2 {
		t.Error("expected unique IDs")
	}
}
