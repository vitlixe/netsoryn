package views

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/vitlixe/netsoryn/internal/collectors"
	"github.com/vitlixe/netsoryn/internal/config"
	"github.com/vitlixe/netsoryn/internal/ui/styles"
)

type tcpQueryMsg struct {
	result collectors.TCPResult
}

type tcpDataMsg struct {
	results []collectors.TCPResult
}

// TCP is an interactive "can I reach host:port" probe, sitting alongside the
// DNS and HTTP checkers. It has no periodic collection — every result is an
// on-demand connect the user triggers.
type TCP struct {
	ctx       context.Context
	checks    []config.TCPCheck
	results   []collectors.TCPResult
	input     textinput.Model
	inputMode bool
	offset    int
	width     int
	height    int
}

func NewTCP(cfg *config.Config, ctx context.Context) *TCP {
	ti := textinput.New()
	ti.Placeholder = "host:port"
	ti.CharLimit = 255
	ti.Width = 40
	return &TCP{ctx: ctx, input: ti, checks: cfg.TCPChecks}
}

// Init probes any tcp_checks configured in the config file so their results are
// ready when the user first opens the view.
func (t *TCP) Init() tea.Cmd {
	if len(t.checks) == 0 {
		return nil
	}
	return fetchTCPData(t.ctx, t.checks)
}

// CapturingInput reports whether the target text field is consuming key input.
func (t *TCP) CapturingInput() bool { return t.inputMode }

func (t *TCP) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case ContentSizeMsg:
		t.width = msg.Width
		t.height = msg.Height

	case tcpQueryMsg:
		t.inputMode = false
		t.results = append([]collectors.TCPResult{msg.result}, t.results...)
		t.offset = 0

	case tcpDataMsg:
		t.results = append(msg.results, t.results...)
		t.clampOffset()

	case tea.KeyMsg:
		if t.inputMode {
			switch msg.String() {
			case "esc":
				t.inputMode = false
				t.input.Blur()
				return t, nil
			case "enter":
				target := strings.TrimSpace(t.input.Value())
				if target != "" {
					t.input.Reset()
					t.input.Blur()
					t.inputMode = false
					return t, checkTCP(t.ctx, target)
				}
				t.inputMode = false
				t.input.Blur()
				return t, nil
			}
			var cmd tea.Cmd
			t.input, cmd = t.input.Update(msg)
			return t, cmd
		}

		switch msg.String() {
		case "/", "i", "n":
			t.inputMode = true
			t.input.Focus()
			return t, textinput.Blink
		case "D", "d":
			if len(t.results) > 0 {
				t.results = t.results[1:]
			}
			t.clampOffset()
		case "j", "down":
			t.scrollResults(1)
		case "k", "up":
			t.scrollResults(-1)
		case "g":
			t.offset = 0
		case "G":
			t.offset = t.maxOffset()
		}
	}

	return t, nil
}

func (t *TCP) maxOffset() int {
	if len(t.results) == 0 {
		return 0
	}
	return len(t.results) - 1
}

func (t *TCP) clampOffset() {
	if t.offset < 0 {
		t.offset = 0
	}
	if t.offset > t.maxOffset() {
		t.offset = t.maxOffset()
	}
}

func (t *TCP) scrollResults(delta int) {
	t.offset += delta
	t.clampOffset()
}

func (t *TCP) View() string {
	header := fmt.Sprintf("  %s  %s",
		styles.Title.Render("TCP Port Checker"),
		styles.Muted.Render("n: new check  j/k: scroll  D: remove first"),
	)

	inputLine := ""
	if t.inputMode {
		inputLine = "\n  " + styles.Label.Render("Target › ") + t.input.View()
	} else {
		inputLine = "\n  " + styles.Muted.Render("Press n to probe a host:port")
	}

	if len(t.results) == 0 {
		return fmt.Sprintf("%s%s\n\n  %s", header, inputLine, styles.Muted.Render("No results yet."))
	}

	blocks := make([]string, len(t.results))
	for i, r := range t.results {
		blocks[i] = t.renderResult(r)
	}
	return renderScrollableList(header+inputLine, blocks, t.offset, t.height)
}

func (t *TCP) renderResult(r collectors.TCPResult) string {
	var badge string
	if r.Open {
		badge = styles.ValueOK.Render("● open")
	} else {
		badge = styles.ValueError.Render("✖ closed")
	}

	line := fmt.Sprintf("  %s  %s  %s",
		badge,
		styles.ValueAccent.Render(styles.Truncate(r.Target, 40)),
		styles.Muted.Render(collectors.FormatElapsed(r.Elapsed)),
	)

	lines := []string{line}
	if r.Error != "" {
		lines = append(lines, "    "+styles.ErrorMsg.Render(r.Error))
	}

	sep := lipgloss.NewStyle().Foreground(styles.ColorBorder).Render(strings.Repeat("─", 50))
	lines = append(lines, "  "+sep)

	return strings.Join(lines, "\n") + "\n"
}

func checkTCP(ctx context.Context, target string) tea.Cmd {
	return func() tea.Msg {
		return tcpQueryMsg{result: collectors.CheckTCP(ctx, target, 5*time.Second)}
	}
}

// fetchTCPData probes every configured target once and returns them as a batch.
func fetchTCPData(ctx context.Context, checks []config.TCPCheck) tea.Cmd {
	return func() tea.Msg {
		results := make([]collectors.TCPResult, 0, len(checks))
		for _, c := range checks {
			results = append(results, collectors.CheckTCP(ctx, c.Target, c.Timeout))
		}
		return tcpDataMsg{results: results}
	}
}
