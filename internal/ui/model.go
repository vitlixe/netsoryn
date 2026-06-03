package ui

import (
	"context"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/vitlixe/netsoryn/internal/config"
	"github.com/vitlixe/netsoryn/internal/ui/keys"
	"github.com/vitlixe/netsoryn/internal/ui/styles"
	"github.com/vitlixe/netsoryn/internal/ui/views"
)

// chromeRows = header(1) + tabbar(1) + border-top(1) + border-bottom(1) + footer(1)
const chromeRows = 5

// ViewID identifies a top-level view.
type ViewID int

const (
	ViewDashboard ViewID = iota
	ViewProcesses
	ViewNetwork
	ViewPorts
	ViewServices
	ViewDocker
	ViewDNS
	ViewHTTP
	numViews
)

var viewMeta = [numViews]struct {
	key  string
	name string
	full string
}{
	{"1", "DASH", "Dashboard"},
	{"2", "PROC", "Processes"},
	{"3", "NET", "Network"},
	{"4", "PORTS", "Ports"},
	{"5", "SVC", "Services"},
	{"6", "DOCKER", "Docker"},
	{"7", "DNS", "DNS"},
	{"8", "HTTP", "HTTP"},
}

// RootModel is the top-level bubbletea model.
type RootModel struct {
	cfg        *config.Config
	viewModels []tea.Model
	active     ViewID
	width      int
	height     int
	showHelp   bool
	version    string
	globalKeys keys.GlobalKeyMap
	ctx        context.Context
	cancel     context.CancelFunc
}

// New constructs the RootModel and initialises all child views.
func New(cfg *config.Config, version string) RootModel {
	ctx, cancel := context.WithCancel(context.Background())

	vms := make([]tea.Model, numViews)
	vms[ViewDashboard] = views.NewDashboard(cfg, ctx)
	vms[ViewProcesses] = views.NewProcesses(cfg, ctx)
	vms[ViewNetwork] = views.NewNetwork(cfg, ctx)
	vms[ViewPorts] = views.NewPorts(cfg, ctx)
	vms[ViewServices] = views.NewServices(cfg, ctx)
	vms[ViewDocker] = views.NewDocker(cfg, ctx)
	vms[ViewDNS] = views.NewDNS(cfg, ctx)
	vms[ViewHTTP] = views.NewHTTP(cfg, ctx)

	defaultView := ViewDashboard
	switch strings.ToLower(cfg.DefaultView) {
	case "processes", "proc":
		defaultView = ViewProcesses
	case "network", "net":
		defaultView = ViewNetwork
	case "ports":
		defaultView = ViewPorts
	case "services", "svc":
		defaultView = ViewServices
	case "docker":
		defaultView = ViewDocker
	case "dns":
		defaultView = ViewDNS
	case "http":
		defaultView = ViewHTTP
	}

	// Only the starting view collects on launch; the rest resume when first
	// shown (see switchTo). The returned refresh cmd is unused here because
	// Init fetches for the active view.
	if a, ok := vms[defaultView].(activatable); ok {
		a.SetActive(true)
	}

	return RootModel{
		cfg:        cfg,
		viewModels: vms,
		active:     defaultView,
		version:    version,
		globalKeys: keys.DefaultGlobalKeyMap(),
		ctx:        ctx,
		cancel:     cancel,
	}
}

func (m RootModel) Init() tea.Cmd {
	cmds := make([]tea.Cmd, numViews)
	for i, v := range m.viewModels {
		cmds[i] = v.Init()
	}
	return tea.Batch(cmds...)
}

// inputCapturer is implemented by views that capture raw key input, such as a
// text field or an active filter editor. While the active view reports that it
// is capturing, the root model routes keys to it instead of interpreting them
// as global shortcuts.
type inputCapturer interface {
	CapturingInput() bool
}

// activatable is implemented by views whose data collection should pause while
// they are not the active view, so every collector (process enumeration,
// `docker ps`, service queries, CPU sampling) does not run in the background.
// SetActive(true) returns a command that refreshes the view immediately on
// focus; SetActive(false) suspends fetching until the view is shown again.
type activatable interface {
	SetActive(bool) tea.Cmd
}

