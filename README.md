# wt

`wt` is a Go framework for [WebTransport](https://www.w3.org/TR/webtransport/) — the protocol that gives you multiplexed streams and unreliable datagrams over QUIC, without TCP's head-of-line blocking.

If you've used Gin or Echo for HTTP, `wt` is the same idea but for WebTransport. You get routing, middleware, rooms, and message framing without touching raw QUIC.

Built on [quic-go/webtransport-go](https://github.com/quic-go/webtransport-go).

```go
server := wt.New(wt.WithAddr(":4433"), wt.WithSelfSignedTLS())

server.Handle("/echo", wt.HandleStream(func(s *wt.Stream, c *wt.Context) {
    defer s.Close()
    for msg := range wt.Messages(s) {
        s.WriteMessage(msg) // echo back
    }
}))

server.ListenAndServe()
```

## What is WebTransport and why should I care

WebSocket gives you one bidirectional pipe over TCP. If any packet is lost, everything behind it waits. You can't send unreliable data. You can't have independent channels.

WebTransport fixes this:

- **Multiple streams** on one connection. Chat on stream 1, game state on stream 2, file upload on stream 3. They don't block each other.
- **Datagrams** — fire-and-forget messages like UDP. Perfect for player positions at 60Hz where you don't care about the packet from 16ms ago.
- **No head-of-line blocking** — a lost packet only stalls the stream it belongs to, not everything.
- **Connection migration** — phone switches from WiFi to cellular without dropping the connection.

The catch: WebTransport is newer (Chrome 97+, Firefox 114+, Safari coming 2026), and the Go libraries only give you raw protocol access. `wt` adds the framework layer.

## Install

```
go get github.com/rarebek/wt
```

Requires Go 1.25+.

## Examples

### Echo server

Accepts streams, echoes back whatever is sent:

```go
server := wt.New(wt.WithAddr(":4433"), wt.WithSelfSignedTLS())

server.Handle("/echo", func(c *wt.Context) {
    for stream := range wt.Streams(c) {
        go func() {
            defer stream.Close()
            for msg := range wt.Messages(stream) {
                stream.WriteMessage(msg)
            }
        }()
    }
})

log.Fatal(server.ListenAndServe())
```

### Chat with rooms

Multiple users join a room by path. Messages broadcast to everyone:

```go
rooms := wt.NewRoomManager()

server.Handle("/chat/{room}", func(c *wt.Context) {
    room := rooms.GetOrCreate(c.Param("room"))
    room.Join(c)
    defer room.Leave(c)

    for stream := range wt.Streams(c) {
        go func() {
            defer stream.Close()
            msg, _ := stream.ReadMessage()
            room.BroadcastExcept(msg, c.ID()) // send to everyone else
        }()
    }
})
```

### Game server with datagrams

Player positions via unreliable datagrams (no head-of-line blocking), game events via reliable streams:

```go
server.Handle("/game/{id}", func(c *wt.Context) {
    // Receive player input as datagrams (unreliable, 60Hz)
    go func() {
        for data := range wt.Datagrams(c) {
            updatePlayerPosition(c.ID(), data)
        }
    }()

    // Push world state at 20Hz
    ticker := wt.NewTicker(c, 50*time.Millisecond, func() []byte {
        return getWorldState()
    })
    defer ticker.Stop()

    // Game events via reliable streams (chat, abilities, etc.)
    mux := wt.NewStreamMux()
    mux.Handle(1, handleChat)    // stream type 1 = chat
    mux.Handle(2, handleAbility) // stream type 2 = ability
    mux.Serve(c)
})
```

This is the pattern that WebSocket can't do — datagrams and streams simultaneously, with no interference between them.

### JSON-RPC over streams

```go
rpc := wt.NewRPCServer()
rpc.Register("add", func(params json.RawMessage) (any, error) {
    var args [2]float64
    json.Unmarshal(params, &args)
    return args[0] + args[1], nil
})

server.Handle("/rpc", wt.HandleStream(func(s *wt.Stream, c *wt.Context) {
    rpc.Serve(s)
}))
```

## Routing

Path parameters with `{param}` and catch-all with `{param...}`:

```go
server.Handle("/user/{id}", handler)
server.Handle("/files/{path...}", fileHandler) // matches /files/a/b/c
```

Route groups with shared middleware:

```go
api := server.Group("/api", authMiddleware)
api.Handle("/data", dataHandler)
api.Handle("/config", configHandler)
```

## Middleware

Middleware runs before the handler. Call `next(c)` to continue, or don't to abort:

```go
server.Use(middleware.Recover(nil))     // catch panics
server.Use(middleware.DefaultLogger())  // log sessions via slog
server.Use(middleware.RateLimit(100))   // max 100 sessions per IP
```

Per-route:

```go
server.Handle("/admin", handler, middleware.BearerAuth(validateToken))
```

Included middleware:

| Middleware | What it does |
|-----------|-------------|
| `Recover` | Catches panics, keeps server alive |
| `Logger` / `DefaultLogger` | Structured logging via slog |
| `RateLimit` | Per-IP concurrent session limit |
| `BearerAuth` / `QueryAuth` | Token authentication |
| `CORS` | Origin validation |
| `MaxSessions` | Global session cap |
| `Timeout` | Auto-close sessions after duration |
| `Compress` | Gzip/deflate for large messages |
| `PrometheusMetrics` | Prometheus-compatible `/metrics` endpoint |
| `CircuitBreaker` | Trip on repeated failures |
| `IPWhitelist` / `IPBlacklist` | IP-based access control |
| `SlidingWindowRateLimit` | More accurate than fixed-window rate limiting |

There are 50+ middleware in total — see [middleware/](middleware/) for the full list.

## Streams vs Datagrams

WebTransport offers two transport modes. Pick the right one:

**Streams** — reliable, ordered. Use for messages that must arrive: chat, game events, file transfers.
```go
stream, _ := c.AcceptStream()
stream.WriteMessage([]byte("reliable"))
```

**Datagrams** — unreliable, unordered. Use for data where the latest value matters more than every value: player positions, sensor readings, cursor locations.
```go
c.SendDatagram(positionBytes) // fire and forget
```

You can use both simultaneously on the same connection. They don't interfere with each other.

## Stream multiplexing

One WebTransport session can have many streams. `StreamMux` routes them by a 2-byte type header:

```go
mux := wt.NewStreamMux()
mux.Handle(1, chatHandler)
mux.Handle(2, gameEventHandler)
mux.Handle(3, fileUploadHandler)
mux.Serve(c) // accepts streams and routes by type
```

The client opens a typed stream with `OpenTypedStream(c, typeID)`.

## Message framing

QUIC streams are byte-oriented (like TCP). `wt` adds length-prefixed framing:

```go
stream.WriteMessage([]byte("hello"))  // writes: [4-byte length][payload]
msg, _ := stream.ReadMessage()        // reads one complete message
```

Or use JSON directly:

```go
stream.WriteJSON(myStruct)
stream.ReadJSON(&result)
```

## Client SDK

Go client with auto-reconnect:

```go
import "github.com/rarebek/wt/client"

c := client.New("https://server:4433/app",
    client.WithReconnect(1*time.Second, 30*time.Second),
    client.WithInsecureSkipVerify(),
)
c.Dial(ctx)
defer c.Close()

c.SendDatagram([]byte("hello"))
stream, _ := c.OpenStream(ctx)
```

Connection pool for server-to-server:

```go
pool := client.NewPool("https://backend:4433/api", 4)
c, _ := pool.Get(ctx)
```

## Fallback

Not all browsers support WebTransport yet. The `fallback` package provides:

- **WebSocket** — multiplexed streams over a single WebSocket connection
- **SSE** — server-push via Server-Sent Events

```go
import "github.com/rarebek/wt/fallback"

// WebSocket fallback (same stream/datagram API)
http.Handle("/ws", fallback.Handler(func(conn *fallback.WSConn) {
    stream, _ := conn.AcceptStream()
    conn.ReceiveDatagram()
}))

// SSE fallback (server-push only)
hub := fallback.NewSSEHub()
http.Handle("/events", hub.Handler())
hub.Broadcast("update", data)
```

## WebTransport vs WebSocket

WebSocket is mature and fast. WebTransport is newer and solves different problems. Pick the right one:

**Use WebSocket when:** you need one data channel, you're on a reliable LAN, or you need Safari support today.

**Use WebTransport when:** you need multiple independent channels, unreliable datagrams, mobile clients on lossy networks, or you want to avoid head-of-line blocking.

WebTransport runs over QUIC in userspace. WebSocket runs over TCP with decades of kernel optimization. On localhost with zero packet loss, WebSocket will be faster. On real networks with packet loss, WebTransport avoids the head-of-line blocking that freezes all WebSocket data when a single packet is retransmitted.

## Production

```go
server := wt.New(
    wt.WithAddr(":443"),
    wt.WithAutoCert("yourdomain.com", "/var/cache/certs"),
)

// Preflight check
if issues := server.Preflight(); len(issues) > 0 {
    for _, issue := range issues {
        log.Printf("WARN: %s", issue)
    }
}

// Prometheus metrics
pm := middleware.NewPrometheusMetrics()
server.Use(pm.Middleware())

// Debug/health endpoint
go http.ListenAndServe(":6060", wt.DebugMux(server))

// Graceful shutdown on SIGTERM
wt.ListenAndServeWithGracefulShutdown(server, 30*time.Second)
```

## Project structure

```
wt.go          Server, config, routing, lifecycle
context.go     Session context, params, key-value store
router.go      Path matching with {param} and {param...}
stream.go      Stream types, message framing, RPC, mux
session.go     Rooms, pub/sub, presence, events, tags
datagram.go    Datagram utilities, batching, throttling
errors.go      Error types and codes
stats.go       Health checks, pprof, Prometheus, flow control
middleware.go  Handler and middleware type definitions

client/        Go client SDK with reconnect and pooling
codec/         JSON, MsgPack, CBOR, Binary, Raw
middleware/    50+ middleware components
fallback/      WebSocket and SSE fallback
```

## License

MIT
