package middleware

import (
	"net"

	"github.com/rarebek/wt"
)

// GeoHint stores a geographic hint based on the client IP.
// This is NOT a full GeoIP lookup — it just stores hints that
// the application can populate from an external GeoIP service.
// Provides a standard context key for geo data.
type GeoInfo struct {
	Country   string `json:"country,omitempty"`
	Region    string `json:"region,omitempty"`
	City      string `json:"city,omitempty"`
	Latitude  float64 `json:"lat,omitempty"`
	Longitude float64 `json:"lon,omitempty"`
}

// GeoLookup is a function that resolves an IP to geographic info.
type GeoLookup func(ip net.IP) *GeoInfo

// Geo returns middleware that performs geo lookup on the client IP
// and stores the result in context as "geo".
func Geo(lookup GeoLookup) wt.MiddlewareFunc {
	return func(c *wt.Context, next wt.HandlerFunc) {
		ip := extractIP(c.RemoteAddr().String())
		if ip != nil && lookup != nil {
			info := lookup(ip)
			if info != nil {
				c.Set("geo", info)
			}
		}
		next(c)
	}
}

// GetGeo retrieves geographic info from context.
func GetGeo(c *wt.Context) *GeoInfo {
	v, ok := c.Get("geo")
	if !ok {
		return nil
	}
	gi, _ := v.(*GeoInfo)
	return gi
}
