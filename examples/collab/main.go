// Example: Real-time collaboration server.
//
// Demonstrates WebTransport's advantage for collaboration tools:
// - Stream 1 (type=1): Document operations (reliable, ordered)
// - Stream 2 (type=2): Cursor positions (can use datagrams — unreliable is fine)
// - Stream 3 (type=3): Chat messages (reliable, ordered)
// - Datagrams: Presence indicators (typing, idle, etc.)
//
// Each channel is independent — a large document operation doesn't block
// cursor updates or chat messages. This is impossible with WebSockets.
package main

import (
	"encoding/json"
	"log"
	"log/slog"

	"github.com/rarebek/wt"
	"github.com/rarebek/wt/middleware"
)

const (
	TypeDocOps uint16 = 1
	TypeCursor uint16 = 2
	TypeChat   uint16 = 3
)

type DocOp struct {
	Position int    `json:"pos"`
	Insert   string `json:"insert,omitempty"`
	Delete   int    `json:"delete,omitempty"`
	UserID   string `json:"user"`
}

type CursorPos struct {
	UserID string `json:"user"`
	Line   int    `json:"line"`
	Col    int    `json:"col"`
}

type ChatMsg struct {
	UserID string `json:"user"`
	Text   string `json:"text"`
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

	mux := wt.NewStreamMux()

	// Document operations — reliable, ordered
	mux.Handle(TypeDocOps, func(s *wt.Stream, c *wt.Context) {
		defer s.Close()
		room := rooms.GetOrCreate(c.Param("doc"))

		for {
			data, err := s.ReadMessage()
			if err != nil {
				return
			}

			var op DocOp
			if err := json.Unmarshal(data, &op); err != nil {
				continue
			}
			op.UserID = c.ID()

			slog.Debug("doc op", "pos", op.Position, "insert", op.Insert, "user", op.UserID)

			// Broadcast to all other collaborators
			encoded, _ := json.Marshal(op)
			room.BroadcastExcept(encoded, c.ID())
		}
	})

	// Cursor positions — broadcast via datagrams (unreliable is fine)
	mux.Handle(TypeCursor, func(s *wt.Stream, c *wt.Context) {
		defer s.Close()
		room := rooms.GetOrCreate(c.Param("doc"))

		for {
			data, err := s.ReadMessage()
			if err != nil {
				return
			}
			var cursor CursorPos
			if err := json.Unmarshal(data, &cursor); err != nil {
				continue
			}
			cursor.UserID = c.ID()
			encoded, _ := json.Marshal(cursor)
			room.BroadcastExcept(encoded, c.ID())
		}
	})

	// Chat — reliable
	mux.Handle(TypeChat, func(s *wt.Stream, c *wt.Context) {
		defer s.Close()
		room := rooms.GetOrCreate(c.Param("doc"))

		for {
			data, err := s.ReadMessage()
			if err != nil {
				return
			}
			var msg ChatMsg
			if err := json.Unmarshal(data, &msg); err != nil {
				continue
			}
			msg.UserID = c.ID()
			encoded, _ := json.Marshal(msg)
			room.Broadcast(encoded)
		}
	})

	server.Handle("/doc/{doc}", func(c *wt.Context) {
		docID := c.Param("doc")
		room := rooms.GetOrCreate(docID)
		room.Join(c)
		defer room.Leave(c)

		slog.Info("collaborator joined",
			"doc", docID,
			"user", c.ID(),
			"total", room.Count(),
		)

		// Presence via datagrams
		go func() {
			for {
				data, err := c.ReceiveDatagram()
				if err != nil {
					return
				}
				room.BroadcastExcept(data, c.ID())
			}
		}()

		// Route streams by type
		mux.Serve(c)
	})

	log.Printf("Collaboration server listening on %s", server.Addr())
	log.Fatal(server.ListenAndServe())
}
