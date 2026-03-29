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

func TestReadWriteJSON(t *testing.T) {
	l, _ := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	port := l.LocalAddr().(*net.UDPAddr).Port
	l.Close()

	addr := fmt.Sprintf("127.0.0.1:%d", port)
	server := New(WithAddr(addr), WithSelfSignedTLS())

	type Msg struct {
		Name string `json:"name"`
		Val  int    `json:"val"`
	}

	server.Handle("/json", HandleStream(func(s *Stream, c *Context) {
		defer s.Close()
		var msg Msg
		if err := s.ReadJSON(&msg); err != nil {
			return
		}
		msg.Val *= 2 // double it
		s.WriteJSON(msg)
	}))

	go server.ListenAndServe()
	time.Sleep(100 * time.Millisecond)
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	dialer := webtransport.Dialer{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
	_, session, _ := dialer.Dial(ctx, fmt.Sprintf("https://%s/json", addr), nil)
	defer session.CloseWithError(0, "")

	raw, _ := session.OpenStreamSync(ctx)
	s := &Stream{raw: raw}

	s.WriteJSON(Msg{Name: "test", Val: 21})

	var reply Msg
	s.ReadJSON(&reply)

	if reply.Name != "test" || reply.Val != 42 {
		t.Errorf("expected {test 42}, got %+v", reply)
	}
}
