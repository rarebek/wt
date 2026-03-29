package wt

import (
	"fmt"
	"sync"
	"testing"
)

func TestKVSyncConcurrent(t *testing.T) {
	kv := NewKVSync()
	var wg sync.WaitGroup

	// Concurrent writes
	for i := range 100 {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			kv.Set(fmt.Sprintf("key-%d", id), id)
		}(i)
	}

	// Concurrent reads
	for range 50 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = kv.Keys()
			_ = kv.Len()
			_ = kv.Snapshot()
		}()
	}

	wg.Wait()

	if kv.Len() != 100 {
		t.Errorf("expected 100 keys, got %d", kv.Len())
	}
}

func TestKVSyncConcurrentDelete(t *testing.T) {
	kv := NewKVSync()
	for i := range 50 {
		kv.Set(fmt.Sprintf("k-%d", i), i)
	}

	var wg sync.WaitGroup
	for i := range 50 {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			kv.Delete(fmt.Sprintf("k-%d", id))
		}(i)
	}
	wg.Wait()

	if kv.Len() != 0 {
		t.Errorf("expected 0 after concurrent delete, got %d", kv.Len())
	}
}
