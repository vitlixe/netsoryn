package views

import (
	"strings"
	"testing"

	"github.com/vitlixe/netsoryn/internal/collectors"
	"github.com/vitlixe/netsoryn/internal/ui/styles"
)

// ── serviceDisplayStatus ─────────────────────────────────────────────────────

func TestServiceDisplayStatus(t *testing.T) {
	tests := []struct {
		name string
		svc  collectors.ServiceStat
		want string
	}{
		{
			name: "SubState present",
			svc:  collectors.ServiceStat{SubState: "running", ActiveState: "active", LoadState: "loaded"},
			want: "running",
		},
		{
			name: "SubState empty, fall back to ActiveState",
			svc:  collectors.ServiceStat{SubState: "", ActiveState: "active", LoadState: "loaded"},
			want: "active",
		},
		{
			name: "SubState and ActiveState empty, fall back to LoadState",
			svc:  collectors.ServiceStat{SubState: "", ActiveState: "", LoadState: "loaded"},
			want: "loaded",
		},
		{
			name: "all empty → unknown",
			svc:  collectors.ServiceStat{},
			want: "unknown",
		},
	}
	for _, tt := range tests {
		got := serviceDisplayStatus(tt.svc)
		if got != tt.want {
			t.Errorf("%s: serviceDisplayStatus() = %q, want %q", tt.name, got, tt.want)
		}
	}
}

// ── serviceStatusLabel ────────────────────────────────────────────────────────

func TestServiceStatusLabel(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"running", "running"},
		{"active", "running"},
		{"stopped", "stopped"},
		{"inactive", "stopped"},
		{"dead", "stopped"},
		{"failed", "failed"},
		{"", "unknown"},
		{"unknown", "unknown"},
		{"waiting", "waiting"}, // pass-through for unrecognised states
	}
	for _, tt := range tests {
		got := serviceStatusLabel(tt.in)
		if got != tt.want {
			t.Errorf("serviceStatusLabel(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

// serviceStatusLabel must never return an empty string.
func TestServiceStatusLabel_NeverEmpty(t *testing.T) {
	for _, in := range []string{"", "unknown", "running", "dead", "whatever"} {
		if got := serviceStatusLabel(in); got == "" {
			t.Errorf("serviceStatusLabel(%q) returned empty string", in)
		}
	}
}

// ── table cell content (no ANSI) ─────────────────────────────────────────────

func TestServiceStatusLabel_NoANSI(t *testing.T) {
	for _, in := range []string{"running", "stopped", "failed", "", "unknown", "waiting"} {
		got := serviceStatusLabel(in)
		if strings.ContainsRune(got, '\x1b') {
			t.Errorf("serviceStatusLabel(%q) contains ANSI escape sequences: %q", in, got)
		}
	}
}

// ── svcColumns ────────────────────────────────────────────────────────────────

func TestSvcColumns_Darwin(t *testing.T) {
	cols := svcColumns(103, "darwin")
	if len(cols) != 2 {
		t.Fatalf("darwin: want 2 columns, got %d", len(cols))
	}
	if cols[1].Title != "State" {
		t.Errorf("darwin col[1] title = %q, want \"State\"", cols[1].Title)
	}
}

func TestSvcColumns_Linux(t *testing.T) {
	cols := svcColumns(103, "linux")
	if len(cols) != 4 {
		t.Fatalf("linux: want 4 columns, got %d", len(cols))
	}
	titles := []string{"Service", "Status", "Active", "Description"}
	for i, want := range titles {
		if cols[i].Title != want {
			t.Errorf("linux col[%d] title = %q, want %q", i, cols[i].Title, want)
		}
	}
}

// ── styles.ServiceBadge edge cases ───────────────────────────────────────────

func TestServiceBadge_EmptyAndUnknown(t *testing.T) {
	for _, state := range []string{"", "unknown"} {
		badge := styles.ServiceBadge(state)
		if !strings.Contains(badge, "unknown") {
			t.Errorf("ServiceBadge(%q) = %q, want it to contain \"unknown\"", state, badge)
		}
	}
}

func TestServiceBadge_KnownStates(t *testing.T) {
	tests := []struct {
		state       string
		wantContain string
	}{
		{"running", "running"},
		{"active", "active"},
		{"failed", "failed"},
		{"inactive", "inactive"},
		{"stopped", "stopped"},
	}
	for _, tt := range tests {
		badge := styles.ServiceBadge(tt.state)
		if !strings.Contains(badge, tt.wantContain) {
			t.Errorf("ServiceBadge(%q) = %q, want it to contain %q", tt.state, badge, tt.wantContain)
		}
	}
}
