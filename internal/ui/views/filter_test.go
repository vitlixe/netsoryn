package views

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/vitlixe/netsoryn/internal/collectors"
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

func TestTrimLastRune(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"", ""},
		{"abc", "ab"},
		{"fé", "f"},
		{"тест", "тес"},
		{"🙂x", "🙂"},
	}
	for _, c := range cases {
		if got := trimLastRune(c.in); got != c.want {
			t.Errorf("trimLastRune(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestPortsView_EntryCount(t *testing.T) {
	ports := []collectors.PortStat{
		{Port: 80, Protocol: "tcp", Address: "0.0.0.0", State: "LISTEN", Process: "nginx"},
		{Port: 443, Protocol: "tcp", Address: "0.0.0.0", State: "LISTEN", Process: "nginx"},
		{Port: 5432, Protocol: "tcp", Address: "127.0.0.1", State: "LISTEN", Process: "postgres"},
	}

	t.Run("no filter shows all entries", func(t *testing.T) {
		p := NewPorts(&config.Config{})
		p.loaded = true
		p.data = ports
		p.rebuildRows()

		out := p.View()
		want := fmt.Sprintf("%d entries", len(ports))
		if !strings.Contains(out, want) {
			t.Errorf("expected %q in View(), got:\n%s", want, out)
		}
	})

	t.Run("filter reduces entry count", func(t *testing.T) {
		p := NewPorts(&config.Config{})
		p.loaded = true
		p.data = ports
		p.filter = "nginx"
		p.rebuildRows()

		out := p.View()
		want := "2 entries"
		if !strings.Contains(out, want) {
			t.Errorf("expected %q in View(), got:\n%s", want, out)
		}
	})
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
