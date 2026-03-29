package wt

import "testing"

func TestJoinPathBasic(t *testing.T) {
	tests := []struct {
		in   []string
		want string
	}{
		{[]string{"a", "b"}, "/a/b"},
		{[]string{"/api", "/v1"}, "/api/v1"},
		{[]string{"/"}, ""},
	}

	for _, tt := range tests {
		got := JoinPath(tt.in...)
		if got != tt.want {
			t.Errorf("JoinPath(%v) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestHashDeterministic(t *testing.T) {
	h1 := Hash([]byte("test"))
	h2 := Hash([]byte("test"))
	if h1 != h2 {
		t.Error("Hash should be deterministic")
	}
	if len(h1) != 64 {
		t.Errorf("expected 64 char hash, got %d", len(h1))
	}
}

func TestHashDifferentInputs(t *testing.T) {
	h1 := Hash([]byte("hello"))
	h2 := Hash([]byte("world"))
	if h1 == h2 {
		t.Error("different inputs should produce different hashes")
	}
}
