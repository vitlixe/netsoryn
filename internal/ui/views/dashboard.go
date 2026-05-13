package views

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/vitlixe/netsoryn/internal/collectors"
	"github.com/vitlixe/netsoryn/internal/config"
	"github.com/vitlixe/netsoryn/internal/ui/styles"
)

type dashTickMsg time.Time
type dashDataMsg struct {
	data collectors.SystemData
	err  error
}

type Dashboard struct {
	cfg    *config.Config
	data   collectors.SystemData
	err    error
	width  int
	height int
	loaded bool
}

func NewDashboard(cfg *config.Config) *Dashboard {
	return &Dashboard{cfg: cfg}
}

func (d *Dashboard) Init() tea.Cmd {
	return tea.Batch(
		fetchDashData(),
		tickDash(d.cfg.RefreshInterval),
	)
}

func (d *Dashboard) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case ContentSizeMsg:
		d.width = msg.Width
		d.height = msg.Height

	case dashDataMsg:
		d.loaded = true
		d.data = msg.data
		d.err = msg.err

	case dashTickMsg:
		return d, tea.Batch(fetchDashData(), tickDash(d.cfg.RefreshInterval))
	}
	return d, nil
}

func (d *Dashboard) View() string {
	if !d.loaded {
		return styles.Muted.Render("  Loading system data…")
	}
	if d.err != nil {
		return styles.ErrorMsg.Render("  Error: " + d.err.Error())
	}

	w := d.width
	if w < 40 {
		w = 80
	}

	lay := dashboardLayout(w)
	if !lay.wide {
		return lipgloss.JoinVertical(lipgloss.Left,
			d.renderCPU(w),
			"",
			d.renderMemory(w),
			"",
			d.renderDisks(w),
			"",
			d.renderLoad(w),
		)
	}

	cpuBlock := d.renderCPU(lay.leftW)
	systemBlock := d.renderLoad(lay.leftW) // hostname truncated to fit left column

	left := lipgloss.JoinVertical(lipgloss.Left, cpuBlock, "", systemBlock)
	right := lipgloss.JoinVertical(lipgloss.Left,
		d.renderMemory(lay.rightW),
		"",
		d.renderDisks(lay.rightW),
	)

	gp := strings.Repeat(" ", lay.gap)
	rp := strings.Repeat(" ", rightPad)
	return lipgloss.JoinHorizontal(lipgloss.Top,
		lipgloss.NewStyle().Width(lay.leftW).Render(left),
		gp,
		lipgloss.NewStyle().Width(lay.rightW).Render(right),
		rp,
	)
}

func (d *Dashboard) renderCPU(w int) string {
	labelW, bw := dashSectionLayout(w, 8, 0, 10, 24)

	row := func(label string, pct float64) string {
		return "  " + padRight(label, labelW) + " " +
			dashboardBar(pct, bw) +
			fmt.Sprintf("  %5.1f%%", pct)
	}

	lines := []string{styles.Title.Render("  CPU")}
	lines = append(lines, row("Total", d.data.CPUTotal))
	for i, pct := range d.data.CPUPercents {
		if i >= 8 {
			lines = append(lines, styles.Muted.Render(fmt.Sprintf("  … +%d cores", len(d.data.CPUPercents)-8)))
			break
		}
		lines = append(lines, row(fmt.Sprintf("Core %-2d", i), pct))
	}
	return strings.Join(lines, "\n")
}

func (d *Dashboard) renderMemory(w int) string {
	memSize := fmtBytes(d.data.MemUsed) + " / " + fmtBytes(d.data.MemTotal)
	swapSize := fmtBytes(d.data.SwapUsed) + " / " + fmtBytes(d.data.SwapTotal)

	// Use visual width (not byte length) so multi-byte chars don't skew layout.
	maxSizeW := lipgloss.Width(memSize)
	if sw := lipgloss.Width(swapSize); sw > maxSizeW {
		maxSizeW = sw
	}

	labelW, bw := dashSectionLayout(w, 8, maxSizeW, 10, 24)

	row := func(label string, pct float64, sizeStr string) string {
		return "  " + padRight(label, labelW) + " " +
			dashboardBar(pct, bw) +
			fmt.Sprintf("  %5.1f%%  ", pct) +
			padRight(sizeStr, maxSizeW)
	}

	return strings.Join([]string{
		styles.Title.Render("  Memory"),
		row("RAM", d.data.MemPercent, memSize),
		row("Swap", d.data.SwapPercent, swapSize),
	}, "\n")
}

