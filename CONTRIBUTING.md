# Contributing to wt

## Development Setup

```bash
git clone https://github.com/rarebek/wt
cd wt
go build ./...
go test ./...
```

Requires Go 1.22+.

## Running Tests

```bash
# All tests
make test

# Benchmarks
make bench

# Fuzz tests
make fuzz

# Lint
make vet
```

## Project Structure

```
wt.go           — Server, config, groups
router.go       — Path routing with {params}
context.go      — Session wrapper
stream.go       — Stream types with message framing
middleware.go    — Handler/middleware types
session.go      — SessionStore, RoomManager, Room
typed.go        — Generic TypedStream/TypedDatagram
mux.go          — StreamMux for sub-stream routing
convenience.go  — HandleStream, HandleDatagram, HandleBoth
errors.go       — Error types and codes
health.go       — HTTP health check endpoint
keepalive.go    — NAT keep-alive utility
batch.go        — Datagram batching
backpressure.go — Backpressure-aware writer
pipe.go         — Stream piping utility
autocert.go     — Let's Encrypt auto-cert
ctxstream.go    — Context-aware streams

codec/          — Message encoding (JSON, MsgPack)
middleware/     — Built-in middleware (auth, logger, etc.)
client/         — Go client SDK with reconnect + 0-RTT
fallback/       — WebSocket fallback with multiplexing
examples/       — Example applications
wtbench/        — Load testing tool
```

## Adding a Feature

1. Write the code in the appropriate file
2. Write tests (unit + integration if network-dependent)
3. Run `go test ./...` — all must pass
4. Run `go vet ./...` — must be clean
5. Add a godoc example if the feature is user-facing
6. Update README.md if the feature is significant

## Adding Middleware

Create a new file in `middleware/`:
1. The middleware function returns `wt.MiddlewareFunc`
2. Call `next(c)` to pass control to the next handler
3. Don't call `next(c)` to abort (e.g., auth failure)
4. Add a test file
5. Add to the README middleware list

## Commit Style

- Short, clear messages
- No AI attribution
- Examples: "add gzip middleware", "fix router wildcard matching", "optimize batch decode"
