package middleware

import (
	"context"
	"log/slog"

	"github.com/rarebek/wt"
)

// Tracer is an interface for distributed tracing integration.
// Implement this with your tracing library (OpenTelemetry, Jaeger, etc.)
// without importing those libraries as dependencies of the wt framework.
type Tracer interface {
	// StartSpan starts a new trace span and returns the span context.
	// The span should be named with the given operation name.
	StartSpan(ctx context.Context, operation string) (context.Context, TraceSpan)
}

// TraceSpan represents an active trace span.
type TraceSpan interface {
	// SetAttribute sets a key-value attribute on the span.
	SetAttribute(key string, value any)
	// SetStatus marks the span as error or ok.
	SetStatus(err error)
	// End completes the span.
	End()
}

// OTelTracing returns middleware that creates trace spans for each session
// using the provided Tracer interface. This allows OpenTelemetry integration
// without importing OTel as a dependency of the framework.
//
// Usage with OpenTelemetry:
//
//	type myTracer struct {
//	    tracer trace.Tracer
//	}
//
//	func (t *myTracer) StartSpan(ctx context.Context, op string) (context.Context, middleware.TraceSpan) {
//	    ctx, span := t.tracer.Start(ctx, op)
//	    return ctx, &mySpan{span}
//	}
//
//	server.Use(middleware.OTelTracing(&myTracer{tracer: otel.Tracer("wt")}))
func OTelTracing(tracer Tracer) wt.MiddlewareFunc {
	return func(c *wt.Context, next wt.HandlerFunc) {
		ctx := c.Context()
		path := c.Request().URL.Path

		ctx, span := tracer.StartSpan(ctx, "wt.session "+path)
		defer span.End()

		span.SetAttribute("wt.session.id", c.ID())
		span.SetAttribute("wt.path", path)
		span.SetAttribute("wt.remote_addr", c.RemoteAddr().String())
		span.SetAttribute("wt.params", c.Params())

		// Store the span context for sub-spans
		c.Set("_trace_ctx", ctx)
		c.Set("_trace_span", span)

		next(c)
	}
}

// GetTraceSpan retrieves the active trace span from the context.
// Returns nil if OTelTracing middleware wasn't applied.
func GetTraceSpan(c *wt.Context) TraceSpan {
	v, ok := c.Get("_trace_span")
	if !ok {
		return nil
	}
	span, _ := v.(TraceSpan)
	return span
}

// GetTraceContext retrieves the trace context for creating sub-spans.
func GetTraceContext(c *wt.Context) context.Context {
	v, ok := c.Get("_trace_ctx")
	if !ok {
		return c.Context()
	}
	ctx, _ := v.(context.Context)
	return ctx
}

// NoopTracer is a tracer that does nothing. Useful for testing.
type NoopTracer struct{}

func (NoopTracer) StartSpan(ctx context.Context, _ string) (context.Context, TraceSpan) {
	return ctx, &noopSpan{}
}

type noopSpan struct{}

func (noopSpan) SetAttribute(_ string, _ any) {}
func (noopSpan) SetStatus(_ error)            {}
func (noopSpan) End()                         {}

// LogTracer is a tracer that logs spans via slog. Useful for development.
type LogTracer struct {
	Logger *slog.Logger
}

func (lt *LogTracer) StartSpan(ctx context.Context, operation string) (context.Context, TraceSpan) {
	logger := lt.Logger
	if logger == nil {
		logger = slog.Default()
	}
	logger.Info("span start", "operation", operation)
	return ctx, &logSpan{logger: logger, operation: operation}
}

type logSpan struct {
	logger    *slog.Logger
	operation string
}

func (ls *logSpan) SetAttribute(key string, value any) {
	ls.logger.Debug("span attribute", "operation", ls.operation, "key", key, "value", value)
}

func (ls *logSpan) SetStatus(err error) {
	if err != nil {
		ls.logger.Error("span error", "operation", ls.operation, "error", err)
	}
}

func (ls *logSpan) End() {
	ls.logger.Info("span end", "operation", ls.operation)
}
