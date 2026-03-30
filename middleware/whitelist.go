package middleware

import (
	"net"
	"strings"

	"github.com/rarebek/wt"
)

// IPWhitelist returns middleware that only allows connections from the given IPs or CIDRs.
// Useful for admin endpoints or internal services.
//
// Usage:
//
//	server.Handle("/admin", handler, middleware.IPWhitelist("10.0.0.0/8", "192.168.1.0/24", "127.0.0.1"))
func IPWhitelist(allowed ...string) wt.MiddlewareFunc {
	var nets []*net.IPNet
	var ips []net.IP

	for _, a := range allowed {
		if strings.Contains(a, "/") {
			_, ipnet, err := net.ParseCIDR(a)
			if err == nil {
				nets = append(nets, ipnet)
			}
		} else {
			ip := net.ParseIP(a)
			if ip != nil {
				ips = append(ips, ip)
			}
		}
	}

	return func(c *wt.Context, next wt.HandlerFunc) {
		remoteIP := extractIP(c.RemoteAddr().String())
		if remoteIP == nil {
			_ = c.CloseWithError(403, "forbidden")
			return
		}

		for _, ip := range ips {
			if ip.Equal(remoteIP) {
				next(c)
				return
			}
		}
		for _, ipnet := range nets {
			if ipnet.Contains(remoteIP) {
				next(c)
				return
			}
		}

		_ = c.CloseWithError(403, "ip not allowed")
	}
}

func extractIP(addr string) net.IP {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return net.ParseIP(addr)
	}
	return net.ParseIP(host)
}
