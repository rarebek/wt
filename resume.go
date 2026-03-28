package wt

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

// ResumeToken is an opaque token that clients can use to restore
// session state after a reconnect.
type ResumeToken string

// ResumeStore persists session state for reconnection.
// When a client disconnects and reconnects with a resume token,
// the framework can restore their context store values (user info, etc.)
// without requiring re-authentication.
type ResumeStore struct {
	mu     sync.Mutex
	states map[ResumeToken]*resumeState
	ttl    time.Duration
}

type resumeState struct {
	store   map[string]any
	expires time.Time
}

// NewResumeStore creates a store with the given TTL for saved states.
// States are automatically expired after TTL.
func NewResumeStore(ttl time.Duration) *ResumeStore {
	rs := &ResumeStore{
		states: make(map[ResumeToken]*resumeState),
		ttl:    ttl,
	}
	go rs.cleanup()
	return rs
}

// Save stores the session's context values and returns a resume token.
// The client should store this token and present it on reconnection.
func (rs *ResumeStore) Save(c *Context) ResumeToken {
	token := generateToken()

	c.mu.RLock()
	storeCopy := make(map[string]any, len(c.store))
	for k, v := range c.store {
		storeCopy[k] = v
	}
	c.mu.RUnlock()

	rs.mu.Lock()
	rs.states[token] = &resumeState{
		store:   storeCopy,
		expires: time.Now().Add(rs.ttl),
	}
	rs.mu.Unlock()

	return token
}

// Restore applies saved state to a new session context.
// Returns true if the token was valid and state was restored.
func (rs *ResumeStore) Restore(c *Context, token ResumeToken) bool {
	rs.mu.Lock()
	state, ok := rs.states[token]
	if ok {
		delete(rs.states, token) // single-use token
	}
	rs.mu.Unlock()

	if !ok || time.Now().After(state.expires) {
		return false
	}

	for k, v := range state.store {
		c.Set(k, v)
	}
	return true
}

// Count returns the number of stored resume states.
func (rs *ResumeStore) Count() int {
	rs.mu.Lock()
	n := len(rs.states)
	rs.mu.Unlock()
	return n
}

func (rs *ResumeStore) cleanup() {
	ticker := time.NewTicker(rs.ttl / 2)
	defer ticker.Stop()
	for range ticker.C {
		rs.mu.Lock()
		now := time.Now()
		for token, state := range rs.states {
			if now.After(state.expires) {
				delete(rs.states, token)
			}
		}
		rs.mu.Unlock()
	}
}

func generateToken() ResumeToken {
	b := make([]byte, 32)
	rand.Read(b)
	return ResumeToken(hex.EncodeToString(b))
}
