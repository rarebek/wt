package fallback

import (
	"fmt"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/net/websocket"
)

func TestWSConnConcurrentStreams(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stress test in short mode")
	}

	var processed atomic.Int64

	server := httptest.NewServer(Handler(func(conn *WSConn) {
		for {
			stream, err := conn.AcceptStream()
			if err != nil {
				return
			}
			go func() {
				defer stream.Close()
				buf := make([]byte, 1024)
				n, err := stream.Read(buf)
				if err != nil {
					return
				}
				stream.Write(buf[:n])
				processed.Add(1)
			}()
		}
	}))
	defer server.Close()

	wsURL := "ws" + server.URL[len("http"):]
	ws, err := websocket.Dial(wsURL, "", server.URL)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer ws.Close()

	client := NewWSConn(ws)
	defer client.Close()

	time.Sleep(50 * time.Millisecond)

	const numStreams = 5
	var wg sync.WaitGroup
	errors := make(chan error, numStreams)

	for i := range numStreams {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			stream, err := client.OpenStream()
			if err != nil {
				errors <- fmt.Errorf("stream %d open: %w", id, err)
				return
			}
			defer stream.Close()

			msg := []byte(fmt.Sprintf("msg-%d", id))
			if _, err := stream.Write(msg); err != nil {
				errors <- fmt.Errorf("stream %d write: %w", id, err)
				return
			}

			done := make(chan struct{})
			go func() {
				buf := make([]byte, 1024)
				n, err := stream.Read(buf)
				if err != nil {
					errors <- fmt.Errorf("stream %d read: %w", id, err)
					close(done)
					return
				}
				if string(buf[:n]) != string(msg) {
					errors <- fmt.Errorf("stream %d: got %q, want %q", id, buf[:n], msg)
				}
				close(done)
			}()

			select {
			case <-done:
			case <-time.After(5 * time.Second):
				errors <- fmt.Errorf("stream %d: read timed out", id)
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Error(err)
	}

	time.Sleep(100 * time.Millisecond)
	got := processed.Load()
	if got < numStreams {
		t.Errorf("server processed %d streams, expected %d", got, numStreams)
	}
}

func TestWSConnDatagramFlood(t *testing.T) {
	var serverReceived atomic.Int64

	server := httptest.NewServer(Handler(func(conn *WSConn) {
		for {
			_, err := conn.ReceiveDatagram()
			if err != nil {
				return
			}
			serverReceived.Add(1)
		}
	}))
	defer server.Close()

	wsURL := "ws" + server.URL[len("http"):]
	ws, err := websocket.Dial(wsURL, "", server.URL)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer ws.Close()

	client := NewWSConn(ws)
	defer client.Close()

	time.Sleep(50 * time.Millisecond)

	// Send 100 datagrams as fast as possible
	const count = 100
	for i := range count {
		client.SendDatagram([]byte(fmt.Sprintf("d-%d", i)))
	}

	time.Sleep(200 * time.Millisecond)

	got := serverReceived.Load()
	t.Logf("sent %d datagrams, server received %d", count, got)

	// At least most should arrive (WebSocket is reliable, unlike real datagrams)
	if got < count/2 {
		t.Errorf("too few datagrams received: %d/%d", got, count)
	}
}
