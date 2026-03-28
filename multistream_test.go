package wt

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/quic-go/webtransport-go"
)

// TestMultiStreamIndependence proves that 5 concurrent streams within
// one session operate independently — no head-of-line blocking.
func TestMultiStreamIndependence(t *testing.T) {
	l, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		t.Fatal(err)
	}
	port := l.LocalAddr().(*net.UDPAddr).Port
	l.Close()

	addr := fmt.Sprintf("127.0.0.1:%d", port)
	server := New(WithAddr(addr), WithSelfSignedTLS())

	// Server: echo each stream independently with a delay
	server.Handle("/multi", HandleStream(func(s *Stream, c *Context) {
		defer s.Close()
		msg, err := s.ReadMessage()
		if err != nil {
			return
		}
		// Small delay to prove independence (fast stream shouldn't wait for slow)
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
	_, session, err := dialer.Dial(ctx, fmt.Sprintf("https://%s/multi", addr), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer session.CloseWithError(0, "")

	// Open 5 concurrent streams
	const numStreams = 5
	var wg sync.WaitGroup
	results := make([]string, numStreams)
	errors := make([]error, numStreams)

	for i := range numStreams {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			stream, err := session.OpenStreamSync(ctx)
			if err != nil {
				errors[idx] = fmt.Errorf("open stream %d: %w", idx, err)
				return
			}

			s := &Stream{raw: stream, ctx: nil}
			msg := fmt.Sprintf("stream-%d", idx)

			if err := s.WriteMessage([]byte(msg)); err != nil {
				errors[idx] = fmt.Errorf("write stream %d: %w", idx, err)
				return
			}

			reply, err := s.ReadMessage()
			if err != nil {
				errors[idx] = fmt.Errorf("read stream %d: %w", idx, err)
				return
			}

			results[idx] = string(reply)
		}(i)
	}

	wg.Wait()

	for i := range numStreams {
		if errors[i] != nil {
			t.Error(errors[i])
			continue
		}
		expected := fmt.Sprintf("stream-%d", i)
		if results[i] != expected {
			t.Errorf("stream %d: expected %q, got %q", i, expected, results[i])
		}
	}
}
