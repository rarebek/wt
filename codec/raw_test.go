package codec

import (
	"bytes"
	"testing"
)

func TestRawCodec(t *testing.T) {
	c := Raw{}
	if c.Name() != "raw" {
		t.Errorf("expected 'raw', got %q", c.Name())
	}

	data := []byte("hello raw")
	encoded, err := c.Marshal(data)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(encoded, data) {
		t.Error("raw marshal should pass through")
	}

	var decoded []byte
	c.Unmarshal(encoded, &decoded)
	if !bytes.Equal(decoded, data) {
		t.Error("raw unmarshal should pass through")
	}
}

func TestRawCodecNonBytes(t *testing.T) {
	c := Raw{}
	result, _ := c.Marshal("not bytes")
	if result != nil {
		t.Error("non-[]byte should return nil")
	}
}
