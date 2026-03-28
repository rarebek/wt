package wt

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/quic-go/webtransport-go"
)

func TestScale100Sessions(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping scale test in short mode")
	}

	l, _ := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	port := l.LocalAddr().(*net.UDPAddr).Port
	l.Close()

	addr := fmt.Sprintf("127.0.0.1:%d", port)
	server := New(WithAddr(addr), WithSelfSignedTLS())

	var active atomic.Int64
	var peak atomic.Int64
	var completed atomic.Int64

	server.Handle("/scale", HandleStream(func(s *Stream, c *Context) {
		cur := active.Add(1)
		defer active.Add(-1)
		// Track peak
		for {
			old := peak.Load()
			if cur <= old || peak.CompareAndSwap(old, cur) {
				break
			}
		}

		defer s.Close()
		msg, err := s.ReadMessage()
		if err != nil {
			return
		}
		s.WriteMessage(msg)
		completed.Add(1)
	}))

	go server.ListenAndServe()
	time.Sleep(150 * time.Millisecond)
	defer server.Close()

	const numClients = 100
	var wg sync.WaitGroup
	var errors atomic.Int64

	start := time.Now()

	for i := range numClients {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			dialer := webtransport.Dialer{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			}
			_, session, err := dialer.Dial(ctx, fmt.Sprintf("https://%s/scale", addr), nil)
			if err != nil {
				errors.Add(1)
				return
			}
			defer session.CloseWithError(0, "")

			stream, err := session.OpenStreamSync(ctx)
			if err != nil {
				errors.Add(1)
				return
			}

			s := &Stream{raw: stream}
			msg := []byte(fmt.Sprintf("client-%d", id))
			if err := s.WriteMessage(msg); err != nil {
				errors.Add(1)
				return
			}

			reply, err := s.ReadMessage()
			if err != nil {
				errors.Add(1)
				return
			}

			if string(reply) != string(msg) {
				errors.Add(1)
			}
		}(i)
	}

	wg.Wait()
	elapsed := time.Since(start)

	t.Logf("100 sessions: %v elapsed, %d completed, %d errors, peak concurrent: %d",
		elapsed.Truncate(time.Millisecond), completed.Load(), errors.Load(), peak.Load())

	if errors.Load() > 5 { // allow a few failures under load
		t.Errorf("too many errors: %d/%d", errors.Load(), numClients)
	}

	time.Sleep(200 * time.Millisecond)
	if server.SessionCount() != 0 {
		t.Errorf("leaked sessions: %d", server.SessionCount())
	}
}
