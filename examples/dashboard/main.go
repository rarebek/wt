// Example: Admin dashboard over WebTransport.
// Pushes server stats to a dashboard client every second via datagrams.
package main

import (
	"log"
	"net/http"
	"time"

	"github.com/rarebek/wt"
	"github.com/rarebek/wt/middleware"
)

func main() {
	server := wt.New(wt.WithAddr(":4433"), wt.WithSelfSignedTLS())
	log.Printf("cert: %s", server.CertHash())

	pm := middleware.NewPrometheusMetrics()
	server.Use(pm.Middleware())

	// Main app handler
	server.Handle("/app", wt.HandleDatagram(func(d []byte, c *wt.Context) []byte {
		return append([]byte("echo:"), d...)
	}))

	// Dashboard: push stats every second
	server.Handle("/dashboard", func(c *wt.Context) {
		ticker := wt.NewTicker(c, time.Second, func() []byte {
			return server.StatsJSON()
		})
		defer ticker.Stop()
		<-c.Context().Done()
	})

	// HTTP endpoints
	go func() {
		mux := wt.DebugMux(server)
		mux.Handle("/metrics", pm.Handler())
		log.Println("Debug: http://localhost:6060")
		http.ListenAndServe(":6060", mux)
	}()

	log.Fatal(server.ListenAndServe())
}
