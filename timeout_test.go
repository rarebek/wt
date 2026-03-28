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

func TestStreamWithTimeout(t *testing.T) {
	l, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		t.Fatal(err)
	}
	port := l.LocalAddr().(*net.UDPAddr).Port
	l.Close()

	addr := fmt.Sprintf("127.0.0.1:%d", port)
	server := New(WithAddr(addr), WithSelfSignedTLS())

	server.Handle("/timeout", HandleStream(func(s *Stream, c *Context) {
		// Use WithTimeout to auto-close stream after 500ms
		cs := s.WithTimeout(500 * time.Millisecond)
		defer cs.Close()

		// This read should fail because the client won't send anything
		// and the timeout will close the stream
		_, err := cs.ReadMessageContext()
		if err == nil {
			t.Error("expected error from timeout")
		}
	}))

	go server.ListenAndServe()
	time.Sleep(100 * time.Millisecond)
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	dialer := webtransport.Dialer{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	_, session, err := dialer.Dial(ctx, fmt.Sprintf("https://%s/timeout", addr), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer session.CloseWithError(0, "")

	// Open stream but don't send anything — server should timeout
	_, err = session.OpenStreamSync(ctx)
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	// Wait for server timeout
	time.Sleep(800 * time.Millisecond)
}
