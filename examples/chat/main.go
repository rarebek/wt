// Example: WebTransport chat server with rooms.
// Clients connect to /chat/{room} and send/receive messages via streams.
// Datagrams are used for presence (typing indicators, cursor position).
package main

import (
	"encoding/json"
	"log"
	"log/slog"

	"github.com/rarebek/wt"
	"github.com/rarebek/wt/middleware"
)

type ChatMessage struct {
	User string `json:"user"`
	Text string `json:"text"`
	Room string `json:"room"`
}

type PresenceUpdate struct {
	User   string `json:"user"`
	Action string `json:"action"` // "typing", "idle", "joined", "left"
}

var rooms = wt.NewRoomManager()

func main() {
	server := wt.New(
		wt.WithAddr(":4433"),
		wt.WithSelfSignedTLS(),
	)

	log.Printf("Certificate hash: %s", server.CertHash())

	server.Use(middleware.DefaultLogger())
	server.Use(middleware.Recover(nil))
	server.Use(middleware.RateLimit(10)) // max 10 connections per IP

	server.Handle("/chat/{room}", handleChat)

	log.Printf("Chat server listening on %s", server.Addr())
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

func handleChat(c *wt.Context) {
	roomName := c.Param("room")
	room := rooms.GetOrCreate(roomName)
	room.Join(c)
	defer room.Leave(c)

	slog.Info("user joined room",
		"room", roomName,
		"session", c.ID(),
		"members", room.Count(),
	)

	// Broadcast presence via datagrams
	go func() {
		broadcastPresence(room, c.ID(), "joined")
		for {
			data, err := c.ReceiveDatagram()
			if err != nil {
				return
			}
			// Relay presence datagrams to all room members except sender
			room.BroadcastExcept(data, c.ID())
		}
	}()

	// Handle chat streams
	for {
		stream, err := c.AcceptStream()
		if err != nil {
			broadcastPresence(room, c.ID(), "left")
			return
		}
		go handleChatStream(stream, room, c.ID())
	}
}

func handleChatStream(s *wt.Stream, room *wt.Room, senderID string) {
	defer s.Close()

	for {
		msg, err := s.ReadMessage()
		if err != nil {
			return
		}

		// Broadcast message to all room members via their open streams
		// In a real app, you'd maintain a list of outbound streams per member
		for _, member := range room.Members() {
			if member.ID() == senderID {
				continue
			}
			outStream, err := member.OpenStream()
			if err != nil {
				continue
			}
			_ = outStream.WriteMessage(msg)
			_ = outStream.Close()
		}
	}
}

func broadcastPresence(room *wt.Room, userID, action string) {
	update := PresenceUpdate{User: userID, Action: action}
	data, _ := json.Marshal(update)
	room.Broadcast(data)
}
