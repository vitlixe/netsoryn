package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/vitlixe/netsoryn/internal/config"
)

func rune1(r rune) tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}} }

func isQuit(cmd tea.Cmd) bool {
	if cmd == nil {
		return false
	}
	_, ok := cmd().(tea.QuitMsg)
	return ok
}

// enterHTTPInput puts the HTTP view into its text-capturing state by sending "n".
func enterHTTPInput(t *testing.T) RootModel {
	t.Helper()
	m := newTestModel(ViewHTTP)
	updated, _ := m.Update(rune1('n'))
	rm := updated.(RootModel)
	if c, ok := rm.viewModels[ViewHTTP].(inputCapturer); !ok || !c.CapturingInput() {
		t.Fatal("HTTP view did not enter input-capturing state after 'n'")
	}
	return rm
}

func TestKeyRouting_DigitDoesNotSwitchViewWhileCapturing(t *testing.T) {
	rm := enterHTTPInput(t)
	updated, _ := rm.Update(rune1('2'))
	rm = updated.(RootModel)
	if rm.active != ViewHTTP {
		t.Errorf("digit key switched view while capturing input: active=%d, want %d", rm.active, ViewHTTP)
	}
}

func TestKeyRouting_QuitKeyDoesNotQuitWhileCapturing(t *testing.T) {
	rm := enterHTTPInput(t)
	updated, cmd := rm.Update(rune1('q'))
	rm = updated.(RootModel)
	if isQuit(cmd) {
		t.Error("'q' quit the program while a view was capturing input")
	}
	if rm.active != ViewHTTP {
		t.Errorf("'q' changed active view while capturing: active=%d", rm.active)
	}
}

func TestKeyRouting_CtrlCQuitsWhileCapturing(t *testing.T) {
	rm := enterHTTPInput(t)
	_, cmd := rm.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if !isQuit(cmd) {
		t.Error("ctrl+c did not quit while a view was capturing input")
	}
}

func TestKeyRouting_DigitSwitchesViewWhenNotCapturing(t *testing.T) {
	m := newTestModel(ViewDashboard)
	updated, _ := m.Update(rune1('2'))
	rm := updated.(RootModel)
	if rm.active != ViewProcesses {
		t.Errorf("digit key did not switch view when idle: active=%d, want %d", rm.active, ViewProcesses)
	}
}

func TestSwitchTo_ChangesActiveAndRefreshes(t *testing.T) {
	m := newTestModel(ViewDashboard)
	m2, cmd := m.switchTo(ViewProcesses)
	if m2.active != ViewProcesses {
		t.Errorf("active = %d, want ViewProcesses (%d)", m2.active, ViewProcesses)
	}
	if cmd == nil {
		t.Error("switching to a collecting view should return a refresh command")
	}
}

func TestSwitchTo_SameViewIsNoOp(t *testing.T) {
	m := newTestModel(ViewDashboard)
	if _, cmd := m.switchTo(ViewDashboard); cmd != nil {
		t.Error("switching to the already-active view should be a no-op (nil cmd)")
	}
}

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

func TestRenderBox_LeftAlignedContent(t *testing.T) {
	m := newTestModel(ViewDashboard)
	m.width = 222 // innerW = 220

	const content = "MARKER"
	out := m.renderBox(content, 3)

	for _, line := range strings.Split(out, "\n") {
		if !strings.Contains(line, content) {
			continue
		}
		idx := strings.Index(line, content)
		if idx < 0 {
			continue
		}
		before := line[:idx]
		spaces := len(before) - len(strings.TrimRight(before, " "))
		if spaces != 0 {
			t.Errorf("renderBox should left-align content: got %d spaces before %q", spaces, content)
		}
		return
	}
	t.Fatalf("renderBox output did not contain %q: %q", content, out)
}

func TestRenderBox_WideContentDoesNotPanic(t *testing.T) {
	m := newTestModel(ViewDashboard)
	m.width = 102 // innerW = 100

	wide := strings.Repeat("y", 200)
	_ = m.renderBox(wide, 3) // must not panic
}

// TestRenderBox_OverflowLineClampedToWidth verifies that a content line wider
// than the inner area is truncated, so every rendered row stays exactly the
// frame width and the right border cannot be pushed off-screen.
func TestRenderBox_OverflowLineClampedToWidth(t *testing.T) {
	m := newTestModel(ViewDashboard)
	m.width = 102

	out := m.renderBox(strings.Repeat("y", 500), 3)
	for i, line := range strings.Split(out, "\n") {
		if w := lipgloss.Width(line); w != m.width {
			t.Errorf("renderBox line %d width = %d, want %d", i, w, m.width)
		}
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

func TestFooterNoMoreAnywhere(t *testing.T) {
	for id := ViewID(0); id < numViews; id++ {
		footer := newTestModel(id).renderFooter()
		if strings.Contains(footer, "more") {
			t.Errorf("view %d footer should not contain scroll indicators, got: %s", id, footer)
		}
	}
}
