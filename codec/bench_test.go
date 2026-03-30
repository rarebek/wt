package codec

import "testing"

func BenchmarkJSONMarshal(b *testing.B) {
	c := JSON{}
	type msg struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
		Tags  []string `json:"tags"`
	}
	v := msg{Name: "test", Value: 42, Tags: []string{"a", "b", "c"}}

	b.ResetTimer()
	for b.Loop() {
		c.Marshal(v)
	}
}

func BenchmarkJSONUnmarshal(b *testing.B) {
	c := JSON{}
	data := []byte(`{"name":"test","value":42,"tags":["a","b","c"]}`)
	type msg struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
		Tags  []string `json:"tags"`
	}

	b.ResetTimer()
	for b.Loop() {
		var v msg
		c.Unmarshal(data, &v)
	}
}

func BenchmarkCBORMarshalInt(b *testing.B) {
	c := CBOR{}
	b.ResetTimer()
	for b.Loop() {
		c.Marshal(42)
	}
}

func BenchmarkCBORMarshalString(b *testing.B) {
	c := CBOR{}
	b.ResetTimer()
	for b.Loop() {
		c.Marshal("hello world benchmark test")
	}
}

func BenchmarkMsgPackMarshalInt(b *testing.B) {
	c := MsgPack{}
	b.ResetTimer()
	for b.Loop() {
		c.Marshal(42)
	}
}

func BenchmarkRegistryGet(b *testing.B) {
	r := NewRegistry()
	b.ResetTimer()
	for b.Loop() {
		r.Get("json")
	}
}
