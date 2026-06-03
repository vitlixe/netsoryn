package styles

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Color palette
const (
	ColorAccent    = lipgloss.Color("#FF7700") // orange
	ColorBlue      = lipgloss.Color("#FFAA33") // amber
	ColorGreen     = lipgloss.Color("#00FF87")
	ColorYellow    = lipgloss.Color("#FFDD55")
	ColorRed       = lipgloss.Color("#FF5F5F")
	ColorOrange    = lipgloss.Color("#FF9933")
	ColorGray      = lipgloss.Color("#6B5A4A")
	ColorLightGray = lipgloss.Color("#9E8A78")
	ColorWhite     = lipgloss.Color("#F0E8DF")
	ColorDarkBg    = lipgloss.Color("#130C04")
	ColorHeaderBg  = lipgloss.Color("#1E1108")
	ColorTabBg     = lipgloss.Color("#160E05")
	ColorBorder    = lipgloss.Color("#5C3A10")
)

var (
	// Layout
	Header = lipgloss.NewStyle().
		Background(ColorHeaderBg).
		Foreground(ColorWhite).
		Padding(0, 1).
		Bold(true)

	HeaderLogo = lipgloss.NewStyle().
			Background(ColorHeaderBg).
			Foreground(ColorAccent).
			Bold(true).
			Padding(0, 1)

	HeaderInfo = lipgloss.NewStyle().
			Background(ColorHeaderBg).
			Foreground(ColorGray).
			Padding(0, 1)

	TabBar = lipgloss.NewStyle().
		Background(ColorTabBg).
		Padding(0, 0)

	TabActive = lipgloss.NewStyle().
			Background(ColorAccent).
			Foreground(ColorDarkBg).
			Padding(0, 1).
			Bold(true)

	TabInactive = lipgloss.NewStyle().
			Background(ColorTabBg).
			Foreground(ColorGray).
			Padding(0, 1)

	Footer = lipgloss.NewStyle().
		Background(ColorTabBg).
		Foreground(ColorGray).
		Padding(0, 1)

	FooterKey = lipgloss.NewStyle().
			Background(ColorTabBg).
			Foreground(ColorAccent).
			Bold(true)

	ContentArea = lipgloss.NewStyle().
			Padding(0, 0)

	// Borders
	PanelBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorBorder).
			Padding(0, 1)

	// Text styles
	Title = lipgloss.NewStyle().
		Foreground(ColorAccent).
		Bold(true)

	Subtitle = lipgloss.NewStyle().
			Foreground(ColorLightGray).
			Bold(false)

	Label = lipgloss.NewStyle().
		Foreground(ColorGray)

	ValueOK = lipgloss.NewStyle().
		Foreground(ColorGreen)

	ValueWarn = lipgloss.NewStyle().
			Foreground(ColorYellow)

	ValueError = lipgloss.NewStyle().
			Foreground(ColorRed)

	ValueAccent = lipgloss.NewStyle().
			Foreground(ColorAccent)

	Muted = lipgloss.NewStyle().
		Foreground(ColorGray)

	Bold = lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorWhite)

	// Table
	TableHeader = lipgloss.NewStyle().
			Foreground(ColorAccent).
			Bold(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderBottom(true).
			BorderForeground(ColorBorder)

	TableSelected = lipgloss.NewStyle().
			Background(lipgloss.Color("#3D1F00")).
			Foreground(ColorWhite)

	TableRow = lipgloss.NewStyle().
			Foreground(ColorWhite)

	TableRowAlt = lipgloss.NewStyle().
			Foreground(ColorLightGray)

	// Status badges
	BadgeRunning = lipgloss.NewStyle().
			Foreground(ColorGreen).
			Bold(true)

	BadgeStopped = lipgloss.NewStyle().
			Foreground(ColorGray)

	BadgeFailed = lipgloss.NewStyle().
			Foreground(ColorRed).
			Bold(true)

	BadgeWarning = lipgloss.NewStyle().
			Foreground(ColorYellow).
			Bold(true)

	// Help overlay
	HelpOverlay = lipgloss.NewStyle().
			Background(ColorHeaderBg).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorAccent).
			Padding(1, 2)

	HelpTitle = lipgloss.NewStyle().
			Foreground(ColorAccent).
			Bold(true).
			MarginBottom(1)

	HelpKey = lipgloss.NewStyle().
		Foreground(ColorAccent).
		Width(12)

	HelpDesc = lipgloss.NewStyle().
			Foreground(ColorLightGray)

	// Input
	InputStyle = lipgloss.NewStyle().
			Foreground(ColorAccent).
			Border(lipgloss.NormalBorder()).
			BorderForeground(ColorBorder).
			Padding(0, 1)

	// Error
	ErrorMsg = lipgloss.NewStyle().
			Foreground(ColorRed)
)

// ServiceBadge returns a styled status badge for a service state.
func ServiceBadge(state string) string {
	switch state {
	case "active", "running":
		return BadgeRunning.Render("● " + state)
	case "failed":
		return BadgeFailed.Render("✖ " + state)
	case "inactive", "stopped":
		return BadgeStopped.Render("○ " + state)
	case "", "unknown":
		return Muted.Render("? unknown")
	default:
		return BadgeWarning.Render("◌ " + state)
	}
}

// DockerStateBadge returns a styled badge for Docker container state.
func DockerStateBadge(state string) string {
	switch state {
	case "running":
		return BadgeRunning.Render("● running")
	case "exited":
		return BadgeStopped.Render("■ exited")
	case "dead":
		return BadgeFailed.Render("■ dead")
	case "paused":
		return BadgeWarning.Render("⏸ paused")
	case "restarting":
		return BadgeWarning.Render("↻ restarting")
	case "created":
		return Muted.Render("○ created")
	case "removing":
		return BadgeWarning.Render("⊗ removing")
	default:
		if state == "" {
			return Muted.Render("? unknown")
		}
		return Muted.Render("? " + state)
	}
}

// Truncate shortens s so its visual width is at most n columns, appending "…"
// when content is dropped. It measures visual width (not rune count) so wide
// runes such as CJK glyphs or emoji do not overflow a fixed-width table column.
// n <= 0 yields the empty string.
func Truncate(s string, n int) string {
	if n <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= n {
		return s
	}
	if n == 1 {
		return "…"
	}
	// Reserve one column for the ellipsis, then keep whole runes that fit.
	limit := n - 1
	width := 0
	var b strings.Builder
	for _, r := range s {
		rw := lipgloss.Width(string(r))
		if width+rw > limit {
			break
		}
		b.WriteRune(r)
		width += rw
	}
	return b.String() + "…"
}
