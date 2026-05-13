package views

import (
	"testing"

	"github.com/charmbracelet/bubbles/table"
)

// tableRenderedWidth returns the actual terminal width a bubbles/table will
// occupy: each column's declared Width plus 2 chars of cell padding (1 left +
// 1 right from the default Padding(0,1) cell style).
func tableRenderedWidth(cols []table.Column) int {
	total := 0
	for _, col := range cols {
		total += col.Width + 2
	}
	return total
}

func TestSvcColumns_DarwinFitsWidth(t *testing.T) {
	widths := []int{34, 40, 80, 120, 200}
	for _, w := range widths {
		cols := svcColumns(w, "darwin")
		got := tableRenderedWidth(cols)
		if got > w {
			t.Errorf("darwin svcColumns(%d): rendered width %d > available %d", w, got, w)
		}
	}
}

func TestSvcColumns_LinuxFitsWidth(t *testing.T) {
	widths := []int{84, 90, 120, 200}
	for _, w := range widths {
		cols := svcColumns(w, "linux")
		got := tableRenderedWidth(cols)
		if got > w {
			t.Errorf("linux svcColumns(%d): rendered width %d > available %d", w, got, w)
		}
	}
}

func TestSvcColumns_WindowsFitsWidth(t *testing.T) {
	widths := []int{84, 90, 120, 200}
	for _, w := range widths {
		cols := svcColumns(w, "windows")
		got := tableRenderedWidth(cols)
		if got > w {
			t.Errorf("windows svcColumns(%d): rendered width %d > available %d", w, got, w)
		}
	}
}

func TestSvcColumns_MinWidths(t *testing.T) {
	// darwin narrow: nameW must be clamped to 20
	darwinCols := svcColumns(30, "darwin")
	if darwinCols[0].Width != 20 {
		t.Errorf("darwin svcColumns(30): Service width = %d, want 20", darwinCols[0].Width)
	}

	// linux narrow: Description must be clamped to 20
	linuxCols := svcColumns(80, "linux")
	descIdx := 3
	if linuxCols[descIdx].Width != 20 {
		t.Errorf("linux svcColumns(80): Description width = %d, want 20", linuxCols[descIdx].Width)
	}
}

func TestSvcDarwinNameWidth(t *testing.T) {
	cases := []struct {
		w    int
		want int
	}{
		{30, 20}, // clamped: 30-14=16 < 20
		{34, 20}, // boundary: 34-14=20, exact minimum
		{40, 26}, // 40-14=26
		{80, 66}, // 80-14=66
		{200, 186},
	}
	for _, c := range cases {
		got := svcDarwinNameWidth(c.w)
		if got != c.want {
			t.Errorf("svcDarwinNameWidth(%d) = %d, want %d", c.w, got, c.want)
		}
	}
}

func TestSvcDescriptionWidth(t *testing.T) {
	cases := []struct {
		w    int
		want int
	}{
		{80, 20},   // below threshold (83): clamped to 20
		{83, 20},   // at threshold: still 20
		{84, 21},   // 84-63=21, first dynamic value
		{120, 57},  // 120-63=57
		{200, 137}, // 200-63=137
	}
	for _, c := range cases {
		got := svcDescriptionWidth(c.w)
		if got != c.want {
			t.Errorf("svcDescriptionWidth(%d) = %d, want %d", c.w, got, c.want)
		}
	}
}

func TestSvcColumns_DescriptionMatchesHelper(t *testing.T) {
	widths := []int{84, 100, 120, 160, 200}
	for _, w := range widths {
		cols := svcColumns(w, "linux")
		wantDesc := svcDescriptionWidth(w)
		if cols[3].Width != wantDesc {
			t.Errorf("linux svcColumns(%d)[3].Width = %d, want svcDescriptionWidth = %d",
				w, cols[3].Width, wantDesc)
		}
		cols = svcColumns(w, "windows")
		if cols[3].Width != wantDesc {
			t.Errorf("windows svcColumns(%d)[3].Width = %d, want svcDescriptionWidth = %d",
				w, cols[3].Width, wantDesc)
		}
	}
}
