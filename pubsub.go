package wt

import "sync"

// PubSub provides topic-based publish/subscribe messaging.
// Sessions subscribe to topics and receive datagrams published to those topics.
// More granular than rooms — a session can subscribe to any combination of topics.
type PubSub struct {
	mu          sync.RWMutex
	subscribers map[string]map[string]*Context // topic -> sessionID -> context
}

// NewPubSub creates a new pub/sub hub.
func NewPubSub() *PubSub {
	return &PubSub{
		subscribers: make(map[string]map[string]*Context),
	}
}

// Subscribe adds a session to a topic.
func (ps *PubSub) Subscribe(topic string, c *Context) {
	ps.mu.Lock()
	if ps.subscribers[topic] == nil {
		ps.subscribers[topic] = make(map[string]*Context)
	}
	ps.subscribers[topic][c.ID()] = c
	ps.mu.Unlock()
}

// Unsubscribe removes a session from a topic.
func (ps *PubSub) Unsubscribe(topic string, c *Context) {
	ps.mu.Lock()
	if m := ps.subscribers[topic]; m != nil {
		delete(m, c.ID())
		if len(m) == 0 {
			delete(ps.subscribers, topic)
		}
	}
	ps.mu.Unlock()
}

// UnsubscribeAll removes a session from all topics.
func (ps *PubSub) UnsubscribeAll(c *Context) {
	ps.mu.Lock()
	for topic, subs := range ps.subscribers {
		delete(subs, c.ID())
		if len(subs) == 0 {
			delete(ps.subscribers, topic)
		}
	}
	ps.mu.Unlock()
}

// Publish sends a datagram to all subscribers of a topic.
func (ps *PubSub) Publish(topic string, data []byte) {
	ps.mu.RLock()
	subs := ps.subscribers[topic]
	ps.mu.RUnlock()

	for _, c := range subs {
		_ = c.SendDatagram(data)
	}
}

// PublishExcept sends to all subscribers except the given session.
func (ps *PubSub) PublishExcept(topic string, data []byte, excludeID string) {
	ps.mu.RLock()
	subs := ps.subscribers[topic]
	ps.mu.RUnlock()

	for id, c := range subs {
		if id != excludeID {
			_ = c.SendDatagram(data)
		}
	}
}

// SubscriberCount returns the number of subscribers for a topic.
func (ps *PubSub) SubscriberCount(topic string) int {
	ps.mu.RLock()
	n := len(ps.subscribers[topic])
	ps.mu.RUnlock()
	return n
}

// Topics returns all active topics.
func (ps *PubSub) Topics() []string {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	topics := make([]string, 0, len(ps.subscribers))
	for t := range ps.subscribers {
		topics = append(topics, t)
	}
	return topics
}

// TopicsForSession returns all topics a session is subscribed to.
func (ps *PubSub) TopicsForSession(sessionID string) []string {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	var topics []string
	for topic, subs := range ps.subscribers {
		if _, ok := subs[sessionID]; ok {
			topics = append(topics, topic)
		}
	}
	return topics
}
