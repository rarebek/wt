// Example: Persistent pub/sub news feed over WebTransport.
// Subscribers get message history on connect, then real-time updates.
package main

import (
	"encoding/json"
	"log"
	"time"

	"github.com/rarebek/wt"
	"github.com/rarebek/wt/middleware"
)

type Article struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	Time  int64  `json:"time"`
}

func main() {
	server := wt.New(wt.WithAddr(":4433"), wt.WithSelfSignedTLS())
	log.Printf("cert: %s", server.CertHash())
	server.Use(middleware.DefaultLogger())

	pps := wt.NewPersistentPubSub(50) // keep last 50 articles per topic

	// Publisher endpoint — accepts articles via streams
	server.Handle("/publish/{topic}", wt.HandleStream(func(s *wt.Stream, c *wt.Context) {
		defer s.Close()
		topic := c.Param("topic")
		msg, _ := s.ReadMessage()
		pps.PublishPersistent(topic, msg)
		s.WriteMessage([]byte(`{"status":"published"}`))
	}))

	// Subscriber endpoint — replay history then stream new articles
	server.Handle("/subscribe/{topic}", func(c *wt.Context) {
		topic := c.Param("topic")
		pps.Subscribe(topic, c)
		defer pps.UnsubscribeAll(c)

		// Replay history
		pps.Replay(topic, c)

		// Keep alive until disconnect
		<-c.Context().Done()
	})

	// Publish a sample article every 10 seconds
	go func() {
		i := 0
		for {
			time.Sleep(10 * time.Second)
			i++
			article := Article{
				Title: "Breaking News",
				Body:  "This is article number",
				Time:  time.Now().UnixMilli(),
			}
			data, _ := json.Marshal(article)
			pps.PublishPersistent("news", data)
		}
	}()

	log.Fatal(server.ListenAndServe())
}
