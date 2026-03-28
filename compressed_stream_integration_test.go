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

func TestCompressedStreamOverQUIC(t *testing.T) {
	l, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		t.Fatal(err)
	}
	port := l.LocalAddr().(*net.UDPAddr).Port
	l.Close()

	addr := fmt.Sprintf("127.0.0.1:%d", port)
	server := New(WithAddr(addr), WithSelfSignedTLS())

	server.Handle("/compress", HandleStream(func(s *Stream, c *Context) {
		cs := NewCompressedStream(s, 64) // compress messages > 64 bytes
		defer cs.Close()

		msg, err := cs.ReadMessage()
		if err != nil {
			return
		}
		_ = cs.WriteMessage(msg) // echo compressed
	}))

	go server.ListenAndServe()
	time.Sleep(100 * time.Millisecond)
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	dialer := webtransport.Dialer{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	_, session, err := dialer.Dial(ctx, fmt.Sprintf("https://%s/compress", addr), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer session.CloseWithError(0, "")

	// Test with large message (should be compressed)
	t.Run("large_message", func(t *testing.T) {
		stream, err := session.OpenStreamSync(ctx)
		if err != nil {
			t.Fatalf("open stream: %v", err)
		}

		cs := NewCompressedStream(&Stream{raw: stream}, 64)

		// Large, compressible message
		bigMsg := bytes.Repeat([]byte("hello world compressed "), 100)

		if err := cs.WriteMessage(bigMsg); err != nil {
			t.Fatalf("write: %v", err)
		}

		reply, err := cs.ReadMessage()
		if err != nil {
			t.Fatalf("read: %v", err)
		}

		if !bytes.Equal(reply, bigMsg) {
			t.Errorf("reply mismatch: got %d bytes, want %d", len(reply), len(bigMsg))
		}
	})

	// Test with small message (should NOT be compressed)
	t.Run("small_message", func(t *testing.T) {
		stream, err := session.OpenStreamSync(ctx)
		if err != nil {
			t.Fatalf("open stream: %v", err)
		}

		cs := NewCompressedStream(&Stream{raw: stream}, 64)

		smallMsg := []byte("hi")

		if err := cs.WriteMessage(smallMsg); err != nil {
			t.Fatalf("write: %v", err)
		}

		reply, err := cs.ReadMessage()
		if err != nil {
			t.Fatalf("read: %v", err)
		}

		if string(reply) != "hi" {
			t.Errorf("expected 'hi', got %q", reply)
		}
	})
}
