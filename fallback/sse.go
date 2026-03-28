// SSE provides a Server-Sent Events fallback for environments where
// both WebTransport (UDP) and WebSocket (TCP upgrade) are blocked.
//
// SSE is unidirectional (server → client only). For client → server,
// pair with regular HTTP POST requests.
//
// This is the most degraded fallback mode — use only when nothing else works.
package fallback

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
)

// SSEClient represents a connected SSE client.
type SSEClient struct {
	w       http.ResponseWriter
	flusher http.Flusher
	done    chan struct{}
	once    sync.Once
	id      string
}

// SSEHub manages SSE client connections.
type SSEHub struct {
	mu      sync.RWMutex
	clients map[string]*SSEClient
	nextID  int
}

// NewSSEHub creates a new SSE hub.
func NewSSEHub() *SSEHub {
	return &SSEHub{
		clients: make(map[string]*SSEClient),
	}
}

// Handler returns an http.Handler that accepts SSE connections.
// Use alongside the WebTransport server for maximum compatibility.
//
// Usage:
//
//	hub := fallback.NewSSEHub()
//	http.Handle("/events", hub.Handler())
func (h *SSEHub) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "SSE not supported", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		h.mu.Lock()
		id := fmt.Sprintf("sse-%d", h.nextID)
		h.nextID++
		client := &SSEClient{
			w:       w,
			flusher: flusher,
			done:    make(chan struct{}),
			id:      id,
		}
		h.clients[id] = client
		h.mu.Unlock()

		defer func() {
			h.mu.Lock()
			delete(h.clients, id)
			h.mu.Unlock()
		}()

		// Send connected event
		fmt.Fprintf(w, "event: connected\ndata: {\"id\":%q}\n\n", id)
		flusher.Flush()

		// Block until client disconnects
		select {
		case <-r.Context().Done():
		case <-client.done:
		}
	})
}

// Send sends an event to a specific client.
func (h *SSEHub) Send(clientID, event string, data any) error {
	h.mu.RLock()
	client, ok := h.clients[clientID]
	h.mu.RUnlock()

	if !ok {
		return fmt.Errorf("client %s not found", clientID)
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	fmt.Fprintf(client.w, "event: %s\ndata: %s\n\n", event, jsonData)
	client.flusher.Flush()
	return nil
}

// Broadcast sends an event to all connected clients.
func (h *SSEHub) Broadcast(event string, data any) {
	jsonData, _ := json.Marshal(data)
	msg := fmt.Sprintf("event: %s\ndata: %s\n\n", event, jsonData)

	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, client := range h.clients {
		fmt.Fprint(client.w, msg)
		client.flusher.Flush()
	}
}

// Count returns the number of connected SSE clients.
func (h *SSEHub) Count() int {
	h.mu.RLock()
	n := len(h.clients)
	h.mu.RUnlock()
	return n
}

// Close disconnects a specific client.
func (h *SSEHub) Close(clientID string) {
	h.mu.RLock()
	client, ok := h.clients[clientID]
	h.mu.RUnlock()

	if ok {
		client.once.Do(func() { close(client.done) })
	}
}
