package wt

import "testing"

func TestContextSetGetString(t *testing.T) {
	c := &Context{store: make(map[string]any)}
	c.Set("name", "alice")

	if c.GetString("name") != "alice" {
		t.Errorf("expected 'alice', got %q", c.GetString("name"))
	}
	if c.GetString("missing") != "" {
		t.Error("missing key should return empty string")
	}
}

func TestContextMustGetPanics(t *testing.T) {
	c := &Context{store: make(map[string]any)}

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic from MustGet on missing key")
		}
	}()
	c.MustGet("nonexistent")
}

func TestContextMustGetSuccess(t *testing.T) {
	c := &Context{store: make(map[string]any)}
	c.Set("key", 42)

	v := c.MustGet("key")
	if v != 42 {
		t.Errorf("expected 42, got %v", v)
	}
}

func TestContextParams(t *testing.T) {
	c := &Context{
		params: map[string]string{"room": "lobby", "id": "123"},
		store:  make(map[string]any),
	}

	if c.Param("room") != "lobby" {
		t.Errorf("expected 'lobby', got %q", c.Param("room"))
	}
	if c.Param("id") != "123" {
		t.Errorf("expected '123', got %q", c.Param("id"))
	}
	if c.Param("missing") != "" {
		t.Error("missing param should return empty")
	}

	params := c.Params()
	if len(params) != 2 {
		t.Errorf("expected 2 params, got %d", len(params))
	}
}

func TestContextID(t *testing.T) {
	c := &Context{id: "test-id-123", store: make(map[string]any)}
	if c.ID() != "test-id-123" {
		t.Errorf("expected 'test-id-123', got %q", c.ID())
	}
}
