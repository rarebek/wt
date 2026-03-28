package codec

import (
	"testing"
)

func TestJSONCodec(t *testing.T) {
	c := JSON{}

	if c.Name() != "json" {
		t.Errorf("expected name 'json', got %q", c.Name())
	}

	type msg struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	original := msg{Name: "test", Value: 42}

	data, err := c.Marshal(original)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var decoded msg
	err = c.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if decoded.Name != original.Name || decoded.Value != original.Value {
		t.Errorf("decoded %+v != original %+v", decoded, original)
	}
}

func TestRegistry(t *testing.T) {
	r := NewRegistry()

	// Default should be JSON
	if r.Default().Name() != "json" {
		t.Errorf("expected default codec 'json', got %q", r.Default().Name())
	}

	// Get existing
	c, err := r.Get("json")
	if err != nil {
		t.Fatalf("expected json codec, got error: %v", err)
	}
	if c.Name() != "json" {
		t.Errorf("expected 'json', got %q", c.Name())
	}

	// Get nonexistent
	_, err = r.Get("protobuf")
	if err == nil {
		t.Error("expected error for unregistered codec")
	}
}

func TestRegistryRegister(t *testing.T) {
	r := NewRegistry()

	// Register a custom codec
	custom := &mockCodec{name: "mock"}
	r.Register(custom)

	c, err := r.Get("mock")
	if err != nil {
		t.Fatalf("expected mock codec, got error: %v", err)
	}
	if c.Name() != "mock" {
		t.Errorf("expected 'mock', got %q", c.Name())
	}
}

func TestRegistrySetDefault(t *testing.T) {
	r := NewRegistry()

	custom := &mockCodec{name: "custom"}
	r.Register(custom)

	err := r.SetDefault("custom")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Default().Name() != "custom" {
		t.Errorf("expected default 'custom', got %q", r.Default().Name())
	}

	err = r.SetDefault("nonexistent")
	if err == nil {
		t.Error("expected error setting nonexistent default")
	}
}

type mockCodec struct {
	name string
}

func (m *mockCodec) Marshal(v any) ([]byte, error)     { return nil, nil }
func (m *mockCodec) Unmarshal(data []byte, v any) error { return nil }
func (m *mockCodec) Name() string                      { return m.name }
