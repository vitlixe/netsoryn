package views

import (
	"fmt"
	"os/exec"
	"sort"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/vitlixe/netsoryn/internal/config"
	"github.com/vitlixe/netsoryn/internal/sshclient"
	"github.com/vitlixe/netsoryn/internal/ui/styles"
)

type sshFinishedMsg struct {
	name   string
	action string
	err    error
}

type SSH struct {
	cfg      *config.Config
	hosts    []config.SSHHost
	cursor   int
	offset   int
	width    int
	height   int
	status   string
	adding   bool
	inputs   []textinput.Model
	focus    int
	formErr  string
	savePath string
}

func NewSSH(cfg *config.Config) *SSH {
	hosts := append([]config.SSHHost(nil), cfg.SSHHosts...)
	sort.SliceStable(hosts, func(i, j int) bool {
		return hosts[i].Name < hosts[j].Name
	})
	return &SSH{cfg: cfg, hosts: hosts, inputs: newSSHInputs()}
}

func (s *SSH) Init() tea.Cmd { return nil }

func (s *SSH) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case ContentSizeMsg:
		s.width = msg.Width
		s.height = msg.Height

	case sshFinishedMsg:
		if msg.err != nil {
			s.status = fmt.Sprintf("%s %s failed: %v", msg.name, msg.action, msg.err)
		} else {
			s.status = fmt.Sprintf("%s %s finished", msg.name, msg.action)
		}

	case tea.KeyMsg:
		if s.adding {
			return s.updateAddForm(msg)
		}
		switch msg.String() {
		case "a":
			s.startAdd()
			return s, textinput.Blink
		case "j", "down":
			s.moveCursor(1)
		case "k", "up":
			s.moveCursor(-1)
		case "g":
			s.cursor = 0
			s.offset = 0
		case "G":
			if len(s.hosts) > 0 {
				s.cursor = len(s.hosts) - 1
				s.clampOffset()
			}
		case "enter":
			host, ok := s.selectedHost()
			if !ok {
				return s, nil
			}
			s.status = "connecting to " + host.Name
			return s, execSSH(host, nil, "session")
		case "d", "D":
			host, ok := s.selectedHost()
			if !ok {
				return s, nil
			}
			s.status = "running dump on " + host.Name
			return s, execSSH(host, sshclient.RemoteDumpCommand("text", "system,ports,services", false), "dump")
		}
	}
	return s, nil
}

func (s *SSH) CapturingInput() bool { return s.adding }

func (s *SSH) updateAddForm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		s.adding = false
		s.formErr = ""
		return s, nil
	case "tab", "down":
		s.focusInput(1)
		return s, nil
	case "shift+tab", "up":
		s.focusInput(-1)
		return s, nil
	case "enter":
		if s.focus < len(s.inputs)-1 {
			s.focusInput(1)
			return s, nil
		}
		s.saveAddForm()
		return s, nil
	}

	var cmd tea.Cmd
	s.inputs[s.focus], cmd = s.inputs[s.focus].Update(msg)
	return s, cmd
}

func (s *SSH) startAdd() {
	s.adding = true
	s.formErr = ""
	s.inputs = newSSHInputs()
	s.focus = 0
	s.inputs[0].Focus()
}

func (s *SSH) focusInput(delta int) {
	if len(s.inputs) == 0 {
		return
	}
	s.inputs[s.focus].Blur()
	s.focus = (s.focus + delta + len(s.inputs)) % len(s.inputs)
	s.inputs[s.focus].Focus()
}

func (s *SSH) saveAddForm() {
	host, err := s.hostFromForm()
	if err != nil {
		s.formErr = err.Error()
		return
	}
	path, err := config.AddSSHHost(s.cfg, host)
	if err != nil {
		s.formErr = err.Error()
		return
	}

	s.hosts = append([]config.SSHHost(nil), s.cfg.SSHHosts...)
	sort.SliceStable(s.hosts, func(i, j int) bool {
		return s.hosts[i].Name < s.hosts[j].Name
	})
	for i, h := range s.hosts {
		if h.Name == host.Name {
			s.cursor = i
			break
		}
	}
	s.clampOffset()
	s.adding = false
	s.formErr = ""
	s.savePath = path
	s.status = fmt.Sprintf("saved %s to %s", host.Name, path)
}

func (s *SSH) hostFromForm() (config.SSHHost, error) {
	port := 22
	portText := strings.TrimSpace(s.inputs[3].Value())
	if portText != "" {
		p, err := strconv.Atoi(portText)
		if err != nil {
			return config.SSHHost{}, fmt.Errorf("port must be a number")
		}
		port = p
	}
	return config.SSHHost{
		Name:    strings.TrimSpace(s.inputs[0].Value()),
		Host:    strings.TrimSpace(s.inputs[1].Value()),
		User:    strings.TrimSpace(s.inputs[2].Value()),
		Port:    port,
		Key:     strings.TrimSpace(s.inputs[4].Value()),
		Options: strings.Fields(s.inputs[5].Value()),
	}, nil
}

func (s *SSH) selectedHost() (config.SSHHost, bool) {
	if s.cursor < 0 || s.cursor >= len(s.hosts) {
		return config.SSHHost{}, false
	}
	return s.hosts[s.cursor], true
}

func (s *SSH) moveCursor(delta int) {
	s.cursor += delta
	if s.cursor < 0 {
		s.cursor = 0
	}
	if s.cursor >= len(s.hosts) {
		s.cursor = len(s.hosts) - 1
	}
	s.clampOffset()
}

