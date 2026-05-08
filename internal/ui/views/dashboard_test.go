package views

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"

	"github.com/vitlixe/netsoryn/internal/collectors"
	"github.com/vitlixe/netsoryn/internal/config"
)

// ── dashGap ──────────────────────────────────────────────────────────────────

func TestDashGap(t *testing.T) {
	tests := []struct {
		termW    int
		wantGap  int
		wantColW int
	}{
		{100, 4, 48},
		// innerW at min terminal (105): 105-2=103 → colW=(103-4)/2=49
		{103, 4, 49},
		{129, 4, 62},
		{130, 6, 62},
		{159, 6, 76},
		{160, 8, 76},
		{200, 8, 96},
	}
	for _, tt := range tests {
		g := dashGap(tt.termW)
		colW := (tt.termW - g) / 2
		if g != tt.wantGap {
			t.Errorf("dashGap(%d) = %d, want %d", tt.termW, g, tt.wantGap)
		}
		if colW != tt.wantColW {
			t.Errorf("dashGap(%d): colW = %d, want %d", tt.termW, colW, tt.wantColW)
		}
	}
}

// ── dashSectionLayout ────────────────────────────────────────────────────────

func TestDashSectionLayout(t *testing.T) {
	tests := []struct {
		colW, baseLabelW, sizeW, minBar, maxBar int
		wantLabelW, wantBarW                    int
	}{
		// CPU at minimum colW=49 (innerW=103 at termW=105, gap=4): (103-4)/2=49
		{49, 8, 0, 10, 24, 8, 24}, // bar = 49-19=30 → capped 24
		// Memory at minimum colW=49 (sizeW=17)
		{49, 8, 17, 10, 24, 8, 11}, // bar = 49-38=11
		// Disk at minimum colW=49 (sizeW=15, minBar=4)
		{49, 12, 15, 4, 20, 12, 9}, // bar = 49-40=9
		// Disk worst-case size (colW=49, sizeW=19, minBar=4)
		{49, 12, 19, 4, 20, 12, 5}, // bar = 49-44=5
		// CPU wider terminal (colW=74, extra=(74-50)/16=1)
		{74, 8, 0, 10, 24, 9, 24}, // labelW=9, bar capped at 24
		// Disk wider (colW=74, sizeW=15, minBar=4)
		{74, 12, 15, 4, 20, 13, 20}, // labelW=13, bar=74-41=33 → capped 20
		// Very wide: extra capped at 6
		{200, 8, 0, 10, 24, 14, 24}, // extra=(200-50)/16=9 → capped 6 → labelW=14
	}
	for _, tt := range tests {
		gotL, gotB := dashSectionLayout(tt.colW, tt.baseLabelW, tt.sizeW, tt.minBar, tt.maxBar)
		if gotL != tt.wantLabelW || gotB != tt.wantBarW {
			t.Errorf("dashSectionLayout(%d,%d,%d,%d,%d) = labelW=%d barW=%d, want %d %d",
				tt.colW, tt.baseLabelW, tt.sizeW, tt.minBar, tt.maxBar,
				gotL, gotB, tt.wantLabelW, tt.wantBarW)
		}
	}
}

// ── dashboardBar ─────────────────────────────────────────────────────────────

func TestDashboardBar_Width(t *testing.T) {
	for _, pct := range []float64{0, 25, 50, 75, 100} {
		for _, w := range []int{6, 10, 14, 20, 24} {
			bar := dashboardBar(pct, w)
			got := lipgloss.Width(bar)
			if got != w {
				t.Errorf("dashboardBar(%.0f%%, %d) visual width = %d, want %d", pct, w, got, w)
			}
		}
	}
}

// ── padRight ─────────────────────────────────────────────────────────────────

func TestPadRight(t *testing.T) {
	tests := []struct {
		s     string
		width int
		want  int // expected visual width of result
	}{
		{"Data", 12, 12},
		{"…/Data", 12, 12}, // "…" is 1 visual col, 3 bytes
		{"Preboot", 12, 12},
		{"longenough12", 12, 12},
		{"toolongstring", 12, 13}, // already wider: not truncated
		{"/", 12, 12},
	}
	for _, tt := range tests {
		got := lipgloss.Width(padRight(tt.s, tt.width))
		if got != tt.want {
			t.Errorf("padRight(%q, %d) visual width = %d, want %d", tt.s, tt.width, got, tt.want)
		}
	}
}

// ── diskMountLabel ───────────────────────────────────────────────────────────

func TestDiskMountLabel(t *testing.T) {
	tests := []struct {
		mountpoint string
		labelW     int
		want       string
	}{
		{"/", 14, "/"},
		{"/boot", 14, "/boot"},
		{"/home", 12, "/home"},
		{"/System/Volumes/Data", 12, "Data"},
		{"/System/Volumes/Preboot", 12, "Preboot"},
		{"/System/Volumes/VM", 12, "VM"},
		{"/System/Volumes/Update/mnt1", 12, "Update/mnt1"},
		{"/private/var/folders", 12, "…/folders"},
		{"/very/long/nested/path", 12, "…/path"},
	}
	for _, tt := range tests {
		got := diskMountLabel(tt.mountpoint, tt.labelW)
		if got != tt.want {
			t.Errorf("diskMountLabel(%q, %d) = %q, want %q", tt.mountpoint, tt.labelW, got, tt.want)
		}
	}
}

