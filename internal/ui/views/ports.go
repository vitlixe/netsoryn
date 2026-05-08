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

type portsTickMsg time.Time
type portsDataMsg struct {
	data collectors.PortData
	err  error
}

type Ports struct {
	cfg       *config.Config
	table     table.Model
	topRow    int
	data      []collectors.PortStat
	err       error
	keys      keys.NavKeyMap
	filter    string
	filtering bool
	filterBuf strings.Builder
	width     int
	height    int
	loaded    bool
}

func NewPorts(cfg *config.Config) *Ports {
	t := table.New(
		table.WithColumns(portsColumns(80)),
		table.WithFocused(true),
		table.WithHeight(20),
		table.WithStyles(tableStyles()),
	)
	return &Ports{
		cfg:   cfg,
		table: t,
		keys:  keys.DefaultNavKeyMap(),
	}
}

func (p *Ports) Init() tea.Cmd {
	return tea.Batch(
		fetchPortsData(p.cfg.PortsListenOnly),
		tickPorts(p.cfg.RefreshInterval),
	)
}

func (p *Ports) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case ContentSizeMsg:
		p.width = msg.Width
		p.height = msg.Height
		p.table.SetHeight(msg.Height - 3)
		p.table.SetColumns(portsColumns(msg.Width))

	case portsDataMsg:
		p.loaded = true
		p.err = msg.err
		if msg.err == nil {
			p.data = msg.data.Ports
			p.rebuildRows()
		}

	case portsTickMsg:
		return p, tea.Batch(fetchPortsData(p.cfg.PortsListenOnly), tickPorts(p.cfg.RefreshInterval))

	case tea.KeyMsg:
		if p.filtering {
			switch msg.String() {
			case "enter", "esc":
				p.filtering = false
				p.filter = p.filterBuf.String()
				p.rebuildRows()
			case "backspace":
				s := p.filterBuf.String()
				if len(s) > 0 {
					p.filterBuf.Reset()
					p.filterBuf.WriteString(s[:len(s)-1])
				}
				p.filter = p.filterBuf.String()
				p.rebuildRows()
			case "/":
				// ignore: re-pressing / during filter input is a no-op
			default:
				p.filterBuf.WriteString(msg.String())
				p.filter = p.filterBuf.String()
				p.rebuildRows()
			}
			return p, nil
		}

		switch {
		case key.Matches(msg, p.keys.Filter):
			p.filtering = true
			p.filterBuf.Reset()
			return p, nil
		case key.Matches(msg, p.keys.Clear):
			p.filter = ""
			p.filtering = false
			p.filterBuf.Reset()
			p.rebuildRows()
			return p, nil
		case key.Matches(msg, p.keys.Top):
			p.table.GotoTop()
			p.topRow = syncTopRow(p.topRow, p.table.Cursor(), p.table.Height())
			return p, nil
		case key.Matches(msg, p.keys.Bottom):
			p.table.GotoBottom()
			p.topRow = syncTopRow(p.topRow, p.table.Cursor(), p.table.Height())
			return p, nil
		}

		var cmd tea.Cmd
		p.table, cmd = p.table.Update(msg)
		p.topRow = syncTopRow(p.topRow, p.table.Cursor(), p.table.Height())
		return p, cmd
	}

	return p, nil
}

func (p *Ports) View() string {
	if !p.loaded {
		return styles.Muted.Render("  Loading ports…")
	}
	if p.err != nil {
		return styles.ErrorMsg.Render("  Error: " + p.err.Error())
	}

	header := fmt.Sprintf("  %s  %s entries",
		styles.Title.Render("Open Ports"),
		styles.ValueAccent.Render(fmt.Sprintf("%d", p.table.Height())),
	)

	filter := ""
	if p.filtering {
		filter = "\n  Filter: " + styles.ValueAccent.Render(p.filterBuf.String()) + styles.Muted.Render("_")
	} else if p.filter != "" {
		filter = "\n  Filter: " + styles.ValueAccent.Render(p.filter)
	}

	return fmt.Sprintf("%s%s\n%s", header, filter, wrapTableWithScrollHints(p.table, p.topRow))
}

func (p *Ports) rebuildRows() {
	rows := make([]table.Row, 0, len(p.data))
	for _, port := range p.data {
		if p.filter != "" {
			needle := strings.ToLower(p.filter)
			haystack := strings.ToLower(fmt.Sprintf("%d %s %s", port.Port, port.Process, port.Address))
			if !strings.Contains(haystack, needle) {
				continue
			}
		}

		pid := ""
		if port.PID > 0 {
			pid = fmt.Sprintf("%d", port.PID)
		}

		rows = append(rows, table.Row{
			fmt.Sprintf("%d", port.Port),
			port.Protocol,
			port.Address,
			port.State,
			pid,
			styles.Truncate(port.Process, 20),
		})
	}
	setTableRows(&p.table, rows)
	p.topRow = clampTopRow(p.topRow, len(rows), p.table.Height())
}

func portsColumns(w int) []table.Column {
	_ = w
	return []table.Column{
		{Title: "Port", Width: 7},
		{Title: "Proto", Width: 6},
		{Title: "Address", Width: 18},
		{Title: "State", Width: 10},
		{Title: "PID", Width: 7},
		{Title: "Process", Width: 20},
	}
}

func fetchPortsData(listenOnly bool) tea.Cmd {
	return func() tea.Msg {
		c := collectors.NewPortCollector(listenOnly)
		raw, err := c.Collect(context.Background())
		if err != nil {
			return portsDataMsg{err: err}
		}
		return portsDataMsg{data: raw.(collectors.PortData)}
	}
}

func tickPorts(interval time.Duration) tea.Cmd {
	if interval <= 0 {
		interval = 3 * time.Second
	}
	return tea.Tick(interval, func(t time.Time) tea.Msg {
		return portsTickMsg(t)
	})
}
