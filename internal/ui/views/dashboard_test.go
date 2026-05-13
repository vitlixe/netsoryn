package views

import (
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"

	"github.com/vitlixe/netsoryn/internal/collectors"
	"github.com/vitlixe/netsoryn/internal/config"
)

// ── dashboardLayout ───────────────────────────────────────────────────────────

func TestDashboardLayout(t *testing.T) {
	// threshold = cpuAnchorW(43) + layoutGap(2) + rightMin(48) + rightPad(2) = 95
	// rightW = w - cpuAnchorW - layoutGap - rightPad = w - 47
	tests := []struct {
		w        int
		wideWant bool
		leftW    int // only checked when wide
		rightW   int
		gap      int
	}{
		// Below threshold: narrow
		{88, false, 0, 0, 0},
		{94, false, 0, 0, 0},
		// At threshold: wide (rightW = 95-47 = 48 = rightMin exactly)
		{95, true, 43, 48, 2},
		// Representative widths used in production (innerW = terminalW - 2)
		{99, true, 43, 52, 2},   // terminal 101
		{100, true, 43, 53, 2},  // terminal 102
		{120, true, 43, 73, 2},  // terminal 122
		{160, true, 43, 113, 2}, // terminal 162
	}
	for _, tt := range tests {
		lay := dashboardLayout(tt.w)
		if lay.wide != tt.wideWant {
			t.Errorf("dashboardLayout(%d).wide = %v, want %v", tt.w, lay.wide, tt.wideWant)
			continue
		}
		if !lay.wide {
			continue
		}
		if lay.leftW != tt.leftW {
			t.Errorf("dashboardLayout(%d).leftW = %d, want %d", tt.w, lay.leftW, tt.leftW)
		}
		if lay.rightW != tt.rightW {
			t.Errorf("dashboardLayout(%d).rightW = %d, want %d", tt.w, lay.rightW, tt.rightW)
		}
		if lay.gap != tt.gap {
			t.Errorf("dashboardLayout(%d).gap = %d, want %d", tt.w, lay.gap, tt.gap)
		}
		if lay.rightW <= lay.leftW {
			t.Errorf("dashboardLayout(%d): rightW=%d should be > leftW=%d", tt.w, lay.rightW, lay.leftW)
		}
	}
}

// ── dashSectionLayout ────────────────────────────────────────────────────────

