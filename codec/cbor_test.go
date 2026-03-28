package codec

import "testing"

func TestCBORName(t *testing.T) {
	c := CBOR{}
	if c.Name() != "cbor" {
		t.Errorf("expected 'cbor', got %q", c.Name())
	}
}

func TestCBORMarshalString(t *testing.T) {
	c := CBOR{}

	data, err := c.Marshal("hello")
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	// "hello" in CBOR: 0x65 (text string, length 5) + "hello"
	if len(data) < 1 {
		t.Fatal("empty output")
	}
	if data[0] != 0x65 { // text(5)
		t.Errorf("expected CBOR text header 0x65, got 0x%02x", data[0])
	}
	if string(data[1:]) != "hello" {
		t.Errorf("expected 'hello', got %q", data[1:])
	}
}

func TestCBORMarshalInt(t *testing.T) {
	c := CBOR{}

	// Small positive integer (0-23)
	data, err := c.Marshal(10)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	if data[0] != 0x0a { // uint(10)
		t.Errorf("expected 0x0a, got 0x%02x", data[0])
	}

	// Zero
	data, err = c.Marshal(0)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	if data[0] != 0x00 {
		t.Errorf("expected 0x00, got 0x%02x", data[0])
	}
}

func TestCBORMarshalBool(t *testing.T) {
	c := CBOR{}

	data, err := c.Marshal(true)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	if data[0] != 0xf5 { // true
		t.Errorf("expected 0xf5 (true), got 0x%02x", data[0])
	}

	data, err = c.Marshal(false)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	if data[0] != 0xf4 { // false
		t.Errorf("expected 0xf4 (false), got 0x%02x", data[0])
	}
}

func TestCBORMarshalNil(t *testing.T) {
	c := CBOR{}

	data, err := c.Marshal(nil)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	if data[0] != 0xf6 { // null
		t.Errorf("expected 0xf6 (null), got 0x%02x", data[0])
	}
}

func TestCBORMarshalArray(t *testing.T) {
	c := CBOR{}

	data, err := c.Marshal([]int{1, 2, 3})
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	if data[0] != 0x83 { // array(3)
		t.Errorf("expected 0x83 (array of 3), got 0x%02x", data[0])
	}
}
