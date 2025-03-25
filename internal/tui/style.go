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

func viewportDivider() string {
	return titleBar("Results")
}

func appFooter() string {
	return titleBar("C: Copy Results | ESC: Exit")
}
