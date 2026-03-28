package wt

import (
	"fmt"
	"net/http"
)

// AltSvcHeader returns the Alt-Svc HTTP header value that tells browsers
// to upgrade from HTTP/2 to HTTP/3 for WebTransport.
//
// Browsers use this header to discover that a server supports HTTP/3.
// Include it in your HTTP/1.1 or HTTP/2 responses.
//
// Usage:
//
//	// On your HTTP/1.1 or HTTP/2 server:
//	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
//	    wt.SetAltSvcHeader(w, 4433) // WebTransport on port 4433
//	    // ... serve regular HTTP
//	})
func AltSvcHeader(port int) string {
	return fmt.Sprintf(`h3=":%d"; ma=86400`, port)
}

// SetAltSvcHeader sets the Alt-Svc header on an HTTP response.
func SetAltSvcHeader(w http.ResponseWriter, port int) {
	w.Header().Set("Alt-Svc", AltSvcHeader(port))
}

// AltSvcMiddleware returns an HTTP middleware that adds the Alt-Svc header
// to every response, advertising HTTP/3 availability.
func AltSvcMiddleware(port int) func(http.Handler) http.Handler {
	header := AltSvcHeader(port)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Alt-Svc", header)
			next.ServeHTTP(w, r)
		})
	}
}
