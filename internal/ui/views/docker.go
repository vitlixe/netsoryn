package views

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/vitlixe/netsoryn/internal/collectors"
	"github.com/vitlixe/netsoryn/internal/config"
	"github.com/vitlixe/netsoryn/internal/ui/keys"
	"github.com/vitlixe/netsoryn/internal/ui/styles"
)

type dockerTickMsg time.Time
type dockerDataMsg struct {
	data collectors.DockerData
	err  error
}

type Docker struct {
	cfg       *config.Config
	data      collectors.DockerData
	rows      []collectors.ContainerStat
	cursor    int
	offset    int
	err       error
	keys      keys.NavKeyMap
	filter    string
	filtering bool
	filterBuf strings.Builder
	width     int
	height    int
	loaded    bool
}

func NewDocker(cfg *config.Config) *Docker {
	return &Docker{
		cfg:  cfg,
		keys: keys.DefaultNavKeyMap(),
	}
}

func (d *Docker) Init() tea.Cmd {
	return tea.Batch(fetchDockerData(d.cfg.DockerSocket), tickDocker(d.cfg.RefreshInterval))
}

func (d *Docker) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case ContentSizeMsg:
		d.width = msg.Width
		d.height = msg.Height

	case dockerDataMsg:
		d.loaded = true
		d.err = msg.err
		if msg.err == nil {
			d.data = msg.data
			d.rebuildRows()
		}

	case dockerTickMsg:
		return d, tea.Batch(fetchDockerData(d.cfg.DockerSocket), tickDocker(d.cfg.RefreshInterval))

	case tea.KeyMsg:
		if d.filtering {
			switch msg.String() {
			case "enter", "esc":
				d.filtering = false
				d.filter = d.filterBuf.String()
				d.rebuildRows()
			case "backspace":
				s := d.filterBuf.String()
				if len(s) > 0 {
					d.filterBuf.Reset()
					d.filterBuf.WriteString(s[:len(s)-1])
				}
				d.filter = d.filterBuf.String()
				d.rebuildRows()
			case "/":
				// ignore: re-pressing / during filter input is a no-op
			default:
				d.filterBuf.WriteString(msg.String())
				d.filter = d.filterBuf.String()
				d.rebuildRows()
			}
			return d, nil
		}

		switch {
		case key.Matches(msg, d.keys.Filter):
			d.filtering = true
			d.filterBuf.Reset()
			return d, nil
		case key.Matches(msg, d.keys.Clear):
			d.filter = ""
			d.filtering = false
			d.filterBuf.Reset()
			d.rebuildRows()
			return d, nil
		case key.Matches(msg, d.keys.Up):
			d.moveCursor(-1)
			return d, nil
		case key.Matches(msg, d.keys.Down):
			d.moveCursor(1)
			return d, nil
		case key.Matches(msg, d.keys.Top):
			d.cursor = 0
			d.offset = 0
			return d, nil
		case key.Matches(msg, d.keys.Bottom):
			if len(d.rows) > 0 {
				d.cursor = len(d.rows) - 1
				d.clampOffset()
			}
			return d, nil
		}
	}

	return d, nil
}

func (d *Docker) moveCursor(delta int) {
	d.cursor += delta
	if d.cursor < 0 {
		d.cursor = 0
	}
	if d.cursor >= len(d.rows) {
		d.cursor = len(d.rows) - 1
	}
	d.clampOffset()
}

func (d *Docker) clampOffset() {
	visH := d.visibleHeight()
	if d.cursor < d.offset {
		d.offset = d.cursor
	}
	if d.cursor >= d.offset+visH {
		d.offset = d.cursor - visH + 1
	}
	if d.offset < 0 {
		d.offset = 0
	}
}

func (d *Docker) visibleHeight() int {
	h := d.height - 4
	if h < 1 {
		h = 1
	}
	return h
}

func (d *Docker) View() string {
	if !d.loaded {
		return styles.Muted.Render("  Loading docker data…")
	}

	if !d.data.Available {
		return fmt.Sprintf("  %s\n\n  %s",
			styles.Title.Render("Docker"),
			styles.Muted.Render("Docker is not available. Is the daemon running and accessible?"),
		)
	}

	header := fmt.Sprintf("  %s  %s containers",
		styles.Title.Render("Docker"),
		styles.ValueAccent.Render(fmt.Sprintf("%d", len(d.data.Containers))),
	)

	filter := ""
	if d.filtering {
		filter = "\n  Filter: " + styles.ValueAccent.Render(d.filterBuf.String()) + styles.Muted.Render("_")
	} else if d.filter != "" {
		filter = "\n  Filter: " + styles.ValueAccent.Render(d.filter)
	}

	return fmt.Sprintf("%s%s\n%s", header, filter, d.renderTable())
}

