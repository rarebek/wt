package wt

import "testing"

func TestValidateDatagramSize(t *testing.T) {
	// Valid sizes
	if err := ValidateDatagramSize([]byte("hello")); err != nil {
		t.Errorf("5 bytes should be valid: %v", err)
	}

	if err := ValidateDatagramSize(make([]byte, MaxDatagramSize)); err != nil {
		t.Errorf("%d bytes should be valid: %v", MaxDatagramSize, err)
	}

	// Empty
	if err := ValidateDatagramSize([]byte{}); err != nil {
		t.Errorf("empty should be valid: %v", err)
	}

	// Too large
	err := ValidateDatagramSize(make([]byte, MaxDatagramSize+1))
	if err == nil {
		t.Error("expected error for oversized datagram")
	}
}

func TestMaxDatagramSize(t *testing.T) {
	if MaxDatagramSize != 1200 {
		t.Errorf("expected MaxDatagramSize 1200, got %d", MaxDatagramSize)
	}
}
