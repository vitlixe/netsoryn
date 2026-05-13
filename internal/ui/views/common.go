package views

import (
	"strings"

	"github.com/charmbracelet/bubbles/table"

	"github.com/vitlixe/netsoryn/internal/ui/styles"
)

// trimLastRune removes the last Unicode code point from s.
// Safe for multibyte UTF-8: "fé" → "f", "тест" → "тес".
func trimLastRune(s string) string {
	r := []rune(s)
	if len(r) == 0 {
		return ""
	}
	return string(r[:len(r)-1])
}

// ContentSizeMsg is sent by the root model to notify views of their available drawing area.
type ContentSizeMsg struct {
	Width  int
	Height int
}

// syncTopRow returns the updated first-visible-row index after a cursor move.
// It mirrors the "keep cursor visible" invariant: scroll down when cursor passes
// the bottom edge, scroll up when cursor passes the top edge, no scroll otherwise.
// Call this after any navigation that changes the table cursor.
func syncTopRow(oldTop, cursor, visibleRows int) int {
	if visibleRows <= 0 || cursor < 0 {
		return 0
	}
	if cursor > oldTop+visibleRows-1 {
		return cursor - visibleRows + 1
	}
	if cursor < oldTop {
		return cursor
	}
	return oldTop
}

func clampTopRow(topRow, rows, visibleRows int) int {
	if topRow < 0 || rows == 0 || visibleRows <= 0 || rows <= visibleRows {
		return 0
	}
	maxTop := rows - visibleRows
	if topRow > maxTop {
		return maxTop
	}
	return topRow
}

func setTableRows(t *table.Model, rows []table.Row) {
	t.SetRows(rows)

	rowCount := len(t.Rows())
	if rowCount == 0 {
		t.SetCursor(0)
		return
	}
	switch cursor := t.Cursor(); {
	case cursor < 0:
		t.SetCursor(0)
	case cursor >= rowCount:
		t.SetCursor(rowCount - 1)
	}
}

// tableScrollFlags returns whether there are hidden rows above/below the visible
// window, using the caller-maintained topRow (first visible data row index).
//
// up   = topRow > 0              — rows hidden above.
// down = bottom < len(rows)-1    — rows hidden below.
//
// topRow is clamped to a valid range internally so stale values from row-count
// changes don't produce wrong output.
func tableScrollFlags(t table.Model, topRow int) (up, down bool) {
	rows := len(t.Rows())
	vis := t.Height()

	if vis <= 0 || rows == 0 {
		return false, false
	}

	topRow = clampTopRow(topRow, rows, vis)
	bottom := topRow + vis - 1
	if bottom >= rows {
		bottom = rows - 1
	}

	up = topRow > 0
	down = bottom < rows-1
	return
}

// tableScrollHint returns "↑ more", "↓ more", "↑/↓ more", or "".
func tableScrollHint(t table.Model, topRow int) string {
	up, down := tableScrollFlags(t, topRow)
	switch {
	case up && down:
		return "↑/↓ more"
	case up:
		return "↑ more"
	case down:
		return "↓ more"
	default:
		return ""
	}
}

// wrapTableWithScrollHints overlays scroll indicators on a bubbles table's rendered output.
// The up indicator replaces the first data row (line index 2, after header+separator).
// The down indicator replaces the last rendered line.
// Line count is preserved — no new rows are added.
func wrapTableWithScrollHints(t table.Model, topRow int) string {
	up, down := tableScrollFlags(t, topRow)

	raw := t.View()
	if !up && !down {
		return raw
	}

	lines := strings.Split(raw, "\n")
	const headerLines = 2 // header + separator
	if len(lines) <= headerLines {
		return raw
	}

	firstData := headerLines
	lastData := len(lines) - 1

	if up && down && firstData == lastData {
		lines[firstData] = styles.Muted.Render("  ↑/↓ more")
	} else {
		if up {
			lines[firstData] = styles.Muted.Render("  ↑ more")
		}
		if down {
			lines[lastData] = styles.Muted.Render("  ↓ more")
		}
	}
	return strings.Join(lines, "\n")
}
