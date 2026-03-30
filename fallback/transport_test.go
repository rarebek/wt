package fallback

import "testing"

func TestTransportConstants(t *testing.T) {
	if TransportWebTransport != 0 {
		t.Error("WebTransport should be 0")
	}
	if TransportWebSocket != 1 {
		t.Error("WebSocket should be 1")
	}
}

func TestFrameConstants(t *testing.T) {
	if frameStream != 0x01 {
		t.Error("frameStream should be 0x01")
	}
	if frameDatagram != 0x02 {
		t.Error("frameDatagram should be 0x02")
	}
	if frameOpen != 0x03 {
		t.Error("frameOpen should be 0x03")
	}
	if frameClose != 0x04 {
		t.Error("frameClose should be 0x04")
	}
}
