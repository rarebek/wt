package middleware

import "testing"

func TestOriginTracker(t *testing.T) {
	ot := NewOriginTracker()

	ot.mu.Lock()
	ot.origins["https://app.example.com"] = 5
	ot.origins["https://other.com"] = 2
	ot.mu.Unlock()

	if ot.UniqueCount() != 2 {
		t.Errorf("expected 2 unique origins, got %d", ot.UniqueCount())
	}

	origins := ot.Origins()
	if origins["https://app.example.com"] != 5 {
		t.Error("expected 5 connections from app.example.com")
	}
}
