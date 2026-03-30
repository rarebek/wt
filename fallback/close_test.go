package fallback

import (
	"net/http/httptest"
	"testing"
	"time"

	"golang.org/x/net/websocket"
)

func TestWSConnCloseIdempotent(t *testing.T) {
	server := httptest.NewServer(Handler(func(conn *WSConn) {
		time.Sleep(100 * time.Millisecond)
	}))
	defer server.Close()

	wsURL := "ws" + server.URL[len("http"):]
	ws, err := websocket.Dial(wsURL, "", server.URL)
	if err != nil {
		t.Fatal(err)
	}

	conn := NewWSConn(ws)
	conn.Close()
	conn.Close() // should not panic
}

func TestWSConnSendAfterClose(t *testing.T) {
	server := httptest.NewServer(Handler(func(conn *WSConn) {
		time.Sleep(100 * time.Millisecond)
	}))
	defer server.Close()

	wsURL := "ws" + server.URL[len("http"):]
	ws, _ := websocket.Dial(wsURL, "", server.URL)

	conn := NewWSConn(ws)
	conn.Close()

	err := conn.SendDatagram([]byte("after close"))
	if err == nil {
		t.Error("expected error sending after close")
	}
}
