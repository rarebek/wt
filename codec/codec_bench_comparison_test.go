package codec

import "testing"

// Side-by-side codec comparison benchmarks

func BenchmarkJSON_Int(b *testing.B) {
	c := JSON{}
	for b.Loop() { c.Marshal(42) }
}

func BenchmarkCBOR_Int(b *testing.B) {
	c := CBOR{}
	for b.Loop() { c.Marshal(42) }
}

func BenchmarkMsgPack_Int(b *testing.B) {
	c := MsgPack{}
	for b.Loop() { c.Marshal(42) }
}

func BenchmarkBinary_Int(b *testing.B) {
	c := BigEndian()
	for b.Loop() { c.Marshal(int32(42)) }
}

func BenchmarkRaw_Bytes(b *testing.B) {
	c := Raw{}
	data := []byte("hello")
	for b.Loop() { c.Marshal(data) }
}
