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
	halfW := w/2 - 2

	left := lipgloss.JoinVertical(lipgloss.Left,
		d.renderCPU(halfW),
		"",
		d.renderMemory(halfW),
		"",
		d.renderLoad(halfW),
	)

	right := lipgloss.JoinVertical(lipgloss.Left,
		d.renderDisks(halfW),
	)

	return lipgloss.JoinHorizontal(lipgloss.Top,
		lipgloss.NewStyle().Width(halfW+2).Render(left),
		lipgloss.NewStyle().Width(halfW+2).Render(right),
	)
}

func (d *Dashboard) renderCPU(w int) string {
	barW := w - 14
	if barW < 10 {
		barW = 10
	}

	lines := []string{
		styles.Title.Render("  CPU"),
	}

	totalLine := fmt.Sprintf("  %-8s %s %5.1f%%",
		"Total",
		styles.ProgressBar(d.data.CPUTotal, barW),
		d.data.CPUTotal,
	)
	lines = append(lines, totalLine)

	for i, pct := range d.data.CPUPercents {
		if i >= 8 {
			lines = append(lines, styles.Muted.Render(fmt.Sprintf("  … +%d cores", len(d.data.CPUPercents)-8)))
			break
		}
		line := fmt.Sprintf("  %-8s %s %5.1f%%",
			fmt.Sprintf("Core %-2d", i),
			styles.ProgressBar(pct, barW),
			pct,
		)
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}

func (d *Dashboard) renderMemory(w int) string {
	barW := w - 14
	if barW < 10 {
		barW = 10
	}

	memLine := fmt.Sprintf("  %-8s %s %5.1f%%  %s / %s",
		"RAM",
		styles.ProgressBar(d.data.MemPercent, barW),
		d.data.MemPercent,
		fmtBytes(d.data.MemUsed),
		fmtBytes(d.data.MemTotal),
	)

	swapLine := fmt.Sprintf("  %-8s %s %5.1f%%  %s / %s",
		"Swap",
		styles.ProgressBar(d.data.SwapPercent, barW),
		d.data.SwapPercent,
		fmtBytes(d.data.SwapUsed),
		fmtBytes(d.data.SwapTotal),
	)

	return strings.Join([]string{
		styles.Title.Render("  Memory"),
		memLine,
		swapLine,
	}, "\n")
}

func (d *Dashboard) renderLoad(w int) string {
	uptime := fmtUptime(d.data.UptimeSeconds)

	return strings.Join([]string{
		styles.Title.Render("  System"),
		fmt.Sprintf("  Load avg  %s  %s  %s",
			styles.ValueAccent.Render(fmt.Sprintf("%.2f", d.data.LoadAvg1)),
			styles.ValueAccent.Render(fmt.Sprintf("%.2f", d.data.LoadAvg5)),
			styles.ValueAccent.Render(fmt.Sprintf("%.2f", d.data.LoadAvg15)),
		),
		fmt.Sprintf("  Uptime    %s", styles.Muted.Render(uptime)),
		fmt.Sprintf("  Host      %s", styles.ValueAccent.Render(d.data.Hostname)),
	}, "\n")
}

func (d *Dashboard) renderDisks(w int) string {
	barW := w - 22
	if barW < 10 {
		barW = 10
	}

	lines := []string{styles.Title.Render("  Disk")}
	for _, disk := range d.data.Disks {
		mp := styles.Truncate(disk.Mountpoint, 12)
		line := fmt.Sprintf("  %-12s %s %5.1f%%  %s / %s",
			mp,
			styles.ProgressBar(disk.UsedPercent, barW),
			disk.UsedPercent,
			fmtBytes(disk.Used),
			fmtBytes(disk.Total),
		)
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
		return dashDataMsg{data: raw.(collectors.SystemData)}
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
