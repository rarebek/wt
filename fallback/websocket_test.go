package fallback

import (
	"net/http/httptest"
	"testing"
	"time"

	"golang.org/x/net/websocket"
)

func TestTransportString(t *testing.T) {
	if TransportWebTransport.String() != "webtransport" {
		t.Errorf("expected 'webtransport', got %q", TransportWebTransport.String())
	}
	if TransportWebSocket.String() != "websocket" {
		t.Errorf("expected 'websocket', got %q", TransportWebSocket.String())
	}
}

func TestWSConnEchoStreams(t *testing.T) {
	// Server: accept streams and echo back
	serverReady := make(chan struct{})
	server := httptest.NewServer(Handler(func(conn *WSConn) {
		close(serverReady)
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
			}()
		}
	}))
	defer server.Close()

	// Client
	wsURL := "ws" + server.URL[len("http"):]
	ws, err := websocket.Dial(wsURL, "", server.URL)
	if err != nil {
		t.Fatalf("dial error: %v", err)
	}
	defer ws.Close()

	client := NewWSConn(ws)
	defer client.Close()

	<-serverReady
	time.Sleep(50 * time.Millisecond)

	// Open stream and send data
	stream, err := client.OpenStream()
	if err != nil {
		t.Fatalf("open stream error: %v", err)
	}

	testData := []byte("hello websocket fallback!")
	_, err = stream.Write(testData)
	if err != nil {
		t.Fatalf("write error: %v", err)
	}

	// Read echo response
	buf := make([]byte, 1024)
	n, err := stream.Read(buf)
	if err != nil {
		t.Fatalf("read error: %v", err)
	}

	if string(buf[:n]) != string(testData) {
		t.Errorf("expected %q, got %q", testData, buf[:n])
	}
}

func TestWSConnDatagrams(t *testing.T) {
	serverGot := make(chan []byte, 1)

	server := httptest.NewServer(Handler(func(conn *WSConn) {
		data, err := conn.ReceiveDatagram()
		if err != nil {
			return
		}
		serverGot <- data
		// Echo back
		conn.SendDatagram(data)
	}))
	defer server.Close()

	wsURL := "ws" + server.URL[len("http"):]
	ws, err := websocket.Dial(wsURL, "", server.URL)
	if err != nil {
		t.Fatalf("dial error: %v", err)
	}
	defer ws.Close()

	client := NewWSConn(ws)
	defer client.Close()

	time.Sleep(50 * time.Millisecond)

	// Send datagram
	testData := []byte("ping datagram")
	if err := client.SendDatagram(testData); err != nil {
		t.Fatalf("send datagram error: %v", err)
	}

	// Server should have received it
	select {
	case got := <-serverGot:
		if string(got) != string(testData) {
			t.Errorf("server got %q, want %q", got, testData)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for server to receive datagram")
	}

	// Client should get echo
	reply, err := client.ReceiveDatagram()
	if err != nil {
		t.Fatalf("receive datagram error: %v", err)
	}
	if string(reply) != string(testData) {
		t.Errorf("expected %q, got %q", testData, reply)
	}
}

func TestWSConnMultipleStreams(t *testing.T) {
	server := httptest.NewServer(Handler(func(conn *WSConn) {
		for {
			stream, err := conn.AcceptStream()
			if err != nil {
				return
			}
			go func() {
				defer stream.Close()
				buf := make([]byte, 1024)
				n, _ := stream.Read(buf)
				// Prefix response with stream ID
				resp := append([]byte{byte(stream.ID())}, buf[:n]...)
				stream.Write(resp)
			}()
		}
	}))
	defer server.Close()

	wsURL := "ws" + server.URL[len("http"):]
	ws, err := websocket.Dial(wsURL, "", server.URL)
	if err != nil {
		t.Fatalf("dial error: %v", err)
	}
	defer ws.Close()

	client := NewWSConn(ws)
	defer client.Close()

	time.Sleep(50 * time.Millisecond)

	// Open 3 streams concurrently
	for i := range 3 {
		stream, err := client.OpenStream()
		if err != nil {
			t.Fatalf("open stream %d error: %v", i, err)
		}

		msg := []byte{byte(i + 65)} // A, B, C
		stream.Write(msg)

		buf := make([]byte, 1024)
		n, err := stream.Read(buf)
		if err != nil {
			t.Fatalf("read stream %d error: %v", i, err)
		}

		// Response should contain the data we sent
		if n < 2 || buf[1] != byte(i+65) {
			t.Errorf("stream %d: unexpected response %v", i, buf[:n])
		}
	}
}
