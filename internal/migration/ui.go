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

func badRedOutput(label string, output string) string {
	return fmt.Sprintf("%s %s\n", textRed(label), output)
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

func (m *Model) viewportDivider() string {
	return m.titleBar("Results")
}

func (m *Model) appFooter() string {
	return m.titleBar("C: Copy Results | CTRL+Q: Exit")
}

func (m *Model) titleBar(t string) string {
	titleBox := titleStyle().Render(t)

	dividerLength := m.windowWidth - lipgloss.Width(titleBox)

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
	return key.NewBinding(key.WithKeys("ctrl+q"))
}

func customKeyMap() *huh.KeyMap {
	k := huh.NewDefaultKeyMap()
	k.Quit = quitKeyMap()

	return k
}

func (m *Model) calculateDimensions(w, h int) tea.Cmd {
	return func() tea.Msg {
		m.windowWidth = w
		m.windowHeight = h
		m.mainHeaderHeight = lipgloss.Height(m.titleBar("Ticket Migration Utility"))
		m.mainFooterHeight = lipgloss.Height(m.appFooter())
		m.viewportDvdrHeight = lipgloss.Height(m.viewportDivider())
		m.verticalMarginHeight = m.mainHeaderHeight + m.mainFooterHeight + m.viewportDvdrHeight
		viewportHeight := (m.windowHeight - m.verticalMarginHeight) * 1 / 2
		m.verticalLeftForMainView = m.windowHeight - m.verticalMarginHeight - viewportHeight
		slog.Debug("got calculateDimensionsMsg")

		if !m.ready {
			m.viewport = viewport.New(m.windowWidth, viewportHeight)
		} else {
			m.viewport.Width = m.windowWidth
			m.viewport.Height = viewportHeight
		}

		m.viewport.SetContent(m.data.Output.String())
		m.setAutoScrollBehavior()
		slog.Debug("setting ready to true")
		m.ready = true

		return nil
	}
}

func (m *Model) runSpinner(text string) string {
	return fmt.Sprintf("%s%s", text, m.spinner.View())
}

func (m *Model) setAutoScrollBehavior() {
	if m.viewport.AtBottom() {
		m.scrollOverride = false
	}

	if !m.scrollOverride {
		m.viewport.GotoBottom()
	}
}

func (m *Model) writeToOutput(s string, level outputLevel) {
	switch level {
	case noActionOutput:
		if m.client.Cfg.OutputLevels.NoAction {
			m.data.Output.WriteString(s)
		}
	case createdOutput:
		if m.client.Cfg.OutputLevels.Created {
			m.data.Output.WriteString(s)
		}
	case warnOutput:
		if m.client.Cfg.OutputLevels.Warn {
			m.data.Output.WriteString(s)
		}
	case errOutput:
		if m.client.Cfg.OutputLevels.Error {
			m.data.Output.WriteString(s)
		}
	}
}
