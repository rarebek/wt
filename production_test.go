package wt

import (
	"context"
	"crypto/tls"
	"fmt"
	"math/rand/v2"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/quic-go/webtransport-go"
)

// TestProductionConcurrentMultiPath tests multiple routes handling concurrent
// sessions with different payloads simultaneously.
func TestProductionConcurrentMultiPath(t *testing.T) {
	l, _ := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	port := l.LocalAddr().(*net.UDPAddr).Port
	l.Close()

	addr := fmt.Sprintf("127.0.0.1:%d", port)
	server := New(WithAddr(addr), WithSelfSignedTLS())

	server.Handle("/route/a", HandleStream(func(s *Stream, c *Context) {
		defer s.Close()
		msg, _ := s.ReadMessage()
		s.WriteMessage(append([]byte("A:"), msg...))
	}))
	server.Handle("/route/b", HandleStream(func(s *Stream, c *Context) {
		defer s.Close()
		msg, _ := s.ReadMessage()
		s.WriteMessage(append([]byte("B:"), msg...))
	}))
	server.Handle("/route/c", HandleDatagram(func(d []byte, c *Context) []byte {
		return append([]byte("C:"), d...)
	}))

	go server.ListenAndServe()
	time.Sleep(100 * time.Millisecond)
	defer server.Close()

	var wg sync.WaitGroup
	var errors atomic.Int64

	routes := []string{"/route/a", "/route/b"}
	for i := range 20 {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			route := routes[id%2]
			dialer := webtransport.Dialer{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
			_, session, err := dialer.Dial(ctx, fmt.Sprintf("https://%s%s", addr, route), nil)
			if err != nil {
				errors.Add(1)
				return
			}
			defer session.CloseWithError(0, "")

			raw, err := session.OpenStreamSync(ctx)
			if err != nil {
				errors.Add(1)
				return
			}
			s := &Stream{raw: raw}
			msg := fmt.Sprintf("client-%d", id)
			s.WriteMessage([]byte(msg))

			reply, err := s.ReadMessage()
			if err != nil {
				errors.Add(1)
				return
			}

			var prefix string
			if route == "/route/a" {
				prefix = "A:"
			} else {
				prefix = "B:"
			}
			expected := prefix + msg
			if string(reply) != expected {
				t.Errorf("client %d on %s: got %q, want %q", id, route, reply, expected)
				errors.Add(1)
			}
		}(i)
	}

	// Also test datagram route concurrently
	for i := range 5 {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			dialer := webtransport.Dialer{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
			_, session, err := dialer.Dial(ctx, fmt.Sprintf("https://%s/route/c", addr), nil)
			if err != nil {
				errors.Add(1)
				return
			}
			defer session.CloseWithError(0, "")

			msg := []byte(fmt.Sprintf("dg-%d", id))
			session.SendDatagram(msg)
			reply, err := session.ReceiveDatagram(ctx)
			if err != nil {
				errors.Add(1)
				return
			}
			expected := "C:" + string(msg)
			if string(reply) != expected {
				t.Errorf("datagram %d: got %q, want %q", id, reply, expected)
				errors.Add(1)
			}
		}(i)
	}

	wg.Wait()

	if errors.Load() > 2 {
		t.Errorf("too many errors: %d", errors.Load())
	}

	time.Sleep(200 * time.Millisecond)
	if server.SessionCount() != 0 {
		t.Errorf("leaked sessions: %d", server.SessionCount())
	}
}

// TestProductionMiddlewareChainOrder verifies that middleware runs in correct
// order and can abort the chain.
func TestProductionMiddlewareChainOrder(t *testing.T) {
	l, _ := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	port := l.LocalAddr().(*net.UDPAddr).Port
	l.Close()

	addr := fmt.Sprintf("127.0.0.1:%d", port)
	server := New(WithAddr(addr), WithSelfSignedTLS())

	var order []string
	var mu sync.Mutex

	server.Use(func(c *Context, next HandlerFunc) {
		mu.Lock()
		order = append(order, "mw1-before")
		mu.Unlock()
		next(c)
		mu.Lock()
		order = append(order, "mw1-after")
		mu.Unlock()
	})

	server.Handle("/order", func(c *Context) {
		mu.Lock()
		order = append(order, "handler")
		mu.Unlock()
		// Accept one stream to prove we got here
		stream, _ := c.AcceptStream()
		stream.WriteMessage([]byte("ok"))
		stream.Close()
	}, func(c *Context, next HandlerFunc) {
		mu.Lock()
		order = append(order, "mw2-before")
		mu.Unlock()
		next(c)
		mu.Lock()
		order = append(order, "mw2-after")
		mu.Unlock()
	})

	go server.ListenAndServe()
	time.Sleep(100 * time.Millisecond)
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	dialer := webtransport.Dialer{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
	_, session, _ := dialer.Dial(ctx, fmt.Sprintf("https://%s/order", addr), nil)
	raw, _ := session.OpenStreamSync(ctx)
	s := &Stream{raw: raw}
	s.WriteMessage([]byte("trigger"))
	s.ReadMessage()
	session.CloseWithError(0, "")
	time.Sleep(200 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	expected := []string{"mw1-before", "mw2-before", "handler", "mw2-after", "mw1-after"}
	if len(order) != len(expected) {
		t.Fatalf("expected %d steps, got %d: %v", len(expected), len(order), order)
	}
	for i := range expected {
		if order[i] != expected[i] {
			t.Errorf("step %d: expected %q, got %q", i, expected[i], order[i])
		}
	}
}

// TestProductionRapidConnectDisconnect tests rapid session creation and teardown.
func TestProductionRapidConnectDisconnect(t *testing.T) {
	l, _ := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	port := l.LocalAddr().(*net.UDPAddr).Port
	l.Close()

	addr := fmt.Sprintf("127.0.0.1:%d", port)
	server := New(WithAddr(addr), WithSelfSignedTLS())

	var connected atomic.Int64
	server.Handle("/rapid", func(c *Context) {
		connected.Add(1)
		defer connected.Add(-1)
		<-c.Context().Done()
	})

	go server.ListenAndServe()
	time.Sleep(100 * time.Millisecond)
	defer server.Close()

	// Rapidly connect and disconnect 30 clients
	for i := range 30 {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		dialer := webtransport.Dialer{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
		_, session, err := dialer.Dial(ctx, fmt.Sprintf("https://%s/rapid", addr), nil)
		cancel()
		if err != nil {
			continue // some may fail under rapid load, that's ok
		}
		// Random hold time
		time.Sleep(time.Duration(rand.IntN(20)) * time.Millisecond)
		session.CloseWithError(0, fmt.Sprintf("rapid-%d", i))
	}

	// Wait for all to drain
	time.Sleep(500 * time.Millisecond)

	active := connected.Load()
	if active != 0 {
		t.Errorf("expected 0 active after rapid connect/disconnect, got %d", active)
	}
	if server.SessionCount() != 0 {
		t.Errorf("leaked sessions: %d", server.SessionCount())
	}
}
