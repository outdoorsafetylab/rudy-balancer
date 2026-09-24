package middleware

import (
	"net/http"
	"testing"
)

func TestRedact(t *testing.T) {
	h := http.Header{}
	h.Set("X-Forwarded-For", "203.0.113.7")
	h.Set("Forwarded", "for=203.0.113.7")
	h.Set("X-Real-Ip", "203.0.113.7")
	h.Set("User-Agent", "wadi/7.03")
	out := redact(h)
	for _, k := range []string{"X-Forwarded-For", "Forwarded", "X-Real-Ip"} {
		if out.Get(k) != "" {
			t.Errorf("%s not redacted", k)
		}
	}
	if out.Get("User-Agent") != "wadi/7.03" {
		t.Error("other headers must be kept")
	}
	if h.Get("X-Forwarded-For") == "" {
		t.Error("redact must not modify the request's own headers")
	}
}
