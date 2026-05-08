package views

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/table"
)

// buildTable creates a focused table with nRows rows and a viewport height of visibleRows.
// Uses tableStyles() to match production (BorderBottom adds a separator row, so
// headersHeight=2). bubbles/table.SetHeight(h) stores h-headersHeight in viewport:
//
//	SetHeight(visibleRows + 2) → Height() = visibleRows.
func buildTable(nRows, visibleRows int) table.Model {
	cols := []table.Column{{Title: "X", Width: 10}}
	rows := make([]table.Row, nRows)
	for i := range rows {
		rows[i] = table.Row{"x"}
	}
	t := table.New(
		table.WithColumns(cols),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithStyles(tableStyles()),
	)
	t.SetHeight(visibleRows + 2)
	return t
}

// navigate presses Down n times from the current cursor and returns the updated
// topRow. This mirrors how the views track topRow: syncTopRow after each step.
func navigate(t *table.Model, topRow, steps int) int {
	for i := 0; i < steps; i++ {
		t.MoveDown(1)
		topRow = syncTopRow(topRow, t.Cursor(), t.Height())
	}
	return topRow
}

// ── tableScrollFlags / tableScrollHint ───────────────────────────────────────

func TestTableScrollHint_Empty_WhenAllFit(t *testing.T) {
	// 5 rows, 10 visible → all fit, no scroll
	tbl := buildTable(5, 10)
	if got := tableScrollHint(tbl, 0); got != "" {
		t.Errorf("expected empty hint, got %q", got)
	}
}

func TestTableScrollHint_NoUp_WhenFirstRowVisible(t *testing.T) {
	// cursor=1, visibleRows=10: top=0 → up=false
	tbl := buildTable(20, 10)
	topRow := navigate(&tbl, 0, 1)
	got := tableScrollHint(tbl, topRow)
	if strings.Contains(got, "↑") {
		t.Errorf("cursor=1 < visibleRows=10: first row still visible, should not show ↑, got %q", got)
	}
}

func TestTableScrollHint_DownOnly_AtTop(t *testing.T) {
	// cursor=0, visibleRows=5, rows=20: top=0, bottom=4 → up=false, down=true
	tbl := buildTable(20, 5)
	if got := tableScrollHint(tbl, 0); got != "↓ more" {
		t.Errorf("expected \"↓ more\", got %q", got)
	}
}

func TestTableScrollHint_UpOnly_AtBottom(t *testing.T) {
	// GotoBottom → cursor=19, topRow synced to 15: top=15, bottom=19 → up=true, down=false
	tbl := buildTable(20, 5)
	topRow := 0
	tbl.GotoBottom()
	topRow = syncTopRow(topRow, tbl.Cursor(), tbl.Height())
	if got := tableScrollHint(tbl, topRow); got != "↑ more" {
		t.Errorf("expected \"↑ more\", got %q", got)
	}
}

func TestTableScrollHint_UpDown_InMiddle(t *testing.T) {
	// cursor=10, visibleRows=5, rows=20: top=6, bottom=10 → up=true, down=true
	tbl := buildTable(20, 5)
	topRow := navigate(&tbl, 0, 10)
	if got := tableScrollHint(tbl, topRow); got != "↑/↓ more" {
		t.Errorf("expected \"↑/↓ more\", got %q", got)
	}
}

// ── spec test cases ──────────────────────────────────────────────────────────

func TestTableScrollFlags_Spec_Cursor1_Rows10_Vis8(t *testing.T) {
	// cursor=1, rows=10, visibleRows=8: top=0, bottom=7 → up=false, down=true
	tbl := buildTable(10, 8)
	topRow := navigate(&tbl, 0, 1)
	up, down := tableScrollFlags(tbl, topRow)
	if up {
		t.Errorf("up should be false (top=0), got true")
	}
	if !down {
		t.Errorf("down should be true (bottom=7 < 9), got false")
	}
}