func (d *Dashboard) renderLoad(w int) string {
	uptime := fmtUptime(d.data.UptimeSeconds)

	// "  Host      " prefix is 12 visual cols; truncate hostname to fit within w.
	const hostPfxW = 12
	hostname := d.data.Hostname
	if w > hostPfxW && lipgloss.Width(hostname) > w-hostPfxW {
		hostname = styles.Truncate(hostname, w-hostPfxW)
	}

	var loadLine string
	if d.data.LoadAvgSupported {
		loadLine = fmt.Sprintf("  Load avg  %s  %s  %s",
			styles.ValueAccent.Render(fmt.Sprintf("%.2f", d.data.LoadAvg1)),
			styles.ValueAccent.Render(fmt.Sprintf("%.2f", d.data.LoadAvg5)),
			styles.ValueAccent.Render(fmt.Sprintf("%.2f", d.data.LoadAvg15)),
		)
	} else {
		loadLine = fmt.Sprintf("  Load avg  %s", styles.Muted.Render("N/A"))
	}

	return strings.Join([]string{
		styles.Title.Render("  System"),
		loadLine,
		fmt.Sprintf("  Uptime    %s", styles.Muted.Render(uptime)),
		fmt.Sprintf("  Host      %s", styles.ValueAccent.Render(hostname)),
	}, "\n")
}

