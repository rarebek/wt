package wt

// Filter returns sessions matching a predicate.
func (ss *SessionStore) Filter(fn func(*Context) bool) []*Context {
	ss.mu.RLock()
	defer ss.mu.RUnlock()
	var result []*Context
	for _, ctx := range ss.sessions {
		if fn(ctx) {
			result = append(result, ctx)
		}
	}
	return result
}

// CountWhere returns the number of sessions matching a predicate.
func (ss *SessionStore) CountWhere(fn func(*Context) bool) int {
	ss.mu.RLock()
	defer ss.mu.RUnlock()
	n := 0
	for _, ctx := range ss.sessions {
		if fn(ctx) {
			n++
		}
	}
	return n
}
