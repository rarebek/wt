package wt

import (
	"encoding/json"
	"testing"
)

func TestRPCErrorInterface(t *testing.T) {
	err := &RPCError{Code: -32600, Message: "invalid request"}

	var e error = err
	if e.Error() != "rpc error -32600: invalid request" {
		t.Errorf("unexpected error: %s", e.Error())
	}
}

func TestRPCServerMethodNotFound(t *testing.T) {
	rpc := NewRPCServer()
	rpc.Register("test", func(params json.RawMessage) (any, error) {
		return nil, nil
	})

	rpc.mu.RLock()
	if _, ok := rpc.handlers["test"]; !ok {
		t.Error("test handler should be registered")
	}
	if _, ok := rpc.handlers["missing"]; ok {
		t.Error("missing handler should not exist")
	}
	rpc.mu.RUnlock()
}
