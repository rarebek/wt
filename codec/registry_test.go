package codec

import "testing"

func TestRegistryMultipleCodecs(t *testing.T) {
	r := NewRegistry()
	r.Register(MsgPack{})
	r.Register(CBOR{})

	mp, err := r.Get("msgpack")
	if err != nil {
		t.Fatalf("get msgpack: %v", err)
	}
	if mp.Name() != "msgpack" {
		t.Error("expected msgpack")
	}

	cb, err := r.Get("cbor")
	if err != nil {
		t.Fatalf("get cbor: %v", err)
	}
	if cb.Name() != "cbor" {
		t.Error("expected cbor")
	}
}

func TestMsgPackName(t *testing.T) {
	c := MsgPack{}
	if c.Name() != "msgpack" {
		t.Errorf("expected 'msgpack', got %q", c.Name())
	}
}

func TestMsgPackMarshalInt(t *testing.T) {
	c := MsgPack{}
	data, err := c.Marshal(42)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if len(data) == 0 {
		t.Error("expected non-empty output")
	}
	// 42 in msgpack is single byte 0x2a
	if data[0] != 0x2a {
		t.Errorf("expected 0x2a, got 0x%02x", data[0])
	}
}

func TestMsgPackMarshalString(t *testing.T) {
	c := MsgPack{}
	data, err := c.Marshal("hi")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	// "hi" = fixstr(2) + "hi" = 0xa2 0x68 0x69
	if data[0] != 0xa2 {
		t.Errorf("expected 0xa2 (fixstr 2), got 0x%02x", data[0])
	}
}

func TestCBORMarshalBytes(t *testing.T) {
	c := CBOR{}
	data, err := c.Marshal([]byte{1, 2, 3})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	// bytes(3) = 0x43 + 0x01 0x02 0x03
	if data[0] != 0x43 {
		t.Errorf("expected 0x43 (bstr 3), got 0x%02x", data[0])
	}
}

func TestCBORMarshalMap(t *testing.T) {
	c := CBOR{}
	data, err := c.Marshal(map[string]int{"x": 1})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	// map(1) = 0xa1
	if data[0] != 0xa1 {
		t.Errorf("expected 0xa1 (map 1), got 0x%02x", data[0])
	}
}
