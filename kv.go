package wt

import (
	"encoding/json"
	"sync"
)

// KVSync provides a synchronized key-value store that can be shared
// between server and client via a stream.
// Useful for game state sync, config sync, or shared document state.
type KVSync struct {
	mu       sync.RWMutex
	data     map[string]json.RawMessage
	onChange func(key string, value json.RawMessage)
}

// NewKVSync creates a new synchronized key-value store.
func NewKVSync() *KVSync {
	return &KVSync{
		data: make(map[string]json.RawMessage),
	}
}

// Set sets a key-value pair and notifies the onChange callback.
func (kv *KVSync) Set(key string, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	kv.mu.Lock()
	kv.data[key] = data
	cb := kv.onChange
	kv.mu.Unlock()

	if cb != nil {
		cb(key, data)
	}
	return nil
}

// Get retrieves a value by key and unmarshals it into v.
func (kv *KVSync) Get(key string, v any) error {
	kv.mu.RLock()
	data, ok := kv.data[key]
	kv.mu.RUnlock()

	if !ok {
		return nil
	}
	return json.Unmarshal(data, v)
}

// GetRaw retrieves the raw JSON for a key.
func (kv *KVSync) GetRaw(key string) (json.RawMessage, bool) {
	kv.mu.RLock()
	data, ok := kv.data[key]
	kv.mu.RUnlock()
	return data, ok
}

// Delete removes a key.
func (kv *KVSync) Delete(key string) {
	kv.mu.Lock()
	delete(kv.data, key)
	kv.mu.Unlock()
}

// Keys returns all keys.
func (kv *KVSync) Keys() []string {
	kv.mu.RLock()
	defer kv.mu.RUnlock()
	keys := make([]string, 0, len(kv.data))
	for k := range kv.data {
		keys = append(keys, k)
	}
	return keys
}

// Snapshot returns a copy of all key-value pairs as raw JSON.
func (kv *KVSync) Snapshot() map[string]json.RawMessage {
	kv.mu.RLock()
	defer kv.mu.RUnlock()
	snap := make(map[string]json.RawMessage, len(kv.data))
	for k, v := range kv.data {
		snap[k] = v
	}
	return snap
}

// OnChange sets a callback that fires when a key is updated.
func (kv *KVSync) OnChange(fn func(key string, value json.RawMessage)) {
	kv.mu.Lock()
	kv.onChange = fn
	kv.mu.Unlock()
}

// Len returns the number of keys.
func (kv *KVSync) Len() int {
	kv.mu.RLock()
	n := len(kv.data)
	kv.mu.RUnlock()
	return n
}
