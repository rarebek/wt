package wt

import (
	"strings"
	"sync"
)

// Route represents a registered path pattern with its handler and middleware.
type Route struct {
	Pattern    string
	Handler    HandlerFunc
	Middleware []MiddlewareFunc
	params     []string // parameter names extracted from pattern
}

// Router handles path-based routing for WebTransport sessions.
type Router struct {
	mu     sync.RWMutex
	routes []*Route
}

// NewRouter creates a new Router.
func NewRouter() *Router {
	return &Router{}
}

// Add registers a handler for the given path pattern.
// Patterns support parameters like "/chat/{room}" and "/game/{id}/input".
func (r *Router) Add(pattern string, handler HandlerFunc, mw ...MiddlewareFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()

	route := &Route{
		Pattern:    pattern,
		Handler:    handler,
		Middleware: mw,
		params:     extractParamNames(pattern),
	}
	r.routes = append(r.routes, route)
}

// Routes returns all registered routes.
func (r *Router) Routes() []*Route {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.routes
}

// Match finds the route matching the given path and extracts parameters.
func (r *Router) Match(path string) (*Route, map[string]string) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, route := range r.routes {
		if params, ok := matchPattern(route.Pattern, path); ok {
			return route, params
		}
	}
	return nil, nil
}

// ExtractParams extracts path parameters from a URL path given a pattern.
func (r *Router) ExtractParams(pattern, path string) map[string]string {
	params, _ := matchPattern(pattern, path)
	return params
}

// extractParamNames returns parameter names from a pattern.
// "/chat/{room}/user/{id}" -> ["room", "id"]
func extractParamNames(pattern string) []string {
	var names []string
	parts := strings.Split(pattern, "/")
	for _, part := range parts {
		if len(part) > 2 && part[0] == '{' && part[len(part)-1] == '}' {
			names = append(names, part[1:len(part)-1])
		}
	}
	return names
}

// matchPattern checks if a path matches a pattern and extracts parameters.
// Supports {param} for single-segment parameters and {param...} for catch-all wildcards.
// Optimized to avoid strings.Split allocations by scanning in place.
func matchPattern(pattern, path string) (map[string]string, bool) {
	// Trim leading/trailing slashes
	pattern = strings.Trim(pattern, "/")
	path = strings.Trim(path, "/")

	params := make(map[string]string, 2)
	pi, pj := 0, 0

	for pi < len(pattern) && pj < len(path) {
		pe := strings.IndexByte(pattern[pi:], '/')
		if pe == -1 {
			pe = len(pattern)
		} else {
			pe += pi
		}
		pSeg := pattern[pi:pe]

		// Check for catch-all wildcard: {name...}
		if len(pSeg) > 5 && pSeg[0] == '{' && pSeg[len(pSeg)-4:] == "...}" {
			name := pSeg[1 : len(pSeg)-4]
			params[name] = path[pj:] // capture rest of path
			return params, true
		}

		je := strings.IndexByte(path[pj:], '/')
		if je == -1 {
			je = len(path)
		} else {
			je += pj
		}
		pathSeg := path[pj:je]

		if len(pSeg) > 2 && pSeg[0] == '{' && pSeg[len(pSeg)-1] == '}' {
			params[pSeg[1:len(pSeg)-1]] = pathSeg
		} else if pSeg != pathSeg {
			return nil, false
		}

		pi = pe + 1
		pj = je + 1
	}

	// Both should be exhausted
	if pi <= len(pattern) && pj > len(path) {
		return params, pi > len(pattern)
	}
	if pi > len(pattern) && pj <= len(path) {
		return nil, false
	}

	return params, true
}

func countSegments(s string) int {
	if s == "" {
		return 1
	}
	n := 1
	for i := range len(s) {
		if s[i] == '/' {
			n++
		}
	}
	return n
}
