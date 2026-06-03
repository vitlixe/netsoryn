package views

import (
	"context"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/vitlixe/netsoryn/internal/collectors"
	"github.com/vitlixe/netsoryn/internal/config"
)

func TestVisibleBlocks(t *testing.T) {
	blocks := []string{"a", "b\nb", "c", "d"} // rendered heights: 1, 2, 1, 1
	cases := []struct {
		offset, height, want int
	}{
		{0, 10, 4}, // everything fits
		{0, 1, 1},  // only the first block
		{0, 3, 2},  // a(1) + b(2) = 3; c would overflow
		{1, 3, 2},  // b(2) + c(1) = 3
		{3, 1, 1},  // last block
		{0, 0, 1},  // height clamped to >= 1, always at least one block
		{4, 10, 0}, // offset past the end
	}
	for _, c := range cases {
		if got := visibleBlocks(blocks, c.offset, c.height); got != c.want {
			t.Errorf("visibleBlocks(offset=%d, height=%d) = %d, want %d", c.offset, c.height, got, c.want)
		}
	}
}

func TestRenderScrollableList_Markers(t *testing.T) {
	blocks := []string{"one", "two", "three", "four", "five"}
	out := renderScrollableList("HEAD", blocks, 1, 4) // small height, scrolled down by 1
	if !strings.Contains(out, "more above") {
		t.Errorf("expected 'more above' marker when offset>0, got:\n%s", out)
	}
	if !strings.Contains(out, "more below") {
		t.Errorf("expected 'more below' marker when content overflows, got:\n%s", out)
	}
}

func TestDNSScroll_Keys(t *testing.T) {
	d := NewDNS(&config.Config{}, context.Background())
	d.results = []collectors.DNSResult{{Domain: "a"}, {Domain: "b"}, {Domain: "c"}}
	d.height = 5

	send := func(s string) {
		m, _ := d.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)})
		d = m.(*DNS)
	}

	send("j")
	if d.offset != 1 {
		t.Errorf("after j: offset=%d, want 1", d.offset)
	}
	send("G")
	if d.offset != 2 {
		t.Errorf("after G: offset=%d, want 2 (last result)", d.offset)
	}
	send("j") // already at bottom — must clamp
	if d.offset != 2 {
		t.Errorf("j past end: offset=%d, want 2 (clamped)", d.offset)
	}
	send("g")
	if d.offset != 0 {
		t.Errorf("after g: offset=%d, want 0", d.offset)
	}
	send("k") // already at top — must clamp
	if d.offset != 0 {
		t.Errorf("k past top: offset=%d, want 0 (clamped)", d.offset)
	}
}

func TestDNSScroll_RemoveClampsOffset(t *testing.T) {
	d := NewDNS(&config.Config{}, context.Background())
	d.results = []collectors.DNSResult{{Domain: "a"}, {Domain: "b"}}
	d.offset = 1

	m, _ := d.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("D")})
	d = m.(*DNS)
	if d.offset != 0 {
		t.Errorf("after removing down to 1 result: offset=%d, want 0", d.offset)
	}
}
