package wt

import "testing"

func BenchmarkSessionCloseError(b *testing.B) {
	for b.Loop() {
		err := &SessionCloseError{Code: 401, Message: "unauthorized"}
		_ = err.Error()
	}
}

func BenchmarkIsSessionClosed(b *testing.B) {
	err := &SessionCloseError{Code: 0, Message: ""}
	b.ResetTimer()
	for b.Loop() {
		IsSessionClosed(err)
	}
}
