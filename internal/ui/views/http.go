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

type httpTickMsg time.Time
type httpDataMsg struct {
	results []collectors.HTTPResult
}
type httpQueryMsg struct {
	result collectors.HTTPResult
}

type HTTP struct {
	cfg       *config.Config
	ctx       context.Context
	results   []collectors.HTTPResult
	input     textinput.Model
	inputMode bool
	offset    int
	width     int
	height    int
	loaded    bool
}

func NewHTTP(cfg *config.Config, ctx context.Context) *HTTP {
	ti := textinput.New()
	ti.Placeholder = "https://example.com"
	ti.CharLimit = 500
	ti.Width = 50

	h := &HTTP{cfg: cfg, ctx: ctx, input: ti}

	if len(cfg.HTTPChecks) > 0 {
		h.loaded = true
	}

	return h
}

func (h *HTTP) Init() tea.Cmd {
	if len(h.cfg.HTTPChecks) == 0 {
		h.loaded = true
		return nil
	}

	return tea.Batch(fetchHTTPData(h.ctx, h.cfg.HTTPChecks), tickHTTP())
}

func (h *HTTP) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case ContentSizeMsg:
		h.width = msg.Width
		h.height = msg.Height

	case httpDataMsg:
		h.loaded = true
		h.results = msg.results
		h.clampOffset()

	case httpQueryMsg:
		h.loaded = true
		h.inputMode = false
		h.results = append([]collectors.HTTPResult{msg.result}, h.results...)
		h.offset = 0

	case httpTickMsg:
		if len(h.cfg.HTTPChecks) > 0 {
			return h, tea.Batch(fetchHTTPData(h.ctx, h.cfg.HTTPChecks), tickHTTP())
		}

	case tea.KeyMsg:
		if h.inputMode {
			switch msg.String() {
			case "esc":
				h.inputMode = false
				h.input.Blur()
				return h, nil
			case "enter":
				url := strings.TrimSpace(h.input.Value())
				if url != "" {
					url = normalizeURL(url)
					h.input.Reset()
					h.input.Blur()
					h.inputMode = false
					return h, checkHTTP(h.ctx, url, 10*time.Second)
				}
				h.inputMode = false
				h.input.Blur()
				return h, nil
			}
			var cmd tea.Cmd
			h.input, cmd = h.input.Update(msg)
			return h, cmd
		}

		switch msg.String() {
		case "/", "i", "n":
			h.inputMode = true
			h.input.Focus()
			return h, textinput.Blink
		case "D", "d":
			if len(h.results) > 0 {
				h.results = h.results[1:]
			}
			h.clampOffset()
		case "j", "down":
			h.scrollResults(1)
		case "k", "up":
			h.scrollResults(-1)
		case "g":
			h.offset = 0
		case "G":
			h.offset = h.maxOffset()
		}
	}

	return h, nil
}

func (h *HTTP) maxOffset() int {
	if len(h.results) == 0 {
		return 0
	}
	return len(h.results) - 1
}

func (h *HTTP) clampOffset() {
	if h.offset < 0 {
		h.offset = 0
	}
	if h.offset > h.maxOffset() {
		h.offset = h.maxOffset()
	}
}

func (h *HTTP) scrollResults(delta int) {
	h.offset += delta
	h.clampOffset()
}

// CapturingInput reports whether the URL text field is consuming key input.
func (h *HTTP) CapturingInput() bool { return h.inputMode }

// normalizeURL prepends https:// when the input lacks an http(s) scheme, so a
// bare host like "httpbin.org" is treated as a URL instead of being rejected
// with an unsupported-scheme error.
func normalizeURL(u string) string {
	if strings.HasPrefix(u, "http://") || strings.HasPrefix(u, "https://") {
		return u
	}
	return "https://" + u
}

