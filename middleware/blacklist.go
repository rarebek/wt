package middleware

import (
	"net"
	"sync"

	"github.com/rarebek/wt"
)

// IPBlacklist returns middleware that blocks connections from specific IPs or CIDRs.
// The blacklist can be updated at runtime via Add/Remove methods.
type IPBlacklist struct {
	mu   sync.RWMutex
	ips  map[string]bool
	nets []*net.IPNet
}

// NewIPBlacklist creates a new runtime-updatable IP blacklist.
func NewIPBlacklist(initial ...string) *IPBlacklist {
	bl := &IPBlacklist{
		ips: make(map[string]bool),
	}
	for _, s := range initial {
		bl.Add(s)
	}
	return bl
}

// Add adds an IP or CIDR to the blacklist.
func (bl *IPBlacklist) Add(ipOrCIDR string) {
	bl.mu.Lock()
	defer bl.mu.Unlock()

	if _, ipnet, err := net.ParseCIDR(ipOrCIDR); err == nil {
		bl.nets = append(bl.nets, ipnet)
	} else if ip := net.ParseIP(ipOrCIDR); ip != nil {
		bl.ips[ip.String()] = true
	}
}

// Remove removes an IP from the blacklist (does not support removing CIDRs).
func (bl *IPBlacklist) Remove(ip string) {
	bl.mu.Lock()
	delete(bl.ips, ip)
	bl.mu.Unlock()
}

// IsBlocked checks if an IP is blacklisted.
func (bl *IPBlacklist) IsBlocked(ip net.IP) bool {
	bl.mu.RLock()
	defer bl.mu.RUnlock()

	if bl.ips[ip.String()] {
		return true
	}
	for _, ipnet := range bl.nets {
		if ipnet.Contains(ip) {
			return true
		}
	}
	return false
}

// Middleware returns wt middleware that blocks blacklisted IPs.
func (bl *IPBlacklist) Middleware() wt.MiddlewareFunc {
	return func(c *wt.Context, next wt.HandlerFunc) {
		ip := extractIP(c.RemoteAddr().String())
		if ip != nil && bl.IsBlocked(ip) {
			_ = c.CloseWithError(403, "ip blacklisted")
			return
		}
		next(c)
	}
}