func TestDashSectionLayout(t *testing.T) {
	tests := []struct {
		colW, baseLabelW, sizeW, minBar, maxBar int
		wantLabelW, wantBarW                    int
	}{
		// CPU at wide breakpoint colW=48 (innerW=100, gap=4): (100-4)/2=48
		{48, 8, 0, 10, 24, 8, 24}, // bar = 48-19=29 → capped 24
		// Memory at wide breakpoint colW=48 (sizeW=17)
		{48, 8, 17, 10, 24, 8, 10}, // bar = 48-38=10
		// Disk at wide breakpoint colW=48 (sizeW=15, minBar=4)
		{48, 12, 15, 4, 20, 12, 8}, // bar = 48-40=8
		// Disk worst-case size (colW=48, sizeW=19, minBar=4)
		{48, 12, 19, 4, 20, 12, 4}, // bar = 48-44=4
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
			CPUTotal:         45.5,
			CPUPercents:      []float64{60.0, 30.0},
			MemPercent:       58.9,
			MemUsed:          10_066_329_600, // ~9.4 GB
			MemTotal:         17_179_869_184, // 16.0 GB
			SwapPercent:      0.0,
			SwapUsed:         0,
			SwapTotal:        0,
			LoadAvgSupported: true,
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

// minNarrowW is a test inner width below the wide layout threshold
// (cpuAnchorW=43 + layoutGap=2 + rightMin=48 + rightPad=2 → threshold=95),
// used to exercise the single-column path. The real minimum terminal is
// MinWidth=100 (innerW=98 ≥ 95), which is always wide.
const minNarrowW = 88

// minColW is the minimum right column width in the wide layout (= rightMin).
const minColW = 48

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
	checkNoWrap(t, "renderCPU(minColW)", newTestDashboard().renderCPU(minColW), minColW)
}

func TestRenderMemory_NoWrap(t *testing.T) {
	checkNoWrap(t, "renderMemory(minColW)", newTestDashboard().renderMemory(minColW), minColW)
}

func TestRenderDisks_NoWrap(t *testing.T) {
	checkNoWrap(t, "renderDisks(minColW)", newTestDashboard().renderDisks(minColW), minColW)
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

// ── narrow layout (single-column, innerW=88, below wide threshold 93) ────────

func TestRenderCPU_NoWrap_Narrow(t *testing.T) {
	checkNoWrap(t, "renderCPU(narrow)", newTestDashboard().renderCPU(minNarrowW), minNarrowW)
}

func TestRenderMemory_NoWrap_Narrow(t *testing.T) {
	checkNoWrap(t, "renderMemory(narrow)", newTestDashboard().renderMemory(minNarrowW), minNarrowW)
}

func TestRenderDisks_NoWrap_Narrow(t *testing.T) {
	checkNoWrap(t, "renderDisks(narrow)", newTestDashboard().renderDisks(minNarrowW), minNarrowW)
}

// TestDashboardView_WideNoWrap tests the full View() output in wide (two-column) mode
// at several representative widths to ensure no line overflows its column.
// w=99 is included to verify terminal width 101 (innerW=99) stays in wide layout.
func TestDashboardView_WideNoWrap(t *testing.T) {
	for _, w := range []int{99, 100, 108, 120, 140, 160, 200} {
		d := newTestDashboard()
		d.width = w
		d.height = 40
		out := d.View()
		checkNoWrap(t, fmt.Sprintf("View(wide w=%d)", w), out, w)
	}
}

// TestDashboardView_WideAsymmetric verifies that the right column (Memory+Disk)
// is wider than the left column at several widths, and that a long hostname
// does not expand the left anchor beyond cpuAnchorW.
func TestDashboardView_WideAsymmetric(t *testing.T) {
	for _, w := range []int{99, 120, 140, 160} {
		lay := dashboardLayout(w)
		if !lay.wide {
			t.Errorf("dashboardLayout(%d) unexpectedly narrow", w)
			continue
		}
		if lay.leftW != cpuAnchorW {
			t.Errorf("w=%d: leftW=%d, want cpuAnchorW=%d", w, lay.leftW, cpuAnchorW)
		}
		if lay.rightW <= lay.leftW {
			t.Errorf("w=%d: rightW=%d should be > leftW=%d", w, lay.rightW, lay.leftW)
		}
		if lay.gap != layoutGap {
			t.Errorf("w=%d: gap=%d, want %d", w, lay.gap, layoutGap)
		}

		// Long hostname must not cause View() to overflow.
		d := newTestDashboard()
		d.data.Hostname = "Vitalijs-MacBook-Air.local"
		d.width = w
		d.height = 40
		checkNoWrap(t, fmt.Sprintf("View(wide asymmetric w=%d)", w), d.View(), w)
	}
}

// TestRenderLoad_LoadAvgSupported checks that load values appear when supported
// and "N/A" appears when not (e.g. Windows).
func TestRenderLoad_LoadAvgSupported(t *testing.T) {
	t.Run("supported shows values", func(t *testing.T) {
		d := newTestDashboard()
		d.data.LoadAvgSupported = true
		d.data.LoadAvg1 = 1.23
		out := d.renderLoad(60)
		if !strings.Contains(out, "1.23") {
			t.Errorf("expected load value in output, got:\n%s", out)
		}
		if strings.Contains(out, "N/A") {
			t.Errorf("unexpected N/A when LoadAvgSupported=true, got:\n%s", out)
		}
	})
	t.Run("unsupported shows N/A", func(t *testing.T) {
		d := newTestDashboard()
		d.data.LoadAvgSupported = false
		out := d.renderLoad(60)
		if !strings.Contains(out, "N/A") {
			t.Errorf("expected N/A when LoadAvgSupported=false, got:\n%s", out)
		}
		if strings.Contains(out, "0.00") {
			t.Errorf("unexpected 0.00 when LoadAvgSupported=false, got:\n%s", out)
		}
	})
}

// TestRenderLoad_NoWrap verifies that renderLoad respects the width limit
// and truncates long hostnames rather than overflowing.
func TestRenderLoad_NoWrap(t *testing.T) {
	d := newTestDashboard()
	d.data.Hostname = "very-long-hostname.internal.company.example.com" // 47 chars
	const w = 43
	out := d.renderLoad(w)
	checkNoWrap(t, "renderLoad(long hostname)", out, w)
}

// TestDashboardView_WideNoWrap_LongHostname ensures a long hostname does not
// cause overflow or push the right column out of bounds.
func TestDashboardView_WideNoWrap_LongHostname(t *testing.T) {
	for _, w := range []int{99, 100, 120, 140} {
		d := newTestDashboard()
		d.data.Hostname = "very-long-hostname.internal.company.example.com"
		d.width = w
		d.height = 40
		out := d.View()
		checkNoWrap(t, fmt.Sprintf("View(wide long-host w=%d)", w), out, w)
	}
}

// TestDashboardView_NarrowNoWrap tests the full View() output in narrow mode.
func TestDashboardView_NarrowNoWrap(t *testing.T) {
	d := newTestDashboard()
	d.width = minNarrowW
	d.height = 40
	out := d.View()
	checkNoWrap(t, "View(narrow)", out, minNarrowW)
}

// TestDashboardView_NarrowOrder checks that sections appear in the expected order.
func TestDashboardView_NarrowOrder(t *testing.T) {
	d := newTestDashboard()
	d.width = minNarrowW
	d.height = 40
	out := d.View()
	cpuIdx := strings.Index(out, "CPU")
	memIdx := strings.Index(out, "Memory")
	diskIdx := strings.Index(out, "Disk")
	sysIdx := strings.Index(out, "System")
	if cpuIdx < 0 || memIdx < 0 || diskIdx < 0 || sysIdx < 0 {
		t.Fatalf("narrow view missing sections: CPU=%d Memory=%d Disk=%d System=%d", cpuIdx, memIdx, diskIdx, sysIdx)
	}
	if cpuIdx >= memIdx || memIdx >= diskIdx || diskIdx >= sysIdx {
		t.Errorf("narrow layout order wrong: CPU=%d Memory=%d Disk=%d System=%d", cpuIdx, memIdx, diskIdx, sysIdx)
	}
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

// TestDashboardView_RightColumnNotHalfWidth verifies the layout is asymmetric:
// the right column (Memory+Disk) is wider than the CPU-anchored left column.
// This guards against regression to a 50/50 split where Memory appears
// unnecessarily far from the left edge.
func TestDashboardView_RightColumnNotHalfWidth(t *testing.T) {
	for _, w := range []int{99, 100, 105, 120, 140, 160} {
		lay := dashboardLayout(w)
		if !lay.wide {
			t.Errorf("w=%d: expected wide layout (threshold=%d)", w, cpuAnchorW+layoutGap+rightMin+rightPad)
			continue
		}
		if lay.leftW != cpuAnchorW {
			t.Errorf("w=%d: leftW=%d, want %d (cpuAnchorW)", w, lay.leftW, cpuAnchorW)
		}
		if lay.rightW <= lay.leftW {
			t.Errorf("w=%d: rightW=%d ≤ leftW=%d (right column should be wider)", w, lay.rightW, lay.leftW)
		}
	}
}
