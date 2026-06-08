package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/vitlixe/netsoryn/internal/ui/styles"
)

const (
	MinWidth  = 98
	MinHeight = 26
)

func tooSmall(w, h int) bool {
	return w < MinWidth || h < MinHeight
}

func renderSizeWarning(w, h int) string {
	bg := styles.ColorDarkBg

	titleStyle := lipgloss.NewStyle().Foreground(styles.ColorAccent).Bold(true).Background(bg)
	labelStyle := lipgloss.NewStyle().Foreground(styles.ColorWhite).Background(bg)
	reqStyle := lipgloss.NewStyle().Foreground(styles.ColorAccent).Background(bg)
	dimColor := func(actual, required int) lipgloss.Color {
		if actual < required {
			return styles.ColorRed
		}
		return styles.ColorGreen
	}

	wColor := dimColor(w, MinWidth)
	hColor := dimColor(h, MinHeight)

	currentLine := labelStyle.Render("Current: ") +
		lipgloss.NewStyle().Foreground(wColor).Background(bg).Render(fmt.Sprintf("%d", w)) +
		labelStyle.Render(" x ") +
		lipgloss.NewStyle().Foreground(hColor).Background(bg).Render(fmt.Sprintf("%d", h))

	requiredLine := labelStyle.Render("Required: ") +
		reqStyle.Render(fmt.Sprintf("%d x %d", MinWidth, MinHeight))

	lines := []string{
		titleStyle.Render("NETSORYN"),
		"",
		labelStyle.Render("Terminal window too small"),
		currentLine,
		requiredLine,
		"",
		labelStyle.Render("Resize the terminal to continue"),
		"",
		shortcutHint("q", "quit", bg),
	}

	content := strings.Join(lines, "\n")

	return lipgloss.Place(
		w, h,
		lipgloss.Center, lipgloss.Center,
		content,
		lipgloss.WithWhitespaceBackground(bg),
	)
}
