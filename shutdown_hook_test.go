package wt

import (
	"context"
	"testing"
	"time"
)

func TestOnShutdownHooks(t *testing.T) {
	server := New(WithAddr(":0"), WithSelfSignedTLS())

	var order []string

	server.OnShutdown(func() { order = append(order, "hook1") })
	server.OnShutdown(func() { order = append(order, "hook2") })
	server.OnShutdown(func() { order = append(order, "hook3") })

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	server.Shutdown(ctx)

	if len(order) != 3 {
		t.Fatalf("expected 3 hooks, got %d", len(order))
	}
	if order[0] != "hook1" || order[1] != "hook2" || order[2] != "hook3" {
		t.Errorf("wrong order: %v", order)
	}
}
