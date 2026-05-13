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
	err     error
}
type httpQueryMsg struct {
	result collectors.HTTPResult
}

type HTTP struct {
	cfg       *config.Config
	results   []collectors.HTTPResult
	input     textinput.Model
	inputMode bool
	err       error
	width     int
	height    int
	loaded    bool
}

func NewHTTP(cfg *config.Config) *HTTP {
	ti := textinput.New()
	ti.Placeholder = "https://example.com"
	ti.CharLimit = 500
	ti.Width = 50

	h := &HTTP{cfg: cfg, input: ti}

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

	urls := make([]string, 0, len(h.cfg.HTTPChecks))
	for _, c := range h.cfg.HTTPChecks {
		urls = append(urls, c.URL)
	}
	return fetchHTTPData(urls, 10*time.Second)
}

func (h *HTTP) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case ContentSizeMsg:
		h.width = msg.Width
		h.height = msg.Height

	case httpDataMsg:
		h.loaded = true
		h.err = msg.err
		if msg.err == nil {
			h.results = msg.results
		}

	case httpQueryMsg:
		h.loaded = true
		h.inputMode = false
		h.results = append([]collectors.HTTPResult{msg.result}, h.results...)

	case httpTickMsg:
		if len(h.cfg.HTTPChecks) > 0 {
			urls := make([]string, 0, len(h.cfg.HTTPChecks))
			for _, c := range h.cfg.HTTPChecks {
				urls = append(urls, c.URL)
			}
			return h, tea.Batch(fetchHTTPData(urls, 10*time.Second), tickHTTP())
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
					if !strings.HasPrefix(url, "http") {
						url = "https://" + url
					}
					h.input.Reset()
					h.input.Blur()
					h.inputMode = false
					return h, checkHTTP(url, 10*time.Second)
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
		}
	}

	return h, nil
}

func (h *HTTP) View() string {
	header := fmt.Sprintf("  %s  %s",
		styles.Title.Render("HTTP Checker"),
		styles.Muted.Render("n: new check  D: remove first"),
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

	var sb strings.Builder
	sb.WriteString(header)
	sb.WriteString(inputLine)
	sb.WriteString("\n")

	for _, r := range h.results {
		sb.WriteString("\n")
		sb.WriteString(h.renderResult(r))
	}

	return sb.String()
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

func fetchHTTPData(urls []string, timeout time.Duration) tea.Cmd {
	// TODO: replace context.Background() with a cancellable context from RootModel
	// so in-flight requests are cancelled on quit. Requires passing ctx through
	// model hierarchy — deferred to avoid architectural refactor pre-release.
	// Goroutines are bounded by the HTTP client Timeout, so no leak.
	return func() tea.Msg {
		c := collectors.NewHTTPCollector(urls, timeout)
		raw, err := c.Collect(context.Background())
		if err != nil {
			return httpDataMsg{err: err}
		}
		results, ok := raw.([]collectors.HTTPResult)
		if !ok {
			return httpDataMsg{err: fmt.Errorf("unexpected collector result type %T", raw)}
		}
		return httpDataMsg{results: results}
	}
}

func checkHTTP(url string, timeout time.Duration) tea.Cmd {
	// TODO: same as fetchHTTPData — replace with cancellable context.
	return func() tea.Msg {
		result := collectors.CheckOnce(context.Background(), url, timeout)
		return httpQueryMsg{result: result}
	}
}

func tickHTTP() tea.Cmd {
	return tea.Tick(30*time.Second, func(t time.Time) tea.Msg {
		return httpTickMsg(t)
	})
}