func TestDiskMountLabel_DistinctApfsVolumes(t *testing.T) {
	const labelW = 12
	mounts := []string{
		"/System/Volumes/Data",
		"/System/Volumes/Preboot",
		"/System/Volumes/VM",
		"/System/Volumes/Update",
	}
	seen := make(map[string]string)
	for _, m := range mounts {
		l := diskMountLabel(m, labelW)
		if prev, ok := seen[l]; ok {
			t.Errorf("label collision: %q and %q both produce %q", prev, m, l)
		}
		seen[l] = m
	}
}

// ── render helpers ────────────────────────────────────────────────────────────

func newTestDashboard() *Dashboard {
	return &Dashboard{
		cfg:    &config.Config{},
		loaded: true,
		data: collectors.SystemData{
			CPUTotal:    45.5,
			CPUPercents: []float64{60.0, 30.0},
			MemPercent:  58.9,
			MemUsed:     10_066_329_600, // ~9.4 GB
			MemTotal:    17_179_869_184, // 16.0 GB
			SwapPercent: 0.0,
			SwapUsed:    0,
			SwapTotal:   0,
			Disks: []collectors.DiskStat{
				// /System/Volumes/Data — sizeStr "1.5 TB / 4.0 TB" = 15 chars
				{Mountpoint: "/System/Volumes/Data", Used: 1_649_267_441_664, Total: 4_398_046_511_104, UsedPercent: 37.5},
				// /System/Volumes/Preboot — sizeStr "205.0 KB / 500.0 MB" = 19 chars (worst-case width)
				{Mountpoint: "/System/Volumes/Preboot", Used: 209_920, Total: 524_288_000, UsedPercent: 0.04},
				// Root — sizeStr "300.0 GB / 500.0 GB" = 19 chars
				{Mountpoint: "/", Used: 322_122_547_200, Total: 536_870_912_000, UsedPercent: 60.0},
			},
		},
	}
}

// minColW is colW at the minimum supported terminal width (105 cols):
// innerW = 105 - 2 = 103; colW = (103 - dashGap(103)) / 2 = (103 - 4) / 2 = 49.
const minColW = 49

// ── no-wrap tests at colW=50 ─────────────────────────────────────────────────

func checkNoWrap(t *testing.T, section, out string, limit int) {
	t.Helper()
	for _, line := range strings.Split(out, "\n") {
		if vw := lipgloss.Width(line); vw > limit {
			t.Errorf("%s: line visual width %d > %d: %q", section, vw, limit, line)
		}
	}
}

func TestRenderCPU_NoWrap(t *testing.T) {
	checkNoWrap(t, "renderCPU(49)", newTestDashboard().renderCPU(minColW), minColW)
}

func TestRenderMemory_NoWrap(t *testing.T) {
	checkNoWrap(t, "renderMemory(49)", newTestDashboard().renderMemory(minColW), minColW)
}

func TestRenderDisks_NoWrap(t *testing.T) {
	checkNoWrap(t, "renderDisks(49)", newTestDashboard().renderDisks(minColW), minColW)
}

// ── no-wrap tests at colW=72 (wider terminal ~150 cols) ──────────────────────

func TestRenderCPU_NoWrap_Wide(t *testing.T) {
	const w = 72
	checkNoWrap(t, "renderCPU(72)", newTestDashboard().renderCPU(w), w)
}

func TestRenderMemory_NoWrap_Wide(t *testing.T) {
	const w = 72
	checkNoWrap(t, "renderMemory(72)", newTestDashboard().renderMemory(w), w)
}

func TestRenderDisks_NoWrap_Wide(t *testing.T) {
	const w = 72
	checkNoWrap(t, "renderDisks(72)", newTestDashboard().renderDisks(w), w)
}

// ── disk column alignment ─────────────────────────────────────────────────────

// TestRenderDisks_UniformLineWidth verifies that all data rows in the disk
// section have the same visual width (bar and percent start at fixed columns).
func TestRenderDisks_UniformLineWidth(t *testing.T) {
	d := newTestDashboard()
	out := d.renderDisks(minColW)
	lines := strings.Split(out, "\n")
	// Skip the title line (index 0).
	if len(lines) < 2 {
		t.Fatal("renderDisks: expected at least 2 lines")
	}
	firstW := lipgloss.Width(lines[1])
	for i, line := range lines[1:] {
		if vw := lipgloss.Width(line); vw != firstW {
			t.Errorf("disk row %d visual width = %d, want %d (all rows must match)", i+1, vw, firstW)
		}
	}
}

// TestRenderDisks_DistinctLabels verifies that APFS volumes produce different labels.
func TestRenderDisks_DistinctLabels(t *testing.T) {
	d := newTestDashboard()
	out := d.renderDisks(minColW)
	if strings.Count(out, "Data") == 0 {
		t.Error("expected 'Data' label in disk output")
	}
	if strings.Count(out, "Preboot") == 0 {
		t.Error("expected 'Preboot' label in disk output")
	}
}
