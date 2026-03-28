package wt

import "testing"

func TestConnInfoTransport(t *testing.T) {
	// ConnInfo always reports "webtransport"
	info := ConnInfo{
		Transport: "webtransport",
	}
	if info.Transport != "webtransport" {
		t.Errorf("expected 'webtransport', got %q", info.Transport)
	}
}
