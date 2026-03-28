// Example: Dual-protocol server (HTTP/2 + WebTransport).
//
// Shows the recommended production setup:
// 1. HTTP/2 server on :443 (TCP) — serves regular HTTP, advertises HTTP/3 via Alt-Svc
// 2. WebTransport server on :443 (UDP) — serves WebTransport sessions
// 3. Debug server on :6060 — pprof + health + Prometheus metrics
package main

import (
	"log"
	"net/http"

	"github.com/rarebek/wt"
	"github.com/rarebek/wt/middleware"
)

func main() {
	// --- WebTransport server (QUIC/UDP) ---
	server := wt.New(
		wt.WithAddr(":4433"),
		wt.WithSelfSignedTLS(), // Use WithTLS() or WithAutoCert() in production
	)

	pm := middleware.NewPrometheusMetrics()

	server.Use(middleware.DefaultLogger())
	server.Use(middleware.Recover(nil))
	server.Use(pm.Middleware())
	server.Use(middleware.RateLimit(100))

	server.Handle("/app", wt.HandleStream(func(s *wt.Stream, c *wt.Context) {
		defer s.Close()
		msg, _ := s.ReadMessage()
		s.WriteMessage(msg)
	}))

	log.Printf("WebTransport cert hash: %s", server.CertHash())
	go func() {
		if err := server.ListenAndServe(); err != nil {
			log.Printf("WebTransport error: %v", err)
		}
	}()

	// --- HTTP/2 server (TCP) — serves regular HTTP + Alt-Svc discovery ---
	httpMux := http.NewServeMux()

	// Add Alt-Svc header to all HTTP responses
	altSvc := wt.AltSvcMiddleware(4433)

	httpMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<!DOCTYPE html><html><body>
<h1>WebTransport Server</h1>
<p>Connect via WebTransport: <code>new WebTransport("https://localhost:4433/app")</code></p>
<p>This HTTP/2 server advertises HTTP/3 via Alt-Svc header.</p>
</body></html>`))
	})

	httpMux.HandleFunc("/api/info", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"transport":"webtransport","port":4433}`))
	})

	go func() {
		log.Println("HTTP/2 server on :8443 (with Alt-Svc discovery)")
		// In production: use TLS for HTTP/2
		log.Fatal(http.ListenAndServe(":8443", altSvc(httpMux)))
	}()

	// --- Debug server ---
	debugMux := wt.DebugMux(server)
	debugMux.Handle("/metrics", pm.Handler())

	log.Println("Debug server on :6060 (pprof + health + metrics)")
	log.Fatal(http.ListenAndServe(":6060", debugMux))
}
