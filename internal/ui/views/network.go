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
	cfg         *config.Config
	ctx         context.Context
	ifaceTable  table.Model
	connTable   table.Model
	topRowIface int
	topRowConn  int
	data        collectors.NetworkData
	err         error
	keys        keys.NavKeyMap
	activeTab   netTab
	filter      string
	filtering   bool
	filterBuf   strings.Builder
	width       int
	height      int
	loaded      bool
}

func NewNetwork(cfg *config.Config, ctx context.Context) *Network {
	return &Network{
		cfg:        cfg,
		ctx:        ctx,
		ifaceTable: table.New(table.WithColumns(ifaceColumns(80)), table.WithFocused(true), table.WithHeight(10), table.WithStyles(tableStyles())),
		connTable:  table.New(table.WithColumns(connColumns(80)), table.WithFocused(false), table.WithHeight(10), table.WithStyles(tableStyles())),
		keys:       keys.DefaultNavKeyMap(),
	}
}

func (n *Network) syncFocus() {
	if n.activeTab == netTabIfaces {
		n.ifaceTable.Focus()
		n.connTable.Blur()
	} else {
		n.connTable.Focus()
		n.ifaceTable.Blur()
	}
}

func (n *Network) Init() tea.Cmd {
	return tea.Batch(fetchNetData(n.ctx), tickNet(n.cfg.RefreshInterval))
}

func (n *Network) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case ContentSizeMsg:
		n.width = msg.Width
		n.height = msg.Height
		tableH := msg.Height - 3
		if tableH < 3 {
			tableH = 3
		}
		n.ifaceTable.SetHeight(tableH)
		n.connTable.SetHeight(tableH)
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
		return n, tea.Batch(fetchNetData(n.ctx), tickNet(n.cfg.RefreshInterval))

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
					n.filterBuf.WriteString(trimLastRune(s))
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
		case msg.String() == "h" || msg.Type == tea.KeyLeft:
			if n.activeTab > 0 {
				n.activeTab--
				n.syncFocus()
			}
			return n, nil
		case msg.String() == "l" || msg.Type == tea.KeyRight:
			if int(n.activeTab) < 1 {
				n.activeTab++
				n.syncFocus()
			}
			return n, nil
		}

		var cmd tea.Cmd
		if n.activeTab == netTabIfaces {
			n.ifaceTable, cmd = n.ifaceTable.Update(msg)
			n.topRowIface = syncTopRow(n.topRowIface, n.ifaceTable.Cursor(), n.ifaceTable.Height())
		} else {
			n.connTable, cmd = n.connTable.Update(msg)
			n.topRowConn = syncTopRow(n.topRowConn, n.connTable.Cursor(), n.connTable.Height())
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

	hint := styles.Muted.Render("  h/← l/→: switch panel  /: filter")
	if n.filtering {
		hint = "  Filter: " + styles.ValueAccent.Render(n.filterBuf.String()) + styles.Muted.Render("_")
	} else if n.filter != "" {
		hint = "  Filter: " + styles.ValueAccent.Render(n.filter)
	}

	content := wrapTableWithScrollHints(n.ifaceTable, n.topRowIface)
	if n.activeTab == netTabConns {
		content = wrapTableWithScrollHints(n.connTable, n.topRowConn)
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
	setTableRows(&n.ifaceTable, rows)
	n.topRowIface = clampTopRow(n.topRowIface, len(rows), n.ifaceTable.Height())
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
	setTableRows(&n.connTable, rows)
	n.topRowConn = clampTopRow(n.topRowConn, len(rows), n.connTable.Height())
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
	// Fixed columns: Proto(5)+Local(22)+Remote(22)+State(12)+PID(7) = 68.
	// Process column fills the remainder, clamped to [8, 16].
	const fixed = 68
	processW := w - fixed
	if processW < 8 {
		processW = 8
	} else if processW > 16 {
		processW = 16
	}
	return []table.Column{
		{Title: "Proto", Width: 5},
		{Title: "Local", Width: 22},
		{Title: "Remote", Width: 22},
		{Title: "State", Width: 12},
		{Title: "PID", Width: 7},
		{Title: "Process", Width: processW},
	}
}

func fetchNetData(ctx context.Context) tea.Cmd {
	return func() tea.Msg {
		c := collectors.NewNetworkCollector()
		raw, err := c.Collect(ctx)
		if err != nil {
			return netDataMsg{err: err}
		}
		data, ok := raw.(collectors.NetworkData)
		if !ok {
			return netDataMsg{err: fmt.Errorf("unexpected collector result type %T", raw)}
		}
		return netDataMsg{data: data}
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
