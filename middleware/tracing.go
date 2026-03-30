package middleware

import (
	"context"
	"log/slog"
	"time"

	"github.com/rarebek/wt"
)

// TraceIDKey is the context key for trace IDs.
const TraceIDKey = "_trace_id"

// Tracing returns middleware that adds basic tracing to sessions.
// It generates a trace ID and logs session start/end with timing.
//
// For OpenTelemetry integration, wrap this with your own middleware
// that creates OTel spans using the trace ID from context.
func Tracing(logger *slog.Logger) wt.MiddlewareFunc {
	if logger == nil {
		logger = slog.Default()
	}

	return func(c *wt.Context, next wt.HandlerFunc) {
		traceID := c.ID() // Use session ID as trace ID
		c.Set(TraceIDKey, traceID)

		start := time.Now()
		logger.Info("trace start",
			"trace_id", traceID,
			"path", c.Request().URL.Path,
			"remote", c.RemoteAddr().String(),
			"params", c.Params(),
		)

		next(c)

		logger.Info("trace end",
			"trace_id", traceID,
			"duration_ms", time.Since(start).Milliseconds(),
		)
	}
}

// GetTraceID retrieves the trace ID from a context.
func GetTraceID(c *wt.Context) string {
	return c.GetString(TraceIDKey)
}

// WithTraceContext returns a Go context.Context with the trace ID attached.
// Useful for passing trace context to downstream services.
func WithTraceContext(ctx context.Context, c *wt.Context) context.Context {
	traceID := GetTraceID(c)
	if traceID == "" {
		return ctx
	}
	return context.WithValue(ctx, traceContextKey{}, traceID)
}

type traceContextKey struct{}

// TraceIDFromContext retrieves a trace ID from a Go context.
func TraceIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(traceContextKey{}).(string)
	return v
}
