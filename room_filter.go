package wt

// FilterMembers returns room members matching a predicate.
func (r *Room) FilterMembers(fn func(*Context) bool) []*Context {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []*Context
	for _, ctx := range r.members {
		if fn(ctx) {
			result = append(result, ctx)
		}
	}
	return result
}

// Has checks if a session is in the room.
func (r *Room) Has(sessionID string) bool {
	r.mu.RLock()
	_, ok := r.members[sessionID]
	r.mu.RUnlock()
	return ok
}
