package wt

import (
	"testing"
)

func TestBuildChainNoMiddleware(t *testing.T) {
	called := false
	handler := HandlerFunc(func(c *Context) {
		called = true
	})

	chain := buildChain(handler, nil)
	chain(nil) // context can be nil for this test

	if !called {
		t.Error("handler was not called")
	}
}

func TestBuildChainOrder(t *testing.T) {
	var order []string

	mw1 := MiddlewareFunc(func(c *Context, next HandlerFunc) {
		order = append(order, "mw1-before")
		next(c)
		order = append(order, "mw1-after")
	})

	mw2 := MiddlewareFunc(func(c *Context, next HandlerFunc) {
		order = append(order, "mw2-before")
		next(c)
		order = append(order, "mw2-after")
	})

	handler := HandlerFunc(func(c *Context) {
		order = append(order, "handler")
	})

	chain := buildChain(handler, []MiddlewareFunc{mw1, mw2})
	chain(nil)

	expected := []string{"mw1-before", "mw2-before", "handler", "mw2-after", "mw1-after"}
	if len(order) != len(expected) {
		t.Fatalf("expected %d calls, got %d: %v", len(expected), len(order), order)
	}
	for i, v := range expected {
		if order[i] != v {
			t.Errorf("order[%d] = %q, want %q", i, order[i], v)
		}
	}
}

func TestBuildChainAbort(t *testing.T) {
	handlerCalled := false

	mw := MiddlewareFunc(func(c *Context, next HandlerFunc) {
		// Don't call next — abort
	})

	handler := HandlerFunc(func(c *Context) {
		handlerCalled = true
	})

	chain := buildChain(handler, []MiddlewareFunc{mw})
	chain(nil)

	if handlerCalled {
		t.Error("handler should not have been called when middleware aborts")
	}
}
