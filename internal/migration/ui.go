package migration

import (
	"fmt"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"log/slog"
	"strings"
)

var (
	titleStyle = func() lipgloss.Style {
		b := lipgloss.NormalBorder()
		return lipgloss.NewStyle().BorderStyle(b).Padding(0, 1)
	}
)

func textRed(s string) string {
	return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("1")).Render(s)
}

func textYellow(s string) string {
	return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("3")).Render(s)
}

func textBlue(s string) string {
	return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("4")).Render(s)
}

// I am not colorblind. I know it's turquoise.
func textGreen(s string) string {
	return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6")).Render(s)
}

func textNormalAdaptive(s string) string {
	return lipgloss.NewStyle().Bold(true).
		Foreground(lipgloss.AdaptiveColor{Light: "236", Dark: "247"}).
		Render(s)
}

func badRedOutput(label string, err error) string {
	return fmt.Sprintf("%s %s\n", textRed(label), err)
}

func warnYellowOutput(label, output string) string {
	return fmt.Sprintf("%s %s\n", textYellow(label), output)
}

func goodBlueOutput(label, output string) string {
	return fmt.Sprintf("%s %s\n", textBlue(label), output)
}

func goodGreenOutput(label, output string) string {
	return fmt.Sprintf("%s %s\n", textGreen(label), output)
}

func infoOutput(label, output string) string {
	return fmt.Sprintf("%s %s\n", textNormalAdaptive(label), output)
}

func viewportDivider() string {
	return titleBar("Results")
}

func appFooter() string {
	return titleBar("C: Copy Results | ESC: Exit")
}

func titleBar(t string) string {
	titleBox := titleStyle().Render(t)

	dividerLength := windowWidth - lipgloss.Width(titleBox)

	return lipgloss.JoinHorizontal(lipgloss.Center, titleBox, line(dividerLength))
}

func line(w int) string {
	line := strings.Repeat("─", maxRepeats(0, w))
	return line
}

func maxRepeats(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func customFormTheme() *huh.Theme {
	t := huh.ThemeBase()
	t.Focused.Base = lipgloss.NewStyle()
	t.Blurred.Base = t.Focused.Base
	return t
}

func quitKeyMap() key.Binding {
	return key.NewBinding(key.WithKeys("esc", "ctrl+c"))
}

func customKeyMap() *huh.KeyMap {
	k := huh.NewDefaultKeyMap()
	k.Quit = quitKeyMap()

	return k
}

func (m *Model) calculateDimensions(w, h int) tea.Cmd {
	return func() tea.Msg {
		windowWidth = w
		windowHeight = h
		mainHeaderHeight = lipgloss.Height(titleBar("Ticket Migration Utility"))
		mainFooterHeight = lipgloss.Height(appFooter())
		viewportDvdrHeight = lipgloss.Height(viewportDivider())
		verticalMarginHeight = mainHeaderHeight + mainFooterHeight + viewportDvdrHeight
		viewportHeight := (windowHeight - verticalMarginHeight) * 1 / 2
		verticalLeftForMainView = windowHeight - verticalMarginHeight - viewportHeight
		slog.Debug("got calculateDimensionsMsg")

		if !m.ready {
			m.viewport = viewport.New(windowWidth, viewportHeight)
		} else {
			m.viewport.Width = windowWidth
			m.viewport.Height = viewportHeight
		}

		m.viewport.SetContent(m.data.Output.String())
		m.setAutoScrollBehavior()
		slog.Debug("setting ready to true")
		m.ready = true

		return nil
	}
}
