package wt

import "iter"

// TopicsIter returns an iterator over pub/sub topics and subscriber counts.
func (ps *PubSub) TopicsIter() iter.Seq2[string, int] {
	return func(yield func(string, int) bool) {
		ps.mu.RLock()
		defer ps.mu.RUnlock()
		for topic, subs := range ps.subscribers {
			if !yield(topic, len(subs)) {
				return
			}
		}
	}
}

// TagsIter returns an iterator over all tags and their session counts.
func (t *Tags) TagsIter() iter.Seq2[string, int] {
	return func(yield func(string, int) bool) {
		t.mu.RLock()
		defer t.mu.RUnlock()
		for tag, sessions := range t.tags {
			if !yield(tag, len(sessions)) {
				return
			}
		}
	}
}
