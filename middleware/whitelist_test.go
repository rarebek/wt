package middleware

import (
	"net"
	"testing"
)

func TestExtractIP(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"127.0.0.1:4433", "127.0.0.1"},
		{"192.168.1.100:8080", "192.168.1.100"},
		{"[::1]:443", "::1"},
		{"10.0.0.1", "10.0.0.1"},
	}

	for _, tt := range tests {
		got := extractIP(tt.input)
		if got == nil {
			t.Errorf("extractIP(%q) = nil, want %q", tt.input, tt.want)
			continue
		}
		if !got.Equal(net.ParseIP(tt.want)) {
			t.Errorf("extractIP(%q) = %s, want %s", tt.input, got, tt.want)
		}
	}
}
