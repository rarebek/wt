package middleware_test

import (
	"fmt"
	"log/slog"

	"github.com/rarebek/wt"
	"github.com/rarebek/wt/middleware"
)

func ExampleDefaultLogger() {
	server := wt.New(wt.WithAddr(":4433"), wt.WithSelfSignedTLS())
	server.Use(middleware.DefaultLogger())
	fmt.Println("Logger middleware added")
	// Output: Logger middleware added
}

func ExampleRecover() {
	server := wt.New(wt.WithAddr(":4433"), wt.WithSelfSignedTLS())
	server.Use(middleware.Recover(nil))
	fmt.Println("Recover middleware added")
	// Output: Recover middleware added
}

func ExampleRateLimit() {
	server := wt.New(wt.WithAddr(":4433"), wt.WithSelfSignedTLS())
	server.Use(middleware.RateLimit(100)) // max 100 connections per IP
	fmt.Println("Rate limit set to 100/IP")
	// Output: Rate limit set to 100/IP
}

func ExampleBearerAuth() {
	validateToken := func(token string) (any, error) {
		if token == "valid" {
			return "user-123", nil
		}
		return nil, fmt.Errorf("invalid")
	}

	server := wt.New(wt.WithAddr(":4433"), wt.WithSelfSignedTLS())
	server.Use(middleware.BearerAuth(validateToken))
	fmt.Println("Auth middleware added")
	// Output: Auth middleware added
}

func ExampleMaxSessions() {
	server := wt.New(wt.WithAddr(":4433"), wt.WithSelfSignedTLS())
	server.Use(middleware.MaxSessions(1000, nil))
	fmt.Println("Max sessions set to 1000")
	// Output: Max sessions set to 1000
}

func ExampleCORS() {
	server := wt.New(wt.WithAddr(":4433"), wt.WithSelfSignedTLS())
	server.Use(middleware.CORS(middleware.CORSConfig{
		AllowedOrigins: []string{"https://example.com"},
	}))
	fmt.Println("CORS configured")
	// Output: CORS configured
}

func ExampleTimeout() {
	server := wt.New(wt.WithAddr(":4433"), wt.WithSelfSignedTLS())
	// Sessions automatically close after 30 minutes
	server.Use(middleware.Timeout(30 * 60 * 1e9)) // 30 minutes
	fmt.Println("Timeout set")
	// Output: Timeout set
}

func ExampleNewPrometheusMetrics() {
	pm := middleware.NewPrometheusMetrics()

	server := wt.New(wt.WithAddr(":4433"), wt.WithSelfSignedTLS())
	server.Use(pm.Middleware())

	// Serve metrics on a separate port
	// go http.ListenAndServe(":9090", pm.Handler())

	fmt.Println("Prometheus metrics enabled")
	// Output: Prometheus metrics enabled
}

func ExampleTracing() {
	server := wt.New(wt.WithAddr(":4433"), wt.WithSelfSignedTLS())
	server.Use(middleware.Tracing(slog.Default()))
	fmt.Println("Tracing middleware added")
	// Output: Tracing middleware added
}

func ExampleRequestID() {
	server := wt.New(wt.WithAddr(":4433"), wt.WithSelfSignedTLS())
	server.Use(middleware.RequestID())
	fmt.Println("RequestID middleware added")
	// Output: RequestID middleware added
}
