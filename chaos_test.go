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

// TestChaosRandomDisconnects tests that the server handles random client
// disconnections gracefully without panics, leaks, or deadlocks.
func TestChaosRandomDisconnects(t *testing.T) {
	l, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		t.Fatal(err)
	}
	port := l.LocalAddr().(*net.UDPAddr).Port
	l.Close()

	addr := fmt.Sprintf("127.0.0.1:%d", port)
	server := New(WithAddr(addr), WithSelfSignedTLS())

	var serverPanics atomic.Int64
	var sessionsHandled atomic.Int64

	server.Handle("/chaos", func(c *Context) {
		defer func() {
			if r := recover(); r != nil {
				serverPanics.Add(1)
			}
		}()
		sessionsHandled.Add(1)

		// Echo streams until disconnect
		for {
			stream, err := c.AcceptStream()
			if err != nil {
				return
			}
			go func() {
				defer stream.Close()
				msg, err := stream.ReadMessage()
				if err != nil {
					return
				}
				_ = stream.WriteMessage(msg)
			}()
		}
	})

	go server.ListenAndServe()
	time.Sleep(100 * time.Millisecond)
	defer server.Close()

	const numClients = 15
	var wg sync.WaitGroup

	for i := range numClients {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()

			dialer := webtransport.Dialer{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			}
			_, session, err := dialer.Dial(ctx, fmt.Sprintf("https://%s/chaos", addr), nil)
			if err != nil {
				return // connection failure is expected in chaos
			}

			// Random behavior: some send data, some disconnect immediately
			action := rand.IntN(3)
			switch action {
			case 0:
				// Disconnect immediately
				session.CloseWithError(0, "chaos")
			case 1:
				// Send one message then disconnect
				stream, err := session.OpenStreamSync(ctx)
				if err != nil {
					session.CloseWithError(0, "")
					return
				}
				s := &Stream{raw: stream}
				s.WriteMessage([]byte("chaos"))
				time.Sleep(time.Duration(rand.IntN(50)) * time.Millisecond)
				session.CloseWithError(0, "chaos")
			case 2:
				// Stay connected briefly, send multiple messages
				for range rand.IntN(5) + 1 {
					stream, err := session.OpenStreamSync(ctx)
					if err != nil {
						break
					}
					s := &Stream{raw: stream}
					s.WriteMessage([]byte(fmt.Sprintf("chaos-%d", id)))
					s.ReadMessage() // wait for echo
				}
				time.Sleep(time.Duration(rand.IntN(100)) * time.Millisecond)
				session.CloseWithError(0, "done")
			}
		}(i)
	}

	wg.Wait()
	time.Sleep(200 * time.Millisecond)

	// Verify no panics on server side
	panics := serverPanics.Load()
	if panics > 0 {
		t.Errorf("server had %d panics during chaos test", panics)
	}

	handled := sessionsHandled.Load()
	t.Logf("chaos test: %d clients, %d sessions handled, %d panics", numClients, handled, panics)

	// Verify server cleaned up properly
	remaining := server.SessionCount()
	if remaining != 0 {
		t.Errorf("expected 0 sessions after chaos, got %d", remaining)
	}
}