func TestTableScrollFlags_Spec_Cursor7_Rows8_Vis8(t *testing.T) {
	// rows=8, visibleRows=8: all fit. Any cursor → up=false, down=false.
	tbl := buildTable(8, 8)
	topRow := navigate(&tbl, 0, 7)
	up, down := tableScrollFlags(tbl, topRow)
	if up {
		t.Errorf("up should be false, got true")
	}
	if down {
		t.Errorf("down should be false (bottom=7 == rows-1=7), got true")
	}
}

func TestTableScrollFlags_Spec_Cursor8_FromTop_Rows10_Vis8(t *testing.T) {
	// Navigate from top to cursor=8: viewport scrolls once when cursor crosses
	// the bottom edge, so top=1, bottom=8. up=true, down=true.
	tbl := buildTable(10, 8)
	topRow := navigate(&tbl, 0, 8)
	up, down := tableScrollFlags(tbl, topRow)
	if !up {
		t.Errorf("up should be true (top=1 > 0)")
	}
	if !down {
		t.Errorf("down should be true (bottom=8 < 9, row 9 not visible)")
	}
}

func TestTableScrollFlags_Spec_Cursor7_FromTop_Rows10_Vis8(t *testing.T) {
	// cursor=7 from the top: first scroll trigger at cursor=8 has not fired yet.
	// top=0, bottom=7. up=false, down=true.
	tbl := buildTable(10, 8)
	topRow := navigate(&tbl, 0, 7)
	up, down := tableScrollFlags(tbl, topRow)
	if up {
		t.Errorf("up should be false (top=0)")
	}
	if !down {
		t.Errorf("down should be true (bottom=7 < 9)")
	}
}

func TestTableScrollFlags_Spec_Cursor9_GotoBottom_Rows10_Vis8(t *testing.T) {
	// GotoBottom → cursor=9, topRow synced to 2: top=2, bottom=9.
	// up=true, down=false (last row visible).
	tbl := buildTable(10, 8)
	topRow := 0
	tbl.GotoBottom()
	topRow = syncTopRow(topRow, tbl.Cursor(), tbl.Height())
	up, down := tableScrollFlags(tbl, topRow)
	if !up {
		t.Errorf("up should be true (top=2 > 0)")
	}
	if down {
		t.Errorf("down should be false (bottom=9 == rows-1=9, last row visible)")
	}
}

func TestTableScrollFlags_Spec_AllFit_Rows10_Vis10(t *testing.T) {
	// All 10 rows fit in a 10-row viewport. Any cursor: up=false, down=false.
	tbl := buildTable(10, 10)
	topRow := navigate(&tbl, 0, 8)
	up, down := tableScrollFlags(tbl, topRow)
	if up {
		t.Errorf("up should be false (all rows fit)")
	}
	if down {
		t.Errorf("down should be false (all rows fit)")
	}
}

func TestTableScrollFlags_StaleTopRow_AllRowsNowFit(t *testing.T) {
	tbl := buildTable(5, 10)
	up, down := tableScrollFlags(tbl, 99)
	if up || down {
		t.Errorf("stale topRow with all rows visible should produce no indicators, got up=%v down=%v", up, down)
	}
}

func TestSyncTopRow_NegativeCursor(t *testing.T) {
	if got := syncTopRow(5, -1, 10); got != 0 {
		t.Errorf("syncTopRow with negative cursor = %d, want 0", got)
	}
}

func TestSetTableRowsClampsCursorAfterShrink(t *testing.T) {
	tbl := buildTable(20, 5)
	tbl.GotoBottom()
	if got := tbl.Cursor(); got != 19 {
		t.Fatalf("test setup cursor = %d, want 19", got)
	}

	rows := make([]table.Row, 3)
	for i := range rows {
		rows[i] = table.Row{"x"}
	}
	setTableRows(&tbl, rows)

	if got := tbl.Cursor(); got != 2 {
		t.Errorf("cursor after row shrink = %d, want 2", got)
	}
	up, down := tableScrollFlags(tbl, 99)
	if up || down {
		t.Errorf("shrunk rows should fit without indicators, got up=%v down=%v", up, down)
	}
}

