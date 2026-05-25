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
