package middleware

import (
	"net"
	"testing"
)

func TestIPBlacklist(t *testing.T) {
	bl := NewIPBlacklist("1.2.3.4", "10.0.0.0/8")

	if !bl.IsBlocked(net.ParseIP("1.2.3.4")) {
		t.Error("1.2.3.4 should be blocked")
	}
	if !bl.IsBlocked(net.ParseIP("10.0.0.1")) {
		t.Error("10.0.0.1 should be blocked (CIDR)")
	}
	if bl.IsBlocked(net.ParseIP("192.168.1.1")) {
		t.Error("192.168.1.1 should not be blocked")
	}
}

func TestIPBlacklistAddRemove(t *testing.T) {
	bl := NewIPBlacklist()

	bl.Add("5.5.5.5")
	if !bl.IsBlocked(net.ParseIP("5.5.5.5")) {
		t.Error("should be blocked after Add")
	}

	bl.Remove("5.5.5.5")
	if bl.IsBlocked(net.ParseIP("5.5.5.5")) {
		t.Error("should not be blocked after Remove")
	}
}
