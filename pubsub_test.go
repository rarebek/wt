package wt

import "testing"

func TestPubSubSubscribePublish(t *testing.T) {
	ps := NewPubSub()

	c1 := &Context{id: "s1", store: make(map[string]any)}
	c2 := &Context{id: "s2", store: make(map[string]any)}

	ps.Subscribe("news", c1)
	ps.Subscribe("news", c2)
	ps.Subscribe("sports", c1)

	if ps.SubscriberCount("news") != 2 {
		t.Errorf("expected 2 news subs, got %d", ps.SubscriberCount("news"))
	}
	if ps.SubscriberCount("sports") != 1 {
		t.Errorf("expected 1 sports sub, got %d", ps.SubscriberCount("sports"))
	}
}

func TestPubSubUnsubscribe(t *testing.T) {
	ps := NewPubSub()

	c1 := &Context{id: "s1", store: make(map[string]any)}
	ps.Subscribe("topic", c1)
	ps.Unsubscribe("topic", c1)

	if ps.SubscriberCount("topic") != 0 {
		t.Errorf("expected 0 after unsubscribe, got %d", ps.SubscriberCount("topic"))
	}
}

func TestPubSubUnsubscribeAll(t *testing.T) {
	ps := NewPubSub()

	c1 := &Context{id: "s1", store: make(map[string]any)}
	ps.Subscribe("a", c1)
	ps.Subscribe("b", c1)
	ps.Subscribe("c", c1)

	ps.UnsubscribeAll(c1)

	topics := ps.TopicsForSession("s1")
	if len(topics) != 0 {
		t.Errorf("expected 0 topics after UnsubscribeAll, got %d", len(topics))
	}
}

func TestPubSubTopics(t *testing.T) {
	ps := NewPubSub()

	c := &Context{id: "s1", store: make(map[string]any)}
	ps.Subscribe("x", c)
	ps.Subscribe("y", c)
	ps.Subscribe("z", c)

	topics := ps.Topics()
	if len(topics) != 3 {
		t.Errorf("expected 3 topics, got %d", len(topics))
	}
}

func TestPubSubTopicsForSession(t *testing.T) {
	ps := NewPubSub()

	c := &Context{id: "s1", store: make(map[string]any)}
	ps.Subscribe("news", c)
	ps.Subscribe("weather", c)

	topics := ps.TopicsForSession("s1")
	if len(topics) != 2 {
		t.Errorf("expected 2 topics, got %d", len(topics))
	}
}

func TestPubSubEmpty(t *testing.T) {
	ps := NewPubSub()

	if ps.SubscriberCount("nothing") != 0 {
		t.Error("expected 0")
	}
	if len(ps.Topics()) != 0 {
		t.Error("expected empty topics")
	}
}
