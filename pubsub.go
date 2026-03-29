package wt

import (
	"iter"
	"sync"

	"github.com/rarebek/wt/codec"
)

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

// PersistentPubSub extends PubSub with per-topic message history.
// When a client subscribes, they can replay messages they missed.
type PersistentPubSub struct {
	*PubSub
	mu      sync.RWMutex
	history map[string]*RingBuffer[[]byte] // topic -> message history
	maxHist int
}

// NewPersistentPubSub creates a pub/sub with per-topic history.
func NewPersistentPubSub(historyPerTopic int) *PersistentPubSub {
	return &PersistentPubSub{
		PubSub:  NewPubSub(),
		history: make(map[string]*RingBuffer[[]byte]),
		maxHist: historyPerTopic,
	}
}

// PublishPersistent publishes to subscribers AND stores in history.
func (pps *PersistentPubSub) PublishPersistent(topic string, data []byte) {
	pps.mu.Lock()
	if pps.history[topic] == nil {
		pps.history[topic] = NewRingBuffer[[]byte](pps.maxHist)
	}
	pps.history[topic].Push(data)
	pps.mu.Unlock()

	pps.PubSub.Publish(topic, data)
}

// Replay sends all stored history for a topic to the given session.
func (pps *PersistentPubSub) Replay(topic string, c *Context) {
	pps.mu.RLock()
	rb, ok := pps.history[topic]
	pps.mu.RUnlock()

	if !ok {
		return
	}

	for _, msg := range rb.Items() {
		_ = c.SendDatagram(msg)
	}
}

// HistoryLen returns the number of stored messages for a topic.
func (pps *PersistentPubSub) HistoryLen(topic string) int {
	pps.mu.RLock()
	defer pps.mu.RUnlock()
	rb, ok := pps.history[topic]
	if !ok {
		return 0
	}
	return rb.Len()
}

// ClearHistory removes all stored messages for a topic.
func (pps *PersistentPubSub) ClearHistory(topic string) {
	pps.mu.Lock()
	delete(pps.history, topic)
	pps.mu.Unlock()
}

// TypedPubSub provides type-safe publish/subscribe with automatic encoding.
type TypedPubSub[T any] struct {
	ps    *PubSub
	codec codec.Codec
}

// NewTypedPubSub wraps a PubSub with typed messages.
func NewTypedPubSub[T any](ps *PubSub, c codec.Codec) *TypedPubSub[T] {
	return &TypedPubSub[T]{ps: ps, codec: c}
}

// Publish encodes and publishes a message to all subscribers of a topic.
func (tp *TypedPubSub[T]) Publish(topic string, msg T) error {
	data, err := tp.codec.Marshal(msg)
	if err != nil {
		return err
	}
	tp.ps.Publish(topic, data)
	return nil
}

// PublishExcept encodes and publishes to all except one session.
func (tp *TypedPubSub[T]) PublishExcept(topic string, msg T, excludeID string) error {
	data, err := tp.codec.Marshal(msg)
	if err != nil {
		return err
	}
	tp.ps.PublishExcept(topic, data, excludeID)
	return nil
}

// Subscribe delegates to the underlying PubSub.
func (tp *TypedPubSub[T]) Subscribe(topic string, c *Context) {
	tp.ps.Subscribe(topic, c)
}

// Unsubscribe delegates to the underlying PubSub.
func (tp *TypedPubSub[T]) Unsubscribe(topic string, c *Context) {
	tp.ps.Unsubscribe(topic, c)
}

// UnsubscribeAll delegates to the underlying PubSub.
func (tp *TypedPubSub[T]) UnsubscribeAll(c *Context) {
	tp.ps.UnsubscribeAll(c)
}

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
