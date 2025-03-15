package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

type updateResultsMsg struct {
	title string
	body  string
}

func sendResultsCmd(title, body string) tea.Cmd {
	return func() tea.Msg {
		return updateResultsMsg{
			title: title,
			body:  body,
		}
	}
}
