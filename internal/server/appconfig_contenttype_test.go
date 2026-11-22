package server

import (
	"net/http/httptest"
	"testing"
)

func TestWriteAppConfigContentNeverEchoesHTML(t *testing.T) {
	for in, want := range map[string]string{
		"application/json; charset=utf-8": "application/json",
		"TEXT/PLAIN":                      "text/plain",
		"text/html":                       "application/octet-stream",
		"image/svg+xml":                   "application/octet-stream",
		"":                                "application/octet-stream",
	} {
		w := httptest.NewRecorder()
		writeAppConfigContent(w, 200, in, []byte("x"))
		if got := w.Header().Get("Content-Type"); got != want {
			t.Errorf("%q: Content-Type = %q, want %q", in, got, want)
		}
		if w.Header().Get("X-Content-Type-Options") != "nosniff" {
			t.Errorf("%q: missing nosniff", in)
		}
	}
}
