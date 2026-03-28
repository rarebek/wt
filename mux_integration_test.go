package wt

import (
	"context"
	"crypto/tls"
	"encoding/binary"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/quic-go/webtransport-go"
)

func TestStreamMuxIntegration(t *testing.T) {
	l, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		t.Fatal(err)
	}
	port := l.LocalAddr().(*net.UDPAddr).Port
	l.Close()

	addr := fmt.Sprintf("127.0.0.1:%d", port)
	server := New(WithAddr(addr), WithSelfSignedTLS())

	const (
		TypeEcho  uint16 = 1
		TypeUpper uint16 = 2
	)

	mux := NewStreamMux()
	mux.Handle(TypeEcho, func(s *Stream, c *Context) {
		defer s.Close()
		msg, err := s.ReadMessage()
		if err != nil {
			return
		}
		_ = s.WriteMessage(msg) // echo as-is
	})
	mux.Handle(TypeUpper, func(s *Stream, c *Context) {
		defer s.Close()
		msg, err := s.ReadMessage()
		if err != nil {
			return
		}
		// Convert to uppercase
		for i, b := range msg {
			if b >= 'a' && b <= 'z' {
				msg[i] = b - 32
			}
		}
		_ = s.WriteMessage(msg)
	})

	server.Handle("/mux", func(c *Context) {
		mux.Serve(c)
	})

	go server.ListenAndServe()
	time.Sleep(100 * time.Millisecond)
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	dialer := webtransport.Dialer{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	_, session, err := dialer.Dial(ctx, fmt.Sprintf("https://%s/mux", addr), nil)
	if err != nil {
		t.Fatalf("dial error: %v", err)
	}
	defer session.CloseWithError(0, "")

	// Test 1: Send to echo handler (type 1)
	t.Run("echo", func(t *testing.T) {
		stream, err := session.OpenStreamSync(ctx)
		if err != nil {
			t.Fatalf("open stream: %v", err)
		}

		// Write type header
		header := make([]byte, 2)
		binary.BigEndian.PutUint16(header, TypeEcho)
		stream.Write(header)

		s := &Stream{raw: stream, ctx: nil}
		_ = s.WriteMessage([]byte("hello"))

		reply, err := s.ReadMessage()
		if err != nil {
			t.Fatalf("read: %v", err)
		}
		if string(reply) != "hello" {
			t.Errorf("expected 'hello', got %q", reply)
		}
	})

	// Test 2: Send to upper handler (type 2)
	t.Run("upper", func(t *testing.T) {
		stream, err := session.OpenStreamSync(ctx)
		if err != nil {
			t.Fatalf("open stream: %v", err)
		}

		header := make([]byte, 2)
		binary.BigEndian.PutUint16(header, TypeUpper)
		stream.Write(header)

		s := &Stream{raw: stream, ctx: nil}
		_ = s.WriteMessage([]byte("hello world"))

		reply, err := s.ReadMessage()
		if err != nil {
			t.Fatalf("read: %v", err)
		}
		if string(reply) != "HELLO WORLD" {
			t.Errorf("expected 'HELLO WORLD', got %q", reply)
		}
	})
}
