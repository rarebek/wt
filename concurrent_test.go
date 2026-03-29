package wt

import (
	"fmt"
	"sync"
	"testing"
)

func TestSessionStoreConcurrent(t *testing.T) {
	ss := NewSessionStore()
	var wg sync.WaitGroup

	// Concurrent add/remove/count
	for i := range 100 {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			ctx := &Context{id: fmt.Sprintf("s-%d", id), store: make(map[string]any)}
			ss.Add(ctx)
			_ = ss.Count()
			ss.Remove(ctx.ID())
		}(i)
	}
	wg.Wait()

	if ss.Count() != 0 {
		t.Errorf("expected 0 after concurrent add/remove, got %d", ss.Count())
	}
}

func TestRoomConcurrentJoinLeave(t *testing.T) {
	rm := NewRoomManager()
	room := rm.GetOrCreate("test")
	var wg sync.WaitGroup

	for i := range 100 {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			ctx := &Context{id: fmt.Sprintf("m-%d", id), store: make(map[string]any)}
			room.Join(ctx)
			_ = room.Count()
			room.Leave(ctx)
		}(i)
	}
	wg.Wait()

	if room.Count() != 0 {
		t.Errorf("expected 0 after concurrent join/leave, got %d", room.Count())
	}
}

func TestPubSubConcurrent(t *testing.T) {
	ps := NewPubSub()
	var wg sync.WaitGroup

	for i := range 50 {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			ctx := &Context{id: fmt.Sprintf("p-%d", id), store: make(map[string]any)}
			ps.Subscribe("topic", ctx)
			_ = ps.SubscriberCount("topic")
			ps.Unsubscribe("topic", ctx)
		}(i)
	}
	wg.Wait()

	if ps.SubscriberCount("topic") != 0 {
		t.Errorf("expected 0 after concurrent sub/unsub, got %d", ps.SubscriberCount("topic"))
	}
}

func TestTagsConcurrent(t *testing.T) {
	tags := NewTags()
	var wg sync.WaitGroup

	for i := range 50 {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			sid := fmt.Sprintf("s-%d", id)
			tags.Tag(sid, "vip")
			_ = tags.HasTag(sid, "vip")
			tags.Untag(sid, "vip")
		}(i)
	}
	wg.Wait()

	if tags.Count("vip") != 0 {
		t.Errorf("expected 0 after concurrent tag/untag, got %d", tags.Count("vip"))
	}
}

func TestEventBusConcurrent(t *testing.T) {
	bus := NewEventBus()
	var count int64
	var mu sync.Mutex

	bus.On(EventConnect, func(e Event) {
		mu.Lock()
		count++
		mu.Unlock()
	})

	var wg sync.WaitGroup
	for range 100 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			bus.Emit(Event{Type: EventConnect})
		}()
	}
	wg.Wait()

	if count != 100 {
		t.Errorf("expected 100 events, got %d", count)
	}
}
