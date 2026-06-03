package views

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/vitlixe/netsoryn/internal/config"
)

func TestFetchHTTPDataUsesConfiguredTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cmd := fetchHTTPData(context.Background(), []config.HTTPCheck{
		{URL: srv.URL, Timeout: time.Millisecond},
	})

	msg, ok := cmd().(httpDataMsg)
	if !ok {
		t.Fatalf("fetchHTTPData returned %T, want httpDataMsg", msg)
	}
	if len(msg.results) != 1 {
		t.Fatalf("len(results) = %d, want 1", len(msg.results))
	}
	if msg.results[0].Error == "" {
		t.Fatal("result error is empty; configured timeout was not applied")
	}
}

func TestNormalizeURL(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"https://example.com", "https://example.com"},
		{"http://example.com", "http://example.com"},
		{"example.com", "https://example.com"},
		{"httpbin.org", "https://httpbin.org"}, // must not be mistaken for a scheme
		{"httpd.apache.org", "https://httpd.apache.org"},
		{"localhost:8080", "https://localhost:8080"},
	}
	for _, c := range cases {
		if got := normalizeURL(c.in); got != c.want {
			t.Errorf("normalizeURL(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
