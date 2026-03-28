package wt

import (
	"fmt"
	"testing"
)

func TestServerInfo(t *testing.T) {
	info := ServerInfo()
	if info["framework"] != "wt" {
		t.Errorf("expected framework 'wt', got %q", info["framework"])
	}
	if info["version"] != Version {
		t.Errorf("expected version %q, got %q", Version, info["version"])
	}
}

func TestHash(t *testing.T) {
	h := Hash([]byte("hello"))
	if len(h) != 64 { // SHA-256 = 32 bytes = 64 hex chars
		t.Errorf("expected 64 char hash, got %d", len(h))
	}
}

func TestJoinPath(t *testing.T) {
	tests := []struct {
		input []string
		want  string
	}{
		{[]string{"a", "b", "c"}, "/a/b/c"},
		{[]string{"/api", "/v1", "/users"}, "/api/v1/users"},
		{[]string{"", "a", ""}, "/a"},
		{[]string{"/", "a"}, "/a"},
	}

	for _, tt := range tests {
		got := JoinPath(tt.input...)
		if got != tt.want {
			t.Errorf("JoinPath(%v) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestMust(t *testing.T) {
	val := Must(42, nil)
	if val != 42 {
		t.Errorf("expected 42, got %d", val)
	}
}

func TestMustPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic")
		}
	}()
	Must(0, fmt.Errorf("boom"))
}
