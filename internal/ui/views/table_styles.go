package views

import (
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"

	"github.com/vitlixe/netsoryn/internal/ui/styles"
)

// tableStyles returns the shared bubbles/table style configuration.
func tableStyles() table.Styles {
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(styles.ColorBorder).
		BorderBottom(true).
		Foreground(styles.ColorAccent).
		Bold(true)
	s.Selected = s.Selected.
		Foreground(styles.ColorWhite).
		Background(lipgloss.Color("#003050")).
		Bold(false)
	s.Cell = s.Cell.
		Foreground(styles.ColorWhite)
	return s
}
