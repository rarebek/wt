package wt

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/quic-go/webtransport-go"
)

func TestHandleStreamConvenience(t *testing.T) {
	l, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		t.Fatal(err)
	}
	port := l.LocalAddr().(*net.UDPAddr).Port
	l.Close()

	addr := fmt.Sprintf("127.0.0.1:%d", port)
	server := New(WithAddr(addr), WithSelfSignedTLS())

	// Use HandleStream convenience
	server.Handle("/streamconv", HandleStream(func(s *Stream, c *Context) {
		defer s.Close()
		msg, err := s.ReadMessage()
		if err != nil {
			return
		}
		_ = s.WriteMessage(append([]byte("echo:"), msg...))
	}))

	go server.ListenAndServe()
	time.Sleep(100 * time.Millisecond)
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	dialer := webtransport.Dialer{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	_, session, err := dialer.Dial(ctx, fmt.Sprintf("https://%s/streamconv", addr), nil)
	if err != nil {
		t.Fatalf("dial error: %v", err)
	}
	defer session.CloseWithError(0, "")

	stream, err := session.OpenStreamSync(ctx)
	if err != nil {
		t.Fatalf("open stream: %v", err)
	}

	s := &Stream{raw: stream, ctx: nil}
	_ = s.WriteMessage([]byte("test"))

	reply, err := s.ReadMessage()
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	if string(reply) != "echo:test" {
		t.Errorf("expected 'echo:test', got %q", reply)
	}
}

func TestHandleDatagramConvenience(t *testing.T) {
	l, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		t.Fatal(err)
	}
	port := l.LocalAddr().(*net.UDPAddr).Port
	l.Close()

	addr := fmt.Sprintf("127.0.0.1:%d", port)
	server := New(WithAddr(addr), WithSelfSignedTLS())

	// Use HandleDatagram convenience
	server.Handle("/dgconv", HandleDatagram(func(data []byte, c *Context) []byte {
		return append([]byte("pong:"), data...)
	}))

	go server.ListenAndServe()
	time.Sleep(100 * time.Millisecond)
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	dialer := webtransport.Dialer{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	_, session, err := dialer.Dial(ctx, fmt.Sprintf("https://%s/dgconv", addr), nil)
	if err != nil {
		t.Fatalf("dial error: %v", err)
	}
	defer session.CloseWithError(0, "")

	if err := session.SendDatagram([]byte("hello")); err != nil {
		t.Fatalf("send: %v", err)
	}

	reply, err := session.ReceiveDatagram(ctx)
	if err != nil {
		t.Fatalf("receive: %v", err)
	}

	if string(reply) != "pong:hello" {
		t.Errorf("expected 'pong:hello', got %q", reply)
	}
}
