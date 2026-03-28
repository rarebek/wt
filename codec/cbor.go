package codec

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"reflect"
)

// CBOR is a lightweight CBOR (Concise Binary Object Representation) codec.
// Implements RFC 8949 major types for basic Go types.
// For full CBOR support, use a dedicated library like github.com/fxamacker/cbor/v2.
type CBOR struct{}

func (CBOR) Name() string { return "cbor" }

func (CBOR) Marshal(v any) ([]byte, error) {
	var buf bytes.Buffer
	if err := cborEncode(&buf, reflect.ValueOf(v)); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (CBOR) Unmarshal(data []byte, v any) error {
	return fmt.Errorf("cbor: Unmarshal not implemented — use github.com/fxamacker/cbor/v2 for full CBOR support")
}

// CBOR major types
const (
	cborUint   byte = 0x00 // Major type 0
	cborNint   byte = 0x20 // Major type 1 (negative integer)
	cborBytes  byte = 0x40 // Major type 2
	cborText   byte = 0x60 // Major type 3
	cborArray  byte = 0x80 // Major type 4
	cborMap    byte = 0xa0 // Major type 5
	cborSimple byte = 0xe0 // Major type 7
)

func cborEncodeHead(buf *bytes.Buffer, major byte, val uint64) {
	if val <= 23 {
		buf.WriteByte(major | byte(val))
	} else if val <= math.MaxUint8 {
		buf.WriteByte(major | 24)
		buf.WriteByte(byte(val))
	} else if val <= math.MaxUint16 {
		buf.WriteByte(major | 25)
		binary.Write(buf, binary.BigEndian, uint16(val))
	} else if val <= math.MaxUint32 {
		buf.WriteByte(major | 26)
		binary.Write(buf, binary.BigEndian, uint32(val))
	} else {
		buf.WriteByte(major | 27)
		binary.Write(buf, binary.BigEndian, val)
	}
}

func cborEncode(buf *bytes.Buffer, v reflect.Value) error {
	if !v.IsValid() {
		buf.WriteByte(0xf6) // null
		return nil
	}

	switch v.Kind() {
	case reflect.Ptr, reflect.Interface:
		if v.IsNil() {
			buf.WriteByte(0xf6) // null
			return nil
		}
		return cborEncode(buf, v.Elem())

	case reflect.Bool:
		if v.Bool() {
			buf.WriteByte(0xf5) // true
		} else {
			buf.WriteByte(0xf4) // false
		}

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n := v.Int()
		if n >= 0 {
			cborEncodeHead(buf, cborUint, uint64(n))
		} else {
			cborEncodeHead(buf, cborNint, uint64(-1-n))
		}

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		cborEncodeHead(buf, cborUint, v.Uint())

	case reflect.Float32:
		buf.WriteByte(0xfa)
		binary.Write(buf, binary.BigEndian, float32(v.Float()))

	case reflect.Float64:
		buf.WriteByte(0xfb)
		binary.Write(buf, binary.BigEndian, v.Float())

	case reflect.String:
		s := v.String()
		cborEncodeHead(buf, cborText, uint64(len(s)))
		buf.WriteString(s)

	case reflect.Slice:
		if v.Type().Elem().Kind() == reflect.Uint8 {
			b := v.Bytes()
			cborEncodeHead(buf, cborBytes, uint64(len(b)))
			buf.Write(b)
			return nil
		}
		cborEncodeHead(buf, cborArray, uint64(v.Len()))
		for i := range v.Len() {
			if err := cborEncode(buf, v.Index(i)); err != nil {
				return err
			}
		}

	case reflect.Map:
		cborEncodeHead(buf, cborMap, uint64(v.Len()))
		for _, key := range v.MapKeys() {
			if err := cborEncode(buf, key); err != nil {
				return err
			}
			if err := cborEncode(buf, v.MapIndex(key)); err != nil {
				return err
			}
		}

	default:
		return fmt.Errorf("cbor: unsupported type %s", v.Type())
	}

	return nil
}
