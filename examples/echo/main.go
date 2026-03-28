// Example: minimal WebTransport echo server.
// Every stream echoes back whatever the client sends.
// Datagrams are also echoed.
package main

import (
	"io"
	"log"
	"log/slog"

	"github.com/rarebek/wt"
	"github.com/rarebek/wt/middleware"
)

func main() {
	server := wt.New(
		wt.WithAddr(":4433"),
		wt.WithSelfSignedTLS(),
	)

	// Log the cert hash for browser usage
	log.Printf("Certificate hash: %s", server.CertHash())
	log.Printf("In browser JS:\n  const transport = new WebTransport('https://localhost:4433/echo', {\n    serverCertificateHashes: [{ algorithm: 'sha-256', value: new Uint8Array([...]) }]\n  });")

	// Global middleware
	server.Use(middleware.DefaultLogger())
	server.Use(middleware.Recover(nil))

	// Echo handler: every stream echoes, every datagram echoes
	server.Handle("/echo", func(c *wt.Context) {
		slog.Info("new echo session", "id", c.ID(), "remote", c.RemoteAddr())

		// Echo datagrams in a goroutine
		go func() {
			for {
				data, err := c.ReceiveDatagram()
				if err != nil {
					return
				}
				_ = c.SendDatagram(data)
			}
		}()

		// Echo streams
		for {
			stream, err := c.AcceptStream()
			if err != nil {
				return
			}
			go func() {
				defer stream.Close()
				_, _ = io.Copy(stream, stream)
			}()
		}
	})

	log.Printf("WebTransport echo server listening on %s", server.Addr())
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