func (s *SSH) clampOffset() {
	visH := s.visibleHeight()
	if s.cursor < s.offset {
		s.offset = s.cursor
	}
	if s.cursor >= s.offset+visH {
		s.offset = s.cursor - visH + 1
	}
	if s.offset < 0 {
		s.offset = 0
	}
}

func (s *SSH) visibleHeight() int {
	h := s.height - 5
	if h < 1 {
		h = 1
	}
	return h
}

func (s *SSH) View() string {
	if s.adding {
		return s.renderAddForm()
	}

	header := fmt.Sprintf("  %s  %s",
		styles.Title.Render("SSH Hosts"),
		styles.Muted.Render("a: add  enter: connect  d: dump  j/k: navigate"),
	)

	if len(s.hosts) == 0 {
		return fmt.Sprintf("%s\n\n  %s\n  %s",
			header,
			styles.Muted.Render("No SSH hosts configured."),
			styles.Muted.Render("Press a to add a host."),
		)
	}

	status := ""
	if s.status != "" {
		status = "\n  " + styles.Muted.Render(s.status)
	}

	return fmt.Sprintf("%s%s\n%s", header, status, s.renderTable())
}

func (s *SSH) renderAddForm() string {
	header := fmt.Sprintf("  %s  %s",
		styles.Title.Render("Add SSH Host"),
		styles.Muted.Render("tab: next  shift+tab: prev  enter: next/save  esc: cancel"),
	)
	labels := []string{"Name", "Host", "User", "Port", "Key", "Options"}
	lines := []string{header, ""}
	for i, input := range s.inputs {
		label := styles.Label.Render(fmt.Sprintf("  %-8s ", labels[i]+"›"))
		if i == s.focus {
			label = styles.ValueAccent.Render(fmt.Sprintf("  %-8s ", labels[i]+"›"))
		}
		lines = append(lines, label+input.View())
	}
	if s.formErr != "" {
		lines = append(lines, "", "  "+styles.ErrorMsg.Render(s.formErr))
	}
	return strings.Join(lines, "\n")
}

func (s *SSH) renderTable() string {
	cols := sshColWidths(s.width)

	headerCells := []string{
		styles.TableHeader.Width(cols[0]).Render("Name"),
		styles.TableHeader.Width(cols[1]).Render("Target"),
		styles.TableHeader.Width(cols[2]).Render("Key"),
		styles.TableHeader.Width(cols[3]).Render("Options"),
	}
	lines := []string{lipgloss.JoinHorizontal(lipgloss.Top, headerCells...)}

	visH := s.visibleHeight()
	end := s.offset + visH
	if end > len(s.hosts) {
		end = len(s.hosts)
	}
	for i := s.offset; i < end; i++ {
		lines = append(lines, s.renderRow(s.hosts[i], cols, i == s.cursor))
	}

	up := s.offset > 0
	bottom := s.offset + visH - 1
	down := len(s.hosts) > 0 && bottom < len(s.hosts)-1
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

func (s *SSH) renderRow(host config.SSHHost, cols [4]int, selected bool) string {
	key := host.Key
	if key == "" {
		key = "-"
	}
	opts := "-"
	if len(host.Options) > 0 {
		opts = strings.Join(host.Options, " ")
	}

	cells := []string{
		styles.Truncate(host.Name, cols[0]),
		styles.Truncate(sshTargetLabel(host), cols[1]),
		styles.Truncate(key, cols[2]),
		styles.Truncate(opts, cols[3]),
	}
	for i, cell := range cells {
		cells[i] = lipgloss.NewStyle().Width(cols[i]).Render(cell)
	}
	row := lipgloss.JoinHorizontal(lipgloss.Top, cells...)
	if selected {
		return styles.TableSelected.Render(row)
	}
	return styles.TableRow.Render(row)
}

func sshTargetLabel(host config.SSHHost) string {
	target := sshclient.Target(host)
	if host.Port > 0 && host.Port != 22 {
		target += ":" + strconv.Itoa(host.Port)
	}
	return target
}

func sshColWidths(w int) [4]int {
	if w < 80 {
		w = 80
	}
	nameW := 16
	targetW := 30
	keyW := 22
	optsW := w - nameW - targetW - keyW
	if optsW < 12 {
		optsW = 12
	}
	return [4]int{nameW, targetW, keyW, optsW}
}

func execSSH(host config.SSHHost, remote []string, action string) tea.Cmd {
	args := sshclient.BuildArgs(host, remote)
	cmd := exec.Command("ssh", args...)
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return sshFinishedMsg{name: host.Name, action: action, err: err}
	})
}

func newSSHInputs() []textinput.Model {
	placeholders := []string{
		"prod",
		"prod.example.com",
		"deploy",
		"22",
		"~/.ssh/prod_ed25519",
		"-o StrictHostKeyChecking=accept-new",
	}
	inputs := make([]textinput.Model, len(placeholders))
	for i, placeholder := range placeholders {
		ti := textinput.New()
		ti.Placeholder = placeholder
		ti.CharLimit = 500
		ti.Width = 56
		if i == 0 {
			ti.CharLimit = 80
		}
		if i == 3 {
			ti.CharLimit = 5
			ti.Width = 12
		}
		inputs[i] = ti
	}
	return inputs
}
