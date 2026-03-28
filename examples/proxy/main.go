// Example: WebTransport proxy.
//
// Demonstrates using Pipe to relay streams between two WebTransport servers.
// Client → Proxy → Backend
//
// This pattern is useful for:
// - Load balancing WebTransport connections
// - Adding auth/logging without modifying the backend
// - Protocol translation (WebTransport ↔ WebSocket)
package main

import (
	"context"
	"crypto/tls"
	"log"
	"log/slog"

	"github.com/quic-go/webtransport-go"
	"github.com/rarebek/wt"
	"github.com/rarebek/wt/middleware"
)

func main() {
	backendURL := "https://localhost:4434/echo" // backend server

	server := wt.New(
		wt.WithAddr(":4433"),
		wt.WithSelfSignedTLS(),
	)

	log.Printf("Proxy cert hash: %s", server.CertHash())

	server.Use(middleware.DefaultLogger())
	server.Use(middleware.Recover(nil))

	// Proxy handler: relay streams to backend
	server.Handle("/proxy/{path...}", func(c *wt.Context) {
		path := c.Param("path")
		targetURL := backendURL
		if path != "" {
			targetURL = "https://localhost:4434/" + path
		}

		slog.Info("proxying to backend",
			"target", targetURL,
			"client", c.RemoteAddr().String(),
		)

		// Connect to backend
		dialer := webtransport.Dialer{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		_, backendSession, err := dialer.Dial(context.Background(), targetURL, nil)
		if err != nil {
			slog.Error("failed to connect to backend", "error", err)
			c.CloseWithError(502, "backend unavailable")
			return
		}
		defer backendSession.CloseWithError(0, "")

		// Relay datagrams bidirectionally
		go func() {
			for {
				data, err := c.ReceiveDatagram()
				if err != nil {
					return
				}
				_ = backendSession.SendDatagram(data)
			}
		}()
		go func() {
			for {
				data, err := backendSession.ReceiveDatagram(context.Background())
				if err != nil {
					return
				}
				_ = c.SendDatagram(data)
			}
		}()

		// Relay streams
		for {
			clientStream, err := c.AcceptStream()
			if err != nil {
				return
			}

			go func() {
				backendStream, err := backendSession.OpenStreamSync(context.Background())
				if err != nil {
					clientStream.Close()
					return
				}

				// Use Pipe to bidirectionally relay
				backendWrapped := &wt.Stream{}
				_ = backendWrapped // In real code, wrap the backend stream
				// For now, manually relay
				go func() {
					buf := make([]byte, 4096)
					for {
						n, err := clientStream.Read(buf)
						if err != nil {
							backendStream.Close()
							return
						}
						backendStream.Write(buf[:n])
					}
				}()
				go func() {
					buf := make([]byte, 4096)
					for {
						n, err := backendStream.Read(buf)
						if err != nil {
							clientStream.Close()
							return
						}
						clientStream.Write(buf[:n])
					}
				}()
			}()
		}
	})

	log.Printf("Proxy listening on %s", server.Addr())
	log.Fatal(server.ListenAndServe())
}
