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

// TestConcurrentSessions tests multiple concurrent WebTransport sessions.
func TestConcurrentSessions(t *testing.T) {
	l, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		t.Fatal(err)
	}
	port := l.LocalAddr().(*net.UDPAddr).Port
	l.Close()

	addr := fmt.Sprintf("127.0.0.1:%d", port)
	server := New(WithAddr(addr), WithSelfSignedTLS())

	var sessionCount atomic.Int64

	server.Handle("/stress", func(c *Context) {
		sessionCount.Add(1)
		defer sessionCount.Add(-1)

		// Echo one stream then close
		stream, err := c.AcceptStream()
		if err != nil {
			return
		}
		msg, err := stream.ReadMessage()
		if err != nil {
			return
		}
		_ = stream.WriteMessage(msg)
		stream.Close()
	})

	go server.ListenAndServe()
	time.Sleep(100 * time.Millisecond)
	defer server.Close()

	const numClients = 10
	var wg sync.WaitGroup
	errors := make(chan error, numClients)

	for i := range numClients {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			dialer := webtransport.Dialer{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			}
			_, session, err := dialer.Dial(ctx, fmt.Sprintf("https://%s/stress", addr), nil)
			if err != nil {
				errors <- fmt.Errorf("client %d dial: %w", id, err)
				return
			}
			defer session.CloseWithError(0, "")

			stream, err := session.OpenStreamSync(ctx)
			if err != nil {
				errors <- fmt.Errorf("client %d open stream: %w", id, err)
				return
			}

			msg := []byte(fmt.Sprintf("hello from client %d", id))
			s := &Stream{raw: stream, ctx: nil}
			if err := s.WriteMessage(msg); err != nil {
				errors <- fmt.Errorf("client %d write: %w", id, err)
				return
			}

			reply, err := s.ReadMessage()
			if err != nil {
				errors <- fmt.Errorf("client %d read: %w", id, err)
				return
			}

			if string(reply) != string(msg) {
				errors <- fmt.Errorf("client %d: expected %q, got %q", id, msg, reply)
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Error(err)
	}
}

// TestConcurrentDatagrams tests multiple concurrent datagram senders.
func TestConcurrentDatagrams(t *testing.T) {
	l, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		t.Fatal(err)
	}
	port := l.LocalAddr().(*net.UDPAddr).Port
	l.Close()

	addr := fmt.Sprintf("127.0.0.1:%d", port)
	server := New(WithAddr(addr), WithSelfSignedTLS())

	var received atomic.Int64

	server.Handle("/dgstress", func(c *Context) {
		for {
			data, err := c.ReceiveDatagram()
			if err != nil {
				return
			}
			received.Add(1)
			_ = c.SendDatagram(data)
		}
	})

	go server.ListenAndServe()
	time.Sleep(100 * time.Millisecond)
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	dialer := webtransport.Dialer{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	_, session, err := dialer.Dial(ctx, fmt.Sprintf("https://%s/dgstress", addr), nil)
	if err != nil {
		t.Fatalf("dial error: %v", err)
	}
	defer session.CloseWithError(0, "")

	// Send 100 datagrams rapidly
	const count = 100
	for i := range count {
		msg := []byte(fmt.Sprintf("dg-%d", i))
		if err := session.SendDatagram(msg); err != nil {
			t.Fatalf("send datagram %d: %v", i, err)
		}
	}

	// Wait for some to be echoed back (datagrams are unreliable, some may be lost)
	time.Sleep(500 * time.Millisecond)

	got := received.Load()
	if got == 0 {
		t.Error("no datagrams received by server")
	}
	t.Logf("sent %d datagrams, server received %d (%.0f%%)", count, got, float64(got)/float64(count)*100)
}
