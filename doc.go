/*
Package wt provides a high-level framework for building WebTransport applications in Go.

WebTransport is a modern protocol built on QUIC and HTTP/3 that provides
multiplexed bidirectional streams, unidirectional streams, and unreliable
datagrams — all without TCP's head-of-line blocking.

This package sits on top of quic-go/webtransport-go and provides:

  - Path-based routing with parameter extraction
  - Middleware stack (auth, logging, rate limiting, compression, metrics)
  - Session management with rooms and pub/sub
  - Message framing (length-prefixed) over streams
  - Type-safe stream handlers using Go generics
  - WebSocket fallback for browsers without WebTransport support
  - Self-signed certificate generation for development

# Quick Start

	server := wt.New(
		wt.WithAddr(":4433"),
		wt.WithSelfSignedTLS(),
	)

	server.Handle("/echo", func(c *wt.Context) {
		for {
			stream, err := c.AcceptStream()
			if err != nil {
				return
			}
			go func() {
				defer stream.Close()
				io.Copy(stream, stream)
			}()
		}
	})

	server.ListenAndServe()

# Architecture

The framework follows familiar Go patterns inspired by Gin, Echo, and Chi:

  - Handlers receive a [Context] wrapping the WebTransport session
  - Middleware uses the func(c *Context, next HandlerFunc) signature
  - Route groups share prefixes and middleware
  - Sessions are tracked in a [SessionStore] for enumeration and broadcast

# Streams vs Datagrams

WebTransport offers two data transport modes:

Streams are reliable and ordered, similar to TCP connections. Use them for
messages that must arrive: chat messages, game events, file transfers.

Datagrams are unreliable and unordered, similar to UDP packets. Use them for
data where the latest value matters more than every value: player positions,
sensor readings, cursor locations.

# WebSocket Fallback

The [fallback] sub-package provides transparent WebSocket fallback for browsers
that don't support WebTransport (notably Safari as of early 2026). Stream
multiplexing is simulated over the single WebSocket connection.
*/
package wt
