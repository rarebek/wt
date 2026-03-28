package wt

import (
	"testing"
)

func TestExtractParamNames(t *testing.T) {
	tests := []struct {
		pattern string
		want    []string
	}{
		{"/chat/{room}", []string{"room"}},
		{"/game/{id}/input", []string{"id"}},
		{"/user/{org}/{team}/{id}", []string{"org", "team", "id"}},
		{"/static/path", nil},
		{"/{a}/{b}", []string{"a", "b"}},
	}

	for _, tt := range tests {
		got := extractParamNames(tt.pattern)
		if len(got) != len(tt.want) {
			t.Errorf("extractParamNames(%q) = %v, want %v", tt.pattern, got, tt.want)
			continue
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("extractParamNames(%q)[%d] = %q, want %q", tt.pattern, i, got[i], tt.want[i])
			}
		}
	}
}

func TestMatchPattern(t *testing.T) {
	tests := []struct {
		pattern string
		path    string
		match   bool
		params  map[string]string
	}{
		{"/chat/{room}", "/chat/general", true, map[string]string{"room": "general"}},
		{"/chat/{room}", "/chat/", false, nil},
		{"/chat/{room}", "/chat/general/extra", false, nil},
		{"/game/{id}/input", "/game/123/input", true, map[string]string{"id": "123"}},
		{"/game/{id}/input", "/game/123/output", false, nil},
		{"/static", "/static", true, map[string]string{}},
		{"/static", "/other", false, nil},
		{"/{a}/{b}", "/hello/world", true, map[string]string{"a": "hello", "b": "world"}},
		{"/", "/", true, map[string]string{}},
		// Catch-all wildcard
		{"/static/{path...}", "/static/css/main.css", true, map[string]string{"path": "css/main.css"}},
		{"/files/{rest...}", "/files/a/b/c/d", true, map[string]string{"rest": "a/b/c/d"}},
		{"/api/{version}/{path...}", "/api/v1/users/list", true, map[string]string{"version": "v1", "path": "users/list"}},
	}

	for _, tt := range tests {
		params, ok := matchPattern(tt.pattern, tt.path)
		if ok != tt.match {
			t.Errorf("matchPattern(%q, %q) match = %v, want %v", tt.pattern, tt.path, ok, tt.match)
			continue
		}
		if !ok {
			continue
		}
		if len(params) != len(tt.params) {
			t.Errorf("matchPattern(%q, %q) params = %v, want %v", tt.pattern, tt.path, params, tt.params)
			continue
		}
		for k, v := range tt.params {
			if params[k] != v {
				t.Errorf("matchPattern(%q, %q) param[%q] = %q, want %q", tt.pattern, tt.path, k, params[k], v)
			}
		}
	}
}

func TestRouterAddAndMatch(t *testing.T) {
	r := NewRouter()

	r.Add("/chat/{room}", func(c *Context) {})
	r.Add("/game/{id}/state", func(c *Context) {})

	route, params := r.Match("/chat/lobby")
	if route == nil {
		t.Fatal("expected to match /chat/{room}")
	}
	if params["room"] != "lobby" {
		t.Errorf("expected room=lobby, got %q", params["room"])
	}

	route, params = r.Match("/game/42/state")
	if route == nil {
		t.Fatal("expected to match /game/{id}/state")
	}
	if params["id"] != "42" {
		t.Errorf("expected id=42, got %q", params["id"])
	}

	route, _ = r.Match("/unknown/path")
	if route != nil {
		t.Error("expected no match for /unknown/path")
	}
}

func TestRouterRoutes(t *testing.T) {
	r := NewRouter()
	r.Add("/a", func(c *Context) {})
	r.Add("/b", func(c *Context) {})
	r.Add("/c", func(c *Context) {})

	routes := r.Routes()
	if len(routes) != 3 {
		t.Errorf("expected 3 routes, got %d", len(routes))
	}
}