func (d *Dashboard) renderDisks(w int) string {
	// w is the actual column visual width (colW = fullTerminalWidth/2).
	// No offset needed: dashSectionLayout works directly against this value.

	maxSizeW := 1
	for _, disk := range d.data.Disks {
		s := fmtBytes(disk.Used) + " / " + fmtBytes(disk.Total)
		if sw := lipgloss.Width(s); sw > maxSizeW {
			maxSizeW = sw
		}
	}

	labelW, bw := dashSectionLayout(w, 12, maxSizeW, 4, 20)

	lines := []string{styles.Title.Render("  Disk")}
	for _, disk := range d.data.Disks {
		mp := diskMountLabel(disk.Mountpoint, labelW)
		sizeStr := fmtBytes(disk.Used) + " / " + fmtBytes(disk.Total)
		line := "  " + padRight(mp, labelW) + " " +
			dashboardBar(disk.UsedPercent, bw) +
			fmt.Sprintf("  %5.1f%%  ", disk.UsedPercent) +
			padRight(sizeStr, maxSizeW)
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

func fetchDashData() tea.Cmd {
	return func() tea.Msg {
		c := collectors.NewSystemCollector()
		raw, err := c.Collect(context.Background())
		if err != nil {
			return dashDataMsg{err: err}
		}
		data, ok := raw.(collectors.SystemData)
		if !ok {
			return dashDataMsg{err: fmt.Errorf("unexpected collector result type %T", raw)}
		}
		return dashDataMsg{data: data}
	}
}

func tickDash(interval time.Duration) tea.Cmd {
	if interval <= 0 {
		interval = 2 * time.Second
	}
	return tea.Tick(interval, func(t time.Time) tea.Msg {
		return dashTickMsg(t)
	})
}

// cpuAnchorW is the visual width of a single CPU row:
// 2(indent) + 8(label) + 1(sep) + 24(bar, capped) + 2(sep) + 6(pct) = 43.
// Moving it left requires shrinking CPU rows or the bar cap.
const cpuAnchorW = 43

// rightMin is the minimum right-column width that keeps Memory/Disk bars
// at or above their minBar values without line overflow.
const rightMin = 48

// layoutGap is the gap between left and right columns.
const layoutGap = 2

// rightPad is the padding on the right side of the wide layout.
// It keeps the right column away from the terminal border/frame.
// No left padding: the left column starts at the content edge.
const rightPad = 2

type dashLayout struct {
	wide   bool
	leftW  int
	rightW int
	gap    int
}

// dashboardLayout returns column widths for the given inner content width.
// Visual structure: [leftW][gap][rightW][rightPad] = w.
// Wide layout is chosen when rightW >= rightMin; otherwise single-column narrow.
func dashboardLayout(w int) dashLayout {
	rightW := w - cpuAnchorW - layoutGap - rightPad
	if rightW >= rightMin {
		return dashLayout{
			wide:   true,
			leftW:  cpuAnchorW,
			rightW: rightW,
			gap:    layoutGap,
		}
	}
	return dashLayout{wide: false}
}

// dashSectionLayout computes aligned label and bar widths for a dashboard section.
// colW is the actual visual column width passed from View().
// baseLabelW is the minimum label width; it grows modestly at wider terminals.
// sizeW is the max visual width of the trailing size suffix (0 if the section has none).
// The bar is clamped to [minBar, maxBar].
func dashSectionLayout(colW, baseLabelW, sizeW, minBar, maxBar int) (labelW, barW int) {
	// Grow label by 1 for every 16 extra cols beyond the baseline column width (50),
	// capped at +6 to prevent the label from dominating.
	extra := (colW - 50) / 16
	if extra < 0 {
		extra = 0
	}
	if extra > 6 {
		extra = 6
	}
	labelW = baseLabelW + extra

	// overhead: indent(2) + label(labelW) + space(1) + space_after_bar(2) + pct(6)
	//           [+ space(2) + size(sizeW) when a size suffix is present]
	overhead := 11 + labelW
	if sizeW > 0 {
		overhead += 2 + sizeW
	}

	barW = colW - overhead
	if barW < minBar {
		barW = minBar
	}
	if barW > maxBar {
		barW = maxBar
	}
	return labelW, barW
}

// padRight pads s to the given visual width using spaces.
// Unlike fmt.Sprintf("%-ws", s) this correctly handles multi-byte Unicode chars
// such as "…" (3 bytes, 1 visual col).
func padRight(s string, width int) string {
	vw := lipgloss.Width(s)
	if vw >= width {
		return s
	}
	return s + strings.Repeat(" ", width-vw)
}

// diskMountLabel returns a short, readable label for a disk mountpoint.
// It keeps the informative tail rather than truncating from the right.
func diskMountLabel(mountpoint string, labelW int) string {
	if mountpoint == "/" || lipgloss.Width(mountpoint) <= labelW {
		return mountpoint
	}
	// macOS APFS: /System/Volumes/<name> → show <name> directly.
	const apfsPrefix = "/System/Volumes/"
	if strings.HasPrefix(mountpoint, apfsPrefix) {
		tail := mountpoint[len(apfsPrefix):]
		if lipgloss.Width(tail) <= labelW {
			return tail
		}
		return shortenTailPath(tail, labelW)
	}
	return shortenTailPath(mountpoint, labelW)
}

// shortenTailPath keeps as many trailing path segments as fit in maxLen,
// prefixed with "…/". If even the last segment is too long it is hard-truncated.
// Uses visual widths so the "…" glyph (1 col, 3 bytes) is counted correctly.
func shortenTailPath(path string, maxLen int) string {
	const pfx = "…/"
	pfxW := lipgloss.Width(pfx) // 2 visual cols, not len(pfx)=4 bytes
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	for i := 0; i < len(parts); i++ {
		candidate := strings.Join(parts[i:], "/")
		if pfxW+lipgloss.Width(candidate) <= maxLen {
			return pfx + candidate
		}
	}
	// Last segment alone is still too long — hard truncate.
	last := parts[len(parts)-1]
	maxSeg := maxLen - pfxW
	if maxSeg <= 0 {
		runes := []rune(path)
		if len(runes) > maxLen {
			return string(runes[:maxLen])
		}
		return path
	}
	runes := []rune(last)
	if len(runes) > maxSeg {
		return pfx + string(runes[:maxSeg])
	}
	return pfx + last
}

// dashboardBar renders a thin horizontal bar using ━ / ─ so that
// adjacent rows don't merge into a solid block like full-block chars do.
func dashboardBar(percent float64, width int) string {
	filled := int(float64(width) * percent / 100)
	if filled > width {
		filled = width
	}
	if filled < 0 {
		filled = 0
	}
	empty := width - filled

	color := styles.ColorGreen
	switch {
	case percent >= 90:
		color = styles.ColorRed
	case percent >= 70:
		color = styles.ColorOrange
	case percent >= 50:
		color = styles.ColorYellow
	}

	return lipgloss.NewStyle().Foreground(color).Render(strings.Repeat("━", filled)) +
		lipgloss.NewStyle().Foreground(styles.ColorBorder).Render(strings.Repeat("─", empty))
}

func fmtBytes(b uint64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := uint64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %s", float64(b)/float64(div), []string{"KB", "MB", "GB", "TB"}[exp])
}

func fmtUptime(s uint64) string {
	d := s / 86400
	h := (s % 86400) / 3600
	m := (s % 3600) / 60
	if d > 0 {
		return fmt.Sprintf("%dd %dh %dm", d, h, m)
	}
	return fmt.Sprintf("%dh %dm", h, m)
}
