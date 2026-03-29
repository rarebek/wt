package codec

import "testing"

func TestBinaryCodec(t *testing.T) {
	c := BigEndian()
	if c.Name() != "binary" {
		t.Errorf("expected 'binary', got %q", c.Name())
	}

	type Point struct {
		X, Y int32
	}

	original := Point{X: 100, Y: -50}
	data, err := c.Marshal(original)
	if err != nil {
		t.Fatal(err)
	}

	var decoded Point
	err = c.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatal(err)
	}

	if decoded != original {
		t.Errorf("expected %+v, got %+v", original, decoded)
	}
}

func TestBinaryLittleEndian(t *testing.T) {
	c := LittleEndian()

	type Vec2 struct {
		X, Y float32
	}

	original := Vec2{X: 1.5, Y: -2.5}
	data, _ := c.Marshal(original)

	var decoded Vec2
	c.Unmarshal(data, &decoded)

	if decoded != original {
		t.Errorf("expected %+v, got %+v", original, decoded)
	}
}

func BenchmarkBinaryMarshal(b *testing.B) {
	c := BigEndian()
	type Packet struct {
		Seq  uint32
		X, Y float32
		HP   int16
	}
	v := Packet{Seq: 42, X: 1.5, Y: 2.5, HP: 100}
	b.ResetTimer()
	for b.Loop() {
		c.Marshal(v)
	}
}

func BenchmarkBinaryUnmarshal(b *testing.B) {
	c := BigEndian()
	type Packet struct {
		Seq  uint32
		X, Y float32
		HP   int16
	}
	data, _ := c.Marshal(Packet{42, 1.5, 2.5, 100})
	b.ResetTimer()
	for b.Loop() {
		var v Packet
		c.Unmarshal(data, &v)
	}
}
