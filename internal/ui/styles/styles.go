package styles

import (
	"fmt"

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

// ProgressBar renders a colored progress bar.
func ProgressBar(percent float64, width int) string {
	if width <= 0 {
		width = 20
	}
	filled := int(float64(width) * percent / 100)
	if filled > width {
		filled = width
	}
	empty := width - filled

	bar := ""
	for i := 0; i < filled; i++ {
		bar += "█"
	}
	for i := 0; i < empty; i++ {
		bar += "░"
	}

	color := ColorGreen
	switch {
	case percent >= 90:
		color = ColorRed
	case percent >= 70:
		color = ColorOrange
	case percent >= 50:
		color = ColorYellow
	}

	return lipgloss.NewStyle().Foreground(color).Render(bar)
}

// ServiceBadge returns a styled status badge for a service state.
func ServiceBadge(state string) string {
	switch state {
	case "active", "running":
		return BadgeRunning.Render("● " + state)
	case "failed":
		return BadgeFailed.Render("✖ " + state)
	case "inactive", "stopped":
		return BadgeStopped.Render("○ " + state)
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

// HTTPStatusBadge returns a styled badge for an HTTP status code.
func HTTPStatusBadge(code int) string {
	text := ""
	if code > 0 {
		text = lipgloss.NewStyle().Render("")
	}
	switch {
	case code >= 500:
		return BadgeFailed.Render(text)
	case code >= 400:
		return BadgeWarning.Render(text)
	case code >= 200 && code < 300:
		return BadgeRunning.Render(text)
	default:
		return Muted.Render("?")
	}
}

// Truncate truncates a string to n runes.
func Truncate(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n-1]) + "…"
}

// FormatBytes formats bytes to human-readable form.
func FormatBytes(b uint64) string {
	const unit = 1024
	if b < unit {
		return lipgloss.NewStyle().Render(fmt.Sprintf("%d B", b))
	}
	div, exp := uint64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	units := []string{"KB", "MB", "GB", "TB"}
	return lipgloss.NewStyle().Render(fmt.Sprintf("%.1f %s", float64(b)/float64(div), units[exp]))
}
