package tui

import (
	"fmt"
	"github.com/charmbracelet/lipgloss"
	"strings"
)

var (
	titleStyle = func() lipgloss.Style {
		b := lipgloss.NormalBorder()
		return lipgloss.NewStyle().BorderStyle(b).Padding(0, 1)
	}

	inactiveTabStyle = func() lipgloss.Style {
		return lipgloss.NewStyle().Border(inactiveTabBorder()).Padding(0, 1).Faint(true)
	}

	activeTabStyle = func() lipgloss.Style {
		return lipgloss.NewStyle().Border(activeTabBorder()).Padding(0, 1)
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

func badRedOutput(label string, err error) string {
	e := textRed(label)
	return fmt.Sprintf("%s %s\n", e, err)
}

func warnYellowOutput(label, output string) string {
	e := textYellow(label)
	return fmt.Sprintf("%s %s\n", e, output)
}

func goodBlueOutput(label, output string) string {
	e := textBlue(label)
	return fmt.Sprintf("%s %s\n", e, output)
}

func goodGreenOutput(label, output string) string {
	e := textGreen(label)
	return fmt.Sprintf("%s %s\n", e, output)
}

func activeTabBorder() lipgloss.Border {
	b := lipgloss.NormalBorder()
	b.BottomLeft = "┘"
	b.BottomRight = "└"
	b.Bottom = ""
	return b
}

func inactiveTabBorder() lipgloss.Border {
	b := lipgloss.NormalBorder()
	b.BottomLeft = "┴"
	b.BottomRight = "┴"
	return b
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

func menuBar(tabs []menuTab, activeTab menuTab) string {
	var tabText []string
	for _, tab := range tabs {
		r := inactiveTabStyle().Render(string(tab))
		if tab == activeTab {
			r = activeTabStyle().Render(string(tab))
		}
		tabText = append(tabText, r)
	}
	renderedTabs := lipgloss.JoinHorizontal(lipgloss.Bottom, tabText...)
	dividerLength := windowWidth - lipgloss.Width(renderedTabs)
	return lipgloss.JoinHorizontal(lipgloss.Bottom, renderedTabs, line(dividerLength))
}

func viewportDivider() string {
	return titleBar("Results")
}

func appFooter() string {
	return titleBar("C: Copy Results | ESC: Exit")
}
