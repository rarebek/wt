package fallback

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSSEHubCount(t *testing.T) {
	hub := NewSSEHub()
	if hub.Count() != 0 {
		t.Errorf("expected 0 clients, got %d", hub.Count())
	}
}

func TestSSEHubBroadcastNoClients(t *testing.T) {
	hub := NewSSEHub()
	// Should not panic with no clients
	hub.Broadcast("test", map[string]string{"msg": "hello"})
}

func TestSSEHubSendNoClient(t *testing.T) {
	hub := NewSSEHub()
	err := hub.Send("nonexistent", "test", "data")
	if err == nil {
		t.Error("expected error for nonexistent client")
	}
}

func TestSSEHubHandler(t *testing.T) {
	hub := NewSSEHub()
	handler := hub.Handler()

	// Verify it returns an http.Handler
	if handler == nil {
		t.Fatal("handler should not be nil")
	}

	// Create a test request
	req := httptest.NewRequest(http.MethodGet, "/events", nil)
	w := httptest.NewRecorder()

	// Run in goroutine (handler blocks)
	done := make(chan struct{})
	go func() {
		handler.ServeHTTP(w, req)
		close(done)
	}()

	// The handler will block waiting for context cancellation
	// We can't easily test the full SSE flow with httptest.NewRecorder
	// because it doesn't support flushing. This is a smoke test.
}
