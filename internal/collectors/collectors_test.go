package collectors_test

import (
	"context"
	"testing"
	"time"

	. "github.com/vitlixe/netsoryn/internal/collectors"
)

func TestSystemCollector(t *testing.T) {
	c := NewSystemCollector()
	if c.Name() != "system" {
		t.Fatalf("expected name 'system', got %q", c.Name())
	}
	if c.Interval() <= 0 {
		t.Fatal("interval must be positive")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	raw, err := c.Collect(ctx)
	if err != nil {
		t.Fatalf("Collect error: %v", err)
	}

	data, ok := raw.(SystemData)
	if !ok {
		t.Fatalf("unexpected type %T", raw)
	}

	if data.MemTotal == 0 {
		t.Error("MemTotal should be non-zero")
	}
	if data.CPUTotal < 0 || data.CPUTotal > 100 {
		t.Errorf("CPUTotal out of range: %.2f", data.CPUTotal)
	}
}

func TestProcessCollector(t *testing.T) {
	c := NewProcessCollector()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	raw, err := c.Collect(ctx)
	if err != nil {
		t.Fatalf("Collect error: %v", err)
	}

	data, ok := raw.(ProcessData)
	if !ok {
		t.Fatalf("unexpected type %T", raw)
	}

	if len(data.Processes) == 0 {
		t.Error("expected at least one process")
	}
	for _, p := range data.Processes {
		if p.Name == "" {
			t.Error("process name should not be empty")
		}
	}
}

func TestPortCollector(t *testing.T) {
	c := NewPortCollector(true)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	raw, err := c.Collect(ctx)
	if err != nil {
		t.Fatalf("Collect error: %v", err)
	}

	_, ok := raw.(PortData)
	if !ok {
		t.Fatalf("unexpected type %T", raw)
	}
}

func TestDNSResolveOnce(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result := ResolveOnce(ctx, "dns.google", []string{"8.8.8.8:53"})
	if result.Error != "" {
		t.Skipf("DNS not available in this environment: %s", result.Error)
	}
	if len(result.ARecords) == 0 {
		t.Error("expected at least one A record for dns.google")
	}
}

func TestHTTPCheckOnce(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	result := CheckOnce(ctx, "https://dns.google", 10*time.Second)
	if result.Error != "" {
		t.Skipf("HTTP not available in this environment: %s", result.Error)
	}
	if result.StatusCode != 200 {
		t.Errorf("expected 200, got %d", result.StatusCode)
	}
	if !result.TLSValid {
		t.Error("TLS should be valid for dns.google")
	}
}

func TestFormatElapsed(t *testing.T) {
	cases := []struct {
		d    time.Duration
		want string
	}{
		{500 * time.Microsecond, "500µs"},
		{42 * time.Millisecond, "42ms"},
		{1500 * time.Millisecond, "1.50s"},
	}
	for _, c := range cases {
		got := FormatElapsed(c.d)
		if got != c.want {
			t.Errorf("FormatElapsed(%v) = %q, want %q", c.d, got, c.want)
		}
	}
}

func TestStatusColor(t *testing.T) {
	if StatusColor(200) != "ok" {
		t.Error("200 should be ok")
	}
	if StatusColor(404) != "warn" {
		t.Error("404 should be warn")
	}
	if StatusColor(500) != "error" {
		t.Error("500 should be error")
	}
	if StatusColor(301) != "redirect" {
		t.Error("301 should be redirect")
	}
}
