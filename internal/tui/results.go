package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

type updateResultsMsg struct {
	body string
	show bool
}

type toggleViewportMsg struct {
	on bool
}

func toggleViewport(on bool) tea.Cmd {
	return func() tea.Msg {
		return toggleViewportMsg{on}
	}
}
