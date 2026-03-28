package wt

import (
	"testing"
	"time"
)

func TestResumeStoreSaveRestore(t *testing.T) {
	rs := NewResumeStore(5 * time.Minute)

	// Create a context with some state
	ctx := &Context{
		store: map[string]any{
			"user":  "alice",
			"role":  "admin",
			"score": 42,
		},
	}

	token := rs.Save(ctx)
	if token == "" {
		t.Fatal("expected non-empty token")
	}

	if rs.Count() != 1 {
		t.Errorf("expected 1 stored state, got %d", rs.Count())
	}

	// Restore to new context
	newCtx := &Context{
		store: make(map[string]any),
	}

	ok := rs.Restore(newCtx, token)
	if !ok {
		t.Fatal("expected successful restore")
	}

	// Verify state was restored
	user, _ := newCtx.Get("user")
	if user != "alice" {
		t.Errorf("expected user 'alice', got %v", user)
	}

	role, _ := newCtx.Get("role")
	if role != "admin" {
		t.Errorf("expected role 'admin', got %v", role)
	}

	score, _ := newCtx.Get("score")
	if score != 42 {
		t.Errorf("expected score 42, got %v", score)
	}

	// Token should be single-use
	if rs.Count() != 0 {
		t.Errorf("expected 0 stored states after restore, got %d", rs.Count())
	}
}

func TestResumeStoreSingleUse(t *testing.T) {
	rs := NewResumeStore(5 * time.Minute)

	ctx := &Context{
		store: map[string]any{"key": "value"},
	}

	token := rs.Save(ctx)

	// First restore succeeds
	newCtx1 := &Context{store: make(map[string]any)}
	ok := rs.Restore(newCtx1, token)
	if !ok {
		t.Fatal("first restore should succeed")
	}

	// Second restore with same token fails (single-use)
	newCtx2 := &Context{store: make(map[string]any)}
	ok = rs.Restore(newCtx2, token)
	if ok {
		t.Error("second restore should fail (token is single-use)")
	}
}

func TestResumeStoreExpiry(t *testing.T) {
	rs := NewResumeStore(50 * time.Millisecond)

	ctx := &Context{
		store: map[string]any{"key": "value"},
	}

	token := rs.Save(ctx)

	// Wait for expiry
	time.Sleep(100 * time.Millisecond)

	newCtx := &Context{store: make(map[string]any)}
	ok := rs.Restore(newCtx, token)
	if ok {
		t.Error("expired token should not restore")
	}
}

func TestResumeStoreInvalidToken(t *testing.T) {
	rs := NewResumeStore(5 * time.Minute)

	newCtx := &Context{store: make(map[string]any)}
	ok := rs.Restore(newCtx, "nonexistent-token")
	if ok {
		t.Error("invalid token should not restore")
	}
}
