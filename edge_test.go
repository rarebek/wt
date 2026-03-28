package wt

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/quic-go/webtransport-go"
)

// TestLargeMessage tests sending messages near the maximum size.
func TestLargeMessage(t *testing.T) {
	l, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		t.Fatal(err)
	}
	port := l.LocalAddr().(*net.UDPAddr).Port
	l.Close()

	addr := fmt.Sprintf("127.0.0.1:%d", port)
	server := New(WithAddr(addr), WithSelfSignedTLS())

	server.Handle("/large", HandleStream(func(s *Stream, c *Context) {
		defer s.Close()
		msg, err := s.ReadMessage()
		if err != nil {
			return
		}
		_ = s.WriteMessage(msg)
	}))

	go server.ListenAndServe()
	time.Sleep(100 * time.Millisecond)
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dialer := webtransport.Dialer{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	_, session, err := dialer.Dial(ctx, fmt.Sprintf("https://%s/large", addr), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer session.CloseWithError(0, "")

	// Send a 1MB message
	stream, err := session.OpenStreamSync(ctx)
	if err != nil {
		t.Fatalf("open stream: %v", err)
	}

	bigMsg := make([]byte, 1024*1024) // 1 MB
	for i := range bigMsg {
		bigMsg[i] = byte(i % 256)
	}

	s := &Stream{raw: stream, ctx: nil}
	if err := s.WriteMessage(bigMsg); err != nil {
		t.Fatalf("write large message: %v", err)
	}

	reply, err := s.ReadMessage()
	if err != nil {
		t.Fatalf("read large message: %v", err)
	}

	if len(reply) != len(bigMsg) {
		t.Errorf("reply length %d, want %d", len(reply), len(bigMsg))
	}
	if !bytes.Equal(reply, bigMsg) {
		t.Error("reply content doesn't match")
	}
}

// TestEmptyMessage tests sending zero-length messages.
func TestEmptyMessage(t *testing.T) {
	l, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		t.Fatal(err)
	}
	port := l.LocalAddr().(*net.UDPAddr).Port
	l.Close()

	addr := fmt.Sprintf("127.0.0.1:%d", port)
	server := New(WithAddr(addr), WithSelfSignedTLS())

	server.Handle("/empty", HandleStream(func(s *Stream, c *Context) {
		defer s.Close()
		msg, err := s.ReadMessage()
		if err != nil {
			return
		}
		_ = s.WriteMessage(msg) // echo empty
	}))

	go server.ListenAndServe()
	time.Sleep(100 * time.Millisecond)
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	dialer := webtransport.Dialer{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	_, session, err := dialer.Dial(ctx, fmt.Sprintf("https://%s/empty", addr), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer session.CloseWithError(0, "")

	stream, err := session.OpenStreamSync(ctx)
	if err != nil {
		t.Fatalf("open stream: %v", err)
	}

	s := &Stream{raw: stream, ctx: nil}
	if err := s.WriteMessage([]byte{}); err != nil {
		t.Fatalf("write empty message: %v", err)
	}

	reply, err := s.ReadMessage()
	if err != nil {
		t.Fatalf("read empty message: %v", err)
	}

	if len(reply) != 0 {
		t.Errorf("expected empty reply, got %d bytes", len(reply))
	}
}

// TestMultipleRoutes tests that different paths route to different handlers.
func TestMultipleRoutes(t *testing.T) {
	l, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		t.Fatal(err)
	}
	port := l.LocalAddr().(*net.UDPAddr).Port
	l.Close()

	addr := fmt.Sprintf("127.0.0.1:%d", port)
	server := New(WithAddr(addr), WithSelfSignedTLS())

	server.Handle("/route/a", HandleStream(func(s *Stream, c *Context) {
		defer s.Close()
		_ = s.WriteMessage([]byte("handler-a"))
	}))
	server.Handle("/route/b", HandleStream(func(s *Stream, c *Context) {
		defer s.Close()
		_ = s.WriteMessage([]byte("handler-b"))
	}))

	go server.ListenAndServe()
	time.Sleep(100 * time.Millisecond)
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	dialer := webtransport.Dialer{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	// Connect to route A
	_, sessionA, err := dialer.Dial(ctx, fmt.Sprintf("https://%s/route/a", addr), nil)
	if err != nil {
		t.Fatalf("dial a: %v", err)
	}
	defer sessionA.CloseWithError(0, "")

	streamA, _ := sessionA.OpenStreamSync(ctx)
	sA := &Stream{raw: streamA, ctx: nil}
	_ = sA.WriteMessage([]byte("trigger"))
	replyA, _ := sA.ReadMessage()

	if string(replyA) != "handler-a" {
		t.Errorf("route /a returned %q, want 'handler-a'", replyA)
	}

	// Connect to route B
	_, sessionB, err := dialer.Dial(ctx, fmt.Sprintf("https://%s/route/b", addr), nil)
	if err != nil {
		t.Fatalf("dial b: %v", err)
	}
	defer sessionB.CloseWithError(0, "")

	streamB, _ := sessionB.OpenStreamSync(ctx)
	sB := &Stream{raw: streamB, ctx: nil}
	_ = sB.WriteMessage([]byte("trigger"))
	replyB, _ := sB.ReadMessage()

	if string(replyB) != "handler-b" {
		t.Errorf("route /b returned %q, want 'handler-b'", replyB)
	}
}
