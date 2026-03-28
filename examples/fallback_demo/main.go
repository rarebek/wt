// Example: Combined WebTransport + WebSocket + SSE fallback.
//
// Serves the same application over three transports:
// 1. WebTransport (QUIC/HTTP3) — fastest, multiplexed streams + datagrams
// 2. WebSocket — fallback for browsers without WebTransport
// 3. SSE + HTTP POST — fallback for environments blocking all upgrades
//
// All three serve the same chat functionality.
package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/rarebek/wt"
	"github.com/rarebek/wt/fallback"
	"github.com/rarebek/wt/middleware"
)

type ChatMessage struct {
	User string `json:"user"`
	Text string `json:"text"`
}

var rooms = wt.NewRoomManager()
var sseHub = fallback.NewSSEHub()

func main() {
	// --- WebTransport server (primary) ---
	server := wt.New(
		wt.WithAddr(":4433"),
		wt.WithSelfSignedTLS(),
	)
	server.Use(middleware.DefaultLogger())
	server.Use(middleware.Recover(nil))
	server.Handle("/chat/{room}", handleWT)

	log.Printf("WebTransport cert hash: %s", server.CertHash())
	go func() {
		log.Fatal(server.ListenAndServe())
	}()

	// --- HTTP server (WebSocket + SSE fallback) ---
	mux := http.NewServeMux()

	// WebSocket fallback
	mux.Handle("/ws/chat/", fallback.Handler(handleWS))

	// SSE fallback
	mux.Handle("/sse/chat/", sseHub.Handler())

	// HTTP POST for SSE clients to send messages
	mux.HandleFunc("/api/chat/send", handleHTTPSend)

	// Health check
	mux.Handle("/health", wt.NewHealthCheck(server))

	log.Println("Fallback HTTP server on :8080")
	log.Println("  WebSocket: ws://localhost:8080/ws/chat/{room}")
	log.Println("  SSE:       http://localhost:8080/sse/chat/{room}")
	log.Println("  POST:      http://localhost:8080/api/chat/send")
	log.Fatal(http.ListenAndServe(":8080", mux))
}

// WebTransport handler
func handleWT(c *wt.Context) {
	roomName := c.Param("room")
	room := rooms.GetOrCreate(roomName)
	room.Join(c)
	defer room.Leave(c)

	for {
		stream, err := c.AcceptStream()
		if err != nil {
			return
		}
		go func() {
			defer stream.Close()
			msg, err := stream.ReadMessage()
			if err != nil {
				return
			}
			// Broadcast to WebTransport clients
			room.Broadcast(msg)
			// Also broadcast to SSE clients
			var chatMsg ChatMessage
			json.Unmarshal(msg, &chatMsg)
			sseHub.Broadcast("message", chatMsg)
		}()
	}
}

// WebSocket fallback handler
func handleWS(conn *fallback.WSConn) {
	defer conn.Close()

	for {
		stream, err := conn.AcceptStream()
		if err != nil {
			return
		}
		go func() {
			defer stream.Close()
			buf := make([]byte, 4096)
			n, err := stream.Read(buf)
			if err != nil {
				return
			}
			// Broadcast via datagram (simulated over WS)
			conn.SendDatagram(buf[:n])
			// Also broadcast to SSE
			var chatMsg ChatMessage
			json.Unmarshal(buf[:n], &chatMsg)
			sseHub.Broadcast("message", chatMsg)
		}()
	}
}

// HTTP POST handler for SSE clients to send messages
func handleHTTPSend(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}

	var msg ChatMessage
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	// Broadcast to all transports
	data, _ := json.Marshal(msg)

	// To SSE clients
	sseHub.Broadcast("message", msg)

	// To WebTransport rooms (if any)
	for _, name := range rooms.Rooms() {
		room, ok := rooms.Get(name)
		if ok {
			room.Broadcast(data)
		}
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "sent"})
}
