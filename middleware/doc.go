/*
Package middleware provides built-in middleware for the wt WebTransport framework.

# Middleware Ordering

Middleware executes in the order it's added. Global middleware runs before
route-specific middleware. The handler runs last.

	server.Use(A)  // runs 1st
	server.Use(B)  // runs 2nd
	server.Handle("/path", handler, C, D)
	// Execution: A → B → C → D → handler → D-after → C-after → B-after → A-after

# Recommended Order

	server.Use(middleware.Recover(nil))       // 1. Catch panics (outermost)
	server.Use(middleware.RequestID())        // 2. Assign request ID
	server.Use(middleware.DefaultLogger())    // 3. Log with request ID
	server.Use(middleware.RateLimit(100))     // 4. Reject excess before auth
	server.Use(middleware.BearerAuth(fn))     // 5. Authenticate
	server.Use(middleware.MaxSessions(1000))  // 6. Global capacity

# Available Middleware

Authentication:
  - [BearerAuth] — validate Bearer token from Authorization header
  - [QueryAuth] — validate token from query parameter
  - [RequireKey] — check static API key

Rate Limiting:
  - [RateLimit] — limit concurrent sessions per IP
  - [TokenBucket] — per-session message rate limiting
  - [RouteRateLimit] — limit concurrent sessions per route
  - [PerPathRateLimit] — limit concurrent sessions per resolved path
  - [MaxSessions] — global session limit

Observability:
  - [Logger] / [DefaultLogger] — structured session logging via slog
  - [Tracing] — trace ID assignment and lifecycle logging
  - [Metrics] — basic session counting
  - [PrometheusMetrics] — Prometheus-compatible metrics endpoint
  - [RequestID] — unique ID per session

Resilience:
  - [Recover] — panic recovery
  - [Timeout] — session timeout
  - [SessionTimeoutWithWarning] — timeout with warning datagram
  - [Compress] — gzip/deflate compression
  - [CircuitBreaker] — trip on repeated failures
  - [DepthGuard] — limit concurrent stream handlers per session

Networking:
  - [IPWhitelist] — allow only specific IPs/CIDRs
  - [IPBlacklist] — block specific IPs/CIDRs (runtime-updatable)
  - [BlockUserAgent] / [RequireUserAgent] — user agent filtering
  - [ExtractHeader] / [ExtractHeaders] — extract handshake headers into context

Session:
  - [Bandwidth] — track bytes sent/received
  - [SlogAttrs] — add session attrs to slog
  - [SessionData] — extract UA, origin, query params into context
  - [OTelTracing] — pluggable distributed tracing
  - [IdleMonitor] — per-session idle detection

Origin Control:
  - [CORS] — origin validation
*/
package middleware
