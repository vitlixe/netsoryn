package views

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/vitlixe/netsoryn/internal/config"
)

func slashKeyMsg() tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}}
}

// filterCount counts non-overlapping occurrences of "Filter:" in s.
func filterCount(s string) int {
	return strings.Count(s, "Filter:")
}

func TestProcessesView_NoDoubleFilter(t *testing.T) {
	p := NewProcesses(&config.Config{})
	p.loaded = true
	p.filter = "foo"
	p.filtering = true
	p.filterBuf.Reset()
	p.filterBuf.WriteString("foo")

	out := p.View()
	if n := filterCount(out); n != 1 {
		t.Errorf("expected exactly 1 'Filter:' in View(), got %d\noutput: %q", n, out)
	}
}

func TestProcessesView_FilteringOnly(t *testing.T) {
	p := NewProcesses(&config.Config{})
	p.loaded = true
	p.filter = ""
	p.filtering = true
	p.filterBuf.Reset()
	p.filterBuf.WriteString("ba")

	out := p.View()
	if n := filterCount(out); n != 1 {
		t.Errorf("expected exactly 1 'Filter:' in View(), got %d", n)
	}
	if !strings.Contains(out, "_") {
		t.Error("expected cursor '_' during filtering")
	}
}

func TestProcessesView_CommittedFilter(t *testing.T) {
	p := NewProcesses(&config.Config{})
	p.loaded = true
	p.filter = "nginx"
	p.filtering = false

	out := p.View()
	if n := filterCount(out); n != 1 {
		t.Errorf("expected exactly 1 'Filter:' in View(), got %d", n)
	}
}

func TestProcessesView_NoFilter(t *testing.T) {
	p := NewProcesses(&config.Config{})
	p.loaded = true

	out := p.View()
	if n := filterCount(out); n != 0 {
		t.Errorf("expected 0 'Filter:' in View() when no filter active, got %d", n)
	}
}

func TestProcessesUpdate_SlashNoOpDuringFiltering(t *testing.T) {
	p := NewProcesses(&config.Config{})
	p.loaded = true
	p.filtering = true
	p.filterBuf.Reset()
	p.filterBuf.WriteString("ab")
	p.filter = "ab"

	msg := slashKeyMsg()
	model, _ := p.Update(msg)
	got := model.(*Processes)

	if got.filterBuf.String() != "ab" {
		t.Errorf("expected filterBuf to stay 'ab' after '/', got %q", got.filterBuf.String())
	}
}
