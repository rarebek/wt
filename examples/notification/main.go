// Example: Server-push notification system.
//
// Demonstrates unidirectional streams (server → client):
// - Server pushes notifications to connected clients
// - Each client subscribes to topics
// - No head-of-line blocking between different notification types
package main

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/rarebek/wt"
	"github.com/rarebek/wt/middleware"
)

type Notification struct {
	Topic   string `json:"topic"`
	Title   string `json:"title"`
	Body    string `json:"body"`
	SentAt  int64  `json:"sent_at"`
}

type Subscription struct {
	Topics []string `json:"topics"`
}

type NotificationHub struct {
	mu          sync.RWMutex
	subscribers map[string][]*wt.Context // topic -> sessions
}

func NewNotificationHub() *NotificationHub {
	return &NotificationHub{
		subscribers: make(map[string][]*wt.Context),
	}
}

func (h *NotificationHub) Subscribe(topic string, c *wt.Context) {
	h.mu.Lock()
	h.subscribers[topic] = append(h.subscribers[topic], c)
	h.mu.Unlock()
}

func (h *NotificationHub) Unsubscribe(c *wt.Context) {
	h.mu.Lock()
	for topic, subs := range h.subscribers {
		filtered := subs[:0]
		for _, s := range subs {
			if s.ID() != c.ID() {
				filtered = append(filtered, s)
			}
		}
		h.subscribers[topic] = filtered
	}
	h.mu.Unlock()
}

func (h *NotificationHub) Publish(notif Notification) {
	h.mu.RLock()
	subs := h.subscribers[notif.Topic]
	h.mu.RUnlock()

	data, _ := json.Marshal(notif)

	for _, c := range subs {
		go func(ctx *wt.Context) {
			// Use unidirectional stream for push notification
			uniStream, err := ctx.OpenUniStream()
			if err != nil {
				return
			}
			uniStream.WriteMessage(data)
			uniStream.Close()
		}(c)
	}
}

var hub = NewNotificationHub()

func main() {
	server := wt.New(
		wt.WithAddr(":4433"),
		wt.WithSelfSignedTLS(),
	)

	log.Printf("Certificate hash: %s", server.CertHash())

	server.Use(middleware.DefaultLogger())
	server.Use(middleware.Recover(nil))

	// Client subscribes and receives push notifications
	server.Handle("/notifications", func(c *wt.Context) {
		defer hub.Unsubscribe(c)

		// Accept subscription stream (client tells us what topics they want)
		stream, err := c.AcceptStream()
		if err != nil {
			return
		}

		msg, err := stream.ReadMessage()
		if err != nil {
			return
		}

		var sub Subscription
		if err := json.Unmarshal(msg, &sub); err != nil {
			return
		}

		for _, topic := range sub.Topics {
			hub.Subscribe(topic, c)
		}

		// Acknowledge subscription
		ack, _ := json.Marshal(map[string]any{
			"subscribed": sub.Topics,
		})
		stream.WriteMessage(ack)
		stream.Close()

		// Keep session alive until client disconnects
		<-c.Context().Done()
	})

	// Simulate periodic notifications
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		i := 0
		for range ticker.C {
			i++
			hub.Publish(Notification{
				Topic:  "system",
				Title:  "System Update",
				Body:   "Server is running smoothly",
				SentAt: time.Now().UnixMilli(),
			})
		}
	}()

	log.Printf("Notification server listening on %s", server.Addr())
	log.Fatal(server.ListenAndServe())
}
