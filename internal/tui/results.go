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

func sendResultsCmd(body string) tea.Cmd {
	return func() tea.Msg {
		return updateResultsMsg{
			body: body,
		}
	}
}

func clearViewport() tea.Msg {
	return updateResultsMsg{body: ""}
}

func toggleViewport(on bool) tea.Cmd {
	return func() tea.Msg {
		return toggleViewportMsg{on}
	}
}
