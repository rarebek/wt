package wt

import "testing"

func TestMaxMessageSizeConst(t *testing.T) {
	if MaxMessageSize != 16*1024*1024 {
		t.Errorf("expected 16MB, got %d", MaxMessageSize)
	}
}

func TestMaxDatagramSizeConst(t *testing.T) {
	if MaxDatagramSize != 1200 {
		t.Errorf("expected 1200, got %d", MaxDatagramSize)
	}
}
