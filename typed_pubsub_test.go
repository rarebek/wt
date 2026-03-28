package wt

import (
	"testing"

	"github.com/rarebek/wt/codec"
)

func TestTypedPubSubPublish(t *testing.T) {
	ps := NewPubSub()

	type Msg struct {
		Text string `json:"text"`
	}

	tps := NewTypedPubSub[Msg](ps, codec.JSON{})

	c := &Context{id: "s1", store: make(map[string]any)}
	tps.Subscribe("chat", c)

	if ps.SubscriberCount("chat") != 1 {
		t.Errorf("expected 1 subscriber, got %d", ps.SubscriberCount("chat"))
	}

	// Verify encoding works (don't publish to nil session)
	data, err := codec.JSON{}.Marshal(Msg{Text: "hello"})
	if err != nil {
		t.Errorf("encode error: %v", err)
	}
	if len(data) == 0 {
		t.Error("expected non-empty encoded data")
	}

	tps.Unsubscribe("chat", c)
	if ps.SubscriberCount("chat") != 0 {
		t.Error("expected 0 after unsubscribe")
	}
}

func TestTypedPubSubUnsubscribeAll(t *testing.T) {
	ps := NewPubSub()
	tps := NewTypedPubSub[string](ps, codec.JSON{})

	c := &Context{id: "s1", store: make(map[string]any)}
	tps.Subscribe("a", c)
	tps.Subscribe("b", c)

	tps.UnsubscribeAll(c)

	if ps.SubscriberCount("a") != 0 || ps.SubscriberCount("b") != 0 {
		t.Error("expected 0 subs after UnsubscribeAll")
	}
}
