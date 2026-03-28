package wt

import (
	"encoding/json"
	"testing"
)

func TestRPCErrorString(t *testing.T) {
	err := &RPCError{Code: -32601, Message: "method not found"}
	if err.Error() != "rpc error -32601: method not found" {
		t.Errorf("unexpected error string: %s", err.Error())
	}
}

func TestRPCServerRegister(t *testing.T) {
	rpc := NewRPCServer()

	rpc.Register("add", func(params json.RawMessage) (any, error) {
		return 42, nil
	})

	rpc.mu.RLock()
	_, ok := rpc.handlers["add"]
	rpc.mu.RUnlock()

	if !ok {
		t.Error("expected 'add' handler to be registered")
	}
}

func TestRPCRequestMarshal(t *testing.T) {
	req := RPCRequest{
		ID:     1,
		Method: "echo",
		Params: json.RawMessage(`{"msg":"hello"}`),
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var decoded RPCRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if decoded.ID != 1 || decoded.Method != "echo" {
		t.Errorf("decoded mismatch: %+v", decoded)
	}
}

func TestRPCResponseWithResult(t *testing.T) {
	resp := RPCResponse{
		ID:     1,
		Result: json.RawMessage(`42`),
	}

	data, _ := json.Marshal(resp)
	var decoded RPCResponse
	json.Unmarshal(data, &decoded)

	if decoded.Error != nil {
		t.Error("expected no error")
	}
	if string(decoded.Result) != "42" {
		t.Errorf("expected '42', got %q", decoded.Result)
	}
}

func TestRPCResponseWithError(t *testing.T) {
	resp := RPCResponse{
		ID:    1,
		Error: &RPCError{Code: -32600, Message: "invalid request"},
	}

	data, _ := json.Marshal(resp)
	var decoded RPCResponse
	json.Unmarshal(data, &decoded)

	if decoded.Error == nil {
		t.Fatal("expected error")
	}
	if decoded.Error.Code != -32600 {
		t.Errorf("expected code -32600, got %d", decoded.Error.Code)
	}
}
