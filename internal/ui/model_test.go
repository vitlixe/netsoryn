package ui

import (
	"strings"
	"testing"

	"github.com/vitlixe/netsoryn/internal/config"
)

func newTestModel(active ViewID) RootModel {
	cfg := &config.Config{}
	m := New(cfg, "test")
	m.active = active
	m.width = 120
	m.height = 30
	return m
}

func TestFooterDashboard_NoNavFilterSortRefresh(t *testing.T) {
	footer := newTestModel(ViewDashboard).renderFooter()
	for _, unwanted := range []string{"navigate", "filter", "sort", "refresh"} {
		if strings.Contains(footer, unwanted) {
			t.Errorf("Dashboard footer should not contain %q, got: %s", unwanted, footer)
		}
	}
}

func TestFooterProcesses_HasNavFilterSort(t *testing.T) {
	footer := newTestModel(ViewProcesses).renderFooter()
	for _, want := range []string{"navigate", "filter", "sort"} {
		if !strings.Contains(footer, want) {
			t.Errorf("Processes footer should contain %q, got: %s", want, footer)
		}
	}
}

func TestFooterTableViews_HasNavFilterNoSort(t *testing.T) {
	tableViews := []ViewID{ViewNetwork, ViewPorts, ViewServices, ViewDocker}
	for _, id := range tableViews {
		footer := newTestModel(id).renderFooter()
		if !strings.Contains(footer, "navigate") {
			t.Errorf("view %d footer should contain navigate", id)
		}
		if !strings.Contains(footer, "filter") {
			t.Errorf("view %d footer should contain filter", id)
		}
		if strings.Contains(footer, "sort") {
			t.Errorf("view %d footer should not contain sort", id)
		}
	}
}

func TestFooterNetwork_HasTabsHint(t *testing.T) {
	footer := newTestModel(ViewNetwork).renderFooter()
	if !strings.Contains(footer, "h/l") {
		t.Errorf("Network footer should contain h/l tabs hint, got: %s", footer)
	}
	if !strings.Contains(footer, "tabs") {
		t.Errorf("Network footer should contain tabs hint, got: %s", footer)
	}
}

func TestFooterDNS_HasNewQueryHint(t *testing.T) {
	footer := newTestModel(ViewDNS).renderFooter()
	if !strings.Contains(footer, "n") {
		t.Errorf("DNS footer should contain n hint, got: %s", footer)
	}
	if !strings.Contains(footer, "query") {
		t.Errorf("DNS footer should contain query hint, got: %s", footer)
	}
}

func TestFooterHTTP_HasNewCheckHint(t *testing.T) {
	footer := newTestModel(ViewHTTP).renderFooter()
	if !strings.Contains(footer, "n") {
		t.Errorf("HTTP footer should contain n hint, got: %s", footer)
	}
	if !strings.Contains(footer, "check") {
		t.Errorf("HTTP footer should contain check hint, got: %s", footer)
	}
}

func TestFooterDNSHTTP_NoNavFilterSort(t *testing.T) {
	for _, id := range []ViewID{ViewDNS, ViewHTTP} {
		footer := newTestModel(id).renderFooter()
		for _, unwanted := range []string{"navigate", "filter", "sort", "refresh"} {
			if strings.Contains(footer, unwanted) {
				t.Errorf("view %d footer should not contain %q", id, unwanted)
			}
		}
	}
}

// ── content-width helpers ─────────────────────────────────────────────────────

func TestViewMaxWidth(t *testing.T) {
	cases := []struct {
		id   ViewID
		want int
	}{
		{ViewDashboard, 120},
		{ViewDNS, 120},
		{ViewHTTP, 120},
		{ViewPorts, 150},
		{ViewServices, 150},
		{ViewProcesses, 160},
		{ViewNetwork, 160},
		{ViewDocker, 160},
	}
	for _, tc := range cases {
		if got := viewMaxWidth(tc.id); got != tc.want {
			t.Errorf("viewMaxWidth(%d) = %d, want %d", tc.id, got, tc.want)
		}
	}
}

