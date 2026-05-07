package views

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"

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
	table     table.Model
	data      collectors.DockerData
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
	t := table.New(
		table.WithColumns(dockerColumns(80)),
		table.WithFocused(true),
		table.WithHeight(20),
		table.WithStyles(tableStyles()),
	)
	return &Docker{
		cfg:   cfg,
		table: t,
		keys:  keys.DefaultNavKeyMap(),
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
		d.table.SetHeight(msg.Height - 4)
		d.table.SetColumns(dockerColumns(msg.Width))

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
		case key.Matches(msg, d.keys.Top):
			d.table.GotoTop()
			return d, nil
		case key.Matches(msg, d.keys.Bottom):
			d.table.GotoBottom()
			return d, nil
		}

		var cmd tea.Cmd
		d.table, cmd = d.table.Update(msg)
		return d, cmd
	}

	return d, nil
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

	return fmt.Sprintf("%s%s\n%s", header, filter, d.table.View())
}

func (d *Docker) rebuildRows() {
	rows := make([]table.Row, 0, len(d.data.Containers))
	for _, c := range d.data.Containers {
		if d.filter != "" {
			needle := strings.ToLower(d.filter)
			hay := strings.ToLower(c.Name + c.Image + c.State)
			if !strings.Contains(hay, needle) {
				continue
			}
		}
		rows = append(rows, table.Row{
			styles.Truncate(c.ID, 12),
			styles.Truncate(c.Name, 20),
			styles.Truncate(c.Image, 25),
			styles.DockerStateBadge(c.State),
			styles.Truncate(c.Ports, 30),
		})
	}
	d.table.SetRows(rows)
}

func dockerColumns(w int) []table.Column {
	portW := 30
	if w > 120 {
		portW = w - 75
	}
	return []table.Column{
		{Title: "ID", Width: 12},
		{Title: "Name", Width: 20},
		{Title: "Image", Width: 25},
		{Title: "State", Width: 14},
		{Title: "Ports", Width: portW},
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
