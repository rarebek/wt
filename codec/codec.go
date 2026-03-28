// Package codec provides message encoding/decoding for the wt framework.
package codec

import (
	"encoding/json"
	"fmt"
)

// Codec defines the interface for message serialization.
type Codec interface {
	// Marshal encodes a value into bytes.
	Marshal(v any) ([]byte, error)
	// Unmarshal decodes bytes into a value.
	Unmarshal(data []byte, v any) error
	// Name returns the codec name (e.g., "json", "protobuf").
	Name() string
}

// JSON is a codec using encoding/json.
type JSON struct{}

func (JSON) Marshal(v any) ([]byte, error)         { return json.Marshal(v) }
func (JSON) Unmarshal(data []byte, v any) error     { return json.Unmarshal(data, v) }
func (JSON) Name() string                           { return "json" }

// Registry holds named codecs.
type Registry struct {
	codecs   map[string]Codec
	fallback Codec
}

// NewRegistry creates a Registry with JSON as the default.
func NewRegistry() *Registry {
	j := JSON{}
	return &Registry{
		codecs:   map[string]Codec{"json": j},
		fallback: j,
	}
}

// Register adds a codec.
func (r *Registry) Register(c Codec) {
	r.codecs[c.Name()] = c
}

// Get returns a codec by name.
func (r *Registry) Get(name string) (Codec, error) {
	c, ok := r.codecs[name]
	if !ok {
		return nil, fmt.Errorf("codec %q not registered", name)
	}
	return c, nil
}

// Default returns the fallback codec.
func (r *Registry) Default() Codec {
	return r.fallback
}

// SetDefault sets the default codec by name.
func (r *Registry) SetDefault(name string) error {
	c, err := r.Get(name)
	if err != nil {
		return err
	}
	r.fallback = c
	return nil
}
