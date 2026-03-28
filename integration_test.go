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

// startTestServer creates and starts a server on a random port for testing.
func startTestServer(t *testing.T, setup func(*Server)) (*Server, string) {
	t.Helper()

	// Find a free port
	l, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		t.Fatal(err)
	}
	port := l.LocalAddr().(*net.UDPAddr).Port
	l.Close()

	addr := fmt.Sprintf("127.0.0.1:%d", port)
	server := New(
		WithAddr(addr),
		WithSelfSignedTLS(),
	)

	setup(server)

	errCh := make(chan error, 1)
	go func() {
		errCh <- server.ListenAndServe()
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	select {
	case err := <-errCh:
		t.Fatalf("server failed to start: %v", err)
	default:
	}

	return server, addr
}

func TestServerEchoIntegration(t *testing.T) {
	server, addr := startTestServer(t, func(s *Server) {
		s.Handle("/echo", func(c *Context) {
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
	})
	defer server.Close()

	// Connect client
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	dialer := webtransport.Dialer{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	_, session, err := dialer.Dial(ctx, fmt.Sprintf("https://%s/echo", addr), nil)
	if err != nil {
		t.Fatalf("dial error: %v", err)
	}
	defer session.CloseWithError(0, "")

	// Open stream and send message
	stream, err := session.OpenStreamSync(ctx)
	if err != nil {
		t.Fatalf("open stream error: %v", err)
	}

	testMsg := []byte("hello webtransport!")
	s := &Stream{raw: stream, ctx: nil}
	if err := s.WriteMessage(testMsg); err != nil {
		t.Fatalf("write error: %v", err)
	}

	reply, err := s.ReadMessage()
	if err != nil {
		t.Fatalf("read error: %v", err)
	}

	if string(reply) != string(testMsg) {
		t.Errorf("expected %q, got %q", testMsg, reply)
	}
}

func TestServerDatagramIntegration(t *testing.T) {
	server, addr := startTestServer(t, func(s *Server) {
		s.Handle("/dgram", func(c *Context) {
			for {
				data, err := c.ReceiveDatagram()
				if err != nil {
					return
				}
				_ = c.SendDatagram(data)
			}
		})
	})
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	dialer := webtransport.Dialer{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	_, session, err := dialer.Dial(ctx, fmt.Sprintf("https://%s/dgram", addr), nil)
	if err != nil {
		t.Fatalf("dial error: %v", err)
	}
	defer session.CloseWithError(0, "")

	testData := []byte("ping")
	if err := session.SendDatagram(testData); err != nil {
		t.Fatalf("send datagram error: %v", err)
	}

	reply, err := session.ReceiveDatagram(ctx)
	if err != nil {
		t.Fatalf("receive datagram error: %v", err)
	}

	if string(reply) != string(testData) {
		t.Errorf("expected %q, got %q", testData, reply)
	}
}

func TestServerMiddlewareIntegration(t *testing.T) {
	var middlewareOrder []string

	server, addr := startTestServer(t, func(s *Server) {
		s.Use(func(c *Context, next HandlerFunc) {
			middlewareOrder = append(middlewareOrder, "global")
			next(c)
		})

		s.Handle("/mw", func(c *Context) {
			middlewareOrder = append(middlewareOrder, "handler")
			// Accept one stream to prove we got here
			stream, err := c.AcceptStream()
			if err != nil {
				return
			}
			_ = stream.WriteMessage([]byte("ok"))
			stream.Close()
		}, func(c *Context, next HandlerFunc) {
			middlewareOrder = append(middlewareOrder, "route")
			next(c)
		})
	})
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	dialer := webtransport.Dialer{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	_, session, err := dialer.Dial(ctx, fmt.Sprintf("https://%s/mw", addr), nil)
	if err != nil {
		t.Fatalf("dial error: %v", err)
	}

	stream, err := session.OpenStreamSync(ctx)
	if err != nil {
		t.Fatalf("open stream error: %v", err)
	}

	// Write something to trigger the server's AcceptStream
	s := &Stream{raw: stream, ctx: nil}
	_ = s.WriteMessage([]byte("ping"))

	msg, err := s.ReadMessage()
	if err != nil {
		t.Fatalf("read error: %v", err)
	}

	if string(msg) != "ok" {
		t.Errorf("expected 'ok', got %q", msg)
	}

	session.CloseWithError(0, "")
	time.Sleep(200 * time.Millisecond)

	if len(middlewareOrder) != 3 {
		t.Fatalf("expected 3 middleware calls, got %d: %v", len(middlewareOrder), middlewareOrder)
	}
	if middlewareOrder[0] != "global" || middlewareOrder[1] != "route" || middlewareOrder[2] != "handler" {
		t.Errorf("wrong order: %v", middlewareOrder)
	}
}

func TestServerSessionStore(t *testing.T) {
	connected := make(chan string, 1)

	server, addr := startTestServer(t, func(s *Server) {
		s.OnConnect(func(c *Context) {
			connected <- c.ID()
		})

		s.Handle("/store", func(c *Context) {
			// Just wait for session to close
			<-c.Context().Done()
		})
	})
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	dialer := webtransport.Dialer{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	_, session, err := dialer.Dial(ctx, fmt.Sprintf("https://%s/store", addr), nil)
	if err != nil {
		t.Fatalf("dial error: %v", err)
	}

	select {
	case id := <-connected:
		if id == "" {
			t.Error("expected non-empty session ID")
		}

		if server.Sessions().Count() != 1 {
			t.Errorf("expected 1 active session, got %d", server.Sessions().Count())
		}
	case <-ctx.Done():
		t.Fatal("timeout waiting for connection")
	}

	session.CloseWithError(0, "")
	time.Sleep(200 * time.Millisecond)

	if server.Sessions().Count() != 0 {
		t.Errorf("expected 0 sessions after disconnect, got %d", server.Sessions().Count())
	}
}
