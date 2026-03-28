package wt

import "sync"

// Tags provides a thread-safe tagging system for sessions.
// Tags can be used to categorize sessions for targeted operations
// like "send to all sessions tagged 'premium'" or "disconnect all 'guest' sessions".
type Tags struct {
	mu   sync.RWMutex
	tags map[string]map[string]bool // tag -> set of session IDs
}

// NewTags creates a new tag registry.
func NewTags() *Tags {
	return &Tags{
		tags: make(map[string]map[string]bool),
	}
}

// Tag adds a tag to a session.
func (t *Tags) Tag(sessionID, tag string) {
	t.mu.Lock()
	if t.tags[tag] == nil {
		t.tags[tag] = make(map[string]bool)
	}
	t.tags[tag][sessionID] = true
	t.mu.Unlock()
}

// Untag removes a tag from a session.
func (t *Tags) Untag(sessionID, tag string) {
	t.mu.Lock()
	if m := t.tags[tag]; m != nil {
		delete(m, sessionID)
		if len(m) == 0 {
			delete(t.tags, tag)
		}
	}
	t.mu.Unlock()
}

// UntagAll removes all tags for a session.
func (t *Tags) UntagAll(sessionID string) {
	t.mu.Lock()
	for tag, m := range t.tags {
		delete(m, sessionID)
		if len(m) == 0 {
			delete(t.tags, tag)
		}
	}
	t.mu.Unlock()
}

// HasTag checks if a session has a specific tag.
func (t *Tags) HasTag(sessionID, tag string) bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.tags[tag][sessionID]
}

// SessionsWithTag returns all session IDs with the given tag.
func (t *Tags) SessionsWithTag(tag string) []string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	m := t.tags[tag]
	if m == nil {
		return nil
	}
	ids := make([]string, 0, len(m))
	for id := range m {
		ids = append(ids, id)
	}
	return ids
}

// TagsForSession returns all tags for a session.
func (t *Tags) TagsForSession(sessionID string) []string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	var tags []string
	for tag, m := range t.tags {
		if m[sessionID] {
			tags = append(tags, tag)
		}
	}
	return tags
}

// Count returns the number of sessions with the given tag.
func (t *Tags) Count(tag string) int {
	t.mu.RLock()
	n := len(t.tags[tag])
	t.mu.RUnlock()
	return n
}

// AllTags returns all known tags.
func (t *Tags) AllTags() []string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	tags := make([]string, 0, len(t.tags))
	for tag := range t.tags {
		tags = append(tags, tag)
	}
	return tags
}