// switchTo changes the active view, pausing the previously active view and
// resuming the new one. It returns any refresh command produced on activation.
func (m RootModel) switchTo(v ViewID) (RootModel, tea.Cmd) {
	if v == m.active {
		return m, nil
	}
	var cmds []tea.Cmd
	if a, ok := m.viewModels[m.active].(activatable); ok {
		if cmd := a.SetActive(false); cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	m.active = v
	if a, ok := m.viewModels[m.active].(activatable); ok {
		if cmd := a.SetActive(true); cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	return m, tea.Batch(cmds...)
}

// ── Update ────────────────────────────────────────────────────────────────────

func (m RootModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if tooSmall(m.width, m.height) {
			return m, nil
		}
		// inner content = total minus chrome rows minus border sides
		innerW := msg.Width - 2
		innerH := msg.Height - chromeRows
		if innerW < 1 {
			innerW = 1
		}
		if innerH < 1 {
			innerH = 1
		}
		sizeMsg := views.ContentSizeMsg{Width: innerW, Height: innerH}
		cmds := make([]tea.Cmd, numViews)
		for i, v := range m.viewModels {
			updated, cmd := v.Update(sizeMsg)
			m.viewModels[i] = updated
			cmds[i] = cmd
		}
		return m, tea.Batch(cmds...)

	case tea.KeyMsg:
		// Ctrl+C always quits, even while a view is capturing text input.
		if msg.Type == tea.KeyCtrlC {
			m.cancel()
			return m, tea.Quit
		}
		// A view capturing raw input (filter editor or a text field) gets first
		// refusal on all other keys, so digits, q, ?, and tab are not hijacked as
		// global shortcuts mid-entry.
		if c, ok := m.viewModels[m.active].(inputCapturer); ok && c.CapturingInput() {
			var cmd tea.Cmd
			m.viewModels[m.active], cmd = m.viewModels[m.active].Update(msg)
			return m, cmd
		}
		if key.Matches(msg, m.globalKeys.Quit) {
			m.cancel()
			return m, tea.Quit
		}
		if key.Matches(msg, m.globalKeys.Help) {
			m.showHelp = !m.showHelp
			return m, nil
		}
		if m.showHelp {
			if msg.String() == "esc" {
				m.showHelp = false
			}
			return m, nil
		}
		switch msg.String() {
		case "1":
			return m.switchTo(ViewDashboard)
		case "2":
			return m.switchTo(ViewProcesses)
		case "3":
			return m.switchTo(ViewNetwork)
		case "4":
			return m.switchTo(ViewPorts)
		case "5":
			return m.switchTo(ViewServices)
		case "6":
			return m.switchTo(ViewDocker)
		case "7":
			return m.switchTo(ViewDNS)
		case "8":
			return m.switchTo(ViewHTTP)
		}
		if key.Matches(msg, m.globalKeys.Tab) {
			return m.switchTo((m.active + 1) % numViews)
		}
		if key.Matches(msg, m.globalKeys.BackTab) {
			return m.switchTo((m.active - 1 + numViews) % numViews)
		}
		var cmd tea.Cmd
		m.viewModels[m.active], cmd = m.viewModels[m.active].Update(msg)
		return m, cmd

	default:
		cmds := make([]tea.Cmd, numViews)
		for i, v := range m.viewModels {
			updated, cmd := v.Update(msg)
			m.viewModels[i] = updated
			cmds[i] = cmd
		}
		return m, tea.Batch(cmds...)
	}
}

// ── View ──────────────────────────────────────────────────────────────────────

func (m RootModel) View() string {
	if m.width == 0 {
		return "Initialising…"
	}
	if tooSmall(m.width, m.height) {
		return renderSizeWarning(m.width, m.height)
	}

	header := m.renderHeader()
	tabBar := m.renderTabBar()
	footer := m.renderFooter()

	innerH := m.height - chromeRows
	if innerH < 1 {
		innerH = 1
	}

	var innerContent string
	if m.showHelp {
		innerContent = lipgloss.Place(
			m.width-2, innerH,
			lipgloss.Center, lipgloss.Center,
			m.renderHelp(),
			lipgloss.WithWhitespaceBackground(styles.ColorDarkBg),
		)
	} else {
		innerContent = m.viewModels[m.active].View()
	}

	box := m.renderBox(innerContent, innerH)

	return lipgloss.JoinVertical(lipgloss.Left, header, tabBar, box, footer)
}

// renderBox wraps the content in a rounded border with the view name as title.
func (m RootModel) renderBox(content string, innerH int) string {
	brd := lipgloss.NewStyle().Foreground(styles.ColorBorder)
	acc := lipgloss.NewStyle().Foreground(styles.ColorAccent).Bold(true)

	viewTitle := " " + viewMeta[m.active].full + " "
	titleW := lipgloss.Width(viewTitle)

	// top border:  ╭── Title ──────────────╮
	remaining := m.width - 2 - 2 - titleW
	if remaining < 0 {
		remaining = 0
	}
	topBorder := brd.Render("╭──") + acc.Render(viewTitle) + brd.Render(strings.Repeat("─", remaining)+"╮")

	// bottom border: ╰──────────────────────╯
	bottomBorder := brd.Render("╰" + strings.Repeat("─", m.width-2) + "╯")

	// side borders around content lines
	lines := strings.Split(content, "\n")
	for len(lines) < innerH {
		lines = append(lines, "")
	}

	innerW := m.width - 2
	var sb strings.Builder
	sb.WriteString(topBorder)
	sb.WriteByte('\n')
	for i := 0; i < innerH; i++ {
		line := ""
		if i < len(lines) {
			line = lines[i]
		}
		if lipgloss.Width(line) > innerW {
			// Clamp overflowing content so it cannot push past the right border
			// and corrupt the frame. MaxWidth truncates ANSI-aware.
			line = lipgloss.NewStyle().MaxWidth(innerW).Render(line)
		}
		pad := innerW - lipgloss.Width(line)
		if pad < 0 {
			pad = 0
		}
		sb.WriteString(brd.Render("│") + line + strings.Repeat(" ", pad) + brd.Render("│"))
		sb.WriteByte('\n')
	}
	sb.WriteString(bottomBorder)
	return sb.String()
}

// ── Sub-renderers ─────────────────────────────────────────────────────────────

func (m RootModel) renderHeader() string {
	hBg := lipgloss.NewStyle().Background(styles.ColorHeaderBg)

	logo := lipgloss.NewStyle().
		Background(styles.ColorHeaderBg).
		Foreground(styles.ColorAccent).
		Bold(true).
		Padding(0, 1).
		Render("⬡ NETSORYN")

	ver := lipgloss.NewStyle().
		Background(styles.ColorHeaderBg).
		Foreground(styles.ColorGray).
		Padding(0, 1).
		Render(formatVersion(m.version))

	pad := max(0, m.width-lipgloss.Width(logo)-lipgloss.Width(ver))
	fill := hBg.Render(strings.Repeat(" ", pad))

	return logo + fill + ver
}

func formatVersion(version string) string {
	if version == "" || version == "dev" || strings.HasPrefix(version, "v") {
		return version
	}
	return "v" + version
}

func (m RootModel) renderTabBar() string {
	kStyle := lipgloss.NewStyle().Background(styles.ColorTabBg).Foreground(styles.ColorAccent).Bold(true)
	activeK := lipgloss.NewStyle().Background(styles.ColorAccent).Foreground(styles.ColorDarkBg).Bold(true)
	inactiveText := lipgloss.NewStyle().Background(styles.ColorTabBg).Foreground(styles.ColorGray)
	activeText := lipgloss.NewStyle().Background(styles.ColorAccent).Foreground(styles.ColorDarkBg).Bold(true)

	var b strings.Builder
	for i, meta := range viewMeta {
		if ViewID(i) == m.active {
			b.WriteString(activeK.Render(" <" + meta.key + "> "))
			b.WriteString(activeText.Render(meta.name + " "))
		} else {
			b.WriteString(kStyle.Render(" <" + meta.key + "> "))
			b.WriteString(inactiveText.Render(meta.name + " "))
		}
	}

	bar := b.String()
	pad := max(0, m.width-lipgloss.Width(bar))
	bar += lipgloss.NewStyle().Background(styles.ColorTabBg).Render(strings.Repeat(" ", pad))
	return bar
}

// viewsWithNav are views where j/k navigation is active.
var viewsWithNav = map[ViewID]bool{
	ViewProcesses: true,
	ViewNetwork:   true,
	ViewPorts:     true,
	ViewServices:  true,
	ViewDocker:    true,
}

// viewsWithFilter are views where / filter is active.
var viewsWithFilter = map[ViewID]bool{
	ViewProcesses: true,
	ViewNetwork:   true,
	ViewPorts:     true,
	ViewServices:  true,
	ViewDocker:    true,
}

func (m RootModel) renderFooter() string {
	type hint struct{ k, v string }
	var hints []hint
	if viewsWithNav[m.active] {
		hints = append(hints, hint{"j/k", "navigate"})
	}
	if viewsWithFilter[m.active] {
		hints = append(hints, hint{"/", "filter"})
	}
	if m.active == ViewProcesses {
		hints = append(hints, hint{"s", "sort"})
	}
	if m.active == ViewNetwork {
		hints = append(hints, hint{"h/l/←/→", "tabs"})
	}
	if m.active == ViewDNS {
		hints = append(hints, hint{"n", "new query"})
	}
	if m.active == ViewHTTP {
		hints = append(hints, hint{"n", "new check"})
	}
	if m.active == ViewDNS || m.active == ViewHTTP {
		hints = append(hints, hint{"j/k", "scroll"})
	}
	hints = append(hints,
		hint{"<tab>", "next view"},
		hint{"?", "help"},
		hint{"q", "quit"},
	)

	var parts []string
	for _, h := range hints {
		parts = append(parts, shortcutHint(h.k, h.v, styles.ColorTabBg))
	}
	bar := lipgloss.NewStyle().Background(styles.ColorTabBg).Render("  ") +
		strings.Join(parts, lipgloss.NewStyle().Background(styles.ColorTabBg).Render("  "))
	pad := max(0, m.width-lipgloss.Width(bar))
	bar += lipgloss.NewStyle().Background(styles.ColorTabBg).Render(strings.Repeat(" ", pad))
	return bar
}

func (m RootModel) renderHelp() string {
	bg := styles.ColorHeaderBg
	acc := lipgloss.NewStyle().Foreground(styles.ColorAccent).Bold(true).Background(bg)
	dim := lipgloss.NewStyle().Foreground(styles.ColorLightGray).Background(bg)
	sec := lipgloss.NewStyle().Foreground(styles.ColorWhite).Bold(true).Background(bg)
	rule := lipgloss.NewStyle().Foreground(styles.ColorBorder).Background(bg).Render(strings.Repeat("─", 38))

	const keyW = 18
	row := func(k, d string) string {
		rendered := acc.Render(k)
		pad := keyW - lipgloss.Width(rendered)
		if pad < 0 {
			pad = 0
		}
		return rendered + dim.Render(strings.Repeat(" ", pad)+d)
	}

	type entry struct{ k, d string }
	nav := []entry{
		{"<1-8>", "Switch view"},
		{"<tab>", "Next view"},
		{"<shift+tab>", "Prev view"},
		{"<j> / <k>", "Navigate (list/table views)"},
		{"<gg> / <G>", "Top / bottom"},
		{"<ctrl+d/u>", "Page down / up"},
	}
	actions := []entry{
		{"</>", "Filter (list/table views)"},
		{"<s>", "Cycle sort column (Processes)"},
		{"<enter>", "Select / expand"},
		{"<esc>", "Clear filter / close"},
		{"<?>", "Toggle this help"},
		{"<q>", "Quit"},
	}

	var sb strings.Builder
	sb.WriteString(acc.Render("Keyboard Shortcuts") + "\n\n")
	sb.WriteString(sec.Render("Navigation") + "\n")
	sb.WriteString(rule + "\n")
	for _, e := range nav {
		sb.WriteString(row(e.k, e.d) + "\n")
	}
	sb.WriteString("\n")
	sb.WriteString(sec.Render("Actions") + "\n")
	sb.WriteString(rule + "\n")
	for _, e := range actions {
		sb.WriteString(row(e.k, e.d) + "\n")
	}
	sb.WriteString("\n")
	sb.WriteString(rule + "\n")
	sb.WriteString(acc.Render("<esc>") + dim.Render("  close"))

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.ColorAccent).
		Background(styles.ColorHeaderBg).
		Padding(1, 3).
		Render(sb.String())
}

// shortcutHint renders a footer-style key badge followed by muted label text.
// bg is the background color of the surrounding surface (used for the label).
func shortcutHint(k, label string, bg lipgloss.Color) string {
	badge := lipgloss.NewStyle().
		Background(styles.ColorGray).
		Foreground(styles.ColorDarkBg).
		Bold(true).
		Padding(0, 1).
		Render(k)
	text := lipgloss.NewStyle().
		Background(bg).
		Foreground(styles.ColorGray).
		Render(" " + label)
	return badge + text
}
