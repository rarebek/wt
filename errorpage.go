package wt

import (
	"encoding/json"
	"net/http"
)

// ErrorResponse is the JSON body returned when WebTransport upgrade fails.
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    int    `json:"code"`
	Message string `json:"message,omitempty"`
}

// ErrorPageHandler is called when a non-WebTransport HTTP request hits a
// WebTransport route. Customize this to return helpful error messages.
type ErrorPageHandler func(w http.ResponseWriter, r *http.Request, code int, msg string)

// DefaultErrorPage returns a JSON error response.
func DefaultErrorPage(w http.ResponseWriter, _ *http.Request, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(ErrorResponse{
		Error:   http.StatusText(code),
		Code:    code,
		Message: msg,
	})
}

// HTMLErrorPage returns an HTML error page.
func HTMLErrorPage(w http.ResponseWriter, _ *http.Request, code int, msg string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(code)
	w.Write([]byte(`<!DOCTYPE html><html><head><title>Error</title>
<style>body{font-family:monospace;background:#0d1117;color:#c9d1d9;display:flex;justify-content:center;align-items:center;height:100vh}
.box{text-align:center;}.code{font-size:48px;color:#f85149}.msg{margin-top:8px;color:#8b949e}</style>
</head><body><div class="box"><div class="code">` + http.StatusText(code) + `</div><div class="msg">` + msg + `</div></div></body></html>`))
}
