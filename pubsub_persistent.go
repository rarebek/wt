package wt

import "sync"

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
