package codec

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

// Binary is a codec using encoding/binary for fixed-size structs.
// Much faster than JSON for known struct layouts, but requires
// fixed-size fields (no strings, slices, or maps).
type Binary struct {
	order binary.ByteOrder
}

// NewBinaryCodec creates a binary codec with the given byte order.
func NewBinaryCodec(order binary.ByteOrder) Binary {
	return Binary{order: order}
}

// BigEndian returns a binary codec with big-endian byte order.
func BigEndian() Binary { return Binary{order: binary.BigEndian} }

// LittleEndian returns a binary codec with little-endian byte order.
func LittleEndian() Binary { return Binary{order: binary.LittleEndian} }

func (b Binary) Name() string { return "binary" }

func (b Binary) Marshal(v any) ([]byte, error) {
	var buf bytes.Buffer
	if err := binary.Write(&buf, b.order, v); err != nil {
		return nil, fmt.Errorf("binary marshal: %w", err)
	}
	return buf.Bytes(), nil
}

func (b Binary) Unmarshal(data []byte, v any) error {
	return binary.Read(bytes.NewReader(data), b.order, v)
}
