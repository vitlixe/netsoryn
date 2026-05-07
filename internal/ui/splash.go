package ui

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/vitlixe/netsoryn/internal/ui/styles"
)

type splashDoneMsg struct{}

// Splash is the startup splash screen shown before the main UI.
type Splash struct {
	width  int
	height int
	main   RootModel
}

func NewSplash(main RootModel) Splash {
	return Splash{main: main}
}

func (s Splash) Init() tea.Cmd {
	return tea.Batch(
		s.main.Init(),
		tea.Tick(1500*time.Millisecond, func(_ time.Time) tea.Msg {
			return splashDoneMsg{}
		}),
	)
}

func (s Splash) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case splashDoneMsg:
		return s.main, nil
	case tea.KeyMsg:
		return s.main, nil
	case tea.WindowSizeMsg:
		s.width = msg.Width
		s.height = msg.Height
		updated, cmd := s.main.Update(msg)
		s.main = updated.(RootModel)
		return s, cmd
	default:
		updated, cmd := s.main.Update(msg)
		s.main = updated.(RootModel)
		return s, cmd
	}
}

// logoASCII is "NETSORYN" in figlet block font (71 cols wide, 6 rows).
const logoASCII = `███╗   ██╗███████╗████████╗███████╗ ██████╗ ██████╗ ██╗   ██╗███╗   ██╗
████╗  ██║██╔════╝╚══██╔══╝██╔════╝██╔═══██╗██╔══██╗╚██╗ ██╔╝████╗  ██║
██╔██╗ ██║█████╗     ██║   ███████╗██║   ██║██████╔╝ ╚████╔╝ ██╔██╗ ██║
██║╚██╗██║██╔══╝     ██║   ╚════██║██║   ██║██╔══██╗  ╚██╔╝  ██║╚██╗██║
██║ ╚████║███████╗   ██║   ███████║╚██████╔╝██║  ██║   ██║   ██║ ╚████║
╚═╝  ╚═══╝╚══════╝   ╚═╝   ╚══════╝ ╚═════╝ ╚═╝  ╚═╝   ╚═╝   ╚═╝  ╚═══╝`

const logoW = 71

func (s Splash) View() string {
	if s.width == 0 {
		return ""
	}

	bg := styles.ColorDarkBg
	fill := func(n int) string {
		if n <= 0 {
			return ""
		}
		return lipgloss.NewStyle().Background(bg).Render(strings.Repeat(" ", n))
	}

	logoStyle := lipgloss.NewStyle().Foreground(styles.ColorAccent).Bold(true).Background(bg)
	verStyle := lipgloss.NewStyle().Foreground(styles.ColorGray).Background(bg)

	verText := formatVersion(s.main.version)
	verRendered := verStyle.Render(verText)
	verW := lipgloss.Width(verRendered)

	logoLines := strings.Split(logoASCII, "\n")
	contentH := len(logoLines) + 2 // logo + blank + version

	topPad := (s.height - contentH) / 2
	if topPad < 0 {
		topPad = 0
	}
	botPad := s.height - topPad - contentH
	if botPad < 0 {
		botPad = 0
	}

	emptyLine := fill(s.width)

	leftLogo := (s.width - logoW) / 2
	if leftLogo < 0 {
		leftLogo = 0
	}
	rightLogo := s.width - leftLogo - logoW
	if rightLogo < 0 {
		rightLogo = 0
	}

	leftVer := (s.width - verW) / 2
	if leftVer < 0 {
		leftVer = 0
	}
	rightVer := s.width - leftVer - verW
	if rightVer < 0 {
		rightVer = 0
	}

	var sb strings.Builder

	for i := 0; i < topPad; i++ {
		sb.WriteString(emptyLine + "\n")
	}
	for _, line := range logoLines {
		sb.WriteString(fill(leftLogo) + logoStyle.Render(line) + fill(rightLogo) + "\n")
	}
	sb.WriteString(emptyLine + "\n")
	sb.WriteString(fill(leftVer) + verRendered + fill(rightVer))
	for i := 0; i < botPad; i++ {
		sb.WriteString("\n" + emptyLine)
	}

	return sb.String()
}
