package wt

import "iter"

// All returns an iterator over all room names.
func (rm *RoomManager) All() iter.Seq2[string, *Room] {
	return func(yield func(string, *Room) bool) {
		rm.mu.RLock()
		defer rm.mu.RUnlock()
		for name, room := range rm.rooms {
			if !yield(name, room) {
				return
			}
		}
	}
}

// MembersIter returns an iterator over room members.
func (r *Room) MembersIter() iter.Seq[*Context] {
	return func(yield func(*Context) bool) {
		r.mu.RLock()
		defer r.mu.RUnlock()
		for _, ctx := range r.members {
			if !yield(ctx) {
				return
			}
		}
	}
}