func (d *Docker) renderTable() string {
	cols := dockerColWidths(d.width)

	// header
	headerCells := []string{
		styles.TableHeader.Width(cols[0]).Render("ID"),
		styles.TableHeader.Width(cols[1]).Render("Name"),
		styles.TableHeader.Width(cols[2]).Render("Image"),
		styles.TableHeader.Width(cols[3]).Render("State"),
		styles.TableHeader.Width(cols[4]).Render("Ports"),
	}
	lines := []string{lipgloss.JoinHorizontal(lipgloss.Top, headerCells...)}

	// rows
	visH := d.visibleHeight()
	nRows := len(d.rows)
	end := d.offset + visH
	if end > nRows {
		end = nRows
	}
	for i := d.offset; i < end; i++ {
		lines = append(lines, d.renderRow(d.rows[i], cols, i == d.cursor))
	}

	// scroll indicators — replace first/last data row, no extra lines added
	// up: d.offset > 0 means rows above the visible window are hidden.
	// down: last visible row index (offset+visH-1) < last row index.
	up := d.offset > 0
	bottom := d.offset + visH - 1
	down := nRows > 0 && bottom < nRows-1
	if (up || down) && len(lines) > 1 {
		first, last := 1, len(lines)-1
		if up && down && first == last {
			lines[first] = styles.Muted.Render("  ↑/↓ more")
		} else {
			if up {
				lines[first] = styles.Muted.Render("  ↑ more")
			}
			if down {
				lines[last] = styles.Muted.Render("  ↓ more")
			}
		}
	}

	return strings.Join(lines, "\n")
}

func (d *Docker) renderRow(c collectors.ContainerStat, cols [5]int, selected bool) string {
	cell := func(s string, w int) string {
		s = styles.Truncate(s, w)
		if selected {
			return styles.TableSelected.Width(w).MaxWidth(w).Render(s)
		}
		return styles.TableRow.Width(w).MaxWidth(w).Render(s)
	}

	// State cell: badge with colors when not selected; plain label on selected row
	// (selected background fights badge foreground — plain text is cleaner)
	var stateCell string
	if selected {
		stateCell = styles.TableSelected.Width(cols[3]).MaxWidth(cols[3]).Render(dockerStateLabel(c.State))
	} else {
		// Width-only style preserves DockerStateBadge ANSI colors
		stateCell = lipgloss.NewStyle().Width(cols[3]).MaxWidth(cols[3]).Render(
			styles.DockerStateBadge(c.State),
		)
	}

	parts := []string{
		cell(c.ID, cols[0]),
		cell(c.Name, cols[1]),
		cell(c.Image, cols[2]),
		stateCell,
		cell(c.Ports, cols[4]),
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, parts...)
}

func (d *Docker) rebuildRows() {
	d.rows = d.rows[:0]
	for _, c := range d.data.Containers {
		if d.filter != "" {
			needle := strings.ToLower(d.filter)
			hay := strings.ToLower(c.Name + c.Image + c.State)
			if !strings.Contains(hay, needle) {
				continue
			}
		}
		d.rows = append(d.rows, c)
	}
	// keep cursor in range
	if d.cursor >= len(d.rows) {
		d.cursor = len(d.rows) - 1
	}
	if d.cursor < 0 {
		d.cursor = 0
	}
	d.clampOffset()
}

// dockerColWidths returns [5]int column widths for the current terminal width.
// Fixed columns: ID(12)+Name(20)+Image(25)+State(16) = 73.
// Ports column fills the remainder, clamped to [10, 30]; above 120 it grows freely.
func dockerColWidths(w int) [5]int {
	const fixed = 73
	portW := w - fixed
	switch {
	case portW < 10:
		portW = 10
	case portW > 30 && w <= 120:
		portW = 30
	case w > 120:
		portW = w - 77
	}
	return [5]int{12, 20, 25, 16, portW}
}

// dockerStateLabel returns plain icon+text (no ANSI) for use in selected rows.
func dockerStateLabel(state string) string {
	switch state {
	case "running":
		return "● running"
	case "exited":
		return "■ exited"
	case "dead":
		return "■ dead"
	case "paused":
		return "⏸ paused"
	case "restarting":
		return "↻ restarting"
	case "created":
		return "○ created"
	case "removing":
		return "⊗ removing"
	default:
		if state == "" {
			return "? unknown"
		}
		return "? " + state
	}
}

func fetchDockerData(socketPath string) tea.Cmd {
	return func() tea.Msg {
		c := collectors.NewDockerCollector(socketPath)
		raw, err := c.Collect(context.Background())
		if err != nil {
			return dockerDataMsg{err: err}
		}
		return dockerDataMsg{data: raw.(collectors.DockerData)}
	}
}

func tickDocker(interval time.Duration) tea.Cmd {
	if interval <= 0 {
		interval = 3 * time.Second
	}
	return tea.Tick(interval, func(t time.Time) tea.Msg {
		return dockerTickMsg(t)
	})
}
