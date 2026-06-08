package keys

import "github.com/charmbracelet/bubbles/key"

// GlobalKeyMap defines keybindings handled by the root model.
type GlobalKeyMap struct {
	Quit    key.Binding
	Help    key.Binding
	Tab     key.Binding
	BackTab key.Binding
	View1   key.Binding
	View2   key.Binding
	View3   key.Binding
	View4   key.Binding
	View5   key.Binding
	View6   key.Binding
	View7   key.Binding
	View8   key.Binding
}

// DefaultGlobalKeyMap returns the default global key bindings.
func DefaultGlobalKeyMap() GlobalKeyMap {
	return GlobalKeyMap{
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		Tab: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "next view"),
		),
		BackTab: key.NewBinding(
			key.WithKeys("shift+tab"),
			key.WithHelp("shift+tab", "prev view"),
		),
		View1: numKey("1"),
		View2: numKey("2"),
		View3: numKey("3"),
		View4: numKey("4"),
		View5: numKey("5"),
		View6: numKey("6"),
		View7: numKey("7"),
		View8: numKey("8"),
	}
}

func numKey(k string) key.Binding {
	return key.NewBinding(key.WithKeys(k))
}

// NavKeyMap defines keybindings shared across navigable list/table views.
type NavKeyMap struct {
	Up       key.Binding
	Down     key.Binding
	PageUp   key.Binding
	PageDown key.Binding
	Top      key.Binding
	Bottom   key.Binding
	Filter   key.Binding
	Clear    key.Binding
	Enter    key.Binding
	Back     key.Binding
	SortNext key.Binding
}

// DefaultNavKeyMap returns the default navigation key map.
func DefaultNavKeyMap() NavKeyMap {
	return NavKeyMap{
		Up: key.NewBinding(
			key.WithKeys("k", "up"),
			key.WithHelp("k/↑", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("j", "down"),
			key.WithHelp("j/↓", "down"),
		),
		PageUp: key.NewBinding(
			key.WithKeys("ctrl+u", "pgup"),
			key.WithHelp("ctrl+u", "page up"),
		),
		PageDown: key.NewBinding(
			key.WithKeys("ctrl+d", "pgdown"),
			key.WithHelp("ctrl+d", "page down"),
		),
		Top: key.NewBinding(
			key.WithKeys("g"),
			key.WithHelp("g", "top"),
		),
		Bottom: key.NewBinding(
			key.WithKeys("G"),
			key.WithHelp("G", "bottom"),
		),
		Filter: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "filter"),
		),
		Clear: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "clear/back"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "select"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "back"),
		),
		SortNext: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "sort"),
		),
	}
}

// HelpEntries returns a formatted list of global shortcuts for the help overlay.
func HelpEntries() [][2]string {
	return [][2]string{
		{"1-9", "Switch view"},
		{"tab", "Next view"},
		{"shift+tab", "Prev view"},
		{"j/k", "Navigate down/up"},
		{"g/G", "Top/Bottom"},
		{"ctrl+d/u", "Page down/up"},
		{"/", "Filter"},
		{"esc", "Clear filter / back"},
		{"s", "Cycle sort column"},
		{"enter", "Expand detail"},
		{"?", "Toggle help"},
		{"q", "Quit"},
	}
}