func TestViewContentWidth(t *testing.T) {
	for id := ViewID(0); id < numViews; id++ {
		maxW := viewMaxWidth(id)
		// Below max: uses available width unchanged.
		below := maxW - 10
		if got := viewContentWidth(id, below); got != below {
			t.Errorf("viewContentWidth(%d, %d) = %d, want %d", id, below, got, below)
		}
		// At max: exactly max.
		if got := viewContentWidth(id, maxW); got != maxW {
			t.Errorf("viewContentWidth(%d, %d) = %d, want %d", id, maxW, got, maxW)
		}
		// Above max: capped at max.
		above := maxW + 60
		if got := viewContentWidth(id, above); got != maxW {
			t.Errorf("viewContentWidth(%d, %d) = %d, want %d (capped)", id, above, got, maxW)
		}
		// Never returns more than available.
		for _, w := range []int{100, 120, 160, 200} {
			if got := viewContentWidth(id, w); got > w {
				t.Errorf("viewContentWidth(%d, %d) = %d > available %d", id, w, got, w)
			}
		}
	}
}

// TestRenderBox_CenteredContent verifies that renderBox adds symmetric left padding
// when the terminal is wider than the view's max content width.
func TestRenderBox_CenteredContent(t *testing.T) {
	// Set terminal wide enough to trigger centering for Dashboard (maxW=120).
	// innerW = 220, leftPad = (220 - 120) / 2 = 50.
	m := newTestModel(ViewDashboard)
	m.width = 222 // innerW = 220

	const content = "MARKER"
	out := m.renderBox(content, 3)

	innerW := m.width - 2                                     // 220
	expectedPad := (innerW - viewMaxWidth(ViewDashboard)) / 2 // 50

	for _, line := range strings.Split(out, "\n") {
		if !strings.Contains(line, "MARKER") {
			continue
		}
		// Strip the leading border char (│ is multi-byte, use TrimLeft on the raw string).
		// The border style adds ANSI codes; find the first space run after any non-space prefix.
		// We count ASCII spaces following the │ border by looking at the raw bytes.
		idx := strings.Index(line, "MARKER")
		if idx < 0 {
			continue
		}
		// Count spaces between the │ and MARKER (ANSI codes may precede │, so we look
		// for the run of spaces that precedes MARKER in the raw string).
		before := line[:idx]
		spaces := len(before) - len(strings.TrimRight(before, " "))
		if spaces < expectedPad {
			t.Errorf("renderBox centering: expected ≥%d spaces before content, got %d (innerW=%d, maxW=%d)",
				expectedPad, spaces, innerW, viewMaxWidth(ViewDashboard))
		}
		break
	}
}

func TestFooterNoRefreshAnywhere(t *testing.T) {
	for id := ViewID(0); id < numViews; id++ {
		footer := newTestModel(id).renderFooter()
		if strings.Contains(footer, "refresh") {
			t.Errorf("view %d footer should not contain refresh (not implemented)", id)
		}
	}
}

func TestFooterSortHint_OnlyInProcesses(t *testing.T) {
	views := []struct {
		id       ViewID
		wantSort bool
	}{
		{ViewDashboard, false},
		{ViewProcesses, true},
		{ViewNetwork, false},
		{ViewPorts, false},
		{ViewServices, false},
		{ViewDocker, false},
		{ViewDNS, false},
		{ViewHTTP, false},
	}

	for _, tc := range views {
		footer := newTestModel(tc.id).renderFooter()
		hasSortHint := strings.Contains(footer, "sort")
		if hasSortHint != tc.wantSort {
			t.Errorf("view %d: footer sort hint present=%v, want=%v", tc.id, hasSortHint, tc.wantSort)
		}
	}
}
