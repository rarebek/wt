package wt

import (
	"encoding/json"
	"testing"
)

func BenchmarkRPCRequestMarshal(b *testing.B) {
	req := RPCRequest{ID: 1, Method: "echo", Params: json.RawMessage(`{"msg":"hello"}`)}
	b.ResetTimer()
	for b.Loop() {
		json.Marshal(req)
	}
}

func BenchmarkRPCResponseMarshal(b *testing.B) {
	resp := RPCResponse{ID: 1, Result: json.RawMessage(`42`)}
	b.ResetTimer()
	for b.Loop() {
		json.Marshal(resp)
	}
}
