package wt

import (
	"fmt"
	"sync"
	"testing"
)

func TestPresenceConcurrent(t *testing.T) {
	pt := NewPresenceTracker()
	var wg sync.WaitGroup

	for i := range 50 {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			ctx := &Context{id: fmt.Sprintf("s-%d", id), store: map[string]any{"user": fmt.Sprintf("u-%d", id)}}
			pt.Join("room", ctx)
			pt.UpdateStatus("room", ctx.ID(), "typing")
			_ = pt.GetPresence("room")
			_ = pt.Count("room")
			pt.Leave("room", ctx)
		}(i)
	}
	wg.Wait()

	if pt.Count("room") != 0 {
		t.Errorf("expected 0 after concurrent join/leave, got %d", pt.Count("room"))
	}
}
