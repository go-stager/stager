package stager

import (
	"net/http"
	"strconv"
)

// beginResponse is a helper to set the two most common headers and status code.
func beginResponse(w http.ResponseWriter, status int, content_type string, content_length int) {
	w.Header().Set("Content-Length", strconv.Itoa(content_length))
	w.Header().Set("Content-Type", content_type)
	w.WriteHeader(status)
}

// simpleTextResponse will send a simple text/plain response to the browser.
func simpleTextResponse(w http.ResponseWriter, status int, output string) {
	beginResponse(w, status, "text/plain; charset=utf-8", len(output))
	w.Write([]byte(output))
}
