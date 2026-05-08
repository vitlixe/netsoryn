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

type svcTickMsg time.Time
type svcDataMsg struct {
	data collectors.ServiceData
	err  error
}

type Services struct {
	cfg       *config.Config
	table     table.Model
	data      collectors.ServiceData
	err       error
	keys      keys.NavKeyMap
	filter    string
	filtering bool
	filterBuf strings.Builder
	width     int
	height    int
	loaded    bool
}

func NewServices(cfg *config.Config) *Services {
	t := table.New(
		table.WithColumns(svcColumns(80, "")),
		table.WithFocused(true),
		table.WithHeight(20),
		table.WithStyles(tableStyles()),
	)
	return &Services{
		cfg:   cfg,
		table: t,
		keys:  keys.DefaultNavKeyMap(),
	}
}

func (s *Services) Init() tea.Cmd {
	return tea.Batch(fetchSvcData(), tickSvc(s.cfg.RefreshInterval))
}

func (s *Services) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case ContentSizeMsg:
		s.width = msg.Width
		s.height = msg.Height
		s.table.SetHeight(msg.Height - 3)
		s.table.SetColumns(svcColumns(msg.Width, s.data.Platform))

	case svcDataMsg:
		s.loaded = true
		s.err = msg.err
		if msg.err == nil {
			s.data = msg.data
			s.rebuildRows()
		}

	case svcTickMsg:
		return s, tea.Batch(fetchSvcData(), tickSvc(s.cfg.RefreshInterval))

	case tea.KeyMsg:
		if s.filtering {
			switch msg.String() {
			case "enter", "esc":
				s.filtering = false
				s.filter = s.filterBuf.String()
				s.rebuildRows()
			case "backspace":
				str := s.filterBuf.String()
				if len(str) > 0 {
					s.filterBuf.Reset()
					s.filterBuf.WriteString(str[:len(str)-1])
				}
				s.filter = s.filterBuf.String()
				s.rebuildRows()
			case "/":
				// ignore: re-pressing / during filter input is a no-op
			default:
				s.filterBuf.WriteString(msg.String())
				s.filter = s.filterBuf.String()
				s.rebuildRows()
			}
			return s, nil
		}

		switch {
		case key.Matches(msg, s.keys.Filter):
			s.filtering = true
			s.filterBuf.Reset()
			return s, nil
		case key.Matches(msg, s.keys.Clear):
			s.filter = ""
			s.filtering = false
			s.filterBuf.Reset()
			s.rebuildRows()
			return s, nil
		case key.Matches(msg, s.keys.Top):
			s.table.GotoTop()
			return s, nil
		case key.Matches(msg, s.keys.Bottom):
			s.table.GotoBottom()
			return s, nil
		}

		var cmd tea.Cmd
		s.table, cmd = s.table.Update(msg)
		return s, cmd
	}

	return s, nil
}

func (s *Services) View() string {
	if !s.loaded {
		return styles.Muted.Render("  Loading services…")
	}

	platformNote := ""
	if s.data.Platform == "darwin" {
		platformNote = styles.Muted.Render("  (launchd)")
	} else if s.data.Platform != "linux" {
		return styles.Muted.Render("  Services view not supported on this platform.")
	}

	header := fmt.Sprintf("  %s%s",
		styles.Title.Render("Services"),
		platformNote,
	)

	if s.err != nil {
		header += "\n" + styles.ErrorMsg.Render("  "+s.err.Error())
	}

	filter := ""
	if s.filtering {
		filter = "\n  Filter: " + styles.ValueAccent.Render(s.filterBuf.String()) + styles.Muted.Render("_")
	} else if s.filter != "" {
		filter = "\n  Filter: " + styles.ValueAccent.Render(s.filter)
	}

	return fmt.Sprintf("%s%s\n%s", header, filter, s.table.View())
}

func (s *Services) rebuildRows() {
	s.table.SetColumns(svcColumns(s.width, s.data.Platform))
	rows := make([]table.Row, 0, len(s.data.Services))
	for _, svc := range s.data.Services {
		if s.filter != "" {
			if !strings.Contains(strings.ToLower(svc.Name), strings.ToLower(s.filter)) {
				continue
			}
		}
		status := serviceStatusLabel(serviceDisplayStatus(svc))
		var row table.Row
		if s.data.Platform == "darwin" {
			nameW := s.width - 12
			if nameW < 20 {
				nameW = 20
			}
			row = table.Row{
				styles.Truncate(svc.Name, nameW),
				status,
			}
		} else {
			row = table.Row{
				styles.Truncate(svc.Name, 35),
				status,
				svc.ActiveState,
				styles.Truncate(svc.Description, 40),
			}
		}
		rows = append(rows, row)
	}
	s.table.SetRows(rows)
}

// serviceDisplayStatus picks the most informative status string available.
// SubState is most specific; falls back through ActiveState, LoadState, "unknown".
func serviceDisplayStatus(svc collectors.ServiceStat) string {
	if svc.SubState != "" {
		return svc.SubState
	}
	if svc.ActiveState != "" {
		return svc.ActiveState
	}
	if svc.LoadState != "" {
		return svc.LoadState
	}
	return "unknown"
}

// serviceStatusLabel normalises a raw state string to a short, readable label.
// It is always plain text — no ANSI styling — safe for use in table cells.
func serviceStatusLabel(status string) string {
	switch status {
	case "running", "active":
		return "running"
	case "stopped", "inactive", "dead":
		return "stopped"
	case "failed":
		return "failed"
	case "", "unknown":
		return "unknown"
	default:
		return status
	}
}

// svcColumns returns platform-appropriate table columns.
// Darwin/launchd uses a 2-column layout (Service + State) because launchd
// provides no description and the active/substate distinction is redundant.
// Linux/systemd uses a 4-column layout (Service, Status, Active, Description).
func svcColumns(w int, platform string) []table.Column {
	if platform == "darwin" {
		nameW := w - 12
		if nameW < 20 {
			nameW = 20
		}
		return []table.Column{
			{Title: "Service", Width: nameW},
			{Title: "State", Width: 10},
		}
	}
	descW := 20
	if w > 67 {
		descW = w - 57
	}
	return []table.Column{
		{Title: "Service", Width: 35},
		{Title: "Status", Width: 10},
		{Title: "Active", Width: 10},
		{Title: "Description", Width: descW},
	}
}

func fetchSvcData() tea.Cmd {
	return func() tea.Msg {
		c := collectors.NewServiceCollector()
		raw, err := c.Collect(context.Background())
		if err != nil {
			return svcDataMsg{err: err}
		}
		return svcDataMsg{data: raw.(collectors.ServiceData)}
	}
}

func tickSvc(interval time.Duration) tea.Cmd {
	if interval <= 0 {
		interval = 5 * time.Second
	}
	return tea.Tick(interval, func(t time.Time) tea.Msg {
		return svcTickMsg(t)
	})
}