func TestTableScrollFlags_Spec_BottomThenUp_LastRowStillVisible(t *testing.T) {
	// Repro for the original bug: navigate to the last row, then move up by one.
	// topRow does not scroll back because cursor is still within [topRow, topRow+vis-1].
	// Last row (9) remains visible → down must be false.
	//
	// rows=10, vis=8: GotoBottom → topRow=2, bottom=9.
	// MoveUp(1) → cursor=8, topRow stays 2, bottom=9. down=false.
	tbl := buildTable(10, 8)
	topRow := 0
	tbl.GotoBottom()
	topRow = syncTopRow(topRow, tbl.Cursor(), tbl.Height())
	tbl.MoveUp(1)
	topRow = syncTopRow(topRow, tbl.Cursor(), tbl.Height())
	_, down := tableScrollFlags(tbl, topRow)
	if down {
		t.Errorf("down should be false: last row (9) is still visible after MoveUp from cursor=9 to cursor=8")
	}
}

// ── wrapTableWithScrollHints ──────────────────────────────────────────────────

func TestWrapTable_NoHint_WhenAllFit(t *testing.T) {
	tbl := buildTable(5, 10)
	raw := tbl.View()
	if got := wrapTableWithScrollHints(tbl, 0); got != raw {
		t.Errorf("expected unchanged output when all rows fit")
	}
}

func TestWrapTable_DownIndicator_AtTop(t *testing.T) {
	// cursor=0, rows=20, visibleRows=5, topRow=0 → only ↓
	tbl := buildTable(20, 5)
	got := wrapTableWithScrollHints(tbl, 0)
	lines := strings.Split(got, "\n")

	last := lines[len(lines)-1]
	if !strings.Contains(last, "↓") {
		t.Errorf("expected ↓ in last line, got: %q", last)
	}
	firstData := lines[2]
	if strings.Contains(firstData, "↑") {
		t.Errorf("should not have ↑ when topRow=0 (first row visible), first data row: %q", firstData)
	}
}

func TestWrapTable_UpIndicator_AtBottom(t *testing.T) {
	// GotoBottom → topRow=15, rows=20, visibleRows=5 → only ↑
	tbl := buildTable(20, 5)
	topRow := 0
	tbl.GotoBottom()
	topRow = syncTopRow(topRow, tbl.Cursor(), tbl.Height())
	got := wrapTableWithScrollHints(tbl, topRow)
	lines := strings.Split(got, "\n")

	firstData := lines[2]
	if !strings.Contains(firstData, "↑") {
		t.Errorf("expected ↑ in first data row, got: %q", firstData)
	}
	last := lines[len(lines)-1]
	if strings.Contains(last, "↓") {
		t.Errorf("should not have ↓ at bottom, last row: %q", last)
	}
}

func TestWrapTable_BothIndicators_InMiddle(t *testing.T) {
	// cursor=10, rows=20, visibleRows=5 → both ↑ and ↓
	tbl := buildTable(20, 5)
	topRow := navigate(&tbl, 0, 10)
	got := wrapTableWithScrollHints(tbl, topRow)
	lines := strings.Split(got, "\n")

	if !strings.Contains(lines[2], "↑") {
		t.Errorf("expected ↑ in first data row, got: %q", lines[2])
	}
	last := lines[len(lines)-1]
	if !strings.Contains(last, "↓") {
		t.Errorf("expected ↓ in last line, got: %q", last)
	}
}

func TestWrapTable_HeaderNotReplaced(t *testing.T) {
	tbl := buildTable(20, 5)
	topRow := navigate(&tbl, 0, 10)
	got := wrapTableWithScrollHints(tbl, topRow)
	lines := strings.Split(got, "\n")

	header := lines[0]
	if strings.Contains(header, "↑") || strings.Contains(header, "↓") {
		t.Errorf("header row must not be replaced by scroll indicator: %q", header)
	}
}

func TestWrapTable_LineCountPreserved(t *testing.T) {
	tbl := buildTable(20, 5)
	rawLines := strings.Split(tbl.View(), "\n")
	topRow := navigate(&tbl, 0, 10)
	wrappedLines := strings.Split(wrapTableWithScrollHints(tbl, topRow), "\n")
	if len(rawLines) != len(wrappedLines) {
		t.Errorf("line count changed: raw=%d, wrapped=%d", len(rawLines), len(wrappedLines))
	}
}