func (h *HTTP) View() string {
	header := fmt.Sprintf("  %s  %s",
		styles.Title.Render("HTTP Checker"),
		styles.Muted.Render("n: new check  j/k: scroll  D: remove first"),
	)

	inputLine := ""
	if h.inputMode {
		inputLine = "\n  " + styles.Label.Render("URL › ") + h.input.View()
	} else {
		inputLine = "\n  " + styles.Muted.Render("Press n to check a URL")
	}

	if len(h.results) == 0 {
		return fmt.Sprintf("%s%s\n\n  %s", header, inputLine, styles.Muted.Render("No results yet."))
	}

	blocks := make([]string, len(h.results))
	for i, r := range h.results {
		blocks[i] = h.renderResult(r)
	}
	return renderScrollableList(header+inputLine, blocks, h.offset, h.height)
}

func (h *HTTP) renderResult(r collectors.HTTPResult) string {
	statusBadge := ""
	if r.Error != "" {
		statusBadge = styles.ValueError.Render("✖ ERR")
	} else {
		cat := collectors.StatusColor(r.StatusCode)
		switch cat {
		case "ok":
			statusBadge = styles.ValueOK.Render(fmt.Sprintf("● %d", r.StatusCode))
		case "redirect":
			statusBadge = styles.ValueAccent.Render(fmt.Sprintf("→ %d", r.StatusCode))
		case "warn":
			statusBadge = styles.ValueWarn.Render(fmt.Sprintf("⚠ %d", r.StatusCode))
		case "error":
			statusBadge = styles.ValueError.Render(fmt.Sprintf("✖ %d", r.StatusCode))
		}
	}

	urlLine := fmt.Sprintf("  %s  %s  %s",
		statusBadge,
		styles.ValueAccent.Render(styles.Truncate(r.URL, 60)),
		styles.Muted.Render(collectors.FormatElapsed(r.Elapsed)),
	)

	lines := []string{urlLine}

	if r.Error != "" {
		lines = append(lines, fmt.Sprintf("    %s", styles.ErrorMsg.Render(r.Error)))
	} else {
		if r.Redirect != "" {
			lines = append(lines, fmt.Sprintf("    %s  %s", styles.Label.Render("Redirect "), r.Redirect))
		}
		if r.ContentType != "" {
			lines = append(lines, fmt.Sprintf("    %s  %s", styles.Label.Render("Type     "), r.ContentType))
		}
		if r.TLSValid {
			tlsLine := fmt.Sprintf("    %s  %s  expires %s  issuer %s",
				styles.Label.Render("TLS      "),
				styles.ValueOK.Render("✓ valid"),
				r.TLSExpiry,
				styles.Muted.Render(r.TLSIssuer),
			)
			lines = append(lines, tlsLine)
		} else if strings.HasPrefix(r.URL, "https://") {
			lines = append(lines, fmt.Sprintf("    %s  %s", styles.Label.Render("TLS      "), styles.ValueError.Render("✖ invalid")))
		}
	}

	sep := lipgloss.NewStyle().Foreground(styles.ColorBorder).Render(strings.Repeat("─", 60))
	lines = append(lines, "  "+sep)

	return strings.Join(lines, "\n") + "\n"
}

func fetchHTTPData(ctx context.Context, checks []config.HTTPCheck) tea.Cmd {
	return func() tea.Msg {
		results := make([]collectors.HTTPResult, 0, len(checks))
		for _, check := range checks {
			results = append(results, collectors.CheckOnce(ctx, check.URL, check.Timeout))
		}
		return httpDataMsg{results: results}
	}
}

func checkHTTP(ctx context.Context, url string, timeout time.Duration) tea.Cmd {
	return func() tea.Msg {
		result := collectors.CheckOnce(ctx, url, timeout)
		return httpQueryMsg{result: result}
	}
}

func tickHTTP() tea.Cmd {
	return tea.Tick(30*time.Second, func(t time.Time) tea.Msg {
		return httpTickMsg(t)
	})
}
