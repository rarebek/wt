package wt

import (
	"fmt"
	"testing"
)

func TestSessionStoreFilter(t *testing.T) {
	ss := NewSessionStore()
	for i := range 10 {
		ctx := &Context{
			id:    fmt.Sprintf("s-%d", i),
			store: map[string]any{"role": "user"},
		}
		if i < 3 {
			ctx.store["role"] = "admin"
		}
		ss.Add(ctx)
	}

	admins := ss.Filter(func(c *Context) bool {
		role, _ := c.Get("role")
		return role == "admin"
	})

	if len(admins) != 3 {
		t.Errorf("expected 3 admins, got %d", len(admins))
	}
}

func TestSessionStoreCountWhere(t *testing.T) {
	ss := NewSessionStore()
	for i := range 20 {
		ctx := &Context{
			id:    fmt.Sprintf("s-%d", i),
			store: map[string]any{"active": i%2 == 0},
		}
		ss.Add(ctx)
	}

	count := ss.CountWhere(func(c *Context) bool {
		v, _ := c.Get("active")
		return v == true
	})

	if count != 10 {
		t.Errorf("expected 10, got %d", count)
	}
}
