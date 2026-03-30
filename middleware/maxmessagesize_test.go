package middleware

import (
	"testing"

	"github.com/rarebek/wt"
)

func TestGetMaxMessageSizeDefault(t *testing.T) {
	// Can't create wt.Context from external package (unexported fields)
	// but we can verify the constant
	if wt.MaxMessageSize != 16*1024*1024 {
		t.Errorf("expected 16MB, got %d", wt.MaxMessageSize)
	}
}
