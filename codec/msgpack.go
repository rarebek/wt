package codec

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"reflect"
)

// MsgPack is a lightweight MessagePack codec.
// It supports basic types: string, int, float, bool, nil, []byte, slices, maps, and structs.
// For production use with complex types, consider importing a full msgpack library.
type MsgPack struct{}

func (MsgPack) Name() string { return "msgpack" }

func (MsgPack) Marshal(v any) ([]byte, error) {
	var buf bytes.Buffer
	if err := msgpackEncode(&buf, reflect.ValueOf(v)); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (MsgPack) Unmarshal(data []byte, v any) error {
	// For simplicity, fall back to JSON for complex unmarshal.
	// A production codec would use a full msgpack library.
	return fmt.Errorf("msgpack: Unmarshal not implemented — use a full msgpack library like github.com/vmihailenco/msgpack/v5")
}

func msgpackEncode(buf *bytes.Buffer, v reflect.Value) error {
	if !v.IsValid() {
		buf.WriteByte(0xc0) // nil
		return nil
	}

	switch v.Kind() {
	case reflect.Ptr, reflect.Interface:
		if v.IsNil() {
			buf.WriteByte(0xc0)
			return nil
		}
		return msgpackEncode(buf, v.Elem())

	case reflect.Bool:
		if v.Bool() {
			buf.WriteByte(0xc3)
		} else {
			buf.WriteByte(0xc2)
		}

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n := v.Int()
		if n >= 0 && n <= 127 {
			buf.WriteByte(byte(n))
		} else if n >= -32 && n < 0 {
			buf.WriteByte(byte(0xe0 | (n + 32)))
		} else if n >= math.MinInt8 && n <= math.MaxInt8 {
			buf.WriteByte(0xd0)
			buf.WriteByte(byte(int8(n)))
		} else if n >= math.MinInt16 && n <= math.MaxInt16 {
			buf.WriteByte(0xd1)
			binary.Write(buf, binary.BigEndian, int16(n))
		} else if n >= math.MinInt32 && n <= math.MaxInt32 {
			buf.WriteByte(0xd2)
			binary.Write(buf, binary.BigEndian, int32(n))
		} else {
			buf.WriteByte(0xd3)
			binary.Write(buf, binary.BigEndian, n)
		}

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		n := v.Uint()
		if n <= 127 {
			buf.WriteByte(byte(n))
		} else if n <= math.MaxUint8 {
			buf.WriteByte(0xcc)
			buf.WriteByte(byte(n))
		} else if n <= math.MaxUint16 {
			buf.WriteByte(0xcd)
			binary.Write(buf, binary.BigEndian, uint16(n))
		} else if n <= math.MaxUint32 {
			buf.WriteByte(0xce)
			binary.Write(buf, binary.BigEndian, uint32(n))
		} else {
			buf.WriteByte(0xcf)
			binary.Write(buf, binary.BigEndian, n)
		}

	case reflect.Float32:
		buf.WriteByte(0xca)
		binary.Write(buf, binary.BigEndian, float32(v.Float()))

	case reflect.Float64:
		buf.WriteByte(0xcb)
		binary.Write(buf, binary.BigEndian, v.Float())

	case reflect.String:
		s := v.String()
		l := len(s)
		if l <= 31 {
			buf.WriteByte(byte(0xa0 | l))
		} else if l <= math.MaxUint8 {
			buf.WriteByte(0xd9)
			buf.WriteByte(byte(l))
		} else if l <= math.MaxUint16 {
			buf.WriteByte(0xda)
			binary.Write(buf, binary.BigEndian, uint16(l))
		} else {
			buf.WriteByte(0xdb)
			binary.Write(buf, binary.BigEndian, uint32(l))
		}
		buf.WriteString(s)

	case reflect.Slice:
		if v.Type().Elem().Kind() == reflect.Uint8 {
			// []byte -> bin format
			b := v.Bytes()
			l := len(b)
			if l <= math.MaxUint8 {
				buf.WriteByte(0xc4)
				buf.WriteByte(byte(l))
			} else if l <= math.MaxUint16 {
				buf.WriteByte(0xc5)
				binary.Write(buf, binary.BigEndian, uint16(l))
			} else {
				buf.WriteByte(0xc6)
				binary.Write(buf, binary.BigEndian, uint32(l))
			}
			buf.Write(b)
			return nil
		}
		// Regular array
		l := v.Len()
		if l <= 15 {
			buf.WriteByte(byte(0x90 | l))
		} else if l <= math.MaxUint16 {
			buf.WriteByte(0xdc)
			binary.Write(buf, binary.BigEndian, uint16(l))
		} else {
			buf.WriteByte(0xdd)
			binary.Write(buf, binary.BigEndian, uint32(l))
		}
		for i := range l {
			if err := msgpackEncode(buf, v.Index(i)); err != nil {
				return err
			}
		}

	case reflect.Map:
		l := v.Len()
		if l <= 15 {
			buf.WriteByte(byte(0x80 | l))
		} else if l <= math.MaxUint16 {
			buf.WriteByte(0xde)
			binary.Write(buf, binary.BigEndian, uint16(l))
		} else {
			buf.WriteByte(0xdf)
			binary.Write(buf, binary.BigEndian, uint32(l))
		}
		for _, key := range v.MapKeys() {
			if err := msgpackEncode(buf, key); err != nil {
				return err
			}
			if err := msgpackEncode(buf, v.MapIndex(key)); err != nil {
				return err
			}
		}

	default:
		return fmt.Errorf("msgpack: unsupported type %s", v.Type())
	}

	return nil
}
