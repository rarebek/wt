package wt

import (
	"fmt"
	"testing"
)

func BenchmarkPubSubSubscribe(b *testing.B) {
	ps := NewPubSub()
	contexts := make([]*Context, b.N)
	for i := range contexts {
		contexts[i] = &Context{id: fmt.Sprintf("s-%d", i), store: make(map[string]any)}
	}
	b.ResetTimer()
	for i := range b.N {
		ps.Subscribe("topic", contexts[i])
	}
}

func BenchmarkPubSubTopics(b *testing.B) {
	ps := NewPubSub()
	c := &Context{id: "s1", store: make(map[string]any)}
	for i := range 100 {
		ps.Subscribe(fmt.Sprintf("topic-%d", i), c)
	}
	b.ResetTimer()
	for b.Loop() {
		ps.Topics()
	}
}

func BenchmarkTagsTag(b *testing.B) {
	tags := NewTags()
	b.ResetTimer()
	for i := range b.N {
		tags.Tag(fmt.Sprintf("s-%d", i), "vip")
	}
}

func BenchmarkTagsLookup(b *testing.B) {
	tags := NewTags()
	for i := range 1000 {
		tags.Tag(fmt.Sprintf("s-%d", i), "vip")
	}
	b.ResetTimer()
	for b.Loop() {
		tags.HasTag("s-500", "vip")
	}
}
