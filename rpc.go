package wt

import (
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
)

// RPCRequest represents a JSON-RPC-like request over a stream.
type RPCRequest struct {
	ID     uint64          `json:"id"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params,omitempty"`
}

// RPCResponse represents a JSON-RPC-like response.
type RPCResponse struct {
	ID     uint64          `json:"id"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  *RPCError       `json:"error,omitempty"`
}

// RPCError represents an error in the RPC response.
type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *RPCError) Error() string {
	return fmt.Sprintf("rpc error %d: %s", e.Code, e.Message)
}

// RPCHandler handles a single RPC method.
type RPCHandler func(params json.RawMessage) (any, error)

// RPCServer handles JSON-RPC requests over a WebTransport stream.
type RPCServer struct {
	mu       sync.RWMutex
	handlers map[string]RPCHandler
}

// NewRPCServer creates a new RPC server.
func NewRPCServer() *RPCServer {
	return &RPCServer{
		handlers: make(map[string]RPCHandler),
	}
}

// Register adds a handler for the given method name.
func (rpc *RPCServer) Register(method string, handler RPCHandler) {
	rpc.mu.Lock()
	rpc.handlers[method] = handler
	rpc.mu.Unlock()
}

// Serve handles RPC requests on the given stream.
// Each request-response pair uses the stream's message framing.
// Call this from a StreamHandler or HandleStream.
func (rpc *RPCServer) Serve(s *Stream) {
	defer s.Close()

	for {
		data, err := s.ReadMessage()
		if err != nil {
			return
		}

		var req RPCRequest
		if err := json.Unmarshal(data, &req); err != nil {
			resp := RPCResponse{
				Error: &RPCError{Code: -32700, Message: "parse error"},
			}
			respData, _ := json.Marshal(resp)
			s.WriteMessage(respData)
			continue
		}

		rpc.mu.RLock()
		handler, ok := rpc.handlers[req.Method]
		rpc.mu.RUnlock()

		var resp RPCResponse
		resp.ID = req.ID

		if !ok {
			resp.Error = &RPCError{Code: -32601, Message: fmt.Sprintf("method not found: %s", req.Method)}
		} else {
			result, err := handler(req.Params)
			if err != nil {
				resp.Error = &RPCError{Code: -32000, Message: err.Error()}
			} else {
				resultData, _ := json.Marshal(result)
				resp.Result = resultData
			}
		}

		respData, _ := json.Marshal(resp)
		if err := s.WriteMessage(respData); err != nil {
			return
		}
	}
}

// RPCClient sends JSON-RPC requests over a WebTransport stream.
type RPCClient struct {
	stream *Stream
	nextID atomic.Uint64
}

// NewRPCClient wraps a stream for RPC calls.
func NewRPCClient(s *Stream) *RPCClient {
	return &RPCClient{stream: s}
}

// Call sends an RPC request and waits for the response.
func (c *RPCClient) Call(method string, params any) (json.RawMessage, error) {
	id := c.nextID.Add(1)

	paramsData, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}

	req := RPCRequest{
		ID:     id,
		Method: method,
		Params: paramsData,
	}

	reqData, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	if err := c.stream.WriteMessage(reqData); err != nil {
		return nil, err
	}

	respData, err := c.stream.ReadMessage()
	if err != nil {
		return nil, err
	}

	var resp RPCResponse
	if err := json.Unmarshal(respData, &resp); err != nil {
		return nil, err
	}

	if resp.Error != nil {
		return nil, resp.Error
	}

	return resp.Result, nil
}

// CallTyped sends an RPC request and unmarshals the result into the given type.
func CallTyped[T any](c *RPCClient, method string, params any) (T, error) {
	var zero T
	result, err := c.Call(method, params)
	if err != nil {
		return zero, err
	}

	var v T
	if err := json.Unmarshal(result, &v); err != nil {
		return zero, err
	}
	return v, nil
}

// Close closes the underlying stream.
func (c *RPCClient) Close() error {
	return c.stream.Close()
}
