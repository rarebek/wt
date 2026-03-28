package wt

import "github.com/rarebek/wt/codec"

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
