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

type dnsTickMsg time.Time
type dnsDataMsg struct {
	results []collectors.DNSResult
	err     error
}
type dnsQueryMsg struct {
	result collectors.DNSResult
}

type DNS struct {
	cfg       *config.Config
	results   []collectors.DNSResult
	input     textinput.Model
	inputMode bool
	err       error
	width     int
	height    int
	loaded    bool
	servers   []string
}

func NewDNS(cfg *config.Config) *DNS {
	ti := textinput.New()
	ti.Placeholder = "domain.com"
	ti.CharLimit = 253
	ti.Width = 40

	servers := []string{"8.8.8.8:53", "1.1.1.1:53"}
	domains := make([]string, 0, len(cfg.DNSChecks))
	for _, c := range cfg.DNSChecks {
		domains = append(domains, c.Domain)
		if len(c.Servers) > 0 {
			servers = c.Servers
		}
	}

	d := &DNS{
		cfg:     cfg,
		input:   ti,
		servers: servers,
	}

	if len(domains) > 0 {
		d.loaded = true
	}

	return d
}

func (d *DNS) Init() tea.Cmd {
	if len(d.cfg.DNSChecks) == 0 {
		d.loaded = true
		return nil
	}

	domains := make([]string, 0, len(d.cfg.DNSChecks))
	for _, c := range d.cfg.DNSChecks {
		domains = append(domains, c.Domain)
	}
	return fetchDNSData(domains, d.servers)
}

func (d *DNS) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case ContentSizeMsg:
		d.width = msg.Width
		d.height = msg.Height

	case dnsDataMsg:
		d.loaded = true
		d.err = msg.err
		if msg.err == nil {
			d.results = msg.results
		}

	case dnsQueryMsg:
		d.loaded = true
		d.inputMode = false
		// prepend the live result
		d.results = append([]collectors.DNSResult{msg.result}, d.results...)

	case dnsTickMsg:
		if len(d.cfg.DNSChecks) > 0 {
			domains := make([]string, 0, len(d.cfg.DNSChecks))
			for _, c := range d.cfg.DNSChecks {
				domains = append(domains, c.Domain)
			}
			return d, tea.Batch(fetchDNSData(domains, d.servers), tickDNS())
		}

	case tea.KeyMsg:
		if d.inputMode {
			switch msg.String() {
			case "esc":
				d.inputMode = false
				d.input.Blur()
				return d, nil
			case "enter":
				domain := strings.TrimSpace(d.input.Value())
				if domain != "" {
					d.input.Reset()
					d.input.Blur()
					d.inputMode = false
					return d, queryDNS(domain, d.servers)
				}
				d.inputMode = false
				d.input.Blur()
				return d, nil
			}
			var cmd tea.Cmd
			d.input, cmd = d.input.Update(msg)
			return d, cmd
		}

		switch msg.String() {
		case "/", "i", "n":
			d.inputMode = true
			d.input.Focus()
			return d, textinput.Blink
		case "D", "d":
			if len(d.results) > 0 {
				d.results = d.results[1:]
			}
		}
	}

	return d, nil
}

func (d *DNS) View() string {
	header := fmt.Sprintf("  %s  %s",
		styles.Title.Render("DNS Checker"),
		styles.Muted.Render("n: new query  D: remove first"),
	)

	inputLine := ""
	if d.inputMode {
		inputLine = "\n  " + styles.Label.Render("Domain › ") + d.input.View()
	} else {
		inputLine = "\n  " + styles.Muted.Render("Press n to resolve a domain")
	}

	if len(d.results) == 0 {
		return fmt.Sprintf("%s%s\n\n  %s", header, inputLine, styles.Muted.Render("No results yet."))
	}

	var sb strings.Builder
	sb.WriteString(header)
	sb.WriteString(inputLine)
	sb.WriteString("\n")

	for _, r := range d.results {
		sb.WriteString("\n")
		sb.WriteString(d.renderResult(r))
	}

	return sb.String()
}

func (d *DNS) renderResult(r collectors.DNSResult) string {
	domainLine := fmt.Sprintf("  %s  %s  %s",
		styles.ValueAccent.Render(r.Domain),
		styles.Muted.Render("via "+r.Server),
		styles.Muted.Render(fmt.Sprintf("(%s)", r.Elapsed.Round(time.Millisecond))),
	)

	if r.Error != "" {
		return fmt.Sprintf("%s\n  %s\n", domainLine, styles.ErrorMsg.Render("✖ "+r.Error))
	}

	lines := []string{domainLine}

	if len(r.ARecords) > 0 {
		lines = append(lines, fmt.Sprintf("    %s  %s", styles.Label.Render("A    "), strings.Join(r.ARecords, ", ")))
	}
	if len(r.AAAARecords) > 0 {
		lines = append(lines, fmt.Sprintf("    %s  %s", styles.Label.Render("AAAA "), strings.Join(r.AAAARecords, ", ")))
	}
	if len(r.MXRecords) > 0 {
		lines = append(lines, fmt.Sprintf("    %s  %s", styles.Label.Render("MX   "), strings.Join(r.MXRecords, ", ")))
	}
	if len(r.NSRecords) > 0 {
		lines = append(lines, fmt.Sprintf("    %s  %s", styles.Label.Render("NS   "), strings.Join(r.NSRecords, ", ")))
	}
	if r.CNAMERecord != "" {
		lines = append(lines, fmt.Sprintf("    %s  %s", styles.Label.Render("CNAME"), r.CNAMERecord))
	}

	sep := lipgloss.NewStyle().Foreground(styles.ColorBorder).Render(strings.Repeat("─", 50))
	lines = append(lines, "  "+sep)

	return strings.Join(lines, "\n") + "\n"
}

func fetchDNSData(domains, servers []string) tea.Cmd {
	// TODO: replace context.Background() with a cancellable context from RootModel.
	// DNS dialer already has a 5s timeout, so goroutines are bounded.
	return func() tea.Msg {
		c := collectors.NewDNSCollector(domains, servers)
		raw, err := c.Collect(context.Background())
		if err != nil {
			return dnsDataMsg{err: err}
		}
		results, ok := raw.([]collectors.DNSResult)
		if !ok {
			return dnsDataMsg{err: fmt.Errorf("unexpected collector result type %T", raw)}
		}
		return dnsDataMsg{results: results}
	}
}

func queryDNS(domain string, servers []string) tea.Cmd {
	// TODO: same as fetchDNSData — replace with cancellable context.
	return func() tea.Msg {
		result := collectors.ResolveOnce(context.Background(), domain, servers)
		return dnsQueryMsg{result: result}
	}
}

func tickDNS() tea.Cmd {
	return tea.Tick(30*time.Second, func(t time.Time) tea.Msg {
		return dnsTickMsg(t)
	})
}
