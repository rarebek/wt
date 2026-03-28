package wt

import (
	"encoding/json"
	"testing"
)

func TestKVSyncSetGet(t *testing.T) {
	kv := NewKVSync()

	if err := kv.Set("name", "alice"); err != nil {
		t.Fatalf("set error: %v", err)
	}

	var name string
	if err := kv.Get("name", &name); err != nil {
		t.Fatalf("get error: %v", err)
	}

	if name != "alice" {
		t.Errorf("expected 'alice', got %q", name)
	}
}

func TestKVSyncDelete(t *testing.T) {
	kv := NewKVSync()
	kv.Set("key", "value")
	kv.Delete("key")

	_, ok := kv.GetRaw("key")
	if ok {
		t.Error("expected key to be deleted")
	}
}

func TestKVSyncKeys(t *testing.T) {
	kv := NewKVSync()
	kv.Set("a", 1)
	kv.Set("b", 2)
	kv.Set("c", 3)

	keys := kv.Keys()
	if len(keys) != 3 {
		t.Errorf("expected 3 keys, got %d", len(keys))
	}
}

func TestKVSyncLen(t *testing.T) {
	kv := NewKVSync()
	if kv.Len() != 0 {
		t.Error("expected 0 length")
	}

	kv.Set("x", true)
	if kv.Len() != 1 {
		t.Errorf("expected 1, got %d", kv.Len())
	}
}

func TestKVSyncOnChange(t *testing.T) {
	kv := NewKVSync()

	var changedKey string
	kv.OnChange(func(key string, _ json.RawMessage) {
		changedKey = key
	})

	kv.Set("test", 42)

	if changedKey != "test" {
		t.Errorf("expected onChange with key 'test', got %q", changedKey)
	}
}

func TestKVSyncSnapshot(t *testing.T) {
	kv := NewKVSync()
	kv.Set("a", 1)
	kv.Set("b", "two")

	snap := kv.Snapshot()
	if len(snap) != 2 {
		t.Errorf("expected 2 items in snapshot, got %d", len(snap))
	}
}

func TestKVSyncGetMissing(t *testing.T) {
	kv := NewKVSync()

	var val int
	err := kv.Get("nonexistent", &val)
	if err != nil {
		t.Errorf("get missing key should not error: %v", err)
	}
	if val != 0 {
		t.Errorf("expected zero value, got %d", val)
	}
}
