package wt

import "testing"

type benchMsg struct {
	Name  string
	Value int
	Tags  []string
}

func (m benchMsg) Validate() error {
	return RequiredFields(m, "Name", "Value")
}

func BenchmarkValidateMessage(b *testing.B) {
	msg := benchMsg{Name: "test", Value: 42, Tags: []string{"a"}}
	b.ResetTimer()
	for b.Loop() {
		ValidateMessage(msg)
	}
}

func BenchmarkRequiredFields(b *testing.B) {
	msg := benchMsg{Name: "test", Value: 42}
	b.ResetTimer()
	for b.Loop() {
		RequiredFields(msg, "Name", "Value")
	}
}
