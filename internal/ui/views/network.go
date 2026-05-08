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

type netTickMsg time.Time
type netDataMsg struct {
	data collectors.NetworkData
	err  error
}

type netTab int

const (
	netTabIfaces netTab = iota
	netTabConns
)

type Network struct {
	cfg        *config.Config
	ifaceTable table.Model
	connTable  table.Model
	data       collectors.NetworkData
	err        error
	keys       keys.NavKeyMap
	activeTab  netTab
	filter     string
	filtering  bool
	filterBuf  strings.Builder
	width      int
	height     int
	loaded     bool
}

func NewNetwork(cfg *config.Config) *Network {
	return &Network{
		cfg:        cfg,
		ifaceTable: table.New(table.WithColumns(ifaceColumns(80)), table.WithFocused(true), table.WithHeight(10), table.WithStyles(tableStyles())),
		connTable:  table.New(table.WithColumns(connColumns(80)), table.WithFocused(false), table.WithHeight(10), table.WithStyles(tableStyles())),
		keys:       keys.DefaultNavKeyMap(),
	}
}

func (n *Network) Init() tea.Cmd {
	return tea.Batch(fetchNetData(), tickNet(n.cfg.RefreshInterval))
}

func (n *Network) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case ContentSizeMsg:
		n.width = msg.Width
		n.height = msg.Height
		half := (msg.Height - 5) / 2
		n.ifaceTable.SetHeight(half)
		n.connTable.SetHeight(half)
		n.ifaceTable.SetColumns(ifaceColumns(msg.Width))
		n.connTable.SetColumns(connColumns(msg.Width))

	case netDataMsg:
		n.loaded = true
		n.err = msg.err
		if msg.err == nil {
			n.data = msg.data
			n.rebuildIfaceRows()
			n.rebuildConnRows()
		}

	case netTickMsg:
		return n, tea.Batch(fetchNetData(), tickNet(n.cfg.RefreshInterval))

	case tea.KeyMsg:
		if n.filtering {
			switch msg.String() {
			case "enter", "esc":
				n.filtering = false
				n.filter = n.filterBuf.String()
				n.rebuildConnRows()
			case "backspace":
				s := n.filterBuf.String()
				if len(s) > 0 {
					n.filterBuf.Reset()
					n.filterBuf.WriteString(s[:len(s)-1])
				}
				n.filter = n.filterBuf.String()
				n.rebuildConnRows()
			case "/":
				// ignore: re-pressing / during filter input is a no-op
			default:
				n.filterBuf.WriteString(msg.String())
				n.filter = n.filterBuf.String()
				n.rebuildConnRows()
			}
			return n, nil
		}

		switch {
		case key.Matches(msg, n.keys.Filter):
			n.filtering = true
			n.filterBuf.Reset()
			return n, nil
		case key.Matches(msg, n.keys.Clear):
			n.filter = ""
			n.filtering = false
			n.filterBuf.Reset()
			n.rebuildConnRows()
			return n, nil
		case msg.String() == "h":
			if n.activeTab > 0 {
				n.activeTab--
			}
			return n, nil
		case msg.String() == "l":
			if int(n.activeTab) < 1 {
				n.activeTab++
			}
			return n, nil
		}

		var cmd tea.Cmd
		if n.activeTab == netTabIfaces {
			n.ifaceTable, cmd = n.ifaceTable.Update(msg)
		} else {
			n.connTable, cmd = n.connTable.Update(msg)
		}
		return n, cmd
	}

	return n, nil
}

func (n *Network) View() string {
	if !n.loaded {
		return styles.Muted.Render("  Loading network data…")
	}
	if n.err != nil {
		return styles.ErrorMsg.Render("  Error: " + n.err.Error())
	}

	tabIface := styles.TabInactive.Render("Interfaces")
	tabConn := styles.TabInactive.Render("Connections")
	if n.activeTab == netTabIfaces {
		tabIface = styles.TabActive.Render("Interfaces")
	} else {
		tabConn = styles.TabActive.Render("Connections")
	}
	tabBar := fmt.Sprintf("  %s  %s  %s", styles.Title.Render("Network"), tabIface, tabConn)

	hint := styles.Muted.Render("  h/l: switch panel  /: filter")
	if n.filtering {
		hint = "  Filter: " + styles.ValueAccent.Render(n.filterBuf.String()) + styles.Muted.Render("_")
	} else if n.filter != "" {
		hint = "  Filter: " + styles.ValueAccent.Render(n.filter)
	}

	content := n.ifaceTable.View()
	if n.activeTab == netTabConns {
		content = n.connTable.View()
	}

	return fmt.Sprintf("%s\n%s\n%s", tabBar, hint, content)
}

func (n *Network) rebuildIfaceRows() {
	rows := make([]table.Row, 0, len(n.data.Interfaces))
	for _, iface := range n.data.Interfaces {
		addrs := strings.Join(iface.Addresses, ", ")
		flags := strings.Join(iface.Flags, ",")
		rows = append(rows, table.Row{
			iface.Name,
			styles.Truncate(addrs, 30),
			fmtBytes(iface.BytesSent),
			fmtBytes(iface.BytesRecv),
			styles.Truncate(flags, 20),
		})
	}
	n.ifaceTable.SetRows(rows)
}

func (n *Network) rebuildConnRows() {
	rows := make([]table.Row, 0, len(n.data.Connections))
	for _, c := range n.data.Connections {
		if n.filter != "" && !strings.Contains(strings.ToLower(c.LocalAddr+c.ProcessName), strings.ToLower(n.filter)) {
			continue
		}
		pid := ""
		if c.PID > 0 {
			pid = fmt.Sprintf("%d", c.PID)
		}
		rows = append(rows, table.Row{
			c.Protocol,
			styles.Truncate(c.LocalAddr, 22),
			styles.Truncate(c.RemoteAddr, 22),
			c.State,
			pid,
			styles.Truncate(c.ProcessName, 16),
		})
	}
	n.connTable.SetRows(rows)
}

func ifaceColumns(w int) []table.Column {
	_ = w
	return []table.Column{
		{Title: "Interface", Width: 12},
		{Title: "Addresses", Width: 30},
		{Title: "Sent", Width: 10},
		{Title: "Received", Width: 10},
		{Title: "Flags", Width: 20},
	}
}

func connColumns(w int) []table.Column {
	_ = w
	return []table.Column{
		{Title: "Proto", Width: 5},
		{Title: "Local", Width: 22},
		{Title: "Remote", Width: 22},
		{Title: "State", Width: 12},
		{Title: "PID", Width: 7},
		{Title: "Process", Width: 16},
	}
}

func fetchNetData() tea.Cmd {
	return func() tea.Msg {
		c := collectors.NewNetworkCollector()
		raw, err := c.Collect(context.Background())
		if err != nil {
			return netDataMsg{err: err}
		}
		return netDataMsg{data: raw.(collectors.NetworkData)}
	}
}

func tickNet(interval time.Duration) tea.Cmd {
	if interval <= 0 {
		interval = 2 * time.Second
	}
	return tea.Tick(interval, func(t time.Time) tea.Msg {
		return netTickMsg(t)
	})
}
