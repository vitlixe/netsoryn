package styles

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestTruncate(t *testing.T) {
	cases := []struct {
		s    string
		n    int
		want string
	}{
		{"hello", 10, "hello"},
		{"hello", 5, "hello"},
		{"hello", 4, "hel…"},
		{"hello", 2, "h…"},
		{"hello", 1, "…"},
		{"hello", 0, ""},  // must not panic on n <= 0
		{"hello", -3, ""}, // must not panic on negative n
		{"", 5, ""},
	}
	for _, c := range cases {
		if got := Truncate(c.s, c.n); got != c.want {
			t.Errorf("Truncate(%q, %d) = %q, want %q", c.s, c.n, got, c.want)
		}
	}
}

func TestTruncate_VisualWidthNotExceeded(t *testing.T) {
	// Each CJK rune is 2 columns wide; rune-count truncation would overflow a
	// fixed-width column, so the result must respect visual width.
	const wide = "日本語テスト"
	for _, n := range []int{2, 3, 4, 5, 6, 7} {
		got := Truncate(wide, n)
		if w := lipgloss.Width(got); w > n {
			t.Errorf("Truncate(%q, %d) visual width = %d, exceeds %d (got %q)", wide, n, w, n, got)
		}
	}
}
