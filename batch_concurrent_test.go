package wt

import (
	"sync"
	"testing"
)

func TestDecodeBatchConcurrent(t *testing.T) {
	batch := defaultBatchEncode([][]byte{
		[]byte("hello"),
		[]byte("world"),
		[]byte("test"),
	})

	var wg sync.WaitGroup
	for range 100 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			msgs := DecodeBatch(batch)
			if len(msgs) != 3 {
				t.Errorf("expected 3 messages, got %d", len(msgs))
			}
		}()
	}
	wg.Wait()
}
