# Changelog

## v0.1.0 (unreleased)

Initial release of the wt WebTransport framework.

### Core
- Server with functional options configuration
- Path-based router with `{param}` and `{param...}` catch-all support
- Session Context with key-value store, path params, lifecycle hooks
- Stream types: bidirectional, unidirectional send/receive
- Length-prefixed message framing (WriteMessage/ReadMessage)
- StreamMux for sub-stream routing by type ID
- Convenience handlers: HandleStream, HandleDatagram, HandleBoth
- TypedStream and TypedDatagram with Go generics
- TypedRoom for type-safe broadcast
- RoomManager with rooms, presence, history (RingBuffer-backed)
- EventBus for session lifecycle pub/sub
- JSON-RPC over streams (RPCServer, RPCClient, CallTyped)
- KVSync for synchronized key-value state
- Session resume with cryptographic tokens
- Backpressure-aware writer with drop detection
- Reliable datagram layer (seq + ack + retransmit)
- Datagram batching with MTU-aware encoding
- Stream interceptors (OnRead/OnWrite hooks)
- Per-message gzip compression (CompressedStream)
- Context-aware streams (WithContext, WithTimeout, WithDeadline)
- Connection migration monitoring (MigrationWatcher)
- NAT keep-alive utility
- Flow control monitoring (FlowControlMonitor)
- Health check HTTP endpoint
- Preflight configuration validator
- Alt-Svc header for HTTP/3 discovery
- pprof debug endpoint (PProfMux, DebugMux)
- Error types: SessionCloseError, StreamCloseError, ConnectionError, UpgradeError, MessageError
- Message validation (Validator interface, RequiredFields)
- Stream priority hints
- QUIC config presets (Default, GameServer, HighThroughput)
- TLS cert rotation without restart (CertRotator)
- Let's Encrypt auto-cert (WithAutoCert)
- Self-signed cert generation with browser hash output

### Middleware (24)
- Logger, Recover, RateLimit, TokenBucket, MaxSessions
- BearerAuth, QueryAuth, RequireKey
- CORS, Timeout, IdleTimeout, SessionTimeoutWithWarning
- Compress (gzip/deflate), RequestID, Tracing
- Metrics, PrometheusMetrics (with histogram), Bandwidth
- OTelTracing (pluggable Tracer interface), SlogAttrs
- RouteRateLimit, PerPathRateLimit, CircuitBreaker, DepthGuard

### Codecs (3)
- JSON, MsgPack (encode), CBOR (encode)

### Fallback Transports
- WebSocket with multiplexed streams (fallback/websocket.go)
- Server-Sent Events (fallback/sse.go)

### Client SDK
- Client with auto-reconnect and exponential backoff
- ZeroRTTClient with TLS session ticket caching
- Connection pool for server-to-server communication

### Examples (10)
- echo, chat, gameserver, iot, aistream, proxy
- notification, collab, fallback_demo, dualprotocol
- Browser: WebTransport tester (index.html), chat UI (chat.html)

### Testing
- 210 tests (unit, integration, stress, chaos, fuzz, memory, scale)
- 25+ benchmarks
- 3 fuzz tests (6.9M+ executions)
- 100-session scale test
- 1000-session memory test (~1KB/session)
- WebTransport vs WebSocket comparison benchmarks

### Infrastructure
- GitHub Actions CI (4 Go versions, fuzz, bench, lint)
- Makefile, .gitignore, CONTRIBUTING.md, README.md, LICENSE
