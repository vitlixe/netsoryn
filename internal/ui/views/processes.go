package views

import (
	"context"
	"fmt"
	"sort"
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

type procTickMsg time.Time
type procDataMsg struct {
	data collectors.ProcessData
	err  error
}

type sortColumn int

const (
	sortCPU sortColumn = iota
	sortMem
	sortPID
	sortName
	numProcSorts
)

var sortNames = [numProcSorts]string{"CPU%", "MEM%", "PID", "Name"}

type Processes struct {
	cfg       *config.Config
	table     table.Model
	data      []collectors.ProcessStat
	err       error
	keys      keys.NavKeyMap
	sortBy    sortColumn
	filter    string
	filtering bool
	filterBuf strings.Builder
	width     int
	height    int
	loaded    bool
}

func NewProcesses(cfg *config.Config) *Processes {
	cols := procColumns(80)
	t := table.New(
		table.WithColumns(cols),
		table.WithFocused(true),
		table.WithHeight(20),
		table.WithStyles(tableStyles()),
	)
	return &Processes{
		cfg:    cfg,
		table:  t,
		keys:   keys.DefaultNavKeyMap(),
		sortBy: sortCPU,
	}
}

func (p *Processes) Init() tea.Cmd {
	return tea.Batch(
		fetchProcData(p.cfg.ProcessLimit),
		tickProc(p.cfg.RefreshInterval),
	)
}

func (p *Processes) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case ContentSizeMsg:
		p.width = msg.Width
		p.height = msg.Height
		p.table.SetHeight(msg.Height - 3)
		p.table.SetColumns(procColumns(msg.Width))
		p.rebuildRows()

	case procDataMsg:
		p.loaded = true
		p.err = msg.err
		if msg.err == nil {
			p.data = msg.data.Processes
			p.sortData()
			p.rebuildRows()
		}

	case procTickMsg:
		return p, tea.Batch(fetchProcData(p.cfg.ProcessLimit), tickProc(p.cfg.RefreshInterval))

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
		case key.Matches(msg, p.keys.SortNext):
			p.sortBy = (p.sortBy + 1) % numProcSorts
			p.sortData()
			p.rebuildRows()
			return p, nil
		case key.Matches(msg, p.keys.Top):
			p.table.GotoTop()
			return p, nil
		case key.Matches(msg, p.keys.Bottom):
			p.table.GotoBottom()
			return p, nil
		}

		var cmd tea.Cmd
		p.table, cmd = p.table.Update(msg)
		return p, cmd
	}

	return p, nil
}

func (p *Processes) View() string {
	if !p.loaded {
		return styles.Muted.Render("  Loading processes…")
	}
	if p.err != nil {
		return styles.ErrorMsg.Render("  Error: " + p.err.Error())
	}

	header := fmt.Sprintf("  %s  Sorting by: %s",
		styles.Title.Render("Processes"),
		styles.ValueAccent.Render(sortNames[p.sortBy]),
	)
	if p.filter != "" {
		header += "  Filter: " + styles.ValueAccent.Render(p.filter)
	}
	if p.filtering {
		header += "  Filter: " + styles.ValueAccent.Render(p.filterBuf.String()) + styles.Muted.Render("_")
	}

	return fmt.Sprintf("%s\n%s", header, p.table.View())
}

func (p *Processes) sortData() {
	data := make([]collectors.ProcessStat, len(p.data))
	copy(data, p.data)

	switch p.sortBy {
	case sortCPU:
		sort.Slice(data, func(i, j int) bool { return data[i].CPUPercent > data[j].CPUPercent })
	case sortMem:
		sort.Slice(data, func(i, j int) bool { return data[i].MemPercent > data[j].MemPercent })
	case sortPID:
		sort.Slice(data, func(i, j int) bool { return data[i].PID < data[j].PID })
	case sortName:
		sort.Slice(data, func(i, j int) bool { return data[i].Name < data[j].Name })
	}

	// Truncate after sorting so the limit applies to the chosen column.
	if p.cfg.ProcessLimit > 0 && len(data) > p.cfg.ProcessLimit {
		data = data[:p.cfg.ProcessLimit]
	}

	p.data = data
}

func (p *Processes) rebuildRows() {
	rows := make([]table.Row, 0, len(p.data))
	for _, proc := range p.data {
		if p.filter != "" && !strings.Contains(strings.ToLower(proc.Name), strings.ToLower(p.filter)) {
			continue
		}
		rows = append(rows, table.Row{
			fmt.Sprintf("%d", proc.PID),
			styles.Truncate(proc.Name, 20),
			fmt.Sprintf("%.1f", proc.CPUPercent),
			fmt.Sprintf("%.1f", proc.MemPercent),
			fmtBytes(proc.MemRSS),
			styles.Truncate(proc.Username, 12),
			proc.Status,
			fmt.Sprintf("%d", proc.Threads),
		})
	}
	p.table.SetRows(rows)
}

func procColumns(w int) []table.Column {
	nameW := 20
	if w > 120 {
		nameW = 30
	}
	return []table.Column{
		{Title: "PID", Width: 7},
		{Title: "Name", Width: nameW},
		{Title: "CPU%", Width: 7},
		{Title: "MEM%", Width: 7},
		{Title: "RSS", Width: 9},
		{Title: "User", Width: 12},
		{Title: "Status", Width: 10},
		{Title: "Threads", Width: 7},
	}
}

func fetchProcData(limit int) tea.Cmd {
	return func() tea.Msg {
		c := collectors.NewProcessCollector(limit)
		raw, err := c.Collect(context.Background())
		if err != nil {
			return procDataMsg{err: err}
		}
		return procDataMsg{data: raw.(collectors.ProcessData)}
	}
}

func tickProc(interval time.Duration) tea.Cmd {
	if interval <= 0 {
		interval = 2 * time.Second
	}
	return tea.Tick(interval, func(t time.Time) tea.Msg {
		return procTickMsg(t)
	})
}
