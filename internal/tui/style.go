package tui

import (
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

func viewportDivider(v viewPort) string {
	return titleBar(v.title)
}

func appFooter() string {
	return titleBar("C: Copy Results | J: Write to File | ESC: Exit")
}
