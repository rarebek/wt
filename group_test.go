package wt

import "testing"

func TestGroupHandle(t *testing.T) {
	server := New(WithAddr(":0"), WithSelfSignedTLS())

	api := server.Group("/api")
	api.Handle("/users", func(c *Context) {})
	api.Handle("/posts", func(c *Context) {})

	routes := server.router.Routes()
	if len(routes) != 2 {
		t.Errorf("expected 2 routes, got %d", len(routes))
	}

	found := false
	for _, r := range routes {
		if r.Pattern == "/api/users" {
			found = true
		}
	}
	if !found {
		t.Error("expected /api/users route")
	}
}

func TestGroupMiddleware(t *testing.T) {
	server := New(WithAddr(":0"), WithSelfSignedTLS())

	api := server.Group("/api", func(c *Context, next HandlerFunc) {
		next(c)
	})
	api.Handle("/test", func(c *Context) {})

	routes := server.router.Routes()
	if len(routes) != 1 {
		t.Errorf("expected 1 route, got %d", len(routes))
	}
	if len(routes[0].Middleware) != 1 {
		t.Errorf("expected 1 middleware on group route, got %d", len(routes[0].Middleware))
	}
}

func TestGroupUse(t *testing.T) {
	server := New(WithAddr(":0"), WithSelfSignedTLS())

	api := server.Group("/api")
	api.Use(func(c *Context, next HandlerFunc) { next(c) })
	api.Handle("/data", func(c *Context) {})

	routes := server.router.Routes()
	if len(routes[0].Middleware) != 1 {
		t.Errorf("expected 1 middleware from Use, got %d", len(routes[0].Middleware))
	}
}
