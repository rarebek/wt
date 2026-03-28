package wt

import (
	"testing"
)

func FuzzMatchPattern(f *testing.F) {
	// Seed corpus
	f.Add("/chat/{room}", "/chat/general")
	f.Add("/game/{id}/input", "/game/123/input")
	f.Add("/{a}/{b}/{c}", "/x/y/z")
	f.Add("/static", "/static")
	f.Add("/", "/")
	f.Add("/a/b/c", "/a/b/c")
	f.Add("/a/{b}", "/a/")
	f.Add("/{x}", "/hello")
	f.Add("/chat/{room}", "/other/path")

	f.Fuzz(func(t *testing.T, pattern, path string) {
		// Should never panic
		params, ok := matchPattern(pattern, path)
		if ok {
			// If matched, params should be non-nil
			if params == nil {
				t.Error("matched but params is nil")
			}
		}
	})
}

func FuzzExtractParamNames(f *testing.F) {
	f.Add("/chat/{room}")
	f.Add("/{a}/{b}/{c}")
	f.Add("/static/path")
	f.Add("")
	f.Add("/{}")
	f.Add("/{{nested}}")
	f.Add("/{valid}/text/{also}")

	f.Fuzz(func(t *testing.T, pattern string) {
		// Should never panic
		names := extractParamNames(pattern)
		_ = names
	})
}

func FuzzCountSegments(f *testing.F) {
	f.Add("")
	f.Add("/")
	f.Add("/a/b/c")
	f.Add("///")

	f.Fuzz(func(t *testing.T, s string) {
		n := countSegments(s)
		if n < 1 {
			t.Errorf("countSegments(%q) = %d, want >= 1", s, n)
		}
	})
}
