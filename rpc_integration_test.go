package wt

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/quic-go/webtransport-go"
)

func TestRPCOverQUIC(t *testing.T) {
	l, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		t.Fatal(err)
	}
	port := l.LocalAddr().(*net.UDPAddr).Port
	l.Close()

	addr := fmt.Sprintf("127.0.0.1:%d", port)
	server := New(WithAddr(addr), WithSelfSignedTLS())

	rpcServer := NewRPCServer()
	rpcServer.Register("add", func(params json.RawMessage) (any, error) {
		var args [2]int
		if err := json.Unmarshal(params, &args); err != nil {
			return nil, err
		}
		return args[0] + args[1], nil
	})
	rpcServer.Register("echo", func(params json.RawMessage) (any, error) {
		var msg string
		json.Unmarshal(params, &msg)
		return msg, nil
	})

	server.Handle("/rpc", HandleStream(func(s *Stream, c *Context) {
		rpcServer.Serve(s)
	}))

	go server.ListenAndServe()
	time.Sleep(100 * time.Millisecond)
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	dialer := webtransport.Dialer{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	_, session, err := dialer.Dial(ctx, fmt.Sprintf("https://%s/rpc", addr), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer session.CloseWithError(0, "")

	stream, err := session.OpenStreamSync(ctx)
	if err != nil {
		t.Fatalf("open stream: %v", err)
	}

	client := NewRPCClient(&Stream{raw: stream})

	// Test add method
	t.Run("add", func(t *testing.T) {
		result, err := CallTyped[int](client, "add", [2]int{3, 7})
		if err != nil {
			t.Fatalf("rpc call: %v", err)
		}
		if result != 10 {
			t.Errorf("expected 10, got %d", result)
		}
	})

	// Test echo method
	t.Run("echo", func(t *testing.T) {
		result, err := CallTyped[string](client, "echo", "hello rpc")
		if err != nil {
			t.Fatalf("rpc call: %v", err)
		}
		if result != "hello rpc" {
			t.Errorf("expected 'hello rpc', got %q", result)
		}
	})

	// Test unknown method
	t.Run("unknown", func(t *testing.T) {
		_, err := client.Call("nonexistent", nil)
		if err == nil {
			t.Error("expected error for unknown method")
		}
	})
}
