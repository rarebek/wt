# wt

A high-level WebTransport framework for Go. Build real-time applications with multiplexed streams, unreliable datagrams, and zero boilerplate.

Built on top of [quic-go/webtransport-go](https://github.com/quic-go/webtransport-go).

## Why

WebTransport is the successor to WebSocket — faster, more flexible, built on QUIC/HTTP3. But the existing Go libraries give you raw protocol access. `wt` gives you a framework:

- **Routing** — path-based handlers with parameters (`/chat/{room}`)
- **Middleware** — auth, logging, rate limiting, compression, metrics
- **Rooms** — pub/sub groups with broadcast and presence
- **Typed streams** — generics-based message encoding/decoding
- **WebSocket fallback** — transparent fallback for browsers without WebTransport support
- **Self-signed certs** — one-line dev setup, outputs the hash for browser `serverCertificateHashes`

## Install

```bash
go get github.com/rarebek/wt
```

Requires Go 1.22+.

## Quick Start

```go
package main

import (
    "io"
    "log"

    "github.com/rarebek/wt"
    "github.com/rarebek/wt/middleware"
)

func main() {
    server := wt.New(
        wt.WithAddr(":4433"),
        wt.WithSelfSignedTLS(),
    )

    log.Printf("Cert hash: %s", server.CertHash())

    server.Use(middleware.DefaultLogger())
    server.Use(middleware.Recover(nil))

    // Echo server: streams echo back, datagrams echo back
    server.Handle("/echo", func(c *wt.Context) {
        // Echo datagrams
        go func() {
            for {
                data, err := c.ReceiveDatagram()
                if err != nil {
                    return
                }
                c.SendDatagram(data)
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
                io.Copy(stream, stream)
            }()
        }
    })

    log.Fatal(server.ListenAndServe())
}
```

Connect from the browser:

```javascript
const transport = new WebTransport("https://localhost:4433/echo", {
    serverCertificateHashes: [{
        algorithm: "sha-256",
        value: hexToBytes("PASTE_CERT_HASH_HERE")
    }]
});
await transport.ready;
```

## Features

### Path Routing with Parameters

```go
server.Handle("/game/{id}", func(c *wt.Context) {
    gameID := c.Param("id")
    // ...
})
```

### Middleware

```go
// Global
server.Use(middleware.DefaultLogger())
server.Use(middleware.Recover(nil))
server.Use(middleware.RateLimit(100))

// Per-group
admin := server.Group("/admin", middleware.BearerAuth(validateToken))
admin.Handle("/control", handleAdmin)

// Per-route
server.Handle("/public", handler, middleware.CORS(middleware.CORSConfig{
    AllowedOrigins: []string{"https://example.com"},
}))
```

Built-in middleware:
- `Logger` — structured logging via slog
- `Recover` — panic recovery
- `RateLimit` — per-IP connection limiting
- `TokenBucket` — per-session message rate limiting
- `BearerAuth` / `QueryAuth` / `RequireKey` — authentication
- `MaxSessions` — global session limit
- `CORS` — origin validation
- `Timeout` — session timeout
- `Compress` — gzip/deflate compression
- `Metrics` — session counting and duration tracking

### Rooms (Pub/Sub)

```go
rooms := wt.NewRoomManager()

server.Handle("/chat/{room}", func(c *wt.Context) {
    room := rooms.GetOrCreate(c.Param("room"))
    room.Join(c)
    defer room.Leave(c)

    room.Broadcast([]byte("someone joined"))
    room.BroadcastExcept([]byte("hello"), c.ID())
})
```

### Typed Streams (Generics)

```go
stream, _ := c.AcceptStream()
typed := wt.NewTypedStream[InputMsg, OutputMsg](stream, codec.JSON{})

input, err := typed.Read()   // auto-decoded InputMsg
typed.Write(OutputMsg{...})  // auto-encoded OutputMsg
```

### Datagrams (Unreliable, Fast)

```go
// Send — fire and forget, no head-of-line blocking
c.SendDatagram(positionData)

// Receive
data, err := c.ReceiveDatagram()
```

### Message Framing

```go
stream, _ := c.AcceptStream()

// Length-prefixed messages (4-byte header + payload)
stream.WriteMessage([]byte("hello"))
msg, _ := stream.ReadMessage()
```

### WebSocket Fallback

```go
import "github.com/rarebek/wt/fallback"

// Serve WebSocket fallback on a separate HTTP port
http.Handle("/ws", fallback.Handler(func(conn *fallback.WSConn) {
    // Same stream/datagram API as WebTransport
    stream, _ := conn.AcceptStream()
    data, _ := conn.ReceiveDatagram()
}))
```

### Self-Signed Certificates

```go
server := wt.New(wt.WithSelfSignedTLS())
fmt.Println("Browser cert hash:", server.CertHash())
```

## Architecture

```
wt (core)
├── Server          — config, lifecycle, route registration
├── Router          — path matching with {param} extraction
├── Context         — session wrapper with params, store, helpers
├── Stream          — bidirectional stream with message framing
├── SendStream      — unidirectional send
├── ReceiveStream   — unidirectional receive
├── TypedStream     — generics-based typed read/write
├── SessionStore    — active session tracking
├── RoomManager     — named rooms with pub/sub
├── Room            — member management, broadcast
│
├── codec/          — message encoding (JSON, MsgPack)
├── middleware/      — auth, logging, rate limit, compress, metrics
├── client/         — Go client with auto-reconnect
└── fallback/       — WebSocket fallback with stream multiplexing
```

## Performance

Benchmarks on Intel i9-14900K:

| Operation | ns/op | allocs/op |
|-----------|-------|-----------|
| Router match | 130 | 2 |
| Middleware chain (3 layers) | 5.9 | 0 |
| Context Get | 10.5 | 0 |
| Session store lookup | 11 | 0 |
| E2E echo (64B over QUIC) | 53,580 | 26 |

## Browser Support

| Browser | WebTransport | Fallback |
|---------|-------------|----------|
| Chrome 97+ | Yes | - |
| Edge 97+ | Yes | - |
| Firefox 114+ | Yes | - |
| Safari | Coming (Interop 2026) | WebSocket |

## License

MIT
