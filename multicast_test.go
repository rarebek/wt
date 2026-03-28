package wt

import "testing"

func TestMulticastFilter(t *testing.T) {
	// Test the filter function pattern works correctly
	sessions := []*Context{
		{id: "s1", store: map[string]any{"role": "admin"}},
		{id: "s2", store: map[string]any{"role": "user"}},
		{id: "s3", store: map[string]any{"role": "admin"}},
		{id: "s4", store: map[string]any{"role": "guest"}},
	}

	filter := func(c *Context) bool {
		role, _ := c.Get("role")
		return role == "admin"
	}

	var matched int
	for _, s := range sessions {
		if filter(s) {
			matched++
		}
	}

	if matched != 2 {
		t.Errorf("expected 2 admin matches, got %d", matched)
	}
}
